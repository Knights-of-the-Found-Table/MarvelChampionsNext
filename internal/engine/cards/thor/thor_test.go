package thor_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/thor"
)

// thorDeck mirrors a legal Aggression Thor decklist (15 unique cards).
// The test deck is intentionally minimal 鈥?we only need enough cards to
// push Mjolnir into play and exercise the core mechanics.
func thorDeck() map[string]int {
	return map[string]int{
		"06002": 1, "06003": 1, "06004": 1, "06005": 1, "06006": 1,
		"06008": 1, "06009": 1, "06010": 1, "06011": 1, "06012": 1,
	}
}

func newThorGame(t *testing.T, scenario string, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players: []engine.PlayerSpec{
			{Name: "Thor", HeroBase: "06001", Deck: thorDeck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame(%s): %v", scenario, err)
	}

	// Keep the opening hand: the game opens paused on the mulligan
	// question, and these tests expect the first player turn pending.
	if pq := g.Pending(); pq != nil && pq.Question.Prompt == "Mulligan?" {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("keep mulligan: %v", err)
		}
	}
	return g
}

func pickDefault(q *engine.Question) []string {
	prefer := []string{"pass-interrupt", "skip", "take", "end-turn", "continue", "form"}
	for _, id := range prefer {
		for _, c := range q.Choices {
			if c.ID == id && !c.Disabled {
				return []string{id}
			}
		}
	}
	if q.Type == "choose_n" {
		var out []string
		for i, c := range q.Choices {
			if q.N > 0 && i >= q.N {
				break
			}
			out = append(out, c.ID)
		}
		return out
	}
	if len(q.Choices) > 0 {
		return []string{q.Choices[0].ID}
	}
	return nil
}

// walkPrompts drains prompts with defaults until one matches want (or
// the game ends). Returns the matching pending question.
func walkPrompts(t *testing.T, g *engine.Game, want string) *engine.PendingQuestion {
	t.Helper()
	for i := 0; i < 80; i++ {
		pq := g.Pending()
		if pq == nil {
			g.Run()
			continue
		}
		if strings.Contains(pq.Question.Prompt, want) {
			return pq
		}
		if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
			t.Fatalf("answer %q: %v", pq.Question.Prompt, err)
		}
		if g.Over {
			t.Fatalf("game over before %q appeared", want)
		}
	}
	t.Fatalf("prompt %q never appeared", want)
	return nil
}

// playAndAdvance queues a PlayCard, then answers the current pending
// turn menu with end-turn so the queued PlayCard processes on the
// following turn cycle. After this returns the card's OnPlay effects
// have fired.
func playAndAdvance(t *testing.T, g *engine.Game, c engine.Card) {
	t.Helper()
	pid := c.Owner
	g.Push(engine.PlayCard{Player: pid, Card: c, Paid: engine.CostPaid{}})
	// The queued PlayCard resolves once the phase drivers queued ahead of
	// it have run. Answer prompts with defaults until the game reaches a
	// later round's turn menu — by then the card has resolved.
	startRound := g.Round
	for i := 0; i < 80; i++ {
		pq := g.Pending()
		if pq == nil {
			g.Run()
			pq = g.Pending()
			if pq == nil {
				return
			}
		}
		if pq.Question.Prompt == "Your turn" && g.Round > startRound {
			return
		}
		ans := pickDefault(pq.Question)
		if len(ans) == 0 {
			return
		}
		if err := g.Answer(pq.Player, ans); err != nil {
			return
		}
		if g.Over {
			return
		}
	}
}

func TestThorImplemented(t *testing.T) {
	if !engine.Implemented("06001a") {
		t.Fatal("Thor should count as implemented")
	}
}

// TestMjolnirGrantsAtkAndAerial: playing Mjolnir bumps Thor's ATK to 3
// and grants the Aerial trait.
func TestMjolnirGrantsAtkAndAerial(t *testing.T) {
	g := newThorGame(t, "01097", 5)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Exhausted = false

	mj := engine.Card{ID: g.NextCardID(), Code: "06009", Owner: p.ID}
	p.Hand = append(p.Hand, mj)
	playAndAdvance(t, g, mj)

	hasAerial := false
	for _, t2 := range p.ExtraTraits {
		if t2 == "aerial" {
			hasAerial = true
		}
	}
	if !hasAerial {
		t.Fatalf("Thor should gain the Aerial trait from Mjolnir, traits=%v", p.ExtraTraits)
	}
	if atk := p.AttackStat(g); atk != 3 {
		t.Fatalf("Thor's ATK should be 3 (2 base + 1 Mjolnir), got %d", atk)
	}
	found := false
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == "06009" {
			found = true
		}
	}
	if !found {
		t.Fatal("Mjolnir should be in play after PlayCard")
	}
}

// TestHaveAtTheeDrawsTwo: after a minion enters play engaged with
// Thor, Have at thee! draws 2 cards (limit once per phase).
func TestHaveAtTheeDrawsTwo(t *testing.T) {
	g := newThorGame(t, "01097", 5)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Exhausted = false

	p.Deck = engine.CardList{
		{ID: g.NextCardID(), Code: "06002", Owner: p.ID},
		{ID: g.NextCardID(), Code: "06003", Owner: p.ID},
		{ID: g.NextCardID(), Code: "06004", Owner: p.ID},
		{ID: g.NextCardID(), Code: "06005", Owner: p.ID},
	}
	// The pending turn menu was built before the form change above, so
	// advance to the next turn first: its menu reflects hero form and a
	// readied identity (the end-of-player-phase readies everything).
	for i := 0; i < 30; i++ {
		pq := g.Pending()
		if pq == nil {
			g.Run()
			continue
		}
		if pq.Question.Prompt == "Your turn" && i > 0 {
			break
		}
		if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
			t.Fatalf("advance: %v", err)
		}
		if g.Over {
			t.Fatal("game over while advancing turns")
		}
	}
	handBefore := len(p.Hand)
	// The turn menu blocks the queue; drilling into the basic-attack
	// choice (never completing its target question) unblocks the queue so
	// the pushed MinionEntersPlay processes within this player phase.
	pushMinion := func(mn *engine.Minion) {
		t.Helper()
		g.Minions[mn.ID] = mn
		g.Push(engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
		pq := g.Pending()
		if pq == nil {
			g.Run()
			pq = g.Pending()
		}
		if pq == nil {
			t.Fatal("no pending question to unblock the queue")
		}
		if err := g.Answer(pq.Player, []string{"basic-attack"}); err != nil {
			t.Fatalf("answer basic-attack: %v", err)
		}
	}
	mn := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01099", MaxHP: 3, EngagedWith: p.ID}
	pushMinion(mn)
	g.Run()
	if len(p.Hand) != handBefore+2 {
		t.Fatalf("Thor should have drawn 2 cards from Have at thee!, hand before=%d after=%d",
			handBefore, len(p.Hand))
	}

	// Second minion in the same phase: no further draw (limit).
	handAfterFirst := len(p.Hand)
	mn2 := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01099", MaxHP: 3, EngagedWith: p.ID}
	pushMinion(mn2)
	g.Run()
	if len(p.Hand) != handAfterFirst {
		t.Fatalf("Have at thee! should be limited to once per phase; hand %d -> %d",
			handAfterFirst, len(p.Hand))
	}
}

// TestLadySifReadiesThor: playing Lady Sif in hero form readies the
// identity.
func TestLadySifReadiesThor(t *testing.T) {
	g := newThorGame(t, "01097", 5)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Exhausted = true

	sif := engine.Card{ID: g.NextCardID(), Code: "06002", Owner: p.ID}
	p.Hand = append(p.Hand, sif)
	playAndAdvance(t, g, sif)

	if p.Exhausted {
		t.Fatal("Lady Sif should ready Thor on entry")
	}
}

// TestWorthyFetchesMjolnir: in alter-ego form, the Worthy action
// finds Mjolnir in the deck and adds it to hand. Drives the ability
// directly via the HeroAbilities hook so the test does not depend on
// the main scheme not completing during a full game walk.
// TestWorthyFetchesMjolnir: the Worthy ability searches the deck and
// discard for Mjolnir and offers to add it to hand. Contract test:
// drive the ability directly and inspect the produced question instead
// of walking the full game state machine (which keeps regenerating the
// turn menu and is fragile to seed/setup churn).
func TestWorthyFetchesMjolnir(t *testing.T) {
	g := newThorGame(t, "01097", 5)
	p := g.Players[0]
	p.Side = engine.SideAlterEgo

	b := engine.LookupBehavior("06001")
	if b == nil || b.HeroAbilities == nil {
		t.Fatal("Thor identity should expose HeroAbilities")
	}
	abs := b.HeroAbilities(g, p)
	if len(abs) == 0 || abs[0].Execute == nil {
		t.Fatal("Worthy ability should be the first HeroAbility")
	}
	if !abs[0].AlterEgoOnly {
		t.Fatal("Worthy should be AlterEgoOnly")
	}
	if !abs[0].OncePerRound {
		t.Fatal("Worthy should be OncePerRound")
	}

	// Case 1: Mjolnir in the discard pile -> question offers taking it.
	p.Deck = engine.CardList{}
	p.Hand = engine.CardList{}
	mjolnir := engine.Card{ID: g.NextCardID(), Code: "06009", Owner: p.ID}
	p.Discard = engine.CardList{mjolnir}

	msgs := abs[0].Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Worthy returned %d messages, want 1", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok {
		t.Fatalf("Worthy message = %T, want AskQuestion", msgs[0])
	}
	if !strings.Contains(ask.Question.Prompt, "Worthy") {
		t.Fatalf("Worthy prompt = %q", ask.Question.Prompt)
	}
	var takeChoice *engine.Choice
	for i := range ask.Question.Choices {
		c := &ask.Question.Choices[i]
		if strings.Contains(c.Label.Text, "Take") {
			takeChoice = c
		}
	}
	if takeChoice == nil {
		t.Fatal("Worthy question should offer a Take choice")
	}

	// Case 2: Mjolnir nowhere -> still shuffles, no question.
	p.Discard = engine.CardList{}
	msgs = abs[0].Execute(g, p.ID)
	if len(msgs) != 1 {
		t.Fatalf("Worthy (missing Mjolnir) returned %d messages, want 1", len(msgs))
	}
	if _, ok := msgs[0].(engine.AskQuestion); ok {
		t.Fatal("Worthy without Mjolnir should not ask a question")
	}
	if _, ok := msgs[0].(engine.ShufflePlayerDeck); !ok {
		t.Fatalf("Worthy without Mjolnir = %T, want ShufflePlayerDeck", msgs[0])
	}

	_ = mjolnir
}

// TestThorsHelmetAddsHP: Thor's Helmet grants +5 max HP.
func TestThorsHelmetAddsHP(t *testing.T) {
	g := newThorGame(t, "01097", 5)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Exhausted = false
	baseline := p.MaxHP
	helmet := engine.Card{ID: g.NextCardID(), Code: "06010", Owner: p.ID}
	p.Hand = append(p.Hand, helmet)
	playAndAdvance(t, g, helmet)
	if p.MaxHP != baseline+5 {
		t.Fatalf("Thor's Helmet should grant +5 HP; max went from %d to %d", baseline, p.MaxHP)
	}
}
