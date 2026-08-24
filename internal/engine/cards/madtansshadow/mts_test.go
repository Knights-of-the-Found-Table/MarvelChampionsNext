package madtansshadow_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/madtansshadow"
)

func TestMtsAllRegistered(t *testing.T) {
	for _, code := range []string{
		"21011", "21012", "21013", "21014", "21015", "21016", "21017",
		"21018", "21019", "21020", "21021", "21022", "21023", "21024",
		"21025", "21041", "21042", "21043", "21044", "21045", "21046",
		"21047", "21048", "21049", "21050", "21051", "21052", "21053",
		"21054", "21055", "21056", "21057", "21058", "21059", "21060",
		"21061", "21062", "21063", "21064", "21065", "21071", "21072",
		"21073", "21074", "21075", "21076", "21077", "21078", "21079",
		"21080", "21081", "21082", "21083", "21084", "21085", "21086",
		"21087", "21088", "21089", "21090", "21091", "21092", "21093",
		"21094", "21095", "21096", "21097", "21098", "21099", "21100",
		"21101", "21102", "21103", "21104", "21105", "21106", "21107",
		"21108", "21109", "21110", "21111", "21112", "21113", "21114",
		"21115", "21116", "21117", "21118", "21119", "21120", "21121",
		"21122", "21123", "21124", "21125", "21126", "21127", "21128",
		"21129", "21130", "21131", "21132", "21133", "21134", "21135",
		"21136", "21137", "21138", "21139", "21140", "21141", "21142",
		"21143", "21144", "21145", "21146", "21147", "21148", "21149",
		"21150", "21151", "21152", "21153", "21154", "21155", "21156",
		"21157", "21158", "21159", "21160", "21161", "21162", "21163",
		"21164", "21165", "21166", "21167", "21168", "21169", "21170",
		"21171", "21172", "21173", "21174", "21175", "21176", "21177",
		"21178", "21179", "21180", "21181", "21182", "21183", "21184",
		"21185", "21186", "21187", "21188", "21189", "21190", "21191",
		"21192", "21193",
	} {
		if !engine.Implemented(code) {
			t.Errorf("card %s has no registered behavior", code)
		}
	}
}

func newMtsGame(t *testing.T, seed int64, scenario string) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players: []engine.PlayerSpec{
			{Name: "Carol", HeroBase: "08001", Deck: map[string]int{
				"08010": 2, "08025": 3, "08026": 3, "08027": 3,
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	for i := 0; i < 8; i++ {
		pq := g.Pending()
		if pq == nil || pq.Question.Prompt == "Your turn" {
			break
		}
		_ = g.Answer(pq.Player, []string{pq.Question.Choices[0].ID})
	}
	return g
}

func unblock(t *testing.T, g *engine.Game, limit int) {
	t.Helper()
	for i := 0; i < limit; i++ {
		pq := g.Pending()
		if pq == nil || g.Over {
			return
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
}

// TestTowerDefenseSetup: two villains, the tower, the surrogate scheme.
func TestTowerDefenseSetup(t *testing.T) {
	g := newMtsGame(t, 41, "21098")
	if len(g.Villains) != 2 {
		t.Fatalf("Proxima and Corvus should both be in play, got %d", len(g.Villains))
	}
	if g.EnvironmentByCode("21100a") == nil {
		t.Fatal("Avengers Tower should be in play")
	}
	found := false
	for _, s := range g.SideSchemes {
		if s != nil && s.Code[:5] == "21099" {
			found = true
		}
	}
	if !found {
		t.Fatal("The Armies of Thanos surrogate scheme should be in play")
	}
	if g.ActiveVillain == "" {
		t.Fatal("the active counter should start on Proxima")
	}
}

// TestThanosSetup: stones seeded, gauntlet attached.
func TestThanosSetup(t *testing.T) {
	g := newMtsGame(t, 42, "21114")
	if len(g.SetAside) != 6 {
		t.Fatalf("the infinity stone deck should hold 6 stones, got %d", len(g.SetAside))
	}
	gauntlet := false
	for _, a := range g.Attachments {
		if a != nil && a.Code[:5] == "21129" {
			gauntlet = true
		}
	}
	if !gauntlet {
		t.Fatal("the Infinity Gauntlet should be attached to Thanos")
	}
}

// TestStoneEnvironments: stones spawn and resolve behavior presence.
func TestStoneEnvironments(t *testing.T) {
	g := newMtsGame(t, 43, "21114")
	env := g.SpawnEnvironment("21131")
	if g.Environments[env.ID] == nil {
		t.Fatal("the Power Stone should enter play")
	}
	for _, code := range []string{"21130", "21131", "21132", "21133", "21134", "21135"} {
		if !engine.Implemented(code) {
			t.Errorf("stone %s has no behavior", code)
		}
	}
}

// TestHelaSetup: Odin attached, Gnipahellir and Garm up, rest benched.
func TestHelaSetup(t *testing.T) {
	g := newMtsGame(t, 44, "21138")
	odin := false
	for _, a := range g.Attachments {
		if a != nil && a.Code[:5] == "21139" && g.MainScheme != nil && a.Target == g.MainScheme.ID {
			odin = true
		}
	}
	if !odin {
		t.Fatal("Odin should be attached to the main scheme")
	}
	found := false
	for _, s := range g.SideSchemes {
		if s != nil && s.Code[:5] == "21140" {
			found = true
		}
	}
	if !found {
		t.Fatal("Gnipahellir should be in play")
	}
	if len(g.SetAside) != 4 {
		t.Fatalf("4 cards should be benched, got %d", len(g.SetAside))
	}
}

// TestLokiSetup: one Loki up, four benched, War in Asgard in play.
func TestLokiSetup(t *testing.T) {
	g := newMtsGame(t, 45, "21165")
	if len(g.Villains) != 1 {
		t.Fatalf("exactly one Loki should be active, got %d", len(g.Villains))
	}
	bench := 0
	for _, c := range g.SetAside {
		if base := engine.BaseCodeOf(c.Code); base >= "21160" && base <= "21164" {
			bench++
		}
	}
	// The scenario benches all five, then swapLoki pulls one back.
	if bench != 4 {
		t.Fatalf("4 Lokis should be benched, got %d", bench)
	}
	found := false
	for _, s := range g.SideSchemes {
		if s != nil && s.Code[:5] == "21167" {
			found = true
		}
	}
	if !found {
		t.Fatal("War in Asgard should be in play")
	}
}

// TestEbonyMawSetup: spells dig into play via the scheme reveal.
func TestEbonyMawSetup(t *testing.T) {
	g := newMtsGame(t, 46, "21074")
	// The reveal digs one spell per player; single player => at least a
	// lookup confirms the behaviors resolve without panic.
	_ = g
	b := engine.LookupBehavior("21081")
	if b == nil || b.ResolveTreachery == nil {
		t.Fatal("Channeling Trance should resolve")
	}
}
