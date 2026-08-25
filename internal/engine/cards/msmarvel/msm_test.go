package msmarvel_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/captainamerica"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/msmarvel"
)

// msmDeck mirrors a legal Protection Ms. Marvel decklist.
func msmDeck() map[string]int {
	return map[string]int{
		"05002": 1, "05003": 2, "05004": 2, "05005": 2, "05006": 1,
		"05007": 1, "05008": 1, "05009": 1, "05010": 1, "05011": 1,
		"05013": 2, "05014": 2, "05015": 2, "05017": 2, "05018": 1,
		"05019": 1, "05022": 1, "05023": 2, "05024": 2, "05030": 2,
	}
}

func newMsmGame(t *testing.T, scenario string, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players: []engine.PlayerSpec{
			{Name: "Kamala", HeroBase: "05001", Deck: msmDeck()},
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
	prefer := []string{"pass-interrupt", "skip", "take", "end-turn", "continue"}
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

func drive(t *testing.T, g *engine.Game, maxAnswers int) {
	t.Helper()
	for i := 0; i < maxAnswers; i++ {
		pq := g.Pending()
		if pq == nil {
			return
		}
		if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
			t.Fatalf("answer: %v", err)
		}
		if g.Over {
			return
		}
	}
}

func TestMsMarvelImplemented(t *testing.T) {
	if !engine.Implemented("05001a") {
		t.Fatal("Ms. Marvel should count as implemented")
	}
}

// TestAllWaveOneHeroesImplemented verifies the full hero roster after the
// first expansion wave: five Core heroes + Captain America + Ms. Marvel.
func TestAllWaveOneHeroesImplemented(t *testing.T) {
	for _, base := range []string{"01001", "01010", "01019", "01029", "01040", "03001", "05001"} {
		if !engine.Implemented(base + "a") {
			t.Errorf("hero %s should count as implemented", base)
		}
	}
}

func TestMsMarvelVersusRhinoRuns(t *testing.T) {
	g := newMsmGame(t, "01097", 17)
	drive(t, g, 800)
	if !g.Over {
		t.Fatalf("game did not end, round=%d", g.Round)
	}
	t.Logf("outcome: won=%v reason=%q rounds=%d", g.Won, g.Reason.Text, g.Round)
}

// TestMidGameCloneRoundTrips: undo/replay persist the whole game as JSON;
// a mid-game snapshot (with a pending question carrying the new message
// payloads) must survive a round trip.
func TestMidGameCloneRoundTrips(t *testing.T) {
	g := newMsmGame(t, "01097", 23)
	drive(t, g, 12) // somewhere mid-game with pending prompts
	clone := g.Clone()
	if clone.Round != g.Round || clone.ScenarioID != g.ScenarioID {
		t.Fatalf("clone mismatch: round %d vs %d", clone.Round, g.Round)
	}
	if len(clone.Players) != len(g.Players) || len(clone.Players[0].Hand) != len(g.Players[0].Hand) {
		t.Fatal("clone player state mismatch")
	}
	if (g.Pending() == nil) != (clone.Pending() == nil) {
		t.Fatal("pending question presence mismatch after clone")
	}
}

// walkPrompts answers prompts with defaults until one matches want (or
// the game ends). Returns the matching pending question.
func walkPrompts(t *testing.T, g *engine.Game, want string) *engine.PendingQuestion {
	t.Helper()
	for i := 0; i < 40; i++ {
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

// TestMorphogeneticsReturnsEvent: after Ms. Marvel plays an [[Attack]]
// event in hero form, the Morphogenetics prompt can return it to hand.
func TestMorphogeneticsReturnsEvent(t *testing.T) {
	g := newMsmGame(t, "01097", 5)
	p := g.Players[0]

	// Hero form, unexhausted, Big Hands in hand.
	p.Side = engine.SideHero
	p.Exhausted = false
	bigHands := engine.Card{ID: g.NextCardID(), Code: "05003", Owner: p.ID}
	p.Hand = append(p.Hand, bigHands)

	// Queue the play behind the current turn, then walk the prompts.
	g.Push(engine.PlayCard{Player: p.ID, Card: bigHands, Paid: engine.CostPaid{}})
	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"end-turn"}); err != nil {
		t.Fatalf("answer: %v", err)
	}

	pq = walkPrompts(t, g, "Morphogenetics")
	if err := g.Answer(pq.Player, []string{"return"}); err != nil {
		t.Fatalf("answer: %v", err)
	}
	// The event's own enemy-choice question follows.
	if pq = g.Pending(); pq != nil {
		if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
			t.Fatalf("answer: %v", err)
		}
	} else {
		g.Run()
	}

	found := false
	for _, c := range p.Hand {
		if c.Code == "05003" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Big Hands should be back in hand, hand=%v", p.Hand.Codes())
	}
	if !p.Exhausted {
		t.Fatal("Ms. Marvel should be exhausted after Morphogenetics")
	}
}

// TestEmbiggenBoostsDamage: Embiggen! adds +2 damage to the next Attack
// event.
func TestEmbiggenBoostsDamage(t *testing.T) {
	g := newMsmGame(t, "01097", 5)
	p := g.Players[0]
	var villainID engine.EntityID
	for id := range g.Villains {
		villainID = id
	}
	v := g.Villains[villainID]
	v.Damage = 0

	// Hero form with Embiggen in play and Big Hands in hand.
	p.Side = engine.SideHero
	p.Exhausted = false
	embiggen := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "05010", Owner: p.ID}
	g.Upgrades[embiggen.ID] = embiggen
	p.Upgrades = append(p.Upgrades, embiggen.ID)
	bigHands := engine.Card{ID: g.NextCardID(), Code: "05003", Owner: p.ID}
	p.Hand = append(p.Hand, bigHands)

	g.Push(engine.PlayCard{Player: p.ID, Card: bigHands, Paid: engine.CostPaid{}})
	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"end-turn"}); err != nil {
		t.Fatalf("answer: %v", err)
	}

	pq = walkPrompts(t, g, "Embiggen")
	if err := g.Answer(pq.Player, []string{"boost"}); err != nil {
		t.Fatalf("answer: %v", err)
	}
	if pq = g.Pending(); pq != nil {
		if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
			t.Fatalf("answer: %v", err)
		}
	} else {
		g.Run()
	}

	// The default target is the first listed enemy (the villain): 4 + 2.
	if v.Damage != 6 {
		t.Fatalf("villain should have taken 6 damage (4+2), got %d", v.Damage)
	}
}

// TestEdisonGate: Thomas Edison cannot take damage while the player is
// engaged with another minion.
func TestEdisonGate(t *testing.T) {
	g := newMsmGame(t, "01097", 3)
	p := g.Players[0]
	edison := &engine.Minion{ID: g.NextEntityID("minion"), Code: "05027", MaxHP: 5, EngagedWith: p.ID}
	g.Minions[edison.ID] = edison

	resolve := func() {
		if pq := g.Pending(); pq != nil {
			if _, err := pq.Question.Leaf("form"); err == nil {
				if err := g.Answer(pq.Player, []string{"form"}); err != nil {
					t.Fatalf("answer: %v", err)
				}
			} else if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
				t.Fatalf("answer: %v", err)
			}
		} else {
			g.Run()
		}
	}

	g.Push(engine.DamageEntity{Target: edison.ID, Damage: 3, Source: p.ID})
	resolve()
	if edison.Damage != 3 {
		t.Fatalf("Edison alone should be damageable, damage=%d", edison.Damage)
	}
	// Now engage a second minion and verify the gate blocks.
	other := &engine.Minion{ID: g.NextEntityID("minion"), Code: "02022", MaxHP: 3, EngagedWith: p.ID}
	g.Minions[other.ID] = other
	edison.Damage = 0
	g.Push(engine.DamageEntity{Target: edison.ID, Damage: 3, Source: p.ID})
	resolve()
	if edison.Damage != 0 {
		t.Fatalf("Edison should be undamageable while another minion is engaged, damage=%d", edison.Damage)
	}
}
