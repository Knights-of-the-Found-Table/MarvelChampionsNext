package onceandfuturekang_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register core + kang content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/onceandfuturekang"
)

func newKangGame(t *testing.T, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "11007",
		Players: []engine.PlayerSpec{
			{Name: "P1", HeroBase: "01001", Deck: map[string]int{"01088": 9, "01089": 9}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if pq := g.Pending(); pq != nil && pq.Question.Prompt == "Mulligan?" {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("keep: %v", err)
		}
	}
	return g
}

// TestKangStageFlow: stage 1 has Kang I; defeating him brings the four
// variants; defeating all variants brings the final Conqueror whose
// defeat wins.
func TestKangStageFlow(t *testing.T) {
	g := newKangGame(t, 11)
	if len(g.Villains) != 1 {
		t.Fatalf("stage 1 should have one Kang, got %d", len(g.Villains))
	}

	killAll := func() {
		for id := range g.Villains {
			// Two hits: Kangs print Toughness, which eats the first one.
			g.Push(engine.DamageEntity{Target: id, Damage: 999, Source: g.Players[0].ID})
			g.Push(engine.DamageEntity{Target: id, Damage: 999, Source: g.Players[0].ID})
		}
		drainKang(t, g)
	}

	// Kill Kang I → variants arrive (via the scheme advancing).
	killAll()
	if len(g.Villains) != 4 {
		t.Fatalf("stage 2 should bring the four variants, got %d villains", len(g.Villains))
	}

	// Kill the variants → the final Conqueror arrives.
	killAll()
	if len(g.Villains) != 1 {
		t.Fatalf("stage 3 should bring the final Conqueror, got %d villains", len(g.Villains))
	}
	if base := firstVillainBase(g); base != "11006" {
		t.Fatalf("stage 3 villain should be 11006, got %s", base)
	}

	// Kill the Conqueror → victory.
	killAll()
	if !g.Over || !g.Won {
		t.Fatalf("defeating the final Kang should win, over=%v won=%v", g.Over, g.Won)
	}
}

// drainKang answers prompts until the queue settles (reveal chains from
// stage advances pend their own questions).
func drainKang(t *testing.T, g *engine.Game) {
	t.Helper()
	for i := 0; i < 60; i++ {
		pq := g.Pending()
		if pq == nil {
			g.Run()
			if g.Pending() == nil || g.Over {
				return
			}
			pq = g.Pending()
		}
		if g.Over {
			return
		}
		prefer := []string{"pass-interrupt", "take", "threat", "keep", "continue", "skip"}
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

func firstVillainBase(g *engine.Game) string {
	for _, v := range g.Villains {
		return v.Code[:5]
	}
	return ""
}

// TestDominionBlocksDamage: while Kang's Dominion is in play, Kang
// cannot take damage.
func TestDominionBlocksDamage(t *testing.T) {
	g := newKangGame(t, 12)
	s := &engine.SideScheme{ID: g.NextEntityID("side_scheme"), Code: "11023", Threat: 2, MaxThreat: 6}
	g.SideSchemes[s.ID] = s
	var vid engine.EntityID
	for id := range g.Villains {
		vid = id
	}
	dmg := g.Villains[vid].Damage
	g.Push(engine.DamageEntity{Target: vid, Damage: 5, Source: g.Players[0].ID})
	if pq := g.Pending(); pq != nil {
		_ = g.Answer(pq.Player, []string{"form"})
	}
	g.Run()
	if g.Villains[vid] != nil && g.Villains[vid].Damage != dmg {
		t.Fatal("Kang should not take damage while Kang's Dominion is in play")
	}
}
