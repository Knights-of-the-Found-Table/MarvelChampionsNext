// Package mirror resolves card-image download sources: plain HTTP mirrors
// serving marvelcdb's exact path layout (/bundles/cards/{code}.png — the
// mirror decides what bytes it returns, png, webp or otherwise; the MIME
// type is detected from content when served).
//
// Languages are separate sources, exactly like marvelcdb's own language
// domains: the default root (IMAGE_MIRROR, marvelcdb.com itself by
// default) and optionally a Chinese root (ZH_IMAGE_MIRROR) — never a path
// prefix. All access is strictly read-only.
package mirror

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ErrNotFound reports that a source definitively lacks the object (HTTP
// 404). A chain treats it as "try the next source"; network failures are
// not ErrNotFound.
var ErrNotFound = errors.New("not found")

const (
	defaultMarvelcdb = "https://marvelcdb.com"
	maxImageBytes    = 8 << 20
)

// Source fetches a card image by its marvelcdb-style path, e.g.
// "/bundles/cards/01001a.png". Implementations must be read-only.
type Source interface {
	Name() string
	Fetch(path string) ([]byte, error)
}

func fetchBody(client *http.Client, req *http.Request) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return io.ReadAll(io.LimitReader(resp.Body, maxImageBytes))
	}
	return nil, readError(resp)
}

// readError converts a non-200 response into an error, mapping 404 to
// ErrNotFound so chains can fall through to the next source.
func readError(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrNotFound, resp.Status)
	default:
		return fmt.Errorf("status %s", resp.Status)
	}
}

// HTTPSource fetches from a site root serving marvelcdb's path layout.
type HTTPSource struct {
	BaseURL string
	Client  *http.Client
}

func (s HTTPSource) Name() string { return "http:" + strings.TrimSuffix(s.BaseURL, "/") }

func (s HTTPSource) Fetch(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimSuffix(s.BaseURL, "/")+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "marvelchampionsnext/server")
	return fetchBody(s.Client, req)
}

// Chain returns a Source that tries the given sources in order and returns
// the first success. Failures are logged and the next source is tried; the
// last error is returned when all sources fail.
func Chain(sources ...Source) Source { return chain{sources: sources} }

type chain struct{ sources []Source }

func (c chain) Name() string {
	parts := make([]string, len(c.sources))
	for i, s := range c.sources {
		parts[i] = s.Name()
	}
	return strings.Join(parts, " -> ")
}

func (c chain) Fetch(path string) ([]byte, error) {
	if len(c.sources) == 0 {
		return nil, fmt.Errorf("%s: no image sources configured", path)
	}
	var lastErr error
	for _, src := range c.sources {
		body, err := src.Fetch(path)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if errors.Is(err, ErrNotFound) {
			log.Printf("images: %s missing at %s, trying next source", path, src.Name())
			continue
		}
		log.Printf("images: %s via %s: %v", path, src.Name(), err)
	}
	return nil, lastErr
}

// Env holds the per-language source chains.
type Env struct {
	// Default is the default-language chain: IMAGE_MIRROR or marvelcdb
	// itself. A single HTTP root.
	Default []Source
	// Zh is the Chinese chain: ZH_IMAGE_MIRROR, empty when unconfigured —
	// the zh cache then serves only locally seeded faces.
	Zh []Source
	// DefaultIsMirror reports whether Default points at a mirror rather
	// than bare marvelcdb, so bulk fetchers can rate-limit.
	DefaultIsMirror bool
}

// SourcesFromEnv resolves the source chains from environment variables:
// IMAGE_MIRROR (default https://marvelcdb.com) and ZH_IMAGE_MIRROR. Both
// serve marvelcdb's exact paths.
func SourcesFromEnv() Env {
	var env Env
	root := envOr("IMAGE_MIRROR", defaultMarvelcdb)
	env.Default = []Source{HTTPSource{BaseURL: root}}
	env.DefaultIsMirror = root != defaultMarvelcdb
	if v := os.Getenv("ZH_IMAGE_MIRROR"); v != "" {
		env.Zh = []Source{HTTPSource{BaseURL: v}}
	}
	return env
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
