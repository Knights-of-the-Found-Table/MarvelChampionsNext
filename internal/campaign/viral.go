package campaign

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// Going Viral (Henry Borkgren). Pym's Antivirus track races the Ultron
// Infection track; two of the three Scenario #2 chapters are played.
func viralSetup(st *State, ctx *engine.CampaignSetup) {
	secondTwo := viralPlayedCount(st) >= 1
	switch st.Index {
	case 1: // Zola
		ctx.ExtraSets = append(ctx.ExtraSets, setRansacked, setDoomsday)
		ctx.PoolMinions = append(ctx.PoolMinions, bioServant)
		if secondTwo {
			ctx.MillRevealAttachment = true
		}
	case 2: // The Sinister Six
		ctx.ExtraSets = append(ctx.ExtraSets, setGuerrillaTac, setOsbornTech)
		if secondTwo {
			ctx.PreShuffle = append(ctx.PreShuffle, roboticEnhance)
		}
	case 3: // Nebula
		ctx.ExtraSets = append(ctx.ExtraSets, setPowerStone, setExperimental)
		if secondTwo {
			ctx.ShipEvasion += 2
		}
	case 4: // Ultron Unlimited
		viralFinale(st, ctx)
	}
	// Pym's Antivirus setup effects (tallies 3/9/12/15).
	if st.Counters["pym"] >= 3 {
		for i := range st.Players {
			if code := st.Players[i].SMShieldTech; code != "" {
				addRoleUpgrade(ctx, i, code)
			}
		}
	}
	if st.Counters["pym"] >= 12 {
		if ctx.PlayerAllies == nil {
			ctx.PlayerAllies = map[int][]string{}
		}
		for i := range st.Players {
			if len(st.Players[i].Allies) > 0 {
				ctx.PlayerAllies[i] = append(ctx.PlayerAllies[i], st.Players[i].Allies[0])
			}
		}
	}
	// Ultron Infection general setup effects (tallies 5/10/13/15/20).
	inf := st.Counters["infection"]
	if inf >= 5 {
		ctx.ExtraSets = append(ctx.ExtraSets, setUnderAttack)
	}
	if inf >= 10 {
		ctx.MillRevealAttachment = true
	}
	if inf >= 13 {
		ctx.VillainTough = true
	}
	if inf >= 20 {
		for i := range st.Players {
			ctx.DealEncounter = append(ctx.DealEncounter, i)
		}
	}
}

// viralFinale applies the Scenario #3 results to the Ultron chapter.
func viralFinale(st *State, ctx *engine.CampaignSetup) {
	if st.Flags["zolaAlgorithm"] {
		for i := range st.Players {
			st.Players[i].Deck[zolasAlgorithm]++
		}
	}
	if st.Flags["modokAway"] {
		ctx.ExtraSets = append(ctx.ExtraSets, setDoomsday)
	}
	// The first player searches and reveals a side scheme.
	ctx.RevealSideSchemeThreat = true
	if st.Flags["sixUnited"] {
		ctx.ExtraSets = append(ctx.ExtraSets, setSinisterAsslt)
		ctx.PoolMinions = append(ctx.PoolMinions, docOckMinion, scorpionSM)
	}
	if st.Flags["scorpionAway"] {
		ctx.ExtraSets = append(ctx.ExtraSets, setAMessThings)
		ctx.PoolMinions = append(ctx.PoolMinions, scorpionGoblin)
	}
	if st.Flags["nebulaAway"] {
		ctx.ExtraSets = append(ctx.ExtraSets, setPowerStone)
		ctx.PreShuffle = append(ctx.PreShuffle, "16126") // Power Stone attachment
	}
	// Scenario #3 column of the infection track (5/10/13/15/20).
	inf := st.Counters["infection"]
	if inf >= 10 {
		ctx.ExtraSets = append(ctx.ExtraSets, setExperimental, setOsbornTech, setRansacked)
	}
	if inf >= 13 {
		for i := range st.Players {
			ctx.DealEncounter = append(ctx.DealEncounter, i)
		}
	}
}

// viralPymAllies lists the scenario-1 captives and their Antivirus
// tallies (hero-pack printings present in the data snapshot).
var viralPymTally = map[string]int{
	"26013": 1, // Jocasta
	"26015": 1, // Victor Mancha
	"13012": 1, // Wasp (Janet Van Dyne)
	"09039": 2, // Iron Man (Tony Stark)
	"01068": 2, // Vision
	"12011": 3, // Ant-Man (Hank Pym)
}

func viralVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	// Ultron Infection: one tally per 3 threat on the main scheme.
	inf := snap.MainThreat / 3
	switch st.Index {
	case 0:
		// Rescued captives join their rescuers and fill Pym's track.
		for code, slot := range snap.AllySlots {
			tally := viralPymTally[code]
			if tally == 0 && !contains(rrsRescuedCaptives, code) {
				continue
			}
			if t := viralPymTally[code]; t == 0 {
				tally = 1
			}
			if pl := st.Slot(slot); pl != nil && living(slot) {
				pl.Deck[code]++
				pl.Allies = appendUnique(pl.Allies, code)
				st.Counters["pym"] += tally
			}
		}
		st.Counters["infection"] += inf
		st.addPending(0, ChoiceViralNext)
	case 1:
		// Zola: M.O.D.O.K. captured when the Doomsday Chair is clear.
		if !contains(snap.MinionCodes, modokMinion) && !contains(snap.SideSchemeBaseCodes, "01183") {
			st.Flags["zolaStopped"] = true
			st.Counters["pym"] += 3
		} else {
			st.Flags["modokAway"] = true
			st.Counters["infection"] += 2
		}
		st.Counters["pym"] += countSetAttachments(snap)
		st.Counters["infection"] += inf
		if st.Flags["modokAway"] {
			st.Counters["infection"] += 2
		}
	case 2:
		// The Sinister Six: Scorpion captured when not in play.
		if !contains(snap.MinionCodes, scorpionSM) && !contains(snap.VillainCodes, scorpionSM) {
			st.Flags["sixStopped"] = true
			st.Counters["pym"]++
		} else {
			st.Flags["scorpionAway"] = true
			st.Counters["infection"]++
		}
		st.Counters["pym"] += snap.VictoryPoints
		st.Counters["pym"] += countSetAttachments(snap)
		st.Counters["infection"] += inf
	case 3:
		// Nebula: captured when her ship ran out of evasion counters.
		if snap.ShipCounters == 0 {
			st.Flags["nebulaStopped"] = true
		} else {
			st.Flags["nebulaAway"] = true
			st.Counters["infection"]++
		}
		st.Counters["pym"] += countSetAttachments(snap)
		st.Counters["infection"] += inf
	case 4:
		// Ultron defeated: the campaign is won.
	}
	viralAdvance(st)
	st.recordExpertHP(snap, living)
}

// countSetAttachments counts ARMOR/TECH/WEAPON attachments that entered
// play (the encounter discard is not observable; in-play attachments are
// the documented approximation).
func countSetAttachments(snap Snapshot) int {
	n := 0
	for _, code := range snap.AttachmentCodes {
		if def, ok := engine.DB.Lookup(code); ok &&
			(def.HasTrait("armor") || def.HasTrait("tech") || def.HasTrait("weapon")) {
			n++
		}
	}
	return n
}

// viralPlayedCount counts the Scenario #2 chapters already played.
func viralPlayedCount(st *State) int {
	played := st.Selections["viralPlayed"]
	if played == "" {
		return 0
	}
	return strings.Count(played, ",") + 1
}

// viralDefeat folds a lost chapter: Scenario #2 cannot be retried — the
// villains win that plot and the infection surges; other chapters replay.
func viralDefeat(st *State) {
	switch st.Index {
	case 1:
		st.Flags["modokAway"] = true
		st.Counters["infection"] += 5
		st.Index = 4
	case 2:
		st.Flags["sixUnited"] = true
		st.Counters["infection"] += 10
		st.Index = 4
	case 3:
		st.Flags["nebulaAway"] = true
		st.Counters["infection"]++
		st.Index = 4
	default:
		st.Status = "interlude"
		return
	}
	st.Status = "interlude"
}

// viralAdvance routes the branching chain: after the opener the group
// plays two of the three Scenario #2 chapters (the pending viral-next
// pick jumps the index), then the finale.
func viralAdvance(st *State) {
	if st.Index == 0 {
		// The viral-next choice sets the actual index; park at 0 until
		// the pick lands so the interlude UI stays consistent.
		st.Status = "interlude"
		return
	}
	played := st.Selections["viralPlayed"]
	if st.Index >= 1 && st.Index <= 3 {
		if played != "" {
			played += ","
		}
		played += strconv.Itoa(st.Index)
		st.Selections["viralPlayed"] = played
		if viralPlayedCount(st) >= 2 {
			st.Index = 4 // both #2 chapters done: Ultron Unlimited
		} else {
			// The remaining #2 the group did not just play.
			for _, idx := range []int{1, 2, 3} {
				if idx != st.Index && !strings.Contains(played, strconv.Itoa(idx)) {
					st.Index = idx
					break
				}
			}
		}
		st.Status = "interlude"
		return
	}
	st.Advance()
}

// ApplyViralChoice resolves the "which Scenario #2 next" group pick.
func ApplyViralChoice(st *State, slot int, kind, cardCode string) error {
	if kind != ChoiceViralNext {
		return fmt.Errorf("unknown choice kind %q", kind)
	}
	idx, err := strconv.Atoi(cardCode)
	if err != nil || idx < 1 || idx > 3 {
		return fmt.Errorf("not a Scenario #2 chapter: %q", cardCode)
	}
	st.Index = idx
	st.ResolvePending(slot, kind)
	return nil
}
