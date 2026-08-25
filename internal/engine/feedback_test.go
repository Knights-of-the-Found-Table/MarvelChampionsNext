package engine_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

// Legal Practice (01023): playing it must first ask the player to choose
// and discard up to 5 hand cards, then remove 1 threat from a scheme for
// each card discarded — not flat-remove 1 threat.
func TestLegalPracticeDiscardsThenRemovesThreatPerCard(t *testing.T) {
	g := newRulesGame(t, 42)
	p := g.Players[0]
	// Deterministic hand (Legal Practice itself is already on the discard
	// pile when OnPlay resolves — handlePlayCard removes the event first).
	p.Hand = engine.CardList{
		{ID: g.NextCardID(), Code: "01088", Owner: p.ID},
		{ID: g.NextCardID(), Code: "01089", Owner: p.ID},
		{ID: g.NextCardID(), Code: "01088", Owner: p.ID},
	}
	g.MainScheme.Threat = 5
	g.MainScheme.MaxThreat = 10

	b := engine.LookupBehavior("01023")
	if b == nil || b.OnPlay == nil {
		t.Fatal("Legal Practice should register an OnPlay hook")
	}
	g.Push(b.OnPlay(g, p)...)

	// Answer the setup/round questions blocking the queue until the Legal
	// Practice discard question surfaces.
	var discardQ *engine.PendingQuestion
	for i := 0; i < 50; i++ {
		pq := g.Pending()
		if pq == nil {
			t.Fatal("queue drained before the Legal Practice discard question")
		}
		if pq.Question.Validate == "threatPerDiscard:5" {
			discardQ = pq
			break
		}
		if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
			t.Fatalf("answer %q: %v", pq.Question.Prompt, err)
		}
	}
	if discardQ == nil {
		t.Fatal("Legal Practice should ask which cards to discard")
	}
	q := discardQ.Question
	if q.Type != "choose_n" {
		t.Fatalf("discard question type = %q, want choose_n", q.Type)
	}
	if len(q.Choices) != 3 {
		t.Fatalf("discard question offers %d choices, want the 3 hand cards", len(q.Choices))
	}
	if err := g.Answer(discardQ.Player, []string{q.Choices[0].ID, q.Choices[1].ID}); err != nil {
		t.Fatalf("answer discard selection: %v", err)
	}

	pq := g.Pending()
	if pq == nil {
		t.Fatal("after discarding, Legal Practice should ask which scheme loses threat")
	}
	schemeQ := pq.Question
	if schemeQ.PromptKey != "q.removeNThreatFromWhichScheme" {
		t.Fatalf("follow-up prompt key = %q, want q.removeNThreatFromWhichScheme", schemeQ.PromptKey)
	}
	if len(schemeQ.Choices) == 0 {
		t.Fatal("scheme question has no choices")
	}
	if err := g.Answer(pq.Player, []string{schemeQ.Choices[0].ID}); err != nil {
		t.Fatalf("answer scheme pick: %v", err)
	}

	if got := len(p.Discard); got < 2 {
		t.Fatalf("discard pile = %d cards, want at least the 2 selected (villain phase may add more)", got)
	}
	if got := len(p.Hand); got != 1 {
		t.Fatalf("hand = %d cards, want 1 remaining", got)
	}
	if got := g.MainScheme.Threat; got != 3 {
		t.Fatalf("main scheme threat = %d, want 5-2=3", got)
	}
}

// The defender question must always offer the attacked identity itself as
// a card-highlighted "take the attack" choice — even when it cannot defend
// (alter-ego form, exhausted) — while in hero form the actual defense
// claims the identity-card highlight (listed before "take").
func TestDefenderQuestionTakeChoiceHighlightsIdentity(t *testing.T) {
	g := newRulesGame(t, 42)
	p := g.Players[0]
	p.Side = engine.SideAlterEgo
	enemy := g.Enemies()[0]

	q := g.AttackQuestion(enemy, 3, p, "attack")
	var take *engine.Choice
	for i := range q.Choices {
		if q.Choices[i].ID == "take" {
			take = &q.Choices[i]
		}
	}
	if take == nil {
		t.Fatal("defender question should always offer taking the attack")
	}
	if take.SourceID != p.ID {
		t.Fatalf("take choice sourceId = %q, want the attacked player %q", take.SourceID, p.ID)
	}
	if take.CardCode != p.AlterEgoCode {
		t.Fatalf("take choice cardCode = %q, want the alter-ego face %q", take.CardCode, p.AlterEgoCode)
	}
	if take.Kind != engine.ChoiceBasicPower {
		t.Fatalf("take choice kind = %q, want basic_power so the board highlights the identity card", take.Kind)
	}
	for _, c := range q.Choices {
		if c.ID == "hero-defend" {
			t.Fatal("alter-ego form must not offer hero defense")
		}
	}

	// Hero form: the defense option must come first so the board maps the
	// identity card (same sourceId) to the defense, not to "take".
	p.Side = engine.SideHero
	q = g.AttackQuestion(enemy, 3, p, "attack")
	takeIdx, defendIdx := -1, -1
	for i, c := range q.Choices {
		switch c.ID {
		case "take":
			takeIdx = i
		case "hero-defend":
			defendIdx = i
		}
	}
	if defendIdx < 0 || takeIdx < 0 || defendIdx > takeIdx {
		t.Fatalf("choice order: hero-defend=%d take=%d, want hero-defend before take", defendIdx, takeIdx)
	}
	hd := q.Choices[defendIdx]
	if hd.SourceID != p.ID || hd.CardCode != p.HeroCode {
		t.Fatalf("hero-defend sourceId=%q cardCode=%q, want the hero identity %q/%q",
			hd.SourceID, hd.CardCode, p.ID, p.HeroCode)
	}
}

// Klaw main scheme stage 1B (01116b): its When Revealed — discard
// encounter cards until a minion is discarded, put it into play engaged
// with the first player — must resolve during setup, right after the
// encounter deck is shuffled (1A: "... Shuffle ... Advance to stage 1B").
func TestKlawStage1BRevealsMinionAtSetup(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       11,
		ScenarioID: "01116",
		Players: []engine.PlayerSpec{
			{Name: "P1", HeroBase: "01001", Deck: fillerDeck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	// NewGame runs until the first blocking question (mulligan). By then
	// the 1B flip must already have spawned its minion.
	if len(g.Minions) == 0 {
		t.Fatal("Klaw 1B When Revealed should have put a minion into play during setup")
	}
	for _, mn := range g.Minions {
		if mn.EngagedWith == "" {
			t.Fatalf("minion %s should be engaged with the first player", mn.Code)
		}
	}
}
