package campaign

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// The Crimson Cowl Conspiracy (Kurt Hake). S.H.I.E.L.D. Tech proficiencies
// reuse the Sinister Motives offer/record fields; Masters of Evil minions
// are tracked as caught (victory display) or escaped (in play) and shuffle
// back into later chapters.
var cowlAspects = []string{"leadership", "aggression", "protection", "justice"}

// cowlGear maps the chapter's aspect to the support the other players put
// into play (Quinjet / Counterattack / Anticipation / Interrogation Room).
var cowlGear = []string{quinjet, counterattack, anticipation, interrogation}

// cowlExtraSets lists the modular sets the campaign adds per chapter
// (beyond the scenario's own deck composition).
var cowlExtraSets = [][]string{
	{setBrothersGrimm, setBeastyBoys, setCrossfire},
	{setSinisterSynd},
	{setMisterHyde},
	nil,
	{setMoE},
}

func cowlSetup(st *State, ctx *engine.CampaignSetup) {
	idx := st.Index
	if idx < len(cowlExtraSets) {
		ctx.ExtraSets = append(ctx.ExtraSets, cowlExtraSets[idx]...)
	}
	// Recorded S.H.I.E.L.D. Tech upgrades join their owner.
	for i := range st.Players {
		if code := st.Players[i].SMShieldTech; code != "" {
			addRoleUpgrade(ctx, i, code)
		}
	}
	// The team support of this chapter (non-aspect players; we give it to
	// everyone — the aspect bonus of an extra opening card is a
	// documented approximation).
	if idx < len(cowlGear) {
		ctx.PoolSupports = append(ctx.PoolSupports, cowlGear[idx])
	}
	// Escaped Masters of Evil minions shuffle back in.
	ctx.PreShuffle = append(ctx.PreShuffle, st.Pool...)
	if idx == 4 {
		// Intel thresholds: with data-snapshot decks the caught list only
		// fills during the finale, so these rarely fire; each unlocked
		// tier grants its gear to every player (the aspect-specific extra
		// opening card is not modelable at setup).
		intel := cowlIntel(st)
		if intel >= 1 {
			ctx.PoolSupports = append(ctx.PoolSupports, quinjet)
		}
		if intel >= 3 {
			ctx.PoolUpgrades = append(ctx.PoolUpgrades, counterattack)
		}
		if intel >= 5 {
			ctx.PoolUpgrades = append(ctx.PoolUpgrades, anticipation)
		}
		if intel >= 7 {
			ctx.PoolSupports = append(ctx.PoolSupports, interrogation)
		}
		if intel >= 9 {
			ctx.VillainTough = true
		}
		// Facedown Masters of Evil minion per player.
		for i := range st.Players {
			ctx.DealEncounter = append(ctx.DealEncounter, i)
		}
	}
	if idx == 3 {
		// A Corrupt Prison Guard per player (documented approximation:
		// any minion, not the Wrecking Crew's own).
		ctx.MinionEngageEachPlayer = true
	}
}

// cowlIntel sums the printed scheme values of the caught minions.
func cowlIntel(st *State) int {
	intel := 0
	for _, code := range st.CowlCaught {
		if def, ok := engine.DB.Lookup(code); ok {
			intel += deref(def.Scheme, 0)
		}
	}
	return intel
}

func cowlVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	caught := func(code string) bool { return contains(snap.VictoryDisplayCodes, code) }
	for _, code := range snap.VictoryDisplayCodes {
		if contains(moeMinions, code) {
			st.CowlCaught = appendUnique(st.CowlCaught, code)
			st.Pool = without(st.Pool, code)
		}
	}
	for _, code := range snap.MinionCodes {
		if contains(moeMinions, code) && !caught(code) {
			st.Pool = appendUnique(st.Pool, code)
		}
	}
	// Each player without a recorded proficiency is dealt three campaign
	// S.H.I.E.L.D. Tech upgrades (same offer/record flow as the Sinister
	// Motives campaign).
	for i := range st.Players {
		pl := &st.Players[i]
		if !living(i) || pl.SMShieldTech != "" {
			continue
		}
		pool := append([]string{}, smTech...)
		picks := []string{}
		for len(picks) < 3 && len(pool) > 0 {
			idx := randInt(len(pool))
			picks = append(picks, pool[idx])
			pool = append(pool[:idx], pool[idx+1:]...)
		}
		pl.SMTechOffer = picks
		st.addPending(i, ChoiceSMTech)
	}
	// Players who already carry a proficiency flip it to its ENHANCED
	// side (the enhanced printings are absent from the data snapshot, so
	// the upgrade keeps its basic effects — documented approximation).
	if st.Index > 0 {
		for i := range st.Players {
			if living(i) && st.Players[i].SMShieldTech != "" {
				st.Players[i].SMEnhanced = true
			}
		}
	}
	st.recordExpertHP(snap, living)
	st.Advance()
}
