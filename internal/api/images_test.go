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
