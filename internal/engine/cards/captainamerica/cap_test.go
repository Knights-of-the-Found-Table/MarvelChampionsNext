package captainamerica_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/captainamerica"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

// capDeck mirrors a legal Leadership Captain America decklist.
func capDeck() map[string]int {
	return map[string]int{
		"03002": 1, "03003": 2, "03004": 2, "03005": 1, "03006": 2,
		"03007": 1, "03008": 1, "03009": 1, "03010": 1,
		"03011": 2, "03013": 2, "03015": 2, "03016": 2, "03017": 3,
		"03018": 1, "03020": 1, "03021": 1, "03024": 1, "03025": 2,
	}
}

func newCapGame(t *testing.T, scenario string, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players: []engine.PlayerSpec{
			{Name: "Cap", HeroBase: "03001", Deck: capDeck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame(%s): %v", scenario, err)
	}
	return g
}

func pickDefault(q *engine.Question) []string {
	prefer := []string{"pass-interrupt", "skip", "take", "end-turn"}
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

func TestCaptainAmericaImplemented(t *testing.T) {
	if !engine.Implemented("03001a") {
		t.Fatal("Captain America should count as implemented")
	}
}

// resolveOne drains one pending question so pushed messages can process.
func resolveOne(t *testing.T, g *engine.Game) {
	t.Helper()
	pq := g.Pending()
	if pq == nil {
		g.Run()
		return
	}
	if _, err := pq.Question.Leaf("form"); err == nil {
		if err := g.Answer(pq.Player, []string{"form"}); err != nil {
			t.Fatalf("answer: %v", err)
		}
		return
	}
	if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
		t.Fatalf("answer: %v", err)
	}
}

func TestShieldBlockPreventsAll(t *testing.T) {
	g := newCapGame(t, "01097", 3)
	p := g.Players[0]
	var villainID engine.EntityID
	for id := range g.Villains {
		villainID = id
	}
	// Put the shield in play so the identity has +1 DEF, then resolve a
	// defense marked PreventAll (Shield Block's Defends payload).
	u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "03009", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)

	before := p.Damage
	g.Push(engine.Defends{Defender: p.ID, Against: villainID, Undefended: true, PreventAll: true})
	resolveOne(t, g)
	if p.Damage != before {
		t.Fatalf("Shield Block should prevent all damage, damage %d -> %d", before, p.Damage)
	}
	if !strings.Contains(strings.Join(g.Log, "\n"), "prevents all damage") {
		t.Fatal("defense resolution never ran")
	}
}

func TestHelmetSavesFromDefeat(t *testing.T) {
	g := newCapGame(t, "01097", 3)
	p := g.Players[0]
	helmet := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "03008", Owner: p.ID}
	g.Upgrades[helmet.ID] = helmet
	p.Upgrades = append(p.Upgrades, helmet.ID)

	p.Damage = p.MaxHP - 1 // one hit from defeat
	g.Push(engine.DamageEntity{Target: p.ID, Damage: 5, Source: engine.EntityID("villain-1")})
	resolveOne(t, g)
	if p.KOed {
		t.Fatal("Helmet should have prevented the KO")
	}
	if p.HP() != 1 {
		t.Fatalf("after the Helmet save HP should be 1, got %d", p.HP())
	}
	if len(p.Upgrades) != 0 {
		t.Fatal("Helmet should be discarded after the save")
	}
}

func TestSetupFetchesShield(t *testing.T) {
	g := newCapGame(t, "01097", 7)
	p := g.Players[0]
	found := false
	for _, c := range p.Hand {
		if c.Code == "03009" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Captain America's Shield should be in the opening hand, hand=%v", p.Hand.Codes())
	}
}

func TestCapVersusRhinoRuns(t *testing.T) {
	g := newCapGame(t, "01097", 13)
	drive(t, g, 800)
	if !g.Over {
		t.Fatalf("game did not end, round=%d", g.Round)
	}
	t.Logf("outcome: won=%v reason=%q rounds=%d", g.Won, g.Reason, g.Round)
}

func TestZemoBlocksThwart(t *testing.T) {
	g := newCapGame(t, "01097", 3)
	p := g.Players[0]
	if g.MainScheme == nil {
		t.Fatal("expected a main scheme")
	}
	// Spawn Zemo engaged with the player.
	mn := &engine.Minion{ID: g.NextEntityID("minion"), Code: "03028", MaxHP: 4, EngagedWith: p.ID}
	g.Minions[mn.ID] = mn

	before := g.MainScheme.Threat
	p.Exhausted = false
	g.Push(engine.BasicThwart{Player: p.ID, N: 2, Target: g.MainScheme.ID})
	// A pending question blocks the queue; resolve it with the harmless
	// form-change choice so the pushed message can process.
	if pq := g.Pending(); pq != nil {
		path := []string{"form"}
		if _, err := pq.Question.Leaf("form"); err != nil {
			t.Fatalf("no form choice: %v", err)
		}
		if err := g.Answer(pq.Player, path); err != nil {
			t.Fatalf("answer: %v", err)
		}
	} else {
		g.Run()
	}

	if g.MainScheme.Threat != before {
		t.Fatalf("thwart should be blocked while Zemo is engaged, threat %d -> %d", before, g.MainScheme.Threat)
	}
	if p.Exhausted {
		t.Fatal("a blocked thwart must not exhaust the identity")
	}
	if !strings.Contains(strings.Join(g.Log, "\n"), "cannot thwart") {
		t.Fatal("expected the blocked-thwart log line")
	}
}
