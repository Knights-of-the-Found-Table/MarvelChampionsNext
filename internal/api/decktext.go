package api

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// Plain-text decklist import, matching the format marvelcdb offers for
// download next to every deck(list):
//
//	The defense rests, your honor
//
//	Daredevil
//	Packs: From Core Set to Fear No Evil
//
//	Upgrades
//	1x Deft Focus (The Galaxy's Most Wanted)
//	...
//
// Line 1 is the deck name, a following line naming a hero is the
// investigator, "Nx Name (Pack)" lines are cards. Everything else — blank
// lines, the "Packs:" hint and the per-type section headers — is skipped.

const deckTextMaxLen = 1 << 16

var (
	deckCardLineRE = regexp.MustCompile(`^(\d+)x (.+)$`)
	deckPackParens = regexp.MustCompile(` \(([^()]+)\)$`)

	deckTextOnce sync.Once
	deckPackName map[string]string            // pack display name -> pack code
	deckByName   map[string][]*engineCardLite // card name -> candidates
	deckHeroName map[string]string            // hero name -> hero-side code
)

// engineCardLite is the sliver of CardDef the resolver needs.
type engineCardLite = struct {
	Code     string
	PackCode string
}

func buildDeckTextIndexes() {
	deckPackName = map[string]string{}
	for _, p := range engine.DB.Packs {
		deckPackName[p.Name] = p.Code
	}
	deckByName = map[string][]*engineCardLite{}
	deckHeroName = map[string]string{}
	for _, def := range engine.DB.All() {
		switch {
		case def.Type == "hero" && def.Side == "a":
			// Heroes reusing a name (core Spider-Man vs. later prints) resolve
			// to the earliest printing; the text export cannot distinguish.
			if _, ok := deckHeroName[def.Name]; !ok {
				deckHeroName[def.Name] = def.Code
			}
		case def.Category == "player" && def.Code != "":
			lite := &engineCardLite{Code: def.Code, PackCode: def.PackCode}
			deckByName[def.Name] = append(deckByName[def.Name], lite)
			if def.Subname != "" {
				composite := def.Name + " (" + def.Subname + ")"
				deckByName[composite] = append(deckByName[composite], lite)
			}
		}
	}
}

type parsedDecklist struct {
	Name             string
	InvestigatorCode string
	Slots            map[string]int
}

func parseDecklistText(text string) (*parsedDecklist, error) {
	if len(text) > deckTextMaxLen {
		return nil, fmt.Errorf("decklist text too large (%d bytes)", len(text))
	}
	deckTextOnce.Do(buildDeckTextIndexes)

	dl := &parsedDecklist{Slots: map[string]int{}}
	sawCard := false
	for i, raw := range strings.Split(strings.TrimPrefix(text, "\ufeff"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "Packs:") {
			continue
		}
		m := deckCardLineRE.FindStringSubmatch(line)
		if m == nil {
			if !sawCard && dl.Name == "" {
				dl.Name = line
			} else if !sawCard && dl.InvestigatorCode == "" {
				dl.InvestigatorCode = deckHeroName[line]
			}
			// After the first card line anything unnumbered is a section
			// header ("Upgrades", "Events", ...): skip it.
			continue
		}
		count, err := strconv.Atoi(m[1])
		if err != nil || count <= 0 {
			return nil, fmt.Errorf("line %d: bad quantity", i+1)
		}
		name, pack := m[2], ""
		if pm := deckPackParens.FindStringSubmatch(name); pm != nil {
			if code, ok := deckPackName[pm[1]]; ok {
				pack, name = code, strings.TrimSpace(name[:len(name)-len(pm[0])])
			}
			// Parens that are not a known pack stay part of the name (cards
			// exported as "Name (Subname)").
		}
		code, err := resolveDeckCard(name, pack)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		dl.Slots[code] += count
		sawCard = true
	}
	if !sawCard {
		return nil, fmt.Errorf("no cards found — not a marvelcdb text export?")
	}
	if dl.InvestigatorCode == "" {
		return nil, fmt.Errorf("no hero name found before the first card line")
	}
	if dl.Name == "" {
		dl.Name = "Imported deck"
	}
	return dl, nil
}

func resolveDeckCard(name, pack string) (string, error) {
	cands := deckByName[name]
	if len(cands) == 0 {
		return "", fmt.Errorf("unknown card %q", name)
	}
	if pack != "" {
		var inPack []*engineCardLite
		for _, c := range cands {
			if c.PackCode == pack {
				inPack = append(inPack, c)
			}
		}
		if len(inPack) == 0 {
			return "", fmt.Errorf("card %q not found in the listed pack", name)
		}
		cands = inPack
	}
	if len(cands) > 1 && cands[0].Code != cands[1].Code {
		return "", fmt.Errorf("ambiguous card %q — specify its pack in parentheses", name)
	}
	return cands[0].Code, nil
}
