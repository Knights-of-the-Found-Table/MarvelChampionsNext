package campaign

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// She-Hulk vs. Deadpool's Game Night (Kurt Hake). One player must be
// She-Hulk; Deadpool's 'Pool cards cycle through the players' decks, and
// every metagame challenge or loss shifts the finale.
func nightValidate(st *State) error {
	sh := 0
	for i := range st.Players {
		if st.Players[i].HeroBase == heroSheHulk {
			sh++
		}
	}
	if sh != 1 {
		return fmt.Errorf("exactly one player must be She-Hulk (found %d)", sh)
	}
	return nil
}

// sheHulkSlot finds the She-Hulk seat (-1 when absent).
func sheHulkSlot(st *State) int {
	for i := range st.Players {
		if st.Players[i].HeroBase == heroSheHulk {
			return i
		}
	}
	return -1
}

// nightDealPool distributes the reward pool across the decks (BuildGame
// copies the decks, so this never touches the campaign log).
func nightDealPool(st *State, specs []engine.PlayerSpec) {
	n := len(st.Players)
	if n == 0 {
		return
	}
	for i, code := range st.Pool {
		specs[i%n].Deck[code]++
	}
}

func nightSetup(st *State, ctx *engine.CampaignSetup) {
	sh := sheHulkSlot(st)
	if ctx.PlayerAllies == nil {
		ctx.PlayerAllies = map[int][]string{}
	}
	if ctx.DealCard == nil {
		ctx.DealCard = map[int]string{}
	}
	switch st.Index {
	case 0:
		ctx.ExtraSets = append(ctx.ExtraSets, setDeathstrike, setReavers)
		if sh >= 0 {
			if ctx.PlayerAllies == nil {
				ctx.PlayerAllies = map[int][]string{}
			}
			ctx.PlayerAllies[sh] = append(ctx.PlayerAllies[sh], deadpoolAlly)
			st.Players[sh].Deck["44058"]++ // War
			ctx.DealCard[sh] = deathstrikeMn
		}
	case 1:
		ctx.ExtraSets = append(ctx.ExtraSets, setSauron, setMojoFantasy)
		ctx.StartEnvironment = gameOfMojos
		ctx.StartSideScheme = sauronLives
	case 2:
		ctx.ExtraSets = append(ctx.ExtraSets, setMojoScifi)
		ctx.StartEnvironment = mojoRunner
		ctx.StartSideScheme = "39059" // ICE-Teroid M
	case 3:
		ctx.ExtraSets = append(ctx.ExtraSets, setClanAkkaba, setMojoHorror)
		ctx.StartEnvironment = mojoFiles
		ctx.PoolMinions = append(ctx.PoolMinions, "39049") // Cultist
	case 4:
		ctx.ExtraSets = append(ctx.ExtraSets, setMojoWestern, setGalacticArt)
		if sh >= 0 {
			if ctx.PlayerAllies == nil {
				ctx.PlayerAllies = map[int][]string{}
			}
			ctx.PlayerAllies[sh] = append(ctx.PlayerAllies[sh], deadpoolAlly)
			ctx.DealCard[sh] = cardSharkMn
		}
		// One modular-set penalty per game Deadpool won (chapters 2-4;
		// chapter 1's War game carries no set penalty).
		if st.Selections["gw2"] == "deadpool" {
			ctx.ExtraSets = append(ctx.ExtraSets, "flight")
		}
		if st.Selections["gw3"] == "deadpool" {
			ctx.ExtraSets = append(ctx.ExtraSets, "telepathy")
		}
		if st.Selections["gw4"] == "deadpool" {
			ctx.ExtraSets = append(ctx.ExtraSets, "super_strength")
		}
		if st.Selections["gw1"] == "deadpool" {
			ctx.StartSideSchemes = append(ctx.StartSideSchemes, magicalTeapot)
		}
		if st.Selections["gw2"] == "deadpool" {
			ctx.StartSideSchemes = append(ctx.StartSideSchemes, crystalBall)
		}
		if st.Selections["gw3"] == "deadpool" {
			ctx.StartSideSchemes = append(ctx.StartSideSchemes, monarchEgg)
		}
		if st.Selections["gw4"] == "deadpool" {
			ctx.StartSideSchemes = append(ctx.StartSideSchemes, philStone)
		}
	}
	// Strength of the Alliance levels 1-8 (draws, max HP, trait grants,
	// ally limit, boost suppression, threat bonuses) are recurring
	// in-game rules the setup pipeline cannot express — documented
	// approximations; the level total still gates the finale's penalties.
}

func nightVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	chapter := st.Index
	st.Counters["chapter"] = chapter
	// The metagame challenge and teamwork goal are self-reported; Deadpool
	// takes the game by default until the host claims the challenge.
	st.Selections[fmt.Sprintf("gw%d", chapter+1)] = "deadpool"
	switch st.Index {
	case 0:
		// +1 Alliance for simply winning is the teamwork goal here.
		st.Counters["alliance"]++
		st.Pool = append(st.Pool, nightCh1Win...)
		st.addPending(0, ChoiceNTMeta)
	case 1:
		st.Pool = append(st.Pool, nightCh2Add...)
		nightPickAdds(st, nightCh2Picks)
		st.addPending(0, ChoiceNTMeta)
		st.addPending(0, ChoiceNTTeam)
	case 2:
		nightPickAdds(st, nightCh3Picks)
		st.addPending(0, ChoiceNTMeta)
		st.addPending(0, ChoiceNTTeam)
	case 3:
		nightPickAdds(st, nightCh4Picks)
		st.addPending(0, ChoiceNTMeta)
		st.addPending(0, ChoiceNTTeam)
	case 4:
		// The Collector beaten: the campaign is won.
	}
	st.recordExpertHP(snap, living)
	st.Advance()
}

// nightPickAdds adds one deterministic distinct pick per player to the
// reward pool (the printed "each player selects a different card" is a
// group table moment; the campaign records it automatically).
func nightPickAdds(st *State, options []string) {
	used := map[string]bool{}
	for range st.Players {
		for _, code := range options {
			if !used[code] && !contains(st.Pool, code) {
				used[code] = true
				st.Pool = append(st.Pool, code)
				break
			}
		}
	}
}

// nightDefeat folds a lost game night: no replay — Deadpool scores and
// the story moves on (with a Time Stone rewind on the finale).
func nightDefeat(st *State) {
	switch st.Index {
	case 0:
		st.Pool = append(st.Pool, nightCh1Lose...)
	case 4:
		st.Counters["rewinds"]++
		st.Status = "interlude"
		st.LastResult = "lost"
		return
	}
	st.Advance()
}

// ApplyNTChoice resolves the metagame / teamwork self-reports.
func ApplyNTChoice(st *State, slot int, kind, cardCode string) error {
	if slot != 0 {
		return fmt.Errorf("the host reports for the group")
	}
	switch kind {
	case ChoiceNTMeta:
		if cardCode == "yes" {
			st.Selections[fmt.Sprintf("gw%d", st.Counters["chapter"]+1)] = "shehulk"
		}
	case ChoiceNTTeam:
		if cardCode == "yes" {
			st.Counters["alliance"]++
		}
	default:
		return fmt.Errorf("unknown choice kind %q", kind)
	}
	st.ResolvePending(slot, kind)
	return nil
}
