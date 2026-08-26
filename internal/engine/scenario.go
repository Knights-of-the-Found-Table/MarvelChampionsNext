package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// ScenarioDef describes a playable scenario. Scenarios are registered by the
// cards packages; the scenario id is its first main scheme code (matching the
// reference implementation convention).
type ScenarioDef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// VillainBases are base codes of the scenario's villains (stages are
	// derived from the card data by stage number).
	VillainBases []string `json:"villainBases"`
	// MainSchemeStages lists main scheme card codes per stage (most
	// scenarios have one or two stages).
	MainSchemeStages []string `json:"mainSchemeStages"`
	// ExtraSets are encounter set codes (marvelcdb card_set_code) added to
	// the villain's own set + Standard.
	ExtraSets []string `json:"extraSets"`
	// Optional per-scenario setup tweaks.
	Setup func(g *Game) []Message `json:"-"`
	// OnMainSchemeDefeated runs when the main scheme is defeated (threat
	// removed to 0); e.g. Klaw advances villain and scheme stage.
	OnMainSchemeDefeated func(g *Game, s *MainScheme) []Message `json:"-"`
	// OnMainSchemeMaxed overrides the default loss when the main scheme
	// completes; e.g. stage-1 schemes advance instead of losing.
	OnMainSchemeMaxed func(g *Game, s *MainScheme) []Message `json:"-"`
	// VillainUndamageable marks villain stages that cannot be damaged
	// (e.g. Rhino stage I); keyed by stage number.
	VillainUndamageable map[int]bool `json:"villainUndamageable,omitempty"`
	// OnVillainDefeated overrides the default final-stage win when set
	// (Wrecking Crew wins only when every villain is defeated).
	OnVillainDefeated func(g *Game, v *Villain) []Message
}

var scenarioRegistry = map[string]*ScenarioDef{}

// RegisterScenario installs a scenario; called from card package inits.
func RegisterScenario(def *ScenarioDef) {
	scenarioRegistry[def.ID] = def
}

// LookupScenario finds a scenario by id.
func LookupScenario(id string) (*ScenarioDef, bool) {
	def, ok := scenarioRegistry[id]
	return def, ok
}

// Scenarios returns all registered scenarios sorted by id.
func Scenarios() []*ScenarioDef {
	out := make([]*ScenarioDef, 0, len(scenarioRegistry))
	for _, def := range scenarioRegistry {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// VillainStageCodes derives the stage codes of a villain base code from the
// card data. Two conventions coexist:
//
//   - Double-sided villains (base+"b" is a villain card, e.g. Green Goblin
//     02001b): the chain is exactly the sides of that one code, a before b
//     (Norman Osborn → Green Goblin). Later stages are scenario-driven.
//   - Single-sided villains: stage II/III cards are separate codes sharing
//     the printed English name within the set (Rhino 01094/01095/01096;
//     Civil War leaders I–IV). Lettered markers ("A1"/"B1") are alternate
//     scenario variants and never chain; alternate a/c forms of one stage
//     collapse to the primary's side.
func VillainStageCodes(base string) []string {
	singleSided := true
	if def, ok := DB.Lookup(base + "b"); ok && def.Type == "villain" {
		base = base + "b"
		singleSided = false
	}
	primary, ok := DB.Lookup(base)
	if !ok {
		return nil
	}
	var family []*data.CardDef
	for _, c := range DB.InSet(primary.CardSet) {
		if (c.Type != "villain" && c.Type != "leader") || c.CardSet != primary.CardSet {
			continue
		}
		if singleSided && !sameVillainStageFamily(c, primary, base) {
			continue
		}
		if !singleSided && data.BaseCode(c.Code) != data.BaseCode(base) {
			continue
		}
		family = append(family, c)
	}
	if !singleSided {
		// Persona sides of one code, a/b order.
		sort.Slice(family, func(i, j int) bool {
			return sideOrder(family[i].Code) < sideOrder(family[j].Code)
		})
		codes := make([]string, len(family))
		for i, c := range family {
			codes[i] = c.Code
		}
		return codes
	}
	// One progression entry per stage value: alternate forms of the same
	// stage (En Sabah Nur's a/c Apocalypse forms) collapse; keep the
	// primary's side, falling back to a stage's only card.
	primarySide := sideOrder(base)
	byStage := map[string]*data.CardDef{}
	var order []string
	for _, c := range family {
		key := c.StageLabel
		if c.Stage != nil {
			key = fmt.Sprintf("%03d", *c.Stage)
		}
		cur, seen := byStage[key]
		if !seen {
			byStage[key] = c
			order = append(order, key)
			continue
		}
		if sideOrder(c.Code) == primarySide && sideOrder(cur.Code) != primarySide {
			byStage[key] = c
		}
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := byStage[order[i]], byStage[order[j]]
		if a.Stage != nil && b.Stage != nil {
			return *a.Stage < *b.Stage
		}
		return a.StageLabel < b.StageLabel
	})
	codes := make([]string, 0, len(order))
	for _, key := range order {
		codes = append(codes, byStage[key].Code)
	}
	return codes
}

// sameVillainStageFamily reports whether candidate c is a progression stage
// of the primary villain card.
func sameVillainStageFamily(c, primary *data.CardDef, base string) bool {
	// Sides of one code chain (persona flips: 45081a/b War).
	if data.BaseCode(c.Code) == data.BaseCode(base) {
		return true
	}
	if c.EName != primary.EName {
		return false
	}
	// Same name: roman/numeric markers are progression stages ("I"→"II"→
	// "III"); lettered markers (A1/B1) or marker-less cards are variants.
	if c.Stage == nil || primary.Stage == nil {
		return false
	}
	// Alternate a/c forms of one stage do not chain; keep the primary side.
	return sideOrder(c.Code) == sideOrder(base)
}

func sideOrder(code string) string {
	if data.BaseCode(code) != code {
		return code[len(code)-1:]
	}
	return ""
}

// EncounterSetCodes returns the card codes (with quantities) making up an
// encounter set, skipping villain/main-scheme entries which are handled by
// the scenario definition.
func EncounterSetCards(set string) []Card {
	var out []Card
	for _, def := range DB.InSet(set) {
		switch def.Type {
		case "villain", "main_scheme":
			continue
		}
		qty := def.Quantity
		if qty <= 0 {
			qty = 1
		}
		// Double-sided cards only ship once.
		if data.BaseCode(def.Code) != def.Code {
			// include the a-side once; the b-side is display-only
			if def.Side != "a" {
				continue
			}
			qty = 1
		}
		for i := 0; i < qty; i++ {
			out = append(out, Card{Code: def.Code})
		}
	}
	return out
}

func (s *ScenarioDef) String() string {
	return fmt.Sprintf("%s [%s] villains=%v sets=%v", s.Name, s.ID, s.VillainBases, s.ExtraSets)
}

// gatherEncounterDeck assembles the encounter deck from the scenario's sets.
func (s *ScenarioDef) gatherEncounterDeck() CardList {
	var deck CardList
	seen := map[string]bool{}
	addSet := func(set string) {
		if set == "" || seen[set] {
			return
		}
		seen[set] = true
		deck = append(deck, EncounterSetCards(set)...)
	}
	for _, base := range s.VillainBases {
		if def, ok := DB.Lookup(base + "b"); ok {
			addSet(def.CardSet)
		} else if def, ok := DB.Lookup(base); ok {
			addSet(def.CardSet)
		}
	}
	for _, set := range s.ExtraSets {
		addSet(set)
	}
	return deck
}

// nemesisSetCode maps a hero base code to its nemesis encounter set code.
func nemesisSetCode(heroBase string) string {
	def, ok := DB.Lookup(heroBase + "a")
	if !ok {
		return ""
	}
	for _, c := range DB.InSet(def.CardSet) {
		if c.Type == "minion" && strings.Contains(strings.ToLower(c.Name), "nemesis") {
			return c.CardSet
		}
	}
	// marvelcdb convention: hero set contains a linked nemesis subset named
	// "<hero> nemesis"; fall back to the hero set itself minus player cards.
	return def.CardSet
}
