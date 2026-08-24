package spiderham_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/spiderham"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

var wolandCodes = []string{
	"30012", "30013", "30014", "30015", "30016", "30017", "30018",
	"30019", "30020", "30021", "30022", "30023", "30029", "30030",
	"30031", "30032", "30033", "30034", "30035", "30036", "30037",
	"30038",
}

func TestWoLANDAllRegistered(t *testing.T) {
	for _, code := range wolandCodes {
		if !engine.Implemented(code) {
			t.Errorf("card %s has no registered behavior", code)
		}
	}
}

// lookups
func behavior(t *testing.T, code string) *engine.Behavior {
	t.Helper()
	b := engine.LookupBehavior(code)
	if b == nil {
		t.Fatalf("behavior %s missing", code)
	}
	return b
}

func firstVillain(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil {
			return v
		}
	}
	return nil
}

// TestGreatResponsibilityWindow: a player holding the event takes the
// threat as damage instead when they answer the interrupt prompt.
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

func TestGreatResponsibilityWindow(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	p.Hand = append(p.Hand, engine.Card{ID: "gr", Code: "01061", Owner: p.ID})
	before := g.MainScheme.Threat
	p.Damage = 0

	g.Push(engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 3, Source: g.MainScheme.ID})
	unblock(t, g, 1) // answer whatever menu question is pending ("Your turn")
	pq := g.Pending()
	if pq == nil || !strings.Contains(pq.Question.Prompt, "Great Responsibility") {
		t.Fatalf("pending = %v, want the Great Responsibility prompt", promptOf(pq))
	}
	if err := g.Answer(pq.Player, []string{"gr-play"}); err != nil {
		t.Fatalf("answer: %v", err)
	}
	if g.MainScheme.Threat != before {
		t.Fatalf("threat moved %d → %d; should be unchanged", before, g.MainScheme.Threat)
	}
	if p.Damage != 3 {
		t.Fatalf("player damage = %d, want 3", p.Damage)
	}
	for _, c := range p.Hand {
		if c.Code == "01061" {
			t.Fatal("Great Responsibility was not consumed")
		}
	}
	// Declining places the threat normally.
	p.Hand = append(p.Hand, engine.Card{ID: "gr2", Code: "30015", Owner: p.ID})
	g.Push(engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: g.MainScheme.ID})
	unblock(t, g, 1)
	pq = g.Pending()
	if pq == nil || !strings.Contains(pq.Question.Prompt, "Great Responsibility") {
		t.Fatalf("pending = %v, want the reprint prompt", promptOf(pq))
	}
	if err := g.Answer(pq.Player, []string{"gr-pass"}); err != nil {
		t.Fatalf("answer: %v", err)
	}
	if g.MainScheme.Threat != before+2 {
		t.Fatalf("threat = %d, want %d", g.MainScheme.Threat, before+2)
	}
}

// TestWarningWindow: Warning reduces incoming damage by 1 when played.
func TestWarningWindow(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	p.Hand = append(p.Hand, engine.Card{ID: "warn", Code: "09021", Owner: p.ID})
	p.Damage = 0

	g.Push(engine.DamageEntity{Target: p.ID, Damage: 3, Source: "test"})
	unblock(t, g, 1)
	pq := g.Pending()
	if pq == nil || !strings.Contains(pq.Question.Prompt, "Warning") {
		t.Fatalf("pending = %v, want the Warning prompt", promptOf(pq))
	}
	if err := g.Answer(pq.Player, []string{"warn-play"}); err != nil {
		t.Fatalf("answer: %v", err)
	}
	if p.Damage != 2 {
		t.Fatalf("player damage = %d, want 2", p.Damage)
	}
}

// TestFoiledWindow: Foiled! cancels a scheme activation's boost icons.
func TestFoiledWindow(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	p.Hand = append(p.Hand, engine.Card{ID: "foiled", Code: "09038", Owner: p.ID})
	v := firstVillain(g)
	if v == nil {
		t.Fatal("no villain in play")
	}
	// Find any encounter card with boost icons.
	var boostCode string
	for _, def := range engine.DB.All() {
		if def.Boost != nil && *def.Boost >= 2 && def.Category == data.CategoryEncounter {
			boostCode = def.Code
			break
		}
	}
	if boostCode == "" {
		t.Skip("no boost card in DB")
	}
	v.BoostCards = append(v.BoostCards, engine.Card{ID: "bc", Code: boostCode, FaceDown: true})
	v.BoostCount = 0

	g.Push(engine.RevealBoost{Enemy: v.ID})
	g.Push(engine.ApplyVillainScheme{VillainID: v.ID, Player: p.ID})
	unblock(t, g, 1)
	pq := g.Pending()
	if pq == nil || !strings.Contains(pq.Question.Prompt, "Foiled") {
		t.Fatalf("pending = %v, want the Foiled! prompt", promptOf(pq))
	}
	if v.BoostCount == 0 {
		t.Fatal("boost icons were not added before the prompt")
	}
	if err := g.Answer(pq.Player, []string{"foiled-play"}); err != nil {
		t.Fatalf("answer: %v", err)
	}
	if v.BoostCount != 0 {
		t.Fatalf("boost count = %d, want 0", v.BoostCount)
	}
}

// TestLadySpiderFollowUp: after she thwarts, the different-scheme prompt
// appears while another Web-Warrior card is controlled.
func TestLadySpiderFollowUp(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero // the hero side carries the Web-Warrior trait
	a := &engine.Ally{ID: g.NextEntityID(engine.KindAlly), Code: "30012", Owner: p.ID, ThwartVal: 2, MaxHP: 3}
	g.AddAlly(a, p.ID)
	side := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: "01099", Threat: 5, MaxThreat: 5}
	g.AddSideScheme(side)

	b := behavior(t, "30012")
	msgs := b.React(g, a, engine.AllyThwartWindow{Ally: a.ID, Scheme: g.MainScheme.ID})
	if len(msgs) != 1 {
		t.Fatalf("react produced %d messages, want the question", len(msgs))
	}
	q, ok := msgs[0].(engine.AskQuestion)
	if !ok || !strings.Contains(q.Question.Prompt, "different scheme") {
		t.Fatalf("message = %#v, want the Lady Spider prompt", msgs[0])
	}
}

// TestSpiderManAlly: on enter play he removes 1 threat per controlled
// Web-Warrior card (identity + himself = 2 in this setup).
func TestSpiderManAlly(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	a := &engine.Ally{ID: g.NextEntityID(engine.KindAlly), Code: "30013", Owner: p.ID, MaxHP: 3}
	g.AddAlly(a, p.ID)

	msgs := behavior(t, "30013").OnPlay(g, a)
	if len(msgs) != 1 {
		t.Fatalf("OnPlay produced %d messages, want the question", len(msgs))
	}
	q, ok := msgs[0].(engine.AskQuestion)
	if !ok || !strings.Contains(q.Question.Prompt, "choose a scheme") || len(q.Question.Choices) == 0 {
		t.Fatalf("message = %#v, want the scheme-choice question", msgs[0])
	}
}

// TestEvenTheOdds: thwarts every side scheme [per hero] and damages the
// villain per scheme defeated that way.
func TestEvenTheOdds(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	ec := &engine.EventCard{Code: "30014", Owner: p.ID}
	low := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: "01099", Threat: 1, MaxThreat: 1}
	g.AddSideScheme(low)
	high := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: "01100", Threat: 9, MaxThreat: 9}
	g.AddSideScheme(high)

	msgs := behavior(t, "30014").OnPlay(g, ec)
	var thwarts, damage int
	for _, m := range msgs {
		switch m := m.(type) {
		case engine.ThwartScheme:
			thwarts++
		case engine.DamageEntity:
			damage += m.Damage
		}
	}
	if thwarts != 2 {
		t.Fatalf("thwarts = %d, want 2", thwarts)
	}
	if damage != 1 {
		t.Fatalf("damage = %d, want 1 (one scheme defeated)", damage)
	}
}

// TestMakingAnEntrance: +2 THW on play; heal 2 when a basic thwart clears
// a scheme entirely.
func TestMakingAnEntrance(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	ec := &engine.EventCard{Code: "30016", Owner: p.ID}
	msgs := behavior(t, "30016").OnPlay(g, ec)
	if sb, ok := msgs[0].(engine.ApplyStatBonus); !ok || sb.THW != 2 {
		t.Fatalf("OnPlay = %#v, want +2 THW", msgs[0])
	}
	p.BonusTHW = 2
	g.MainScheme.Threat = 2
	msgs = behavior(t, "30016").React(g, ec, engine.BasicThwart{Player: p.ID, N: 2, Target: g.MainScheme.ID})
	if len(msgs) != 1 {
		t.Fatalf("react = %d messages, want the heal", len(msgs))
	}
	if h, ok := msgs[0].(engine.HealEntity); !ok || h.N != 2 {
		t.Fatalf("message = %#v, want HealEntity 2", msgs[0])
	}
	// Partial removal heals nothing.
	g.MainScheme.Threat = 5
	if msgs := behavior(t, "30016").React(g, ec, engine.BasicThwart{Player: p.ID, N: 2, Target: g.MainScheme.ID}); msgs != nil {
		t.Fatalf("partial clear healed: %#v", msgs)
	}
}

// TestOneWayOrAnother: reveals a side scheme from the encounter deck,
// draws 3, shuffles; max once per round.
func TestOneWayOrAnother(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	ec := &engine.EventCard{Code: "30017", Owner: p.ID}
	// Find any side scheme def to seed the encounter deck with.
	var sideCode string
	for _, def := range engine.DB.All() {
		if def.Type == "side_scheme" {
			sideCode = def.Code
			break
		}
	}
	if sideCode == "" {
		t.Skip("no side scheme in DB")
	}
	g.EncounterDeck = engine.CardList{{ID: "deck-side", Code: sideCode}}

	msgs := behavior(t, "30017").OnPlay(g, ec)
	if len(msgs) != 1 {
		t.Fatalf("OnPlay = %d messages, want the question", len(msgs))
	}
	q := msgs[0].(engine.AskQuestion)
	if len(q.Question.Choices) != 1 {
		t.Fatalf("choices = %d, want the seeded side scheme", len(q.Question.Choices))
	}
	if !g.UsedThisRound["30017"] {
		t.Fatal("per-round guard not set")
	}
	if msgs := behavior(t, "30017").OnPlay(g, ec); msgs != nil {
		t.Fatalf("second play in one round produced %#v", msgs)
	}
}

// TestSPDrReturnsToHand: lethal consequential damage returns her to hand.
func TestSPDrReturnsToHand(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	a := &engine.Ally{ID: g.NextEntityID(engine.KindAlly), Code: "30021", Owner: p.ID, MaxHP: 4, Damage: 3}
	g.AddAlly(a, p.ID)
	handBefore := len(p.Hand)

	behavior(t, "30021").React(g, a, engine.DamageEntity{Target: a.ID, Damage: 1, Source: a.ID})
	if g.Allies[a.ID] != nil {
		t.Fatal("SP//dr should have left play")
	}
	if len(p.Hand) != handBefore+1 || p.Hand[len(p.Hand)-1].Code != "30021" {
		t.Fatalf("hand = %d cards; want SP//dr returned", len(p.Hand))
	}
	// Non-consequential (enemy-sourced) lethal damage does not trigger.
	a2 := &engine.Ally{ID: g.NextEntityID(engine.KindAlly), Code: "30021", Owner: p.ID, MaxHP: 4, Damage: 3}
	g.AddAlly(a2, p.ID)
	behavior(t, "30021").React(g, a2, engine.DamageEntity{Target: a2.ID, Damage: 1, Source: firstVillain(g).ID})
	if g.Allies[a2.ID] == nil {
		t.Fatal("enemy damage should not return SP//dr to hand")
	}
}

// TestWoLANDSupportCost + response draw.
func TestWoLANDSupport(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	def := engine.DB.MustLookup("30023")
	b := behavior(t, "30023")
	if b.CardCost == nil || b.CardCost(g, p, def) != 3 {
		t.Fatalf("Web-Warrior identity should play Web of Life and Destiny for free")
	}
	// Response: a Web-Warrior ally leaving play draws someone a card.
	s := &engine.Support{ID: g.NextEntityID(engine.KindSupport), Code: "30023", Owner: p.ID}
	g.Supports[s.ID] = s
	p.Supports = append(p.Supports, s.ID)
	ww := &engine.Ally{ID: g.NextEntityID(engine.KindAlly), Code: "30012", Owner: p.ID, MaxHP: 3}
	g.AddAlly(ww, p.ID)
	msgs := b.React(g, s, engine.AllyDefeated{AllyID: ww.ID})
	if len(msgs) != 1 {
		t.Fatalf("react = %d messages, want the draw question", len(msgs))
	}
	if _, ok := msgs[0].(engine.AskQuestion); !ok {
		t.Fatalf("message = %#v, want AskQuestion", msgs[0])
	}
}

// TestWarriorAttachesToSpider: the attach question offers the
// Spider-titled identity and grants Web-Warrior.
func TestWarriorAttachesToSpider(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	u := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: "30029", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)

	msgs := behavior(t, "30029").OnPlay(g, u)
	if len(msgs) != 1 {
		t.Fatalf("OnPlay = %d messages, want the question", len(msgs))
	}
	q := msgs[0].(engine.AskQuestion)
	if len(q.Question.Choices) == 0 || !strings.Contains(q.Question.Choices[0].Label, "Spider") {
		t.Fatalf("choices = %#v, want the Spider-titled identity", q.Question.Choices)
	}
}

// TestHuntingTheSpiderTotems: the villain-phase reveal mills 3 and spawns
// Inheritor minions with Web-Warrior players.
func TestHuntingTheSpiderTotems(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	s := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: "30030", Threat: 4, MaxThreat: 6}
	g.AddSideScheme(s)
	g.EncounterDeck = engine.CardList{
		{ID: "m1", Code: "30036"},
		{ID: "x1", Code: "01081"},
		{ID: "m2", Code: "30031"},
	}

	msgs := behavior(t, "30030").React(g, s, engine.BeginPhase{Phase: engine.PhaseVillain})
	spawns := 0
	for _, m := range msgs {
		if _, ok := m.(engine.MinionEntersPlay); ok {
			spawns++
		}
	}
	if spawns != 2 {
		t.Fatalf("spawns = %d, want 2 Inheritors", spawns)
	}
	if len(g.EncounterDeck) != 0 {
		t.Fatalf("deck = %d cards, want emptied", len(g.EncounterDeck))
	}
	engaged := 0
	for _, mn := range g.Minions {
		if mn != nil && mn.EngagedWith == p.ID {
			engaged++
		}
	}
	if engaged != 2 {
		t.Fatalf("engaged with the Web-Warrior player = %d, want 2", engaged)
	}
}

// TestInheritorReveals sweeps the eight When Revealed effects.
func TestInheritorReveals(t *testing.T) {
	g := newHamGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero // Web-Warrior character in play gates the reveals
	addInheritor := func(code string) *engine.Minion {
		mn := &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: code, MaxHP: 4}
		g.AddMinion(mn, p.ID)
		return mn
	}

	// Bora: 1 threat on each scheme.
	g.MainScheme.Threat = 0
	mn := addInheritor("30031")
	msgs := behavior(t, "30031").React(g, mn, engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
	if len(msgs) < 1 {
		t.Fatal("Bora reveal produced no threat")
	}
	for _, m := range msgs {
		if st, ok := m.(engine.SchemeThreat); !ok || st.N != 1 {
			t.Fatalf("Bora message = %#v, want 1 threat per scheme", m)
		}
	}

	// Brix: 2 threat on the main scheme.
	mn = addInheritor("30032")
	msgs = behavior(t, "30032").React(g, mn, engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
	if st, ok := msgs[0].(engine.SchemeThreat); !ok || st.N != 2 {
		t.Fatalf("Brix message = %#v, want 2 threat on the main scheme", msgs[0])
	}

	// Daemos: stun a controlled character.
	mn = addInheritor("30033")
	msgs = behavior(t, "30033").React(g, mn, engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
	if len(msgs) != 1 {
		t.Fatalf("Daemos = %d messages, want the stun question", len(msgs))
	}

	// Jennix: tough status card.
	mn = addInheritor("30034")
	msgs = behavior(t, "30034").React(g, mn, engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
	if _, ok := msgs[0].(engine.ToughEntity); !ok {
		t.Fatalf("Jennix message = %#v, want ToughEntity", msgs[0])
	}

	// Karn: discard an upgrade or support.
	sup := &engine.Support{ID: g.NextEntityID(engine.KindSupport), Code: "30008", Owner: p.ID}
	g.Supports[sup.ID] = sup
	p.Supports = append(p.Supports, sup.ID)
	mn = addInheritor("30035")
	msgs = behavior(t, "30035").React(g, mn, engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
	if len(msgs) != 1 {
		t.Fatalf("Karn = %d messages, want the discard question", len(msgs))
	}

	// Morlun: +1 ATK to other Inheritors and 2 damage to the revealer.
	other := addInheritor("30031")
	mn = addInheritor("30036")
	msgs = behavior(t, "30036").React(g, mn, engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
	var boost, dmg bool
	for _, m := range msgs {
		switch m := m.(type) {
		case engine.BoostEnemyAttack:
			boost = m.Enemy == other.ID && m.N == 1
		case engine.DamageEntity:
			dmg = m.Damage == 2 && m.Target == p.ID
		}
	}
	if !boost || !dmg {
		t.Fatalf("Morlun messages = %#v, want +1 ATK on the other Inheritor and 2 damage", msgs)
	}

	// Solus: a facedown boost card.
	mn = addInheritor("30037")
	behavior(t, "30037").React(g, mn, engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
	if mn.BoostCount != 1 {
		t.Fatalf("Solus boost count = %d, want 1", mn.BoostCount)
	}

	// Verna: 1 damage to each controlled character.
	ally := &engine.Ally{ID: g.NextEntityID(engine.KindAlly), Code: "30012", Owner: p.ID, MaxHP: 3}
	g.AddAlly(ally, p.ID)
	mn = addInheritor("30038")
	msgs = behavior(t, "30038").React(g, mn, engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
	if len(msgs) != 2 {
		t.Fatalf("Verna = %d messages, want damage to identity and ally", len(msgs))
	}
	for _, m := range msgs {
		if d, ok := m.(engine.DamageEntity); !ok || d.Damage != 1 {
			t.Fatalf("Verna message = %#v, want 1 damage", m)
		}
	}
}

func promptOf(pq *engine.PendingQuestion) string {
	if pq == nil {
		return "<none>"
	}
	return pq.Question.Prompt
}
