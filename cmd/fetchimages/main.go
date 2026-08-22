// Command fetchimages downloads card images into the cache directory and
// writes a manifest of content hashes. It fetches through the same source
// chain as the server (IMAGE_MIRROR when configured, marvelcdb.com
// otherwise); images are never committed to the repository. This tool runs
// at docker build time and optionally during local development.
//
// The manifest powers permanent client-side caching: image URLs carry
// ?v=<sha256-prefix> and the server marks them immutable.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/api"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/dotenv"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/mirror"
)

func main() {
	outDir := flag.String("out", filepath.Join(envOr("MC_CACHE_DIR", "cache"), "images"), "output directory for card images")
	packs := flag.String("packs", "", "comma-separated pack codes to limit the fetch (default: all)")
	flag.Parse()

	// Optional repo-root .env, e.g. with an HTTP mirror override; real
	// environment variables take precedence.
	if n, err := dotenv.Load(".env"); err != nil {
		log.Printf(".env: %v", err)
	} else if n > 0 {
		log.Printf("loaded %d variables from .env", n)
	}

	sources := mirror.SourcesFromEnv()
	chain := mirror.Chain(sources.Default...)
	// Only rate-limit when hitting marvelcdb directly; a mirror is our own
	// infrastructure.
	polite := !sources.DefaultIsMirror
	// Mirrors follow the face convention; bare marvelcdb needs its legacy
	// per-face paths.
	pathFor := api.DefaultImagePathFor(sources.DefaultIsMirror)

	db := data.MustLoad()

	// Target set: every card code, faces included (back faces are rendered
	// by the client as the same asset for now).
	codes := make([]string, 0, len(db.All()))
	for _, def := range db.All() {
		if *packs != "" && !contains(strings.Split(*packs, ","), def.PackCode) {
			continue
		}
		codes = append(codes, def.Code)
	}
	sort.Strings(codes)
	log.Printf("fetching %d card images from %s into %s", len(codes), chain.Name(), *outDir)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}

	manifest := map[string]string{}
	fetched, skipped, failed := 0, 0, 0
	for i, code := range codes {
		remote := pathFor(code)
		ext := filepath.Ext(remote)
		if ext == "" {
			ext = ".png"
		}
		path := filepath.Join(*outDir, code+ext)
		if existing, err := os.ReadFile(path); err == nil {
			manifest[code] = hashBytes(existing)
			skipped++
			continue
		}
		body, err := chain.Fetch(remote)
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
		if polite {
			time.Sleep(150 * time.Millisecond) // stay polite with marvelcdb
		}
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
