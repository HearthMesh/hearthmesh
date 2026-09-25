package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/HearthMesh/hearthmesh/mvp/internal/hearthmesh"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: hearthmesh <identity|pack|verify|publish|serve> [flags]")
	}
	switch args[0] {
	case "identity":
		fs := flag.NewFlagSet("identity", flag.ContinueOnError)
		keyPath := fs.String("key", "identity.key", "persistent Ed25519 key file")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		key, err := hearthmesh.LoadIdentity(*keyPath)
		if err != nil {
			return err
		}
		fmt.Println("publisher:", hearthmesh.PublicKey(key))
	case "pack":
		fs := flag.NewFlagSet("pack", flag.ContinueOnError)
		src := fs.String("src", "", "directory of files")
		out := fs.String("out", "", "output .zip path")
		keyPath := fs.String("key", "identity.key", "persistent publisher key file")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *src == "" || *out == "" {
			return errors.New("-src and -out are required")
		}
		key, err := hearthmesh.LoadIdentity(*keyPath)
		if err != nil {
			return err
		}
		id, err := hearthmesh.CreatePack(*src, *out, key)
		if err != nil {
			return err
		}
		fmt.Println("pack ID:", id)
	case "verify":
		fs := flag.NewFlagSet("verify", flag.ContinueOnError)
		file := fs.String("pack", "", "signed pack .zip")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *file == "" {
			return errors.New("-pack is required")
		}
		b, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		m, err := hearthmesh.VerifyPack(b)
		if err != nil {
			return err
		}
		fmt.Printf("verified pack ID: %s\npublisher: %s\nfiles: %d\n", m.ID(), m.Publisher, len(m.Files))
	case "publish":
		fs := flag.NewFlagSet("publish", flag.ContinueOnError)
		file := fs.String("pack", "", "signed pack .zip")
		node := fs.String("node", "http://127.0.0.1:8081", "destination node URL")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *file == "" {
			return errors.New("-pack is required")
		}
		b, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		m, err := hearthmesh.VerifyPack(b)
		if err != nil {
			return err
		}
		client := &http.Client{Timeout: 15 * time.Second}
		url := strings.TrimRight(*node, "/") + "/v1/packs/" + m.ID()
		req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(b))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/zip")
		res, err := client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
			return fmt.Errorf("node returned %d: %s", res.StatusCode, body)
		}
		fmt.Println("published pack ID:", m.ID())
	case "serve":
		fs := flag.NewFlagSet("serve", flag.ContinueOnError)
		data := fs.String("data", "./node-data", "persistent node data directory")
		listen := fs.String("listen", "127.0.0.1:8080", "HTTP listen address")
		peers := fs.String("peers", "", "comma-separated explicit bootstrap peer URLs")
		interval := fs.Duration("sync", 2*time.Second, "replication and integrity scan interval")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *interval < time.Second {
			return errors.New("-sync must be at least 1s")
		}
		if err := os.MkdirAll(filepath.Clean(*data), 0700); err != nil {
			return err
		}
		n, err := hearthmesh.NewNode(*data, strings.Split(*peers, ","))
		if err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		go n.RunSync(ctx, *interval)
		srv := &http.Server{Addr: *listen, Handler: n.Handler(), ReadHeaderTimeout: 5 * time.Second}
		go func() {
			<-ctx.Done()
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdown)
		}()
		fmt.Printf("experimental node %s listening on %s\n", n.ID, *listen)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	default:
		return fmt.Errorf("unknown command %q; expected identity, pack, verify, publish, serve", args[0])
	}
	return nil
}
