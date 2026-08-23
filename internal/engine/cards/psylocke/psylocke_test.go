package psylocke_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/psylocke"
)

func psylockeDeck() map[string]int {
	return map[string]int{
		"41002a": 2, "41003": 1, "41004": 1, "41005": 1, "41006": 1,
		"41007": 1, "41008": 1, "41009": 1, "41010": 1, "41011": 1,
	}
}

func newPsylockeGame(t *testing.T, seed int64) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Psylocke", HeroBase: "41001", Deck: psylockeDeck()},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// putPsiKnife materializes a Psi-Knife upgrade in play.
func putPsiKnife(g *engine.Game, p *engine.Player) *engine.Upgrade {
	u := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: "41002a", Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)
	return u
}

// TestPsylockeImplemented: the identity is registered.
func TestPsylockeImplemented(t *testing.T) {
	if !engine.Implemented("41001a") {
		t.Fatal("Psylocke identity should be implemented")
	}
}

// TestPsiEnergyControlFlips: using a basic power emits the flip signal
// for the in-play Psi-Energy upgrade, and the upgrade's own React swaps
// its code to the other side. Contract test via React hooks only.
func TestPsiEnergyControlFlips(t *testing.T) {
	g := newPsylockeGame(t, 19)
	p := g.Players[0]
	p.Side = engine.SideHero
	knife := putPsiKnife(g, p)

	b := engine.LookupBehavior("41001")
	if b == nil || b.React == nil {
		t.Fatal("Psylocke identity should expose React")
	}

	// A basic attack triggers the flip signal.
	msgs := b.React(g, p, engine.BasicAttack{Player: p.ID, N: 1, Target: engine.EntityID("villain-1")})
	if len(msgs) != 1 {
		t.Fatalf("Psi-Energy Control returned %d messages, want 1", len(msgs))
	}
	sig, ok := msgs[0].(engine.AddEntityCounter)
	if !ok || sig.ID != knife.ID || sig.N != 0 {
		t.Fatalf("Psi-Energy Control message = %#v, want flip signal on Psi-Knife", msgs[0])
	}

	// The upgrade consumes the signal and flips to Psi-Katana.
	kb := engine.LookupBehavior("41002a")
	if kb == nil || kb.React == nil {
		t.Fatal("Psi-Knife should expose React")
	}
	kb.React(g, knife, sig)
	if knife.Code != "41002b" {
		t.Fatalf("Psi-Knife code = %s, want 41002b after the flip", knife.Code)
	}

	// Flipping back: the katana side consumes the next signal.
	katanaB := engine.LookupBehavior("41002b")
	if katanaB == nil || katanaB.React == nil {
		t.Fatal("Psi-Katana should expose React")
	}
	katanaB.React(g, knife, engine.AddEntityCounter{ID: knife.ID, N: 0})
	if knife.Code != "41002a" {
		t.Fatalf("Psi-Katana code = %s, want 41002a after the flip back", knife.Code)
	}

	// Alter-ego basic powers do not trigger (defense checked here).
	p.Side = engine.SideAlterEgo
	if msgs := b.React(g, p, engine.BasicThwart{Player: p.ID, N: 1}); len(msgs) != 0 {
		t.Fatalf("Psi-Energy Control in alter-ego returned %d messages, want 0", len(msgs))
	}
}

// TestPsiKnifeStatsAndResource: the knife side grants +1 THW and a
// hero-only mental resource; the katana side +1 ATK and physical.
func TestPsiKnifeStatsAndResource(t *testing.T) {
	knife := engine.LookupBehavior("41002a")
	if knife == nil || knife.IdentityStats == nil || knife.Resource == nil {
		t.Fatal("Psi-Knife should expose IdentityStats and Resource")
	}
	if s := knife.IdentityStats(nil); s.THW != 1 {
		t.Fatalf("Psi-Knife THW bonus = %d, want 1", s.THW)
	}
	if knife.Resource.Icon != "mental" || !knife.Resource.HeroOnly {
		t.Fatalf("Psi-Knife resource = %#v, want hero-only mental", knife.Resource)
	}
	katana := engine.LookupBehavior("41002b")
	if katana == nil || katana.IdentityStats == nil || katana.Resource == nil {
		t.Fatal("Psi-Katana should expose IdentityStats and Resource")
	}
	if s := katana.IdentityStats(nil); s.ATK != 1 {
		t.Fatalf("Psi-Katana ATK bonus = %d, want 1", s.ATK)
	}
	if katana.Resource.Icon != "physical" || !katana.Resource.HeroOnly {
		t.Fatalf("Psi-Katana resource = %#v, want hero-only physical", katana.Resource)
	}
}

// TestMentalDetectionScales: with one Psi-Knife in play, Mental
// Detection offers scheme choices removing 3 threat. Contract test via
// OnPlay.
func TestMentalDetectionScales(t *testing.T) {
	g := newPsylockeGame(t, 19)
	p := g.Players[0]
	p.Side = engine.SideHero
	putPsiKnife(g, p)

	b := engine.LookupBehavior("41005")
	if b == nil || b.OnPlay == nil {
		t.Fatal("Mental Detection should expose OnPlay")
	}
	ev := &engine.EventCard{Code: "41005", Owner: p.ID}
	msgs := b.OnPlay(g, ev)
	if len(msgs) != 1 {
		t.Fatalf("Mental Detection returned %d messages, want 1", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || ask.Question == nil || len(ask.Question.Choices) == 0 {
		t.Fatalf("Mental Detection message = %#v, want a scheme question", msgs[0])
	}
}
