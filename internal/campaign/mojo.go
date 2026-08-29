package campaign

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// House of Mojo (Amanda Shagoury). A Mojo-produced MCU show: each player
// picks a role (condition upgrade, market card, MG Skill), the group
// attempts a training player side scheme, and the story runs En Sabah Nur
// → Magneto → Thanos → the Sentinels → Mojo.
func mojoSetup(st *State, ctx *engine.CampaignSetup) {
	roleConds := func() {
		for i := range st.Players {
			if r, ok := mojoRoles[st.Players[i].MojoRole]; ok {
				addRoleUpgrade(ctx, i, r.Condition)
			}
		}
	}
	trainingScheme := st.Selections["training"]
	switch st.Index {
	case 0:
		ctx.ExtraSets = append(ctx.ExtraSets, setClanAkkaba)
		// One NX setting set at random; the optional Hellfire Club joins
		// (its "Disbanded" check feeds chapter 2).
		ctx.ExtraSets = append(ctx.ExtraSets, []string{"super_strength", "telepathy", "flight"}[randInt(3)])
		ctx.ExtraSets = append(ctx.ExtraSets, setHellfire)
		roleConds()
		ctx.PlayerSideScheme = trainingScheme
	case 1:
		ctx.ExtraSets = append(ctx.ExtraSets, setSavageLand, setSauron)
		ctx.PoolSupports = append(ctx.PoolSupports, xjetMG)
		roleConds()
		if st.Flags["training"] {
			// The earned environment side is absent from the data
			// snapshot; the player-side scheme itself returns to play
			// (documented approximation).
			ctx.PlayerSideScheme = trainingScheme
		}
		if st.Flags["hellfire"] {
			// Each player adds one recorded Scenario #1 event to hand.
			for i := range st.Players {
				if e := st.Players[i].MojoEvent; e != "" {
					if ctx.HandFetch == nil {
						ctx.HandFetch = map[int]string{}
					}
					ctx.HandFetch[i] = e
				}
			}
		}
	case 2:
		ctx.ExtraSets = append(ctx.ExtraSets, setLegionsHydra, setHydraAssault)
		ctx.StartEnvironment = smPublicOutcry
		ctx.PreShuffle = append(ctx.PreShuffle, smSmearCampaign)
		roleConds()
		if st.Flags["training"] {
			ctx.PlayerSideScheme = trainingScheme
		}
		ctx.PoolUpgrades = append(ctx.PoolUpgrades, childrenAtom)
		if st.IsExpert() && st.Flags["advanced1"] {
			ctx.MainSchemeThreat += len(st.Players)
		}
	case 3:
		ctx.ExtraSets = append(ctx.ExtraSets, setZeroTolerance, setDystopian, setGenosha)
		ctx.PoolSupports = append(ctx.PoolSupports, xjetMG, utopiaSupport())
		roleConds()
		// Build Support joins as the group's player side scheme (the
		// earned training environment would need the second slot; the
		// env printings are absent from the data snapshot anyway).
		ctx.PlayerSideScheme = "40027"
		ctx.MissionScheme = findLostMutant
	case 4:
		// Sitcom plus one genre set per player.
		ctx.ExtraSets = append(ctx.ExtraSets, setMojoSitcom)
		genres := []string{setMojoFantasy, setMojoScifi, setMojoHorror, setMojoWestern, setMojoSpiral}
		for range st.Players {
			ctx.ExtraSets = append(ctx.ExtraSets, genres[randInt(len(genres))])
		}
		roleConds()
		if st.Flags["training"] {
			ctx.PlayerSideScheme = trainingScheme
		}
		if st.Flags["genosha"] {
			if sch := st.Players[0].MojoScheme; sch != "" {
				ctx.PlayerSideScheme = sch
			}
		}
		if ctx.PlayerAllies == nil {
			ctx.PlayerAllies = map[int][]string{}
		}
		ctx.PlayerAllies[0] = append(ctx.PlayerAllies[0], longshotAlly)
	}
}

// utopiaSupport picks a Utopia printing present in the data snapshot.
func utopiaSupport() string {
	if _, ok := engine.DB.Lookup(utopiaCyclops); ok {
		return utopiaCyclops
	}
	return utopiaStorm
}

func mojoVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	switch st.Index {
	case 0:
		// The training side scheme counts as attempted when it is no
		// longer in play; the Hellfire Club's fate is not observable, so
		// its flag is marked whenever the set was added (documented
		// approximation).
		if t := st.Selections["training"]; t != "" && !contains(snap.SideSchemeBaseCodes, data.BaseCode(t)) {
			st.Flags["training"] = true
		}
		st.Flags["hellfire"] = true
	case 1:
		if snap.MainStage >= 2 {
			st.Flags["advanced2"] = true
		} else {
			st.Flags["advanced1"] = true
		}
	case 2:
		// The team has gotten stronger when the Infinity Stones never
		// reached Balance the Scales.
		if snap.MainStage < 2 {
			st.Flags["stronger"] = true
		}
		for i := range st.Players {
			if living(i) {
				st.addPending(i, ChoiceMojoMarket)
			}
		}
	case 3:
		for i := range st.Players {
			if living(i) {
				st.addPending(i, ChoiceMojoScheme)
			}
		}
	case 4:
		// Mojo defeated: the campaign is won.
	}
	st.recordExpertHP(snap, living)
	st.Advance()
}

// ApplyMojoChoice resolves the House of Mojo interlude picks.
func ApplyMojoChoice(st *State, slot int, kind, cardCode string) error {
	pl := st.Slot(slot)
	if pl == nil {
		return fmt.Errorf("unknown slot %d", slot)
	}
	switch kind {
	case ChoiceMojoRole:
		if _, ok := mojoRoles[cardCode]; !ok {
			return fmt.Errorf("not a role: %q", cardCode)
		}
		pl.MojoRole = cardCode
	case ChoiceMojoTraining:
		def, ok := engine.DB.Lookup(cardCode)
		if !ok || def.Type != "player_side_scheme" {
			return fmt.Errorf("not a player side scheme: %q", cardCode)
		}
		st.Selections["training"] = cardCode
	case ChoiceMojoEvent:
		def, ok := engine.DB.Lookup(cardCode)
		if !ok || def.Type != "event" {
			return fmt.Errorf("not an event: %q", cardCode)
		}
		qty := def.Quantity
		if qty <= 0 || qty > 3 {
			qty = 3
		}
		pl.Deck[cardCode] += qty
		pl.MojoEvent = cardCode
	case ChoiceMojoMarket:
		if cardCode != shawarmaCard && cardCode != mojoRoles[pl.MojoRole].Market {
			return fmt.Errorf("not Shawarma or the role's market card: %q", cardCode)
		}
		pl.Deck[cardCode]++
		pl.MojoMarket = cardCode
	case ChoiceMojoScheme:
		def, ok := engine.DB.Lookup(cardCode)
		if !ok || def.Type != "player_side_scheme" {
			return fmt.Errorf("not a player side scheme: %q", cardCode)
		}
		pl.Deck[cardCode]++
		pl.MojoScheme = cardCode
	default:
		return fmt.Errorf("unknown choice kind %q", kind)
	}
	st.ResolvePending(slot, kind)
	return nil
}
