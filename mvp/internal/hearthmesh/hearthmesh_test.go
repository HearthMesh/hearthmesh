package hearthmesh

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func fixture(t *testing.T) ([]byte, Manifest) {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.Mkdir(src, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "hello.txt"), []byte("hello hearthmesh"), 0600); err != nil {
		t.Fatal(err)
	}
	key, err := LoadIdentity(filepath.Join(dir, "identity.key"))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "pack.zip")
	if _, err = CreatePack(src, out, key); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	m, err := VerifyPack(b)
	if err != nil {
		t.Fatal(err)
	}
	return b, m
}

func TestIdentityPersists(t *testing.T) {
	p := filepath.Join(t.TempDir(), "key")
	a, err := LoadIdentity(p)
	if err != nil {
		t.Fatal(err)
	}
	b, err := LoadIdentity(p)
	if err != nil {
		t.Fatal(err)
	}
	if PublicKey(a) != PublicKey(b) {
		t.Fatal("identity changed")
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("key mode: %v", info.Mode())
	}
}

func TestPackRejectsTampering(t *testing.T) {
	b, m := fixture(t)
	if m.Version != 1 || len(m.Files) != 1 {
		t.Fatal("bad manifest")
	}
	if _, err := VerifyPack(b); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"file", "signature"} {
		var out bytes.Buffer
		zw := zip.NewWriter(&out)
		for _, f := range zr.File {
			r, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			content, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatal(err)
			}
			if target == "file" && f.Name == "files/hello.txt" {
				content = []byte("hello tampered!!")
			}
			if target == "signature" && f.Name == "manifest.json" {
				var altered Manifest
				if err := json.Unmarshal(content, &altered); err != nil {
					t.Fatal(err)
				}
				if altered.Publisher[:2] == "00" {
					altered.Publisher = "01" + altered.Publisher[2:]
				} else {
					altered.Publisher = "00" + altered.Publisher[2:]
				}
				content, _ = json.Marshal(altered)
			}
			w, err := zw.Create(f.Name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write(content); err != nil {
				t.Fatal(err)
			}
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := VerifyPack(out.Bytes()); err == nil {
			t.Fatalf("accepted %s tampering", target)
		}
	}
}

func TestReplicationAndRepair(t *testing.T) {
	b, m := fixture(t)
	a, err := NewNode(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(a.Handler())
	defer server.Close()
	peer, err := NewNode(t.TempDir(), []string{server.URL})
	if err != nil {
		t.Fatal(err)
	}
	put, err := http.NewRequest(http.MethodPut, server.URL+"/v1/packs/"+m.ID(), bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(put)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 201 {
		t.Fatalf("publish status: %d", res.StatusCode)
	}
	peer.SyncOnce(context.Background())
	if got := peer.Status().Packs; len(got) != 1 || got[0] != m.ID() {
		t.Fatalf("replication: %v", got)
	}
	if err := os.WriteFile(peer.packPath(m.ID()), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := peer.Status().Corrupt; len(got) != 1 {
		t.Fatalf("corruption detection: %v", got)
	}
	peer.SyncOnce(context.Background())
	if got := peer.Status(); len(got.Packs) != 1 || len(got.Corrupt) != 0 {
		t.Fatalf("repair: %+v", got)
	}
}

func TestRejectUnexpectedEntry(t *testing.T) {
	b, _ := fixture(t)
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		r, _ := f.Open()
		data, _ := io.ReadAll(r)
		r.Close()
		w, _ := zw.Create(f.Name)
		_, _ = w.Write(data)
	}
	w, _ := zw.Create("files/extra")
	_, _ = w.Write([]byte("surprise"))
	_ = zw.Close()
	if _, err := VerifyPack(out.Bytes()); err == nil {
		t.Fatal("accepted unexpected file")
	}
}
