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
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/mojo"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/mutantgenesis"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nextevolution"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/trickster"
)

// Scheme stage codes follow the repo-wide face convention: the b side is the
// gameplay face (carries the threat stats) and the one shown in play. Two
// registrations deliberately deviate: 32088a is the Mutants at the Mall flip
// card (a side scheme whose b face is an ally) reused as a scheme stage, and
// the whole 40139 family is statless in the pack data (a known next_evol
// snapshot gap).
var (
	nonBSchemeStages = map[string]bool{"32088a": true,
		// the sm snapshot codes its gameplay faces without the b suffix
		"27064": true, "27087": true, "27088": true, "27100": true,
		// the mts snapshot follows the same unsuffixed convention
		"21074": true, "21075": true, "21098": true, "21114": true,
		"21115": true, "21138": true, "21165": true,
		// the trors snapshot follows the same convention
		"04061": true, "04062": true, "04063": true, "04079": true,
		"04096": true, "04112": true, "04113": true, "04128": true,
		"04129":  true,
		"27076a": true, "27116a": true, "27101a": true,
		// the next_evol snapshot codes its gameplay faces without the b
		// suffix (the a side is the setup face)
		"40077": true, "40078": true, "40103": true, "40104": true,
		"40121": true, "40139": true, "40140": true, "40141": true,
		"40142": true, "40143": true, "40166": true, "40167": true,
		// the aoa snapshot follows the same convention
		"45062": true, "45085": true, "45103": true, "45121": true,
		"45147": true, "45148": true,
	}
	statlessSchemeStages = map[string]bool{
		"40139":  true, // transition stage, no threat target
		"40139b": true, // transition stage, no threat target
		"45103":  true, // The Age of Apocalypse: X is villain HP based
		"11008b": true, // Master of Time: variants-arrival transition
		"50168b": true, // The Accusation: guess-the-traitor transition
		// sm snapshot gaps: single-face schemes with no threat stats
		"27076a": true, "27116a": true, "27101a": true,
	}
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

	// A scheme caught mid-reveal (its flip still queued) keeps the a face
	// and only has its stage list refreshed.
	blob := `{"scenarioId":"01097","mainScheme":{"id":"m1","code":"01097a","stageCodes":["01097b"],"stage":1},` +
		`"queue":[{"t":"engine.FlipMainScheme","m":{"Scheme":"m1"}}]}`
	g := &engine.Game{}
	if err := g.UnmarshalJSON([]byte(blob)); err != nil {
		t.Fatalf("mid-reveal: unmarshal: %v", err)
	}
	if g.MainScheme.Code != "01097a" || !slices.Equal(g.MainScheme.StageCodes, []string{"01097b"}) {
		t.Errorf("mid-reveal: code=%q stageCodes=%v, want 01097a / [01097b]",
			g.MainScheme.Code, g.MainScheme.StageCodes)
	}
}

// TestSchemeAFaceLifecycle covers the a→b reveal flow: a stage enters on
// its a face, the a face's reveal effects settle, then the scheme flips to
// the b face (the code the view exposes in play).
func TestSchemeAFaceLifecycle(t *testing.T) {
	// Rhino 1A has no reveal effect; the scheme simply ends on 1B.
	g := newScenarioGame(t, "01097")
	if g.MainScheme.Code != "01097b" {
		t.Errorf("rhino: scheme code = %q, want 01097b", g.MainScheme.Code)
	}

	// Klaw 1A setup searches the encounter deck for the Defense Network
	// side scheme and reveals it.
	g = newScenarioGame(t, "01116")
	if g.MainScheme.Code != "01116b" {
		t.Errorf("klaw: scheme code = %q, want 01116b", g.MainScheme.Code)
	}
	found := false
	for _, s := range g.SideSchemes {
		if s.Code == "01125" {
			found = true
		}
	}
	if !found {
		t.Error("klaw: Defense Network side scheme not revealed at setup")
	}

	// Ultron 1A setup puts the Ultron Drones environment into play.
	g = newScenarioGame(t, "01137")
	if g.MainScheme.Code != "01137b" {
		t.Errorf("ultron: scheme code = %q, want 01137b", g.MainScheme.Code)
	}
	if len(g.Environments) == 0 {
		t.Error("ultron: Ultron Drones environment not in play after setup")
	}

	// Klaw stage 2A: advancing discards encounter cards until a minion
	// shows up; it enters play engaged with the first player. The advance
	// is front-inserted at the current pause point; answers resume their
	// interrupted flow ahead of it, so pick a seed whose first villain
	// activation doesn't complete the stage-1 scheme first.
	var g2 *engine.Game
	for seed := int64(1); seed <= 50 && g2 == nil; seed++ {
		cand := newScenarioGameSeed(t, "01116", seed)
		cand.PushFront(engine.ReplaceMainScheme{Scheme: cand.MainScheme.ID})
		pq := cand.Pending()
		if pq == nil {
			continue
		}
		paths := pickDefault(pq.Question)
		if paths == nil {
			continue
		}
		if err := cand.Answer(pq.Player, paths); err != nil {
			t.Fatalf("seed %d answer: %v", seed, err)
		}
		if !cand.Over {
			g2 = cand
		}
	}
	if g2 == nil {
		t.Fatal("no seed survived the first villain activation")
	}
	g = g2
	for i := 0; i < 30 && !g.Over && g.MainScheme.Code != "01117b"; i++ {
		pq := g.Pending()
		if pq == nil {
			break
		}
		paths := pickDefault(pq.Question)
		if paths == nil {
			t.Fatalf("no answerable choice in question %q", pq.Question.Prompt)
		}
		if err := g.Answer(pq.Player, paths); err != nil {
			t.Fatalf("answer: %v", err)
		}
	}
	if g.Over {
		start := max(0, len(g.Log)-20)
		for _, e := range g.Log[start:] {
			t.Log(e.Text)
		}
		t.Fatalf("klaw: game over during advance: %s", g.Reason.Text)
	}
	if g.MainScheme.Code != "01117b" {
		t.Errorf("klaw: scheme code after advance = %q, want 01117b", g.MainScheme.Code)
	}
	engaged := false
	for _, mn := range g.Minions {
		if mn.EngagedWith != "" {
			engaged = true
		}
	}
	if !engaged {
		t.Error("klaw: no engaged minion after stage 2A reveal")
	}
}

func newScenarioGame(t *testing.T, scenarioID string) *engine.Game {
	return newScenarioGameSeed(t, scenarioID, 1)
}

func newScenarioGameSeed(t *testing.T, scenarioID string, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenarioID,
		Players: []engine.PlayerSpec{
			{Name: "Tester", HeroBase: "01001", Deck: spiderDeck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
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
