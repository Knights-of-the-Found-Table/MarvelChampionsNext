package wonderman_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/wonderman"
)

// TestWonderManGapCardsRegistered spot-checks the 5xxxx completions.
func TestWonderManGapCardsRegistered(t *testing.T) {
	for _, code := range []string{
		"58012", "58013", "58014", "58015", "58018", "58021", "58024", "58034",
	} {
		if !engine.Implemented(code) {
			t.Errorf("card %s should be implemented", code)
		}
	}
}
