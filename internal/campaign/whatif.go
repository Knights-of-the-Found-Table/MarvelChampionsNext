package campaign

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// What If...? (Amanda Shagoury). Each hero records a What If...? trait,
// rescues trait allies and adds trait cards; the finale scales with the
// recorded campaign flags.
func wiSetup(st *State, ctx *engine.CampaignSetup) {
	shawarma := contains(st.Pool, shawarmaCard)
	switch st.Index {
	case 0:
		ctx.ExtraSets = append(ctx.ExtraSets, setDownToEarth)
		if remaining := withoutAll(smCommunity, st.SMCommunity); len(remaining) > 0 {
			ctx.PreShuffle = append(ctx.PreShuffle, remaining[randInt(len(remaining))])
		}
	case 1:
		ctx.StartSideScheme = saveShawarma
		ctx.ExtraSets = append(ctx.ExtraSets, setGoblinGimmicks, setLegionsHydra)
		// The Justice aspect's option is modelable; the other aspect
		// setups (Aggression engage, Protection glider, Leadership
		// damage) are documented approximations.
		ctx.RevealSideSchemeThreat = true
	case 2:
		ctx.StartSideScheme = captByHydra
		ctx.ExtraSets = append(ctx.ExtraSets, setHydraPatrol, setBeastyBoys)
		if shawarma {
			for i := range st.Players {
				st.Players[i].Deck[shawarmaCard]++
			}
		}
	case 3:
		// Four modular sets from the group's What If...? traits. The
		// printed rule lets the group choose; the campaign picks
		// deterministically from the trait tables (documented
		// approximation).
		chosen := wiPickModularSets(st, 4)
		ctx.ExtraSets = append(ctx.ExtraSets, chosen...)
		for _, s := range chosen {
			if s == setTemporal {
				st.Flags["dinosaurs"] = true
			}
		}
		// One of the seven set-aside sets (Osborn Tech, Temporal,
		// Wrecking Crew) shuffles one random card into the deck.
		setAside := append([]string{}, setOsbornTech, setTemporal, setWreckingMod)
		if card := pickFromSet(setAside[randInt(len(setAside))]); card != "" {
			ctx.PreShuffle = append(ctx.PreShuffle, card)
		}
		ctx.ExtraSets = append(ctx.ExtraSets, setWreckingMod)
	case 4:
		ctx.ExtraSets = append(ctx.ExtraSets, setInfinityGaunt)
		if shawarma {
			for i := range st.Players {
				st.Players[i].Deck[shawarmaCard]++
			}
		}
		if ctx.PlayerAllies == nil {
			ctx.PlayerAllies = map[int][]string{}
		}
		for i := range st.Players {
			if saved := st.Players[i].WIAllies; len(saved) > 0 {
				ctx.PlayerAllies[i] = append(ctx.PlayerAllies[i], saved[0])
			}
		}
		if st.Flags["dinosaurs"] {
			ctx.PoolMinions = append(ctx.PoolMinions, trexMinion)
		}
	}
}

// wiDataTrait maps the campaign trait keys to the printed trait strings.
func wiDataTrait(key string) string {
	if key == "webwarrior" {
		return "web-warrior"
	}
	if key == "shield" {
		return "s.h.i.e.l.d."
	}
	return key
}

// wiPickModularSets samples n distinct sets from the trait options of the
// recorded traits (deterministically).
func wiPickModularSets(st *State, n int) []string {
	var options []string
	seen := map[string]bool{}
	for i := range st.Players {
		for _, s := range whatifTraitSets[st.Players[i].WITrait] {
			if !seen[s] {
				seen[s] = true
				options = append(options, s)
			}
		}
	}
	var out []string
	for len(out) < n && len(options) > 0 {
		idx := randInt(len(options))
		out = append(out, options[idx])
		options = append(options[:idx], options[idx+1:]...)
	}
	return out
}

func wiVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	// Community Service side schemes defeated join the log; each grants
	// another trait-card pick.
	for _, code := range snap.VictoryDisplayCodes {
		if contains(smCommunity, code) && !contains(st.SMCommunity, code) {
			st.SMCommunity = appendUnique(st.SMCommunity, code)
			for i := range st.Players {
				if living(i) && st.Players[i].WITrait != "" {
					st.addPending(i, ChoiceWICard)
				}
			}
		}
	}
	switch st.Index {
	case 0:
		for i := range st.Players {
			if living(i) && st.Players[i].WITrait == "" {
				st.addPending(i, ChoiceWITrait)
			}
		}
	case 1:
		// The Shawarma Place defeated (not in play) feeds the campaign
		// Shawarma pool; a Damaged Avengers Tower is recorded.
		if !contains(snap.SideSchemeBaseCodes, "21182") {
			st.Pool = appendUnique(st.Pool, shawarmaCard)
			st.Flags["shawarma"] = true
		}
		if contains(snap.EnvironmentCodes, "21100") {
			st.Flags["towerDamaged"] = true
		}
	case 2:
		// Each player picks another trait ally.
		for i := range st.Players {
			if living(i) && st.Players[i].WITrait != "" {
				st.addPending(i, ChoiceWIAlly)
			}
		}
	case 3:
		if snap.MainStage >= 3 {
			st.Flags["crime"] = true
		}
	case 4:
		// Ultron defeated: the campaign is won.
	}
	st.recordExpertHP(snap, living)
	st.Advance()
}

// ApplyWIChoice resolves the What If...? interlude picks.
func ApplyWIChoice(st *State, slot int, kind, cardCode string) error {
	pl := st.Slot(slot)
	if pl == nil {
		return fmt.Errorf("unknown slot %d", slot)
	}
	switch kind {
	case ChoiceWITrait:
		if !contains(whatifTraits, cardCode) {
			return fmt.Errorf("not a What If...? trait: %q", cardCode)
		}
		pl.WITrait = cardCode
		st.addPending(slot, ChoiceWIAlly)
	case ChoiceWIAlly:
		def, ok := engine.DB.Lookup(cardCode)
		if !ok || def.Type != "ally" || !def.HasTrait(wiDataTrait(pl.WITrait)) {
			return fmt.Errorf("not a %s ally: %q", pl.WITrait, cardCode)
		}
		pl.Deck[cardCode]++
		pl.WIAllies = appendUnique(pl.WIAllies, cardCode)
	case ChoiceWICard:
		def, ok := engine.DB.Lookup(cardCode)
		if !ok || def.Category != "player" || !def.HasTrait(wiDataTrait(pl.WITrait)) {
			return fmt.Errorf("not a %s card: %q", pl.WITrait, cardCode)
		}
		qty := def.Quantity
		if qty <= 0 {
			qty = 1
		}
		pl.Deck[cardCode] += qty
		pl.WIRewards = appendUnique(pl.WIRewards, cardCode)
	default:
		return fmt.Errorf("unknown choice kind %q", kind)
	}
	st.ResolvePending(slot, kind)
	return nil
}

// wiRouteIndex maps the trait names for display.
func wiTraitLabel(t string) string { return t }

// pickFromSet returns one random card code from an encounter set.
func pickFromSet(set string) string {
	cards := engine.DB.InSet(set)
	if len(cards) == 0 {
		return ""
	}
	return cards[randInt(len(cards))].Code
}
