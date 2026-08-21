package mirror

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPSource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bundles/cards/01001a.png" {
			_, _ = w.Write([]byte("en-image"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	src := HTTPSource{BaseURL: srv.URL + "/"} // trailing slash must be tolerated
	if body, err := src.Fetch("/bundles/cards/01001a.png"); err != nil || string(body) != "en-image" {
		t.Fatalf("fetch = %q, %v", body, err)
	}
	if _, err := src.Fetch("/bundles/cards/missing.png"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing image err = %v, want ErrNotFound", err)
	}
}

func TestR2Source(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotAuth = r.Header.Get("Authorization")
		if r.URL.EscapedPath() == "/my-bucket/bundles/cards/01001a.png" {
			_, _ = w.Write([]byte("r2-image"))
			return
		}
		if r.URL.EscapedPath() == "/my-bucket/forbidden.png" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	src := R2Source{
		Endpoint:        srv.URL,
		Bucket:          "my-bucket",
		AccessKeyID:     "akid",
		SecretAccessKey: "secret",
	}
	body, err := src.Fetch("/bundles/cards/01001a.png")
	if err != nil || string(body) != "r2-image" {
		t.Fatalf("fetch = %q, %v", body, err)
	}
	// Path-style addressing with the marvelcdb path appended verbatim.
	if gotPath != "/my-bucket/bundles/cards/01001a.png" {
		t.Fatalf("request path = %q", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "AWS4-HMAC-SHA256 Credential=akid/") || !strings.Contains(gotAuth, "SignedHeaders=host;x-amz-content-sha256;x-amz-date") {
		t.Fatalf("authorization = %q", gotAuth)
	}

	if _, err := src.Fetch("/bundles/cards/missing.png"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing object err = %v, want ErrNotFound", err)
	}
	if _, err := src.Fetch("/forbidden.png"); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("forbidden object err = %v, want a non-404 error", err)
	}

	// Keys needing percent-encoding stay encoded in the request path.
	_, _ = src.Fetch("/bundles/cards/a b.png")
	if gotPath != "/my-bucket/bundles/cards/a%20b.png" {
		t.Fatalf("encoded path = %q", gotPath)
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

// Paths from card data end in .png; Chinese mirrors keep the pack's .jpg
// filenames, so .png paths are requested as .jpg first with the original
// as fallback.
func TestPreferJpg(t *testing.T) {
	var requested []string
	var fetchFn func(path string) ([]byte, error)
	inner := stubFunc{fetch: func(path string) ([]byte, error) { return fetchFn(path) }}
	src := PreferJpg(inner)

	// Mirror with .jpg keys: the .png path is requested as .jpg, one GET.
	fetchFn = func(path string) ([]byte, error) {
		requested = append(requested, path)
		if strings.HasSuffix(path, ".jpg") {
			return []byte("jpg-image"), nil
		}
		return nil, ErrNotFound
	}
	if body, err := src.Fetch("/bundles/cards/01001a.png"); err != nil || string(body) != "jpg-image" {
		t.Fatalf("fetch = %q, %v", body, err)
	}
	if len(requested) != 1 || requested[0] != "/bundles/cards/01001a.jpg" {
		t.Fatalf("requested = %v", requested)
	}

	// Mirror without .jpg keys: falls back to the exact path.
	requested = nil
	fetchFn = func(path string) ([]byte, error) {
		requested = append(requested, path)
		return nil, ErrNotFound
	}
	if _, err := src.Fetch("/bundles/cards/01001a.png"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if len(requested) != 2 || requested[0] != "/bundles/cards/01001a.jpg" || requested[1] != "/bundles/cards/01001a.png" {
		t.Fatalf("requested = %v", requested)
	}

	// Non-.png paths pass through unchanged.
	requested = nil
	_, _ = src.Fetch("/bundles/cards/01001a.jpg")
	if len(requested) != 1 || requested[0] != "/bundles/cards/01001a.jpg" {
		t.Fatalf("requested = %v", requested)
	}
}

type stubFunc struct {
	fetch func(path string) ([]byte, error)
}

func (s stubFunc) Name() string { return "stub" }

func (s stubFunc) Fetch(path string) ([]byte, error) { return s.fetch(path) }

func TestSourcesFromEnv(t *testing.T) {
	for _, k := range []string{"R2_ACCESS_KEY_ID", "R2_SECRET_ACCESS_KEY", "R2_BUCKET", "R2_ENDPOINT", "CLOUDFLARE_ACCOUNT_ID", "MC_MARVELCDB_IMAGES"} {
		t.Setenv(k, "")
	}

	// No mirror: bare marvelcdb, fetchers should rate-limit.
	env := SourcesFromEnv()
	if len(env.Sources) != 1 || !env.DirectMarvelcdb {
		t.Fatalf("default: sources=%d direct=%v", len(env.Sources), env.DirectMarvelcdb)
	}

	// Full R2 config: one shared chain, mirror first, marvelcdb fallback.
	t.Setenv("R2_ACCESS_KEY_ID", "akid")
	t.Setenv("R2_SECRET_ACCESS_KEY", "secret")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "acct")
	t.Setenv("R2_BUCKET", "marvel-champion-cn")
	env = SourcesFromEnv()
	if len(env.Sources) != 2 || env.DirectMarvelcdb {
		t.Fatalf("r2: sources=%d direct=%v", len(env.Sources), env.DirectMarvelcdb)
	}
	wrapped, ok := env.Sources[0].(preferJpg)
	if !ok {
		t.Fatalf("sources[0] = %#v, want a jpg-preferring R2 source", env.Sources[0])
	}
	r2, ok := wrapped.src.(R2Source)
	if !ok || r2.Bucket != "marvel-champion-cn" || r2.Endpoint != "https://acct.r2.cloudflarestorage.com" {
		t.Fatalf("r2 = %#v", r2)
	}
	if http, ok := env.Sources[1].(HTTPSource); !ok || http.BaseURL != "https://marvelcdb.com" {
		t.Fatalf("sources[1] = %#v", env.Sources[1])
	}

	// Explicit R2_ENDPOINT wins over the account id.
	t.Setenv("R2_ENDPOINT", "https://custom.example.com")
	env = SourcesFromEnv()
	wrapped, ok = env.Sources[0].(preferJpg)
	if !ok {
		t.Fatal("sources[0] not preferJpg")
	}
	r2, _ = wrapped.src.(R2Source)
	if r2.Endpoint != "https://custom.example.com" {
		t.Fatalf("endpoint = %q", r2.Endpoint)
	}

	// English HTTP mirror replaces the marvelcdb root.
	t.Setenv("R2_ENDPOINT", "")
	t.Setenv("MC_MARVELCDB_IMAGES", "https://mirror.example.com")
	env = SourcesFromEnv()
	if len(env.Sources) != 2 || env.Sources[1].Name() != "http:https://mirror.example.com" || env.DirectMarvelcdb {
		t.Fatalf("sources = %#v", env.Sources)
	}

	// Partial R2 config: mirror off, chain back to the bare root.
	t.Setenv("R2_BUCKET", "")
	env = SourcesFromEnv()
	if len(env.Sources) != 1 {
		t.Fatalf("partial config sources = %#v", env.Sources)
	}
}
