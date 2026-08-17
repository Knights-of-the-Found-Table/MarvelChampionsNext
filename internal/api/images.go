package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// imageCache serves card images with on-demand fetching: the first request
// for a card downloads it from marvelcdb into the cache directory, records
// its content hash in manifest.json, and every later request is served
// locally. Hash-versioned URLs (?v=<hash>) get immutable caching, exactly
// like the previous build-time pipeline.
type imageCache struct {
	dir      string
	imageURL string // marvelcdb site root
	client   *http.Client

	mu       sync.Mutex
	manifest map[string]string
}

// NewImageCache builds an on-demand image cache rooted at dir.
func NewImageCache(dir string) (*imageCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	c := &imageCache{
		dir:      dir,
		imageURL: envOrImage("MC_MARVELCDB_IMAGES", "https://marvelcdb.com"),
		client:   &http.Client{Timeout: 60 * time.Second},
		manifest: map[string]string{},
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "manifest.json")); err == nil {
		_ = json.Unmarshal(raw, &c.manifest)
	}
	return c, nil
}

func envOrImage(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
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
	body, err := c.fetch(remote)
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

func (c *imageCache) imagePath(code string) string {
	return filepath.Join(c.dir, code+".img")
}

func (c *imageCache) remotePath(code string) string {
	if def, ok := engine.DB.Lookup(code); ok && def.ImageSrc != "" {
		return def.ImageSrc
	}
	return "/bundles/cards/" + code + ".png"
}

func (c *imageCache) fetch(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.imageURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "marvelchampionsnext/server")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
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

// ImageHandler serves card images on demand with immutable caching for
// hash-versioned URLs.
func (s *Server) ImageHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Base(r.URL.Path)
		if name == "" || name == "." || name == "/" {
			writeErr(w, http.StatusBadRequest, "missing image name")
			return
		}
		code := strings.TrimSuffix(name, filepath.Ext(name))
		if !validCardCode(code) {
			writeErr(w, http.StatusBadRequest, "invalid card code")
			return
		}
		img, err := s.Images.get(code)
		if err != nil {
			writeErr(w, http.StatusBadGateway, "card image unavailable: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", img.mimeType)
		w.Header().Set("ETag", `"`+img.hash+`"`)
		if r.URL.Query().Get("v") != "" {
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
// build immutable URLs.
func (s *Server) ManifestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(s.Images.manifestJSON())
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
