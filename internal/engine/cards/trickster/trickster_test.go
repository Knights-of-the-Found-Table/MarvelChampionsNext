package trickster_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/trickster"
)

func newTTGame(t *testing.T, seed int64, scenario string) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players:    []engine.PlayerSpec{{Name: "P1", HeroBase: "01001", Deck: map[string]int{"01088": 9, "01089": 9}}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	for i := 0; i < 12; i++ {
		pq := g.Pending()
		if pq == nil || pq.Question.Prompt == "Your turn" {
			break
		}
		idx := 0
		for j, c := range pq.Question.Choices {
			if c.Then == nil && !c.Disabled {
				idx = j
				break
			}
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[idx].ID})
	}
	return g
}

// TestEnchantressSetup: one gaze attached, Future of Despair set aside.
func TestEnchantressSetup(t *testing.T) {
	g := newTTGame(t, 101, "55004b")
	var v *engine.Villain
	for _, vv := range g.Villains {
		v = vv
	}
	if v == nil || engine.BaseCodeOf(v.Code) != "55001" {
		t.Fatalf("Enchantress I should lead, got %v", v)
	}
	gaze := false
	for _, a := range g.Attachments {
		if a != nil {
			base := engine.BaseCodeOf(a.Code)
			if base >= "55007" && base <= "55011" {
				gaze = true
			}
		}
	}
	if !gaze {
		t.Fatal("each identity should start with a Hypnotic Gaze")
	}
	for _, c := range g.SetAside {
		if c.Code == "55006" {
			return // found
		}
	}
	t.Fatal("Future of Despair should start set aside")
}

// TestWorldsCollideSetup: one avatar in play, synergy environments out.
func TestWorldsCollideSetup(t *testing.T) {
	g := newTTGame(t, 102, "55028b")
	avatars := 0
	var inPlay string
	for _, v := range g.Villains {
		if v != nil && v.EDef().HasTrait("avatar of loki") {
			avatars++
			inPlay = v.Code
		}
	}
	if avatars != 1 {
		t.Fatalf("exactly one avatar expected, got %d", avatars)
	}
	benched := 0
	for _, c := range g.SetAside {
		if c.Def().HasTrait("avatar of loki") {
			benched++
		}
	}
	if benched != 3 {
		t.Fatalf("3 avatars should be benched, got %d (in play: %s)", benched, inPlay)
	}
	envs := 0
	for _, env := range g.Environments {
		if env != nil {
			envs++
		}
	}
	if envs != 4 {
		t.Fatalf("4 synergy environments expected, got %d", envs)
	}
}

// TestTTGapCardsRegistered: spot-check.
func TestTTGapCardsRegistered(t *testing.T) {
	for _, code := range []string{
		"55001", "55002", "55003", "55006", "55007", "55011", "55016",
		"55017", "55020", "55025", "55026", "55027", "55029", "55034",
		"55038", "55045", "55051", "55052", "55055", "55056", "55060",
		"55063", "55066", "55004", "55005", "55028", "55033",
	} {
		if !engine.Implemented(code) {
			t.Errorf("card %s should be implemented", code)
		}
	}
}
