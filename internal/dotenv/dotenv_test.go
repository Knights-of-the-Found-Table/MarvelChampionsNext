package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "# comment\n\nZH_IMAGE_MIRROR=https://zh.example.com\n IMAGE_MIRROR = https://en.example.com \nBROKEN_LINE_NO_EQUALS\ncrlf=ok\r\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("IMAGE_MIRROR", "from-real-env")
	if n, err := Load(path); err != nil || n != 2 {
		t.Fatalf("load = %d, %v (want 2 set)", n, err)
	}
	if os.Getenv("ZH_IMAGE_MIRROR") != "https://zh.example.com" {
		t.Fatalf("ZH_IMAGE_MIRROR = %q", os.Getenv("ZH_IMAGE_MIRROR"))
	}
	if got := os.Getenv("IMAGE_MIRROR"); got != "from-real-env" {
		t.Fatalf("real env must win: IMAGE_MIRROR = %q", got)
	}
	if os.Getenv("crlf") != "ok" {
		t.Fatalf("crlf = %q", os.Getenv("crlf"))
	}
}

func TestLoadMissingFile(t *testing.T) {
	if n, err := Load(filepath.Join(t.TempDir(), "nope.env")); err != nil || n != 0 {
		t.Fatalf("missing file: %d, %v", n, err)
	}
}
