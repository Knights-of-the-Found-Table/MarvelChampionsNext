package campaign

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// The NeXt Evolution campaign: the group picks one campaign player side
// scheme per chapter; defeating it earns the environment (b-side) for all
// future chapters, while the scheme's paired encounter card haunts every
// later encounter deck.
var nxSchemes = []struct {
	a, env, encounter string
}{
	{"40190a", "40190b", "40199"}, // Assemble the Team / Malice
	{"40191a", "40191b", "40201"}, // Establish Safehouse / Vanisher
	{"40192a", "40192b", "40203"}, // Gear Up / Overburdened
	{"40193a", "40193b", "40200"}, // Mission Prep / Scrambler
	{"40194a", "40194b", "40198"}, // Practice Maneuvers / Lady Mastermind
	{"40195a", "40195b", "40202"}, // Prepare Defenses / Under Pressure
}

const (
	ChoiceNXScheme = "nx-scheme" // group pick for the next chapter
	nxHopeAlly     = "40130"     // Hope Summers ally whose damage is tracked
)

// nxQueueScheme queues the scheme pick for the host slot after a won
// chapter (a defeat keeps the current pick for the retry).
func nxQueueScheme(st *State) {
	if st.NXCurrent == "" && len(st.NXChosen) < len(nxSchemes) && len(st.NXAvailable()) > 0 {
		if !st.PendingFor(0, ChoiceNXScheme) {
			st.AddPending(0, ChoiceNXScheme)
		}
	}
}

// ApplyNXScheme records the group's chosen player side scheme.
func ApplyNXScheme(st *State, slot int, code string) error {
	for _, s := range st.NXAvailable() {
		if s == code {
			st.NXCurrent = code
			st.NXChosen = append(st.NXChosen, code)
			st.ResolvePending(slot, ChoiceNXScheme)
			return nil
		}
	}
	return fmt.Errorf("player side scheme not available: %q", code)
}

// nxVictory implements the five victory programs (pages 9-18).
func nxVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	// Earned environments: any campaign environment (b-side) in play at
	// the end marks the pairing as earned.
	for _, s := range nxSchemes {
		if contains(snap.EnvironmentCodes, s.env) && !contains(st.NXEnvEarned, s.env) {
			st.NXEnvEarned = append(st.NXEnvEarned, s.env)
		}
	}
	// Hope Summers damage tracking (chapters 3-5).
	if st.Index >= 2 {
		if dmg, ok := snap.AllyDamage[nxHopeAlly]; ok {
			st.Counters["hopeDamage"] = dmg
		}
	}
	// A won chapter releases the chosen scheme pick; the failed scheme's
	// card leaves the campaign (it can never be chosen again — it was
	// already recorded as chosen).
	st.NXCurrent = ""
	st.recordExpertHP(snap, living)
	st.Advance()
	if st.Status == "interlude" {
		nxQueueScheme(st)
	}
}

// nxSetup applies the setups (pages 9-18).
func nxSetup(st *State, ctx *engine.CampaignSetup, opts *engine.NewGameOptions) {
	// Earned environments come into play.
	ctx.StartEnvironments = append(ctx.StartEnvironments, st.NXEnvEarned...)
	// Paired encounter cards of every chosen scheme haunt the deck.
	for _, chosen := range st.NXChosen {
		for _, s := range nxSchemes {
			if s.a == chosen {
				ctx.PreShuffle = append(ctx.PreShuffle, s.encounter)
			}
		}
	}
	// The current chapter's player side scheme (retry keeps the pick).
	if st.NXCurrent != "" {
		ctx.PlayerSideScheme = st.NXCurrent
	}
	switch st.Index {
	case 2: // Juggernaut: one momentum counter per environment.
		ctx.EnvCounters = map[string]int{"40123": len(st.NXEnvEarned)}
	case 3: // Mister Sinister: threat on Teleported Away (approximated
		// onto the main scheme) instead of damaging Hope.
		if dmg := st.Counters["hopeDamage"]; dmg > 0 {
			ctx.MainSchemeThreat += dmg
		}
	case 4: // Stryfe: threat on Stryfe's Grasp per environment; mill.
		if len(st.NXEnvEarned) > 0 {
			ctx.MainSchemeThreat += len(st.NXEnvEarned)
		}
		ctx.MillRevealMinionOrPsionic = true
	}
	// Expert heal costs: a facedown encounter card in chapters 2, 3 and
	// 5; an acceleration token in chapter 4.
	if st.IsExpert() && st.Index >= 1 {
		for i := range st.Players {
			if !st.Players[i].HealNext {
				continue
			}
			if st.Index == 3 {
				ctx.MainSchemeAcceleration++
			} else {
				ctx.DealEncounter = append(ctx.DealEncounter, i)
			}
		}
	}
}

// QueueNXScheme exposes the scheme-pick queueing for the API layer.
func QueueNXScheme(st *State) { nxQueueScheme(st) }

// NXAllSchemes lists every campaign player side scheme code.
func NXAllSchemes() []string {
	out := make([]string, 0, len(nxSchemes))
	for _, s := range nxSchemes {
		out = append(out, s.a)
	}
	return out
}
