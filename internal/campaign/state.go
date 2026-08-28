package campaign

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

	// Interlude bookkeeping: slot -> outstanding choice kind
	// ("tech" | "condition" | "improve"); players not listed are done
	// (or have nothing to choose).
	PendingChoices map[int]string `json:"pendingChoices,omitempty"`
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
)

// PendingFor reports whether the slot still owes an interlude choice.
func (st *State) PendingFor(slot int) (string, bool) {
	k, ok := st.PendingChoices[slot]
	return k, ok
}

// Advance moves to the next scenario (or completes the campaign).
func (st *State) Advance() {
	st.Index++
	if st.Index >= len(st.BoxDef().Scenarios) {
		st.Status = "complete"
	} else {
		st.Status = "interlude"
	}
}
