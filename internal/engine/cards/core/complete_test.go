package core_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register core set content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

// newCoreGame builds a game paused on the mulligan; setup runs before the
// mulligan is kept so the first turn menu is built with the test state
// already in place (avoiding stale-menu matching).
func newCoreGame(t *testing.T, seed int64, deck map[string]int, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "P1", HeroBase: "01001", Deck: deck},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if setup != nil {
		setup(g)
	}
	if pq := g.Pending(); pq != nil && pq.Question.Prompt == "Mulligan?" {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("keep: %v", err)
		}
	}
	return g
}

func drive(t *testing.T, g *engine.Game, max int) {
	t.Helper()
	for i := 0; i < max; i++ {
		pq := g.Pending()
		if pq == nil {
			g.Run()
			if g.Pending() == nil {
				return
			}
			pq = g.Pending()
		}
		if g.Over {
			return
		}
		prefer := []string{"pass-interrupt", "skip", "continue", "keep", "take", "end-turn"}
		var ans []string
		for _, id := range prefer {
			for _, c := range pq.Question.Choices {
				if c.ID == id && !c.Disabled {
					ans = []string{id}
					break
				}
			}
			if ans != nil {
				break
			}
		}
		if ans == nil && pq.Question.Type == "choose_n" {
			for i, c := range pq.Question.Choices {
				_ = i
				_ = c
				break
			}
			ans = []string{pq.Question.Choices[0].ID}
		}
		if ans == nil && len(pq.Question.Choices) > 0 {
			ans = []string{pq.Question.Choices[0].ID}
		}
		if ans == nil {
			return
		}
		if err := g.Answer(pq.Player, ans); err != nil {
			return
		}
	}
}

// TestHaymakerDealsThree: the simplest event template works end to end.
func TestHaymakerDealsThree(t *testing.T) {
	g := newCoreGame(t, 31, map[string]int{
		"01087": 2, // Haymaker
		"01088": 6, "01089": 6,
	}, nil)
	p := g.Players[0]
	hay := engine.Card{ID: g.NextCardID(), Code: "01087", Owner: p.ID}
	p.Hand = append(p.Hand, hay)

	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	hp := g.Villains[villain].HP()

	g.Push(engine.PlayCard{Player: p.ID, Card: hay, Paid: engine.CostPaid{CardIDs: []string{"x"}, Icons: []string{"energy", "energy", "energy"}}})
	drive(t, g, 10)

	if hp2 := g.Villains[villain].HP(); hp2 != hp-3 {
		t.Fatalf("Haymaker should deal 3 damage, HP %d -> %d", hp, hp2)
	}
}

// TestGoldenCityDrawsTwo: exhaust-to-draw support ability works.
func TestGoldenCityDrawsTwo(t *testing.T) {
	var city *engine.Support
	g := newCoreGame(t, 32, map[string]int{
		"01045": 1, // The Golden City
		"01088": 6, "01089": 6,
	}, func(g *engine.Game) {
		p := g.Players[0]
		city = &engine.Support{ID: g.NextEntityID("support"), Code: "01045", Owner: p.ID}
		g.Supports[city.ID] = city
		p.Supports = append(p.Supports, city.ID)
		for len(p.Deck) > 2 {
			p.Deck = p.Deck[:2]
		}
	})
	p := g.Players[0]
	hand := len(p.Hand)

	pq := g.Pending()
	found := false
	for _, c := range pq.Question.Choices {
		if c.CardCode == "01045" {
			found = true
		}
	}
	if !found {
		t.Fatal("Golden City action should be offered in alter-ego form")
	}
	// answer it via label drill: choose by matching label
	var path []string
	for _, c := range pq.Question.Choices {
		if strings.HasPrefix(c.Label.Text, "Exhaust The Golden City") {
			path = []string{c.ID}
		}
	}
	if path == nil {
		t.Fatal("Golden City ability missing from menu")
	}
	if err := g.Answer(pq.Player, path); err != nil {
		t.Fatalf("use Golden City: %v", err)
	}
	if len(p.Hand) != hand+2 {
		t.Fatalf("Golden City should draw 2, hand %d -> %d", hand, len(p.Hand))
	}
	if !city.Exhausted {
		t.Fatal("Golden City should be exhausted after use")
	}
}

// TestFocusedRageDraws: pay-damage-to-draw upgrade ability works.
func TestFocusedRageDraws(t *testing.T) {
	var rage *engine.Upgrade
	g := newCoreGame(t, 33, map[string]int{
		"01027": 1, // Focused Rage
		"01088": 6, "01089": 6,
	}, func(g *engine.Game) {
		p := g.Players[0]
		p.Side = engine.SideHero
		p.Exhausted = false
		rage = &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "01027", Owner: p.ID}
		g.Upgrades[rage.ID] = rage
		p.Upgrades = append(p.Upgrades, rage.ID)
		for len(p.Deck) > 1 {
			p.Deck = p.Deck[:1]
		}
	})
	_ = g
	p := g.Players[0]
	hand := len(p.Hand)
	dmg := p.Damage

	pq := g.Pending()
	var path []string
	for _, c := range pq.Question.Choices {
		if strings.Contains(c.Label.Text, "Focused Rage") {
			path = []string{c.ID}
		}
	}
	if path == nil {
		t.Fatal("Focused Rage action should be offered in hero form")
	}
	if err := g.Answer(pq.Player, path); err != nil {
		t.Fatalf("use Focused Rage: %v", err)
	}
	if len(p.Hand) != hand+1 {
		t.Fatalf("Focused Rage should draw 1, hand %d -> %d", hand, len(p.Hand))
	}
	if p.Damage != dmg+1 {
		t.Fatalf("Focused Rage should deal 1 damage to the hero, %d -> %d", dmg, p.Damage)
	}
}

// TestHeroicIntuitionBoostsThwart: +1 THW stat upgrade shows in the menu.
func TestHeroicIntuitionBoostsThwart(t *testing.T) {
	g := newCoreGame(t, 34, map[string]int{
		"01065": 1, // Heroic Intuition
		"01088": 6, "01089": 6,
	}, nil)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Exhausted = false
	hi := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "01065", Owner: p.ID}
	g.Upgrades[hi.ID] = hi
	p.Upgrades = append(p.Upgrades, hi.ID)

	if thw := p.ThwartStat(g); thw != 2 { // Spider-Man 1 + 1
		t.Fatalf("Heroic Intuition should give +1 THW (total 2), got %d", thw)
	}
}

// TestGroundStompHitsEveryEnemy.
func TestGroundStompHitsEveryEnemy(t *testing.T) {
	g := newCoreGame(t, 35, map[string]int{
		"01022": 1, // Ground Stomp
		"01088": 6, "01089": 6,
	}, nil)
	p := g.Players[0]
	stomp := engine.Card{ID: g.NextCardID(), Code: "01022", Owner: p.ID}
	p.Hand = append(p.Hand, stomp)

	var villain engine.EntityID
	for id := range g.Villains {
		villain = id
	}
	mn := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01101", MaxHP: 3, EngagedWith: p.ID}
	g.Minions[mn.ID] = mn

	vHP := g.Villains[villain].HP()
	mHP := mn.HP()

	g.Push(engine.PlayCard{Player: p.ID, Card: stomp, Paid: engine.CostPaid{}})
	drive(t, g, 10)

	if g.Villains[villain].HP() != vHP-1 || mn.HP() != mHP-1 {
		t.Fatalf("Ground Stomp should hit every enemy: villain %d->%d, minion %d->%d",
			vHP, g.Villains[villain].HP(), mHP, mn.HP())
	}
}
