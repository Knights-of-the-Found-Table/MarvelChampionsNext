package falcon_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/falcon"
)

// TestFalconGapCardsRegistered spot-checks the 5xxxx completions.
func TestFalconGapCardsRegistered(t *testing.T) {
	for _, code := range []string{
		"53014", "53015", "53016", "53017", "53018", "53019", "53020",
		"53021", "53022", "53023", "53034", "53035", "53037", "53038", "53042",
	} {
		if !engine.Implemented(code) {
			t.Errorf("card %s should be implemented", code)
		}
	}
}
