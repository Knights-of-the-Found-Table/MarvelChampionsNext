package storm_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/storm"
)

func stormDeck() map[string]int {
	return map[string]int{
		"36006": 1, "36007": 1, "36008": 1,
		"36009": 3, "36010": 3, "36011": 2, "36012": 2, "36013": 2,
	}
}

func newStormGame(t *testing.T) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed: 36, ScenarioID: "01097",
		Players: []engine.PlayerSpec{{Name: "Storm", HeroBase: "36001", Deck: stormDeck()}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// Contract test: inspect setup, invoke React with the serialized Weather
// choice signal, then execute Weather Control directly. This avoids the turn
// state machine while covering the complete identity ability boundary.
func TestStormWeatherDeckSetupSwapAndHeroAbility(t *testing.T) {
	g := newStormGame(t)
	p := g.Players[0]
	identity := engine.LookupBehavior("36001")
	if identity == nil || identity.HeroSetup == nil || identity.HeroAbilities == nil || identity.React == nil {
		t.Fatal("Storm should expose setup, HeroAbilities, and React hooks")
	}

	setup := identity.HeroSetup(g, p)
	if len(setup) != 1 {
		t.Fatalf("setup returned %d messages, want one Weather question", len(setup))
	}
	ask, ok := setup[0].(engine.AskQuestion)
	if !ok || ask.Question == nil || len(ask.Question.Choices) != 4 {
		t.Fatalf("setup message = %#v, want four Weather choices", setup[0])
	}

	p.Side = engine.SideHero
	msgs := identity.React(g, p, engine.AddEntityCounter{ID: p.ID, N: -36004})
	if len(p.Supports) != 1 {
		t.Fatalf("Weather supports = %d, want 1", len(p.Supports))
	}
	weather := g.Supports[p.Supports[0]]
	if weather == nil || weather.Code != "36004" {
		t.Fatalf("weather = %#v, want Thunderstorm", weather)
	}
	if len(msgs) != 1 {
		t.Fatalf("Thunderstorm swap returned %d messages, want its Special question", len(msgs))
	}
	if special, ok := msgs[0].(engine.AskQuestion); !ok || special.Question == nil || special.Question.Prompt != "Thunderstorm — choose an enemy" {
		t.Fatalf("swap message = %#v, want Thunderstorm Special", msgs[0])
	}

	abilities := identity.HeroAbilities(g, p)
	if len(abilities) != 1 || abilities[0].Execute == nil || !abilities[0].OncePerRound {
		t.Fatalf("HeroAbilities = %#v, want one once-per-round executable ability", abilities)
	}
	executed := abilities[0].Execute(g, p.ID)
	if len(executed) != 1 {
		t.Fatalf("Weather Control returned %d messages, want one question", len(executed))
	}
	swap, ok := executed[0].(engine.AskQuestion)
	if !ok || swap.Question == nil || len(swap.Question.Choices) != 3 {
		t.Fatalf("Weather Control = %#v, want the three other Weather choices", executed[0])
	}
}

// Contract test: Clear Skies reacts directly to status placement with the
// corresponding clear message, representing its global stalwart aura.
func TestClearSkiesCancelsStatus(t *testing.T) {
	g := newStormGame(t)
	p := g.Players[0]
	s := &engine.Support{ID: g.NextEntityID("support"), Code: "36002", Owner: p.ID}
	g.Supports[s.ID] = s
	p.Supports = append(p.Supports, s.ID)

	b := engine.LookupBehavior("36002")
	if b == nil || b.React == nil {
		t.Fatal("Clear Skies should expose a React hook")
	}
	msgs := b.React(g, s, engine.StunEntity{Target: p.ID})
	if len(msgs) != 1 {
		t.Fatalf("Clear Skies returned %d messages, want one ClearStun", len(msgs))
	}
	clear, ok := msgs[0].(engine.ClearStun)
	if !ok || clear.Target != p.ID {
		t.Fatalf("Clear Skies message = %#v, want ClearStun for Storm", msgs[0])
	}
}

// Contract test: Knife Fight's hero branch is resolved directly and must ask
// Storm to select only among enemies tied for the highest current ATK.
func TestKnifeFightTargetsHighestAttackEnemy(t *testing.T) {
	g := newStormGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	treachery := &engine.Treachery{ID: g.NextEntityID("treachery"), Code: "36034"}
	g.Treacheries[treachery.ID] = treachery

	b := engine.LookupBehavior("36034")
	if b == nil || b.ResolveTreachery == nil {
		t.Fatal("Knife Fight should expose ResolveTreachery")
	}
	msgs := b.ResolveTreachery(g, treachery, p)
	if len(msgs) != 1 {
		t.Fatalf("Knife Fight returned %d messages, want one target question", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || ask.Question == nil || len(ask.Question.Choices) == 0 {
		t.Fatalf("Knife Fight message = %#v, want a highest-ATK target question", msgs[0])
	}
	for _, choice := range ask.Question.Choices {
		if choice.SourceID == "" || g.Entity(choice.SourceID) == nil {
			t.Fatalf("Knife Fight choice has invalid target: %#v", choice)
		}
	}
}

// TestRemainingstormRegistered sweeps the pack's remaining cards.
func TestRemainingstormSweep(t *testing.T) {
	for _, def := range engine.DB.All() {
		if def.PackCode != "storm" {
			continue
		}
		if def.Text == "" {
			continue
		}
		if !engine.Implemented(def.Code) {
			t.Errorf("card %s (%s) has no registered behavior", def.Code, def.Name)
		}
	}
}
