package ghostspider_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/ghostspider"
)

func gwDeck() map[string]int {
	return map[string]int{
		"27002": 3, "27003": 1, "27004": 3, "27005": 2, "27006": 2,
		"27007": 1, "27008": 1, "27009": 2, "27010": 1, "27011": 1,
	}
}

func newGWGame(t *testing.T) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed: 27, ScenarioID: "01097",
		Players: []engine.PlayerSpec{{Name: "Ghost-Spider", HeroBase: "27001", Deck: gwDeck()}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// Contract test: Dizzying Reflexes readies an exhausted Ghost-Spider after
// she plays an Interrupt/Response event, once per phase, and ignores plain
// action events.
func TestDizzyingReflexesReadyOncePerPhase(t *testing.T) {
	g := newGWGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Exhausted = true
	g.UsedThisTurn = map[string]bool{}

	b := engine.LookupBehavior("27001")
	if b == nil || b.React == nil {
		t.Fatal("Ghost-Spider should expose a React hook")
	}
	// 27005 Pirouette and Punch is an Interrupt event.
	msgs := b.React(g, p, engine.EventPlayed{Player: p.ID, Card: engine.Card{Code: "27005", Owner: p.ID}})
	if len(msgs) != 1 {
		t.Fatalf("interrupt event returned %d messages, want one ReadyEntity", len(msgs))
	}
	if r, ok := msgs[0].(engine.ReadyEntity); !ok || r.ID != p.ID {
		t.Fatalf("message = %#v, want ReadyEntity for Ghost-Spider", msgs[0])
	}
	if !g.UsedThisTurn["gw-reflexes"] {
		t.Fatal("Dizzying Reflexes did not record its per-phase use")
	}
	// Second interrupt event in the same phase does not retrigger.
	if msgs := b.React(g, p, engine.EventPlayed{Player: p.ID, Card: engine.Card{Code: "27009", Owner: p.ID}}); msgs != nil {
		t.Fatalf("second event retriggered Dizzying Reflexes: %#v", msgs)
	}
	// Phase reset re-arms it.
	g.UsedThisTurn = map[string]bool{}
	p.Exhausted = true
	if msgs := b.React(g, p, engine.EventPlayed{Player: p.ID, Card: engine.Card{Code: "27005", Owner: p.ID}}); len(msgs) != 1 {
		t.Fatalf("after phase reset, returned %d messages, want 1", len(msgs))
	}
	// A pure Action event (27003 Parental Guidance) never triggers.
	g.UsedThisTurn = map[string]bool{}
	if msgs := b.React(g, p, engine.EventPlayed{Player: p.ID, Card: engine.Card{Code: "27003", Owner: p.ID}}); msgs != nil {
		t.Fatalf("action event triggered Dizzying Reflexes: %#v", msgs)
	}
	// Alter-ego form never triggers.
	p.Side = engine.SideAlterEgo
	if msgs := b.React(g, p, engine.EventPlayed{Player: p.ID, Card: engine.Card{Code: "27005", Owner: p.ID}}); msgs != nil {
		t.Fatalf("alter-ego event triggered Dizzying Reflexes: %#v", msgs)
	}
}

// Contract test: Pirouette and Punch cancels a revealed treachery and deals
// 1 + its boost icons to the villain.
func TestPirouetteAndPunchCancelsTreachery(t *testing.T) {
	g := newGWGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	b := engine.LookupBehavior("27005")
	if b == nil || b.TreacheryInterrupt == nil {
		t.Fatal("Pirouette and Punch should expose TreacheryInterrupt")
	}
	// 27029 In Cold Blood: 1 boost icon.
	repl := b.TreacheryInterrupt(g, p, engine.Card{Code: "27029"})
	if len(repl) != 2 {
		t.Fatalf("replacement = %d messages, want damage + cancellation", len(repl))
	}
	dmg, ok := repl[0].(engine.DamageEntity)
	if !ok || dmg.Damage != 2 {
		t.Fatalf("first message = %#v, want 2 damage to the villain", repl[0])
	}
	if res, ok := repl[1].(engine.TreacheryResolve); !ok || !res.Cancelled {
		t.Fatalf("second message = %#v, want cancelled TreacheryResolve", repl[1])
	}
}

// Contract test: Ticket to the Multiverse discards the hand, shuffles the
// discard pile into the deck, redraws to hand size and readies the identity.
func TestTicketToTheMultiverse(t *testing.T) {
	g := newGWGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	u := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: "27008", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)

	b := engine.LookupBehavior("27008")
	if b == nil || b.Abilities == nil {
		t.Fatal("Ticket to the Multiverse should expose Abilities")
	}
	abilities := b.Abilities(g, u)
	if len(abilities) != 1 || abilities[0].Execute == nil {
		t.Fatalf("abilities = %#v, want one executable action", abilities)
	}
	handBefore := len(p.Hand)
	discardBefore := len(p.Discard)
	msgs := abilities[0].Execute(g, u.ID)
	// DiscardControlled + DiscardCards + Shuffle + Draw + Ready.
	if len(msgs) != 5 {
		t.Fatalf("Ticket returned %d messages, want 5, hand=%d discard=%d", len(msgs), handBefore, discardBefore)
	}
	if len(p.Discard) != 0 {
		t.Fatalf("discard pile should have been shuffled into the deck, still has %d", len(p.Discard))
	}
	draw, ok := msgs[3].(engine.DrawCards)
	if !ok || draw.N != p.HandSize(g) {
		t.Fatalf("fourth message = %#v, want DrawCards up to hand size", msgs[3])
	}
	if r, ok := msgs[4].(engine.ReadyEntity); !ok || r.ID != p.ID {
		t.Fatalf("fifth message = %#v, want ReadyEntity for the identity", msgs[4])
	}
}

// Contract test: Silk discards the first treachery from the encounter deck
// when another Web-Warrior card is controlled (the identity counts).
func TestSilkSearchesTreachery(t *testing.T) {
	g := newGWGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero // Ghost-Spider herself is the other Web-Warrior card
	b := engine.LookupBehavior("27010")
	if b == nil || b.OnPlay == nil {
		t.Fatal("Silk should expose OnPlay")
	}
	ally := &engine.Ally{ID: g.NextEntityID(engine.KindAlly), Code: "27010", Owner: p.ID, MaxHP: 2}
	g.Allies[ally.ID] = ally
	p.Allies = append(p.Allies, ally.ID)

	// Guarantee a treachery on top of the encounter deck.
	g.EncounterDeck = append(engine.CardList{engine.Card{ID: "test-treachery", Code: "27029"}}, g.EncounterDeck...)
	msgs := b.OnPlay(g, ally)
	if len(msgs) != 1 {
		t.Fatalf("Silk returned %d messages, want one DiscardEncounterCard", len(msgs))
	}
	if d, ok := msgs[0].(engine.DiscardEncounterCard); !ok || d.Card.Code != "27029" {
		t.Fatalf("message = %#v, want DiscardEncounterCard for In Cold Blood", msgs[0])
	}
}
