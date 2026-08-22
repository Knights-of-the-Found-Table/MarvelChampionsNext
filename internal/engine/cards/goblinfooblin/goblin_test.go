package goblinfooblin_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/goblinfooblin"
)

func spideyDeck() map[string]int {
	return map[string]int{
		"01002": 1, "01003": 2, "01004": 2, "01005": 2, "01006": 1,
		"01007": 2, "01008": 2,
		"01088": 3, "01089": 3, "01090": 3, "01054": 2, "01055": 1,
	}
}

func newGame(t *testing.T, scenario string, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players: []engine.PlayerSpec{
			{Name: "Tester", HeroBase: "01001", Deck: spideyDeck()},
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
			if i >= q.N {
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

func TestRiskyBusinessRuns(t *testing.T) {
	g := newGame(t, "02004", 11)
	drive(t, g, 600)
	if !g.Over {
		t.Fatalf("game did not end, round=%d", g.Round)
	}
	logText := g.LogText()
	if !strings.Contains(logText, "Criminal Enterprise") {
		t.Fatal("Criminal Enterprise environment never entered play")
	}
	t.Logf("outcome: won=%v reason=%q rounds=%d", g.Won, g.Reason, g.Round)
}

func TestRiskyBusinessStartsAsNorman(t *testing.T) {
	g := newGame(t, "02004", 5)
	var villainCode string
	for _, v := range g.Villains {
		villainCode = v.Code
	}
	if !strings.HasSuffix(villainCode, "a") {
		t.Fatalf("villain should start on the Norman (a) side, got %s", villainCode)
	}
	if env := g.EnvironmentByCode("02006a"); env == nil {
		t.Fatal("Criminal Enterprise should be in play at setup")
	} else if env.Counters != 2 {
		t.Fatalf("infamy counters = %d, want 2 (solo)", env.Counters)
	}
}

func TestNormanDamageConvertsToInfamy(t *testing.T) {
	g := newGame(t, "02004", 5)
	var villainID engine.EntityID
	for id := range g.Villains {
		villainID = id
	}
	env := g.EnvironmentByCode("02006a")
	before := env.Counters

	pushAndResolve := func(dmg int) {
		g.Push(engine.DamageEntity{Target: villainID, Damage: dmg, Source: engine.EntityID("player-1")})
		// A question may be pending; resolve it so the queue drains.
		if pq := g.Pending(); pq != nil {
			if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
				t.Fatalf("answer: %v", err)
			}
		} else {
			g.Run()
		}
	}

	pushAndResolve(1)
	if v := g.Villains[villainID]; v.Damage != 0 {
		t.Fatalf("Norman took real damage (%d), want 0", v.Damage)
	}
	if !strings.Contains(g.LogText(), "Damage converted") {
		t.Fatal("damage should have been converted to infamy removal")
	}
	// Drain the counters -> flip to Green Goblin.
	for i := 0; i < before+4; i++ {
		if v, ok := g.Villains[villainID]; ok && !strings.HasSuffix(v.Code, "a") {
			break
		}
		pushAndResolve(1)
	}
	if v := g.Villains[villainID]; strings.HasSuffix(v.Code, "a") {
		t.Fatalf("villain should have flipped to the Goblin side, still %s", v.Code)
	}
	if env := g.EnvironmentByCode("02006b"); env == nil {
		t.Fatal("State of Madness should be active after the flip")
	}
}

func TestMutagenFormulaRuns(t *testing.T) {
	g := newGame(t, "02017", 21)
	drive(t, g, 600)
	if !g.Over {
		t.Fatalf("game did not end, round=%d", g.Round)
	}
	t.Logf("outcome: won=%v reason=%q rounds=%d", g.Won, g.Reason, g.Round)
}

func TestAllFiveScenariosRegistered(t *testing.T) {
	ids := map[string]bool{}
	for _, s := range engine.Scenarios() {
		ids[s.ID] = true
	}
	for _, want := range []string{"01097", "01116", "01137", "02004", "02017"} {
		if !ids[want] {
			t.Errorf("scenario %s not registered", want)
		}
	}
}
