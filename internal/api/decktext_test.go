package api

import (
	"strings"
	"testing"

	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

const sampleDeckText = "Sample Deck\r\n\r\nSpider-Man\r\nPacks: Core Set\r\n\r\nEvents\r\n2x Uppercut (Core Set)\r\n1x Genius (Core Set)\r\n\r\nResources\r\n3x Energy (Core Set)\r\n"

func TestParseDecklistText(t *testing.T) {
	dl, err := parseDecklistText(sampleDeckText)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if dl.Name != "Sample Deck" {
		t.Errorf("name: %q", dl.Name)
	}
	if dl.InvestigatorCode != "01001a" {
		t.Errorf("investigator: %q", dl.InvestigatorCode)
	}
	want := map[string]int{"01054": 2, "01089": 1, "01088": 3}
	if len(dl.Slots) != len(want) {
		t.Fatalf("slots: %v", dl.Slots)
	}
	for code, n := range want {
		if dl.Slots[code] != n {
			t.Errorf("slot %s: got %d want %d", code, dl.Slots[code], n)
		}
	}
}

func TestParseDecklistTextErrors(t *testing.T) {
	cases := []struct {
		name, text, wantErr string
	}{
		{"unknown card", "D\n\nSpider-Man\n\n1x No Such Card (Core Set)\n", "unknown card"},
		{"wrong pack", "D\n\nSpider-Man\n\n1x Backflip (Hulk)\n", "not found in the listed pack"},
		{"no hero", "D\n\nNope\n\n1x Energy (Core Set)\n", "no hero name"},
		{"no cards", "D\n\nSpider-Man\n", "no cards found"},
	}
	for _, tc := range cases {
		_, err := parseDecklistText(tc.text)
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("%s: got %v, want error containing %q", tc.name, err, tc.wantErr)
		}
	}
}
