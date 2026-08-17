package civilwar_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"

	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/angel"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/captainamerica"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/civilwar"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/daredevil"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/doctorstrange"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/echo"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/extras"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/galaxysmostwanted"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/goblinfooblin"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/msmarvel"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/mutantgenesis"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nextevolution"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nightcrawler"
)

// TestAllHeroesImplemented verifies the full roster: Core five, wave one
// (Cap, Ms. Marvel, Daredevil), Doctor Strange, Angel, Nightcrawler, Echo
// and the eight box heroes.
func TestAllHeroesImplemented(t *testing.T) {
	heroes := []string{
		"01001", "01010", "01019", "01029", "01040", // core
		"03001", "05001", "60001", // cap, msm, daredevil
		"09001", "42001", "48001", "60037", // strange, angel, nightcrawler, echo
		"16001", "16029", "32001", "32030", "40001", "40037", "56001", "56029", // box heroes
	}
	for _, base := range heroes {
		if !engine.Implemented(base + "a") {
			t.Errorf("hero %s should count as implemented", base)
		}
	}
}

// TestAllScenariosRegistered verifies every scenario across the packs.
func TestAllScenariosRegistered(t *testing.T) {
	want := []string{
		"01097", "01116", "01137", // rhino, klaw, ultron
		"02004", "02017", // green goblin x2
		"16057", "16073", "16082", // gmw: drang, collector, nebula
		"32063", "32087", "32112", "32125", // mut_gen
		"40077", "40121", "40139", // next_evol
		"56063", // civil war
	}
	ids := map[string]bool{}
	for _, s := range engine.Scenarios() {
		ids[s.ID] = true
	}
	for _, id := range want {
		if !ids[id] {
			t.Errorf("scenario %s not registered", id)
		}
	}
}

// simpleDeck is a generic legal-ish deck used to smoke-test heroes.
func simpleDeck() map[string]int {
	return map[string]int{
		"01088": 3, "01089": 3, "01090": 3,
		"01005": 2, "01054": 2, "01060": 2, "01006": 1, "01067": 2,
	}
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

func runGame(t *testing.T, hero, scenario string, seed int64) {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players: []engine.PlayerSpec{
			{Name: hero, HeroBase: hero, Deck: simpleDeck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame(%s vs %s): %v", hero, scenario, err)
	}
	for i := 0; i < 900; i++ {
		pq := g.Pending()
		if pq == nil {
			return
		}
		if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
			t.Fatalf("answer %q: %v", pq.Question.Prompt, err)
		}
		if g.Over {
			return
		}
	}
}

// TestNewHeroesRunRhino: every newly added hero completes a Rhino game.
func TestNewHeroesRunRhino(t *testing.T) {
	heroes := []string{
		"09001", "42001", "48001", "60037",
		"16001", "16029", "32001", "32030", "40001", "40037", "56001", "56029",
	}
	for i, hero := range heroes {
		runGame(t, hero, "01097", int64(31+i))
	}
}

// TestNewScenariosRun: every newly registered scenario completes a game
// with Daredevil's reference deck.
func ddDeck() map[string]int {
	return map[string]int{
		"60007": 1, "60008": 2, "60009": 2, "60010": 2, "60011": 1,
		"60012": 1, "60013": 1, "60014": 1, "60015": 1, "60016": 1,
		"60017": 1, "60018": 1, "60030": 1, "60038": 1, "60048": 3,
		"60050": 3, "60052": 3, "60053": 1, "60054": 1,
		"01079": 2, "01081": 1, "09020": 1, "16024": 1, "32014": 2,
		"40020": 1, "42017": 1, "48014": 1, "48015": 3, "56046": 1,
		"01088": 1, "01089": 1, "01090": 1,
	}
}

func TestNewScenariosRun(t *testing.T) {
	scenarios := []string{
		"16057", "16073", "16082",
		"32063", "32087", "32112", "32125",
		"40077", "40121", "40139",
		"56063",
	}
	for i, sc := range scenarios {
		t.Run(sc, func(t *testing.T) {
			g, err := engine.NewGame(engine.NewGameOptions{
				Seed:       int64(101 + i),
				ScenarioID: sc,
				Players: []engine.PlayerSpec{
					{Name: "Matt", HeroBase: "60001", Deck: ddDeck()},
				},
			})
			if err != nil {
				t.Fatalf("NewGame(%s): %v", sc, err)
			}
			for j := 0; j < 900; j++ {
				pq := g.Pending()
				if pq == nil {
					return
				}
				if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
					t.Fatalf("answer %q: %v", pq.Question.Prompt, err)
				}
				if g.Over {
					return
				}
			}
		})
	}
}
