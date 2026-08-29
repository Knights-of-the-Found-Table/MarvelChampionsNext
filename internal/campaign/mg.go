package campaign

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// The Mutant Genesis campaign: player roles (Brawler/Commander/Defender/
// Peacekeeper) with earned one-shot Skill upgrades, and the time-travelling
// Future Past deck whose cards churn between the campaign log and the
// encounter deck.
var (
	mgRoles = map[string][]string{
		"brawler":     {"32176", "32177", "32178", "32179", "32180"},
		"commander":   {"32181", "32182", "32183", "32184", "32185"},
		"defender":    {"32186", "32187", "32188", "32189", "32190"},
		"peacekeeper": {"32191", "32192", "32193", "32194", "32195"},
	}
	mgAllRoles      = []string{"brawler", "commander", "defender", "peacekeeper"}
	mgFuturePastAll = []string{"32166", "32167", "32168", "32169", "32170"}
	mgJubilee       = "32088b"
	mgCaptiveAllies = []string{"32089", "32090", "32091", "32092"}
	// Per scenario: the campaign side scheme revealed at setup (its
	// defeat earns the role upgrades for the NEXT chapter).
	mgSchemeByIndex = []string{"32171a", "32172a", "32173a", "32174a", "32175a"}
)

// ChoiceMGRole is picked during the interlude before chapter 1.
const ChoiceMGRole = "mg-role"

// ApplyMGRoleChoice records a player's role (validated once).
func ApplyMGRoleChoice(st *State, slot int, role string) error {
	pl := st.Slot(slot)
	if pl == nil {
		return fmt.Errorf("unknown slot %d", slot)
	}
	if _, ok := mgRoles[role]; !ok {
		return fmt.Errorf("unknown role %q", role)
	}
	pl.MGRole = role
	st.ResolvePending(slot, ChoiceMGRole)
	return nil
}

// mgVictory implements the five victory programs (pages 7-19).
func mgVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	// Role upgrade earned for the next chapter when this scenario's
	// campaign side scheme was defeated.
	if st.Index < len(mgSchemeByIndex) && !contains(snap.SideSchemeBaseCodes, mgSchemeByIndex[st.Index]) {
		st.Flags["mgEarned"] = true
	}
	// Future Past churn: cards defeated into the victory display leave
	// the campaign; the rest return to the log.
	kept := st.MGFuturePast[:0]
	for _, code := range st.MGFuturePast {
		if !contains(snap.VictoryDisplayCodes, code) {
			kept = append(kept, code)
		}
	}
	st.MGFuturePast = kept
	switch st.Index {
	}
	// Jubilee (chapters 2-4): recorded while she is in play, removed
	// from the campaign otherwise.
	if st.Index >= 1 && st.Index <= 3 {
		st.Flags["jubilee"] = contains(snap.MinionCodes, mgJubilee) || contains(snap.VillainCodes, mgJubilee)
	}
	// Captive allies (recorded from chapter 2 on) are tracked; allies
	// tucked under Find the Prisoners (chapter 3) leave the campaign.
	if st.Index == 2 {
		for _, c := range snap.CaptiveAllyCodes {
			st.MGCaptives = appendUnique(st.MGCaptives, c)
		}
		for _, code := range snap.PrisonStoredCodes {
			st.MGRemovedAllies = appendUnique(st.MGRemovedAllies, code)
			for i := range st.Players {
				delete(st.Players[i].Deck, code)
			}
		}
	}
	st.recordExpertHP(snap, living)
	st.Advance()
}

// mgSetup applies the setups (pages 7-19).
func mgSetup(st *State, ctx *engine.CampaignSetup, opts *engine.NewGameOptions) {
	// Future Past deck: logged cards shuffle into the encounter deck
	// (the remainder stays aside as the deck).
	ctx.PreShuffle = append(ctx.PreShuffle, st.MGFuturePast...)
	// Jubilee.
	if st.Flags["jubilee"] {
		ctx.PoolAllies = append(ctx.PoolAllies, mgJubilee)
	}
	// Recorded CAPTIVE allies may be shuffled into any player's deck
	// (approximation: the first player's).
	for _, code := range st.MGCaptives {
		if !contains(st.MGRemovedAllies, code) {
			st.Players[0].Deck[code]++
		}
	}
	// Earned role upgrades: one random upgrade per player, in play.
	if st.Flags["mgEarned"] || st.Index == 0 {
		for i := range st.Players {
			role := st.Players[i].MGRole
			if role == "" {
				continue
			}
			set := mgRoles[role]
			pick := set[randInt(len(set))]
			if ctx.RoleUpgrades == nil {
				ctx.RoleUpgrades = map[int][]string{}
			}
			ctx.RoleUpgrades[i] = []string{pick}
		}
		st.Flags["mgEarned"] = false
	}
	// This chapter's campaign side scheme.
	if st.Index < len(mgSchemeByIndex) {
		ctx.StartSideScheme = mgSchemeByIndex[st.Index]
	}
	// Expert heal: acceleration token (same plumbing as MTS).
	if st.IsExpert() && st.Index >= 1 {
		for i := range st.Players {
			if st.Players[i].HealNext {
				ctx.MainSchemeAcceleration++
			}
		}
	}
}

// FuturePastSeed exposes the full Future Past set for campaign init.
func FuturePastSeed() []string { return append([]string{}, mgFuturePastAll...) }
