package campaign

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// The Watcher's Team: What If...? (Zach Goscha). Every chapter demands a
// specific identity; the deck is rebuilt between chapters via the deck
// swap. The setup rides campaign-granted setup-keyword cards into play.
var watchersRequired = [][]string{
	{heroIronMan},
	{heroSpiderman, heroPeterSM},
	{heroRogue},
	{heroCap},
	{heroIronMan, heroSpiderman, heroPeterSM, heroRogue, heroCap},
}

// watchersValidate enforces the per-chapter identity requirement.
func watchersValidate(st *State) error {
	for i := range st.Players {
		base := st.Players[i].HeroBase
		if !contains(watchersRequired[st.Index], base) {
			name := st.BoxDef().Scenarios[st.Index].Requires
			return fmt.Errorf("chapter requires identity: %s (player %d is %s)", name, i+1, base)
		}
	}
	return nil
}

func watchersSetup(st *State, ctx *engine.CampaignSetup) {
	slotOf := func(bases ...string) int {
		for i := range st.Players {
			if contains(bases, st.Players[i].HeroBase) {
				return i
			}
		}
		return -1
	}
	switch st.Index {
	case 0:
		ctx.ExtraSets = append(ctx.ExtraSets, setMojoFantasy, setFrostGiants)
		for i := range st.Players {
			st.Players[i].Deck[systemShock]++
		}
		if s := slotOf(heroIronMan); s >= 0 {
			st.Players[s].Deck[godslayer]++
			ctx.SetupKeywordCards = append(ctx.SetupKeywordCards, godslayer)
		}
	case 1:
		ctx.ExtraSets = append(ctx.ExtraSets, setHopeSummers, setMojoHorror, setBlackTom)
		if s := slotOf(heroSpiderman, heroPeterSM); s >= 0 {
			st.Players[s].Deck[sorcerSupreme]++
			ctx.SetupKeywordCards = append(ctx.SetupKeywordCards, sorcerSupreme)
			ctx.DealCard[s] = "40132" // Black Tom Cassidy minion (NX)
		}
	case 2:
		// The printed Infinity set is absent from the data snapshot
		// (documented gap); the rest of the deck is exact.
		ctx.ExtraSets = append(ctx.ExtraSets, setLegionsHel, setEnchantress)
		if s := slotOf(heroRogue); s >= 0 {
			st.Players[s].Deck[jarnbjorn]++
			ctx.SetupKeywordCards = append(ctx.SetupKeywordCards, jarnbjorn)
		}
		ctx.MillRevealAttachment = true
	case 3:
		ctx.ExtraSets = append(ctx.ExtraSets, setStreetsMayhem, setZzzax, setRansacked,
			setRunningIntf, setDownToEarth, setWeaponMaster, setIronSpiderSix)
		if s := slotOf(heroCap); s >= 0 {
			st.Players[s].Deck[lockAndLoad]++
			st.Players[s].Deck[laserCannon]++
		}
		ctx.StartDamageEachPlayer = 2
	case 4:
		ctx.ExtraSets = append(ctx.ExtraSets, setInfinityGaunt)
	}
}

func watchersVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	// The mid-scenario bonus abilities are self-reported as optional
	// named-card picks (the triggers — a SPELL kill, a fourth WOUNDED
	// flip, weapon kills — are not observable in the snapshot).
	switch st.Index {
	case 2:
		for i := range st.Players {
			if living(i) {
				st.addPending(i, ChoiceWAPortal)
			}
		}
	case 3:
		for i := range st.Players {
			if living(i) {
				st.addPending(i, ChoiceWAIntervened)
			}
		}
	}
	// Scenario 2's Cosmic Sight triggers on a SPELL-card kill (chapter
	// index 1); report it alongside the Portal for the following chapter.
	if st.Index == 1 {
		for i := range st.Players {
			if living(i) {
				st.addPending(i, ChoiceWASight)
			}
		}
	}
	st.recordExpertHP(snap, living)
	st.Advance()
}

// ApplyWAChoice records an optional named-card reward (empty declines).
func ApplyWAChoice(st *State, slot int, kind, cardCode string) error {
	pl := st.Slot(slot)
	if pl == nil {
		return fmt.Errorf("unknown slot %d", slot)
	}
	if cardCode == "" {
		st.ResolvePending(slot, kind)
		return nil
	}
	if pl.Deck[cardCode] <= 0 {
		return fmt.Errorf("card not in this player's deck: %q", cardCode)
	}
	var wantType string
	switch kind {
	case ChoiceWASight:
		wantType = "resource"
	case ChoiceWAPortal:
		wantType = "ally"
	case ChoiceWAIntervened:
		wantType = ""
	default:
		return fmt.Errorf("unknown choice kind %q", kind)
	}
	if def, ok := engine.DB.Lookup(cardCode); ok && wantType != "" && def.Type != wantType {
		return fmt.Errorf("not a %s: %q", wantType, cardCode)
	}
	pl.SetupHand = cardCode
	st.ResolvePending(slot, kind)
	return nil
}

// WatchersCheck validates the first chapter's identity requirement at
// campaign start (exported for the API layer).
func WatchersCheck(st *State) error { return watchersValidate(st) }
