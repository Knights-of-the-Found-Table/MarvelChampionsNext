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
	// MainScheme: runs when a stage's a face is revealed (game setup or
	// stage advance), before the scheme flips to its b face; keyed by the
	// stage's a-side card code. The flip is queued after the returned
	// messages, so the effects settle first.
	MainSchemeRevealed func(g *Game, s *MainScheme) []Message
	// Villain forced behavior on activation; default scheme/attack applies
	// when nil.
	VillainActivate func(g *Game, v *Villain, p *Player) []Message
	// Minion forced behavior on activation (Hobgoblin discarding boost
	// cards instead of attacking); default attack/scheme applies when nil.
	MinionActivate func(g *Game, mn *Minion, p *Player) []Message
	// Treachery resolution; default discards the treachery with no effect.
	ResolveTreachery func(g *Game, t *Treachery, p *Player) []Message
	// Boost: resolves when the card is revealed faceup as a boost card
	// (the printed "[star] Boost:" ability).
	Boost func(g *Game, card Card) []Message
	// Attachment effects on attach/detach.
	OnAttach func(g *Game, t *Attachment, target EntityID) []Message
	OnDetach func(g *Game, t *Attachment) []Message
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
	// Living Legend: first ally each round costs 1 less). Also consulted
	// for the played card itself and controlled allies (self discounts,
	// Iron Man's upgrade rebate).
	CardCost func(g *Game, p *Player, def *data.CardDef) int
	// Playable gates the card's presence in the play menu (Scarlet
	// Spider / SP//dr: "play only if you control a Web-Warrior card").
	// nil = always playable.
	Playable func(g *Game, p *Player, def *data.CardDef) bool
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
	// IdentityStatsG is the game-state-aware variant of IdentityStats
	// (Rogue's Touched / Rogue's Jacket, keyed off the upgrade's
	// AttachTo target). Both hooks stack when present.
	IdentityStatsG func(g *Game, p *Player, u *Upgrade) StatBonus
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
	// Minion: source-aware damage gate (Killmonger ignoring Black Panther
	// upgrades); consulted before MinionDamageable.
	MinionDamageableSrc func(g *Game, m *Minion, damage int, src EntityID) bool
	// Minion: the engaged player must defend this minion's attacks with
	// an ally if able (Melter).
	ForceAllyDefense bool
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
	// Upgrade attached to an enemy: flat scheme reduction while attached
	// (Legal Trouble); consulted by schemeValueOf, never consumed.
	AttachedEnemySchemeMod int
	// Ally: consequential damage hits the owner instead (Elektra).
	ConsequentialToOwner bool
	// Side scheme: runs when the scheme is defeated, before it leaves
	// play (victory rewards; the engine handles the removal).
	SideSchemeDefeated func(g *Game, s *SideScheme) []Message
	// Villain/Minion: dynamic attack & scheme bonuses (Juggernaut's
	// momentum counters, Stryfe's hand-type scaling); consulted by
	// attackValue and the scheme placement paths.
	EnemyStatBonus func(g *Game, e Entity) (atk, sch int)
}

// ResourceAbility describes an exhaust-to-generate-resources ability.
type ResourceAbility struct {
	Icon         string // generated icon: energy | physical | mental | wild
	HeroOnly     bool   // "Hero Resource": hero form required
	EventOnly    bool   // only usable when paying for an event
	// UsesCounters consumes one counter from the source per use.
	UsesCounters bool
	// NoExhaust skips the source exhaustion (identity counter resources,
	// e.g. Spider-Ham's toon counters spent as wild).
	NoExhaust bool
	// DamageAttached deals N damage to the upgrade's attached character
	// per use (Clarity of Purpose).
	DamageAttached int
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
