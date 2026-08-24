package engine

// Message is a unit of game progression. The engine processes messages from a
// queue; entities may react to any message by returning additional messages.
// This mirrors the message-driven design of the reference Haskell engine.
type Message interface{ msg() }

// Phase names for the round structure.
type Phase string

const (
	PhaseSetup    Phase = "setup"
	PhaseResource Phase = "resource"
	PhasePlayer   Phase = "player"
	PhaseVillain  Phase = "villain"
)

type (
	// StartGame kicks off scenario setup after decks are loaded.
	StartGame struct{}

	BeginRound struct{}
	EndRound   struct{}

	BeginPhase struct{ Phase Phase }
	EndPhase   struct{ Phase Phase }

	// DiscardToHandSize runs the end-of-player-phase step where each player
	// (in player order) may discard any number of cards and must discard
	// down to their hand size.
	DiscardToHandSize struct{ Player PlayerID }
	// FinishPlayerPhase completes the end-of-player-phase sequence: players
	// draw up to hand size, ready all of their cards (and exhausted
	// encounter cards), and until-end-of-player-phase effects expire.
	FinishPlayerPhase struct{}
	// PassFirstPlayerToken passes the first player token clockwise at the
	// end of the round.
	PassFirstPlayerToken struct{}

	// ResolveMulligan offers the setup mulligan question to a player.
	ResolveMulligan struct{ Player PlayerID }
	// MulliganCard discards one mulliganned hand card and draws a
	// replacement.
	MulliganCard struct {
		Player PlayerID
		CardID string
	}

	// PlayerTurnStart / PlayerTurnEnd bracket one player's turn in the
	// player phase.
	PlayerTurnStart struct{ Player PlayerID }
	PlayerTurnEnd   struct{ Player PlayerID }

	ReadyAll      struct{ Player PlayerID }
	ReadyEntity   struct{ ID EntityID }
	ExhaustEntity struct{ ID EntityID }

	DrawCards struct {
		Player PlayerID
		N      int
	}
	ShufflePlayerDeck struct{ Player PlayerID }
	DiscardCards      struct {
		Player PlayerID
		Cards  []Card
	}

	// ChangeForm flips a player's identity (hero <-> alter-ego).
	ChangeForm struct{ Player PlayerID }

	// PlayCard requests the engine to play the hand card (payment already
	// resolved by the choice layer).
	PlayCard struct {
		Player PlayerID
		Card   Card
		Paid   CostPaid
	}

	// ResourcePay discards cards from hand as resources; used when paying
	// for cards and ability costs.
	ResourcePay struct {
		Player PlayerID
		Cards  []Card
	}

	// BasicPowers
	BasicThwart struct {
		Player PlayerID
		N      int
		Target EntityID // scheme
	}
	BasicAttack struct {
		Player PlayerID
		N      int
		Target EntityID // enemy
	}
	BasicRecover struct{ Player PlayerID }

	// Villain / minion activation
	VillainActivates struct {
		VillainID EntityID
		// Player is the player the villain activates against.
		Player PlayerID
	}
	MinionActivates struct {
		MinionID EntityID
		Player   PlayerID
	}
	// MinionActivations drives the villain-phase step where the minions
	// engaged with a player activate against them, in that player's chosen
	// order.
	MinionActivations struct{ Player PlayerID }
	// AskMinionOrder asks a player which of the remaining engaged minions
	// activates next.
	AskMinionOrder struct {
		Player    PlayerID
		Remaining []EntityID
	}
	// AskAttack builds and asks the defense (and interrupt) prompt for an
	// enemy attack at resolve time, after boost cards have been revealed
	// (Trigger "" asks the defense question only, for minion attacks).
	AskAttack struct {
		Enemy   EntityID
		Player  PlayerID
		Trigger string `json:"trigger,omitempty"`
	}
	// OtherDefenders offers the defense of an attack to the remaining
	// players after the attacked player chose to ask for a substitute
	// defender (first willing player defends; if all decline, the attack
	// resolves undefended).
	OtherDefenders struct {
		Against   EntityID
		For       PlayerID
		Remaining []PlayerID
	}
	// AskOtherAction hands the asked player a turn-like menu at the
	// active (requester) player's request: they perform one action of
	// their choice — or nothing — after which the requester's turn
	// continues.
	AskOtherAction struct {
		Asked     PlayerID
		Requester PlayerID
	}

	// SchemeThreat adds threat; ThwartScheme removes it.
	SchemeThreat struct {
		Scheme EntityID
		N      int
		Source EntityID
	}
	ThwartScheme struct {
		Scheme EntityID
		N      int
		Source EntityID
	}

	// Damage / heal / status
	DamageEntity struct {
		Target EntityID
		Damage int
		Source EntityID
	}
	HealEntity struct {
		Target EntityID
		N      int
	}
	StunEntity    struct{ Target EntityID }
	ConfuseEntity struct{ Target EntityID }
	ToughEntity   struct{ Target EntityID }
	ClearStun     struct{ Target EntityID }
	ClearConfuse  struct{ Target EntityID }

	// Defend selection resolved: Defender may be a player or ally.
	Defends struct {
		Defender EntityID
		Against  EntityID // attacking enemy
		// Undefended marks "take the attack" (no defense bonus).
		Undefended bool
		// NoExhaust: apply the DEF bonus without exhausting the identity
		// (Bamf!).
		NoExhaust bool
		// Defense-event modifiers:
		PreventAll   bool // Shield Block: prevent all damage
		ExtraPrevent int  // Wiggle Room: flat prevention
		DefBonus     int  // Expert Defense: extra DEF while defending
		// Via names a substitute-defense card (Bamf!).
		Via string `json:"via,omitempty"`
	}

	// Boost cards
	DealBoost struct {
		Enemy EntityID
	}
	RevealBoost struct {
		Enemy EntityID
	}
	ClearBoosts struct{ Enemy EntityID }

	// Encounter card flow
	RevealEncounterCard struct {
		Player PlayerID
		Card   Card
	}

	// Villain lifecycle
	VillainDefeated     struct{ VillainID EntityID }
	AdvanceVillainStage struct{ VillainID EntityID }
	MinionDefeated      struct{ MinionID EntityID }

	// Schemes
	MainSchemeMaxed   struct{ Scheme EntityID }
	SchemeDefeated    struct{ Scheme EntityID }
	ReplaceMainScheme struct {
		Scheme   EntityID
		NextCode string
	}
	// FlipMainScheme turns a main scheme from its revealed a face to the
	// b face of the current stage; queued after the a face's reveal
	// effects so they settle first.
	FlipMainScheme struct{ Scheme EntityID }

	// Game end
	GameOver struct {
		Won    bool
		Reason string
	}

	// AskQuestion surfaces a question to a player and pauses the queue.
	AskQuestion struct {
		Player   PlayerID
		Question *Question
	}

	// Window notifications that reactive abilities can hook into.
	WindowAfterEnemyAttacked struct {
		Enemy  EntityID
		Player PlayerID
	}
	WindowAfterThwarted struct {
		Player PlayerID
		Scheme EntityID
	}

	// RunAbility records an activated ability usage for limits.
	RunAbility struct {
		Player PlayerID
		Source EntityID
		Index  int
	}

	// RevealNemesisSet spawns a player's nemesis set (minion + side scheme
	// into play, rest to nemesis discard).
	RevealNemesisSet struct{ Player PlayerID }

	// SpawnDrone puts the top card of a player's deck into play as a
	// facedown Drone minion engaged with them.
	SpawnDrone struct{ Player PlayerID }

	// TakeDeckCard moves a specific card from a player's deck to their
	// hand. FromTop > 0 discards the other top-N cards instead of the
	// whole-deck shuffle that callers usually push separately.
	TakeDeckCard struct {
		Player  PlayerID
		CardID  string
		FromTop int
	}

	// FlipVillainPersona swaps a double-sided villain between its a/b
	// sides (Risky Business).
	FlipVillainPersona struct {
		VillainID    EntityID
		FlipToNorman bool
	}

	// MillPlayerDeck discards N cards from the top of a player's deck.
	MillPlayerDeck struct {
		Player PlayerID
		N      int
	}

	// DealEncounterToPlayer deals one facedown encounter card to a player
	// (revealed in the next villain-phase reveal step).
	DealEncounterToPlayer struct{ Player PlayerID }
	// EngageMinion engages a minion with a player (Get Over Here!).
	EngageMinion struct {
		MinionID EntityID
		Player   PlayerID
	}
	// DiscardEncounterCard removes a specific card from the encounter deck
	// and discards it (Heimdall).
	DiscardEncounterCard struct{ Card Card }
	// AddInfamyMsg adds infamy counters to the Criminal Enterprise
	// environment, removing madness counters from State of Madness
	// instead when Criminal Enterprise is not in play.
	AddInfamyMsg struct {
		Env       string
		N         int
		OrMadness int
	}

	// BoostEnemyAttack permanently raises an enemy's attack (attachments
	// like Goblin Glider).
	BoostEnemyAttack struct {
		Enemy EntityID
		N     int
	}
	// BoostActivation raises an enemy's boost count for the current
	// activation only (cleared with ClearBoosts).
	BoostActivation struct {
		Enemy EntityID
		N     int
	}

	// ObligationResolve moves a resolving obligation to its owner's
	// discard pile or removes it from the game.
	ObligationResolve struct {
		Player PlayerID
		Card   Card
		Remove bool
	}

	// DiscardControlled removes a player's ally/support/upgrade from play
	// and puts the card into their discard pile.
	DiscardControlled struct {
		Player PlayerID
		ID     EntityID
	}

	// AddAccelerationToken adds an acceleration token to a scheme.
	AddAccelerationToken struct{ Scheme EntityID }

	// RevealNextEncounter reveals the top card of the encounter deck to a
	// player (surge and similar effects).
	RevealNextEncounter struct{ Player PlayerID }

	// PlayDefenseEvent plays an event chosen from the defense prompt; the
	// behavior's DefenseEvent hook builds the resulting Defends message.
	PlayDefenseEvent struct {
		Player  PlayerID
		Card    Card
		Paid    CostPaid
		Against EntityID
	}

	// AddEntityCounter adjusts the counters on an entity (Hawkeye's
	// arrows, Quinjet time counters, "Uses (N X)" upgrades...).
	AddEntityCounter struct {
		ID EntityID
		N  int
	}

	// ReturnControlled moves a player's in-play card back to their hand
	// (Shield Toss returning the shield).
	ReturnControlled struct {
		Player PlayerID
		ID     EntityID
	}

	// AllyEntersPlayFree puts an ally into play under a player's control
	// without paying its cost (Quinjet, Make the Call). FromOwner ""
	// takes the card from the player's hand, otherwise from that
	// player's discard pile.
	AllyEntersPlayFree struct {
		Player    PlayerID
		Card      Card
		FromOwner PlayerID `json:"fromOwner,omitempty"`
	}

	// MinionEntersPlay announces a minion entering play (Hawkeye's
	// response window).
	MinionEntersPlay struct {
		MinionID EntityID
		Player   PlayerID
	}

	// AttachUpgrade attaches an in-play upgrade to a friendly character
	// or scheme and applies its bonuses (Honorary Avenger, Enraged,
	// Followed).
	AttachUpgrade struct {
		ID         EntityID
		Target     EntityID
		MaxHP      int    // target max HP bonus
		ATK        int    // target attack bonus (allies)
		THW        int    // target thwart bonus (allies)
		GrantTrait string // trait granted to the target
	}
	// CostDiscountApply grants a player a pending one-shot cost reduction
	// (Helicarrier).
	CostDiscountApply struct {
		Player PlayerID
		Amount int
	}
	// SetAntForm grants/removes the giant/tiny identity traits (Ant-Man).
	SetAntForm struct {
		Player PlayerID
		Form   string // "giant" | "tiny" | "" to clear
	}
	// SetMassForm flips or sets Vision's dense/intangible mass form
	// (empty Form flips).
	SetMassForm struct {
		Player PlayerID
		Form   string `json:"form,omitempty"` // "dense" | "intangible" | "" to flip
	}
	// SpawnSymbiote puts an Enraged Symbiote into play engaged with the
	// first player (Struggle for Control).
	SpawnSymbiote struct{}
	// AllForOneDamage deals 3 + 1-per-exhausted-Avenger damage (computed
	// at resolve time).
	AllForOneDamage struct {
		Player PlayerID
		Target EntityID
	}
	// HoodFoulPlay resolves The Hood's Foul Play against a player.
	HoodFoulPlay struct {
		Player PlayerID
		N      int
	}
	// AddProgressCounters adjusts Ironheart's progress counters.
	AddProgressCounters struct {
		Player PlayerID
		N      int
	}
	// SwapHeroSide switches the identity to another printed hero side
	// (Ironheart's Level Up!).
	SwapHeroSide struct {
		Player   PlayerID
		HeroCode string
	}
	// ReplaySideSchemeReveal re-runs a side scheme's When Revealed
	// (Citywide Crisis).
	ReplaySideSchemeReveal struct{ Scheme EntityID }
	// ChangeFormAgain flips the identity ignoring the once-per-turn limit
	// (Resize, Swarm Tactics).
	ChangeFormAgain struct{ Player PlayerID }
	// TempHandSizeMsg grants an until-end-of-phase hand-size bonus
	// (Assess the Situation).
	TempHandSizeMsg struct {
		Player PlayerID
		N      int
	}
	// RapidReturn puts a just-defeated ally back into play from its
	// owner's discard pile with 1 damage (Rapid Response).
	RapidReturn struct {
		Player PlayerID
		Code   string
	}
	// AddVengeance adds a vengeance counter to Drax and raises his ATK
	// bonus accordingly.
	AddVengeance struct{ Player PlayerID }
	// ResolveTechnique runs a technique upgrade's Special after it was
	// put into play by an effect (Combat Ready).
	ResolveTechnique struct {
		Player PlayerID
		Code   string
	}
	// DiscardAttachmentMsg removes an attachment from play into the
	// encounter discard (Lethal Weapon).
	DiscardAttachmentMsg struct{ ID EntityID }
	// BunkerDiscard asks a player to discard 2 cards (Champions Mobile
	// Bunker).
	BunkerDiscard struct{ Player PlayerID }
	// MillEncounter discards N cards off the encounter deck (Chaos
	// Magic).
	MillEncounter struct{ N int }
	// TopDeckPick moves one of the deck's top 3 to hand and the rest to
	// the bottom (Agatha Harkness).
	TopDeckPick struct {
		Player PlayerID
		CardID string
	}
	// SlippingSanityMill mills 5 encounter cards, placing 1 main-scheme
	// threat per star (boost) icon.
	SlippingSanityMill struct{ Player PlayerID }
	// IndirectDamage deals N damage that the player distributes among
	// their own characters, one point at a time (GMW box).
	IndirectDamage struct {
		Player PlayerID
		N      int
	}
	// BarrageCharge resolves the Badoon Ship's "Charge Up" special: one
	// barrage counter, then the 4-counter payoff (GMW).
	BarrageCharge struct{}
	// CollectCard moves a card faceup into The Collection (the
	// Collector's game area).
	CollectCard struct{ Card Card }

	// EventPlayed announces that a player played an event card
	// (Morphogenetics, Embiggen!, Shrink).
	EventPlayed struct {
		Player PlayerID
		Card   Card
	}

	// SetEventBonus records a pending bonus for the event currently being
	// resolved (Embiggen! +2 damage, Shrink +2 threat removal); consumed
	// by the first damage/threat removal from that player.
	SetEventBonus struct {
		Player PlayerID
		Damage int
		Threat int
	}

	// ReturnDiscardCard moves a card from a player's discard pile back to
	// their hand (Morphogenetics).
	ReturnDiscardCard struct {
		Player PlayerID
		CardID string
	}

	// DiscardToBottom moves a discard-pile card to the bottom of the
	// player's deck (Aamir Khan).
	DiscardToBottom struct {
		Player PlayerID
		CardID string
	}

	// AllyDefeated opens the interrupt window before an ally is
	// destroyed; AllyDestroyed performs the destruction.
	AllyDefeated  struct{ AllyID EntityID }
	AllyDestroyed struct{ AllyID EntityID }

	// SupportStoreCard attaches a facedown hand card to a support
	// (Bruno Carrelli); SupportRetrieveCards takes stored cards back.
	SupportStoreCard struct {
		ID   EntityID
		Card Card
	}
	SupportRetrieveCards struct {
		ID    EntityID
		Cards CardList
	}

	// TreacheryWindow opens the interrupt window before a revealed
	// treachery resolves (Get Behind Me!); TreacheryResolve performs the
	// (possibly cancelled) resolution.
	TreacheryWindow struct {
		Player PlayerID
		Card   Card
	}
	TreacheryResolve struct {
		Player    PlayerID
		Card      Card
		Cancelled bool
	}

	// ConsumeHandCard moves a hand card to its owner's discard pile (the
	// event card of a played interrupt).
	ConsumeHandCard struct {
		Player PlayerID
		CardID string
	}

	// PlayDiscardAlly plays an ally from the player's discard pile paying
	// its cost (Lockjaw).
	PlayDiscardAlly struct {
		Player PlayerID
		Card   Card
		Paid   CostPaid
	}

	// ApplyStatBonus grants an identity until-end-of-phase stat bonuses
	// (Morale Boost).
	ApplyStatBonus struct {
		Target        PlayerID
		ATK, THW, DEF int
	}
	// AllyStatBonus grants an ally until-end-of-phase stat bonuses
	// (Vision, Lead from the Front).
	AllyStatBonus struct {
		Ally     EntityID
		ATK, THW int
	}

	// WindowDefended announces a resolved defense with the damage actually
	// taken (Unflappable, Under Control). Via names a substitute-defense
	// card (Bamf!).
	WindowDefended struct {
		Defender    EntityID
		Against     EntityID
		DamageTaken int
		Via         string `json:"via,omitempty"`
	}

	// SenseEnterPlay plays a Sense upgrade from a player's Sense deck into
	// play (Daredevil), optionally paying its cost first.
	SenseEnterPlay struct {
		Player PlayerID
		Card   Card
	}

	// ShuffleIntoDeck moves a discard-pile card into the player's deck and
	// shuffles (Karen Page).
	ShuffleIntoDeck struct {
		Player PlayerID
		CardID string
	}

	// GrantTrait grants an entity a dynamic trait (Billy Club's Aerial).
	GrantTrait struct {
		Target EntityID
		Trait  string
	}

	// InvokeSpecial resolves the top card of a side deck (Doctor
	// Strange's Invocations); ReturnToTop keeps the card on top of the
	// side deck instead of the side discard (Master of the Mystic Arts).
	InvokeSpecial struct {
		Player      PlayerID
		Card        Card
		ReturnToTop bool
	}

	// AllyAttackWindow announces an ally attacking (Iron Fist's rider).
	AllyAttackWindow struct {
		Ally   EntityID
		Target EntityID
	}
	// AllyThwartWindow announces an ally thwarting (Daredevil's rider).
	AllyThwartWindow struct {
		Ally   EntityID
		Scheme EntityID
	}

	// SideDeckDiscardTop discards the top card of a player's side deck
	// (Wong, Natural Talent).
	SideDeckDiscardTop struct{ Player PlayerID }

	// UpgradeEnterPlay puts an upgrade card into play from the player's
	// hand, deck or discard pile without paying (Daytripper's Bamf!).
	UpgradeEnterPlay struct {
		Player PlayerID
		Card   Card
	}

	// SideDeckToHand moves a card from the player's side deck to hand
	// (Echo's tucked events).
	SideDeckToHand struct {
		Player PlayerID
		CardID string
	}

	// RecycleFromDiscard moves a card from another player's discard pile
	// to a player's hand (Study the Tape).
	RecycleFromDiscard struct {
		Player PlayerID
		From   PlayerID
		CardID string
	}

	// SwapHandWithDeckTop exchanges a hand card with the deck's top card
	// (Domino).
	SwapHandWithDeckTop struct {
		Player PlayerID
		CardID string
	}
)

func (StartGame) msg()                {}
func (BeginRound) msg()               {}
func (EndRound) msg()                 {}
func (BeginPhase) msg()               {}
func (EndPhase) msg()                 {}
func (DiscardToHandSize) msg()        {}
func (FinishPlayerPhase) msg()        {}
func (PassFirstPlayerToken) msg()     {}
func (ResolveMulligan) msg()          {}
func (MulliganCard) msg()             {}
func (PlayerTurnStart) msg()          {}
func (PlayerTurnEnd) msg()            {}
func (ReadyAll) msg()                 {}
func (ReadyEntity) msg()              {}
func (ExhaustEntity) msg()            {}
func (DrawCards) msg()                {}
func (ShufflePlayerDeck) msg()        {}
func (DiscardCards) msg()             {}
func (ChangeForm) msg()               {}
func (PlayCard) msg()                 {}
func (ResourcePay) msg()              {}
func (BasicThwart) msg()              {}
func (BasicAttack) msg()              {}
func (BasicRecover) msg()             {}
func (VillainActivates) msg()         {}
func (MinionActivates) msg()          {}
func (MinionActivations) msg()        {}
func (AskMinionOrder) msg()           {}
func (AskAttack) msg()                {}
func (OtherDefenders) msg()           {}
func (AskOtherAction) msg()           {}
func (SchemeThreat) msg()             {}
func (ThwartScheme) msg()             {}
func (DamageEntity) msg()             {}
func (HealEntity) msg()               {}
func (StunEntity) msg()               {}
func (ConfuseEntity) msg()            {}
func (ToughEntity) msg()              {}
func (ClearStun) msg()                {}
func (ClearConfuse) msg()             {}
func (Defends) msg()                  {}
func (DealBoost) msg()                {}
func (RevealBoost) msg()              {}
func (ClearBoosts) msg()              {}
func (RevealEncounterCard) msg()      {}
func (VillainDefeated) msg()          {}
func (AdvanceVillainStage) msg()      {}
func (MinionDefeated) msg()           {}
func (MainSchemeMaxed) msg()          {}
func (SchemeDefeated) msg()           {}
func (ReplaceMainScheme) msg()        {}
func (FlipMainScheme) msg()           {}
func (GameOver) msg()                 {}
func (AskQuestion) msg()              {}
func (WindowAfterEnemyAttacked) msg() {}
func (WindowAfterThwarted) msg()      {}
func (RunAbility) msg()               {}
func (RevealNemesisSet) msg()         {}
func (SpawnDrone) msg()               {}
func (TakeDeckCard) msg()             {}
func (FlipVillainPersona) msg()       {}
func (MillPlayerDeck) msg()           {}
func (DealEncounterToPlayer) msg()    {}
func (EngageMinion) msg()             {}
func (DiscardEncounterCard) msg()     {}
func (AddInfamyMsg) msg()             {}
func (BoostEnemyAttack) msg()         {}
func (BoostActivation) msg()          {}
func (ObligationResolve) msg()        {}
func (DiscardControlled) msg()        {}
func (AddAccelerationToken) msg()     {}
func (RevealNextEncounter) msg()      {}
func (PlayDefenseEvent) msg()         {}
func (AddEntityCounter) msg()         {}
func (ReturnControlled) msg()         {}
func (AllyEntersPlayFree) msg()       {}
func (MinionEntersPlay) msg()         {}
func (AttachUpgrade) msg()            {}
func (CostDiscountApply) msg()        {}
func (SetAntForm) msg()               {}
func (SetMassForm) msg()              {}
func (SpawnSymbiote) msg()            {}
func (AllForOneDamage) msg()          {}
func (HoodFoulPlay) msg()             {}
func (AddProgressCounters) msg()      {}
func (SwapHeroSide) msg()             {}
func (ReplaySideSchemeReveal) msg()   {}
func (ChangeFormAgain) msg()          {}
func (TempHandSizeMsg) msg()          {}
func (RapidReturn) msg()              {}
func (AddVengeance) msg()             {}
func (ResolveTechnique) msg()         {}
func (DiscardAttachmentMsg) msg()     {}
func (BunkerDiscard) msg()            {}
func (MillEncounter) msg()            {}
func (TopDeckPick) msg()              {}
func (SlippingSanityMill) msg()       {}
func (IndirectDamage) msg()        {}
func (BarrageCharge) msg()         {}
func (CollectCard) msg()           {}
func (EventPlayed) msg()              {}
func (SetEventBonus) msg()            {}
func (ReturnDiscardCard) msg()        {}
func (DiscardToBottom) msg()          {}
func (AllyDefeated) msg()             {}
func (AllyDestroyed) msg()            {}
func (SupportStoreCard) msg()         {}
func (SupportRetrieveCards) msg()     {}
func (TreacheryWindow) msg()          {}
func (TreacheryResolve) msg()         {}
func (ConsumeHandCard) msg()          {}
func (PlayDiscardAlly) msg()          {}
func (ApplyStatBonus) msg()           {}
func (AllyStatBonus) msg()            {}
func (WindowDefended) msg()           {}
func (SenseEnterPlay) msg()           {}
func (ShuffleIntoDeck) msg()          {}
func (GrantTrait) msg()               {}
func (InvokeSpecial) msg()            {}
func (AllyAttackWindow) msg()         {}
func (AllyThwartWindow) msg()         {}
func (SideDeckDiscardTop) msg()       {}
func (UpgradeEnterPlay) msg()         {}
func (SideDeckToHand) msg()           {}
func (RecycleFromDiscard) msg()       {}
func (SwapHandWithDeckTop) msg()      {}
