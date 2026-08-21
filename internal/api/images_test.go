package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
type stubSource struct {
	body  []byte
	err   error
	calls int
}

func (s *stubSource) Name() string { return "stub" }

func (s *stubSource) Fetch(string) ([]byte, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.body, nil
}

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
	if missing.calls != 1 || serving.calls != 1 {
		t.Fatalf("calls: missing=%d serving=%d", missing.calls, serving.calls)
	}
}

// With Chinese mirror sources configured, zh misses are fetched from them;
// when they lack the image the English chain serves it without ever storing
// an English face in the zh cache.
func TestZhImageHandlerMirrorFetchAndFallback(t *testing.T) {
	enSrc := &stubSource{body: []byte("en-image")}
	en, err := NewImageCache(t.TempDir(), enSrc)
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{Images: en}

	get := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/img/cards/zh/01001a.png", nil)
		rec := httptest.NewRecorder()
		srv.ZhImageHandler().ServeHTTP(rec, req)
		return rec
	}

	// zh mirror has the image: served from the zh chain, English untouched.
	zhHit, err := NewImageCache(t.TempDir(), &stubSource{body: []byte("zh-image")})
	if err != nil {
		t.Fatal(err)
	}
	srv.ZhImages = zhHit
	if rec := get(); rec.Code != http.StatusOK || rec.Body.String() != "zh-image" {
		t.Fatalf("zh hit: status %d body %q", rec.Code, rec.Body.String())
	}
	if enSrc.calls != 0 {
		t.Fatalf("english source used %d times for a zh hit", enSrc.calls)
	}

	// zh mirror misses: English fallback, nothing written into the zh cache.
	zhDir := t.TempDir()
	zhMiss, err := NewImageCache(zhDir, &stubSource{err: mirror.ErrNotFound})
	if err != nil {
		t.Fatal(err)
	}
	srv.ZhImages = zhMiss
	if rec := get(); rec.Code != http.StatusOK || rec.Body.String() != "en-image" {
		t.Fatalf("en fallback: status %d body %q", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(zhDir, "01001a.img")); !os.IsNotExist(err) {
		t.Fatalf("zh cache stored an English face: %v", err)
	}
}

// Seeded faces win over network sources without a single fetch.
func TestZhImageHandlerSeededFaceWins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "01001a.img"), []byte("seeded-image"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"01001a": hashString([]byte("seeded-image"))})
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	network := &stubSource{body: []byte("network-image")}
	zh, err := NewImageCache(dir, network)
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{Images: zh, ZhImages: zh}

	req := httptest.NewRequest(http.MethodGet, "/img/cards/zh/01001a.png", nil)
	rec := httptest.NewRecorder()
	srv.ZhImageHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "seeded-image" {
		t.Fatalf("status %d body %q", rec.Code, rec.Body.String())
	}
	if network.calls != 0 {
		t.Fatalf("network source called %d times", network.calls)
	}
}
