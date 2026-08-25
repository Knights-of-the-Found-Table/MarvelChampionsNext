package engine_test

import (
	"encoding/json"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register core set content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

func spiderDeck() map[string]int {
	return map[string]int{
		// Spider-Man signature cards
		"01002": 1, // Black Cat
		"01003": 2, // Backflip
		"01004": 2, // Enhanced Spider-Sense
		"01005": 2, // Swinging Web Kick
		"01006": 1, // Aunt May
		"01007": 2, // Spider-Tracer
		"01008": 2, // Web-Shooter
		// filler player cards
		"01088": 3, // Energy
		"01089": 3, // Genius
		"01090": 3, // Strength
		"01055": 1, // The Power of Aggression
		"01054": 2, // Uppercut
	}
}

func newTestGame(t *testing.T, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Tester", HeroBase: "01001", Deck: spiderDeck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// pickDefault answers questions with a simple policy: prefer the scripted
// choice kinds over card plays.
func pickDefault(q *engine.Question) []string {
	// deterministic: pick by preference
	prefer := []string{"pass-interrupt", "keep", "take", "end-turn"}
	for _, id := range prefer {
		for _, c := range q.Choices {
			if c.ID == id && !c.Disabled {
				return []string{id}
			}
		}
	}
	// resource payment questions: select the first N choices
	if q.Type == "choose_n" {
		var out []string
		for _, c := range q.Choices {
			if len(out) >= q.N {
				break
			}
			out = append(out, c.ID)
		}
		return out
	}
	if len(q.Choices) > 0 {
		// choose_one: skip branches chaining into a choose_n payment
		// subtree — answering those needs the full subtree selection set
		// ({"0.0","0.1",…}), which this helper cannot compute; take the
		// first plain branch instead (e.g. Sonic Boom's "exhaust").
		for _, c := range q.Choices {
			if c.Then != nil && c.Then.Type == "choose_n" {
				continue
			}
			return []string{c.ID}
		}
		return []string{q.Choices[0].ID}
	}
	return nil
}

func driveGame(t *testing.T, g *engine.Game, maxAnswers int) {
	t.Helper()
	for i := 0; i < maxAnswers; i++ {
		pq := g.Pending()
		if pq == nil {
			return
		}
		paths := pickDefault(pq.Question)
		if paths == nil {
			t.Fatalf("no answerable choice in question %q", pq.Question.Prompt)
		}
		if err := g.Answer(pq.Player, paths); err != nil {
			t.Fatalf("answer %v: %v", paths, err)
		}
		if g.Over {
			return
		}
	}
}

func TestRhinoGameRunsToLossByScheme(t *testing.T) {
	g := newTestGame(t, 42)
	driveGame(t, g, 500)
	if !g.Over {
		t.Fatalf("expected game to end, round=%d", g.Round)
	}
	if g.Won {
		t.Fatalf("expected loss with passive script, won instead: %s", g.Reason.Text)
	}
	t.Logf("game over: %s after %d rounds", g.Reason.Text, g.Round)
}

func TestDeterminism(t *testing.T) {
	g1 := newTestGame(t, 7)
	g2 := newTestGame(t, 7)
	for i := 0; i < 50; i++ {
		if g1.Pending() == nil || g1.Over {
			break
		}
		paths := pickDefault(g1.Pending().Question)
		if err := g1.Answer(g1.Pending().Player, paths); err != nil {
			t.Fatalf("g1 answer: %v", err)
		}
		if err := g2.Answer(g2.Pending().Player, paths); err != nil {
			t.Fatalf("g2 answer: %v", err)
		}
	}
	b1, _ := json.Marshal(g1)
	b2, _ := json.Marshal(g2)
	if string(b1) != string(b2) {
		t.Fatalf("same seed + answers diverged:\n%s\n%s", b1[:min(400, len(b1))], b2[:min(400, len(b2))])
	}
}

func TestSerializationRoundTrip(t *testing.T) {
	g := newTestGame(t, 99)
	// answer a few questions
	for i := 0; i < 10 && g.Pending() != nil && !g.Over; i++ {
		paths := pickDefault(g.Pending().Question)
		if err := g.Answer(g.Pending().Player, paths); err != nil {
			t.Fatalf("answer: %v", err)
		}
	}
	raw, err := g.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	restored := &engine.Game{}
	if err := restored.UnmarshalJSON(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if restored.Round != g.Round {
		t.Fatalf("round mismatch %d != %d", restored.Round, g.Round)
	}
	// continue both games with identical answers and compare
	for i := 0; i < 20; i++ {
		if g.Pending() == nil || g.Over {
			break
		}
		paths := pickDefault(g.Pending().Question)
		if err := g.Answer(g.Pending().Player, paths); err != nil {
			t.Fatalf("orig answer: %v", err)
		}
		if restored.Pending() == nil || restored.Over {
			t.Fatalf("restored game stopped early at i=%d", i)
		}
		if err := restored.Answer(restored.Pending().Player, paths); err != nil {
			t.Fatalf("restored answer: %v", err)
		}
	}
	b1, _ := json.Marshal(g)
	b2, _ := json.Marshal(restored)
	if string(b1) != string(b2) {
		t.Fatal("restored game diverged from original")
	}
}

func TestVillainStageProgressionAndWin(t *testing.T) {
	g := newTestGame(t, 1)
	// Keep the opening hand (setup mulligan) so the first turn starts.
	if pq := g.Pending(); pq != nil {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("answer mulligan: %v", err)
		}
	}
	// Force Rhino through all three stages via repeated lethal damage.
	var villainID engine.EntityID
	for id := range g.Villains {
		villainID = id
	}
	// Rhino III has Toughness, so a 4th hit clears the tough status first.
	for i := 0; i < 4; i++ {
		g.Push(engine.DamageEntity{Target: villainID, Damage: 99, Source: engine.EntityID("player-1")})
	}
	// The turn menu question is pending; answering it lets the queue drain
	// (our damage messages resolve after the end-of-player-phase prompts).
	driveGame(t, g, 50)
	g.Run()
	if !g.Over || !g.Won {
		t.Fatalf("expected win after defeating all stages, over=%v won=%v round=%d", g.Over, g.Won, g.Round)
	}
}
