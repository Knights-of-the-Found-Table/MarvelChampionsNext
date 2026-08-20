package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// zhCard is one entry of a Simplified Chinese translation pack file
// (tools/zh/out/<pack>.json): card code -> translated fields.
type zhCard struct {
	Name    string `json:"name"`
	Subname string `json:"subname"`
	Text    string `json:"text"`
	Traits  string `json:"traits"`
}

// ApplyChinese overlays Simplified Chinese translations onto db. dir holds
// one JSON file per pack (any *.json directly inside dir; subdirectories are
// skipped). Only non-empty translated fields replace the English values, so
// untranslated cards keep working as before. Returns the number of cards
// overlaid.
func ApplyChinese(db *data.Database, dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("read zh dir: %w", err)
	}
	overlaid := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return overlaid, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		raw = trimBOM(raw)
		var pack map[string]zhCard
		if err := json.Unmarshal(raw, &pack); err != nil {
			return overlaid, fmt.Errorf("decode %s: %w", e.Name(), err)
		}
		for code, tr := range pack {
			def, ok := db.Cards[code]
			if !ok {
				continue
			}
			if tr.Name != "" {
				def.Name = tr.Name
			}
			if tr.Subname != "" {
				def.Subname = tr.Subname
			}
			if tr.Text != "" {
				def.Text = tr.Text
			}
			if tr.Traits != "" {
				def.Traits = splitZhTraits(tr.Traits)
			}
			overlaid++
		}
	}
	return overlaid, nil
}

// RelabelScenarios renames registered scenarios from the overlaid card data:
// "VillainName——SchemeName" when the scenario has villains, else the scheme
// card's name. Scheme cards are looked up by the scenario id with an
// optional "a" side suffix (scheme cards carry one in the card database).
// Scenarios whose cards were not translated keep their English names.
func RelabelScenarios(db *data.Database) {
	for _, sc := range scenarioRegistry {
		scheme, ok := schemeDef(db, sc.ID)
		if !ok || !hasHan(scheme.Name) {
			continue
		}
		if len(sc.VillainBases) > 0 {
			if v, vok := schemeDef(db, sc.VillainBases[0]); vok && hasHan(v.Name) {
				sc.Name = v.Name + "——" + scheme.Name
				continue
			}
		}
		sc.Name = scheme.Name
	}
}

// schemeDef looks up a scenario/villain card by base code, tolerating the
// "a" side suffix some scheme cards carry.
func schemeDef(db *data.Database, code string) (*data.CardDef, bool) {
	if def, ok := db.Cards[code]; ok {
		return def, true
	}
	def, ok := db.Cards[code+"a"]
	return def, ok
}

// splitZhTraits splits a translated traits line ("神盾局。间谍。") into the
// slice form used by CardDef.Traits.
func splitZhTraits(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '。' || r == '.' || r == '；'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// hasHan reports whether s contains at least one Han character, i.e. whether
// a value was actually translated.
func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// trimBOM strips a UTF-8 byte order mark, which some editors and Windows
// tooling prepend and encoding/json refuses to parse.
func trimBOM(b []byte) []byte {
	return bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))
}
