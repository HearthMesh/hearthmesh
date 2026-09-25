package hearthmesh

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Node struct {
	Dir    string
	ID     string
	Peers  []string
	Client *http.Client
}

type Status struct {
	Node    string   `json:"node"`
	Peers   []string `json:"peers"`
	Packs   []string `json:"packs"`
	Corrupt []string `json:"corrupt"`
}

func NewNode(dir string, peers []string) (*Node, error) {
	key, err := LoadIdentity(filepath.Join(dir, "identity.key"))
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Join(dir, "packs"), 0700); err != nil {
		return nil, err
	}
	clean := make([]string, 0, len(peers))
	for _, peer := range peers {
		peer = strings.TrimRight(strings.TrimSpace(peer), "/")
		if peer != "" {
			clean = append(clean, peer)
		}
	}
	return &Node{Dir: dir, ID: PublicKey(key), Peers: clean, Client: &http.Client{Timeout: 8 * time.Second}}, nil
}

func validID(id string) bool {
	b, err := hex.DecodeString(id)
	return err == nil && len(b) == 32 && hex.EncodeToString(b) == id
}

func (n *Node) packPath(id string) string { return filepath.Join(n.Dir, "packs", id+".zip") }

func (n *Node) localPack(id string) ([]byte, error) {
	if !validID(id) {
		return nil, fmt.Errorf("invalid pack ID")
	}
	b, err := os.ReadFile(n.packPath(id))
	if err != nil {
		return nil, err
	}
	m, err := VerifyPack(b)
	if err != nil {
		return nil, err
	}
	if m.ID() != id {
		return nil, fmt.Errorf("pack ID mismatch")
	}
	return b, nil
}

func (n *Node) savePack(id string, b []byte) error {
	m, err := VerifyPack(b)
	if err != nil {
		return err
	}
	if !validID(id) || m.ID() != id {
		return fmt.Errorf("pack ID mismatch")
	}
	f, err := os.CreateTemp(filepath.Join(n.Dir, "packs"), ".incoming-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp, n.packPath(id))
}

func (n *Node) Status() Status {
	s := Status{Node: n.ID, Peers: append([]string{}, n.Peers...), Packs: []string{}, Corrupt: []string{}}
	entries, err := os.ReadDir(filepath.Join(n.Dir, "packs"))
	if err != nil {
		return s
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".zip") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".zip")
		if !validID(id) {
			continue
		}
		if _, err := n.localPack(id); err != nil {
			s.Corrupt = append(s.Corrupt, id)
		} else {
			s.Packs = append(s.Packs, id)
		}
	}
	sort.Strings(s.Packs)
	sort.Strings(s.Corrupt)
	return s
}

func (n *Node) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = portal.Execute(w, n.Status())
	})
	mux.HandleFunc("/v1/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET required", 405)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(n.Status())
	})
	mux.HandleFunc("/v1/packs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET required", 405)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(n.Status().Packs)
	})
	mux.HandleFunc("/v1/packs/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/v1/packs/")
		if !validID(id) {
			http.Error(w, "invalid pack ID", 400)
			return
		}
		switch r.Method {
		case http.MethodGet:
			b, err := n.localPack(id)
			if err != nil {
				http.Error(w, "missing or corrupt pack", 404)
				return
			}
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(b)
		case http.MethodPut:
			b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxPackSize))
			if err != nil {
				http.Error(w, "pack exceeds limit", 413)
				return
			}
			if err = n.savePack(id, b); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			w.WriteHeader(http.StatusCreated)
		default:
			http.Error(w, "GET or PUT required", 405)
		}
	})
	return mux
}

var portal = template.Must(template.New("portal").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>HearthMesh</title><style>body{font:16px system-ui;max-width:850px;margin:3rem auto;padding:0 1rem;background:#faf8f4;color:#302b28}code{word-break:break-all}li{margin:.7rem 0}</style><h1>HearthMesh <small>experimental</small></h1><p>Local node: <code>{{.Node}}</code></p><h2>Verified public packs ({{len .Packs}})</h2><ul>{{range .Packs}}<li><a href="/v1/packs/{{.}}"><code>{{.}}</code></a></li>{{else}}<li>No packs yet</li>{{end}}</ul><h2>Corrupt local packs ({{len .Corrupt}})</h2><ul>{{range .Corrupt}}<li><code>{{.}}</code></li>{{else}}<li>None detected</li>{{end}}</ul><h2>Bootstrap peers</h2><ul>{{range .Peers}}<li><code>{{.}}</code></li>{{end}}</ul><p>Read-only portal. Publishing uses the HTTP API. This prototype is not production-secure.</p></html>`))

// SyncOnce discovers verified public packs on configured peers and repairs
// corrupt local copies. Peers are untrusted: every downloaded pack is verified.
func (n *Node) SyncOnce(ctx context.Context) {
	status := n.Status()
	wanted := map[string]bool{}
	for _, id := range status.Corrupt {
		wanted[id] = true
	}
	for _, peer := range n.Peers {
		var ids []string
		if err := n.getJSON(ctx, peer+"/v1/packs", &ids); err != nil {
			continue
		}
		for _, id := range ids {
			if validID(id) {
				wanted[id] = true
			}
		}
	}
	for id := range wanted {
		if _, err := n.localPack(id); err == nil {
			continue
		}
		for _, peer := range n.Peers {
			if err := n.fetchPack(ctx, peer, id); err == nil {
				break
			}
		}
	}
}

func (n *Node) getJSON(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := n.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("peer returned %d", res.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(dst)
}

func (n *Node) fetchPack(ctx context.Context, peer, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, peer+"/v1/packs/"+id, nil)
	if err != nil {
		return err
	}
	res, err := n.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("peer returned %d", res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, MaxPackSize+1))
	if err != nil {
		return err
	}
	return n.savePack(id, b)
}

func (n *Node) RunSync(ctx context.Context, interval time.Duration) {
	n.SyncOnce(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n.SyncOnce(ctx)
		}
	}
}
