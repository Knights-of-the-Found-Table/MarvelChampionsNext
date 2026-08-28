package campaign

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// The Age of Apocalypse campaign: side missions run in a separate mission
// area next to each scenario (chapters 1-4), striking missions and
// Overseers that resolve; the finale succeeds or fails on Protect the
// Professor.
var (
	aoaSet         = []string{"45177"} // North American Sea Wall
	aoaObligation  = "45178"           // Panicked Refugees
	aoaMissionsAll = []string{"45166a", "45167a", "45168a", "45169a", "45170a"}
	aoaOverseers   = []string{"45179a", "45180a", "45181a", "45182a", "45183a"}
	aoaProfessor   = "45170a" // Protect the Professor (finale only)
)

// aoaVictory implements the victory programs (pages 8-20).
func aoaVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	if st.Index < 4 {
		// Strike the mission (it resolved one way or the other) and the
		// Overseer if it was defeated.
		if st.AOMission != "" {
			st.AOMissionLog = appendUnique(st.AOMissionLog, st.AOMission)
			st.AOMission = ""
		}
		if st.AOOverseer != "" && !contains(snap.MinionCodes, st.AOOverseer) {
			st.AOOverseerLog = appendUnique(st.AOOverseerLog, st.AOOverseer)
		}
		st.AOOverseer = ""
	} else {
		// Finale: Protect the Professor decides the campaign.
		professorSaved := !contains(snap.SideSchemeBaseCodes, aoaProfessor)
		st.Flags["professorSaved"] = professorSaved
		if !professorSaved {
			st.Status = "lost"
			return
		}
	}
	st.recordExpertHP(snap, living)
	st.Advance()
}

// aoaSetup applies the setups (pages 8-20).
func aoaSetup(st *State, ctx *engine.CampaignSetup, opts *engine.NewGameOptions) {
	// The Age of Apocalypse modular set rides with every chapter.
	ctx.PreShuffle = append(ctx.PreShuffle, aoaSet...)
	// The Panicked Refugees obligation joins the first player's deck.
	ctx.ObligationFirstPlayer = aoaObligation
	// The mission: a random unstruck scheme (only Protect the Professor
	// in the finale), plus a random unstruck Overseer.
	var missions []string
	if st.Index == 4 {
		missions = []string{aoaProfessor}
	} else {
		missions = withoutAll(aoaMissionsAll, st.AOMissionLog)
	}
	if len(missions) > 0 {
		st.AOMission = missions[randInt(len(missions))]
		ctx.MissionScheme = st.AOMission
	}
	overseers := withoutAll(aoaOverseers, st.AOOverseerLog)
	if len(overseers) > 0 {
		st.AOOverseer = overseers[randInt(len(overseers))]
		ctx.MissionOverseer = st.AOOverseer
	}
	ctx.MissionTeam = true
	// Every player searches their deck for an ally and adds it to hand.
	for i := range st.Players {
		if ctx.HandFetch == nil {
			ctx.HandFetch = map[int]string{}
		}
		ctx.HandFetch[i] = "ally"
	}
	// Expert heal: 3 threat on the MISSION scheme.
	if st.IsExpert() {
		for i := range st.Players {
			if st.Players[i].HealNext {
				if st.AOMission != "" {
					ctx.SideSchemeThreat = map[string]int{st.AOMission: 3}
				}
			}
		}
	}
}
