package campaign

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// BuildGame builds the engine options for the campaign's current chapter
// from the campaign log (rulebook "Campaign Instructions — Setup"). The
// per-player deck snapshots carry every campaign addition; obligations and
// starting hit points apply the expert rules.
func BuildGame(st *State, seed int64) (engine.NewGameOptions, error) {
	box := st.BoxDef()
	if box == nil {
		return engine.NewGameOptions{}, fmt.Errorf("unknown campaign box %q", st.Box)
	}
	if st.Index < 0 || st.Index >= len(box.Scenarios) {
		return engine.NewGameOptions{}, fmt.Errorf("scenario index %d out of range", st.Index)
	}
	if len(st.PendingChoices) > 0 {
		return engine.NewGameOptions{}, fmt.Errorf("interlude choices are still pending")
	}
	// Campaign deckbuilding restrictions (Avenger-only, no Guardians...).
	if box.HeroOK != nil {
		for i := range st.Players {
			if err := box.HeroOK(st.Players[i].HeroBase); err != nil {
				return engine.NewGameOptions{}, fmt.Errorf("player %d: %w", i+1, err)
			}
		}
	}
	opts := engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: box.Scenarios[st.Index].ID,
		Difficulty: st.Difficulty,
	}
	ctx := &engine.CampaignSetup{}
	switch st.Box {
	case "rrs":
		buildRRS(st, ctx)
	case "mts":
		mtsSetup(st, ctx, &opts)
	case "sm":
		smSetup(st, ctx, &opts)
	case "mg":
		mgSetup(st, ctx, &opts)
	case "nx":
		nxSetup(st, ctx, &opts)
	case "aoa":
		aoaSetup(st, ctx, &opts)
	case "aos":
		aosSetup(st, ctx, &opts)
	case "gmw":
		if err := buildGMW(st, ctx); err != nil {
			return opts, err
		}
	// Contest campaigns.
	case "cowl":
		cowlSetup(st, ctx)
	case "whatif":
		wiSetup(st, ctx)
	case "awesome":
		awSetup(st, ctx)
	case "alias":
		aliasSetup(st, ctx)
	case "watchers":
		if err := watchersValidate(st); err != nil {
			return opts, err
		}
		watchersSetup(st, ctx)
	case "mojo":
		mojoSetup(st, ctx)
	case "bord":
		if err := bordSetup(st, ctx, &opts); err != nil {
			return opts, err
		}
	case "night":
		if err := nightValidate(st); err != nil {
			return opts, err
		}
		nightSetup(st, ctx)
	case "viral":
		viralSetup(st, ctx)
	case "entropy":
		entSetup(st, ctx, &opts)
	}
	opts.Campaign = ctx
	for i := range st.Players {
		pl := &st.Players[i]
		spec := engine.PlayerSpec{
			Name:     pl.Name,
			UserID:   fmt.Sprintf("%d", pl.UserID),
			HeroBase: pl.HeroBase,
			Deck:     copyDeck(pl.Deck),
		}
		// Expert persistent damage, and the heal option chosen in the
		// interlude. The cost is paid when the flag is consumed (GMW:
		// one unit; RRS: a random obligation joins the deck).
		if st.IsExpert() {
			if pl.HealNext {
				pl.HP = 0
				pl.HealNext = false
				switch st.Box {
				case "gmw":
					if pl.Units < 1 {
						return opts, fmt.Errorf("player %d cannot pay the healing unit", i)
					}
					pl.Units--
				case "rrs":
					spec.DeckEncounters = append(spec.DeckEncounters, rrsObligations[(st.Index+i)%len(rrsObligations)])
				}
				// mts: the cost is an acceleration token on the main
				// scheme, applied via the campaign context below.
			}
			if pl.HP > 0 {
				spec.StartHP = pl.HP
			}
		} else {
			pl.HealNext = false
		}
		// Red Skull expert: players recorded as engaged with an enemy
		// deal themselves an encounter card.
		if st.Box == "rrs" && st.Index == 4 && st.IsExpert() && pl.EngagedEnemy {
			ctx.DealEncounter = append(ctx.DealEncounter, i)
		}
		if i == 0 && ctx.ObligationFirstPlayer != "" {
			spec.DeckEncounters = append(spec.DeckEncounters, ctx.ObligationFirstPlayer)
		}
		if pl.SetupHand != "" {
			if ctx.HandFetch == nil {
				ctx.HandFetch = map[int]string{}
			}
			ctx.HandFetch[i] = pl.SetupHand
		}
		opts.Players = append(opts.Players, spec)
	}
	// Deadpool's Game Night: the reward pool is dealt into the copied
	// decks (the log keeps the pool itself).
	if st.Box == "night" {
		nightDealPool(st, opts.Players)
	}
	return opts, nil
}

func copyDeck(src map[string]int) map[string]int {
	out := make(map[string]int, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// buildRRS applies the Rise of Red Skull campaign setup: recorded
// experimental weapons shuffle back in from scenario 2 on; Red Skull
// opens with the recorded delay counters (1 in an expert campaign).
func buildRRS(st *State, ctx *engine.CampaignSetup) {
	if st.Index >= 1 {
		ctx.PreShuffle = append(ctx.PreShuffle, st.Experimental...)
	}
	if st.Index == 4 {
		if st.IsExpert() {
			ctx.MainSchemeThreat = 1
		} else {
			ctx.MainSchemeThreat = st.DelayCounters
		}
	}
}

// buildGMW applies the Galaxy's Most Wanted campaign setup: the challenge
// side schemes, the Badoon Headhunter modular with its mark-gated cards,
// the recorded Galactic Artifacts with their printed setup riders, and
// the Ronan finale's Power Stone / evasion bookkeeping.
func buildGMW(st *State, ctx *engine.CampaignSetup) error {
	marks := st.headhunterMarks()
	shuffle := []string{gmwHeadhunter}
	if marks >= 1 {
		shuffle = append(shuffle, gmwOnTheHunt)
	}
	if marks >= 2 {
		shuffle = append(shuffle, gmwDeadToRights)
	}
	if marks >= 3 {
		shuffle = append(shuffle, gmwHenchman)
	}
	switch st.Index {
	case 0:
		ctx.StartSideScheme = gmwBlitz
		ctx.PreShuffle = shuffle
		if st.IsExpert() {
			ctx.MillMinionEngage = true
		}
	case 1:
		ctx.StartSideScheme = gmwGallery
		ctx.PreShuffle = shuffle
		ctx.CollectionHandCard = true
	case 2:
		ctx.StartSideScheme = gmwNoEscape
		ctx.PreShuffle = shuffle
		ctx.RemoveFromGame = st.Collection
		ctx.DrawUpAfterRemove = true
		ctx.MillRevealAttachment = true
	case 3:
		ctx.StartSideScheme = gmwGuerrilla
		ctx.PreShuffle = append(append([]string{}, st.Artifacts...), shuffle...)
		for _, code := range st.Artifacts {
			switch code {
			case "16127": // Hujahdarian Monarch Egg
				ctx.ShipEvasion++
			case "16128": // Magical Teapot
				ctx.FirstPlayerEncounterFacedown = true
			case "16129": // Philosopher's Stone
				ctx.VillainBoostFacedown = true
			case "16130": // Crystal Ball
				ctx.VillainTough = true
			default:
				return fmt.Errorf("unknown galactic artifact %q", code)
			}
		}
		ctx.MillRevealAttachment = true
	case 4:
		ctx.StartSideScheme = gmwSupremacy
		ctx.PreShuffle = append([]string{gmwFugitive, gmwPincer}, shuffle...)
		ctx.SideSchemeThreat = map[string]int{gmwPincer: 3 * st.Evasion}
		if st.PowerStone >= 0 {
			if ctx.DealCard == nil {
				ctx.DealCard = map[int]string{}
			}
			ctx.DealCard[st.PowerStone] = gmwAccused
		}
		if st.IsExpert() {
			ctx.MainSchemeThreat = 1
		}
	default:
		return fmt.Errorf("scenario index %d out of range", st.Index)
	}
	return nil
}

func (st *State) headhunterMarks() int {
	n := 0
	for _, m := range st.Headhunter {
		if m {
			n++
		}
	}
	return n
}

// MarketCards lists the GMW Market cards with their parsed unit costs,
// cheapest first.
func MarketCards() []MarketCard {
	var out []MarketCard
	for _, def := range engine.DB.InSet(gmwMarketSet) {
		if def.UnitCost <= 0 {
			continue
		}
		out = append(out, MarketCard{Code: def.Code, Name: def.EName, Cost: def.UnitCost})
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Cost < out[j-1].Cost; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// MarketCard is one shop entry.
type MarketCard struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Cost int    `json:"cost"`
}

// SpendOrBuyMarket routes the shared shop endpoint: units for the GMW
// Market, Guardian Influence for the Awesome Campaign.
func SpendOrBuyMarket(st *State, slot int, code string) error {
	switch st.Box {
	case "awesome":
		return SpendInfluence(st, slot, code)
	default:
		return BuyMarket(st, slot, code)
	}
}

// BuyMarket spends units on a Market card (one copy per campaign for the
// group; the card stays in that player's deck for the rest of the
// campaign).
func BuyMarket(st *State, slot int, code string) error {
	if st.Box != "gmw" {
		return fmt.Errorf("the Market belongs to the Galaxy's Most Wanted campaign")
	}
	var card *MarketCard
	for _, mc := range MarketCards() {
		if mc.Code == code {
			c := mc
			card = &c
			break
		}
	}
	if card == nil {
		return fmt.Errorf("not a Market card: %q", code)
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
	if pl.Units < card.Cost {
		return fmt.Errorf("not enough units: %d < %d", pl.Units, card.Cost)
	}
	pl.Units -= card.Cost
	pl.Market = append(pl.Market, code)
	pl.Deck[code]++
	return nil
}

// SetHeal toggles the "heal to printed HP at the next scenario's setup"
// option (GMW: 1 unit; RRS expert: a random obligation).
func SetHeal(st *State, slot int, on bool) error {
	pl := st.Slot(slot)
	if pl == nil {
		return fmt.Errorf("unknown slot %d", slot)
	}
	if !on {
		pl.HealNext = false
		return nil
	}
	if !st.IsExpert() {
		return fmt.Errorf("healing applies only in expert campaigns")
	}
	if st.Box == "gmw" && pl.Units < 1 {
		return fmt.Errorf("healing costs 1 unit")
	}
	pl.HealNext = true
	return nil
}
