package aos_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

func nickDeck() map[string]int {
	return map[string]int{
		"50035a": 1, "50036": 1, "50037": 2, "50038": 3,
		"50039": 2, "50040": 1, "50041": 1, "50042": 1,
		"50043": 1, "50044": 1, "50045": 1, "50046": 1,
	}
}

func TestNickFuryImplemented(t *testing.T) {
	if !engine.Implemented("50034a") {
		t.Fatal("Nick Fury should count as implemented")
	}
	if !engine.Implemented("50035a") || !engine.Implemented("50035b") {
		t.Fatal("both Nick Fury suit faces should count as implemented")
	}
}

// Contract test: call HeroAbilities and Execute directly, inspect the
// package-local flip message, then feed it to the suit behavior. No game turn
// menu is traversed.
func TestNickFuryInfiltrateChangesSuitToStealth(t *testing.T) {
	g := newAOSGame(t, "50034", nickDeck())
	p := g.Players[0]
	p.Side = engine.SideAlterEgo

	var suit *engine.Upgrade
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && len(u.Code) >= 5 && u.Code[:5] == "50035" {
			suit = u
			break
		}
	}
	if suit == nil {
		suit = &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "50035a", Owner: p.ID}
		g.Upgrades[suit.ID] = suit
		p.Upgrades = append(p.Upgrades, suit.ID)
	}
	suit.Code = "50035a"

	identity := engine.LookupBehavior("50034")
	if identity == nil || identity.HeroAbilities == nil {
		t.Fatal("Nick Fury should expose HeroAbilities")
	}
	abilities := identity.HeroAbilities(g, p)
	if len(abilities) != 1 || abilities[0].Execute == nil {
		t.Fatalf("Nick Fury abilities = %d, want executable Infiltrate", len(abilities))
	}
	if !abilities[0].AlterEgoOnly {
		t.Fatal("Infiltrate should be alter-ego-only")
	}

	msgs := abilities[0].Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Infiltrate returned %d messages, want 1", len(msgs))
	}
	flip, ok := msgs[0].(engine.AddEntityCounter)
	if !ok || flip.ID != suit.ID || flip.N != 0 {
		t.Fatalf("Infiltrate message = %#v, want neutral suit flip signal", msgs[0])
	}

	suitBehavior := engine.LookupBehavior("50035a")
	if suitBehavior == nil || suitBehavior.React == nil {
		t.Fatal("Assault suit should react to the flip signal")
	}
	suitBehavior.React(g, suit, msgs[0])
	if suit.Code != "50035b" {
		t.Fatalf("suit code = %s, want Stealth face 50035b", suit.Code)
	}
}

func TestNickFuryGatherIntelAddsSuitThreatMessage(t *testing.T) {
	g := newAOSGame(t, "50034", nickDeck())
	p := g.Players[0]
	var suit *engine.Upgrade
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && len(u.Code) >= 5 && u.Code[:5] == "50035" {
			suit = u
			break
		}
	}
	identity := engine.LookupBehavior("50034")
	if suit == nil {
		// Verify Suit Up's direct message contract, then materialize the
		// upgrade for the independent Gather Intel contract below.
		card := engine.Card{ID: g.NextCardID(), Code: "50035a", Owner: p.ID}
		p.Deck = append(p.Deck, card)
		setupMsgs := identity.HeroSetup(g, p)
		if len(setupMsgs) != 1 {
			t.Fatalf("Suit Up returned %d messages, want 1", len(setupMsgs))
		}
		if enter, ok := setupMsgs[0].(engine.UpgradeEnterPlay); !ok || len(enter.Card.Code) < 5 || enter.Card.Code[:5] != "50035" {
			t.Fatalf("Suit Up message = %#v, want suit UpgradeEnterPlay", setupMsgs[0])
		}
		suit = &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "50035a", Owner: p.ID}
		g.Upgrades[suit.ID] = suit
		p.Upgrades = append(p.Upgrades, suit.ID)
	}
	msgs := identity.React(g, p, engine.BasicThwart{Player: p.ID, N: 2, Target: g.MainScheme.ID})
	if len(msgs) != 1 {
		t.Fatalf("Gather Intel returned %d messages, want 1", len(msgs))
	}
	add, ok := msgs[0].(engine.AddEntityCounter)
	if !ok || add.ID != suit.ID || add.N != 1 {
		t.Fatalf("Gather Intel message = %#v, want +1 suit threat", msgs[0])
	}
}
