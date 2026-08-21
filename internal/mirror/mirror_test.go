package mirror

import (
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestSourcesFromEnv(t *testing.T) {
	for _, k := range []string{"IMAGE_MIRROR", "ZH_IMAGE_MIRROR"} {
		t.Setenv(k, "")
	}

	// No mirrors: default = bare marvelcdb, zh unconfigured.
	env := SourcesFromEnv()
	if len(env.Default) != 1 || env.DefaultIsMirror || len(env.Zh) != 0 {
		t.Fatalf("default: default=%d zh=%d mirror=%v", len(env.Default), len(env.Zh), env.DefaultIsMirror)
	}
	if http, ok := env.Default[0].(HTTPSource); !ok || http.BaseURL != "https://marvelcdb.com" {
		t.Fatalf("default[0] = %#v", env.Default[0])
	}

	// Both language mirrors configured.
	t.Setenv("IMAGE_MIRROR", "https://images.example.com")
	t.Setenv("ZH_IMAGE_MIRROR", "https://zh-images.example.com")
	env = SourcesFromEnv()
	if !env.DefaultIsMirror {
		t.Fatal("custom default root must be reported as a mirror")
	}
	if http, ok := env.Default[0].(HTTPSource); !ok || http.BaseURL != "https://images.example.com" {
		t.Fatalf("default[0] = %#v", env.Default[0])
	}
	if http, ok := env.Zh[0].(HTTPSource); !ok || http.BaseURL != "https://zh-images.example.com" {
		t.Fatalf("zh[0] = %#v", env.Zh[0])
	}

	// zh only.
	t.Setenv("IMAGE_MIRROR", "")
	env = SourcesFromEnv()
	if env.DefaultIsMirror || len(env.Zh) != 1 {
		t.Fatalf("zh-only: mirror=%v zh=%d", env.DefaultIsMirror, len(env.Zh))
	}
}
