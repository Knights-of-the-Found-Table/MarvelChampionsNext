package data

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed packs/*.json
var packsFS embed.FS

// Card categories.
const (
	CategoryPlayer    = "player"
	CategoryEncounter = "encounter"
	CategoryOther     = "other"
)

// Pack describes one product in the marvelcdb database.
type Pack struct {
	Name      string `json:"name"`
	Code      string `json:"code"`
	Position  int    `json:"position"`
	Available string `json:"available"`
	Known     int    `json:"known"`
	Total     int    `json:"total"`
}

// Database is the immutable, read-only card database built from the snapshot.
type Database struct {
	Packs  []Pack
	Cards  map[string]*CardDef
	byPack map[string][]*CardDef
	bySet  map[string][]*CardDef
	sorted []*CardDef
}

// add registers a definition unless the code was already seen; reprints in
// later packs keep their original printing because packs are read in
// packs.json order.
func add(db *Database, def *CardDef) {
	if _, exists := db.Cards[def.Code]; exists {
		return
	}
	db.Cards[def.Code] = def
	db.byPack[def.PackCode] = append(db.byPack[def.PackCode], def)
	if def.CardSet != "" {
		db.bySet[def.CardSet] = append(db.bySet[def.CardSet], def)
	}
}

// Load parses the embedded snapshot. It is safe to call multiple times but is
// cheap enough to call once and share.
func Load() (*Database, error) {
	db := &Database{
		Cards:  map[string]*CardDef{},
		byPack: map[string][]*CardDef{},
		bySet:  map[string][]*CardDef{},
	}

	packsRaw, err := packsFS.ReadFile("packs/packs.json")
	if err != nil {
		return nil, fmt.Errorf("read packs.json: %w", err)
	}
	if err := json.Unmarshal(packsRaw, &db.Packs); err != nil {
		return nil, fmt.Errorf("decode packs.json: %w", err)
	}
	sort.Slice(db.Packs, func(i, j int) bool { return db.Packs[i].Position < db.Packs[j].Position })

	entries, err := packsFS.ReadDir("packs")
	if err != nil {
		return nil, fmt.Errorf("read packs dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || e.Name() == "packs.json" {
			continue
		}
		rawCards, err := packsFS.ReadFile("packs/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		var raws []rawCard
		if err := json.Unmarshal(rawCards, &raws); err != nil {
			return nil, fmt.Errorf("decode %s: %w", e.Name(), err)
		}
		for _, raw := range raws {
			if raw.Code == "" {
				continue
			}
			def := &CardDef{}
			normalize(def, raw)
			add(db, def)
			// Double-sided back faces (alter egos, villain personas,
			// environment flips) only exist as a nested linked_card object.
			if raw.LinkedCard != nil && raw.LinkedCard.Code != "" {
				back := &CardDef{}
				normalize(back, *raw.LinkedCard)
				add(db, back)
			}
		}
	}

	db.sorted = make([]*CardDef, 0, len(db.Cards))
	for _, def := range db.Cards {
		db.sorted = append(db.sorted, def)
	}
	sort.Slice(db.sorted, func(i, j int) bool { return db.sorted[i].Code < db.sorted[j].Code })
	return db, nil
}

// MustLoad is Load for package-level initialization.
func MustLoad() *Database {
	db, err := Load()
	if err != nil {
		panic(err)
	}
	return db
}

// Lookup returns the card definition for an OCTGN-style card code
// (e.g. "01001a").
func (db *Database) Lookup(code string) (*CardDef, bool) {
	def, ok := db.Cards[code]
	return def, ok
}

// MustLookup errors loudly for unknown codes; used by the engine where a
// missing definition is a programming error.
func (db *Database) MustLookup(code string) *CardDef {
	def, ok := db.Cards[code]
	if !ok {
		panic(fmt.Sprintf("data: unknown card code %q", code))
	}
	return def
}

// All returns every card sorted by code.
func (db *Database) All() []*CardDef { return db.sorted }

// InSet returns all cards belonging to a card_set_code (encounter set or hero
// set), sorted by code.
func (db *Database) InSet(set string) []*CardDef {
	out := append([]*CardDef(nil), db.bySet[set]...)
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// BaseCode strips an a/b/c side suffix ("01001a" -> "01001").
func BaseCode(code string) string {
	if len(code) == 6 && hasSideSuffix(code) {
		return code[:5]
	}
	return code
}

// HeroSideCode returns the hero ("...a") code for a base hero code.
func HeroSideCode(base string) string { return base + "a" }

// AlterEgoSideCode returns the alter-ego ("...b") code for a base hero code.
func AlterEgoSideCode(base string) string { return base + "b" }
