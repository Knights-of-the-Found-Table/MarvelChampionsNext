package campaign

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// Awesome Campaign Vol. 1 (Steele Hull). Guardian Influence buys cards
// before the finale; scenario results feed M.O.D.O.K., delay counters,
// the last Wrecker and the Sleeper into chapter 5.
func awSetup(st *State, ctx *engine.CampaignSetup) {
	switch st.Index {
	case 0:
		ctx.ExtraSets = append(ctx.ExtraSets, setDoomsday, setPowerStone, setShipCommand)
		if ctx.PlayerAllies == nil {
			ctx.PlayerAllies = map[int][]string{}
		}
		if ctx.RoleUpgrades == nil {
			ctx.RoleUpgrades = map[int][]string{}
		}
		// Market upgrades: one Campaign unit-cost-4 card per player by
		// seat rotation (the unit-cost-6 card is a documented
		// approximation).
		market4 := []string{"16162", "16163", "16164", "16165"}
		for i := range st.Players {
			upgrades := []string{honoraryGuard}
			if _, ok := engine.DB.Lookup(market4[i%len(market4)]); ok {
				upgrades = append(upgrades, market4[i%len(market4)])
			}
			ctx.RoleUpgrades[i] = upgrades
			if code := st.Players[i].AWAlly; code != "" {
				ctx.PlayerAllies[i] = append(ctx.PlayerAllies[i], code)
			}
			if code := st.Players[i].AWIdentity; code != "" {
				if ctx.HandFetch == nil {
					ctx.HandFetch = map[int]string{}
				}
				ctx.HandFetch[i] = code
			}
		}
	case 1:
		ctx.StartSideScheme = unnaturalStorm
		ctx.ExtraSets = append(ctx.ExtraSets, setFrostGiants)
		ctx.MainSchemeThreat += 3 * len(st.Players)
	case 2:
		// Delay counters from chapter 2 deal each player encounter cards.
		for i := range st.Players {
			for k := 0; k < st.Counters["delay"]/5; k++ {
				ctx.DealEncounter = append(ctx.DealEncounter, i)
			}
		}
		if card := pickFromSet(setExperimental); card != "" {
			ctx.PreShuffle = append(ctx.PreShuffle, card)
		}
	case 3:
		ctx.ExtraSets = append(ctx.ExtraSets, setWreckingMod, "children_of_thanos")
		// The recorded last Wrecker searches the deck; any minion is the
		// documented approximation.
		ctx.MillMinionEngage = true
	case 4:
		ctx.ExtraSets = append(ctx.ExtraSets, setInfinityGaunt, setKreeFanatic, setDoomsday)
		if st.Flags["modok"] {
			ctx.PoolMinions = append(ctx.PoolMinions, modokMinion)
		}
		if st.Flags["sleeper"] {
			ctx.PreShuffle = append(ctx.PreShuffle, sleepSide)
		}
		if ctx.PlayerAllies == nil {
			ctx.PlayerAllies = map[int][]string{}
		}
		for i := range st.Players {
			if code := st.Players[i].AWAlly; code != "" {
				ctx.PlayerAllies[i] = append(ctx.PlayerAllies[i], code)
			}
		}
	}
}

func awVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	// Guardian Influence: +1 per living player per met condition.
	gain := 0
	anyAlive := false
	for i := range st.Players {
		if living(i) {
			anyAlive = true
		}
	}
	if anyAlive {
		gain++
	}
	if snap.NoMinions {
		gain++
	}
	if len(snap.SideSchemeBaseCodes) == 0 {
		gain++
	}
	if snap.MainThreat == 0 {
		gain++
	}
	if snap.Acceleration < len(st.Players) {
		gain++
	}
	for i := range st.Players {
		if living(i) {
			st.Players[i].Influence += gain
		}
	}
	switch st.Index {
	case 0:
		st.Flags["modok"] = contains(snap.MinionCodes, modokMinion)
	case 1:
		st.Counters["delay"] = snap.DelayCounters
	case 2:
		// "The last villain defeated" is not observable in the snapshot;
		// record a deterministic pick (documented approximation).
		st.Selections["lastVillain"] = wreckers[randInt(len(wreckers))]
	case 3:
		st.Flags["sleeper"] = !contains(snap.VictoryDisplayCodes, sleepSide)
	case 4:
		// Thanos defeated: the campaign is won.
	}
	st.recordExpertHP(snap, living)
	st.Advance()
}

// AwesomeRewards lists the shop entries present in the data snapshot.
func AwesomeRewards() []awesomeReward {
	var out []awesomeReward
	for _, r := range awesomeRewards {
		if _, ok := engine.DB.Lookup(r.Code); ok {
			out = append(out, r)
		}
	}
	return out
}

// ApplyAWAlly records the player's basic GUARDIAN ally (chapter 1 and the
// finale put it into play).
func ApplyAWAlly(st *State, slot int, cardCode string) error {
	pl := st.Slot(slot)
	if pl == nil {
		return fmt.Errorf("unknown slot %d", slot)
	}
	def, ok := engine.DB.Lookup(cardCode)
	if !ok || def.Type != "ally" || !def.HasTrait("guardian") {
		return fmt.Errorf("not a GUARDIAN ally: %q", cardCode)
	}
	if a := def.Aspect; a != "" && a != "basic" {
		return fmt.Errorf("not a basic ally: %q", cardCode)
	}
	if pl.AWAlly != "" {
		return fmt.Errorf("a guardian ally is already recorded")
	}
	pl.AWAlly = cardCode
	st.ResolvePending(slot, ChoiceAWAlly)
	return nil
}

// ApplyAWIdentity records the identity-specific card fetched to the
// opening hand (chapter 1 setup).
func ApplyAWIdentity(st *State, slot int, cardCode string) error {
	pl := st.Slot(slot)
	if pl == nil {
		return fmt.Errorf("unknown slot %d", slot)
	}
	if pl.Deck[cardCode] <= 0 {
		return fmt.Errorf("card not in this player's deck: %q", cardCode)
	}
	pl.AWIdentity = cardCode
	st.ResolvePending(slot, ChoiceAWIdentity)
	return nil
}

// SpendInfluence buys one Awesome Campaign reward card.
func SpendInfluence(st *State, slot int, code string) error {
	if st.Box != "awesome" {
		return fmt.Errorf("Guardian Influence belongs to the Awesome Campaign")
	}
	var rew *awesomeReward
	for _, r := range AwesomeRewards() {
		if r.Code == code {
			c := r
			rew = &c
			break
		}
	}
	if rew == nil {
		return fmt.Errorf("not an influence reward: %q", code)
	}
	for i := range st.Players {
		if contains(st.Players[i].Market, code) {
			return fmt.Errorf("%q was already bought this campaign", code)
		}
	}
	pl := st.Slot(slot)
	if pl == nil {
		return fmt.Errorf("unknown slot %d", slot)
	}
	if pl.Influence < rew.Cost {
		return fmt.Errorf("not enough influence: %d < %d", pl.Influence, rew.Cost)
	}
	pl.Influence -= rew.Cost
	pl.Market = append(pl.Market, code)
	pl.Deck[code]++
	return nil
}
