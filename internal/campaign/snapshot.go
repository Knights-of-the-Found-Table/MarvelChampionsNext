package campaign

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// Snapshot is the campaign layer's reading of a finished game. Player
// indexes align with the campaign's slot order (BuildGame creates specs
// in slot order, and the engine preserves that order).
type Snapshot struct {
	Won    bool
	Expert bool

	// Per player.
	HP       []int // remaining hit points (MaxHP - damage)
	KOed     []bool
	HeroForm []bool
	// DiscardCodes holds every card code in each player's discard pile
	// (spent TECH upgrades are recognized there).
	Discard [][]string

	// RRS reads.
	Experimental      []string       // experimental weapons that entered play
	DelayCounters     int            // counters left on None Shall Pass
	Rescued           map[int]string // slot -> captive ally under their control
	Engaged           map[int]bool   // slot -> engaged with a minion
	HydraPrisonInPlay bool
	PrisonAllies      []string // codes of allies beneath Hydra Prison

	// Generic reads shared by later campaigns.
	MainStage           int            // stage the main scheme reached
	VictoryDisplayCodes []string       // codes in the victory display
	SideSchemeBaseCodes []string       // base codes of side schemes still in play
	EnvironmentCodes    []string       // environment codes in play
	MinionCodes         []string       // minion codes in play
	VillainCodes        []string       // villain codes in play
	CaptiveAllyCodes    []string       // captive-trait allies in play (MG)
	AllyDamage          map[string]int // damage on allies in play, by code
	AllySlots           map[string]int // ally code -> controlling slot
	AttachmentCodes     []string       // attachment codes in play
	EnvCountersByCode   map[string]int // counters on environments in play
	PrisonStoredCodes   []string       // cards tucked beneath side schemes (MG)
	Acceleration        int            // acceleration tokens on the main scheme
	DeckIllusion        int            // Illusion-trait cards inside player decks

	// GMW reads.
	VictoryPoints         int  // summed Victory values in the victory display
	NoMinions             bool // no minions in play at the end
	MainSchemeCode        string
	MainThreat            int      // threat left on the main scheme
	CollectionCount       int      // all cards in The Collection (any category)
	CollectionPlayerCodes []string // player cards in The Collection
	ArtifactsInDisplay    []string // Galactic Artifacts side schemes defeated
	HeadhunterDown        bool     // the Headhunter minion is in the victory display
	ShipCounters          int      // evasion counters on Nebula's Ship
	PowerStoneSlot        int      // slot controlling the Power Stone; -1 = none
}

// Observe reads the final engine state. scenarioDelayScheme (RRS) gates
// the delay-counter read to None Shall Pass.
func Observe(g *engine.Game) Snapshot {
	n := len(g.Players)
	snap := Snapshot{
		Won:            g.Won,
		Expert:         g.Difficulty == "expert",
		HP:             make([]int, n),
		KOed:           make([]bool, n),
		HeroForm:       make([]bool, n),
		Discard:        make([][]string, n),
		Rescued:        map[int]string{},
		Engaged:        map[int]bool{},
		PowerStoneSlot: -1,
		NoMinions:      true,
		MainSchemeCode: mainSchemeCode(g),
	}
	for i, p := range g.Players {
		snap.HP[i] = p.MaxHP - p.Damage
		if snap.HP[i] < 0 {
			snap.HP[i] = 0
		}
		snap.KOed[i] = p.KOed
		snap.HeroForm[i] = p.IsHero()
		for _, c := range p.Deck {
			if def, ok := engine.DB.Lookup(c.Code); ok && def.HasTrait("illusion") {
				snap.DeckIllusion++
			}
		}
		for _, c := range p.Discard {
			snap.Discard[i] = append(snap.Discard[i], c.Code)
		}
	}
	// Experimental weapons count as "entered the game" only while (or
	// after) being attached to the villain.
	for _, a := range g.Attachments {
		if a == nil {
			continue
		}
		if contains(rrsExperimental, a.Code) {
			snap.Experimental = appendUnique(snap.Experimental, a.Code)
		}
		// Power Stone control: attached to an identity.
		if a.Code == gmwPowerStone && a.Target.Kind() == engine.KindPlayer {
			for i, p := range g.Players {
				if p.ID == a.Target {
					snap.PowerStoneSlot = i
				}
			}
		}
	}
	// Rescued captives: Taskmaster-set allies under player control.
	for _, a := range g.Allies {
		if a == nil {
			continue
		}
		if contains(rrsRescued, a.Code) {
			for i, p := range g.Players {
				if a.EOwner() == p.ID {
					snap.Rescued[i] = a.Code
				}
			}
		}
	}
	for _, mn := range g.Minions {
		if mn == nil {
			continue
		}
		snap.NoMinions = false
		for i, p := range g.Players {
			if mn.EngagedWith == p.ID {
				snap.Engaged[i] = true
			}
		}
	}
	if s := g.MainScheme; s != nil {
		snap.DelayCounters = s.Counters
		snap.MainThreat = s.Threat
		snap.MainStage = s.Stage
	}
	for _, sc := range g.SideSchemes {
		if sc != nil {
			snap.SideSchemeBaseCodes = appendUnique(snap.SideSchemeBaseCodes, data.BaseCode(sc.Code))
		}
	}
	for _, env := range g.Environments {
		if env != nil {
			snap.EnvironmentCodes = appendUnique(snap.EnvironmentCodes, data.BaseCode(env.Code))
			if env.Counters > 0 {
				if snap.EnvCountersByCode == nil {
					snap.EnvCountersByCode = map[string]int{}
				}
				snap.EnvCountersByCode[data.BaseCode(env.Code)] = env.Counters
			}
		}
	}
	for _, mn := range g.Minions {
		if mn != nil {
			snap.MinionCodes = appendUnique(snap.MinionCodes, mn.Code)
		}
	}
	for _, a := range g.Allies {
		if a != nil {
			if snap.AllyDamage == nil {
				snap.AllyDamage = map[string]int{}
			}
			if snap.AllySlots == nil {
				snap.AllySlots = map[string]int{}
			}
			snap.AllyDamage[a.Code] = a.Damage
			snap.AllySlots[a.Code] = slotOfPlayer(g, a.EOwner())
			if def, ok := engine.DB.Lookup(a.Code); ok && def.HasTrait("captive") {
				snap.CaptiveAllyCodes = appendUnique(snap.CaptiveAllyCodes, a.Code)
			}
		}
	}
	for _, at := range g.Attachments {
		if at != nil {
			snap.AttachmentCodes = appendUnique(snap.AttachmentCodes, at.Code)
		}
	}
	for _, sc := range g.SideSchemes {
		if sc != nil {
			for _, c := range sc.StoredCards {
				snap.PrisonStoredCodes = appendUnique(snap.PrisonStoredCodes, data.BaseCode(c.Code))
			}
		}
	}
	for _, v := range g.Villains {
		if v != nil {
			snap.VillainCodes = appendUnique(snap.VillainCodes, data.BaseCode(v.Code))
		}
	}
	if g.MainScheme != nil {
		snap.Acceleration = g.MainScheme.AccelerationTokens
	}
	for _, c := range g.VictoryDisplay {
		snap.VictoryDisplayCodes = appendUnique(snap.VictoryDisplayCodes, data.BaseCode(c.Code))
	}
	// Hydra Prison: still in play, with the recorded allies beneath it.
	for _, s := range g.SideSchemes {
		if s == nil {
			continue
		}
		if data.BaseCode(s.Code) == rrsHydraPrison {
			snap.HydraPrisonInPlay = true
			for _, c := range s.StoredCards {
				snap.PrisonAllies = appendUnique(snap.PrisonAllies, c.Code)
			}
		}
	}
	for _, c := range g.VictoryDisplay {
		def, ok := engine.DB.Lookup(c.Code)
		if !ok {
			continue
		}
		snap.VictoryPoints += def.Victory
		if c.Code == gmwHeadhunter {
			snap.HeadhunterDown = true
		}
		if contains(galacticArtifacts, data.BaseCode(c.Code)) {
			snap.ArtifactsInDisplay = appendUnique(snap.ArtifactsInDisplay, data.BaseCode(c.Code))
		}
	}
	snap.CollectionCount = len(g.Collection)
	for _, c := range g.Collection {
		if def, ok := engine.DB.Lookup(c.Code); ok && def.Category == data.CategoryPlayer {
			snap.CollectionPlayerCodes = appendUnique(snap.CollectionPlayerCodes, c.Code)
		}
	}
	if ship := g.EnvironmentByCode(gmwShipEnv); ship != nil {
		snap.ShipCounters = ship.Counters
	}
	return snap
}

func mainSchemeCode(g *engine.Game) string {
	if g.MainScheme != nil {
		return g.MainScheme.Code
	}
	return ""
}

// slotOfPlayer maps a player id to its slot index (-1 when unknown).
func slotOfPlayer(g *engine.Game, id engine.PlayerID) int {
	for i, p := range g.Players {
		if p.ID == id {
			return i
		}
	}
	return -1
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func appendUnique(list []string, s string) []string {
	if !contains(list, s) {
		list = append(list, s)
	}
	return list
}
