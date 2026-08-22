package engine_test

import (
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
