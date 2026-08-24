package silk_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/silk"
)

// TestSilkGapCardsRegistered spot-checks the 5xxxx completions.
func TestSilkGapCardsRegistered(t *testing.T) {
	for _, code := range []string{
		"52013", "52015", "52016", "52020", "52021", "52022", "52023",
		"52032", "52033", "52034", "52035", "52036", "52038",
	} {
		if !engine.Implemented(code) {
			t.Errorf("card %s should be implemented", code)
		}
	}
}
