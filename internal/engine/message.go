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

	// PlayerTurnStart / PlayerTurnEnd bracket one player's turn in the
	// player phase.
	PlayerTurnStart struct{ Player PlayerID }
	PlayerTurnEnd   struct{ Player PlayerID }

	ReadyAll struct{ Player PlayerID }
	ReadyEntity struct{ ID EntityID }
	ExhaustEntity struct{ ID EntityID }

	DrawCards struct {
		Player PlayerID
		N      int
	}
	ShufflePlayerDeck struct{ Player PlayerID }
	DiscardCards struct {
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
	MainSchemeMaxed struct{ Scheme EntityID }
	SchemeDefeated  struct{ Scheme EntityID }
	ReplaceMainScheme struct{ Scheme EntityID; NextCode string }

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
		VillainID   EntityID
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

	// BoostEnemyAttack permanently raises an enemy's attack (attachments
	// like Goblin Glider).
	BoostEnemyAttack struct {
		Enemy EntityID
		N     int
	}
)

func (StartGame) msg()             {}
func (BeginRound) msg()            {}
func (EndRound) msg()              {}
func (BeginPhase) msg()            {}
func (EndPhase) msg()              {}
func (PlayerTurnStart) msg()       {}
func (PlayerTurnEnd) msg()         {}
func (ReadyAll) msg()              {}
func (ReadyEntity) msg()           {}
func (ExhaustEntity) msg()        {}
func (DrawCards) msg()             {}
func (ShufflePlayerDeck) msg()     {}
func (DiscardCards) msg()          {}
func (ChangeForm) msg()            {}
func (PlayCard) msg()              {}
func (ResourcePay) msg()           {}
func (BasicThwart) msg()           {}
func (BasicAttack) msg()           {}
func (BasicRecover) msg()          {}
func (VillainActivates) msg()      {}
func (MinionActivates) msg()       {}
func (SchemeThreat) msg()          {}
func (ThwartScheme) msg()          {}
func (DamageEntity) msg()          {}
func (HealEntity) msg()            {}
func (StunEntity) msg()            {}
func (ConfuseEntity) msg()         {}
func (ToughEntity) msg()           {}
func (ClearStun) msg()             {}
func (ClearConfuse) msg()          {}
func (Defends) msg()               {}
func (DealBoost) msg()             {}
func (RevealBoost) msg()           {}
func (ClearBoosts) msg()           {}
func (RevealEncounterCard) msg()   {}
func (VillainDefeated) msg()       {}
func (AdvanceVillainStage) msg()   {}
func (MinionDefeated) msg()        {}
func (MainSchemeMaxed) msg()       {}
func (SchemeDefeated) msg()        {}
func (ReplaceMainScheme) msg()     {}
func (GameOver) msg()              {}
func (AskQuestion) msg()           {}
func (WindowAfterEnemyAttacked) msg() {}
func (WindowAfterThwarted) msg()   {}
func (RunAbility) msg()            {}
func (RevealNemesisSet) msg()      {}
func (SpawnDrone) msg()            {}
func (TakeDeckCard) msg()          {}
func (FlipVillainPersona) msg()    {}
func (MillPlayerDeck) msg()        {}
func (DealEncounterToPlayer) msg() {}
func (BoostEnemyAttack) msg()      {}
