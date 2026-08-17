// Command fetchimages downloads card images from marvelcdb.com into
// web/public/img/cards and writes a manifest of content hashes. Images are
// never committed to the repository; this tool runs at docker build time and
// optionally during local development.
//
// The manifest powers permanent client-side caching: image URLs carry
// ?v=<sha256-prefix> and the server marks them immutable.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

const imageBase = "https://marvelcdb.com"

func main() {
	outDir := flag.String("out", filepath.Join(envOr("MC_CACHE_DIR", "cache"), "images"), "output directory for card images")
	packs := flag.String("packs", "", "comma-separated pack codes to limit the fetch (default: all)")
	flag.Parse()

	db := data.MustLoad()

	// Target set: one image per card code (front images; back faces are
	// rendered by the client as the same asset for now).
	codes := make([]string, 0, len(db.All()))
	for _, def := range db.All() {
		if def.ImageSrc == "" {
			continue
		}
		if *packs != "" && !contains(strings.Split(*packs, ","), def.PackCode) {
			continue
		}
		codes = append(codes, def.Code)
	}
	sort.Strings(codes)
	log.Printf("fetching %d card images into %s", len(codes), *outDir)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	manifest := map[string]string{}
	fetched, skipped, failed := 0, 0, 0
	for i, code := range codes {
		def, _ := db.Lookup(code)
		ext := filepath.Ext(def.ImageSrc)
		if ext == "" {
			ext = ".png"
		}
		path := filepath.Join(*outDir, code+ext)
		if existing, err := os.ReadFile(path); err == nil {
			manifest[code] = hashBytes(existing)
			skipped++
			continue
		}
		url := imageBase + def.ImageSrc
		body, err := fetch(client, url)
		if err != nil {
			log.Printf("WARN %s: %v", code, err)
			failed++
			continue
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			log.Fatalf("write %s: %v", path, err)
		}
		manifest[code] = hashBytes(body)
		fetched++
		if i%200 == 0 {
			log.Printf("progress: %d/%d", i, len(codes))
		}
		time.Sleep(150 * time.Millisecond) // stay polite
	}

	manifestPath := filepath.Join(*outDir, "manifest.json")
	raw, err := json.MarshalIndent(manifest, "", " ")
	if err != nil {
		log.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		log.Fatalf("write manifest: %v", err)
	}
	log.Printf("done: fetched=%d cached=%d failed=%d manifest=%s", fetched, skipped, failed, manifestPath)
}

func fetch(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "marvelchampions-go/fetchimages")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

func hashBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])[:16] // 64 bits is plenty for cache busting
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
