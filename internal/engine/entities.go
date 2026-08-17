package engine

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// ---------------------------------------------------------------- Villain

type Villain struct {
	ID   EntityID `json:"id"`
	Code string   `json:"code"`
	// stageCodes lists the card codes for each villain stage (index 0 =
	// stage 1); advancing swaps the code.
	stageCodes []string
	Stage     int `json:"stage"`
	MaxHP     int `json:"maxHp"`
	Damage    int `json:"damage"`
	SchemeVal int `json:"scheme"`
	AttackVal int `json:"attack"`

	Stunned  bool `json:"stunned"`
	Confused bool `json:"confused"`
	Tough    bool `json:"tough"`

	// BoostCards are facedown boost cards dealt during activation; revealed
	// count contributes to scheme/attack totals.
	BoostCards CardList `json:"boostCards"`
	BoostCount int      `json:"boostCount"`
	// RevealedBoosts are face-up boost cards awaiting cleanup.
	RevealedBoosts CardList `json:"revealedBoosts"`

	Attachments []EntityID `json:"attachments"`
	// Undamageable marks special stages (e.g. Rhino I).
	Undamageable bool `json:"undamageable,omitempty"`
}

func (v *Villain) EID() EntityID          { return v.ID }
func (v *Villain) ECode() string          { return v.Code }
func (v *Villain) EDef() *data.CardDef    { return DB.MustLookup(v.Code) }
func (v *Villain) EOwner() PlayerID       { return "" }
func (v *Villain) EExhausted() bool       { return false }
func (v *Villain) React(g *Game, msg Message) []Message {
	b := behavior(v.Code)
	if b.React != nil {
		return b.React(g, v, msg)
	}
	return nil
}

func (v *Villain) HP() int { return v.MaxHP - v.Damage }

// syncFromDef refreshes stats after a stage change.
func (v *Villain) syncFromDef() {
	def := v.EDef()
	v.MaxHP = deref(def.HP, v.MaxHP)
	v.SchemeVal = deref(def.Scheme, v.SchemeVal)
	v.AttackVal = deref(def.Attack, v.AttackVal)
	v.Undamageable = def.HasKeyword("Toughness") // placeholder; scenarios set this explicitly
}

// ---------------------------------------------------------------- Minion

type Minion struct {
	ID        EntityID `json:"id"`
	Code      string   `json:"code"`
	Owner     PlayerID `json:"owner,omitempty"` // "" = encounter side
	MaxHP     int      `json:"maxHp"`
	Damage    int      `json:"damage"`
	AttackVal int      `json:"attack"`
	SchemeVal int      `json:"scheme"`

	Stunned  bool `json:"stunned"`
	Confused bool `json:"confused"`
	Tough    bool `json:"tough"`
	// GuardMinion marks Guard keyword (must be attacked before others).
	Guard bool `json:"guard"`
	// IsDrone marks facedown player-deck drones (Ultron scenario).
	IsDrone bool     `json:"isDrone,omitempty"`
	Source  *Card    `json:"source,omitempty"` // the facedown card, if any
	// EngagedWith records which player the minion is engaged with.
	EngagedWith PlayerID `json:"engagedWith,omitempty"`

	Attachments []EntityID `json:"attachments"`
}

func (m *Minion) EID() EntityID       { return m.ID }
func (m *Minion) ECode() string       { return m.Code }
func (m *Minion) EDef() *data.CardDef { return DB.MustLookup(m.Code) }
func (m *Minion) EOwner() PlayerID    { return m.Owner }
func (m *Minion) EExhausted() bool    { return false }
func (m *Minion) React(g *Game, msg Message) []Message {
	b := behavior(m.Code)
	if b.React != nil {
		return b.React(g, m, msg)
	}
	return nil
}
func (m *Minion) HP() int { return m.MaxHP - m.Damage }

// ---------------------------------------------------------------- Ally

type Ally struct {
	ID        EntityID `json:"id"`
	Code      string   `json:"code"`
	Owner     PlayerID `json:"owner"`
	MaxHP     int      `json:"maxHp"`
	Damage    int      `json:"damage"`
	AttackVal int      `json:"attack"`
	ThwartVal int      `json:"thwart"`

	Exhausted bool `json:"exhausted"`
	Stunned   bool `json:"stunned"`
	Confused  bool `json:"confused"`
	Tough     bool `json:"tough"`
}

func (a *Ally) EID() EntityID       { return a.ID }
func (a *Ally) ECode() string       { return a.Code }
func (a *Ally) EDef() *data.CardDef { return DB.MustLookup(a.Code) }
func (a *Ally) EOwner() PlayerID    { return a.Owner }
func (a *Ally) EExhausted() bool    { return a.Exhausted }
func (a *Ally) React(g *Game, msg Message) []Message {
	b := behavior(a.Code)
	if b.React != nil {
		return b.React(g, a, msg)
	}
	return nil
}
func (a *Ally) HP() int { return a.MaxHP - a.Damage }

// ---------------------------------------------------------------- Support

type Support struct {
	ID        EntityID `json:"id"`
	Code      string   `json:"code"`
	Owner     PlayerID `json:"owner"`
	Exhausted bool     `json:"exhausted"`
}

func (s *Support) EID() EntityID       { return s.ID }
func (s *Support) ECode() string       { return s.Code }
func (s *Support) EDef() *data.CardDef { return DB.MustLookup(s.Code) }
func (s *Support) EOwner() PlayerID    { return s.Owner }
func (s *Support) EExhausted() bool    { return s.Exhausted }
func (s *Support) React(g *Game, msg Message) []Message {
	b := behavior(s.Code)
	if b.React != nil {
		return b.React(g, s, msg)
	}
	return nil
}

// ---------------------------------------------------------------- Upgrade

type Upgrade struct {
	ID        EntityID `json:"id"`
	Code      string   `json:"code"`
	Owner     PlayerID `json:"owner"`
	Exhausted bool     `json:"exhausted"`
}

func (u *Upgrade) EID() EntityID       { return u.ID }
func (u *Upgrade) ECode() string       { return u.Code }
func (u *Upgrade) EDef() *data.CardDef { return DB.MustLookup(u.Code) }
func (u *Upgrade) EOwner() PlayerID    { return u.Owner }
func (u *Upgrade) EExhausted() bool    { return u.Exhausted }
func (u *Upgrade) React(g *Game, msg Message) []Message {
	b := behavior(u.Code)
	if b.React != nil {
		return b.React(g, u, msg)
	}
	return nil
}

// ---------------------------------------------------------------- Treachery

type Treachery struct {
	ID     EntityID `json:"id"`
	Code   string   `json:"code"`
	Target EntityID `json:"target,omitempty"` // e.g. scheme it attaches to
}

func (t *Treachery) EID() EntityID       { return t.ID }
func (t *Treachery) ECode() string       { return t.Code }
func (t *Treachery) EDef() *data.CardDef { return DB.MustLookup(t.Code) }
func (t *Treachery) EOwner() PlayerID    { return "" }
func (t *Treachery) EExhausted() bool    { return false }
func (t *Treachery) React(g *Game, msg Message) []Message {
	b := behavior(t.Code)
	if b.React != nil {
		return b.React(g, t, msg)
	}
	return nil
}

// ---------------------------------------------------------------- Attachment

type Attachment struct {
	ID     EntityID `json:"id"`
	Code   string   `json:"code"`
	Target EntityID `json:"target,omitempty"`
}

func (t *Attachment) EID() EntityID       { return t.ID }
func (t *Attachment) ECode() string       { return t.Code }
func (t *Attachment) EDef() *data.CardDef { return DB.MustLookup(t.Code) }
func (t *Attachment) EOwner() PlayerID    { return "" }
func (t *Attachment) EExhausted() bool    { return false }
func (t *Attachment) React(g *Game, msg Message) []Message {
	b := behavior(t.Code)
	if b.React != nil {
		return b.React(g, t, msg)
	}
	return nil
}

// ---------------------------------------------------------------- Schemes

type MainScheme struct {
	ID EntityID `json:"id"`
	// Code is the current stage's card code.
	Code string `json:"code"`
	// StageCodes lists codes for each scheme stage.
	StageCodes []string `json:"stageCodes"`
	Stage      int      `json:"stage"`

	Threat    int `json:"threat"`
	MaxThreat int `json:"maxThreat"`
	// AccelerationTokens add threat each resource phase.
	AccelerationTokens int `json:"accelerationTokens"`
	// Crisis blocks thwarting; Hazard adds encounter cards.
	Crisis bool `json:"crisis,omitempty"`
	Hazard int  `json:"hazard,omitempty"`
}

func (s *MainScheme) EID() EntityID       { return s.ID }
func (s *MainScheme) ECode() string       { return s.Code }
func (s *MainScheme) EDef() *data.CardDef { return DB.MustLookup(s.Code) }
func (s *MainScheme) EOwner() PlayerID    { return "" }
func (s *MainScheme) EExhausted() bool    { return false }
func (s *MainScheme) React(g *Game, msg Message) []Message {
	b := behavior(s.Code)
	if b.React != nil {
		return b.React(g, s, msg)
	}
	return nil
}

type SideScheme struct {
	ID        EntityID `json:"id"`
	Code      string   `json:"code"`
	Threat    int      `json:"threat"`
	MaxThreat int      `json:"maxThreat"`
	Crisis    bool     `json:"crisis,omitempty"`
	Hazard    int      `json:"hazard,omitempty"`
}

func (s *SideScheme) EID() EntityID       { return s.ID }
func (s *SideScheme) ECode() string       { return s.Code }
func (s *SideScheme) EDef() *data.CardDef { return DB.MustLookup(s.Code) }
func (s *SideScheme) EOwner() PlayerID    { return "" }
func (s *SideScheme) EExhausted() bool    { return false }
func (s *SideScheme) React(g *Game, msg Message) []Message {
	b := behavior(s.Code)
	if b.React != nil {
		return b.React(g, s, msg)
	}
	return nil
}

// ---------------------------------------------------------------- Environment

type Environment struct {
	ID        EntityID `json:"id"`
	Code      string   `json:"code"`
	Exhausted bool     `json:"exhausted"`
	// Counters tracks scenario counters (infamy/madness...).
	Counters int `json:"counters,omitempty"`
}

func (e *Environment) EID() EntityID       { return e.ID }
func (e *Environment) ECode() string       { return e.Code }
func (e *Environment) EDef() *data.CardDef { return DB.MustLookup(e.Code) }
func (e *Environment) EOwner() PlayerID    { return "" }
func (e *Environment) EExhausted() bool    { return e.Exhausted }
func (e *Environment) React(g *Game, msg Message) []Message {
	b := behavior(e.Code)
	if b.React != nil {
		return b.React(g, e, msg)
	}
	return nil
}
