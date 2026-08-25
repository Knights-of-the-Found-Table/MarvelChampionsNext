package engine_test

import (
	"strings"
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

// The main-scheme reveal/flip journal lines must carry the scheme as a
// structured card arg (its a face at spawn, the b face at the flip) so
// clients can show a card preview when hovering the line, and journal Seq
// numbers must be strictly increasing for snapshot diffing.
func TestSchemeJournalCarriesCardArgsAndSeq(t *testing.T) {
	g := newRulesGame(t, 42)

	var reveal, flip *engine.LogEntry
	for i := range g.Log {
		switch g.Log[i].Key {
		case "log.mainSchemeReveals":
			reveal = &g.Log[i]
		case "log.mainSchemeFlips":
			flip = &g.Log[i]
		}
	}
	if reveal == nil || flip == nil {
		t.Fatal("setup should journal both the scheme reveal and the stage flip")
	}
	if reveal.Args[0].Kind != "card" || reveal.Args[0].Code != "01097a" {
		t.Fatalf("reveal arg0 = %s/%s, want card 01097a", reveal.Args[0].Kind, reveal.Args[0].Code)
	}
	if flip.Args[0].Kind != "card" || flip.Args[0].Code != "01097b" {
		t.Fatalf("flip arg0 = %s/%s, want card 01097b", flip.Args[0].Kind, flip.Args[0].Code)
	}

	last := 0
	for i, e := range g.Log {
		if e.Seq <= last {
			t.Fatalf("log[%d] seq = %d, want > %d (strictly increasing)", i, e.Seq, last)
		}
		last = e.Seq
	}
}

// Klaw's main scheme has two stages. Completing stage 1 (threat reaching
// max) must advance to stage 2 (01117b) — the printed lose condition is on
// the final stage only ("If this stage is completed, the players lose").
func TestMainSchemeStageAdvanceOnMaxed(t *testing.T) {
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
	g.Push(engine.MainSchemeMaxed{Scheme: g.MainScheme.ID})
	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
		t.Fatalf("answer mulligan: %v", err)
	}
	if g.Over {
		t.Fatal("completing stage 1 of 2 must advance, not lose the game")
	}
	if g.MainScheme.Stage != 2 || g.MainScheme.Code != "01117b" {
		t.Fatalf("main scheme = stage %d (%s), want stage 2 (01117b)", g.MainScheme.Stage, g.MainScheme.Code)
	}
}

// Revealed encounter cards route correctly: a minion enters play and only
// reaches the encounter discard when defeated; a treachery resolves and
// lands in the discard right away.
func TestRevealEncounterRouting(t *testing.T) {
	g := newRulesGame(t, 42)
	p := g.Players[0]
	g.Push(engine.RevealEncounterCard{Player: p.ID, Card: engine.Card{ID: g.NextCardID(), Code: "01103"}})
	g.Push(engine.RevealEncounterCard{Player: p.ID, Card: engine.Card{ID: g.NextCardID(), Code: "01104"}})
	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
		t.Fatalf("answer mulligan: %v", err)
	}
	inDiscard := func(code string) bool {
		for _, c := range g.EncounterDiscard {
			if c.Code == code {
				return true
			}
		}
		return false
	}
	if inDiscard("01103") {
		t.Fatal("a revealed minion must not sit in the encounter discard while it is in play")
	}
	if !inDiscard("01104") {
		t.Fatal("a resolved treachery must land in the encounter discard")
	}
	found := false
	for _, mn := range g.Minions {
		if mn.Code == "01103" {
			found = true
		}
	}
	if !found {
		t.Fatal("Shocker should have entered play")
	}
	// The reveal journal lines carry the card as a structured arg so the
	// client can pop a preview / hover the name.
	for _, e := range g.Log {
		if e.Key == "log.reveals" {
			var cardArg *engine.Arg
			for i := range e.Args {
				if e.Args[i].Kind == "card" {
					cardArg = &e.Args[i]
				}
			}
			if cardArg == nil {
				t.Fatalf("log.reveals %q should carry a card arg: %+v", e.Text, e.Args)
			}
		}
	}
}

// The crisis icon is structured data now: revealing Crowd Control (01108,
// scheme_crisis: 1) must flag the side scheme so crisisInPlay locks the
// main scheme (enforcement asserted in feedback_internal_test.go).
func TestCrisisSchemeWiring(t *testing.T) {
	g := newRulesGame(t, 42)
	p := g.Players[0]
	g.Push(engine.RevealEncounterCard{Player: p.ID, Card: engine.Card{ID: g.NextCardID(), Code: "01108"}})
	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
		t.Fatalf("answer mulligan: %v", err)
	}
	crisis := false
	for _, s := range g.SideSchemes {
		if s.Code == "01108" && s.Crisis && !s.PlayerSide {
			crisis = true
		}
	}
	if !crisis {
		t.Fatal("01108 Crowd Control should carry the crisis flag from scheme_crisis data")
	}
}

// Data layer: scheme_crisis and the per-ally consequential damage costs
// (attack_cost / thwart_cost) parse into structured fields; records without
// the field default to consequential damage 1.
func TestSchemeCrisisAndConsequentialData(t *testing.T) {
	if d := engine.DB.MustLookup("01108"); !d.Crisis {
		t.Fatal("01108 scheme_crisis should parse into CardDef.Crisis")
	}
	wm := engine.DB.MustLookup("03014")
	if wm.AttackCost != 1 || wm.ConsequentialFor("attack") != 1 {
		t.Fatalf("03014 attack consequential = %d/%d, want 1/1", wm.AttackCost, wm.ConsequentialFor("attack"))
	}
	bc := engine.DB.MustLookup("01002")
	if bc.AttackCost != 0 || bc.ConsequentialFor("attack") != 1 {
		t.Fatalf("01002 missing cost fields should default to 1, got %d/%d", bc.AttackCost, bc.ConsequentialFor("attack"))
	}
}

// Wonder Man (03014): his attack cost is a discard, presented as the same
// select-and-confirm flow as payments (choose_n validated to exactly one
// discarded card, board-highlighted hand cards) — not a bespoke single
// select ask. The discard replaces his consequential damage.
func TestWonderManAttackDiscardCost(t *testing.T) {
	g := newRulesGame(t, 42)
	p := g.Players[0]
	p.Hand = engine.CardList{
		{ID: g.NextCardID(), Code: "01088", Owner: p.ID},
		{ID: g.NextCardID(), Code: "01089", Owner: p.ID},
	}
	wm := &engine.Ally{ID: g.NextEntityID("ally"), Code: "03014", Owner: p.ID, AttackVal: 3, MaxHP: 4}
	g.Allies[wm.ID] = wm
	p.Allies = append(p.Allies, wm.ID)

	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
		t.Fatalf("answer mulligan: %v", err)
	}
	pq = g.Pending()
	var atk *engine.Choice
	for i := range pq.Question.Choices {
		if strings.HasPrefix(pq.Question.Choices[i].ID, "ally-atk-") {
			atk = &pq.Question.Choices[i]
		}
	}
	if atk == nil || atk.Then == nil {
		t.Fatal("turn menu should offer Wonder Man's attack with an enemy-target step")
	}
	target := atk.Then.Choices[0]
	if target.Then == nil {
		t.Fatal("enemy target should chain into the discard-cost question")
	}
	q := target.Then
	if q.Type != "choose_n" {
		t.Fatalf("discard cost question type = %q, want choose_n (payment-style)", q.Type)
	}
	if q.Validate != "discardCost:1" {
		t.Fatalf("discard cost validate = %q, want discardCost:1", q.Validate)
	}
	if len(q.Choices) != 2 {
		t.Fatalf("discard cost offers %d choices, want the 2 hand cards", len(q.Choices))
	}
	for _, c := range q.Choices {
		if c.Kind != engine.ChoiceResource || c.SourceID == "" {
			t.Fatalf("hand-card choice %q should be a pay-style board-highlighted card pick", c.ID)
		}
	}

	// Answer through the full path (the leaf id carries the whole prefix):
	// ally → target → discard one card.
	if err := g.Answer(pq.Player, []string{q.Choices[0].ID}); err != nil {
		t.Fatalf("answer wonder man attack: %v", err)
	}
	if len(p.Hand) != 1 || len(p.Discard) != 1 {
		t.Fatalf("hand=%d discard=%d, want 1 remaining / 1 discarded", len(p.Hand), len(p.Discard))
	}
	if wm.Damage != 0 {
		t.Fatalf("wonder man took %d damage; the discard replaces his consequential damage", wm.Damage)
	}
	var villain *engine.Villain
	for _, v := range g.Villains {
		villain = v
		break
	}
	if villain == nil {
		t.Fatal("no villain in play")
	}
	if villain.HP() != villain.MaxHP-3 {
		t.Fatalf("villain hp = %d, want %d (atk 3)", villain.HP(), villain.MaxHP-3)
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
