package spdr_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/spdr"
)

var ironSpiderCodes = []string{
	"31014", "31015", "31016", "31017", "31018", "31019", "31020",
	"31021", "31022", "31023", "31024", "31029", "31030", "31031",
	"31032", "31033", "31034", "31035", "31036", "31037",
}

func TestIronSpiderAllRegistered(t *testing.T) {
	for _, code := range ironSpiderCodes {
		if !engine.Implemented(code) {
			t.Errorf("card %s has no registered behavior", code)
		}
	}
}

func ispBehavior(t *testing.T, code string) *engine.Behavior {
	t.Helper()
	b := engine.LookupBehavior(code)
	if b == nil {
		t.Fatalf("behavior %s missing", code)
	}
	return b
}

func firstVillainID(g *engine.Game) engine.EntityID {
	for _, v := range g.Villains {
		if v != nil {
			return v.ID
		}
	}
	return ""
}

// TestDaredevilDefenseShift: after defending, 1 damage moves to the
// attacker.
func TestDaredevilDefenseShift(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	a := &engine.Ally{ID: g.NextEntityID(engine.KindAlly), Code: "31014", Owner: p.ID, MaxHP: 3, Damage: 2}
	g.AddAlly(a, p.ID)
	villain := firstVillainID(g)

	msgs := ispBehavior(t, "31014").React(g, a, engine.WindowDefended{Defender: a.ID, Against: villain})
	if len(msgs) != 1 {
		t.Fatalf("react = %d messages, want the shift", len(msgs))
	}
	if d, ok := msgs[0].(engine.DamageEntity); !ok || d.Target != villain || d.Damage != 1 {
		t.Fatalf("message = %#v, want 1 damage to the attacker", msgs[0])
	}
	if a.Damage != 1 {
		t.Fatalf("Daredevil damage = %d, want 1", a.Damage)
	}
}

// TestSpiderManNoirTucks: resolving a treachery raises X while another
// Web-Warrior card is controlled (max 3).
func TestSpiderManNoirTucks(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	a := &engine.Ally{ID: g.NextEntityID(engine.KindAlly), Code: "31015", Owner: p.ID, MaxHP: 3}
	g.AddAlly(a, p.ID)
	card := engine.Card{ID: "t1", Code: "31028"}

	ispBehavior(t, "31015").React(g, a, engine.TreacheryResolve{Player: p.ID, Card: card})
	if a.Counters != 1 || a.BonusATK != 1 || a.BonusTHW != 1 {
		t.Fatalf("after one tuck: counters=%d atk=%d thw=%d, want 1/1/1", a.Counters, a.BonusATK, a.BonusTHW)
	}
	// Cap at 3.
	ispBehavior(t, "31015").React(g, a, engine.TreacheryResolve{Player: p.ID, Card: card})
	ispBehavior(t, "31015").React(g, a, engine.TreacheryResolve{Player: p.ID, Card: card})
	ispBehavior(t, "31015").React(g, a, engine.TreacheryResolve{Player: p.ID, Card: card})
	if a.Counters != 3 {
		t.Fatalf("counters = %d, want capped at 3", a.Counters)
	}
	// Cancelled resolutions do not tuck.
	if ispBehavior(t, "31015").React(g, a, engine.TreacheryResolve{Player: p.ID, Card: card, Cancelled: true}); a.Counters != 3 {
		t.Fatal("a cancelled treachery should not tuck")
	}
}

// TestBarriers: Energy Barrier prevents 1 and reflects 1 per counter;
// Forcefield Generator prevents the whole hit while counters last.
func TestBarriers(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	eb := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: "31018", Owner: p.ID, Counters: 2}
	g.Upgrades[eb.ID] = eb
	pv, refl := ispBehavior(t, "31018").DamagePrevention(g, eb, p, 3)
	if pv != 1 || refl != 1 || eb.Counters != 1 {
		t.Fatalf("Energy Barrier: prevented=%d reflect=%d counters=%d", pv, refl, eb.Counters)
	}
	eb.Counters = 0
	if pv, _ = ispBehavior(t, "31018").DamagePrevention(g, eb, p, 3); pv != 0 {
		t.Fatal("empty Energy Barrier should prevent nothing")
	}

	fg := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: "31019", Owner: p.ID, Counters: 6}
	g.Upgrades[fg.ID] = fg
	pv, refl = ispBehavior(t, "31019").DamagePrevention(g, fg, p, 4)
	if pv != 4 || refl != 0 || fg.Counters != 2 {
		t.Fatalf("Forcefield: prevented=%d counters=%d", pv, fg.Counters)
	}
}

// TestLimitlessStaminaGate: the 14-HP play requirement gates both cards.
func TestLimitlessStaminaGate(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	b23 := ispBehavior(t, "31023")
	b24 := ispBehavior(t, "31024")
	if b23.Playable == nil || b24.Playable == nil {
		t.Fatal("play gates missing")
	}
	// SP//dr Suit has 14 printed hit points, so she qualifies.
	if !b23.Playable(g, p, nil) || !b24.Playable(g, p, nil) {
		t.Fatal("a 14-HP identity should be allowed to play these")
	}
}

// TestUnshakableSteady: stun/confuse are cleared right after application.
func TestUnshakableSteady(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	u := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: "31024", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)

	msgs := ispBehavior(t, "31024").React(g, u, engine.StunEntity{Target: p.ID})
	if len(msgs) != 1 {
		t.Fatalf("react = %d messages, want the clear", len(msgs))
	}
	if _, ok := msgs[0].(engine.ClearStun); !ok {
		t.Fatalf("message = %#v, want ClearStun", msgs[0])
	}
}

// TestClarityResource: the wild resource exposes DamageAttached.
func TestClarityResource(t *testing.T) {
	b := ispBehavior(t, "31029")
	if b.Resource == nil || b.Resource.Icon != "wild" || !b.Resource.HeroOnly || b.Resource.DamageAttached != 1 {
		t.Fatalf("Clarity resource = %#v", b.Resource)
	}
}

// TestSpiderHamMill: attacking with him mills one encounter card and
// damages him per boost icon.
func TestSpiderHamMill(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	a := &engine.Ally{ID: g.NextEntityID(engine.KindAlly), Code: "31021", Owner: p.ID, MaxHP: 4}
	g.AddAlly(a, p.ID)
	// Top encounter card with boost icons.
	var boostCode string
	for _, def := range engine.DB.All() {
		if def.Boost != nil && *def.Boost == 2 {
			boostCode = def.Code
			break
		}
	}
	if boostCode == "" {
		t.Skip("no boost card in DB")
	}
	g.EncounterDeck = engine.CardList{{ID: "top", Code: boostCode}}

	msgs := ispBehavior(t, "31021").React(g, a, engine.AllyAttackWindow{Ally: a.ID, Target: firstVillainID(g)})
	if len(msgs) != 1 {
		t.Fatalf("react = %d messages, want self-damage", len(msgs))
	}
	if d, ok := msgs[0].(engine.DamageEntity); !ok || d.Damage != 2 || d.Target != a.ID {
		t.Fatalf("message = %#v, want 2 damage to Spider-Ham", msgs[0])
	}
	if len(g.EncounterDeck) != 0 {
		t.Fatal("the encounter card was not milled")
	}
}

// TestHobgoblinActivation: attacking mills ATK cards and converts boost
// icons to indirect damage.
func TestHobgoblinActivation(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	mn := &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: "31033", MaxHP: 5, AttackVal: 2}
	g.AddMinion(mn, p.ID)
	var boostCode string
	for _, def := range engine.DB.All() {
		if def.Boost != nil && *def.Boost == 1 {
			boostCode = def.Code
			break
		}
	}
	if boostCode == "" {
		t.Skip("no boost card in DB")
	}
	g.EncounterDeck = engine.CardList{{ID: "h1", Code: boostCode}, {ID: "h2", Code: boostCode}}

	msgs := ispBehavior(t, "31033").MinionActivate(g, mn, p)
	if len(msgs) != 1 {
		t.Fatalf("activation = %d messages, want the indirect damage", len(msgs))
	}
	if d, ok := msgs[0].(engine.IndirectDamage); !ok || d.N != 2 || d.Player != p.ID {
		t.Fatalf("message = %#v, want 2 indirect damage", msgs[0])
	}
}

// TestElectroAttach: engaging Electro offers the energy-card attach, and
// the handler grants +1 max HP per energy icon.
func TestElectroAttach(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	mn := &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: "31032", MaxHP: 4}
	g.AddMinion(mn, p.ID)
	energy := engine.Card{ID: "e1", Code: "01088"} // Energy: printed [energy]
	p.Hand = append(p.Hand, energy)

	msgs := ispBehavior(t, "31032").React(g, mn, engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
	if len(msgs) != 1 {
		t.Fatalf("react = %d messages, want the attach question", len(msgs))
	}
	handBefore := len(p.Hand)
	g.Push(msgs...)
	unblockISP(t, g, 2) // mulligan, then the attach choice
	if mn.MaxHP != 5 {
		t.Fatalf("Electro max HP = %d, want 5", mn.MaxHP)
	}
	if len(p.Hand) != handBefore-1 || len(mn.TuckedCards) != 1 {
		t.Fatalf("hand=%d tucked=%d, want the card attached", len(p.Hand), len(mn.TuckedCards))
	}
}

// TestGrandLarcenyGate: threat cannot be removed while a Criminal is in
// play.
func TestGrandLarcenyGate(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	s := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: "31030", Threat: 3, MaxThreat: 6}
	g.AddSideScheme(s)
	mn := &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: "31031", MaxHP: 4}
	g.AddMinion(mn, p.ID) // Criminal trait

	g.Push(engine.ThwartScheme{Scheme: s.ID, N: 2, Source: p.ID})
	unblockISP(t, g, 2)
	if s.Threat != 3 {
		t.Fatalf("threat = %d, want locked at 3", s.Threat)
	}
	// Criminal gone — the lock lifts.
	g.Delete(mn.ID)
	g.Push(engine.ThwartScheme{Scheme: s.ID, N: 2, Source: p.ID})
	unblockISP(t, g, 2)
	if s.Threat != 1 {
		t.Fatalf("threat = %d, want 1", s.Threat)
	}
}

// TestSurgeInCrimeAbility: with no Criminals the discard action exists.
func TestSurgeInCrimeAbility(t *testing.T) {
	g := newSPDRGame(t)
	s := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: "31037", Threat: 2, MaxThreat: 4}
	g.AddSideScheme(s)
	abilities := ispBehavior(t, "31037").Abilities(g, s)
	if len(abilities) != 1 {
		t.Fatalf("abilities = %d, want the discard action", len(abilities))
	}
	mn := &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: "31031", MaxHP: 4}
	g.AddMinion(mn, g.Players[0].ID)
	if abilities = ispBehavior(t, "31037").Abilities(g, s); abilities != nil {
		t.Fatal("the action should vanish while a Criminal is in play")
	}
}

// TestSpiderTingleCancel: the upgrade's treachery interrupt cancels the
// reveal and discards itself (offered through the treachery window).
func TestSpiderTingleCancel(t *testing.T) {
	g := newSPDRGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	u := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: "31020", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)
	card := engine.Card{ID: "tr", Code: "31028"}
	repl := ispBehavior(t, "31020").TreacheryInterrupt(g, p, card)
	if len(repl) != 3 {
		t.Fatalf("replacement = %d messages, want damage + discard + cancel", len(repl))
	}
	if d, ok := repl[0].(engine.DamageEntity); !ok || d.Damage != 1 {
		t.Fatalf("first message = %#v, want the cost damage", repl[0])
	}
	if _, ok := repl[1].(engine.DiscardControlled); !ok {
		t.Fatalf("second message = %#v, want the self-discard", repl[1])
	}
	if res, ok := repl[2].(engine.TreacheryResolve); !ok || !res.Cancelled {
		t.Fatalf("third message = %#v, want cancelled resolution", repl[2])
	}
}

func unblockISP(t *testing.T, g *engine.Game, limit int) {
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
