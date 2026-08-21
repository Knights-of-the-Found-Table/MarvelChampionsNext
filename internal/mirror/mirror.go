// Package mirror resolves card-image download sources.
//
// There is a single source chain shared by everything the server serves:
// the R2 mirror first when configured, then the HTTP root
// (MC_MARVELCDB_IMAGES or marvelcdb.com itself). The mirror serves
// marvelcdb's exact path layout (/bundles/cards/{code}.png|jpg) — its
// content defines what gets distributed: a bucket holding the Chinese
// card pack makes every served image Chinese, with no language-specific
// source or path anywhere in the server. The Chinese pack keeps original
// .jpg filenames, so paths from card data (.png) are retried as .jpg.
//
// All access is strictly read-only.
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

// ErrNotFound reports that a source definitively lacks the object (HTTP 404
// / S3 NoSuchKey). A chain treats it as "try the next source"; network and
// auth failures are not ErrNotFound.
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

// readError converts a non-200 response into an error, mapping 404 (S3
// NoSuchKey) to ErrNotFound so chains can fall through to the next source.
func readError(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrNotFound, resp.Status)
	case http.StatusForbidden, http.StatusUnauthorized:
		return fmt.Errorf("auth failed (%s): check the mirror credentials", resp.Status)
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

// R2Source fetches from a private Cloudflare R2 bucket through the
// S3-compatible API: path-style addressing ({endpoint}/{bucket}/{key}) with
// a SigV4-signed GET. Read-only — no write or delete calls exist here.
type R2Source struct {
	Endpoint        string // e.g. https://{account}.r2.cloudflarestorage.com
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string

	Client *http.Client
}

func (s R2Source) Name() string { return "r2:" + s.Bucket }

func (s R2Source) Fetch(path string) ([]byte, error) {
	key := awsURIEncode(strings.TrimPrefix(path, "/"), false)
	req, err := http.NewRequest(http.MethodGet, strings.TrimSuffix(s.Endpoint, "/")+"/"+s.Bucket+"/"+key, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "marvelchampionsnext/server")
	signer := sigv4Signer{
		accessKeyID:     s.AccessKeyID,
		secretAccessKey: s.SecretAccessKey,
		region:          r2DefaultRegion,
		service:         "s3",
	}
	signer.sign(req, emptyPayloadSHA, time.Now())
	return fetchBody(s.Client, req)
}

// PreferJpg wraps a source for Chinese community mirrors: the card pack
// keeps original .jpg filenames while paths from card data end in .png, so
// a .png path is requested as .jpg first and the exact given path is the
// fallback. Non-.png paths pass through unchanged.
func PreferJpg(src Source) Source { return preferJpg{src: src} }

type preferJpg struct{ src Source }

func (p preferJpg) Name() string { return p.src.Name() }

func (p preferJpg) Fetch(path string) ([]byte, error) {
	if strings.HasSuffix(path, ".png") {
		if body, err := p.src.Fetch(strings.TrimSuffix(path, ".png") + ".jpg"); err == nil {
			return body, nil
		}
	}
	return p.src.Fetch(path)
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

// Env describes the resolved image sources.
type Env struct {
	// Sources is the fetch chain, used for every image the server
	// distributes: the R2 mirror first when configured, then the HTTP
	// root (MC_MARVELCDB_IMAGES or marvelcdb.com). Whatever the mirror
	// holds is what gets served — the Chinese pack, naturally.
	Sources []Source
	// DirectMarvelcdb is true when the chain is bare marvelcdb with no
	// mirror in front, so bulk fetchers can rate-limit.
	DirectMarvelcdb bool
}

// SourcesFromEnv resolves the source chain from environment variables:
// the R2 mirror (R2_ACCESS_KEY_ID + R2_SECRET_ACCESS_KEY + R2_BUCKET and
// an endpoint — R2_ENDPOINT, or CLOUDFLARE_ACCOUNT_ID from which
// https://{id}.r2.cloudflarestorage.com is derived) followed by the HTTP
// root (MC_MARVELCDB_IMAGES, default https://marvelcdb.com). The mirror
// serves marvelcdb's exact paths and is used directly as the preferred
// image source; the HTTP root only fills in codes the mirror lacks.
func SourcesFromEnv() Env {
	var env Env
	root := envOr("MC_MARVELCDB_IMAGES", defaultMarvelcdb)
	env.Sources = []Source{HTTPSource{BaseURL: root}}
	env.DirectMarvelcdb = root == defaultMarvelcdb

	if r2, ok := r2ConfigFromEnv(); ok {
		env.Sources = []Source{PreferJpg(r2), HTTPSource{BaseURL: root}}
		env.DirectMarvelcdb = false
	} else if r2EnvPartiallySet() {
		log.Printf("images: R2_* environment partially set; mirror disabled " +
			"(needs R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_BUCKET and R2_ENDPOINT or CLOUDFLARE_ACCOUNT_ID)")
	}
	return env
}

func r2ConfigFromEnv() (R2Source, bool) {
	s := R2Source{
		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		Bucket:          os.Getenv("R2_BUCKET"),
		Endpoint:        os.Getenv("R2_ENDPOINT"),
	}
	if s.Endpoint == "" {
		if acct := os.Getenv("CLOUDFLARE_ACCOUNT_ID"); acct != "" {
			s.Endpoint = "https://" + acct + ".r2.cloudflarestorage.com"
		}
	}
	if s.AccessKeyID == "" || s.SecretAccessKey == "" || s.Bucket == "" || s.Endpoint == "" {
		return R2Source{}, false
	}
	return s, true
}

func r2EnvPartiallySet() bool {
	for _, k := range []string{"R2_ACCESS_KEY_ID", "R2_SECRET_ACCESS_KEY", "R2_BUCKET", "R2_ENDPOINT", "CLOUDFLARE_ACCOUNT_ID"} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
