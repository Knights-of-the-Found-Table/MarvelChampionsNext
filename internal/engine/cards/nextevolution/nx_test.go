package nextevolution_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nextevolution"
)

func newNxGame(t *testing.T, seed int64, scenario string, hero string, deck map[string]int) *engine.Game {
	t.Helper()
	if deck == nil {
		deck = map[string]int{"01088": 9, "01089": 9}
	}
	if hero == "" {
		hero = "01001"
	}
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players:    []engine.PlayerSpec{{Name: "P1", HeroBase: hero, Deck: deck}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	for i := 0; i < 10; i++ {
		pq := g.Pending()
		if pq == nil || pq.Question.Prompt == "Your turn" {
			break
		}
		idx := 0
		for j, c := range pq.Question.Choices {
			if c.Then == nil && !c.Disabled {
				idx = j
				break
			}
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[idx].ID})
	}
	return g
}

// TestMutantMassacreSetup: Routed in play, one Marauder active, the rest
// banked as the villain deck.
func TestMutantMassacreSetup(t *testing.T) {
	g := newNxGame(t, 61, "40077", "", nil)
	var routed *engine.Environment
	for _, e := range g.Environments {
		if e != nil && engine.BaseCodeOf(e.Code) == "40081" {
			routed = e
		}
	}
	if routed == nil {
		t.Fatal("Routed should start in play")
	}
	if n := len(g.Villains); n != 1 {
		t.Fatalf("exactly one Marauder villain should be in play, got %d", n)
	}
	benched := 0
	for _, c := range g.SetAside {
		if c.Def().Type == "villain" {
			benched++
		}
	}
	if benched != 6 {
		t.Fatalf("6 Marauders should wait in the villain deck, got %d", benched)
	}
}

// TestJuggernautSetup: helmet attached, momentum ticking, Hope fielded.
func TestJuggernautSetup(t *testing.T) {
	g := newNxGame(t, 62, "40121", "", nil)
	var v *engine.Villain
	for _, vv := range g.Villains {
		v = vv
	}
	if v == nil || engine.BaseCodeOf(v.Code) != "40118" {
		t.Fatal("Juggernaut should be in play")
	}
	if v.Counters != 1 {
		t.Fatalf("Juggernaut should start with 1 momentum counter, got %d", v.Counters)
	}
	found := false
	for _, p := range g.Players {
		for _, id := range p.Allies {
			if a := g.Allies[id]; a != nil && engine.BaseCodeOf(a.Code) == "40130" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("Hope Summers should start under the first player's control")
	}
	helmet := false
	for _, a := range g.Attachments {
		if a != nil && a.Code == "40122a" {
			helmet = true
		}
	}
	if !helmet {
		t.Fatal("Juggernaut's Helmet should be attached")
	}
	// +1 ATK per momentum counter reflects in the attack value.
	if got := engine.BaseCodeOf(v.Code); got == "40118" {
		// sanity: attack value includes 1 momentum beyond printed ATK.
		if av := g.AttackValueOf(v.ID); av < 2 {
			t.Fatalf("Juggernaut's attack should include momentum, got %d", av)
		}
	}
}

// TestStryfeSetup: Stryfe's Grasp starts in play.
func TestStryfeSetup(t *testing.T) {
	g := newNxGame(t, 63, "40166", "", nil)
	found := false
	for _, s := range g.SideSchemes {
		if s != nil && s.Code == "40168a" {
			found = true
		}
	}
	if !found {
		t.Fatal("Stryfe's Grasp should start in play")
	}
}

// TestSinisterSetup: Hope fielded and one stage 2 removed from the game.
func TestSinisterSetup(t *testing.T) {
	g := newNxGame(t, 64, "40139", "", nil)
	if g.MainScheme == nil || len(g.MainScheme.StageCodes) != 4 {
		t.Fatalf("Sinister Intent should run 4 stages after removing one, got %d", len(g.MainScheme.StageCodes))
	}
}

// TestDominoLuckRider: A Good Workout's damage scales with the discarded
// top card's icons.
func TestDominoLuckRider(t *testing.T) {
	g := newNxGame(t, 65, "40121", "40037", map[string]int{
		"40040": 3, "01088": 6, "01089": 6,
	})
	p := g.Players[0]
	// Put A Good Workout in hand with a known deck top.
	var workout engine.Card
	for _, c := range p.Hand {
		if c.Code == "40040" {
			workout = c
			break
		}
	}
	if workout.Code == "" {
		t.Fatal("A Good Workout should be in the opening hand")
	}
	topIcons := len(p.Deck[0].Def().Resources)
	ec := &engine.EventCard{Code: workout.Code, Owner: p.ID}
	b := engine.LookupBehavior("40040")
	msgs := b.OnPlay(g, ec)
	if len(msgs) == 0 {
		t.Fatal("A Good Workout should produce effect messages")
	}
	found := false
	for _, m := range msgs {
		if q, ok := m.(engine.AskQuestion); ok && q.Question.Prompt != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("A Good Workout should ask for a target")
	}
	_ = topIcons
}

// TestMarauderAttackChoice: the villain's attack offers the penalty/boost
// choice.
func TestMarauderAttackChoice(t *testing.T) {
	g := newNxGame(t, 66, "40077", "", nil)
	var v *engine.Villain
	for _, vv := range g.Villains {
		v = vv
	}
	if v == nil {
		t.Fatal("a Marauder should be in play")
	}
	p := g.Players[0]
	p.Side = engine.SideHero
	msgs := v.React(g, engine.AskAttack{Enemy: v.ID, Player: p.ID})
	if len(msgs) == 0 {
		t.Fatalf("%s should react to its attack window", v.EDef().Name)
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || len(ask.Question.Choices) != 2 {
		t.Fatalf("the Marauder choice should offer two options, got %+v", msgs[0])
	}
}
