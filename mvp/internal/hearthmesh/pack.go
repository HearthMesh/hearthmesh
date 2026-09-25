package hearthmesh

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const MaxPackSize = 32 << 20
const maxFiles = 256

type File struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	Version   int    `json:"version"`
	Publisher string `json:"publisher"`
	Files     []File `json:"files"`
	Signature string `json:"signature"`
}

type unsignedManifest struct {
	Version   int    `json:"version"`
	Publisher string `json:"publisher"`
	Files     []File `json:"files"`
}

// LoadIdentity creates a persistent Ed25519 private key if the path is absent.
// Keep this file private: it is the publisher and node's long-lived identity.
func LoadIdentity(filename string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			return nil, err
		}
		_, key, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		data = []byte(base64.StdEncoding.EncodeToString(key))
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if errors.Is(err, os.ErrExist) {
			return LoadIdentity(filename)
		}
		if err != nil {
			return nil, err
		}
		if _, err = f.Write(data); err != nil {
			f.Close()
			return nil, err
		}
		if err = f.Close(); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(key) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid identity key")
	}
	return ed25519.PrivateKey(key), nil
}

func PublicKey(key ed25519.PrivateKey) string {
	return hex.EncodeToString(key.Public().(ed25519.PublicKey))
}

func validPath(name string) bool {
	return name != "" && name != "." && !strings.Contains(name, "\\") && !strings.HasPrefix(name, "/") && path.Clean(name) == name && !strings.HasPrefix(name, "../") && name != ".." && !strings.HasPrefix(name, "files/") && name != "manifest.json"
}

func (m Manifest) signingBytes() ([]byte, error) {
	return json.Marshal(unsignedManifest{m.Version, m.Publisher, m.Files})
}

func (m Manifest) ID() string {
	// A canonical JSON manifest identifies a pack, independent of ZIP metadata.
	b, _ := json.Marshal(m)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (m Manifest) Validate() error {
	if m.Version != 1 || len(m.Files) == 0 || len(m.Files) > maxFiles {
		return errors.New("invalid manifest version or file count")
	}
	pub, err := hex.DecodeString(m.Publisher)
	if err != nil || len(pub) != ed25519.PublicKeySize || hex.EncodeToString(pub) != m.Publisher {
		return errors.New("invalid publisher")
	}
	sig, err := base64.StdEncoding.DecodeString(m.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return errors.New("invalid signature encoding")
	}
	var total int64
	for i, f := range m.Files {
		if !validPath(f.Path) || i > 0 && m.Files[i-1].Path >= f.Path || f.Size < 0 || f.Size > MaxPackSize {
			return errors.New("invalid file entry or order")
		}
		h, err := hex.DecodeString(f.SHA256)
		if err != nil || len(h) != sha256.Size || hex.EncodeToString(h) != f.SHA256 {
			return errors.New("invalid file hash")
		}
		total += f.Size
	}
	if total > MaxPackSize {
		return errors.New("pack too large")
	}
	message, _ := m.signingBytes()
	if !ed25519.Verify(ed25519.PublicKey(pub), message, sig) {
		return errors.New("signature verification failed")
	}
	return nil
}

// CreatePack writes a signed ZIP with manifest.json and files/<relative path>.
func CreatePack(src, out string, key ed25519.PrivateKey) (string, error) {
	var files []File
	var total int64
	content := make(map[string][]byte)
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if p == src {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("not a regular file: %s", p)
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if !validPath(name) {
			return fmt.Errorf("invalid path: %s", name)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > MaxPackSize || total+info.Size() > MaxPackSize {
			return errors.New("pack too large")
		}
		file, err := os.Open(p)
		if err != nil {
			return err
		}
		b, err := io.ReadAll(io.LimitReader(file, MaxPackSize-total+1))
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
		if int64(len(b)) > MaxPackSize || total+int64(len(b)) > MaxPackSize {
			return errors.New("pack too large")
		}
		total += int64(len(b))
		h := sha256.Sum256(b)
		content[name] = b
		files = append(files, File{name, int64(len(b)), hex.EncodeToString(h[:])})
		if len(files) > maxFiles {
			return errors.New("too many files")
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	m := Manifest{Version: 1, Publisher: PublicKey(key), Files: files}
	message, _ := m.signingBytes()
	m.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(key, message))
	if err := m.Validate(); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	mb, _ := json.MarshalIndent(m, "", "  ")
	w, err := zw.Create("manifest.json")
	if err != nil {
		return "", err
	}
	if _, err = w.Write(mb); err != nil {
		return "", err
	}
	for _, f := range files {
		w, err = zw.Create("files/" + f.Path)
		if err != nil {
			return "", err
		}
		if _, err = w.Write(content[f.Path]); err != nil {
			return "", err
		}
	}
	if err = zw.Close(); err != nil {
		return "", err
	}
	if buf.Len() > MaxPackSize {
		return "", errors.New("compressed pack too large")
	}
	if err = os.WriteFile(out, buf.Bytes(), 0600); err != nil {
		return "", err
	}
	return m.ID(), nil
}

func VerifyPack(data []byte) (Manifest, error) {
	if len(data) > MaxPackSize {
		return Manifest{}, errors.New("pack too large")
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Manifest{}, err
	}
	if len(zr.File) < 2 || len(zr.File) > maxFiles+1 {
		return Manifest{}, errors.New("invalid ZIP entry count")
	}
	entries := make(map[string]*zip.File)
	for _, f := range zr.File {
		if _, exists := entries[f.Name]; exists || f.FileInfo().IsDir() || f.Mode()&os.ModeSymlink != 0 {
			return Manifest{}, errors.New("duplicate or unsupported ZIP entry")
		}
		entries[f.Name] = f
	}
	mf := entries["manifest.json"]
	if mf == nil || mf.UncompressedSize64 > 1<<20 {
		return Manifest{}, errors.New("missing or large manifest")
	}
	manifestBytes, err := readZip(mf, 1<<20)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err = json.Unmarshal(manifestBytes, &m); err != nil {
		return m, err
	}
	if err = m.Validate(); err != nil {
		return m, err
	}
	if len(entries) != len(m.Files)+1 {
		return m, errors.New("unexpected ZIP entries")
	}
	for _, f := range m.Files {
		zf := entries["files/"+f.Path]
		if zf == nil || zf.UncompressedSize64 != uint64(f.Size) {
			return m, fmt.Errorf("missing or wrong size: %s", f.Path)
		}
		b, err := readZip(zf, f.Size)
		if err != nil {
			return m, err
		}
		h := sha256.Sum256(b)
		if hex.EncodeToString(h[:]) != f.SHA256 {
			return m, fmt.Errorf("hash mismatch: %s", f.Path)
		}
	}
	return m, nil
}

func readZip(f *zip.File, limit int64) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("decompressed entry too large")
	}
	return b, nil
}
