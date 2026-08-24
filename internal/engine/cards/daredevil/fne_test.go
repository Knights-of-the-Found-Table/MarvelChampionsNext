package daredevil_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/daredevil"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/echo"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/extras"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/goblinfooblin"
)

// start answers the pending setup prompt (the mulligan); any messages
// pushed before it process during the resume. The engine blocks newly
// pushed messages behind a pending question, so triggers must be queued
// before the setup answer.
func start(t *testing.T, g *engine.Game) {
	t.Helper()
	pq := g.Pending()
	if pq == nil {
		return
	}
	if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
		t.Fatalf("setup answer %q: %v", pq.Question.Prompt, err)
	}
}

// answerUntil keeps answering pending questions with the default pick
// until one offers the wanted choice id (or the budget runs out); it then
// answers that choice and settles any follow-up questions it opened.
func answerUntil(t *testing.T, g *engine.Game, wantID string) bool {
	t.Helper()
	for i := 0; i < 6; i++ {
		pq := g.Pending()
		if pq == nil {
			return false
		}
		for _, c := range pq.Question.Choices {
			if c.ID != wantID || c.Disabled {
				continue
			}
			// a choice with a nested Then question is answered as one
			// path: "<outer>.<first inner choice>".
			path := wantID
			if c.Then != nil && len(c.Then.Choices) > 0 {
				path = c.Then.Choices[0].ID // inner ids carry the prefix
			}
			if err := g.Answer(pq.Player, []string{path}); err != nil {
				t.Fatalf("answer %q: %v", pq.Question.Prompt, err)
			}
			return true
		}
		if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
			t.Fatalf("answer %q: %v", pq.Question.Prompt, err)
		}
		if g.Over {
			return false
		}
	}
	return false
}

// TestFNECardsImplemented: every remaining Fear No Evil box card has a
// hand-written behavior.
func TestFNECardsImplemented(t *testing.T) {
	codes := []string{
		"60019", "60020", "60021", "60022", "60023", "60024", "60025",
		"60026", "60027", "60028", "60029", "60031", "60032",
		"60039", "60049", "60051", "60055", "60056",
		"60057", "60058", "60059",
		"60060", "60061", "60062", "60063", "60064",
	}
	for _, c := range codes {
		if !engine.Implemented(c) {
			t.Errorf("card %s has no hand-written behavior", c)
		}
	}
}

// TestLegalTroubleSchemeReduction: an attached Legal Trouble lowers the
// minion's scheme placement by 2.
func TestLegalTroubleSchemeReduction(t *testing.T) {
	g := newDaredevilGame(t, "01097", 7)
	p := g.Players[0]
	p.Side = engine.SideAlterEgo
	mn := &engine.Minion{
		ID: g.NextEntityID("minion"), Code: "01103", MaxHP: 3,
		AttackVal: 2, SchemeVal: 2, EngagedWith: p.ID,
	}
	g.Minions[mn.ID] = mn
	u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "60026", Owner: p.ID, AttachTo: mn.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)

	before := g.MainScheme.Threat
	g.Push(engine.MinionActivates{MinionID: mn.ID, Player: p.ID})
	start(t, g)
	if got := g.MainScheme.Threat - before; got != 0 {
		t.Fatalf("Legal Trouble should reduce the scheme placement to 0, placed %d", got)
	}
}

// TestDeEscalationRemovesToken: defeating De-escalation strips an
// acceleration token from the main scheme.
func TestDeEscalationRemovesToken(t *testing.T) {
	g := newDaredevilGame(t, "01097", 7)
	p := g.Players[0]
	g.MainScheme.AccelerationTokens = 1
	s := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: "60024", Threat: 1, MaxThreat: 4, PlayerSide: true, Owner: p.ID}
	g.SideSchemes[s.ID] = s

	g.Push(engine.ThwartScheme{Scheme: s.ID, N: 1, Source: p.ID})
	start(t, g)
	if g.SideSchemes[s.ID] != nil {
		t.Fatal("De-escalation should have been defeated")
	}
	if g.MainScheme.AccelerationTokens != 0 {
		t.Fatalf("acceleration token should be removed, have %d", g.MainScheme.AccelerationTokens)
	}
}

// TestChanceEncounterFetch: when the attached side scheme is defeated, an
// ally is fetched to hand and the upgrade is discarded.
func TestChanceEncounterFetch(t *testing.T) {
	g := newDaredevilGame(t, "01097", 7)
	p := g.Players[0]
	s := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: "01109", Threat: 1, MaxThreat: 3}
	g.SideSchemes[s.ID] = s
	u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "60025", Owner: p.ID, AttachTo: s.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)
	allyCard := engine.Card{ID: g.NextCardID(), Code: "60007", Owner: p.ID}
	p.Discard = append(p.Discard, allyCard)

	handBefore := len(p.Hand)
	g.Push(engine.ThwartScheme{Scheme: s.ID, N: 1, Source: p.ID})
	start(t, g)
	if len(p.Hand) != handBefore+1 {
		t.Fatalf("Chance Encounter should fetch an ally to hand, hand %d -> %d", handBefore, len(p.Hand))
	}
	if g.Upgrades[u.ID] != nil {
		t.Fatal("Chance Encounter should be discarded after fetching")
	}
}

// TestMoveInShadowThreat: playing an event removes 1 threat.
func TestMoveInShadowThreat(t *testing.T) {
	g := newDaredevilGame(t, "01097", 7)
	p := g.Players[0]
	u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "60027", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)

	g.MainScheme.Threat = 5
	g.Push(engine.EventPlayed{Player: p.ID, Card: engine.Card{ID: g.NextCardID(), Code: "60023"}})
	start(t, g)
	if g.MainScheme.Threat != 4 {
		t.Fatalf("Move in Shadow should remove 1 threat, have %d", g.MainScheme.Threat)
	}
}

// TestMoveInShadowTemporary: the upgrade returns to hand at the end of
// the round.
func TestMoveInShadowTemporary(t *testing.T) {
	g := newDaredevilGame(t, "01097", 7)
	p := g.Players[0]
	u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "60027", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)

	handBefore := len(p.Hand)
	g.Push(engine.EndRound{})
	start(t, g)
	if g.Upgrades[u.ID] != nil {
		t.Fatal("Move in Shadow should leave play at the end of the round")
	}
	if len(p.Hand) < handBefore+1 {
		t.Fatalf("Move in Shadow should return to hand, hand %d -> %d", handBefore, len(p.Hand))
	}
}

// TestStealthTrainingStun: defeating a side scheme with a thwart opens the
// stun window.
func TestStealthTrainingStun(t *testing.T) {
	g := newDaredevilGame(t, "01097", 7)
	p := g.Players[0]
	u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "60028", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)
	s := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: "01109", Threat: 1, MaxThreat: 3}
	g.SideSchemes[s.ID] = s
	mn := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01103", MaxHP: 3, EngagedWith: p.ID}
	g.Minions[mn.ID] = mn

	g.Push(engine.ThwartScheme{Scheme: s.ID, N: 1, Source: p.ID})
	start(t, g)
	if !answerUntil(t, g, "use") {
		t.Fatal("Stealth Training should offer its stun window")
	}
	if !u.Exhausted || !anyEnemyStunned(g) {
		t.Fatal("Stealth Training should exhaust and stun an enemy")
	}
}

// TestStickBoost: exhausting Stick adds +1 to a friendly Martial Artist
// basic thwart (Daredevil qualifies).
func TestStickBoost(t *testing.T) {
	g := newDaredevilGame(t, "01097", 7)
	p := g.Players[0]
	p.Side = engine.SideHero
	sp := &engine.Support{ID: g.NextEntityID("support"), Code: "60029", Owner: p.ID}
	g.Supports[sp.ID] = sp
	p.Supports = append(p.Supports, sp.ID)

	g.MainScheme.Threat = 6
	g.Push(engine.BasicThwart{Player: p.ID, N: 1, Target: g.MainScheme.ID})
	start(t, g)
	if !answerUntil(t, g, "use") {
		t.Fatal("Stick should offer its boost window")
	}
	if got := 6 - g.MainScheme.Threat; got != 2 {
		t.Fatalf("Stick should add 1 threat removal, removed %d", got)
	}
}

// anyEnemyStunned reports whether any villain or minion is stunned.
func anyEnemyStunned(g *engine.Game) bool {
	for _, v := range g.Villains {
		if v != nil && v.Stunned {
			return true
		}
	}
	for _, mn := range g.Minions {
		if mn != nil && mn.Stunned {
			return true
		}
	}
	return false
}
