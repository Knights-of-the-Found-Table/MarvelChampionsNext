package spectrum_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/spectrum"
)

func spectrumDeck() map[string]int {
	return map[string]int{"21005": 1, "21006": 2, "21007": 3, "21008": 3, "21009": 3, "21010": 3}
}

func TestSpectrumSetupAndEnergyTransformationContract(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{Seed: 21, ScenarioID: "01097", Players: []engine.PlayerSpec{{Name: "Spectrum", HeroBase: "21001", Deck: spectrumDeck()}}})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	p := g.Players[0]
	b := engine.LookupBehavior("21001")
	if b == nil || b.HeroSetup == nil || b.React == nil {
		t.Fatal("Spectrum must expose setup and React hooks")
	}
	setup := b.HeroSetup(g, p)
	if len(p.Upgrades) != 3 || len(setup) != 1 {
		t.Fatalf("setup created %d forms and %d messages, want 3 and 1", len(p.Upgrades), len(setup))
	}
	ask, ok := setup[0].(engine.AskQuestion)
	if !ok || ask.Question == nil || len(ask.Question.Choices) != 3 {
		t.Fatalf("setup = %#v, want three-form question", setup)
	}
	p.Side = engine.SideHero
	b.React(g, p, engine.AddEntityCounter{ID: p.ID, N: -21002})
	if got := p.AttackStat(g); got != 3 {
		t.Fatalf("Gamma ATK = %d, want printed 1 + 2", got)
	}
	b.React(g, p, engine.AddEntityCounter{ID: p.ID, N: -21003})
	if got := p.ThwartStat(g); got != 3 {
		t.Fatalf("Photon THW = %d, want printed 1 + 2", got)
	}
	if got := p.AttackStat(g); got != 1 {
		t.Fatalf("Gamma remained faceup after Photon switch: ATK %d", got)
	}
}
