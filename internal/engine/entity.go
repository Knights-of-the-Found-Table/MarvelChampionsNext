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
	// Obligation resolution when revealed from the encounter deck; the
	// hook moves the card via ObligationResolve messages.
	ResolveObligation func(g *Game, p *Player, card Card) []Message
	// Minion: while engaged with a player, that player cannot thwart
	// (Baron Zemo). Applies to basic thwarts (approximation).
	EngagedBlocksThwart bool
	// Identity: passive cost reduction for a card being paid for (e.g.
	// Living Legend: first ally each round costs 1 less).
	CardCost func(g *Game, p *Player, def *data.CardDef) int
	// Event: resolves a defense event played from the defense prompt.
	// Returns the (possibly modified) Defends message plus extra messages
	// (e.g. draw a card); ok=false when it cannot be played right now.
	DefenseEvent func(g *Game, p *Player, e *EventCard, against EntityID) (Defends, []Message, bool)
	// Support/Upgrade: declares a resource-generating ability that is
	// offered automatically in payment prompts (Super-Soldier Serum,
	// Enhanced Awareness...).
	Resource *ResourceAbility
	// Upgrade: saves the identity from being defeated (Captain America's
	// Helmet); the hook performs the save (set HP, discard the card) and
	// reports whether the defeat was prevented.
	DefeatSave func(g *Game, p *Player, u *Upgrade) bool
	// Upgrade: persistent identity stat bonuses while in play (Captain
	// America's Shield: +1 DEF, retaliate 1).
	IdentityStats func(p *Player) StatBonus
	// Ally: attacking requires discarding a card from hand as an
	// additional cost (Wonder Man).
	AllyAttackDiscardCost bool
	// Upgrade attached to an ally: extra consequential damage after that
	// ally attacks (Enraged).
	ConsequentialBonus int
	// Ally: opens the defeat interrupt window (Red Dagger); destroy runs
	// the standard destruction when the save is declined or impossible.
	AllyDefeatInterrupt func(g *Game, a *Ally, destroy func()) []Message
	// Minion: gates damage dynamically (Thomas Edison while another
	// minion is engaged, Edison's Giant Robot); may inspect state and
	// returns whether the damage still applies.
	MinionDamageable func(g *Game, m *Minion, damage int) bool
	// Event in hand: offered when a treachery is about to resolve against
	// the player (Get Behind Me!); returns the replacement effect, nil =
	// currently unplayable.
	TreacheryInterrupt func(g *Game, p *Player, card Card) []Message
	// Upgrade: automatic damage prevention while in play (Energy
	// Barrier); consumes counters and returns damage prevented plus
	// reflection damage (0 = none).
	DamagePrevention func(g *Game, u *Upgrade, p *Player, n int) (prevented, reflect int)
	// Identity: damage prevention from identity counters (Groot's
	// growth counters); returns the amount prevented.
	IdentityDamagePrevention func(g *Game, p *Player, n int) int
	// Upgrade: offered in the defense prompt as a substitute defense
	// (Bamf! — defend without exhausting); returns the Defends payload
	// (Via is filled in by the engine) plus extra messages.
	DefenseSubstitute func(g *Game, p *Player, u *Upgrade, against EntityID) (Defends, []Message, bool)
	// Ally: may be played from the owner's discard pile (Lockjaw).
	PlayableFromDiscard bool
	// Upgrade attached to an enemy: flat attack reduction while that
	// enemy attacks the owner (Heightened Hearing); auto-consumed.
	AttachedEnemyAttackMod int
	// Ally: consequential damage hits the owner instead (Elektra).
	ConsequentialToOwner bool
}

// ResourceAbility describes an exhaust-to-generate-resources ability.
type ResourceAbility struct {
	Icon    string // generated icon: energy | physical | mental | wild
	HeroOnly    bool // "Hero Resource": hero form required
	EventOnly   bool // only usable when paying for an event
	UsesCounters bool // consumes one counter from the source per use
}

// StatBonus is a persistent stat adjustment applied to an identity.
type StatBonus struct {
	ATK, THW, DEF int
	REC           int
	Retaliate     int
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

// LookupBehavior returns the registered behavior for a card code, or
// the shared generic one. Exported for tests that need to inspect
// hand-written behaviors (e.g. presence of a HeroAbilities hook).
func LookupBehavior(code string) *Behavior {
	return behavior(code)
}

// genericBehavior is the fallback for cards without hand-written logic: pure
// stats, keywords from the data layer, no text effects.
var genericBehavior = &Behavior{}
