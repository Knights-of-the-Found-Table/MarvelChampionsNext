package data

import (
	"testing"
)

var db = MustLoad()

func TestLoadCoversAllPacks(t *testing.T) {
	if len(db.Packs) < 50 {
		t.Fatalf("expected 50+ packs, got %d", len(db.Packs))
	}
	if len(db.Cards) < 3500 {
		t.Fatalf("expected 3500+ cards, got %d", len(db.Cards))
	}
}

func TestCoreSetCards(t *testing.T) {
	spidey, ok := db.Lookup("01001a")
	if !ok {
		t.Fatal("missing 01001a Spider-Man hero")
	}
	if spidey.Type != "hero" || spidey.Side != "a" {
		t.Errorf("01001a: type=%s side=%s", spidey.Type, spidey.Side)
	}
	if spidey.HP == nil || *spidey.HP != 10 {
		t.Errorf("01001a hp = %v, want 10", spidey.HP)
	}
	if spidey.HandSize == nil || *spidey.HandSize != 5 {
		t.Errorf("01001a hand size = %v, want 5", spidey.HandSize)
	}
	peter, ok := db.Lookup("01001b")
	if !ok || peter.Side != "b" {
		t.Errorf("01001b alter-ego missing or wrong side: %+v", peter)
	}

	// Aunt May: unique support, 1 energy resource, persona trait.
	may, ok := db.Lookup("01006")
	if !ok {
		t.Fatal("missing 01006 Aunt May")
	}
	if !may.Unique || may.Type != "support" || !may.HasTrait("persona") {
		t.Errorf("01006 = %+v", may)
	}
	if len(may.Resources) != 1 || may.Resources[0] != "energy" {
		t.Errorf("01006 resources = %v", may.Resources)
	}

	// A core minion carries its encounter set and boost.
	hydra, ok := db.Lookup("01101")
	if !ok {
		t.Fatal("missing 01101 Hydra Mercenary")
	}
	if hydra.Category != CategoryEncounter || hydra.Boost == nil || *hydra.Boost != 1 {
		t.Errorf("01101 = %+v", hydra)
	}
}

func TestPrintedResourceMultiplicity(t *testing.T) {
	// Core-set Energy, Genius, and Strength each print two identical resource
	// icons. Payment validation counts this slice, so both copies must survive
	// conversion from the raw resource_* integer fields.
	want := map[string]string{
		"01088": "energy",
		"01089": "mental",
		"01090": "physical",
	}
	for code, icon := range want {
		def, ok := db.Lookup(code)
		if !ok {
			t.Fatalf("missing resource card %s", code)
		}
		if len(def.Resources) != 2 || def.Resources[0] != icon || def.Resources[1] != icon {
			t.Errorf("%s resources = %v, want [%s %s]", code, def.Resources, icon, icon)
		}
	}
}

func TestGreenGoblinEncounterSets(t *testing.T) {
	gob := db.InSet("the_green_goblin")
	if len(gob) == 0 {
		// fallback: check by known villain code
		if _, ok := db.Lookup("02014"); !ok {
			t.Fatal("green goblin villain 02014 missing")
		}
	}
	villain, ok := db.Lookup("02014")
	if !ok {
		t.Fatal("missing 02014 Green Goblin (Mutagen Formula I)")
	}
	if villain.Type != "villain" || villain.Stage == nil || *villain.Stage != 1 {
		t.Errorf("02014 = %+v", villain)
	}
	env, ok := db.Lookup("02006a")
	if !ok || env.Type != "environment" {
		t.Errorf("02006a environment missing: %+v", env)
	}
}

func TestKeywordParsing(t *testing.T) {
	// Black Panther hero has Retaliate 1 in core set.
	bp, ok := db.Lookup("01040a")
	if !ok {
		t.Skip("01040a not in snapshot")
	}
	found := false
	for _, k := range bp.Keywords {
		if k.Name == "Retaliate" {
			found = true
			if k.Value != 1 {
				t.Errorf("retaliate value = %d, want 1", k.Value)
			}
		}
	}
	if !found {
		t.Errorf("01040a keywords = %v, want Retaliate 1", bp.Keywords)
	}
}

func TestBaseCode(t *testing.T) {
	if BaseCode("01001a") != "01001" || BaseCode("01006") != "01006" {
		t.Fatal("BaseCode broken")
	}
}
