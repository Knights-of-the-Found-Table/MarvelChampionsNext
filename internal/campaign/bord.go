package campaign

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// Revenge of the Black Order (Karl Resch). Three narrative paths share the
// chain: the group's path pick (recorded at campaign start) routes the
// opening chapter, the encounter decks and the setups of chapters 2-3.
var bordPathIndex = map[string]int{"first": 0, "specops": 1, "assault": 2}

// bordFlagFor maps a Black Order minion code to its log flag.
var bordFlagFor = map[string]string{
	"21085":         "blackDwarf",
	"21086":         "supergiant",
	"21125":         "corvus",
	"21126":         "proxima",
	blackSwanMarker: "blackSwan",
}

func bordPath(st *State) string { return st.Selections["path"] }

func bordSetup(st *State, ctx *engine.CampaignSetup, opts *engine.NewGameOptions) error {
	path := bordPath(st)
	if path == "" {
		return fmt.Errorf("the group has not picked a narrative path yet")
	}
	// The opening chapter is the path's own scenario (the index is set
	// when the path pick resolves).
	if st.Index == 0 {
		opts.ScenarioID = boxScenarioID(st, bordPathIndex[path])
	}
	mark := func(flag string) { st.Flags[flag] = true }
	switch st.Index {
	case 0:
		switch path {
		case "first":
			ctx.ExtraSets = append(ctx.ExtraSets, "black_order")
			ctx.PoolSupports = append(ctx.PoolSupports, metroPD)
			ctx.StartSideScheme = saveShawarma
			mark("blackDwarf")
			mark("supergiant")
			mark("blackSwan")
		case "specops":
			ctx.ExtraSets = append(ctx.ExtraSets, setStreetsMayhem)
			ctx.PlayerSideScheme = "40191a" // Establish Safehouse
			ctx.PreShuffle = append(ctx.PreShuffle, "21181")
		case "assault":
			ctx.ExtraSets = append(ctx.ExtraSets, setBandBadoon, "longshot")
		}
	case 1: // Badoon Bombardment
		ctx.ExtraSets = append(ctx.ExtraSets, setBrotherhoodBd)
		switch path {
		case "first":
			ctx.ExtraSets = append(ctx.ExtraSets, "flight")
			ctx.PoolSupports = append(ctx.PoolSupports, metroPD)
			ctx.StartSideScheme = saveShawarma
			if st.Flags["shawarma"] {
				for i := range st.Players {
					st.Players[i].Deck[shawarmaCard]++
				}
			}
			for _, pl := range st.Players {
				for _, o := range pl.BordObligations {
					pl.Deck[o]++
				}
			}
			bordOrderShuffle(st, ctx)
			if st.IsExpert() {
				for i := range st.Players {
					st.Players[i].Deck["04164"]++ // Medical Emergency
				}
			}
		case "specops":
			ctx.StartEnvironment = rogueVessel
			if st.Flags["safehouse"] {
				ctx.StartEnvironments = append(ctx.StartEnvironments, "40191b")
			}
			ctx.PreShuffle = append(ctx.PreShuffle, "21125", "21126")
			if !st.Flags["breach"] {
				ctx.PreShuffle = append(ctx.PreShuffle, "21181")
			}
			if st.Flags["towerDamaged"] {
				ctx.StartDamageEachPlayer = 3
			}
		case "assault":
			ctx.ExtraSets = append(ctx.ExtraSets, "blue_moon")
			ctx.PoolMinions = append(ctx.PoolMinions, "45140") // Gladiator
			bordLongshot(st, ctx, "ls2")
			bordGear(st, ctx)
		}
	case 2: // Mysteries and Magic
		switch path {
		case "first":
			ctx.ExtraSets = append(ctx.ExtraSets, setMojoSitcom, "telepathy")
			ctx.PoolSupports = append(ctx.PoolSupports, metroPD)
			ctx.StartSideScheme = saveShawarma
			if st.Flags["shawarma"] {
				for i := range st.Players {
					st.Players[i].Deck[shawarmaCard]++
				}
			}
			for _, pl := range st.Players {
				for _, o := range pl.BordObligations {
					pl.Deck[o]++
				}
			}
			bordOrderShuffle(st, ctx)
			ctx.StartEnvironment = mojoMiddle
			if st.IsExpert() {
				for i := range st.Players {
					st.Players[i].Deck["04164"]++
					st.Players[i].Deck["11018"]++
				}
			}
		case "specops":
			ctx.ExtraSets = append(ctx.ExtraSets, setLegionsHel, setMenagerie)
			if st.Flags["safehouse"] {
				ctx.StartEnvironments = append(ctx.StartEnvironments, "40191b")
			}
			if !st.Flags["breach"] {
				ctx.PreShuffle = append(ctx.PreShuffle, "21181")
			}
			ctx.PreShuffle = append(ctx.PreShuffle, "21125", "21126")
			if st.Flags["towerDamaged"] {
				ctx.StartDamageEachPlayer = 3
			}
		case "assault":
			ctx.ExtraSets = append(ctx.ExtraSets, setMenagerie, "symbiotic_strength")
			bordLongshot(st, ctx, "ls3")
			bordGear(st, ctx)
		}
	}
	return nil
}

// bordOrderShuffle shuffles the marked Black Order minions into the deck.
func bordOrderShuffle(st *State, ctx *engine.CampaignSetup) {
	for code, flag := range bordFlagFor {
		if st.Flags[flag] && contains(bordOrderMinions, code) {
			ctx.PreShuffle = append(ctx.PreShuffle, code)
		}
	}
}

// bordLongshot handles the Longshot ally progression (Direct Assault).
func bordLongshot(st *State, ctx *engine.CampaignSetup, flag string) {
	if st.Flags[flag] {
		if ctx.PlayerAllies == nil {
			ctx.PlayerAllies = map[int][]string{}
		}
		ctx.PlayerAllies[0] = append(ctx.PlayerAllies[0], longshotAlly)
	} else {
		ctx.PreShuffle = append(ctx.PreShuffle, longshotAlly)
	}
}

// bordGear replays the recorded Gear Up cards at the cost of their total
// price in threat (supports and upgrades alike ride the support pool;
// documented approximation).
func bordGear(st *State, ctx *engine.CampaignSetup) {
	total := 0
	for _, pl := range st.Players {
		for _, g := range pl.BordGear {
			if def, ok := engine.DB.Lookup(g); ok {
				total += deref(def.Cost, 0)
				if def.Type == "upgrade" {
					addRoleUpgrade(ctx, 0, g)
				} else {
					ctx.PoolSupports = append(ctx.PoolSupports, g)
				}
			}
		}
	}
	ctx.MainSchemeThreat += total
}

func bordVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	// Every Black Order minion in the victory display is unmarked.
	for _, code := range snap.VictoryDisplayCodes {
		if flag, ok := bordFlagFor[code]; ok {
			st.Flags[flag] = false
		}
	}
	path := bordPath(st)
	switch st.Index {
	case 0:
		switch path {
		case "first":
			if !contains(snap.SideSchemeBaseCodes, "21182") {
				st.Flags["shawarma"] = true
			}
		case "specops":
			if contains(snap.EnvironmentCodes, "21100") {
				st.Flags["towerDamaged"] = true
			}
			if contains(snap.EnvironmentCodes, "40191b") {
				st.Flags["safehouse"] = true
			}
			if contains(snap.VictoryDisplayCodes, "21181") {
				st.Flags["breach"] = true
			}
			st.Flags["corvus"] = true
			st.Flags["proxima"] = true
		case "assault":
			if _, ok := snap.AllySlots[longshotAlly]; ok {
				st.Flags["ls2"] = true
			}
			for i := range st.Players {
				if living(i) && len(st.Players[i].BordGear) == 0 {
					st.addPending(i, ChoiceBordGear)
				}
			}
		}
	case 1:
		switch path {
		case "specops":
			if contains(snap.VictoryDisplayCodes, "21181") {
				st.Flags["breach"] = true
			}
			st.Flags["corvus"] = true
			st.Flags["proxima"] = true
		case "assault":
			if _, ok := snap.AllySlots[longshotAlly]; ok {
				st.Flags["ls3"] = true
			}
		}
	case 2:
		// Ebony Maw: the Black Order's revenge ends.
	}
	// Advance past the two openings the path did not use.
	if st.Index <= 2 {
		st.Index = 3
		st.Status = "interlude"
	} else {
		st.Advance()
	}
	st.recordExpertHP(snap, living)
}

// ApplyBordChoice resolves the path pick and the Direct Assault Gear Up.
func ApplyBordChoice(st *State, slot int, kind, cardCode string) error {
	switch kind {
	case ChoiceBordPath:
		if _, ok := bordPathIndex[cardCode]; !ok {
			return fmt.Errorf("not a path: %q", cardCode)
		}
		st.Selections["path"] = cardCode
		// The chain jumps to the path's own opening chapter.
		st.Index = bordPathIndex[cardCode]
	case ChoiceBordGear:
		pl := st.Slot(slot)
		if pl == nil {
			return fmt.Errorf("unknown slot %d", slot)
		}
		def, ok := engine.DB.Lookup(cardCode)
		if !ok || (def.Type != "support" && def.Type != "upgrade") {
			return fmt.Errorf("not a support or upgrade: %q", cardCode)
		}
		if deref(def.Cost, 0) > 2 {
			return fmt.Errorf("cost exceeds 2: %q", cardCode)
		}
		pl.BordGear = appendUnique(pl.BordGear, cardCode)
	default:
		return fmt.Errorf("unknown choice kind %q", kind)
	}
	st.ResolvePending(slot, kind)
	return nil
}

// boxScenarioID returns the chain id at the given index.
func boxScenarioID(st *State, index int) string {
	box := st.BoxDef()
	if box == nil || index < 0 || index >= len(box.Scenarios) {
		return ""
	}
	return box.Scenarios[index].ID
}
