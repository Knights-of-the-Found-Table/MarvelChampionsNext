package engine

import "fmt"

// Ability types.
const (
	AbilityAction  = "action"
	AbilityTrigger = "triggered" // interrupt or response
	AbilityForced  = "forced"
)

// Triggers recognized by the engine; reactive abilities declare one.
const (
	TriggerVillainAttacksYou = "villain_attacks_you"
	TriggerAfterYouThwart    = "after_you_thwart"
	TriggerWhenDefended      = "when_defended"
	TriggerWhenDamaged       = "when_you_take_damage"
)

// Ability is an activated or triggered ability on an entity. Abilities are
// recreated on demand from card behaviors; their identity within an entity is
// the index in the ability slice, which must stay deterministic.
type Ability struct {
	Label string `json:"label"`
	Type  string `json:"type"` // action | triggered | forced
	// Trigger is set for reactive abilities; the engine gathers matching
	// abilities at known points and asks the player.
	Trigger string `json:"trigger,omitempty"`

	// Cost flags
	Exhaust bool `json:"exhaust,omitempty"` // exhaust the source to activate
	Cost    int  `json:"cost,omitempty"`    // resource icons to pay
	// CostIcons constrains the resource types the cost must be paid with,
	// e.g. "physical:3" or "energy:1 mental:1" (wild resources count as
	// any type).
	CostIcons string `json:"costIcons,omitempty"`

	// Form criteria (identity-side abilities).
	HeroOnly     bool `json:"heroOnly,omitempty"`
	AlterEgoOnly bool `json:"alterEgoOnly,omitempty"`

	// Usage limits
	OncePerRound bool `json:"oncePerRound,omitempty"`
	OncePerTurn  bool `json:"oncePerTurn,omitempty"`

	// Execute runs the ability effect; messages returned are enqueued.
	Execute func(g *Game, self EntityID) []Message `json:"-"`
}

// Key identifies an ability instance for usage-limit tracking.
func AbilityKey(id EntityID, idx int) string {
	return fmt.Sprintf("%s#%d", id, idx)
}

// usable applies limit / cost gating common to all abilities.
func (a Ability) usable(g *Game, id EntityID, idx int, p *Player) bool {
	key := AbilityKey(id, idx)
	if a.OncePerRound && g.UsedThisRound[key] {
		return false
	}
	if a.OncePerTurn && g.UsedThisTurn[key] {
		return false
	}
	if a.HeroOnly && !p.IsHero() {
		return false
	}
	if a.AlterEgoOnly && p.IsHero() {
		return false
	}
	if a.Cost > 0 && len(p.Hand) < a.Cost {
		return false
	}
	if src := g.Entity(id); src != nil && a.Exhaust && src.EExhausted() {
		return false
	}
	return true
}
