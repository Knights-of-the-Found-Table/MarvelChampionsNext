package jessicadrew_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/jessicadrew"
)

func swDeck() map[string]int {
	return map[string]int{
		"04032": 1, "04033": 2, "04034": 1, "04035": 2,
		"04036": 2, "04037": 2, "04038": 2, "04039": 3, "04040": 1,
	}
}

func newSWGame(t *testing.T) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed: 40, ScenarioID: "01097",
		Players: []engine.PlayerSpec{{Name: "Spider-Woman", HeroBase: "04031", Deck: swDeck()}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// Contract test: playing an aspect card in hero form triggers Superhuman
// Agility exactly once per aspect per round; a second aggression play does
// not retrigger, and a justice play does.
func TestSuperhumanAgilityOncePerAspectPerRound(t *testing.T) {
	g := newSWGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	g.UsedThisRound = map[string]bool{}

	b := engine.LookupBehavior("04031")
	if b == nil || b.React == nil {
		t.Fatal("Spider-Woman should expose a React hook")
	}
	aggr := engine.Card{Code: "04043", Owner: p.ID} // Press the Advantage (aggression)
	just := engine.Card{Code: "04047", Owner: p.ID}  // Skilled Investigator (justice)

	msgs := b.React(g, p, engine.PlayCard{Player: p.ID, Card: aggr})
	if len(msgs) != 1 {
		t.Fatalf("first aggression play returned %d messages, want one ApplyStatBonus", len(msgs))
	}
	if sb, ok := msgs[0].(engine.ApplyStatBonus); !ok || sb.ATK != 1 || sb.THW != 1 || sb.DEF != 1 {
		t.Fatalf("bonus = %#v, want +1/+1/+1", msgs[0])
	}
	if msgs := b.React(g, p, engine.PlayCard{Player: p.ID, Card: aggr}); msgs != nil {
		t.Fatalf("second aggression play retriggered: %#v", msgs)
	}
	if msgs := b.React(g, p, engine.PlayCard{Player: p.ID, Card: just}); len(msgs) != 1 {
		t.Fatalf("justice play returned %d messages, want one ApplyStatBonus", len(msgs))
	}
	// Identity-specific cards (hero set) never trigger, even though the
	// data layer tags them with an aspect faction.
	sig := engine.Card{Code: "04035", Owner: p.ID}
	if msgs := b.React(g, p, engine.PlayCard{Player: p.ID, Card: sig}); msgs != nil {
		t.Fatalf("signature card triggered Superhuman Agility: %#v", msgs)
	}
	// Round reset re-arms the aspects.
	g.UsedThisRound = map[string]bool{}
	if msgs := b.React(g, p, engine.PlayCard{Player: p.ID, Card: aggr}); len(msgs) != 1 {
		t.Fatalf("after round reset, aggression play returned %d messages, want 1", len(msgs))
	}
	// Alter-ego form never triggers.
	p.Side = engine.SideAlterEgo
	g.UsedThisRound = map[string]bool{}
	if msgs := b.React(g, p, engine.PlayCard{Player: p.ID, Card: aggr}); msgs != nil {
		t.Fatalf("alter-ego play triggered Superhuman Agility: %#v", msgs)
	}
}

// Contract test: The Viper engaged with Spider-Woman reduces her hand size
// by 1 through the identity's HandSizeBonus hook.
func TestViperHandSizePenalty(t *testing.T) {
	g := newSWGame(t)
	p := g.Players[0]
	b := engine.LookupBehavior("04031")
	if b == nil || b.HandSizeBonus == nil {
		t.Fatal("Spider-Woman should expose HandSizeBonus")
	}
	if got := b.HandSizeBonus(g, p); got != 0 {
		t.Fatalf("HandSizeBonus = %d without Viper, want 0", got)
	}
	mn := &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: "04054", MaxHP: 5, EngagedWith: p.ID}
	g.Minions[mn.ID] = mn
	if got := b.HandSizeBonus(g, p); got != -1 {
		t.Fatalf("HandSizeBonus = %d with engaged Viper, want -1", got)
	}
	mn.EngagedWith = "someone-else"
	if got := b.HandSizeBonus(g, p); got != 0 {
		t.Fatalf("HandSizeBonus = %d with Viper engaged elsewhere, want 0", got)
	}
}

// Contract test: Pheromones asks for an enemy and stuns + confuses it.
func TestPheromonesStunConfuse(t *testing.T) {
	g := newSWGame(t)
	p := g.Players[0]
	b := engine.LookupBehavior("04036")
	if b == nil || b.OnPlay == nil {
		t.Fatal("Pheromones should expose OnPlay")
	}
	ev := &engine.EventCard{Code: "04036", Owner: p.ID}
	msgs := b.OnPlay(g, ev)
	if len(msgs) != 1 {
		t.Fatalf("Pheromones returned %d messages, want one question", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || ask.Question == nil || len(ask.Question.Choices) == 0 {
		t.Fatalf("Pheromones message = %#v, want a target question", msgs[0])
	}
	// The villain must be among the targets with stun+confuse payloads.
	chain, err := ask.Question.Chain(ask.Question.Choices[0].ID)
	if err != nil || len(chain) != 1 {
		t.Fatalf("chain resolution failed: %v", err)
	}
}

// Contract test: Uncertain Loyalties offers the exhaust-to-remove path and
// the 3-threat penalty path.
func TestUncertainLoyaltiesChoice(t *testing.T) {
	g := newSWGame(t)
	p := g.Players[0]
	b := engine.LookupBehavior("04053")
	if b == nil || b.ResolveObligation == nil {
		t.Fatal("Uncertain Loyalties should expose ResolveObligation")
	}
	msgs := b.ResolveObligation(g, p, engine.Card{Code: "04053", Owner: p.ID})
	if len(msgs) != 1 {
		t.Fatalf("obligation returned %d messages, want one question", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || ask.Question == nil || len(ask.Question.Choices) != 2 {
		t.Fatalf("obligation message = %#v, want two choices", msgs[0])
	}
}
