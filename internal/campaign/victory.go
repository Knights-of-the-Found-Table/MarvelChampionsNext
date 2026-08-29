package campaign

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// printedHP looks up an identity's printed hit points (the cap for expert
// persistent damage).
func printedHP(heroBase string) int {
	if def, ok := engine.DB.Lookup(data.HeroSideCode(heroBase)); ok && def.HP != nil {
		return *def.HP
	}
	return 0
}

// ApplyVictory folds a won scenario into the campaign log: auto-recorded
// fields land immediately, player choices are queued in PendingChoices.
// Deterministic: safe to re-run after an undo/replay of the same game.
func ApplyVictory(st *State, snap Snapshot) {
	st.LastResult = "won"
	living := func(i int) bool { return !snap.KOed[i] }
	switch st.Box {
	case "rrs":
		// Experimental weapons that entered play (scenario 1 victory).
		if st.Index == 0 {
			for _, code := range snap.Experimental {
				st.Experimental = appendUnique(st.Experimental, code)
			}
		}
		// Delay counters left on None Shall Pass (scenario 2 victory).
		if st.Index == 1 {
			st.DelayCounters = snap.DelayCounters
		}
		// Rescued captives join their rescuer's deck (scenario 3).
		if st.Index == 2 {
			for slot, code := range snap.Rescued {
				if pl := st.Slot(slot); pl != nil && living(slot) {
					pl.Deck[code]++
					pl.Allies = appendUnique(pl.Allies, code)
				}
			}
		}
		// Zola (scenario 4): engagement markers + Hydra Prison or the
		// Improved-condition offers.
		if st.Index == 3 {
			for i := range st.Players {
				if living(i) {
					st.Players[i].EngagedEnemy = snap.Engaged[i]
				}
			}
			if snap.HydraPrisonInPlay {
				for _, code := range snap.PrisonAllies {
					st.RemovedAllies = appendUnique(st.RemovedAllies, code)
					for i := range st.Players {
						delete(st.Players[i].Deck, code)
						st.Players[i].Allies = without(st.Players[i].Allies, code)
					}
				}
			} else {
				for i := range st.Players {
					pl := &st.Players[i]
					if living(i) && snap.HeroForm[i] && pl.Condition != "" && !pl.Improved {
						st.addPending(i, ChoiceImprove)
					}
				}
			}
		}
		// Scenario 1/2 upgrade offers.
		if st.Index == 0 {
			for i := range st.Players {
				if living(i) && st.Players[i].Tech == "" {
					st.addPending(i, ChoiceTech)
				}
			}
		}
		if st.Index == 1 {
			for i := range st.Players {
				if living(i) && st.Players[i].Condition == "" {
					st.addPending(i, ChoiceCondition)
				}
			}
		}
	case "mts":
		mtsVictory(st, snap)
		return
	case "sm":
		smVictory(st, snap)
		return
	case "mg":
		mgVictory(st, snap)
		return
	case "nx":
		nxVictory(st, snap)
		return
	case "aoa":
		aoaVictory(st, snap)
		return
	case "aos":
		aosVictory(st, snap)
		return
	// Contest campaigns.
	case "cowl":
		cowlVictory(st, snap)
		return
	case "whatif":
		wiVictory(st, snap)
		return
	case "awesome":
		awVictory(st, snap)
		return
	case "alias":
		aliasVictory(st, snap)
		return
	case "watchers":
		watchersVictory(st, snap)
		return
	case "mojo":
		mojoVictory(st, snap)
		return
	case "bord":
		bordVictory(st, snap)
		return
	case "night":
		nightVictory(st, snap)
		return
	case "viral":
		viralVictory(st, snap)
		return
	case "entropy":
		entVictory(st, snap)
		return
	case "gmw":
		gain := 0
		for i := range st.Players {
			if living(i) {
				gain++
			}
		}
		// Shared victory-value units, capped at 3 per player.
		vp := snap.VictoryPoints
		if vp > 3 {
			vp = 3
		}
		gain += vp
		switch st.Index {
		case 0:
			if snap.NoMinions {
				gain += livingCount(st, living)
			}
			if snap.MainSchemeCode == "16062b" {
				gain += livingCount(st, living)
			}
		case 1:
			if snap.CollectionCount <= 1 {
				gain += livingCount(st, living)
			}
			if snap.MainThreat == 0 {
				gain += livingCount(st, living)
			}
			for _, code := range snap.CollectionPlayerCodes {
				st.Collection = appendUnique(st.Collection, code)
			}
		case 2:
			gain += 2 * (len(snap.ArtifactsInDisplay) / 2)
			for _, code := range snap.ArtifactsInDisplay {
				st.Artifacts = appendUnique(st.Artifacts, code)
			}
		case 3:
			if snap.ShipCounters <= 1 {
				gain += livingCount(st, living)
			}
			if snap.MainSchemeCode == "16092b" {
				gain += livingCount(st, living)
			}
			st.Evasion = snap.ShipCounters
			if snap.PowerStoneSlot >= 0 {
				st.PowerStone = snap.PowerStoneSlot
			}
		}
		for i := range st.Players {
			if living(i) {
				st.Players[i].Units += gain
			}
		}
		if st.Index < 4 && snap.HeadhunterDown {
			if st.Headhunter == nil {
				st.Headhunter = make([]bool, 4)
			}
			st.Headhunter[st.Index] = true
		}
	}
	st.recordExpertHP(snap, living)
	st.Advance()
}

// ApplyDefeat folds a lost scenario: the chapter may be retried with no
// penalty — except the expert Ronan finale, where the campaign is lost.
func ApplyDefeat(st *State) {
	st.LastResult = "lost"
	if (st.Box == "gmw" || st.Box == "mts" || st.Box == "mg") && st.Index == 4 && st.IsExpert() {
		st.Status = "lost"
		return
	}
	// Contest campaign defeat programs.
	switch st.Box {
	case "night":
		nightDefeat(st)
		return
	case "viral":
		viralDefeat(st)
		return
	case "entropy":
		if st.Index == 4 && st.IsExpert() {
			st.Status = "lost"
			return
		}
	case "awesome":
		if st.Index == 4 && st.IsExpert() {
			st.Status = "lost"
			return
		}
	}
	st.Status = "interlude"
}

func livingCount(st *State, living func(int) bool) int {
	n := 0
	for i := range st.Players {
		if living(i) {
			n++
		}
	}
	return n
}

// recordExpertHP writes each surviving identity's remaining hit points
// (capped at the printed base); eliminated heroes rejoin at full health.
func (st *State) recordExpertHP(snap Snapshot, living func(int) bool) {
	if !st.IsExpert() {
		return
	}
	for i := range st.Players {
		if !living(i) {
			st.Players[i].HP = 0
			continue
		}
		hp := snap.HP[i]
		if base := printedHP(st.Players[i].HeroBase); base > 0 && hp > base {
			hp = base
		}
		st.Players[i].HP = hp
	}
}

func (st *State) addPending(slot int, kind string) {
	if st.PendingChoices == nil {
		st.PendingChoices = map[string]string{}
	}
	st.PendingChoices[PendingKey(slot, kind)] = kind
}

func without(list []string, s string) []string {
	out := make([]string, 0, len(list))
	for _, x := range list {
		if x != s {
			out = append(out, x)
		}
	}
	return out
}

// ApplyChoice resolves one interlude choice. cardCode "" declines an
// optional choice (condition / improve).
func ApplyChoice(st *State, slot int, kind, cardCode string) error {
	if !st.PendingFor(slot, kind) {
		return fmt.Errorf("player %d has no pending %s choice", slot, kind)
	}
	pl := st.Slot(slot)
	if pl == nil {
		return fmt.Errorf("unknown slot %d", slot)
	}
	switch kind {
	case ChoiceSMTech, ChoiceSMAspect, ChoiceSMPlan:
		return ApplySMChoice(st, slot, kind, cardCode)
	case ChoiceMGRole:
		return ApplyMGRoleChoice(st, slot, cardCode)
	case ChoiceNXScheme:
		return ApplyNXScheme(st, slot, cardCode)
	case ChoiceAOAccuse:
		return ApplyAOAccuse(st, slot, cardCode)
	// Contest campaign picks.
	case ChoiceWITrait, ChoiceWIAlly, ChoiceWICard:
		return ApplyWIChoice(st, slot, kind, cardCode)
	case ChoiceAWAlly:
		return ApplyAWAlly(st, slot, cardCode)
	case ChoiceAWIdentity:
		return ApplyAWIdentity(st, slot, cardCode)
	case ChoiceMojoRole, ChoiceMojoTraining, ChoiceMojoEvent, ChoiceMojoMarket, ChoiceMojoScheme:
		return ApplyMojoChoice(st, slot, kind, cardCode)
	case ChoiceBordPath, ChoiceBordGear:
		return ApplyBordChoice(st, slot, kind, cardCode)
	case ChoiceNTMeta, ChoiceNTTeam:
		return ApplyNTChoice(st, slot, kind, cardCode)
	case ChoiceWASight, ChoiceWAPortal, ChoiceWAIntervened:
		return ApplyWAChoice(st, slot, kind, cardCode)
	case ChoiceViralNext:
		return ApplyViralChoice(st, slot, kind, cardCode)
	case ChoiceEnPath1, ChoiceEnPath2, ChoiceEnPath3:
		return ApplyEnChoice(st, slot, kind, cardCode)
	case ChoiceTech:
		if !contains(rrsTech, cardCode) {
			return fmt.Errorf("not a TECH upgrade: %q", cardCode)
		}
		pl.Deck[cardCode]++
		pl.Tech = cardCode
	case ChoiceCondition:
		if cardCode != "" {
			if !contains(rrsCond, cardCode) {
				return fmt.Errorf("not a Condition upgrade: %q", cardCode)
			}
			pl.Deck[cardCode]++
			pl.Condition = cardCode
		}
	case ChoiceImprove:
		if cardCode != "" {
			imp := improvedSide(pl.Condition)
			if imp == "" || cardCode != imp {
				return fmt.Errorf("expected %q", imp)
			}
			delete(pl.Deck, pl.Condition)
			pl.Deck[imp]++
			pl.Improved = true
		}
	default:
		return fmt.Errorf("unknown choice kind %q", kind)
	}
	st.ResolvePending(slot, kind)
	return nil
}

// improvedSide maps a basic Condition upgrade to its Improved back face.
func improvedSide(code string) string {
	switch code {
	case "04159a":
		return "04159b"
	case "04160a":
		return "04160b"
	case "04161a":
		return "04161b"
	case "04162a":
		return "04162b"
	}
	return ""
}

// TechUpgrades lists the Hydra Campaign TECH upgrade pool (RRS scenario 1
// victory choices).
func TechUpgrades() []string { return rrsTech }

// ConditionUpgrades lists the Basic Condition upgrade pool (RRS scenario
// 2 victory choices).
func ConditionUpgrades() []string { return rrsCond }

// randInt is a deterministic pseudo-random draw: identical campaign
// states re-report identical results (undo/replay safety).
func randInt(n int) int {
	if n <= 0 {
		return 0
	}
	// FNV-1a over a marker string keeps this import-light and stable.
	h := uint64(14695981039346656037)
	for _, b := range []byte("campaign-rng") {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return int((h >> 33) % uint64(n))
}

func withoutAll(list []string, exclude []string) []string {
	out := make([]string, 0, len(list))
	for _, x := range list {
		if !contains(exclude, x) {
			out = append(out, x)
		}
	}
	return out
}

// ApplySMChoice routes the Sinister Motives picks from ApplyChoice.
