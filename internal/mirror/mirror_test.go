package mirror

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestHTTPSource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bundles/cards/01001a.png" {
			_, _ = w.Write([]byte("mirror-image"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	src := HTTPSource{BaseURL: srv.URL + "/"} // trailing slash must be tolerated
	if body, err := src.Fetch("/bundles/cards/01001a.png"); err != nil || string(body) != "mirror-image" {
		t.Fatalf("fetch = %q, %v", body, err)
	}
	if _, err := src.Fetch("/bundles/cards/missing.png"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing image err = %v, want ErrNotFound", err)
	}
}

type fakeSource struct {
	name  string
	body  []byte
	err   error
	calls *int
}

func (f fakeSource) Name() string { return f.name }

func (f fakeSource) Fetch(string) ([]byte, error) {
	if f.calls != nil {
		*f.calls++
	}
	return f.body, f.err
}

// funcSource routes every fetch through a test-supplied function.
type funcSource struct {
	name  string
	fetch func(path string) ([]byte, error)
}

func (f funcSource) Name() string { return f.name }

func (f funcSource) Fetch(path string) ([]byte, error) { return f.fetch(path) }

func TestChainFallsThrough(t *testing.T) {
	firstCalls, secondCalls := 0, 0
	first := fakeSource{name: "first", err: ErrNotFound, calls: &firstCalls}
	second := fakeSource{name: "second", body: []byte("img"), calls: &secondCalls}

	got, err := Chain(first, second).Fetch("/bundles/cards/01001a.png")
	if err != nil || string(got) != "img" {
		t.Fatalf("chain fetch = %q, %v", got, err)
	}
	if firstCalls != 1 || secondCalls != 1 {
		t.Fatalf("calls = first:%d second:%d", firstCalls, secondCalls)
	}

	// A non-404 error on the first source also falls through to the second.
	failing := fakeSource{name: "failing", err: errors.New("boom")}
	got, err = Chain(failing, second).Fetch("/bundles/cards/01001a.png")
	if err != nil || string(got) != "img" {
		t.Fatalf("chain fetch after error = %q, %v", got, err)
	}
}

func TestChainExhausted(t *testing.T) {
	failing := fakeSource{name: "failing", err: errors.New("boom")}
	if _, err := Chain(failing).Fetch("/bundles/cards/01001a.png"); err == nil || err.Error() != "boom" {
		t.Fatalf("err = %v, want boom", err)
	}
	if _, err := Chain().Fetch("/bundles/cards/01001a.png"); err == nil {
		t.Fatal("empty chain must error")
	}
}

// The exact path is tried first; a 404 retries the other extensions
// (.png/.jpg/.webp) in order.
func TestTryExtensions(t *testing.T) {
	var requested []string
	var fetchFn func(path string) ([]byte, error)
	src := TryExtensions(funcSource{name: "mirror", fetch: func(path string) ([]byte, error) {
		requested = append(requested, path)
		return fetchFn(path)
	}})

	// Exact hit: no extra requests.
	fetchFn = func(path string) ([]byte, error) {
		if path == "/bundles/cards/01001a.png" {
			return []byte("img"), nil
		}
		return nil, ErrNotFound
	}
	if body, err := src.Fetch("/bundles/cards/01001a.png"); err != nil || string(body) != "img" {
		t.Fatalf("exact fetch = %q, %v", body, err)
	}
	if len(requested) != 1 {
		t.Fatalf("requested = %v", requested)
	}

	// .png missing: .jpg is retried before .webp.
	requested = nil
	fetchFn = func(path string) ([]byte, error) {
		if strings.HasSuffix(path, ".jpg") {
			return []byte("jpg"), nil
		}
		return nil, ErrNotFound
	}
	if body, err := src.Fetch("/bundles/cards/01001a.png"); err != nil || string(body) != "jpg" {
		t.Fatalf("jpg fetch = %q, %v", body, err)
	}
	want := []string{"/bundles/cards/01001a.png", "/bundles/cards/01001a.jpg"}
	if !reflect.DeepEqual(requested, want) {
		t.Fatalf("requested = %v, want %v", requested, want)
	}

	// Only .webp exists.
	requested = nil
	fetchFn = func(path string) ([]byte, error) {
		if strings.HasSuffix(path, ".webp") {
			return []byte("webp"), nil
		}
		return nil, ErrNotFound
	}
	if body, err := src.Fetch("/bundles/cards/01001a.png"); err != nil || string(body) != "webp" {
		t.Fatalf("webp fetch = %q, %v", body, err)
	}

	// A genuine .jpg path retries .png then .webp, deduplicated.
	requested = nil
	fetchFn = func(path string) ([]byte, error) { return nil, ErrNotFound }
	if _, err := src.Fetch("/bundles/cards/01001b.jpg"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	want = []string{"/bundles/cards/01001b.jpg", "/bundles/cards/01001b.png", "/bundles/cards/01001b.webp"}
	if !reflect.DeepEqual(requested, want) {
		t.Fatalf("requested = %v, want %v", requested, want)
	}

	// Non-404 failures surface immediately without extension retries.
	requested = nil
	fetchFn = func(path string) ([]byte, error) { return nil, errors.New("boom") }
	if _, err := src.Fetch("/bundles/cards/01001a.png"); err == nil || err.Error() != "boom" {
		t.Fatalf("err = %v, want boom", err)
	}
	if len(requested) != 1 {
		t.Fatalf("requested = %v", requested)
	}
}

func TestSourcesFromEnv(t *testing.T) {
	for _, k := range []string{"IMAGE_MIRROR", "ZH_IMAGE_MIRROR"} {
		t.Setenv(k, "")
	}

	unwrap := func(s Source) HTTPSource {
		wrapped, ok := s.(tryExtensions)
		if !ok {
			t.Fatalf("source %#v is not extension-fallback wrapped", s)
		}
		http, ok := wrapped.src.(HTTPSource)
		if !ok {
			t.Fatalf("inner source %#v is not HTTPSource", wrapped.src)
		}
		return http
	}

	// No mirrors: default = bare marvelcdb, zh unconfigured.
	env := SourcesFromEnv()
	if len(env.Default) != 1 || env.DefaultIsMirror || len(env.Zh) != 0 {
		t.Fatalf("default: default=%d zh=%d mirror=%v", len(env.Default), len(env.Zh), env.DefaultIsMirror)
	}
	if http := unwrap(env.Default[0]); http.BaseURL != "https://marvelcdb.com" {
		t.Fatalf("default[0] = %#v", env.Default[0])
	}

	// Both language mirrors configured.
	t.Setenv("IMAGE_MIRROR", "https://images.example.com")
	t.Setenv("ZH_IMAGE_MIRROR", "https://zh-images.example.com")
	env = SourcesFromEnv()
	if !env.DefaultIsMirror {
		t.Fatal("custom default root must be reported as a mirror")
	}
	if http := unwrap(env.Default[0]); http.BaseURL != "https://images.example.com" {
		t.Fatalf("default[0] = %#v", env.Default[0])
	}
	if http := unwrap(env.Zh[0]); http.BaseURL != "https://zh-images.example.com" {
		t.Fatalf("zh[0] = %#v", env.Zh[0])
	}

	// zh only.
	t.Setenv("IMAGE_MIRROR", "")
	env = SourcesFromEnv()
	if env.DefaultIsMirror || len(env.Zh) != 1 {
		t.Fatalf("zh-only: mirror=%v zh=%d", env.DefaultIsMirror, len(env.Zh))
	}
}
