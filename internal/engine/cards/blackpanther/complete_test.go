package blackpanther_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/blackpanther"
)

// TestBPGapCardsRegistered spot-checks the 5xxxx completions.
func TestBPGapCardsRegistered(t *testing.T) {
	for _, code := range []string{
		"51006", "51014", "51015", "51018", "51020", "51025", "51037", "51039", "51042",
	} {
		if !engine.Implemented(code) {
			t.Errorf("card %s should be implemented", code)
		}
	}
}
