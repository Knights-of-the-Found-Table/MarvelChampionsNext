package engine_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register every scenario-defining card package
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/civilwar"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/galaxysmostwanted"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/goblinfooblin"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/mutantgenesis"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nextevolution"
)

// Scheme stage codes follow the repo-wide face convention: the b side is the
// gameplay face (carries the threat stats) and the one shown in play. Two
// registrations deliberately deviate: 32088a is the Mutants at the Mall flip
// card (a side scheme whose b face is an ally) reused as a scheme stage, and
// the whole 40139 family is statless in the pack data (a known next_evol
// snapshot gap).
var (
	nonBSchemeStages     = map[string]bool{"32088a": true}
	statlessSchemeStages = map[string]bool{"40139b": true}
)

// TestScenarioSchemeStages verifies every registered scenario's main scheme
// stage codes resolve to b-face main scheme definitions with threat stats.
func TestScenarioSchemeStages(t *testing.T) {
	for _, scen := range engine.Scenarios() {
		for _, code := range scen.MainSchemeStages {
			def, ok := engine.DB.Lookup(code)
			if !ok {
				t.Errorf("%s: scheme stage %s not in card db", scen.ID, code)
				continue
			}
			if nonBSchemeStages[code] {
				continue
			}
			if def.Type != "main_scheme" {
				t.Errorf("%s: scheme stage %s is a %s, want main_scheme", scen.ID, code, def.Type)
			}
			if def.Side != "b" {
				t.Errorf("%s: scheme stage %s is side %q, want the b face", scen.ID, code, def.Side)
			}
			if def.Threat == nil && def.BaseThreat == nil && def.EscalationThreat == nil && !statlessSchemeStages[code] {
				t.Errorf("%s: scheme stage %s carries no threat stats", scen.ID, code)
			}
		}
	}
}

// TestUnmarshalMigratesSchemeCodes restores pre-convention games: base
// codes ("01097"), a-face registrations ("56063a") and the old Drang
// mis-registration (the "16057" treachery) must come back keyed by the
// current b-face stage codes from the scenario registry.
func TestUnmarshalMigratesSchemeCodes(t *testing.T) {
	cases := []struct {
		scenarioID string
		oldCode    string
		oldStages  []string
		stage      int
		wantCode   string
		wantStages []string
	}{
		{"01097", "01097", []string{"01097"}, 1, "01097b", []string{"01097b"}},
		{"56063", "56063a", []string{"56063a"}, 1, "56063b", []string{"56063b"}},
		{"01116", "01117", []string{"01116", "01117"}, 2, "01117b", []string{"01116b", "01117b"}},
		{"16057", "16057", []string{"16057"}, 1, "16061b", []string{"16061b", "16062b"}},
	}
	for _, tc := range cases {
		stages, _ := json.Marshal(tc.oldStages)
		blob := fmt.Sprintf(`{"scenarioId":%q,"mainScheme":{"code":%q,"stageCodes":%s,"stage":%d}}`,
			tc.scenarioID, tc.oldCode, stages, tc.stage)
		g := &engine.Game{}
		if err := g.UnmarshalJSON([]byte(blob)); err != nil {
			t.Fatalf("%s: unmarshal: %v", tc.scenarioID, err)
		}
		if g.MainScheme.Code != tc.wantCode {
			t.Errorf("%s: code = %q, want %q", tc.scenarioID, g.MainScheme.Code, tc.wantCode)
		}
		if !slices.Equal(g.MainScheme.StageCodes, tc.wantStages) {
			t.Errorf("%s: stageCodes = %v, want %v", tc.scenarioID, g.MainScheme.StageCodes, tc.wantStages)
		}
	}
}

// TestSchemeStageStats spot-checks that switching stage codes to the b face
// preserved gameplay stats (they live on the b records).
func TestSchemeStageStats(t *testing.T) {
	cases := []struct {
		code                 string
		maxThreat, base, esc int
	}{
		{"01097b", 7, 0, 1},   // Rhino: The Break-In
		{"02018b", 11, 4, -1}, // Mutagen Cloud
		{"16061b", 8, 2, 2},   // Drang: Terrestrial Invasion
		{"56063b", 7, 0, 1},   // Civil War: Cut Off Support
	}
	for _, tc := range cases {
		def, ok := engine.DB.Lookup(tc.code)
		if !ok {
			t.Fatalf("%s not in card db", tc.code)
		}
		if def.Threat == nil || *def.Threat != tc.maxThreat {
			t.Errorf("%s: threat = %v, want %d", tc.code, def.Threat, tc.maxThreat)
		}
		if def.BaseThreat == nil || *def.BaseThreat != tc.base {
			t.Errorf("%s: base threat = %v, want %d", tc.code, def.BaseThreat, tc.base)
		}
		if def.EscalationThreat == nil || *def.EscalationThreat != tc.esc {
			t.Errorf("%s: escalation threat = %v, want %d", tc.code, def.EscalationThreat, tc.esc)
		}
	}
}
