package echo_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/echo"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/extras"
)

func newEchoGame(t *testing.T, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Maya", HeroBase: "60037", Deck: map[string]int{
				"60039": 1, "60045": 1, "60049": 1, "60051": 1,
				"01088": 2, "01089": 2, "01090": 2,
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// start answers the pending setup prompt (the mulligan); messages pushed
// before it process during the resume (the engine blocks pushes behind a
// pending question).
func start(t *testing.T, g *engine.Game) {
	t.Helper()
	pq := g.Pending()
	if pq == nil {
		return
	}
	// default pick: first choice ("keep" on the mulligan)
	if err := g.Answer(pq.Player, []string{pq.Question.Choices[0].ID}); err != nil {
		t.Fatalf("setup answer %q: %v", pq.Question.Prompt, err)
	}
}

func spawnKingpin(g *engine.Game, pid engine.PlayerID) *engine.Minion {
	mn := &engine.Minion{
		ID: g.NextEntityID("minion"), Code: "60061", MaxHP: 6,
		AttackVal: 2, SchemeVal: 3, EngagedWith: pid,
	}
	g.Minions[mn.ID] = mn
	return mn
}

// TestKingpinSchemesAgainstEcho: Kingpin schemes instead of attacking the
// Maya Lopez player.
func TestKingpinSchemesAgainstEcho(t *testing.T) {
	g := newEchoGame(t, 13)
	p := g.Players[0]
	p.Side = engine.SideHero
	mn := spawnKingpin(g, p.ID)

	before := g.MainScheme.Threat
	g.Push(engine.MinionActivates{MinionID: mn.ID, Player: p.ID})
	start(t, g)
	if got := g.MainScheme.Threat - before; got != 3 {
		t.Fatalf("Kingpin should scheme for 3 against Echo, placed %d", got)
	}
	if pq := g.Pending(); pq != nil && pq.Question.Prompt != "Your turn" {
		t.Fatalf("Kingpin should not attack Echo, pending %q", pq.Question.Prompt)
	}
}

// TestKingpinAttacksOthers: without the Maya Lopez redirect Kingpin runs
// the default attack against another hero.
func TestKingpinAttacksOthers(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       13,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Matt", HeroBase: "60001", Deck: map[string]int{
				"01088": 2, "01089": 2, "01090": 2,
			}}},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	p := g.Players[0]
	p.Side = engine.SideHero
	mn := spawnKingpin(g, p.ID)

	g.Push(engine.MinionActivates{MinionID: mn.ID, Player: p.ID})
	start(t, g)
	if pq := g.Pending(); pq == nil || pq.Question.Prompt == "Your turn" {
		t.Fatalf("Kingpin should attack a non-Echo hero, pending %v", pq)
	}
}

// TestKingpinDamageBlocked: Kingpin cannot take damage while Master
// Manipulator is in play.
func TestKingpinDamageBlocked(t *testing.T) {
	g := newEchoGame(t, 13)
	p := g.Players[0]
	mn := spawnKingpin(g, p.ID)
	s := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: "60062", Threat: 2, MaxThreat: 5}
	g.SideSchemes[s.ID] = s

	g.Push(engine.DamageEntity{Target: mn.ID, Damage: 4, Source: p.ID})
	start(t, g)
	if mn.HP() != 6 {
		t.Fatalf("Kingpin should be immune while Master Manipulator is in play, HP %d", mn.HP())
	}
}

// TestKingpinDamageUnblocked: without the scheme Kingpin takes damage.
func TestKingpinDamageUnblocked(t *testing.T) {
	g := newEchoGame(t, 13)
	p := g.Players[0]
	mn := spawnKingpin(g, p.ID)

	g.Push(engine.DamageEntity{Target: mn.ID, Damage: 4, Source: p.ID})
	start(t, g)
	if mn.HP() != 2 {
		t.Fatalf("Kingpin should take damage once the scheme is gone, HP %d", mn.HP())
	}
}

// TestPawnOfTheKingpin: hero — self-damage equal to ATK.
func TestPawnOfTheKingpin(t *testing.T) {
	g := newEchoGame(t, 13)
	p := g.Players[0]
	p.Side = engine.SideHero

	atk := p.AttackStat(g)
	hpBefore := p.HP()
	g.Push(engine.TreacheryResolve{Player: p.ID, Card: engine.Card{ID: g.NextCardID(), Code: "60064"}})
	start(t, g)
	if p.HP() != hpBefore-atk {
		t.Fatalf("hero should take %d damage, HP %d -> %d", atk, hpBefore, p.HP())
	}
}

// TestPawnOfTheKingpinKingpinSchemes: alter-ego with Kingpin in play —
// Kingpin schemes.
func TestPawnOfTheKingpinKingpinSchemes(t *testing.T) {
	g := newEchoGame(t, 13)
	p := g.Players[0]
	p.Side = engine.SideAlterEgo
	spawnKingpin(g, p.ID)

	before := g.MainScheme.Threat
	g.Push(engine.TreacheryResolve{Player: p.ID, Card: engine.Card{ID: g.NextCardID(), Code: "60064"}})
	start(t, g)
	if got := g.MainScheme.Threat - before; got != 3 {
		t.Fatalf("Kingpin should scheme for 3 via Pawn of the Kingpin, placed %d", got)
	}
}

// TestSuperpowerTrainingFieldsUpgrade: defeating the scheme fields an
// identity-specific upgrade from the deck.
func TestSuperpowerTrainingFieldsUpgrade(t *testing.T) {
	g := newEchoGame(t, 13)
	p := g.Players[0]
	s := &engine.SideScheme{ID: g.NextEntityID(engine.KindSideScheme), Code: "60056", Threat: 1, MaxThreat: 3, PlayerSide: true, Owner: p.ID}
	g.SideSchemes[s.ID] = s
	katana := engine.Card{ID: g.NextCardID(), Code: "60045", Owner: p.ID}
	p.Deck = append(p.Deck, katana)

	upgradesBefore := len(p.Upgrades)
	g.Push(engine.ThwartScheme{Scheme: s.ID, N: 1, Source: p.ID})
	start(t, g)
	if len(p.Upgrades) != upgradesBefore+1 {
		t.Fatalf("Superpower Training should field an upgrade, %d -> %d", upgradesBefore, len(p.Upgrades))
	}
}

// TestDaredevilAllyDiscount: after Daredevil uses a basic power, the next
// event costs 1 less.
func TestDaredevilAllyDiscount(t *testing.T) {
	g := newEchoGame(t, 13)
	p := g.Players[0]
	a := &engine.Ally{ID: g.NextEntityID("ally"), Code: "60039", Owner: p.ID, MaxHP: 3, AttackVal: 2, ThwartVal: 2}
	g.Allies[a.ID] = a
	p.Allies = append(p.Allies, a.ID)

	g.Push(engine.AllyThwartWindow{Ally: a.ID, Scheme: g.MainScheme.ID})
	start(t, g)
	if len(p.CostDiscounts) != 1 || p.CostDiscounts[0].Type != "event" || p.CostDiscounts[0].Amount != 1 {
		t.Fatalf("expected one event discount, got %+v", p.CostDiscounts)
	}
}
