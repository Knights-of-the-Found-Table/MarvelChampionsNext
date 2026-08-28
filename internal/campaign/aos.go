package campaign

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// The Agents of S.H.I.E.L.D. campaign: one Executive Board member is a
// mole. The mole's means/motive/opportunity evidence is hidden in the
// A.I.M. envelope at campaign start; gaining evidence cards from the
// S.H.I.E.L.D. envelope eliminates combinations and narrows the suspect
// before the chapter-5 accusation.
var (
	aosBoard        = []string{"50181a", "50182a", "50183a"}
	aosInterference = []string{"50184a", "50184b", "50184c"}
	aosMeans        = []string{"50185", "50186", "50187"}
	aosMotive       = []string{"50188", "50189", "50190"}
	aosOpportunity  = []string{"50191", "50192", "50193"}
)

// ChoiceAOAccuse names the accusation pick in the finale interlude.
const ChoiceAOAccuse = "aos-accuse"

// AOSCombo is one means/motive/opportunity accusation.
type AOSCombo [3]string

// aosCombos lists the printed combination table (campaign log page 24),
// keyed by the accused board member.
func aosCombos() map[string][]AOSCombo {
	m, o, p := aosMeans, aosMotive, aosOpportunity
	return map[string][]AOSCombo{
		"50181a": { // Chief Medical Officer
			{m[0], o[0], p[0]}, {m[0], o[0], p[1]}, {m[0], o[1], p[0]},
			{m[0], o[1], p[2]}, {m[0], o[2], p[1]},
			{m[1], o[0], p[0]}, {m[1], o[0], p[1]}, {m[1], o[1], p[0]},
			{m[2], o[0], p[0]},
		},
		"50182a": { // Chief Surveillance Officer
			{m[0], o[1], p[1]}, {m[1], o[0], p[2]}, {m[1], o[1], p[1]},
			{m[1], o[1], p[2]}, {m[1], o[2], p[0]}, {m[1], o[2], p[1]},
			{m[2], o[1], p[1]}, {m[2], o[1], p[2]}, {m[2], o[2], p[1]},
		},
		"50183a": { // Chief Tactical Officer
			{m[0], o[0], p[2]}, {m[0], o[2], p[0]}, {m[0], o[2], p[2]},
			{m[1], o[2], p[2]}, {m[2], o[0], p[1]}, {m[2], o[0], p[2]},
			{m[2], o[1], p[0]}, {m[2], o[2], p[0]}, {m[2], o[2], p[2]},
		},
	}
}

// PrepareAOSEvidence rolls the mole's hidden evidence at campaign start.
func PrepareAOSEvidence(st *State) {
	var zero AOSCombo
	if st.AOImEnvelope != zero {
		return
	}
	pick := func(list []string) string { return list[randInt(len(list))] }
	st.AOImEnvelope = AOSCombo{pick(aosMeans), pick(aosMotive), pick(aosOpportunity)}
	for _, list := range [][]string{aosMeans, aosMotive, aosOpportunity} {
		for _, code := range list {
			if !contains(st.AOImEnvelope[:], code) {
				st.AOShieldEnvelope = append(st.AOShieldEnvelope, code)
			}
		}
	}
	st.AOCounters = map[string]int{aosBoard[0]: 2, aosBoard[1]: 2, aosBoard[2]: 2}
}

// aosCombosRemaining lists accusation combinations not eliminated by the
// gained evidence.
func aosCombosRemaining(st *State) map[string][]AOSCombo {
	out := map[string][]AOSCombo{}
	for member, combos := range aosCombos() {
		for _, combo := range combos {
			eliminated := false
			for _, gained := range st.AOEvidence {
				if contains(combo[:], gained) {
					eliminated = true
					break
				}
			}
			if !eliminated {
				out[member] = append(out[member], combo)
			}
		}
	}
	return out
}

// QueueAOAccuse offers the accusation before the finale.
func QueueAOAccuse(st *State) {
	if st.Index == 4 && len(st.AOEvidence) < 6 {
		if !st.PendingFor(0, ChoiceAOAccuse) {
			st.AddPending(0, ChoiceAOAccuse)
		}
	}
}

// ApplyAOAccuse records the finale accusation.
func ApplyAOAccuse(st *State, slot int, member string) error {
	if member != aosBoard[0] && member != aosBoard[1] && member != aosBoard[2] {
		return fmt.Errorf("not a board member: %q", member)
	}
	st.Flags["accused"] = true
	if contains(aosBoard, member) && aosMemberIsMole(st, member) {
		st.Flags["accusedCorrect"] = true
	} else {
		st.Flags["accusedCorrect"] = false
	}
	st.ResolvePending(slot, ChoiceAOAccuse)
	return nil
}

func aosMemberIsMole(st *State, member string) bool {
	combos := aosCombos()[member]
	for _, combo := range combos {
		if combo == st.AOImEnvelope {
			return true
		}
	}
	return false
}

// aoaVictory's sibling: the AoS victory programs (pages 9-19).
func aosVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	// Remaining secret counters carry to the next chapter; a member
	// cleared to 0 grants that member's category evidence from the
	// S.H.I.E.L.D. envelope (approximation of the in-scenario earning).
	st.AOCounters = map[string]int{}
	for i, member := range aosBoard {
		n := snap.EnvCountersByCode[member]
		st.AOCounters[member] = n
		if n == 0 {
			var pool []string
			switch i {
			case 0:
				pool = aosMeans
			case 1:
				pool = aosMotive
			default:
				pool = aosOpportunity
			}
			for _, code := range pool {
				if contains(st.AOShieldEnvelope, code) && !contains(st.AOEvidence, code) {
					st.AOEvidence = append(st.AOEvidence, code)
					st.AOShieldEnvelope = without(st.AOShieldEnvelope, code)
					break
				}
			}
		}
	}
	switch st.Index {
	case 0: // Black Widow: minions and side schemes in play.
		st.Counters["aosMinions"] = len(snap.MinionCodes) + len(snap.SideSchemeBaseCodes)
	case 2: // M.O.D.O.K.: surviving Adaptoid environments are absent
		// from the data snapshot (documented gap).
	case 3: // Thunderbolts: surviving Thunderbolt minions.
		for _, code := range snap.MinionCodes {
			if def, ok := engine.DB.Lookup(code); ok && def.HasTrait("thunderbolt") {
				st.AOSurvivors = appendUnique(st.AOSurvivors, code)
			}
		}
	}
	st.recordExpertHP(snap, living)
	st.Advance()
	if st.Status == "interlude" {
		QueueAOAccuse(st)
	}
}

// aosSetup applies the setups (pages 9-19).
func aosSetup(st *State, ctx *engine.CampaignSetup, opts *engine.NewGameOptions) {
	ctx.BoardCounters = map[string]int{}
	for _, member := range aosBoard {
		ctx.BoardCounters[member] = st.AOCounters[member]
	}
	ctx.PreShuffle = append(ctx.PreShuffle, aosInterference...)
	switch st.Index {
	case 1: // Batroc: Alert Level threat (approximated onto the main scheme).
		if n := st.Counters["aosMinions"]; n > 0 {
			ctx.MainSchemeThreat += n
		}
	case 4: // Baron Zemo: surviving Thunderbolts rejoin him.
		ctx.PreShuffle = append(ctx.PreShuffle, st.AOSurvivors...)
	}
	// S2+ counters were recorded at the previous victory; S1's fresh 2
	// per member came from PrepareAOSEvidence.
}
