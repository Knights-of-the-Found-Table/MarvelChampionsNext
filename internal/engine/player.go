package engine

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// Side values for identity form.
const (
	SideHero     = "hero"
	SideAlterEgo = "alterego"
)

// CostDiscount is a pending reduction applied to the next matching payment.
type CostDiscount struct {
	Type   string `json:"type,omitempty"`  // card type filter, "" = any
	Trait  string `json:"trait,omitempty"` // trait filter, "" = any
	Amount int    `json:"amount"`
}

// Player is one identity (hero + alter-ego) controlled by a user.
type Player struct {
	ID           PlayerID `json:"id"`
	Name         string   `json:"name"` // display name
	UserID       string   `json:"userId,omitempty"`
	HeroCode     string   `json:"heroCode"`
	AlterEgoCode string   `json:"alterEgoCode"`
	Side         string   `json:"side"`
	FormChanged  bool     `json:"formChanged"` // once per turn

	MaxHP  int `json:"maxHp"`
	Damage int `json:"damage"`

	Exhausted bool `json:"exhausted"`
	Stunned   bool `json:"stunned"`
	Confused  bool `json:"confused"`
	Tough     bool `json:"tough"`

	Deck    CardList `json:"deck"`
	Hand    CardList `json:"hand"`
	Discard CardList `json:"discard"`

	// Obligation deck for hero-specific obligations; the cards are merged
	// into the encounter deck at game start and resolve for their owner
	// when revealed.
	ObligationDeck    CardList `json:"obligationDeck"`
	ObligationDiscard CardList `json:"obligationDiscard"`
	ObligationRemoved CardList `json:"obligationRemoved,omitempty"`

	// SenseDeck is the hero's side deck of Sense upgrades (Daredevil);
	// Sense cards cycle back here when they leave play. It doubles as the
	// Invocation deck for Doctor Strange.
	SenseDeck CardList `json:"senseDeck,omitempty"`
	// SideDiscard is the side deck's discard pile (resolved Invocations).
	SideDiscard CardList `json:"sideDiscard,omitempty"`

	// CostDiscounts are pending one-shot cost modifiers (Nakia Bahadir,
	// Avengers Tower; negative amounts are cost increases such as Physical
	// Toll). Matching discounts stack, the next matching payment consumes
	// them all, and they are cleared at each phase change.
	CostDiscounts []CostDiscount `json:"costDiscounts,omitempty"`
	// AllyPlayedThisRound marks whether an ally has been played this round
	// (Living Legend discount).
	AllyPlayedThisRound bool `json:"allyPlayedThisRound,omitempty"`

	// Controlled entities.
	Allies    []EntityID `json:"allies"`
	Supports  []EntityID `json:"supports"`
	Upgrades  []EntityID `json:"upgrades"`
	Resources []EntityID `json:"resources"` // resource-type cards in play

	// Facedown encounter cards dealt to this player.
	EncounterDown CardList `json:"encounterDown"`

	// Nemesis set state.
	NemesisDeck    CardList   `json:"nemesisDeck"`
	NemesisDiscard CardList   `json:"nemesisDiscard"`
	NemesisInPlay  []EntityID `json:"nemesisInPlay"`

	EndedTurn   bool `json:"endedTurn"`
	FirstPlayer bool `json:"firstPlayer"`
	KOed        bool `json:"koed"`

	// Until-end-of-phase stat modifiers (Fearless Determination,
	// Avengers Assemble...); cleared at each phase change.
	BonusTHW int `json:"bonusThw,omitempty"`
	// TempHandSize is a until-end-of-phase hand-size bonus (Assess the
	// Situation); cleared at each phase change.
	TempHandSize int `json:"tempHandSize,omitempty"`
	BonusATK     int `json:"bonusAtk,omitempty"`
	BonusDEF     int `json:"bonusDef,omitempty"`
	// ExtraTraits are dynamically granted traits (Honorary Avenger).
	ExtraTraits []string `json:"extraTraits,omitempty"`
	// GrowthCounters are identity-level prevention counters (Groot).
	GrowthCounters int `json:"growthCounters,omitempty"`
	// Counters are generic counters placed on the identity card itself
	// (Gambit's charge counters); adjusted via AddEntityCounter.
	Counters int `json:"counters,omitempty"`

	// UsedAbilityRounds tracks once-per-round ability usage for abilities
	// owned by this identity.
	UsedAbilityRounds map[string]int `json:"usedAbilityRounds,omitempty"`
}

func (p *Player) EID() EntityID { return p.ID }
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
		// Upgrades in play can also modify hand size (Star-Lord's
		// Helmet, The Sorcerer Supreme).
		for _, id := range p.Upgrades {
			if u := g.Upgrades[id]; u != nil {
				if b := behavior(u.Code); b.HandSizeBonus != nil {
					n += b.HandSizeBonus(g, p)
				}
			}
		}
		// Left to Your Fate (40167): each identity gets +2 hand size.
		if g.MainScheme != nil && data.BaseCode(g.MainScheme.Code) == "40167" {
			n += 2
		}
		// Live Dangerously (44024): each identity gets +2 hand size
		// while the scheme is in play.
		for _, s := range g.SideSchemes {
			if s != nil && s.Code == "44024" {
				n += 2
			}
		}
		// Tempo (40181): +1 hand size while engaged with you.
		for _, mn := range g.Minions {
			if mn != nil && mn.Code == "40181" && mn.EngagedWith == p.ID {
				n++
			}
		}
	}
	return n + p.TempHandSize
}

// Attack/Thwart/Defense/Recover values for the current side; -1 when the
// side doesn't have the stat. Includes until-end-of-phase bonuses and
// persistent bonuses from upgrades in play.
func (p *Player) AttackStat(g *Game) int {
	if !p.IsHero() {
		return -1
	}
	return deref(p.HeroDef().Attack, 0) + p.BonusATK + p.upgradeStats(g).ATK
}
func (p *Player) ThwartStat(g *Game) int {
	if !p.IsHero() {
		return -1
	}
	return deref(p.HeroDef().Thwart, 0) + p.BonusTHW + p.upgradeStats(g).THW
}
func (p *Player) DefenseStat(g *Game) int {
	if !p.IsHero() {
		return -1
	}
	return deref(p.HeroDef().Defense, 0) + p.BonusDEF + p.upgradeStats(g).DEF
}
func (p *Player) RecoverStat(g *Game) int {
	if p.IsHero() {
		return -1
	}
	return deref(p.AlterEgoDef().Recover, 0) + p.upgradeStats(g).REC
}

// upgradeStats sums the IdentityStats bonuses of the upgrades in play.
func (p *Player) upgradeStats(g *Game) (b StatBonus) {
	if g == nil {
		return b
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil {
			bh := behavior(u.Code)
			var s StatBonus
			if hook := bh.IdentityStats; hook != nil {
				s = hook(p)
			}
			if hook := bh.IdentityStatsG; hook != nil {
				sg := hook(g, p, u)
				s.ATK += sg.ATK
				s.THW += sg.THW
				s.DEF += sg.DEF
				s.REC += sg.REC
				s.Retaliate += sg.Retaliate
			}
			b.ATK += s.ATK
			b.THW += s.THW
			b.DEF += s.DEF
			b.REC += s.REC
			b.Retaliate += s.Retaliate
		}
	}
	return b
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
