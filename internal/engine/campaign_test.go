package engine_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// campaign card behaviors (Setup upgrades, obligation hazards)
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/campaign"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

// deckWith builds the Spider-Man test deck plus extra campaign codes.
func deckWith(codes map[string]int) map[string]int {
	deck := spiderDeck()
	for code, n := range codes {
		deck[code] += n
	}
	return deck
}

func findUpgrade(g *engine.Game, code string) *engine.Upgrade {
	for _, p := range g.Players {
		for _, id := range p.Upgrades {
			if u := g.Upgrades[id]; u != nil && u.Code == code {
				return u
			}
		}
	}
	return nil
}

func deckCount(g *engine.Game, code string) int {
	n := 0
	for _, p := range g.Players {
		for _, c := range p.Deck {
			if c.Code == code {
				n++
			}
		}
	}
	return n
}

func handCount(g *engine.Game, code string) int {
	n := 0
	for _, p := range g.Players {
		for _, c := range p.Hand {
			if c.Code == code {
				n++
			}
		}
	}
	return n
}

func encounterDeckCount(g *engine.Game, code string) int {
	n := 0
	for _, c := range g.EncounterDeck {
		if c.Code == code {
			n++
		}
	}
	return n
}

// A campaign side scheme reveals at setup, and the campaign log's extra
// threat lands on top of the reveal's own placement (control comparison:
// the engine's Hinder placement is not under test here).
func TestCampaignStartSideScheme(t *testing.T) {
	build := func(threat map[string]int) *engine.SideScheme {
		t.Helper()
		g, err := engine.NewGame(engine.NewGameOptions{
			Seed:       11,
			ScenarioID: "01097",
			Players:    []engine.PlayerSpec{{Name: "Tester", HeroBase: "01001", Deck: spiderDeck()}},
			Campaign: &engine.CampaignSetup{
				StartSideScheme:  "16178a",
				SideSchemeThreat: threat,
			},
		})
		if err != nil {
			t.Fatalf("NewGame: %v", err)
		}
		keepHands(t, g)
		for _, s := range g.SideSchemes {
			if s != nil && s.Code == "16178a" {
				return s
			}
		}
		t.Fatalf("campaign side scheme not in play")
		return nil
	}
	base := build(nil)
	extra := build(map[string]int{"16178": 4})
	if extra.Threat != base.Threat+4 {
		t.Fatalf("blitz threat %d, want reveal %d + 4", extra.Threat, base.Threat)
	}
}

// Expert obligations shuffle into the player deck and enter play when
// drawn instead of joining the hand. A deck of only the obligation makes
// the opening draw exercise the branch deterministically.
func TestCampaignDeckEncounters(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       12,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{{
			Name: "Tester", HeroBase: "01001",
			DeckEncounters: []string{"04165"},
		}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	// The empty hand skips the mulligan prompt; answer whatever remains.
	for {
		pq := g.Pending()
		if pq == nil || pq.Question.Prompt != "Mulligan?" {
			break
		}
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("answer mulligan: %v", err)
		}
	}
	if findUpgrade(g, "04165") == nil {
		t.Fatalf("Martial Law did not enter play when drawn at setup")
	}
	if handCount(g, "04165") != 0 {
		t.Fatalf("obligation leaked into the hand")
	}
	if deckCount(g, "04165") != 0 {
		t.Fatalf("obligation stayed in the deck")
	}
}

// Setup-keyword campaign upgrades begin the game in play and apply their
// stat bonus (+2 max HP for the Basic Thwart Upgrade).
func TestCampaignSetupKeyword(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       13,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{{
			Name: "Tester", HeroBase: "01001", Deck: deckWith(map[string]int{"04159a": 1}),
		}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	keepHands(t, g)
	if findUpgrade(g, "04159a") == nil {
		t.Fatalf("Basic Thwart Upgrade not in play at start")
	}
	if deckCount(g, "04159a") != 0 {
		t.Fatalf("setup card stayed in the deck")
	}
	if got := g.Players[0].MaxHP; got != 10+2 {
		t.Fatalf("MaxHP = %d, want 12", got)
	}
}

// Expert persistent damage starts the identity at the recorded hit points.
func TestCampaignStartHP(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       14,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{{
			Name: "Tester", HeroBase: "01001", Deck: spiderDeck(), StartHP: 7,
		}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	keepHands(t, g)
	if got := g.Players[0].MaxHP; got != 7 {
		t.Fatalf("MaxHP = %d, want 7", got)
	}
}

// The campaign setup messages queued behind the mulligan prompt survive a
// save/reload round trip (rooms rehydrate from the marshaled JSON).
func TestCampaignSetupSurvivesMarshal(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       21,
		ScenarioID: "01097",
		Players:    []engine.PlayerSpec{{Name: "T", HeroBase: "01001", Deck: spiderDeck()}},
		Campaign: &engine.CampaignSetup{
			StartSideScheme:  "16178a",
			PreShuffle:       []string{"16183"},
			MillMinionEngage: true,
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	blob, err := g.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	g2 := &engine.Game{}
	if err := g2.UnmarshalJSON(blob); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	keepHands(t, g2)
	found := false
	for _, s := range g2.SideSchemes {
		if s != nil && s.Code == "16178a" {
			found = true
		}
	}
	if !found {
		t.Fatalf("campaign side scheme lost after marshal round trip")
	}
}
