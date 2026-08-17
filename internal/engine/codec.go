package engine

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// The engine's queue and pending question trees hold Message values behind
// an interface. Plain encoding/json cannot round-trip interfaces, so
// messages are wrapped in typed envelopes: {"t": "ChangeForm", "m": {...}}.

type msgEnvelope struct {
	T string          `json:"t"`
	M json.RawMessage `json:"m"`
}

// messageRegistry lists every message type; adding a new message type
// requires one entry here.
var messageRegistry = map[string]reflect.Type{}

func init() {
	prototypes := []Message{
		StartGame{}, BeginRound{}, EndRound{}, BeginPhase{}, EndPhase{},
		PlayerTurnStart{}, PlayerTurnEnd{}, ReadyAll{}, ReadyEntity{},
		ExhaustEntity{}, DrawCards{}, ShufflePlayerDeck{}, DiscardCards{},
		ChangeForm{}, PlayCard{}, ResourcePay{}, BasicThwart{}, BasicAttack{},
		BasicRecover{}, VillainActivates{}, MinionActivates{}, SchemeThreat{},
		ThwartScheme{}, DamageEntity{}, HealEntity{}, StunEntity{},
		ConfuseEntity{}, ToughEntity{}, ClearStun{}, ClearConfuse{},
		Defends{}, DealBoost{}, RevealBoost{}, ClearBoosts{},
		RevealEncounterCard{}, VillainDefeated{}, AdvanceVillainStage{},
		MinionDefeated{}, MainSchemeMaxed{}, SchemeDefeated{},
		ReplaceMainScheme{}, GameOver{}, AskQuestion{},
		WindowAfterEnemyAttacked{}, WindowAfterThwarted{}, RunAbility{},
		ApplyVillainScheme{}, ResourcePayStub{}, RevealNemesisSet{}, SpawnDrone{},
		TakeDeckCard{}, FlipVillainPersona{}, MillPlayerDeck{}, DealEncounterToPlayer{}, BoostEnemyAttack{},
	}
	for _, p := range prototypes {
		messageRegistry[reflect.TypeOf(p).String()] = reflect.TypeOf(p)
	}
}

// MarshalMessage wraps a message in its typed envelope.
func MarshalMessage(m Message) (msgEnvelope, error) {
	raw, err := json.Marshal(m)
	if err != nil {
		return msgEnvelope{}, err
	}
	return msgEnvelope{T: reflect.TypeOf(m).String(), M: raw}, nil
}

// UnmarshalMessage restores a message from its envelope.
func UnmarshalMessage(e msgEnvelope) (Message, error) {
	t, ok := messageRegistry[e.T]
	if !ok {
		return nil, fmt.Errorf("unknown message type %q", e.T)
	}
	v := reflect.New(t)
	if err := json.Unmarshal(e.M, v.Interface()); err != nil {
		return nil, fmt.Errorf("decode %s: %w", e.T, err)
	}
	return v.Elem().Interface().(Message), nil
}

func marshalMessages(msgs []Message) ([]msgEnvelope, error) {
	if len(msgs) == 0 {
		return nil, nil
	}
	out := make([]msgEnvelope, 0, len(msgs))
	for _, m := range msgs {
		e, err := MarshalMessage(m)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func unmarshalMessages(envs []msgEnvelope) ([]Message, error) {
	if len(envs) == 0 {
		return nil, nil
	}
	out := make([]Message, 0, len(envs))
	for _, e := range envs {
		m, err := UnmarshalMessage(e)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}
