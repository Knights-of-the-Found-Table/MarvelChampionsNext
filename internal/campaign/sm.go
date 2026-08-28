package campaign

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// The Sinister Motives campaign: a shared reputation track whose marked
// nodes grant boons (immediate) and recurring setup riders. Node numbers
// follow the printed track; a few box-to-node alignments are ambiguous in
// print and are resolved here with a fixed mapping (documented
// approximation).
var (
	smPublicOutcry   = "27174a" // environment (27174b = expert side)
	smSmearCampaign  = "27175"  // treachery
	smCommunity      = []string{"27176", "27177", "27178", "27179", "27180"}
	smSnitches       = "27181" // attachment, Victory -1
	smVenomAlly      = "27190"
	smSymbioteSuit   = "27191"
	smHelicarrier    = "01092"
	smLightAtEnd     = "27102a"
	smCityStreets    = "27065"
	smTech           = []string{"27182a", "27183a", "27184a", "27185a", "27186a", "27187a", "27188a", "27189a"}
	smOsborn         = []string{"27147", "27148", "27149", "27150", "27151", "27152"}
	smAssaultMinions = map[string]string{ // Last Ones Standing name -> Sinister Assault minion
		"27158": "27158", "27159": "27159", "27160": "27160",
		"27161": "27161", "27162": "27162", "27163": "27163",
	}
)

// smTrack maps reputation nodes to their effects. kind: 'i' = immediate
// (resolve when marked), 's' = recurring setup instruction.
type smNode struct {
	n    int
	kind byte
	key  string
}

var smTrack = []smNode{
	{1, 'i', "tech"},
	{2, 'i', "osborn"},
	{3, 's', "osbornShuffle"},
	{5, 's', "mulligan"},
	{6, 's', "threat"},
	{8, 'i', "aspect"},
	{10, 's', "minions"},
	{11, 'i', "enhanced"},
	{12, 'i', "osborn2"},
	{14, 'i', "plan"},
	{15, 's', "sideScheme"},
	{16, 's', "planSearch"},
	{18, 's', "helicarrier"},
	{20, 's', "symbiote"},
	{22, 'i', "osborn3"},
	{25, 's', "facedown"},
}

// smReputation computes the group's reputation value from the Conditions
// section of the reputation track.
func smReputation(st *State, snap Snapshot) int {
	rep := 0
	if snap.VictoryPoints > 0 {
		rep += snap.VictoryPoints // (+X) victory points
	}
	if len(snap.SideSchemeBaseCodes) == 0 {
		rep++
	}
	if snap.Acceleration < 1 {
		rep++
	}
	if snap.NoMinions {
		rep++
	}
	if snap.MainThreat == 0 {
		rep++
	}
	anyAlive := false
	for i := range st.Players {
		if !snap.KOed[i] {
			anyAlive = true
		}
	}
	if anyAlive {
		rep++
	}
	return rep
}

// smMark applies the reputation gain: newly marked nodes fire their
// immediate effects (choices queue as pending interlude picks).
func smMark(st *State, rep int) {
	for _, node := range smTrack {
		if node.n > rep {
			continue
		}
		if node.n <= st.Counters["marked"] {
			continue // already marked
		}
		st.Counters["marked"] = node.n
		if node.kind != 'i' {
			continue
		}
		switch node.key {
		case "tech":
			// Deal 3 random S.H.I.E.L.D. Tech upgrades; the player keeps
			// one (pending choice).
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
		case "osborn", "osborn2", "osborn3":
			st.SMOsborn = append(st.SMOsborn, smOsborn[randInt(len(smOsborn))])
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
		case "enhanced":
			for i := range st.Players {
				st.Players[i].SMEnhanced = true
			}
		}
	}
	st.Counters["reputation"] = rep
}

// smVictory implements the five victory programs (rulebook pages 9-17).
func smVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	rep := smReputation(st, snap)
	smMark(st, rep)
	// Community Service titles in the victory display.
	for _, code := range snap.VictoryDisplayCodes {
		if contains(smCommunity, code) {
			st.SMCommunity = appendUnique(st.SMCommunity, code)
		}
	}
	switch st.Index {
	case 2: // Mysterio: count Illusion cards in all player decks.
		st.SMWaking = snap.DeckIllusion
	case 3: // Sinister Six: record villains in play.
		st.SMLastStanding = append([]string{}, snap.VillainCodes...)
	}
	st.recordExpertHP(snap, living)
	st.Advance()
}

// smSetup applies the common and per-scenario setups (pages 9-17).
func smSetup(st *State, ctx *engine.CampaignSetup, opts *engine.NewGameOptions) {
	// Common: Public Outcry (expert flips to its expert side), Smear
	// Campaign.
	if st.IsExpert() {
		ctx.StartEnvironment = "27174b"
	} else {
		ctx.StartEnvironment = smPublicOutcry
	}
	ctx.PreShuffle = append(ctx.PreShuffle, smSmearCampaign)
	// One Community Service side scheme whose title is not yet recorded.
	if remaining := withoutAll(smCommunity, st.SMCommunity); len(remaining) > 0 {
		ctx.PreShuffle = append(ctx.PreShuffle, remaining[randInt(len(remaining))])
	}
	// Recurring reputation setup instructions, top to bottom.
	for _, node := range smTrack {
		if node.kind != 's' || node.n > st.Counters["marked"] {
			continue
		}
		switch node.key {
		case "osbornShuffle":
			ctx.PreShuffle = append(ctx.PreShuffle, st.SMOsborn...)
		case "threat":
			ctx.MainSchemeThreat++
		case "minions":
			ctx.MinionEngageEachPlayer = true
		case "sideScheme":
			ctx.RevealSideSchemeThreat = true
		case "planSearch":
			for i := range st.Players {
				st.Players[i].SetupHand = st.Players[i].SMPlanning
			}
		case "helicarrier":
			ctx.PoolUpgrades = append(ctx.PoolUpgrades, smHelicarrier)
		case "symbiote":
			ctx.PoolUpgrades = append(ctx.PoolUpgrades, smSymbioteSuit)
		case "facedown":
			for i := range st.Players {
				ctx.DealEncounter = append(ctx.DealEncounter, i)
			}
		}
		// "mulligan" (an additional mulligan during setup step 13) is
		// not modelable in the current setup pipeline (documented gap).
	}
	switch st.Index {
	case 0: // Sandman
		if st.IsExpert() {
			ctx.EnvCounters = map[string]int{smCityStreets: 2}
		}
	case 1: // Venom
		if st.IsExpert() {
			ctx.FacedownBoostEachPlayer = true
		}
	case 2: // Mysterio
		ctx.PoolAllies = append(ctx.PoolAllies, smVenomAlly)
		ctx.PreShuffle = append(ctx.PreShuffle, smSnitches)
		if st.IsExpert() {
			ctx.DeckShuffleEncounter = 2
		}
	case 3: // The Sinister Six
		ctx.PoolAllies = append(ctx.PoolAllies, smVenomAlly)
		ctx.PreShuffle = append(ctx.PreShuffle, smSnitches)
		if st.SMWaking > 0 {
			ctx.SideSchemeThreat = map[string]int{smLightAtEnd: st.SMWaking}
		}
		if st.IsExpert() {
			ctx.DeckShuffleEncounter = 2
		}
	case 4: // Venom Goblin
		for _, code := range st.SMLastStanding {
			if mn, ok := smAssaultMinions[code]; ok {
				ctx.PreShuffle = append(ctx.PreShuffle, mn)
			}
		}
		if st.IsExpert() {
			ctx.MainSchemeThreat++
		}
	}
	// Expert heal cost: facedown encounter cards (1/1/2/2/3 by chapter).
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

// ApplySMChoice resolves the SM-specific interlude picks.
func ApplySMChoice(st *State, slot int, kind, cardCode string) error {
	pl := st.Slot(slot)
	if pl == nil {
		return fmt.Errorf("unknown slot %d", slot)
	}
	switch kind {
	case ChoiceSMTech:
		if !contains(pl.SMTechOffer, cardCode) {
			return fmt.Errorf("not one of the offered TECH upgrades")
		}
		pl.SMShieldTech = cardCode
		pl.Deck[cardCode]++
		pl.SMTechOffer = nil
	case ChoiceSMAspect:
		def, ok := engine.DB.Lookup(cardCode)
		if !ok || def.Category != "player" || def.Aspect == "" {
			return fmt.Errorf("not an aspect card: %q", cardCode)
		}
		qty := def.Quantity
		if qty <= 0 {
			qty = 1
		}
		pl.Deck[cardCode] += qty
		pl.SMAspect = cardCode
	case ChoiceSMPlan:
		if pl.Deck[cardCode] <= 0 {
			return fmt.Errorf("card not in this player's deck: %q", cardCode)
		}
		pl.SMPlanning = cardCode
	default:
		return fmt.Errorf("unknown choice kind %q", kind)
	}
	st.ResolvePending(slot, kind)
	return nil
}
