package winter_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/winter"
)

// TestWinterGapCardsRegistered spot-checks the 5xxxx completions.
func TestWinterGapCardsRegistered(t *testing.T) {
	for _, code := range []string{
		"54012", "54013", "54016", "54019", "54021", "54023", "54034", "54037",
	} {
		if !engine.Implemented(code) {
			t.Errorf("card %s should be implemented", code)
		}
	}
}
