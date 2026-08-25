package deadpool_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/deadpool"
)

func deadpoolDeck() map[string]int {
	return map[string]int{
		"44002": 1, "44003": 1, "44004": 1, "44005": 1, "44006": 1,
		"44007": 1, "44008": 1, "44009": 1, "44010": 1, "44011": 1,
		"44012": 1,
	}
}

func newDeadpoolGame(t *testing.T, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Deadpool", HeroBase: "44001", Deck: deadpoolDeck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// TestDeadpoolImplemented: the identity is registered.
func TestDeadpoolImplemented(t *testing.T) {
	if !engine.Implemented("44001a") {
		t.Fatal("Deadpool identity should be implemented")
	}
}

// TestRegeneratinDegenerate: when Deadpool would be defeated, the
// identity-level DefeatSave sets the dial to 1, flips him to alter-ego
// and feeds the main scheme an acceleration token. Contract test via
// the DefeatSave hook (nil upgrade = identity-level).
func TestRegeneratinDegenerate(t *testing.T) {
	g := newDeadpoolGame(t, 44)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Damage = p.MaxHP // lethal
	if g.MainScheme == nil {
		t.Fatal("game should have a main scheme")
	}
	tokens := g.MainScheme.AccelerationTokens

	b := engine.LookupBehavior("44001")
	if b == nil || b.DefeatSave == nil {
		t.Fatal("Deadpool identity should expose DefeatSave")
	}
	if !b.DefeatSave(g, p, nil) {
		t.Fatal("The Regeneratin' Degenerate should prevent the defeat")
	}
	if p.HP() != 1 {
		t.Fatalf("HP after the save = %d, want 1", p.HP())
	}
	if p.Side != engine.SideAlterEgo {
		t.Fatalf("side after the save = %q, want alter-ego", p.Side)
	}
	if g.MainScheme.AccelerationTokens != tokens+1 {
		t.Fatalf("acceleration tokens = %d, want %d",
			g.MainScheme.AccelerationTokens, tokens+1)
	}

	// Already in alter-ego: the save still applies (and still feeds the
	// scheme).
	p.Damage = p.MaxHP
	if !b.DefeatSave(g, p, nil) {
		t.Fatal("the save should also apply from alter-ego form")
	}
	if p.HP() != 1 || p.Side != engine.SideAlterEgo {
		t.Fatalf("alter-ego save: HP %d, side %q; want 1 HP alter-ego", p.HP(), p.Side)
	}
	if g.MainScheme.AccelerationTokens != tokens+2 {
		t.Fatalf("acceleration tokens = %d, want %d",
			g.MainScheme.AccelerationTokens, tokens+2)
	}
}

// TestBreakTheFourthWall: the alter-ego action discards a hand card,
// then searches the deck for a Deadpool event only (allies, resources
// and non-Deadpool cards are excluded). Contract test via HeroAbilities.
func TestBreakTheFourthWall(t *testing.T) {
	g := newDeadpoolGame(t, 44)
	p := g.Players[0]
	p.Side = engine.SideAlterEgo
	p.Hand = engine.CardList{{ID: g.NextCardID(), Code: "44009", Owner: p.ID}}
	p.Deck = engine.CardList{
		{ID: g.NextCardID(), Code: "44004", Owner: p.ID}, // Deadpool event
		{ID: g.NextCardID(), Code: "44002", Owner: p.ID}, // Deadpool ally
		{ID: g.NextCardID(), Code: "44007", Owner: p.ID}, // Deadpool resource
	}

	b := engine.LookupBehavior("44001")
	if b == nil || b.HeroAbilities == nil {
		t.Fatal("Deadpool identity should expose HeroAbilities")
	}
	abs := b.HeroAbilities(g, p)
	if len(abs) != 1 {
		t.Fatalf("HeroAbilities = %d, want 1", len(abs))
	}
	ab := abs[0]
	if !ab.AlterEgoOnly || !ab.OncePerRound {
		t.Fatal("Break the Fourth Wall should be AlterEgoOnly and OncePerRound")
	}
	msgs := ab.Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Break the Fourth Wall returned %d messages, want 1", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || ask.Question == nil || len(ask.Question.Choices) != 1 {
		t.Fatalf("Break the Fourth Wall message = %#v, want a 1-choice discard question", msgs[0])
	}
	search := ask.Question.Choices[0].Then
	if search == nil {
		t.Fatal("the discard choice should chain into the search question")
	}
	var foundEvent, foundDecline bool
	for _, c := range search.Choices {
		if strings.Contains(c.Label.Text, "Maximum Effort") {
			foundEvent = true
		}
		if strings.Contains(c.Label.Text, "Cable") || strings.Contains(c.Label.Text, "Montage") {
			t.Fatalf("search should not offer %q (not a Deadpool event)", c.Label.Text)
		}
		if c.Kind == engine.ChoicePass {
			foundDecline = true
		}
	}
	if !foundEvent {
		t.Fatal("search should offer Maximum Effort (a Deadpool event)")
	}
	if !foundDecline {
		t.Fatal("search should offer a decline (failed search) option")
	}
}

// TestMaximumEffortChoices: the attack offers one choice per point of
// remaining hit points, each chaining into an enemy target question.
// Contract test via OnPlay.
func TestMaximumEffortChoices(t *testing.T) {
	g := newDeadpoolGame(t, 44)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Damage = 3 // 6 HP remaining

	b := engine.LookupBehavior("44004")
	if b == nil || b.OnPlay == nil {
		t.Fatal("Maximum Effort should expose OnPlay")
	}
	ev := &engine.EventCard{Code: "44004", Owner: p.ID}
	msgs := b.OnPlay(g, ev)
	if len(msgs) != 1 {
		t.Fatalf("Maximum Effort returned %d messages, want 1", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || ask.Question == nil {
		t.Fatalf("Maximum Effort message = %#v, want a damage question", msgs[0])
	}
	if len(ask.Question.Choices) != 6 {
		t.Fatalf("Maximum Effort offers %d amounts, want 6 (one per remaining HP)",
			len(ask.Question.Choices))
	}
	for _, c := range ask.Question.Choices {
		if c.Then == nil || len(c.Then.Choices) == 0 {
			t.Fatalf("choice %q should chain into an enemy target question", c.Label.Text)
		}
	}

	// No remaining hit points: the event fizzles.
	p.Damage = p.MaxHP
	if msgs := b.OnPlay(g, ev); len(msgs) != 0 {
		t.Fatalf("Maximum Effort at 0 HP returned %d messages, want 0", len(msgs))
	}
}

// TestMetaknowledgeCancels: the interrupt cancels a revealed treachery
// in hero form and is unplayable in alter-ego form. Contract test via
// TreacheryInterrupt.
func TestMetaknowledgeCancels(t *testing.T) {
	g := newDeadpoolGame(t, 44)
	p := g.Players[0]

	b := engine.LookupBehavior("44005")
	if b == nil || b.TreacheryInterrupt == nil {
		t.Fatal("Metaknowledge should expose TreacheryInterrupt")
	}
	p.Side = engine.SideHero
	if repl := b.TreacheryInterrupt(g, p, engine.Card{Code: "44005", Owner: p.ID}); repl == nil {
		t.Fatal("Metaknowledge should cancel a revealed treachery in hero form")
	}
	p.Side = engine.SideAlterEgo
	if repl := b.TreacheryInterrupt(g, p, engine.Card{Code: "44005", Owner: p.ID}); repl != nil {
		t.Fatal("Metaknowledge should be unplayable in alter-ego form")
	}
}

// TestChimichangaTruck: after an identity recovers, the truck exhausts
// to ready it. Contract test via the support React hook.
func TestChimichangaTruck(t *testing.T) {
	g := newDeadpoolGame(t, 44)
	p := g.Players[0]
	p.Exhausted = true // recovering exhausts the identity

	s := &engine.Support{ID: "support-90", Code: "44008", Owner: p.ID}
	g.Supports[s.ID] = s
	p.Supports = append(p.Supports, s.ID)

	b := engine.LookupBehavior("44008")
	if b == nil || b.React == nil {
		t.Fatal("Chimichanga Truck should expose React")
	}
	msgs := b.React(g, s, engine.BasicRecover{Player: p.ID})
	if len(msgs) != 2 {
		t.Fatalf("Chimichanga Truck returned %d messages, want 2", len(msgs))
	}
	if ex, ok := msgs[0].(engine.ExhaustEntity); !ok || ex.ID != s.ID {
		t.Fatalf("first message = %#v, want ExhaustEntity on the truck", msgs[0])
	}
	if rd, ok := msgs[1].(engine.ReadyEntity); !ok || rd.ID != p.ID {
		t.Fatalf("second message = %#v, want ReadyEntity on the identity", msgs[1])
	}

	// An exhausted truck cannot trigger again.
	s.Exhausted = true
	if msgs := b.React(g, s, engine.BasicRecover{Player: p.ID}); len(msgs) != 0 {
		t.Fatalf("exhausted truck returned %d messages, want 0", len(msgs))
	}
}

// TestInvoluntaryProcedures: damage to Deadpool places threat on the
// nemesis side scheme; damage to anything else does not. Contract test
// via the side scheme React hook.
func TestInvoluntaryProcedures(t *testing.T) {
	g := newDeadpoolGame(t, 44)
	p := g.Players[0]

	s := &engine.SideScheme{ID: "sidescheme-90", Code: "44034"}
	g.SideSchemes[s.ID] = s

	b := engine.LookupBehavior("44034")
	if b == nil || b.React == nil {
		t.Fatal("Involuntary Procedures should expose React")
	}
	msgs := b.React(g, s, engine.DamageEntity{Target: p.ID, Damage: 2, Source: "villain-1"})
	if len(msgs) != 1 {
		t.Fatalf("Involuntary Procedures returned %d messages, want 1", len(msgs))
	}
	if th, ok := msgs[0].(engine.SchemeThreat); !ok || th.Scheme != s.ID || th.N != 1 {
		t.Fatalf("message = %#v, want SchemeThreat +1 on Involuntary Procedures", msgs[0])
	}

	// Damage to someone else stays silent.
	for id := range g.Villains {
		if msgs := b.React(g, s, engine.DamageEntity{Target: id, Damage: 2}); len(msgs) != 0 {
			t.Fatalf("villain damage returned %d messages, want 0", len(msgs))
		}
		break
	}
	// Zero damage is not "any amount of damage".
	if msgs := b.React(g, s, engine.DamageEntity{Target: p.ID, Damage: 0}); len(msgs) != 0 {
		t.Fatalf("zero damage returned %d messages, want 0", len(msgs))
	}
}
