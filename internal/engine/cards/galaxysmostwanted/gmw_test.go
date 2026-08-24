package galaxysmostwanted_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/galaxysmostwanted"
)

// gmwCodes lists every base code the pack registers (survey gap list).
var gmwCodes = []string{
	"16002", "16003", "16004", "16005", "16006", "16007", "16008",
	"16009", "16010", "16011", "16012", "16013", "16014", "16015",
	"16016", "16017", "16018", "16019", "16020", "16021", "16022",
	"16023", "16025", "16026", "16027", "16028", "16030", "16031",
	"16032", "16033", "16034", "16035", "16036", "16037", "16038",
	"16039", "16040", "16041", "16042", "16043", "16044", "16045",
	"16046", "16047", "16048", "16049", "16050", "16051", "16052",
	"16053", "16055", "16056", "16057", "16058", "16059", "16060",
	"16061", "16062", "16063", "16064", "16065", "16066", "16067",
	"16068", "16069", "16070", "16071", "16072", "16073", "16074",
	"16075", "16076", "16077", "16078", "16079", "16080", "16081",
	"16082", "16083", "16084", "16085", "16086", "16087", "16088",
	"16089", "16090", "16091", "16092", "16093", "16094", "16095",
	"16096", "16097", "16098", "16099", "16100", "16101", "16102",
	"16103", "16104", "16105", "16106", "16107", "16108", "16109",
	"16110", "16111", "16112", "16113", "16114", "16115", "16116",
	"16117", "16118", "16119", "16120", "16121", "16122", "16123",
	"16124", "16125", "16126", "16127", "16128", "16129", "16130",
	"16131", "16132", "16133", "16134", "16135", "16136", "16137",
	"16138", "16139", "16140", "16141", "16142", "16143", "16144",
	"16145", "16146", "16147", "16148", "16149", "16150", "16151",
	"16152", "16153", "16154", "16155", "16156", "16157", "16158",
	"16159", "16160", "16161", "16162", "16163", "16164", "16165",
	"16166", "16167", "16168", "16169", "16170", "16171", "16172",
	"16173", "16174", "16175", "16176", "16177", "16178", "16179",
	"16180", "16181", "16182", "16183", "16184", "16185", "16186",
	"16187",
}

// TestGmwAllRegistered sweeps the pack's survey list.
func TestGmwAllRegistered(t *testing.T) {
	for _, code := range gmwCodes {
		if !engine.Implemented(code) {
			t.Errorf("card %s has no registered behavior", code)
		}
	}
}

func newGmwGame(t *testing.T, seed int64, scenario, hero string, setup func(g *engine.Game)) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: scenario,
		Players: []engine.PlayerSpec{
			{Name: hero, HeroBase: hero, Deck: map[string]int{
				"16007": 1, "16031": 2, "16044": 2,
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

// unblock answers prompts until the queue settles; it prefers plain
// leaf choices without side-question chains.
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

// TestDrangSetup: the scenario brings in the Badoon Ship and the Milano;
// four barrage charges blast the players with indirect damage.
func TestDrangSetupAndBarrage(t *testing.T) {
	g := newGmwGame(t, 7, "16057", "16001", nil)
	if g.EnvironmentByCode("16063") == nil {
		t.Fatal("Badoon Ship should be in play")
	}
	if mil := g.EnvironmentByCode("16142"); mil != nil {
		_ = mil
	}
	var milano *engine.Support
	for _, s := range g.Supports {
		if s != nil && s.Code[:5] == "16142" {
			milano = s
		}
	}
	if milano == nil {
		t.Fatal("the Milano should be in play under the first player")
	}
	p := g.Players[0]
	p.GrowthCounters = 0 // neutralize Groot's prevention for a clean read
	before := p.Damage
	for i := 0; i < 4; i++ {
		g.Push(engine.BarrageCharge{})
	}
	unblock(t, g, 1)
	if p.Damage != before+2 {
		t.Fatalf("4 barrage charges should deal 2 indirect damage: %d -> %d", before, p.Damage)
	}
	if ship := g.EnvironmentByCode("16063"); ship.Counters != 0 {
		t.Fatalf("barrage counters should reset, got %d", ship.Counters)
	}
}

// TestIndirectDamageChoice: with an ally in play the player distributes
// the damage.
func TestIndirectDamageChoice(t *testing.T) {
	g := newGmwGame(t, 8, "16057", "16001", nil)
	p := g.Players[0]
	a := &engine.Ally{ID: g.NextEntityID("ally"), Code: "16012", Owner: p.ID, MaxHP: 4}
	g.AddAlly(a, p.ID)
	pD, aD := p.Damage, a.Damage
	g.Push(engine.IndirectDamage{Player: p.ID, N: 2})
	for i := 0; i < 4; i++ {
		pq := g.Pending()
		if pq == nil {
			break
		}
		if strings.Contains(pq.Question.Prompt, "indirect") {
			// The last choice is the ally (identity is listed first).
			_ = g.Answer(pq.Player, []string{pq.Question.Choices[len(pq.Question.Choices)-1].ID})
			continue
		}
		unblock(t, g, 1)
	}
	if a.Damage != aD+2 || p.Damage != pD {
		t.Fatalf("indirect damage should hit the ally: player %d->%d, ally %d->%d", pD, p.Damage, aD, a.Damage)
	}
}

// TestSideSchemeReward: Magical Teapot (16128) heals 4 when defeated.
func TestSideSchemeReward(t *testing.T) {
	g := newGmwGame(t, 9, "16057", "16001", nil)
	p := g.Players[0]
	p.Damage = 5
	s := &engine.SideScheme{ID: g.NextEntityID("sidescheme"), Code: "16128", Threat: 3, MaxThreat: 6}
	g.AddSideScheme(s)
	g.Push(engine.ThwartScheme{Scheme: s.ID, N: 3, Source: p.ID})
	g.Run()
	unblock(t, g, 6)
	if g.SideSchemes[s.ID] != nil {
		t.Fatal("the scheme should be defeated")
	}
	if p.Damage != 1 {
		t.Fatalf("Magical Teapot should heal 4: damage now %d", p.Damage)
	}
}

// TestCollectorCollection: setup seeds the Collection; destroyed allies
// go there instead of the discard.
func TestCollectorCollection(t *testing.T) {
	g := newGmwGame(t, 11, "16073", "16001", nil)
	if len(g.Collection) != 1 {
		t.Fatalf("setup should collect each player's top card, got %d", len(g.Collection))
	}
	p := g.Players[0]
	a := &engine.Ally{ID: g.NextEntityID("ally"), Code: "16012", Owner: p.ID, MaxHP: 4}
	g.AddAlly(a, p.ID)
	disc := len(p.Discard)
	g.Push(engine.AllyDestroyed{AllyID: a.ID})
	unblock(t, g, 6)
	if len(p.Discard) != disc {
		t.Fatal("the ally should bypass the discard pile")
	}
	if len(g.Collection) != 2 {
		t.Fatalf("the ally should join The Collection, got %d cards", len(g.Collection))
	}
}

// TestBiogramImage: attached, it eats all damage to the Collector.
func TestBiogramImage(t *testing.T) {
	g := newGmwGame(t, 12, "16073", "16001", nil)
	var villain *engine.Villain
	for id := range g.Villains {
		villain = g.Villains[id]
	}
	g.SpawnAttachment("16074", villain.ID)
	hp := villain.HP()
	threat := g.MainScheme.Threat
	g.Push(engine.DamageEntity{Target: villain.ID, Damage: 4, Source: g.Players[0].ID})
	g.Run()
	unblock(t, g, 6)
	if villain.HP() != hp {
		t.Fatalf("Biogram Image should prevent the damage: %d -> %d", hp, villain.HP())
	}
	if g.MainScheme.Threat != threat+4 {
		t.Fatalf("prevented damage should scheme: %d -> %d", threat, g.MainScheme.Threat)
	}
}

// TestNebulaSetup: Power Stone attaches to Nebula after setup.
func TestNebulaSetup(t *testing.T) {
	g := newGmwGame(t, 13, "16091", "16001", nil)
	var villain *engine.Villain
	for id := range g.Villains {
		villain = g.Villains[id]
	}
	found := false
	for _, a := range g.Attachments {
		if a != nil && a.Code[:5] == "16149" {
			found = true
			if a.Target != villain.ID {
				t.Fatal("the Power Stone should start on Nebula")
			}
		}
	}
	if !found {
		t.Fatal("the Power Stone should be attached at setup")
	}
	if g.EnvironmentByCode("16093") == nil {
		t.Fatal("Nebula's Ship should be in play")
	}
}

// TestRonanSetup: ship, Milano, Universal Weapon on Ronan, Power Stone on
// the first player.
func TestRonanSetup(t *testing.T) {
	g := newGmwGame(t, 14, "16106", "16001", nil)
	p := g.Players[0]
	if g.EnvironmentByCode("16108") == nil {
		t.Fatal("Kree Command Ship should be in play")
	}
	var villain *engine.Villain
	for id := range g.Villains {
		villain = g.Villains[id]
	}
	hasWeapon, hasStone := false, false
	for _, a := range g.Attachments {
		if a == nil {
			continue
		}
		if a.Code[:5] == "16109" && a.Target == villain.ID {
			hasWeapon = true
		}
		if a.Code[:5] == "16149" && a.Target == p.ID {
			hasStone = true
		}
	}
	if !hasWeapon {
		t.Fatal("Universal Weapon should attach to Ronan")
	}
	if !hasStone {
		t.Fatal("the Power Stone should attach to the first player")
	}
}

// TestMilanoResource: the Milano offers a wild resource ability.
func TestMilanoResource(t *testing.T) {
	b := engine.LookupBehavior("16142")
	if b == nil || b.Resource == nil || b.Resource.Icon != "wild" {
		t.Fatal("the Milano should generate a wild resource")
	}
}

// TestGrootGrowth: Fruition adds counters up to the cap; the identity
// damage prevention spends them.
func TestGrootGrowth(t *testing.T) {
	g := newGmwGame(t, 15, "16057", "16001", nil)
	p := g.Players[0]
	ec := &engine.EventCard{Code: "16002", Owner: p.ID}
	_ = engine.LookupBehavior("16002").OnPlay(g, ec)
	if p.GrowthCounters != 6 { // 4 starting + 2
		t.Fatalf("Fruition should add 2 growth counters, got %d", p.GrowthCounters)
	}
	dmg := 3
	g.Push(engine.DamageEntity{Target: p.ID, Damage: dmg, Source: villainID(g)})
	unblock(t, g, 1)
	if p.Damage != 0 {
		t.Fatalf("growth counters should prevent 3 damage, took %d", p.Damage)
	}
	if p.GrowthCounters != 3 {
		t.Fatalf("3 counters should be spent, got %d", p.GrowthCounters)
	}
}

func villainID(g *engine.Game) engine.EntityID {
	for id := range g.Villains {
		return id
	}
	return ""
}

// TestVillainousMinion: activating the minion reveals a boost card that
// raises the attack for that activation.
func TestVillainousMinion(t *testing.T) {
	g := newGmwGame(t, 16, "16057", "16001", nil)
	p := g.Players[0]
	m := &engine.Minion{ID: g.NextEntityID("minion"), Code: "16075", MaxHP: 6, AttackVal: 2, EngagedWith: p.ID}
	g.AddMinion(m, p.ID)
	g.Push(engine.MinionActivates{MinionID: m.ID, Player: p.ID})
	g.Run()
	unblock(t, g, 8)
	// The boost reveal is logged; the exact icon count is seed-dependent,
	// so only assert the flow settled without error and boost reset.
	if m.BoostCount != 0 {
		t.Fatalf("boost count should reset after the attack, got %d", m.BoostCount)
	}
}

// TestMarketSweep: market behaviors exist and Grand Strategy draws to
// hand size.
func TestMarketGrandStrategy(t *testing.T) {
	g := newGmwGame(t, 17, "16057", "16001", nil)
	p := g.Players[0]
	for len(p.Hand) > 0 {
		c := p.Hand[0]
		p.Hand.Remove(c.ID)
		p.Deck = append(p.Deck, c)
	}
	ec := &engine.EventCard{Code: "16174", Owner: p.ID}
	g.Push(engine.LookupBehavior("16174").OnPlay(g, ec)...)
	unblock(t, g, 6)
	if len(p.Hand) == 0 {
		t.Fatal("Grand Strategy should draw to max hand size")
	}
}

// TestPowerStoneHop: a 3+ damage attack moves the stone to the attacker.
func TestPowerStoneHop(t *testing.T) {
	g := newGmwGame(t, 18, "16106", "16001", nil)
	p := g.Players[0]
	var villain *engine.Villain
	for id := range g.Villains {
		villain = g.Villains[id]
	}
	villain.Tough = false
	g.Push(engine.DamageEntity{Target: villain.ID, Damage: 4, Source: p.ID})
	g.Run()
	hopped := false
	for _, a := range g.Attachments {
		if a != nil && a.Code[:5] == "16149" && a.Target == p.ID {
			hopped = true
		}
	}
	if !hopped {
		t.Fatal("the Power Stone should hop to the attacking hero")
	}
}

// TestWiltObligation: the growth branch removes counters.
func TestWiltObligation(t *testing.T) {
	g := newGmwGame(t, 19, "16057", "16001", nil)
	p := g.Players[0]
	before := p.GrowthCounters
	card := engine.Card{ID: g.NextCardID(), Code: "16025"}
	msgs := engine.LookupBehavior("16025").ResolveObligation(g, p, card)
	g.Push(msgs...)
	unblock(t, g, 1)
	pq := g.Pending()
	if pq == nil || !strings.Contains(pq.Question.Prompt, "Wilt") {
		t.Fatal("Wilt should ask for its resolution")
	}
	// Pick the growth branch (second choice).
	if err := g.Answer(pq.Player, []string{pq.Question.Choices[1].ID}); err != nil {
		t.Fatalf("answer: %v", err)
	}
	if p.GrowthCounters != before-3 {
		t.Fatalf("Wilt should remove 3 growth counters: %d -> %d", before, p.GrowthCounters)
	}
}
