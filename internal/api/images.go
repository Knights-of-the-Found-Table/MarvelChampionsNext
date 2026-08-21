package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/mirror"
)

// imageCache serves card images with on-demand fetching: the first request
// for a card downloads it through the configured source chain (R2/HTTP
// mirrors first, marvelcdb last) into the cache directory, records its
// content hash in manifest.json, and every later request is served locally.
// Content-addressed URLs (/img/cards/{code}.{hash}.png) get immutable
// caching, exactly like the previous build-time pipeline.
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

// get returns an image for a card code, fetching and caching on miss.
func (c *imageCache) get(code string) (*cachedImage, error) {
	if hash, ok := c.manifest[code]; ok {
		if body, err := os.ReadFile(c.imagePath(code)); err == nil {
			return &cachedImage{body: body, mimeType: detectImage(body), hash: hash}, nil
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// Double-check after acquiring the lock.
	if hash, ok := c.manifest[code]; ok {
		if body, err := os.ReadFile(c.imagePath(code)); err == nil {
			return &cachedImage{body: body, mimeType: detectImage(body), hash: hash}, nil
		}
	}
	remote := c.remotePath(code)
	body, err := c.source.Fetch(remote)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", remote, err)
	}
	hash := hashBytes(body)
	if err := os.WriteFile(c.imagePath(code), body, 0o644); err != nil {
		return nil, err
	}
	c.manifest[code] = hash
	c.saveManifest()
	return &cachedImage{body: body, mimeType: detectImage(body), hash: hash}, nil
}

// peek returns the cached image for a code without ever touching the
// network. Used for the zh cache so locally seeded faces win over the
// shared source chain.
func (c *imageCache) peek(code string) (*cachedImage, bool) {
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
	// English faces: fetch through the source chain on miss (R2/HTTP
	// mirrors first, marvelcdb last).
	return s.imageHandler(func(code string) (*cachedImage, error) {
		return s.Images.get(code)
	})
}

// ZhImageHandler serves Chinese card faces from the zh cache (mounted at
// /img/cards/zh/). Locally seeded faces first; misses are fetched through
// the same source chain as the English cache — the mirror when configured
// holds the Chinese pack, and its fallback may store an English face here,
// which is no different from what the English cache would serve. A chain
// with nothing at all falls back to the English cache.
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
		w.Header().Set("Content-Type", img.mimeType)
		w.Header().Set("ETag", `"`+img.hash+`"`)
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
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(c.manifestJSON())
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
