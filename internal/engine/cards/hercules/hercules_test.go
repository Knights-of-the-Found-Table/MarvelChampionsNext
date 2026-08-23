package hercules_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/hercules"
)

func herculesDeck() map[string]int {
	return map[string]int{
		"59008": 1, "59009": 3, "59010": 2, "59011": 2, "59012": 1,
		"59013": 1, "59014": 1, "59015": 1, "59016": 2, "59017": 1,
	}
}

func newHerculesGame(t *testing.T) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed: 59, ScenarioID: "01097",
		Players: []engine.PlayerSpec{{Name: "Hercules", HeroBase: "59001", Deck: herculesDeck()}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// Contract test: invoke HeroSetup and the alter-ego action directly. The top
// Labor is deterministic and leaves the Labor deck as a RevealEncounterCard,
// without using the turn or encounter state machine.
func TestHerculesRevealsTopLaborFromDedicatedDeck(t *testing.T) {
	g := newHerculesGame(t)
	p := g.Players[0]
	identity := engine.LookupBehavior("59001")
	if identity == nil || identity.HeroSetup == nil || identity.HeroAbilities == nil || identity.React == nil {
		t.Fatal("Hercules should expose setup, HeroAbilities, and React hooks")
	}

	identity.HeroSetup(g, p)
	if len(p.SenseDeck) != 3 || len(p.SideDiscard) != 3 {
		t.Fatalf("side decks = Labor %d / Gift %d, want 3 / 3", len(p.SenseDeck), len(p.SideDiscard))
	}
	p.SenseDeck = engine.CardList{
		{ID: g.NextCardID(), Code: "59002", Owner: p.ID},
		{ID: g.NextCardID(), Code: "59003", Owner: p.ID},
		{ID: g.NextCardID(), Code: "59004", Owner: p.ID},
	}
	p.Side = engine.SideAlterEgo
	abilities := identity.HeroAbilities(g, p)
	if len(abilities) != 1 || abilities[0].Execute == nil || !abilities[0].AlterEgoOnly {
		t.Fatalf("HeroAbilities = %#v, want one alter-ego Labor action", abilities)
	}

	msgs := abilities[0].Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Labor action returned %d messages, want one reveal", len(msgs))
	}
	reveal, ok := msgs[0].(engine.RevealEncounterCard)
	if !ok || reveal.Player != p.ID || reveal.Card.Code != "59002" {
		t.Fatalf("Labor action message = %#v, want Defeat the Hydra reveal", msgs[0])
	}
	if len(p.SenseDeck) != 2 || p.SenseDeck[0].Code != "59003" {
		t.Fatalf("remaining Labor deck = %#v, want Embody Pathos on top", p.SenseDeck)
	}
}

// Contract test: simulate a Hydra Labor attached to a defeated minion, then
// call the identity React hook directly. Resolution records the Labor in the
// victory-display proxy, puts the deterministic top Gift into play, readies
// Hercules, and offers the optional form change.
func TestHerculesCompletesLaborAndClaimsTopGift(t *testing.T) {
	g := newHerculesGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Exhausted = true
	p.SideDiscard = engine.CardList{
		{ID: g.NextCardID(), Code: "59005", Owner: p.ID},
		{ID: g.NextCardID(), Code: "59006", Owner: p.ID},
	}

	minionID := g.NextEntityID("minion")
	minion := &engine.Minion{ID: minionID, Code: "01113", MaxHP: 8, EngagedWith: p.ID}
	g.Minions[minionID] = minion
	attachmentID := g.NextEntityID("attachment")
	attachment := &engine.Attachment{ID: attachmentID, Code: "59002", Target: minionID}
	g.Attachments[attachmentID] = attachment
	minion.Attachments = append(minion.Attachments, attachmentID)
	g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: "59002", Owner: p.ID})

	identity := engine.LookupBehavior("59001")
	msgs := identity.React(g, p, engine.MinionDefeated{MinionID: minionID})
	if g.Attachments[attachmentID] != nil {
		t.Fatal("completed Labor attachment should leave play")
	}
	if len(p.ObligationRemoved) != 1 || p.ObligationRemoved[0].Code != "59002" {
		t.Fatalf("victory-display proxy = %#v, want completed Defeat the Hydra", p.ObligationRemoved)
	}
	if len(p.SideDiscard) != 1 || p.SideDiscard[0].Code != "59006" {
		t.Fatalf("Gift deck = %#v, want Shield of Perseus remaining", p.SideDiscard)
	}
	if len(p.Upgrades) != 1 {
		t.Fatalf("upgrades = %d, want claimed Gift", len(p.Upgrades))
	}
	gift := g.Upgrades[p.Upgrades[0]]
	if gift == nil || gift.Code != "59005" {
		t.Fatalf("claimed Gift = %#v, want Nemean Lion Skin", gift)
	}
	if p.MaxHP != 16 {
		t.Fatalf("MaxHP = %d, want 16 after Nemean Lion Skin", p.MaxHP)
	}
	if len(msgs) != 3 {
		t.Fatalf("Atonement returned %d messages, want draw, ready, and form question", len(msgs))
	}
	if draw, ok := msgs[0].(engine.DrawCards); !ok || draw.Player != p.ID || draw.N != 4 {
		t.Fatalf("first Atonement message = %#v, want DrawCards 4", msgs[0])
	}
	if ready, ok := msgs[1].(engine.ReadyEntity); !ok || ready.ID != p.ID {
		t.Fatalf("second Atonement message = %#v, want ReadyEntity", msgs[1])
	}
	if ask, ok := msgs[2].(engine.AskQuestion); !ok || ask.Question == nil || len(ask.Question.Choices) != 2 {
		t.Fatalf("third Atonement message = %#v, want optional flip question", msgs[2])
	}
}
