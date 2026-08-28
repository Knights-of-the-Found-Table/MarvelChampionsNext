package campaign

import (
	"fmt"
	"sort"
	"strings"
)

// PlayerLog is one hero's campaign-long state (one column of the printed
// campaign log).
type PlayerLog struct {
	Slot     int            `json:"slot"`
	UserID   int64          `json:"userId"`
	Name     string         `json:"name"`
	HeroBase string         `json:"heroBase"`
	Deck     map[string]int `json:"deck"`
	// Expert persistent damage: remaining hit points recorded after each
	// win (capped at the printed base); 0 = printed value.
	HP     int      `json:"hp,omitempty"`
	Units  int      `json:"units,omitempty"`
	Market []string `json:"market,omitempty"`
	// HealNext: at the next scenario's setup, heal to printed HP (GMW:
	// spend 1 unit; RRS expert: add a random obligation to the deck).
	HealNext bool `json:"healNext,omitempty"`
	// RRS fields.
	Tech      string   `json:"tech,omitempty"`
	Condition string   `json:"condition,omitempty"` // basic side code
	Improved  bool     `json:"improved,omitempty"`
	Allies    []string `json:"allies,omitempty"`
	// Recorded at Zola: the player ended the game engaged with an enemy;
	// Red Skull expert makes them deal themselves an encounter card.
	EngagedEnemy bool `json:"engagedEnemy,omitempty"`
	// Sinister Motives per-player sections.
	SMShieldTech string   `json:"smTech,omitempty"`
	SMTechOffer  []string `json:"smTechOffer,omitempty"`
	SMAspect     string   `json:"smAspect,omitempty"`
	SMPlanning   string   `json:"smPlanning,omitempty"`
	SMEnhanced   bool     `json:"smEnhanced,omitempty"`
	// Setup-time fetch: search deck and discard for this card and add it
	// to the opening hand (SM reputation node "Planning Ahead").
	SetupHand string `json:"setupHand,omitempty"`
	// Mutant Genesis role (brawler|commander|defender|peacekeeper).
	MGRole string `json:"mgRole,omitempty"`
}

// State serializes into the campaigns table's state column.
type State struct {
	Box        string      `json:"box"`
	Difficulty string      `json:"difficulty"`
	Index      int         `json:"index"`
	Status     string      `json:"status"` // forming | interlude | active | complete | lost
	Players    []PlayerLog `json:"players"`

	// RRS log fields.
	Experimental  []string `json:"experimental,omitempty"`
	DelayCounters int      `json:"delayCounters,omitempty"`
	RemovedAllies []string `json:"removedAllies,omitempty"`

	// GMW log fields.
	Collection []string `json:"collection,omitempty"`
	Artifacts  []string `json:"artifacts,omitempty"`
	Headhunter []bool   `json:"headhunter,omitempty"`
	PowerStone int      `json:"powerStone,omitempty"` // slot index; -1 = uncontrolled
	Evasion    int      `json:"evasion,omitempty"`

	// Shared pool of campaign cards (MTS): codes enter on per-scenario
	// results and the later setups pull from it. Unknown codes (cards
	// missing from the data snapshot) degrade to no-ops.
	Pool []string `json:"pool,omitempty"`
	// Named boolean records (MTS: "stones", "towerDamaged"; other boxes
	// reuse this for their log checkboxes).
	Flags map[string]bool `json:"flags,omitempty"`
	// Named numeric records (SM: reputation).
	Counters map[string]int `json:"counters,omitempty"`

	// Agents of S.H.I.E.L.D.: the mole's hidden evidence, the envelope
	// pools, per-member secret counters and the accusation. AOImEnvelope
	// is redacted from API payloads (players must deduce it!).
	AOImEnvelope     AOSCombo       `json:"aoImEnvelope"`
	AOShieldEnvelope []string       `json:"aoShieldEnvelope,omitempty"`
	AOEvidence       []string       `json:"aoEvidence,omitempty"`
	AOCounters       map[string]int `json:"aoCounters,omitempty"`
	AOSurvivors      []string       `json:"aoSurvivors,omitempty"`

	// Age of Apocalypse: this chapter's mission/Overseer and the struck
	// ones from previous chapters.
	AOMission     string   `json:"aoMission,omitempty"`
	AOOverseer    string   `json:"aoOverseer,omitempty"`
	AOMissionLog  []string `json:"aoMissionLog,omitempty"`
	AOOverseerLog []string `json:"aoOverseerLog,omitempty"`

	// NeXt Evolution: earned environments (b-sides), schemes chosen so
	// far, and the pick owed to the current (possibly retried) chapter.
	NXEnvEarned []string `json:"nxEnvEarned,omitempty"`
	NXChosen    []string `json:"nxChosen,omitempty"`
	NXCurrent   string   `json:"nxCurrent,omitempty"`

	// Mutant Genesis log sections.
	MGFuturePast    []string `json:"mgFuturePast,omitempty"`
	MGCaptives      []string `json:"mgCaptives,omitempty"`
	MGRemovedAllies []string `json:"mgRemovedAllies,omitempty"`

	// Sinister Motives log sections.
	SMOsborn       []string `json:"smOsborn,omitempty"`
	SMCommunity    []string `json:"smCommunity,omitempty"`
	SMWaking       int      `json:"smWaking,omitempty"`
	SMLastStanding []string `json:"smLastStanding,omitempty"`

	// Interlude bookkeeping: "slot:kind" -> kind; a player may owe
	// several choices at once (e.g. SM tech + aspect + planning).
	PendingChoices map[string]string `json:"pendingChoices,omitempty"`
	// LastResult reports the most recent scenario outcome for the UI.
	LastResult string `json:"lastResult,omitempty"` // "won" | "lost"
}

// New builds the initial state for a campaign at the forming stage.
func New(box, difficulty string, players []PlayerLog) *State {
	st := &State{
		Box:        box,
		Difficulty: difficulty,
		Status:     "forming",
		Players:    players,
		PowerStone: -1,
		Flags:      map[string]bool{},
		Counters:   map[string]int{},
	}
	if st.Difficulty == "" {
		st.Difficulty = "standard"
	}
	return st
}

// Box returns the box definition (nil for unknown keys).
func (st *State) BoxDef() *BoxDef { return Boxes[st.Box] }

// IsExpert reports whether the campaign runs in expert mode.
func (st *State) IsExpert() bool { return st.Difficulty == "expert" }

// Slot finds a player log by slot index.
func (st *State) Slot(i int) *PlayerLog {
	if i < 0 || i >= len(st.Players) {
		return nil
	}
	return &st.Players[i]
}

// Choice kinds offered in the interlude.
const (
	ChoiceTech      = "tech"      // RRS scenario 1: pick one TECH upgrade
	ChoiceCondition = "condition" // RRS scenario 2: optionally pick a Basic Condition upgrade
	ChoiceImprove   = "improve"   // RRS scenario 4: optionally flip to the Improved side
	// Sinister Motives reputation picks.
	ChoiceSMTech   = "sm-tech"   // keep one of three dealt S.H.I.E.L.D. Tech upgrades
	ChoiceSMAspect = "sm-aspect" // aspect advantage: name any aspect card
	ChoiceSMPlan   = "sm-plan"   // planning ahead: name a card from your deck
)

// PendingKey composes the pending-choice map key.
func PendingKey(slot int, kind string) string { return fmt.Sprintf("%d:%s", slot, kind) }

// PendingFor reports whether the slot owes the given choice kind.
func (st *State) PendingFor(slot int, kind string) bool {
	_, ok := st.PendingChoices[PendingKey(slot, kind)]
	return ok
}

// HasPending reports whether the slot owes any choice.
func (st *State) HasPending(slot int) bool {
	prefix := fmt.Sprintf("%d:", slot)
	for k := range st.PendingChoices {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

// PendingKinds lists every owed choice kind for the slot.
func (st *State) PendingKinds(slot int) []string {
	prefix := fmt.Sprintf("%d:", slot)
	var out []string
	for k, kind := range st.PendingChoices {
		if strings.HasPrefix(k, prefix) {
			out = append(out, kind)
		}
	}
	sort.Strings(out)
	return out
}

// AnyPending reports whether anyone still owes a choice.
func (st *State) AnyPending() bool { return len(st.PendingChoices) > 0 }

// Advance moves to the next scenario (or completes the campaign).
func (st *State) Advance() {
	st.Index++
	if st.Index >= len(st.BoxDef().Scenarios) {
		st.Status = "complete"
	} else {
		st.Status = "interlude"
	}
}

// AddPending queues an interlude choice for a player (exported for the
// API layer).
func (st *State) AddPending(slot int, kind string) { st.addPending(slot, kind) }

// NXAvailable lists the campaign player side schemes not chosen yet.
func (st *State) NXAvailable() []string {
	var out []string
	for _, s := range []string{"40190a", "40191a", "40192a", "40193a", "40194a", "40195a"} {
		if !contains(st.NXChosen, s) {
			out = append(out, s)
		}
	}
	return out
}

// MGRoles lists the Mutant Genesis campaign roles.
func MGRoles() []string { return []string{"brawler", "commander", "defender", "peacekeeper"} }

// AOBoardMembers lists the S.H.I.E.L.D. Executive Board environment codes.
func AOBoardMembers() []string { return []string{"50181a", "50182a", "50183a"} }

// ResolvePending clears one owed choice.
func (st *State) ResolvePending(slot int, kind string) {
	delete(st.PendingChoices, PendingKey(slot, kind))
}
