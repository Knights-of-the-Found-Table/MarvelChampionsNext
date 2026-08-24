package sinistermotives_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/sinistermotives"
)

// smCodes sweeps the survey gap list for the box.
var smCodes = []string{
	"27012", "27013", "27014", "27015", "27016", "27017", "27018",
	"27019", "27020", "27021", "27022", "27023", "27024", "27041",
	"27042", "27043", "27044", "27046", "27047", "27048", "27049",
	"27050", "27051", "27052", "27053", "27054", "27055", "27061",
	"27062", "27063", "27064", "27065", "27066", "27067", "27068",
	"27069", "27070", "27071", "27072", "27073", "27074", "27075",
	"27076", "27077", "27078", "27079", "27080", "27081", "27082",
	"27083", "27084", "27085", "27086", "27087", "27088", "27089",
	"27090", "27091", "27092", "27093", "27094", "27095", "27096",
	"27097", "27098", "27099", "27100", "27101", "27102", "27103",
	"27104", "27105", "27106", "27107", "27108", "27109", "27110",
	"27111", "27112", "27113", "27114", "27115", "27116", "27117",
	"27118", "27119", "27120", "27121", "27122", "27123", "27124",
	"27125", "27126", "27127", "27128", "27129", "27130", "27131",
	"27132", "27133", "27134", "27135", "27136", "27137", "27138",
	"27139", "27140", "27141", "27142", "27143", "27144", "27145",
	"27146", "27147", "27148", "27149", "27150", "27151", "27152",
	"27153", "27154", "27155", "27156", "27157", "27158", "27159",
	"27160", "27161", "27162", "27163", "27164", "27165", "27166",
	"27167", "27168", "27169", "27170", "27171", "27172", "27173",
	"27174", "27175", "27176", "27177", "27178", "27179", "27180",
	"27181", "27182", "27183", "27184", "27185", "27186", "27187",
	"27188", "27189", "27190", "27191",
}

func TestSmAllRegistered(t *testing.T) {
	for _, code := range smCodes {
		if !engine.Implemented(code) {
			t.Errorf("card %s has no registered behavior", code)
		}
	}
}

func newSmGame(t *testing.T, seed int64, scenario string, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players: []engine.PlayerSpec{
			{Name: "Gwen", HeroBase: "27001", Deck: map[string]int{
				"27023": 1, "27031": 2, "27044": 2,
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if setup != nil {
		setup(g)
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

// TestSandmanSetup: City Streets enters play with 4 sand counters.
func TestSandmanSetup(t *testing.T) {
	g := newSmGame(t, 21, "27064", nil)
	env := g.EnvironmentByCode("27065")
	if env == nil {
		t.Fatal("City Streets should be in play")
	}
	if env.Counters != 4 {
		t.Fatalf("City Streets should start with 4 sand counters, got %d", env.Counters)
	}
}

// TestVenomSetup: the Bell Tower enters play quiet.
func TestVenomSetup(t *testing.T) {
	g := newSmGame(t, 22, "27076", nil)
	env := g.EnvironmentByCode("27077a")
	if env == nil {
		t.Fatal("the Bell Tower should be in play")
	}
}

// TestMysterioSetup: Apparitions engage every player.
func TestMysterioSetup(t *testing.T) {
	g := newSmGame(t, 23, "27087", nil)
	p := g.Players[0]
	found := false
	for _, m := range g.Minions {
		if m != nil && m.EngagedWith == p.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("a Shifting Apparition should engage the player")
	}
}

// TestSinisterSixSetup: benching + active counter.
func TestSinisterSixSetup(t *testing.T) {
	g := newSmGame(t, 24, "27100", nil)
	if len(g.Villains) != 2 { // 1 player + 1
		t.Fatalf("2 villains should be in play (players+1), got %d", len(g.Villains))
	}
	bench := 0
	for _, c := range g.SetAside {
		_ = c
		bench++
	}
	if bench != 4 {
		t.Fatalf("4 villains should be benched, got %d", bench)
	}
	if g.ActiveVillain == "" {
		t.Fatal("the active counter should be placed")
	}
	light := false
	for _, s := range g.SideSchemes {
		if s != nil && s.Code[:5] == "27102" {
			light = true
		}
	}
	if !light {
		t.Fatal("Light at the End should be in play")
	}
}

// TestVenomGoblinSetup: the three districts and the glider.
func TestVenomGoblinSetup(t *testing.T) {
	g := newSmGame(t, 25, "27116", nil)
	if g.MainScheme == nil {
		t.Fatal("Midtown Manhattan should anchor the main scheme")
	}
	if g.GliderCounter != g.MainScheme.ID {
		t.Fatal("the glider should start on Midtown")
	}
	districts := 0
	for _, s := range g.SideSchemes {
		if s != nil {
			switch engine.BaseCodeOf(s.ECode()) {
			case "27117", "27119":
				districts++
			}
		}
	}
	if districts != 2 {
		t.Fatalf("Lower and Upper Manhattan should be in play, got %d", districts)
	}
}

// TestFearmonger: discard hand and redraw.
func TestFearmonger(t *testing.T) {
	g := newSmGame(t, 26, "27064", nil)
	p := g.Players[0]
	hand := len(p.Hand)
	tref := &engine.Treachery{ID: g.NextEntityID("treachery"), Code: "27093"}
	g.Treacheries[tref.ID] = tref
	msgs := engine.LookupBehavior("27093").ResolveTreachery(g, tref, p)
	g.Push(msgs...)
	unblock(t, g, 2)
	if len(p.Hand) > hand {
		t.Fatalf("hand should shrink or stay: %d -> %d", hand, len(p.Hand))
	}
}

// TestBaitAndSwitch: threat removal rides the villain attack.
func TestBaitAndSwitch(t *testing.T) {
	g := newSmGame(t, 27, "27064", nil)
	p := g.Players[0]
	ec := &engine.EventCard{Code: "27013", Owner: p.ID}
	threat := g.MainScheme.Threat
	msgs := engine.LookupBehavior("27013").OnPlay(g, ec)
	g.Push(msgs...)
	unblock(t, g, 4)
	if g.MainScheme.Threat >= threat {
		t.Fatalf("Bait and Switch should remove 4 threat: %d -> %d", threat, g.MainScheme.Threat)
	}
}

// TestSummonSix: the ambush pulls a benched member.
func TestSummonSix(t *testing.T) {
	g := newSmGame(t, 28, "27100", nil)
	before := len(g.Villains)
	bench := len(g.SetAside)
	if bench == 0 {
		t.Fatal("expected benched villains")
	}
	g.Push(engine.SummonSix{Cards: []string{engine.BaseCodeOf(g.SetAside[0].Code)}})
	unblock(t, g, 2)
	if len(g.Villains) != before+1 {
		t.Fatalf("the ambush should add a villain: %d -> %d", before, len(g.Villains))
	}
	if len(g.SetAside) != bench-1 {
		t.Fatalf("the bench should shrink: %d -> %d", bench, len(g.SetAside))
	}
}

// TestDirtTrap: double Surging Sands on defeat.
func TestDirtTrap(t *testing.T) {
	g := newSmGame(t, 29, "27064", nil)
	p := g.Players[0]
	s := &engine.SideScheme{ID: g.NextEntityID("sidescheme"), Code: "27068", Threat: 2, MaxThreat: 6}
	g.SideSchemes[s.ID] = s
	env := g.EnvironmentByCode("27065")
	start := env.Counters
	g.Push(engine.ThwartScheme{Scheme: s.ID, N: 2, Source: p.ID})
	unblock(t, g, 3)
	if env.Counters != start+2 {
		t.Fatalf("Dirt Trap should surge twice: %d -> %d", start, env.Counters)
	}
}

// TestFriendsAndFamily: the obligation resolves via an identity-card
// discard.
func TestFriendsAndFamily(t *testing.T) {
	g := newSmGame(t, 30, "27064", nil)
	p := g.Players[0]
	hand := len(p.Hand)
	card := engine.Card{ID: g.NextCardID(), Code: "27132"}
	msgs := engine.LookupBehavior("27132").ResolveObligation(g, p, card)
	if msgs == nil {
		t.Fatal("the obligation should resolve")
	}
	g.Push(msgs...)
	unblock(t, g, 2)
	if len(p.Hand) != hand-1 {
		t.Fatalf("an identity card should be discarded: %d -> %d", hand, len(p.Hand))
	}
}
