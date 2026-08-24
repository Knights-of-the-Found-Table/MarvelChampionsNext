package hercules_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/hercules"
)

// TestHerculesGapCardsRegistered spot-checks the 5xxxx completions.
func TestHerculesGapCardsRegistered(t *testing.T) {
	for _, code := range []string{
		"59018", "59019", "59020", "59022", "59026", "59032", "59041", "59045",
	} {
		if !engine.Implemented(code) {
			t.Errorf("card %s should be implemented", code)
		}
	}
}
