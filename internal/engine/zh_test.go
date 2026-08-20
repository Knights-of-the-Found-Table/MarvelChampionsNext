package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func writePack(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestApplyChinese(t *testing.T) {
	db, err := data.Load()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writePack(t, dir, "core.json", `{
		"01001a": {"name": "蜘蛛侠", "text": "<b>响应</b>：测试。", "traits": "复仇者。"},
		"99999": {"name": "不存在的卡"}
	}`)
	// A BOM-prefixed file must still decode (Windows tooling adds them).
	if err := os.WriteFile(filepath.Join(dir, "bom.json"), append([]byte("\xef\xbb\xbf"), []byte(`{"01002": {"name": "黑猫"}}`)...), 0o644); err != nil {
		t.Fatal(err)
	}
	// A subdirectory must be skipped, not decoded.
	if err := os.Mkdir(filepath.Join(dir, "chunks"), 0o755); err != nil {
		t.Fatal(err)
	}
	writePack(t, dir, "chunks/skipme.json", `{not valid json`)

	n, err := ApplyChinese(db, dir)
	if err != nil {
		t.Fatalf("ApplyChinese: %v", err)
	}
	if n != 2 {
		t.Fatalf("overlaid = %d, want 2", n)
	}
	if got := db.MustLookup("01002").Name; got != "黑猫" {
		t.Errorf("BOM file Name = %q, want 黑猫", got)
	}
	def := db.MustLookup("01001a")
	if def.Name != "蜘蛛侠" {
		t.Errorf("Name = %q, want 蜘蛛侠", def.Name)
	}
	if def.Text != "<b>响应</b>：测试。" {
		t.Errorf("Text = %q", def.Text)
	}
	if len(def.Traits) != 1 || def.Traits[0] != "复仇者" {
		t.Errorf("Traits = %v, want [复仇者]", def.Traits)
	}
	// Untranslated cards keep English values.
	other := db.MustLookup("01003")
	if hasHan(other.Name) {
		t.Errorf("untranslated card got Chinese name: %q", other.Name)
	}
}

func TestSplitZhTraits(t *testing.T) {
	got := splitZhTraits("神盾局。间谍。")
	if len(got) != 2 || got[0] != "神盾局" || got[1] != "间谍" {
		t.Errorf("split = %v", got)
	}
}

func TestHasHan(t *testing.T) {
	if !hasHan("蜘蛛侠") || hasHan("Spider-Man") {
		t.Error("hasHan wrong")
	}
}
