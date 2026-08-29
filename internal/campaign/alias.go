package campaign

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// Alias Investigations (Christian Fecteau). Jessica Jones works every
// chapter; the Clue Deck persists between scenarios and the clue drawn at
// the mall picks the final villain.
var (
	aliasClues1 = []string{radiactiveMan, madameHydra, mystiqueMG}
	aliasClues2 = []string{"04108", "04123", "32119"} // Training Camp / Test Subjects / Intruder Alert!
	// aliasFinal maps the chapter-two clue to its chain index.
	aliasFinal = map[string]int{"04108": 2, "04123": 3, "32119": 4}
	// aliasClueSets maps the chapter-one clue to the modular set it adds
	// to chapters 2 and 3.
	aliasClueSets = map[string]string{
		radiactiveMan: setMoE,
		madameHydra:   setLegionsHydra,
		mystiqueMG:    setMystique,
	}
)

func aliasSetup(st *State, ctx *engine.CampaignSetup) {
	// Jessica Jones works the case (first copy: Victory -1 and no ally
	// limit are printed effects; the ally-limit part is a documented
	// approximation).
	ctx.PoolAllies = append(ctx.PoolAllies, "01059")
	if clue := aliasClueSets[st.Selections["clue1"]]; clue != "" {
		ctx.ExtraSets = append(ctx.ExtraSets, clue)
	}
	switch st.Index {
	case 0:
		ctx.ExtraSets = append(ctx.ExtraSets, setWeaponMaster)
	case 1:
		ctx.ExtraSets = append(ctx.ExtraSets, setProjectWideaw)
		// One copy of Abduction Protocols leaves the game (data gap: the
		// whole set plays — documented approximation).
	case 2, 3, 4:
		ctx.ExtraSets = append(ctx.ExtraSets, aliasFinalSets(st)...)
	}
}

// aliasFinalSets tailors the final battle to the villain the second clue
// selected (the scenarios' own default decks already match the rulebook).
func aliasFinalSets(st *State) []string {
	switch st.Selections["clue2"] {
	case "04108":
		return []string{setHydraPatrol}
	}
	return nil
}

func aliasVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	// Jessica Jones: a wound for every ending without her on the board.
	_, jjInPlay := snap.AllySlots["01059"]
	if !jjInPlay {
		st.Counters["wounds"]++
	}
	// Victory Tally.
	st.Counters["tally"] += snap.VictoryPoints
	switch st.Index {
	case 0:
		st.Selections["clue1"] = aliasDrawClue(st, aliasClues1, "clue1")
	case 1:
		// Rescued captive allies join their rescuers' decks.
		for code, slot := range snap.AllySlots {
			if contains(mgCaptiveAllies, code) {
				if pl := st.Slot(slot); pl != nil && living(slot) {
					pl.Deck[code]++
					pl.Allies = appendUnique(pl.Allies, code)
					st.Victims = appendUnique(st.Victims, code)
				}
			}
		}
		// The clue drawn at the mall picks the final battle; jump the
		// chain straight to that chapter.
		st.Selections["clue2"] = aliasDrawClue(st, aliasClues2, "clue2")
		st.Index = aliasFinal[st.Selections["clue2"]]
		st.Status = "interlude"
		st.recordExpertHP(snap, living)
		return
	case 2, 3, 4:
		st.Status = "complete"
		st.recordExpertHP(snap, living)
		return
	}
	st.recordExpertHP(snap, living)
	st.Advance()
}

// aliasDrawClue reveals the next clue: still-unrevealed candidates are
// shuffled by the deterministic campaign draw and the top card enters
// play; in this approximation the reveal happens at the scenario's end,
// and the card always counts as Evidence Gathered.
func aliasDrawClue(st *State, candidates []string, key string) string {
	var left []string
	for _, c := range candidates {
		if st.Selections[key] != c && !contains(st.Pool, "evidence:"+c) {
			left = append(left, c)
		}
	}
	if len(left) == 0 {
		return ""
	}
	pick := left[randInt(len(left))]
	st.Pool = append(st.Pool, "evidence:"+pick)
	return pick
}

// AliasEvidence lists the evidence codes recorded in the Pool (exported
// for the API payload's name table).
func AliasEvidence(st *State) []string {
	var out []string
	for _, e := range st.Pool {
		if len(e) > 9 && e[:9] == "evidence:" {
			out = append(out, e[9:])
		}
	}
	return out
}
