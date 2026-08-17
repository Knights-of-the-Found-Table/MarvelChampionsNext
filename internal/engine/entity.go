package engine

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// Entity is anything addressable in play. Concrete kinds: Player, Villain,
// Minion, Ally, Support, Upgrade, Attachment, Treachery, SideScheme,
// MainScheme, Environment.
type Entity interface {
	EID() EntityID
	ECode() string
	EDef() *data.CardDef
	// EOwner returns the controlling player id ("" for scenario entities).
	EOwner() PlayerID
	// EExhausted reports readiness (false for kinds that never exhaust).
	EExhausted() bool
	// React lets the entity react to a message being processed; returned
	// messages are enqueued after the current one. Generic keyword handling
	// and card behavior hooks are merged here per kind.
	React(g *Game, msg Message) []Message
}

// Behavior bundles the per-card hooks a card implementation installs. A nil
// hook means the generic engine behavior applies. Behaviors are looked up
// from the registry by card code and are never serialized.
type Behavior struct {
	// Ally/Support/Upgrade/Event: runs when the card enters play (for
	// events: when played, before discarding).
	OnPlay func(g *Game, e Entity) []Message
	// Reaction hook in addition to the kind-generic handling.
	React func(g *Game, e Entity, msg Message) []Message
	// Extra activated abilities.
	Abilities func(g *Game, e Entity) []Ability
	// Hero-specific hooks.
	HeroAbilities func(g *Game, p *Player) []Ability
	// HeroSetup runs at game start after opening hands are drawn.
	HeroSetup func(g *Game, p *Player) []Message
	// HandSizeBonus adds to the identity's hand size (e.g. Iron Man).
	HandSizeBonus func(g *Game, p *Player) int
	// VillainDamageable gates damage dynamically (e.g. Ultron III while
	// a Drone is in play, Norman Osborn converting damage to infamy).
	// The hook may mutate state (consume counters, flip personas) and
	// returns whether the damage still applies.
	VillainDamageable func(g *Game, v *Villain, damage int) bool
	// Villain: customize stage advancement (e.g. Rhino stage I undamageable).
	VillainStage func(g *Game, v *Villain, nextStage int) []Message
	// Villain forced behavior on activation; default scheme/attack applies
	// when nil.
	VillainActivate func(g *Game, v *Villain, p *Player) []Message
	// Treachery resolution; default discards the treachery with no effect.
	ResolveTreachery func(g *Game, t *Treachery, p *Player) []Message
	// Attachment effects on attach/detach.
	OnAttach   func(g *Game, t *Attachment, target EntityID) []Message
	OnDetach   func(g *Game, t *Attachment) []Message
	// Damage modifiers, e.g. Retaliate-like auras or extra consequential
	// damage. Return added effects keyed by event.
	Modifiers func(g *Game, e Entity, target EntityID) []Modifier
}

// Modifier is a persistent stat adjustment an entity applies to a target.
type Modifier struct {
	Kind  string // "attack" | "thwart" | "defense" | "scheme" | "hp" | "atk_cost"...
	Value int
	// Until holds an optional expiry ("end_of_round").
	Until string
}

// implemented is filled by the cards packages via RegisterBehavior.
var behaviorRegistry = map[string]*Behavior{}

// Implemented reports whether the card has hand-written game logic.
func Implemented(code string) bool {
	_, ok := behaviorRegistry[data.BaseCode(code)]
	return ok
}

// RegisterBehavior installs hand-written behavior for a card code. Intended
// to be called from card package init functions.
func RegisterBehavior(code string, b *Behavior) {
	behaviorRegistry[code] = b
}

// behavior returns the registered behavior or the shared generic one.
// Side-specific registrations (e.g. Norman Osborn vs Green Goblin) win
// over base-code registrations.
func behavior(code string) *Behavior {
	if b, ok := behaviorRegistry[code]; ok {
		return b
	}
	if b, ok := behaviorRegistry[data.BaseCode(code)]; ok {
		return b
	}
	return genericBehavior
}

// genericBehavior is the fallback for cards without hand-written logic: pure
// stats, keywords from the data layer, no text effects.
var genericBehavior = &Behavior{}
