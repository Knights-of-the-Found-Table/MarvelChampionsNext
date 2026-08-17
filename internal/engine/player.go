package engine

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// Side values for identity form.
const (
	SideHero     = "hero"
	SideAlterEgo = "alterego"
)

// Player is one identity (hero + alter-ego) controlled by a user.
type Player struct {
	ID            PlayerID `json:"id"`
	Name          string   `json:"name"` // display name
	UserID        string   `json:"userId,omitempty"`
	HeroCode      string   `json:"heroCode"`
	AlterEgoCode  string   `json:"alterEgoCode"`
	Side          string   `json:"side"`
	FormChanged   bool     `json:"formChanged"` // once per turn

	MaxHP  int `json:"maxHp"`
	Damage int `json:"damage"`

	Exhausted bool `json:"exhausted"`
	Stunned   bool `json:"stunned"`
	Confused  bool `json:"confused"`
	Tough     bool `json:"tough"`

	Deck     CardList `json:"deck"`
	Hand     CardList `json:"hand"`
	Discard  CardList `json:"discard"`

	// Obligation deck for hero-specific obligations.
	ObligationDeck    CardList `json:"obligationDeck"`
	ObligationDiscard CardList `json:"obligationDiscard"`

	// Controlled entities.
	Allies    []EntityID `json:"allies"`
	Supports  []EntityID `json:"supports"`
	Upgrades  []EntityID `json:"upgrades"`
	Resources []EntityID `json:"resources"` // resource-type cards in play

	// Facedown encounter cards dealt to this player.
	EncounterDown CardList `json:"encounterDown"`

	// Nemesis set state.
	NemesisDeck    CardList `json:"nemesisDeck"`
	NemesisDiscard CardList `json:"nemesisDiscard"`
	NemesisInPlay  []EntityID `json:"nemesisInPlay"`

	EndedTurn    bool `json:"endedTurn"`
	FirstPlayer  bool `json:"firstPlayer"`
	KOed         bool `json:"koed"`

	// UsedAbilityRounds tracks once-per-round ability usage for abilities
	// owned by this identity.
	UsedAbilityRounds map[string]int `json:"usedAbilityRounds,omitempty"`
}

func (p *Player) EID() EntityID    { return p.ID }
func (p *Player) ECode() string {
	if p.IsHero() {
		return p.HeroCode
	}
	return p.AlterEgoCode
}
func (p *Player) EDef() *data.CardDef { return DB.MustLookup(p.ECode()) }
func (p *Player) EOwner() PlayerID    { return p.ID }
func (p *Player) EExhausted() bool    { return p.Exhausted }

func (p *Player) IsHero() bool { return p.Side == SideHero }

// HeroDef / AlterEgoDef return the two identity card definitions.
func (p *Player) HeroDef() *data.CardDef     { return DB.MustLookup(p.HeroCode) }
func (p *Player) AlterEgoDef() *data.CardDef { return DB.MustLookup(p.AlterEgoCode) }

// HP remaining.
func (p *Player) HP() int { return p.MaxHP - p.Damage }

// HandSize is the draw-to target for the resource phase.
func (p *Player) HandSize(g *Game) int {
	n := deref(p.AlterEgoDef().HandSize, 6)
	if p.IsHero() {
		n = deref(p.HeroDef().HandSize, 5)
	}
	if g != nil {
		if b := behavior(p.HeroCode); b.HandSizeBonus != nil {
			n += b.HandSizeBonus(g, p)
		}
	}
	return n
}

// Attack/Thwart/Defense/Recover values for the current side; -1 when the
// side doesn't have the stat.
func (p *Player) AttackStat() int {
	if !p.IsHero() {
		return -1
	}
	return deref(p.HeroDef().Attack, 0)
}
func (p *Player) ThwartStat() int {
	if !p.IsHero() {
		return -1
	}
	return deref(p.HeroDef().Thwart, 0)
}
func (p *Player) DefenseStat() int {
	if !p.IsHero() {
		return -1
	}
	return deref(p.HeroDef().Defense, 0)
}
func (p *Player) RecoverStat() int {
	if p.IsHero() {
		return -1
	}
	return deref(p.AlterEgoDef().Recover, 0)
}

// React implements identity reactions via the hero behavior hook.
func (p *Player) React(g *Game, msg Message) []Message {
	if b := behavior(p.HeroCode); b.React != nil {
		return b.React(g, p, msg)
	}
	return nil
}

func deref(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
