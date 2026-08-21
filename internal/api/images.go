package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/mirror"
)

// imageCache serves card images with on-demand fetching: the first request
// for a card downloads it through the configured source chain (a mirror
// when configured, marvelcdb otherwise) into the cache directory, records
// its content hash in manifest.json, and every later request is served
// locally. Content-addressed URLs (/img/cards/{code}.{hash}.png) get
// immutable caching, exactly like the previous build-time pipeline.
type imageCache struct {
	dir    string
	source mirror.Source

	mu       sync.Mutex
	manifest map[string]string
}

// NewImageCache builds an on-demand image cache rooted at dir. Sources are
// tried in order on a miss; with no sources the cache serves only what is
// already on disk.
func NewImageCache(dir string, sources ...mirror.Source) (*imageCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	c := &imageCache{
		dir:      dir,
		source:   mirror.Chain(sources...),
		manifest: map[string]string{},
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "manifest.json")); err == nil {
		_ = json.Unmarshal(raw, &c.manifest)
	}
	return c, nil
}

type cachedImage struct {
	body     []byte
	mimeType string
	hash     string
}

// get returns an image for a card code, fetching and caching on miss. The
// network fetch happens outside the lock so concurrent prewarm workers and
// requests do not serialize; a duplicate fetch of the same code writes
// identical bytes and is harmless.
func (c *imageCache) get(code string) (*cachedImage, error) {
	c.mu.Lock()
	img, ok := c.cached(code)
	c.mu.Unlock()
	if ok {
		return img, nil
	}
	remote := c.remotePath(code)
	body, err := c.source.Fetch(remote)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", remote, err)
	}
	img = &cachedImage{body: body, mimeType: detectImage(body), hash: hashBytes(body)}
	if err := os.WriteFile(c.imagePath(code), body, 0o644); err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.manifest[code] = img.hash
	c.saveManifest()
	c.mu.Unlock()
	return img, nil
}

// cached returns the on-disk image for a code recorded in the manifest.
// Callers must hold c.mu (the manifest map is only mutated under it).
func (c *imageCache) cached(code string) (*cachedImage, bool) {
	hash, ok := c.manifest[code]
	if !ok {
		return nil, false
	}
	body, err := os.ReadFile(c.imagePath(code))
	if err != nil {
		return nil, false
	}
	return &cachedImage{body: body, mimeType: detectImage(body), hash: hash}, true
}

// peek returns the cached image for a code without ever touching the
// network. Used for the zh cache so locally seeded faces win over the
// shared source chain.
func (c *imageCache) peek(code string) (*cachedImage, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cached(code)
}

// PrewarmImages downloads the given card codes into the cache so the
// manifest — and with it every content-addressed image URL — covers them.
// Meant for mirror-backed deployments, started in the background at server
// startup; the on-disk cache keeps repeat runs nearly free. Returns the
// number of codes resolved from the cache and the number that failed.
func PrewarmImages(img *imageCache, codes []string, workers int, delay time.Duration) (cached, failed int) {
	if workers < 1 {
		workers = 1
	}
	var (
		mu    sync.Mutex
		wg    sync.WaitGroup
		sem   = make(chan struct{}, workers)
		total = len(codes)
	)
	for _, code := range codes {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			_, err := img.get(code)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed++
				log.Printf("images: prewarm %s: %v", code, err)
			} else {
				cached++
			}
			done := cached + failed
			if done%500 == 0 {
				log.Printf("images: prewarm %d/%d (failed=%d)", done, total, failed)
			}
		}(code)
		time.Sleep(delay)
	}
	wg.Wait()
	log.Printf("images: prewarm finished: %d/%d resolved, %d failed", cached, total, failed)
	return cached, failed
}

func (c *imageCache) imagePath(code string) string {
	return filepath.Join(c.dir, code+".img")
}

func (c *imageCache) remotePath(code string) string {
	if def, ok := engine.DB.Lookup(code); ok && def.ImageSrc != "" {
		return def.ImageSrc
	}
	return "/bundles/cards/" + code + ".png"
}

func (c *imageCache) saveManifest() {
	raw, err := json.MarshalIndent(c.manifest, "", " ")
	if err != nil {
		return
	}
	tmp := filepath.Join(c.dir, "manifest.json.tmp")
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, filepath.Join(c.dir, "manifest.json"))
}

func (c *imageCache) manifestJSON() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	raw, err := json.Marshal(c.manifest)
	if err != nil {
		return []byte("{}")
	}
	return raw
}

func hashBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])[:16]
}

func detectImage(body []byte) string {
	if t := http.DetectContentType(body); strings.HasPrefix(t, "image/") {
		return t
	}
	return "image/png"
}

// ImageHandler serves card images on demand. Two URL shapes:
//
//	/img/cards/{code}.png              unversioned, no-cache
//	/img/cards/{code}.{hash}.png       content-addressed, immutable
//
// The hash is the sha256 recorded in manifest.json; a request whose hash
// does not match the current content returns 404 so the client refetches
// the manifest.
func (s *Server) ImageHandler() http.Handler {
	// Default (English) faces: fetch through the default chain on miss
	// (IMAGE_MIRROR or marvelcdb).
	return s.imageHandler(func(code string) (*cachedImage, error) {
		return s.Images.get(code)
	})
}

// ZhImageHandler serves the zh routes (/img/cards/zh/): locally seeded
// faces first, then on-demand fetches from the Chinese mirror
// (ZH_IMAGE_MIRROR) — a separate language source with marvelcdb's exact
// paths, never a path prefix. Codes the zh chain cannot resolve fall back
// to the default chain, so zh mode always shows an image.
func (s *Server) ZhImageHandler() http.Handler {
	return s.imageHandler(func(code string) (*cachedImage, error) {
		if img, ok := s.ZhImages.peek(code); ok {
			return img, nil
		}
		if img, err := s.ZhImages.get(code); err == nil {
			return img, nil
		}
		return s.Images.get(code)
	})
}

func (s *Server) imageHandler(getImage func(code string) (*cachedImage, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		if name == "" || name == "." || name == "/" {
			writeErr(w, http.StatusBadRequest, "missing image name")
			return
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		code := stem
		wantHash := ""
		if i := strings.LastIndexByte(stem, '.'); i > 0 {
			code = stem[:i]
			wantHash = stem[i+1:]
		}
		if !validCardCode(code) || (wantHash != "" && !validHash(wantHash)) {
			writeErr(w, http.StatusBadRequest, "invalid image name")
			return
		}
		img, err := getImage(code)
		if err != nil {
			writeErr(w, http.StatusBadGateway, "card image unavailable: "+err.Error())
			return
		}
		if wantHash != "" && wantHash != img.hash {
			writeErr(w, http.StatusNotFound, "stale content hash")
			return
		}
		etag := `"` + img.hash + `"`
		w.Header().Set("Content-Type", img.mimeType)
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			// Cheap revalidation for un-hashed URLs.
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if wantHash != "" {
			// Content-addressed URL: cache forever.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(img.body)
	})
}

// ManifestHandler exposes the current {code: hash} manifest so clients can
// build content-addressed URLs.
func (s *Server) ManifestHandler() http.Handler {
	return s.manifestHandler(s.Images)
}

// ZhManifestHandler exposes the zh cache's manifest (mounted at
// /img/cards/zh/manifest.json); empty until Chinese faces are seeded.
func (s *Server) ZhManifestHandler() http.Handler {
	return s.manifestHandler(s.ZhImages)
}

func (s *Server) manifestHandler(c *imageCache) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := c.manifestJSON()
		etag := `"` + hashBytes(body) + `"`
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_, _ = w.Write(body)
	})
}

func validCardCode(code string) bool {
	if len(code) < 5 || len(code) > 6 {
		return false
	}
	for _, r := range code {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'c') {
			continue
		}
		return false
	}
	return true
}

func validHash(h string) bool {
	if len(h) != 16 {
		return false
	}
	for _, r := range h {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			continue
		}
		return false
	}
	return true
}
