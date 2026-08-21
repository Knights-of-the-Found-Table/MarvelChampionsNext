package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/mirror"
)

// seedImage places a fake card image in a fresh cache dir and returns it.
func seedImage(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	body := []byte("fake-image-bytes")
	hash := hashString(body)
	if err := os.WriteFile(filepath.Join(dir, "01001a.img"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]string{"01001a": hash}
	raw, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	images, err := NewImageCache(dir)
	if err != nil {
		t.Fatal(err)
	}
	return &Server{Images: images}, hash
}

func hashString(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])[:16]
}

func TestImageHandlerHashInPath(t *testing.T) {
	srv, hash := seedImage(t)
	h := srv.ImageHandler()

	get := func(url string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// Content-addressed path: immutable caching.
	rec := get("/img/cards/01001a." + hash + ".png")
	if rec.Code != http.StatusOK {
		t.Fatalf("hashed URL: %d %s", rec.Code, rec.Body.String())
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Fatalf("hashed URL cache-control = %q", cc)
	}
	if rec.Body.String() != "fake-image-bytes" {
		t.Fatal("wrong body")
	}

	// Unversioned path: no-cache with ETag.
	rec = get("/img/cards/01001a.png")
	if rec.Code != http.StatusOK {
		t.Fatalf("plain URL: %d", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("plain URL cache-control = %q", cc)
	}
	if rec.Header().Get("ETag") != `"`+hash+`"` {
		t.Fatalf("etag = %q", rec.Header().Get("ETag"))
	}

	// Stale hash: 404 so the client refetches the manifest.
	rec = get("/img/cards/01001a.deadbeefdeadbeef.png")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("stale hash: %d", rec.Code)
	}

	// Invalid names are rejected.
	for _, url := range []string{"/img/cards/../etc.png", "/img/cards/zzz.png"} {
		if rec := get(url); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d", url, rec.Code)
		}
	}
}

func TestManifestHandler(t *testing.T) {
	srv, hash := seedImage(t)
	req := httptest.NewRequest(http.MethodGet, "/img/cards/manifest.json", nil)
	rec := httptest.NewRecorder()
	srv.ManifestHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var m map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["01001a"] != hash {
		t.Fatalf("manifest = %v", m)
	}
	if rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatal("manifest must be no-cache")
	}
}

// stubSource is a canned image source for exercising the source chain.
// Concurrent-safe: PrewarmImages exercises sources from worker goroutines.
type stubSource struct {
	body  []byte
	err   error
	calls int64
}

func (s *stubSource) Name() string { return "stub" }

func (s *stubSource) Fetch(string) ([]byte, error) {
	atomic.AddInt64(&s.calls, 1)
	if s.err != nil {
		return nil, s.err
	}
	return s.body, nil
}

func (s *stubSource) callCount() int64 { return atomic.LoadInt64(&s.calls) }

// A miss at the first source (404) falls through to the next one.
func TestImageCacheFallsBackAcrossSources(t *testing.T) {
	missing := &stubSource{err: mirror.ErrNotFound}
	serving := &stubSource{body: []byte("mirrored-image")}
	images, err := NewImageCache(t.TempDir(), missing, serving)
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{Images: images}

	req := httptest.NewRequest(http.MethodGet, "/img/cards/01006a.png", nil)
	rec := httptest.NewRecorder()
	srv.ImageHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "mirrored-image" {
		t.Fatalf("status %d body %q", rec.Code, rec.Body.String())
	}
	if missing.callCount() != 1 || serving.callCount() != 1 {
		t.Fatalf("calls: missing=%d serving=%d", missing.callCount(), serving.callCount())
	}
}

// zh routes: seeded faces first (untouched by the network), then the zh
// chain (ZH_IMAGE_MIRROR); codes the zh chain lacks fall back to the
// default chain without storing anything in the zh directory.
func TestZhImageHandlerSeededMirrorAndFallback(t *testing.T) {
	zhDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(zhDir, "01001a.img"), []byte("seeded-image"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"01001a": hashString([]byte("seeded-image"))})
	if err := os.WriteFile(filepath.Join(zhDir, "manifest.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	zhSrc := &stubSource{body: []byte("zh-mirror-image")}
	zh, err := NewImageCache(zhDir, zhSrc)
	if err != nil {
		t.Fatal(err)
	}
	enSrc := &stubSource{body: []byte("default-image")}
	en, err := NewImageCache(t.TempDir(), enSrc)
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{Images: en, ZhImages: zh}

	get := func(url string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		srv.ZhImageHandler().ServeHTTP(rec, req)
		return rec
	}

	// Seeded face wins without a single fetch.
	if rec := get("/img/cards/zh/01001a.png"); rec.Code != http.StatusOK || rec.Body.String() != "seeded-image" {
		t.Fatalf("seeded: status %d body %q", rec.Code, rec.Body.String())
	}
	if zhSrc.callCount() != 0 || enSrc.callCount() != 0 {
		t.Fatalf("fetches for a seeded face: zh=%d en=%d", zhSrc.callCount(), enSrc.callCount())
	}

	// Unseeded code present at the zh mirror: fetched from it.
	if rec := get("/img/cards/zh/01006a.png"); rec.Code != http.StatusOK || rec.Body.String() != "zh-mirror-image" {
		t.Fatalf("zh mirror: status %d body %q", rec.Code, rec.Body.String())
	}
	if enSrc.callCount() != 0 {
		t.Fatalf("default chain used %d times for a zh mirror hit", enSrc.callCount())
	}
}

// When the zh chain cannot resolve a code (mirror not configured or
// missing), the default chain serves it and nothing lands in the zh dir.
func TestZhImageHandlerDefaultFallback(t *testing.T) {
	zhDir := t.TempDir()
	zh, err := NewImageCache(zhDir, &stubSource{err: mirror.ErrNotFound})
	if err != nil {
		t.Fatal(err)
	}
	en, err := NewImageCache(t.TempDir(), &stubSource{body: []byte("default-image")})
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{Images: en, ZhImages: zh}

	req := httptest.NewRequest(http.MethodGet, "/img/cards/zh/01006a.png", nil)
	rec := httptest.NewRecorder()
	srv.ZhImageHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "default-image" {
		t.Fatalf("fallback: status %d body %q", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(zhDir, "01006a.img")); !os.IsNotExist(err) {
		t.Fatalf("zh cache stored a default-chain image: %v", err)
	}

	// No zh sources at all: get errors out fast and the default chain serves.
	bare, err := NewImageCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv.ZhImages = bare
	rec = httptest.NewRecorder()
	srv.ZhImageHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/img/cards/zh/01007a.png", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "default-image" {
		t.Fatalf("no-sources fallback: status %d body %q", rec.Code, rec.Body.String())
	}
}

// Mirrors may serve webp (or jpeg) bytes under the .png paths; the MIME
// type is sniffed from content, not from the URL.
func TestDetectImageWebP(t *testing.T) {
	body := append([]byte(nil), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	copy(body, "RIFF")
	copy(body[8:], "WEBPVP8 ")
	if got := detectImage(body); got != "image/webp" {
		t.Fatalf("detectImage(webp) = %q", got)
	}
}

func TestImageHandlerETagRevalidation(t *testing.T) {
	srv, hash := seedImage(t)
	h := srv.ImageHandler()
	etag := `"` + hash + `"`

	req := httptest.NewRequest(http.MethodGet, "/img/cards/01001a.png", nil)
	req.Header.Set("If-None-Match", etag)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotModified || rec.Body.Len() != 0 {
		t.Fatalf("matching etag: %d %q", rec.Code, rec.Body.String())
	}

	req.Header.Set("If-None-Match", `"deadbeefdeadbeef"`)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "fake-image-bytes" {
		t.Fatalf("stale etag: %d %q", rec.Code, rec.Body.String())
	}
}

func TestManifestHandlerETag(t *testing.T) {
	srv, _ := seedImage(t)
	h := srv.ManifestHandler()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/img/cards/manifest.json", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") == "" {
		t.Fatalf("first: %d etag=%q", rec.Code, rec.Header().Get("ETag"))
	}

	req := httptest.NewRequest(http.MethodGet, "/img/cards/manifest.json", nil)
	req.Header.Set("If-None-Match", rec.Header().Get("ETag"))
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusNotModified || rec2.Body.Len() != 0 {
		t.Fatalf("revalidation: %d %q", rec2.Code, rec2.Body.String())
	}
}

func TestPrewarmImages(t *testing.T) {
	src := &stubSource{body: []byte("prewarmed")}
	images, err := NewImageCache(t.TempDir(), src)
	if err != nil {
		t.Fatal(err)
	}

	cached, failed := PrewarmImages(images, []string{"01001a", "01006a", "01007"}, 2, 0)
	if cached != 3 || failed != 0 {
		t.Fatalf("prewarm = %d cached, %d failed", cached, failed)
	}
	for _, code := range []string{"01001a", "01006a", "01007"} {
		if img, ok := images.peek(code); !ok || string(img.body) != "prewarmed" {
			t.Fatalf("%s not in cache after prewarm", code)
		}
	}
	if string(images.manifestJSON()) == "{}" {
		t.Fatal("manifest empty after prewarm")
	}

	failing, err := NewImageCache(t.TempDir(), &stubSource{err: errors.New("boom")})
	if err != nil {
		t.Fatal(err)
	}
	cached, failed = PrewarmImages(failing, []string{"01001a", "01006a"}, 2, 0)
	if cached != 0 || failed != 2 {
		t.Fatalf("failing prewarm = %d cached, %d failed", cached, failed)
	}
}
