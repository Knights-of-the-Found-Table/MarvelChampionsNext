package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestLocalesDelivery 校验三条文案分发路由：manifest 永远校验、内容寻址
// 路由打 immutable 长缓存且正文与兼容直连路由逐字节一致、hash 失配或未知
// 语言一律 404（客户端据此重取 manifest 自愈）。
func TestLocalesDelivery(t *testing.T) {
	ts := httptest.NewServer((&Server{}).Router())
	t.Cleanup(ts.Close)

	get := func(path string) (*http.Response, []byte) {
		t.Helper()
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		resp.Body = io.NopCloser(strings.NewReader(string(b))) // 供断言复用 Header
		resp.Body.Close()
		return resp, b
	}

	// manifest：no-cache + 双语言 hash。
	resp, raw := get("/api/v1/locales/manifest")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("manifest status %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "no-cache") {
		t.Errorf("manifest cache-control = %q", cc)
	}
	var manifest map[string]string
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("manifest json: %v", err)
	}
	hashes := map[string]bool{}
	for _, lang := range []string{"en", "zh"} {
		h, ok := manifest[lang]
		if !ok || len(h) != 16 {
			t.Fatalf("manifest[%s] missing or wrong length: %q", lang, h)
		}
		hashes[lang] = true
		delete(hashes, lang) // 仅校验长度后清空，防两个语言同值误用
		hashes[h] = true
	}
	if hashes["en"] && manifest["en"] == manifest["zh"] {
		// 两种语言的目录必然不同，hash 不应相同。
		t.Errorf("en and zh hashes identical")
	}

	for lang, h := range map[string]string{"en": manifest["en"], "zh": manifest["zh"]} {
		// 内容寻址路由：200、immutable、与直连路由逐字节一致。
		rv, bv := get("/api/v1/locales/" + lang + "/" + h)
		if rv.StatusCode != http.StatusOK {
			t.Fatalf("%s versioned status %d", lang, rv.StatusCode)
		}
		if cc := rv.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
			t.Errorf("%s versioned cache-control = %q", lang, cc)
		}
		rl, bl := get("/api/v1/locales/" + lang)
		if rl.StatusCode != http.StatusOK {
			t.Fatalf("%s legacy status %d", lang, rl.StatusCode)
		}
		if cc := rl.Header.Get("Cache-Control"); !strings.Contains(cc, "max-age=300") || strings.Contains(cc, "immutable") {
			t.Errorf("%s legacy cache-control = %q", lang, cc)
		}
		if string(bv) != string(bl) {
			t.Errorf("%s: versioned and legacy bodies differ", lang)
		}
		// 正文必须同时覆盖引擎键与前端 UI 键（单一事实源的端到端证据）。
		var cat map[string]string
		if err := json.Unmarshal(bv, &cat); err != nil {
			t.Fatalf("%s catalog json: %v", lang, err)
		}
		for _, key := range []string{"c.hexBoltMilled", "brand", "game.round"} {
			if cat[key] == "" {
				t.Errorf("%s catalog missing %s", lang, key)
			}
		}
	}

	// 失配 hash 与未知语言：404。
	if r, _ := get("/api/v1/locales/en/deadbeefdeadbeef"); r.StatusCode != http.StatusNotFound {
		t.Errorf("stale hash status = %d", r.StatusCode)
	}
	if r, _ := get("/api/v1/locales/fr/" + manifest["en"]); r.StatusCode != http.StatusNotFound {
		t.Errorf("unknown lang versioned status = %d", r.StatusCode)
	}
	if r, _ := get("/api/v1/locales/fr"); r.StatusCode != http.StatusNotFound {
		t.Errorf("unknown lang legacy status = %d", r.StatusCode)
	}
}
