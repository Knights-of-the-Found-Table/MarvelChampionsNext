package engine_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
	// register all shipped content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/angel"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/ant"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/blackwidow"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/captainamerica"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/civilwar"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/daredevil"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/doctorstrange"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/drax"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/echo"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/extras"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/galaxysmostwanted"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/gamora"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/goblinfooblin"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/hulk"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/msmarvel"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/mutantgenesis"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nebula"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nextevolution"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nova"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/onceandfuturekang"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/quicksilver"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/spiderwoman"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/starlord"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/thor"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/valkyrie"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/venom"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/vision"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/warmachine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/wasp"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/wreckingcrew"
)

// TestSurvey prints the implementation gap for cards with base codes <=
// 29999. Textless cards are fully served by the generic behavior; the gap
// is cards with ability text and no registered behavior.
func TestSurvey(t *testing.T) {
	type row struct {
		pack, code, name, typ, text string
	}
	var gapText, gapNoText []row
	var okText, okNoText int
	byPack := map[string]int{}
	for _, def := range engine.DB.All() {
		code := def.Code
		base := data.BaseCode(code)
		if base > "29999" || len(base) != 5 {
			continue
		}
		hasText := strings.TrimSpace(def.Text) != ""
		impl := engine.Implemented(base)
		if impl {
			if hasText {
				okText++
			} else {
				okNoText++
			}
			continue
		}
		if hasText {
			gapText = append(gapText, row{def.PackCode, base, def.Name, def.Type, clip(def.Text)})
			byPack[def.PackCode]++
		} else {
			gapNoText = append(gapNoText, row{def.PackCode, base, def.Name, def.Type, ""})
		}
	}
	sort.Slice(gapText, func(i, j int) bool { return gapText[i].code < gapText[j].code })
	fmt.Printf("implemented: text=%d notext=%d | gap: text=%d notext=%d\n",
		okText, okNoText, len(gapText), len(gapNoText))
	fmt.Println("== gap with text by pack ==")
	var packs []string
	for p := range byPack {
		packs = append(packs, p)
	}
	sort.Strings(packs)
	for _, p := range packs {
		fmt.Printf("%-12s %d\n", p, byPack[p])
	}
	for _, r := range gapText {
		fmt.Printf("GAP %-10s %s %-18s %-16s %s\n", r.pack, r.code, r.name, r.typ, r.text)
	}
}

func clip(s string) string {
	out := strings.Join(strings.Fields(s), " ")
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}
