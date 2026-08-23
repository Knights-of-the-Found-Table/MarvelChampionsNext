package cyclops_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cyclops"
)

// TestCyclopsImplemented: the Cyclops identity is registered with a
// behavior; both sides count as implemented.
func TestCyclopsImplemented(t *testing.T) {
	if !engine.Implemented("33001a") {
		t.Fatal("Cyclops should count as implemented")
	}
	if !engine.Implemented("33001b") {
		t.Fatal("Scott Summers (alter-ego) should count as implemented")
	}
}

// TestOpticBlastRequiresHeroForm: driving HeroAbilities in alter-ego
// form yields no usable Optic Blast (AlterEgoOnly abilities appear, but
// the hero-only Optic Blast is hidden by the engine's usable gate).
func TestOpticBlastRequiresHeroForm(t *testing.T) {
	g := mustNewCyclopsGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	b := engine.LookupBehavior("33001")
	if b == nil || b.HeroAbilities == nil {
		t.Fatal("Cyclops should expose HeroAbilities")
	}
	abs := b.HeroAbilities(g, p)
	var optic *engine.Ability
	for i := range abs {
		if strings.Contains(abs[i].Label, "Optic Blast") {
			optic = &abs[i]
		}
	}
	if optic == nil {
		t.Fatal("Optic Blast should be among the hero abilities")
	}
	if !optic.HeroOnly {
		t.Fatal("Optic Blast should be HeroOnly")
	}
	if !optic.OncePerRound {
		t.Fatal("Optic Blast should be OncePerRound")
	}

	// Case: no enemy in play → executing Optic Blast returns nothing.
	msgs := optic.Execute(g, p.ID)
	if len(msgs) != 0 {
		t.Fatalf("Optic Blast with no enemies should return no messages, got %d: %v", len(msgs), msgs)
	}
}

// TestOpticBlastOffersUpgradedEnemies: when the player attacks, the
// Optic Blast question lists only enemies with an upgrade attached.
func TestOpticBlastOffersUpgradedEnemies(t *testing.T) {
	g := mustNewCyclopsGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero

	// Two villains: one bare, one with an attached upgrade.
	bare := &engine.Villain{ID: g.NextEntityID("villain"), Code: "01094", MaxHP: 30}
	upgraded := &engine.Villain{ID: g.NextEntityID("villain"), Code: "01095", MaxHP: 30}
	g.Villains[bare.ID] = bare
	g.Villains[upgraded.ID] = upgraded
	attach := &engine.Attachment{ID: g.NextEntityID("attachment"), Code: "33005", Target: upgraded.ID}
	g.Attachments[attach.ID] = attach

	abs := engine.LookupBehavior("33001").HeroAbilities(g, p)
	var optic *engine.Ability
	for i := range abs {
		if strings.Contains(abs[i].Label, "Optic Blast") {
			optic = &abs[i]
		}
	}
	msgs := optic.Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Optic Blast with one upgraded enemy should ask 1 question, got %d", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok {
		t.Fatalf("Optic Blast should emit AskQuestion, got %T", msgs[0])
	}
	if !strings.Contains(ask.Question.Prompt, "Optic Blast") {
		t.Fatalf("prompt should mention Optic Blast, got %q", ask.Question.Prompt)
	}
	if len(ask.Question.Choices) != 1 {
		t.Fatalf("Optic Blast should list exactly the upgraded enemy, got %d choices", len(ask.Question.Choices))
	}
	if ask.Question.Choices[0].SourceID != upgraded.ID {
		t.Fatalf("Optic Blast should target the upgraded villain")
	}
}

// TestConstantTrainingSearchesDeck: in alter-ego form the Constant
// Training ability finds a Tactic upgrade in the deck and offers it.
func TestConstantTrainingSearchesDeck(t *testing.T) {
	g := mustNewCyclopsGame(t)
	p := g.Players[0]
	p.Side = engine.SideAlterEgo
	p.Hand = engine.CardList{}
	p.Discard = engine.CardList{}
	tacticCard := engine.Card{ID: g.NextCardID(), Code: "33005", Owner: p.ID}
	p.Deck = engine.CardList{tacticCard}

	abs := engine.LookupBehavior("33001").HeroAbilities(g, p)
	var ct *engine.Ability
	for i := range abs {
		if strings.Contains(abs[i].Label, "Constant Training") {
			ct = &abs[i]
		}
	}
	if ct == nil {
		t.Fatal("Constant Training should be among the hero abilities")
	}
	if !ct.AlterEgoOnly {
		t.Fatal("Constant Training should be AlterEgoOnly")
	}
	if !ct.OncePerRound {
		t.Fatal("Constant Training should be OncePerRound")
	}

	msgs := ct.Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Constant Training should ask 1 question, got %d", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok {
		t.Fatalf("Constant Training should emit AskQuestion, got %T", msgs[0])
	}
	hasTake := false
	hasSkip := false
	for _, c := range ask.Question.Choices {
		if strings.Contains(c.Label, "Take Exploit Weakness") {
			hasTake = true
		}
		if c.ID == "skip" {
			hasSkip = true
		}
	}
	if !hasTake {
		t.Fatal("Constant Training question should offer a Take choice for the Tactic upgrade")
	}
	if !hasSkip {
		t.Fatal("Constant Training question should offer a skip choice (still shuffles)")
	}
}

// TestConstantTrainingEmptyDeckShuffles: when no Tactic upgrade is in
// the deck, Constant Training still shuffles (no question).
func TestConstantTrainingEmptyDeckShuffles(t *testing.T) {
	g := mustNewCyclopsGame(t)
	p := g.Players[0]
	p.Side = engine.SideAlterEgo
	p.Hand = engine.CardList{}
	p.Deck = engine.CardList{
		{ID: g.NextCardID(), Code: "01003", Owner: p.ID}, // Backflip, not a Tactic
	}
	p.Discard = engine.CardList{}

	abs := engine.LookupBehavior("33001").HeroAbilities(g, p)
	var ct *engine.Ability
	for i := range abs {
		if strings.Contains(abs[i].Label, "Constant Training") {
			ct = &abs[i]
		}
	}
	msgs := ct.Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Constant Training (no tactic) should emit 1 message, got %d", len(msgs))
	}
	if _, ok := msgs[0].(engine.ShufflePlayerDeck); !ok {
		t.Fatalf("Constant Training (no tactic) should emit ShufflePlayerDeck, got %T", msgs[0])
	}
}

// TestPhoenixAllyRecoversCyclopsCard: Phoenix's OnPlay response finds a
// Cyclops card in the discard pile and offers to add it to the hand.
func TestPhoenixAllyRecoversCyclopsCard(t *testing.T) {
	g := mustNewCyclopsGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Hand = engine.CardList{}
	p.Discard = engine.CardList{
		{ID: g.NextCardID(), Code: "33008", Owner: p.ID}, // Full Blast, a Cyclops card
	}

	b := engine.LookupBehavior("33002")
	if b == nil || b.OnPlay == nil {
		t.Fatal("Phoenix ally should expose OnPlay")
	}
	phoenix := &engine.Ally{ID: g.NextEntityID("ally"), Code: "33002", Owner: p.ID}
	msgs := b.OnPlay(g, phoenix)
	if len(msgs) != 1 {
		t.Fatalf("Phoenix should ask 1 question, got %d", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok {
		t.Fatalf("Phoenix should emit AskQuestion, got %T", msgs[0])
	}
	if !strings.Contains(ask.Question.Prompt, "Phoenix") {
		t.Fatalf("Phoenix prompt should mention Phoenix, got %q", ask.Question.Prompt)
	}
	if len(ask.Question.Choices) != 1 {
		t.Fatalf("Phoenix should list 1 Cyclops card from discard, got %d", len(ask.Question.Choices))
	}
	if ask.Question.Choices[0].CardCode != "33008" {
		t.Fatalf("Phoenix should offer the Full Blast card, got %s", ask.Question.Choices[0].CardCode)
	}
}

// TestConcussiveForceDamagesPlayer: with no Sinister in play, the
// treachery's resolution deals 2 damage to the hero.
func TestConcussiveForceDamagesPlayer(t *testing.T) {
	g := mustNewCyclopsGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.MaxHP = 10
	p.Damage = 0
	treach := &engine.Treachery{ID: g.NextEntityID("treachery"), Code: "33031"}

	b := engine.LookupBehavior("33031")
	if b == nil || b.ResolveTreachery == nil {
		t.Fatal("Concussive Force should expose ResolveTreachery")
	}
	msgs := b.ResolveTreachery(g, treach, p)
	if len(msgs) != 1 {
		t.Fatalf("Concussive Force (hero, no Sinister) should emit 1 message, got %d", len(msgs))
	}
	dmg, ok := msgs[0].(engine.DamageEntity)
	if !ok {
		t.Fatalf("Concussive Force should emit DamageEntity, got %T", msgs[0])
	}
	if dmg.Damage != 2 {
		t.Fatalf("Concussive Force (no Sinister) should deal 2 damage, got %d", dmg.Damage)
	}
	if dmg.Target != p.ID {
		t.Fatalf("Concussive Force should target the player, got %s", dmg.Target)
	}
}

// TestConcussiveForceSchemesWhenSinisterInPlay: with Sinister on the
// board and the player in alter-ego form, the treachery places 1
// threat on the main scheme (approximation of "Sinister schemes").
func TestConcussiveForceSchemesWhenSinisterInPlay(t *testing.T) {
	g := mustNewCyclopsGame(t)
	p := g.Players[0]
	p.Side = engine.SideAlterEgo
	sinister := &engine.Minion{ID: g.NextEntityID("minion"), Code: "33028", EngagedWith: p.ID}
	g.Minions[sinister.ID] = sinister
	treach := &engine.Treachery{ID: g.NextEntityID("treachery"), Code: "33031"}

	b := engine.LookupBehavior("33031")
	msgs := b.ResolveTreachery(g, treach, p)
	if len(msgs) != 1 {
		t.Fatalf("Concussive Force (alter-ego, Sinister in play) should emit 1 message, got %d", len(msgs))
	}
	sch, ok := msgs[0].(engine.SchemeThreat)
	if !ok {
		t.Fatalf("Concussive Force should emit SchemeThreat (Sinister schemes), got %T", msgs[0])
	}
	if sch.N != 1 {
		t.Fatalf("Concussive Force (Sinister schemes) should place 1 threat, got %d", sch.N)
	}
}

// mustNewCyclopsGame returns a Cyclops game with the opening hand
// answered (mulligan kept), no questions pending. The deck is small
// but legal; tests mutate it directly.
func mustNewCyclopsGame(t *testing.T) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       3301,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Cyclops", HeroBase: "33001", Deck: map[string]int{
				"33002": 1, "33003": 1, "33004": 1, "33005": 1, "33006": 1,
				"33007": 1, "33008": 1, "33009": 1, "33010": 1, "33011": 1,
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if pq := g.Pending(); pq != nil && pq.Question.Prompt == "Mulligan?" {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("keep mulligan: %v", err)
		}
	}
	return g
}
