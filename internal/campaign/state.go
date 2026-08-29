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

	// Contest campaigns (What If...?, Awesome, House of Mojo, Black
	// Order, Deadpool's Game Night).
	// What If...?: the trait recorded at scenario 1, the trait ally and
	// the trait-card rewards added to the deck.
	WITrait   string   `json:"wiTrait,omitempty"`
	WIAllies  []string `json:"wiAllies,omitempty"`
	WIRewards []string `json:"wiRewards,omitempty"`
	// Awesome Campaign: Guardian Influence, the guardian ally in play and
	// the identity-specific card fetched to the opening hand.
	Influence  int    `json:"influence,omitempty"`
	AWAlly     string `json:"awAlly,omitempty"`
	AWIdentity string `json:"awIdentity,omitempty"`
	// House of Mojo: role and its recorded cards.
	MojoRole    string `json:"mojoRole,omitempty"`
	MojoMarket  string `json:"mojoMarket,omitempty"`
	MojoScheme  string `json:"mojoScheme,omitempty"`
	MojoEvent   string `json:"mojoEvent,omitempty"`
	MojoUpgrade string `json:"mojoUpgrade,omitempty"`
	// Black Order: obligations carried in the deck and the Gear Up card
	// recorded at the Direct Assault opener.
	BordObligations []string `json:"bordObligations,omitempty"`
	BordGear        []string `json:"bordGear,omitempty"`
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
	// Named string records shared by the contest campaigns (branch
	// paths, Crime Wave lines, the Alias clue trail, Deadpool's game
	// results). Display names come from the per-box UI tables.
	Selections map[string]string `json:"selections,omitempty"`
	// Alias Investigations: rescued captive allies (victims) and Jessica
	// Jones's wound count.
	Victims []string `json:"victims,omitempty"`
	// Crimson Cowl Conspiracy: Masters of Evil minions caught (victory
	// display); escaped ones ride the shared Pool.
	CowlCaught []string `json:"cowlCaught,omitempty"`
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
		Selections: map[string]string{},
	}
	if st.Difficulty == "" {
		st.Difficulty = "standard"
	}
	return st
}

// EnsureMaps guarantees the named map fields are writable. Maps left out
// of a saved JSON blob (empty + omitempty) deserialize as nil.
func (st *State) EnsureMaps() {
	if st.Flags == nil {
		st.Flags = map[string]bool{}
	}
	if st.Counters == nil {
		st.Counters = map[string]int{}
	}
	if st.Selections == nil {
		st.Selections = map[string]string{}
	}
	if st.AOCounters == nil {
		st.AOCounters = map[string]int{}
	}
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
	// Sinister Motives reputation picks (also reused by the contest
	// campaigns whose designs build on the same cards).
	ChoiceSMTech   = "sm-tech"   // keep one of three dealt S.H.I.E.L.D. Tech upgrades
	ChoiceSMAspect = "sm-aspect" // aspect advantage: name any aspect card
	ChoiceSMPlan   = "sm-plan"   // planning ahead: name a card from your deck
	// What If...? (Amanda Shagoury).
	ChoiceWITrait = "wi-trait" // record the group-hero trait
	ChoiceWIAlly  = "wi-ally"  // add an ally with that trait to the deck
	ChoiceWICard  = "wi-card"  // add any player card with that trait
	// Awesome Campaign.
	ChoiceAWAlly     = "aw-ally"     // basic GUARDIAN ally into play
	ChoiceAWIdentity = "aw-identity" // identity-specific card to the hand
	// House of Mojo.
	ChoiceMojoRole     = "mojo-role"     // pick the campaign role (per player)
	ChoiceMojoTraining = "mojo-training" // pick the training player side scheme (group)
	ChoiceMojoEvent    = "mojo-event"    // add an aspect event to the deck
	ChoiceMojoMarket   = "mojo-market"   // Shawarma or the role's market card
	ChoiceMojoScheme   = "mojo-scheme"   // add a player side scheme to the deck
	// Revenge of the Black Order.
	ChoiceBordPath = "bord-path" // pick the narrative path (group)
	ChoiceBordGear = "bord-gear" // record the Gear Up support/upgrade
	// She-Hulk vs. Deadpool's Game Night self-reports.
	ChoiceNTMeta = "nt-meta" // was the metagame challenge completed?
	ChoiceNTTeam = "nt-team" // was the teamwork goal completed?
	ChoiceNTPick = "nt-pick" // pick a reward-pool card offer
	// The Watcher's Team: optional named-card rewards.
	ChoiceWASight      = "wa-sight"      // resource card from deck to hand
	ChoiceWAPortal     = "wa-portal"     // ally card from deck to hand
	ChoiceWAIntervened = "wa-intervened" // identity-specific card to hand
	// Going Viral / Entropic Ascension branching.
	ChoiceViralNext = "viral-next" // which Scenario #2 to play next (group)
	ChoiceEnPath1   = "en-path1"   // Investigative Journalism vs Blackout (group)
	ChoiceEnPath2   = "en-path2"   // Police Transport vs Raft Transport / Strangers vs Friends (group)
	ChoiceEnPath3   = "en-path3"   // Engineering vs Biology (group)
)

// SelfReportKinds lists the optional choice kinds where an empty answer
// records "the players did not achieve it" instead of cancelling a
// reward. Group self-reports resolve on the host's seat.
var SelfReportKinds = map[string]bool{
	ChoiceNTMeta: true,
	ChoiceNTTeam: true,
}

// GroupChoiceKinds lists choice kinds owed by seat 0 on behalf of the
// group.
var GroupChoiceKinds = map[string]bool{
	ChoiceBordPath:     true,
	ChoiceViralNext:    true,
	ChoiceEnPath1:      true,
	ChoiceEnPath2:      true,
	ChoiceEnPath3:      true,
	ChoiceMojoTraining: true,
}

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
