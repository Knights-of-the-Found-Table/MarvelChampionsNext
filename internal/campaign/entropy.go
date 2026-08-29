package campaign

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// Entropic Ascension (Karl Resch). A group reputation track (shorter than
// the Sinister Motives one, same conditions) pays out upgrades and
// recurring setup curses; the recorded Crime Wave lines assemble the
// finale's modular sets.

// entTrack follows the printed node order (the left and right columns
// interleave on one 1-20 axis; the alignment of a few boxes is ambiguous
// in print and fixed here — documented approximation, same convention as
// the SM track).
var entTrack = []smNode{
	{1, 'i', "tech"},
	{3, 'i', "soeRec"},
	{3, 's', "soe1"},
	{5, 's', "mulligan"},
	{6, 's', "threat"},
	{8, 'i', "aspect"},
	{10, 's', "minions"},
	{12, 's', "enhanced"},
	{14, 'i', "plan"},
	{16, 's', "planSearch"},
	{18, 's', "helicarrier"},
	{20, 'i', "soeRec"},
	{20, 's', "soe2"},
}

// entFinalSet maps each recorded Crime Wave line to the modular set it
// sets aside for The Hood's finale.
var entFinalSet = map[string]string{
	"Charge Complete":         setZzzax,
	"Superior Foes":           setSinisterAsslt,
	"Fears & Doubts":          setWhispers,
	"Brute Force":             setWreckingMod,
	"Raft Weapons":            setUnderAttack,
	"Contraband":              setRansacked,
	"On A Roll":               setArmadillo,
	"Loved Ones in Peril":     setDownToEarth,
	"City Cries for a Savior": setCityInChaos,
	"Oscorp R&D Raided":       setOsbornTech,
	"Oscorp Biolab Raided":    "symbiotic_strength",
	"Criminal Empire":         setSinisterSynd,
	"Unleash the Beast!":      setBeastyBoys,
}

// entSoeOptions lists the State of Emergency side schemes (Hood).
var entSoeOptions = []string{"24055", "24056", "24057", "24058"}

// entCommon applies the setup every chapter shares.
func entCommon(st *State, ctx *engine.CampaignSetup) {
	if st.IsExpert() {
		ctx.StartEnvironment = "27174b"
	} else {
		ctx.StartEnvironment = smPublicOutcry
	}
	ctx.PreShuffle = append(ctx.PreShuffle, smSmearCampaign)
	if remaining := withoutAll(smCommunity, st.SMCommunity); len(remaining) > 0 {
		ctx.PreShuffle = append(ctx.PreShuffle, remaining[randInt(len(remaining))])
	}
	// Reputation-track setup instructions, top to bottom.
	for _, node := range entTrack {
		if node.kind != 's' || node.n > st.Counters["marked"] {
			continue
		}
		switch node.key {
		case "soe1", "soe2":
			if code := st.Selections[node.key]; code != "" {
				ctx.PreShuffle = append(ctx.PreShuffle, code)
			}
		case "threat":
			ctx.MainSchemeThreat++
		case "minions":
			ctx.MinionEngageEachPlayer = true
		case "enhanced":
			// The ENHANCED printings are absent from the data snapshot;
			// the basic upgrade stays in play (documented gap).
		case "planSearch":
			for i := range st.Players {
				st.Players[i].SetupHand = st.Players[i].SMPlanning
			}
		case "helicarrier":
			ctx.PoolUpgrades = append(ctx.PoolUpgrades, smHelicarrier)
		}
		// "mulligan" is not modelable at setup (documented gap).
	}
	// Recorded tech upgrades join their owners.
	for i := range st.Players {
		if code := st.Players[i].SMShieldTech; code != "" {
			addRoleUpgrade(ctx, i, code)
		}
	}
}

func entSetup(st *State, ctx *engine.CampaignSetup, opts *engine.NewGameOptions) {
	entCommon(st, ctx)
	switch st.Index {
	case 0:
		// The group's A/B pick was made before the chapter started.
		if st.Selections["path1"] == "a" {
			ctx.ExtraSets = append(ctx.ExtraSets, setRunningIntf)
		} else {
			ctx.ExtraSets = append(ctx.ExtraSets, setPowerDrain)
		}
	case 1, 2:
		if st.Selections["path2"] == "a" {
			if st.Index == 1 {
				ctx.ExtraSets = append(ctx.ExtraSets, setRansacked)
			} else {
				ctx.ExtraSets = append(ctx.ExtraSets, setCityInChaos)
			}
		} else {
			if st.Index == 1 {
				ctx.ExtraSets = append(ctx.ExtraSets, setUnderAttack)
			} else {
				ctx.ExtraSets = append(ctx.ExtraSets, setDownToEarth)
			}
		}
		if st.IsExpert() {
			ctx.DeckShuffleEncounter = 2
		}
	case 3:
		if st.Selections["path3"] == "a" {
			ctx.ExtraSets = append(ctx.ExtraSets, setCrossfire)
		} else {
			ctx.ExtraSets = append(ctx.ExtraSets, setIronSpiderSix)
		}
		if st.IsExpert() {
			ctx.VillainBoostFacedown = true
		}
	case 4:
		// The Hood: the Crime Wave lines set the finale's modulars.
		for i := 1; i <= 7; i++ {
			if set, ok := entFinalSet[st.Selections[fmt.Sprintf("cw%d", i)]]; ok {
				ctx.ExtraSets = append(ctx.ExtraSets, set)
			}
		}
		if st.IsExpert() {
			ctx.MainSchemeThreat += len(st.Players)
		}
	}
	// Expert heal cost: 1 / 1 / 2 / 2 / 3 facedown encounter cards.
	if st.IsExpert() && st.Index >= 1 {
		cost := []int{1, 1, 2, 2, 3}[st.Index]
		for i := range st.Players {
			if st.Players[i].HealNext {
				for k := 0; k < cost; k++ {
					ctx.DealEncounter = append(ctx.DealEncounter, i)
				}
			}
		}
	}
}

func entVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	rep := smReputation(st, snap)
	entMark(st, rep)
	for _, code := range snap.VictoryDisplayCodes {
		if contains(smCommunity, code) {
			st.SMCommunity = appendUnique(st.SMCommunity, code)
		}
	}
	switch st.Index {
	case 0:
		// Crime Wave lines 1-2 and the route to Rhino or Mysterio.
		if st.Selections["path1"] == "a" {
			st.Selections["cw1"] = "Charge Complete"
			st.Selections["cw2"] = "Superior Foes"
			st.Index = 1
		} else {
			st.Selections["cw1"] = "Fears & Doubts"
			st.Selections["cw2"] = "Brute Force"
			st.Index = 2
		}
		st.addPending(0, ChoiceEnPath2)
		st.Status = "interlude"
		st.recordExpertHP(snap, living)
		return
	case 1, 2:
		// Crime Wave lines 3-5; both branches lead to Klaw.
		if st.Index == 1 { // Rhino
			if st.Selections["path2"] == "a" { // Police Transport
				st.Selections["cw3"], st.Selections["cw4"], st.Selections["cw5"] =
					"Raft Weapons", "Loved Ones in Peril", "On A Roll"
			} else { // Raft Transport
				st.Selections["cw3"], st.Selections["cw4"], st.Selections["cw5"] =
					"Contraband", "Loved Ones in Peril", "On A Roll"
			}
		} else { // Mysterio
			st.Selections["cw3"], st.Selections["cw4"] = "Raft Weapons", "Contraband"
			if st.Selections["path2"] == "a" { // Strangers & Acquaintances
				st.Selections["cw5"] = "Loved Ones in Peril"
			} else { // Friends & Family
				st.Selections["cw5"] = "City Cries for a Savior"
			}
		}
		st.Index = 3
		st.addPending(0, ChoiceEnPath3)
		st.Status = "interlude"
		st.recordExpertHP(snap, living)
		return
	case 3:
		if st.Selections["path3"] == "a" {
			st.Selections["cw6"] = "Oscorp Biolab Raided"
		} else {
			st.Selections["cw6"] = "Oscorp R&D Raided"
		}
		if rep >= 15 {
			st.Selections["cw7"] = "Criminal Empire"
		} else {
			st.Selections["cw7"] = "Unleash the Beast!"
		}
		st.Index = 4
		st.Status = "interlude"
		st.recordExpertHP(snap, living)
		return
	case 4:
		// The Hood defeated: the campaign is won.
	}
	st.recordExpertHP(snap, living)
	st.Advance()
}

// entMark applies newly reached reputation nodes (same node mechanics as
// the SM track).
func entMark(st *State, rep int) {
	for _, node := range entTrack {
		if node.n > rep || node.n <= st.Counters["marked"] || node.kind != 'i' {
			continue
		}
		st.Counters["marked"] = node.n
		switch node.key {
		case "tech":
			pool := append([]string{}, smTech...)
			for i := range st.Players {
				pl := &st.Players[i]
				if pl.SMShieldTech != "" {
					continue
				}
				picks := []string{}
				for len(picks) < 3 && len(pool) > 0 {
					idx := randInt(len(pool))
					picks = append(picks, pool[idx])
					pool = append(pool[:idx], pool[idx+1:]...)
				}
				pl.SMTechOffer = picks
				st.addPending(i, ChoiceSMTech)
			}
		case "aspect":
			for i := range st.Players {
				if st.Players[i].SMAspect == "" {
					st.addPending(i, ChoiceSMAspect)
				}
			}
		case "plan":
			for i := range st.Players {
				if st.Players[i].SMPlanning == "" {
					st.addPending(i, ChoiceSMPlan)
				}
			}
		case "soeRec":
			entRecordSoE(st, "soe1")
			if st.Counters["marked"] >= 20 {
				entRecordSoE(st, "soe2")
			}
		}
	}
	st.Counters["reputation"] = rep
}

// entRecordSoE draws a random State of Emergency side scheme into the log
// (track node 1 / node 20).
func entRecordSoE(st *State, key string) {
	if st.Selections[key] != "" {
		return
	}
	st.Selections[key] = entSoeOptions[randInt(len(entSoeOptions))]
}

// ApplyEnChoice resolves the Entropic Ascension A/B group picks.
func ApplyEnChoice(st *State, slot int, kind, cardCode string) error {
	keys := map[string]string{
		ChoiceEnPath1: "path1",
		ChoiceEnPath2: "path2",
		ChoiceEnPath3: "path3",
	}
	key, ok := keys[kind]
	if !ok {
		return fmt.Errorf("unknown choice kind %q", kind)
	}
	if cardCode != "a" && cardCode != "b" {
		return fmt.Errorf("option must be a or b")
	}
	st.Selections[key] = cardCode
	st.ResolvePending(slot, kind)
	return nil
}
