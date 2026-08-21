package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "# comment\n\nR2_ENDPOINT=https://e.example.com\n R2_BUCKET = my-bucket \nBROKEN_LINE_NO_EQUALS\ncrlf=ok\r\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("R2_BUCKET", "from-real-env")
	if n, err := Load(path); err != nil || n != 2 {
		t.Fatalf("load = %d, %v (want 2 set)", n, err)
	}
	if os.Getenv("R2_ENDPOINT") != "https://e.example.com" {
		t.Fatalf("R2_ENDPOINT = %q", os.Getenv("R2_ENDPOINT"))
	}
	if got := os.Getenv("R2_BUCKET"); got != "from-real-env" {
		t.Fatalf("real env must win: R2_BUCKET = %q", got)
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
