// Package rooms keeps active games in memory, applies player answers,
// persists snapshots and broadcasts per-viewer game views over WebSocket.
package rooms

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// GameView is the client-facing projection of an engine.Game. Hands are
// redacted per viewer; the pending question is only included for the player
// being asked (and for spectators it is omitted entirely).
type GameView struct {
	// ID is the game's opaque public token (URL-safe), not the DB key.
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Scenario string     `json:"scenario"`
	Round    int        `json:"round"`
	Over     bool       `json:"over"`
	Won      bool       `json:"won"`
	Reason   engine.Msg `json:"reason,omitempty"`

	Villains    []VillainView `json:"villains"`
	MainScheme  *SchemeView   `json:"mainScheme"`
	SideSchemes []SchemeView  `json:"sideSchemes"`
	Minions     []MinionView  `json:"minions"`
	Players     []PlayerView  `json:"players"`
	// Attachments are cards attached to entities (host id resolves the
	// stacking position on the board); Treacheries are persistent in-play
	// treacheries; Environments sit in the encounter area.
	Attachments  []AttachmentView  `json:"attachments,omitempty"`
	Treacheries  []AttachmentView  `json:"treacheries,omitempty"`
	Environments []EntityLite      `json:"environments,omitempty"`
	Log          []engine.LogEntry `json:"log"`

	// Question is the pending question when it belongs to the viewer.
	Question *engine.Question `json:"question,omitempty"`
	// WaitingFor names the player being asked (public info).
	WaitingFor *string `json:"waitingFor,omitempty"`
	// EncounterCount is the encounter draw deck size (public info).
	EncounterCount int `json:"encounterCount,omitempty"`
	// EncounterDiscardCount / EncounterDiscardTop expose the encounter
	// discard pile: its size and the top card (revealed face-up).
	EncounterDiscardCount int      `json:"encounterDiscardCount,omitempty"`
	EncounterDiscardTop   *CardRef `json:"encounterDiscardTop,omitempty"`
}

type VillainView struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Stage    int    `json:"stage"`
	StageLbl string `json:"stageLabel"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"maxHp"`
	Scheme   int    `json:"scheme"`
	Attack   int    `json:"attack"`
	Stunned  bool   `json:"stunned"`
	Confused bool   `json:"confused"`
	Tough    bool   `json:"tough"`
	Boosts   int    `json:"boosts"`
}

type SchemeView struct {
	ID           string `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	Threat       int    `json:"threat"`
	MaxThreat    int    `json:"maxThreat"`
	Stage        int    `json:"stage,omitempty"`
	Crisis       bool   `json:"crisis,omitempty"`
	Hazard       int    `json:"hazard,omitempty"`
	PlayerSide   bool   `json:"playerSide,omitempty"`
	Acceleration int    `json:"acceleration,omitempty"`
}

type MinionView struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"maxHp"`
	Attack   int    `json:"attack"`
	Scheme   int    `json:"scheme"`
	Guard    bool   `json:"guard"`
	Stunned  bool   `json:"stunned"`
	Confused bool   `json:"confused,omitempty"`
	Tough    bool   `json:"tough,omitempty"`
	// EngagedWith is the player the minion is engaged with (board layout).
	EngagedWith string `json:"engagedWith,omitempty"`
	// FaceDown marks a facedown Drone (Ultron): render the player card back.
	FaceDown bool `json:"faceDown,omitempty"`
}

type AllyView struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	HP        int    `json:"hp"`
	MaxHP     int    `json:"maxHp"`
	Attack    int    `json:"attack"`
	Thwart    int    `json:"thwart"`
	Exhausted bool   `json:"exhausted"`
	Stunned   bool   `json:"stunned"`
	Confused  bool   `json:"confused,omitempty"`
	Tough     bool   `json:"tough,omitempty"`
	Counters  int    `json:"counters,omitempty"`
}

type CardRef struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type PlayerView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	UserID   string `json:"userId,omitempty"`
	Side     string `json:"side"`
	HeroCode string `json:"heroCode"`
	AlterEgo string `json:"alterEgoCode"`
	// 当前面的卡牌名（title 展示用，区别于玩家名）。
	HeroName     string `json:"heroName,omitempty"`
	AlterEgoName string `json:"alterEgoName,omitempty"`
	HP           int    `json:"hp"`
	MaxHP        int    `json:"maxHp"`
	Exhausted    bool   `json:"exhausted"`
	Stunned      bool   `json:"stunned"`
	Confused     bool   `json:"confused"`
	Tough        bool   `json:"tough"`
	FirstPlayer  bool   `json:"firstPlayer"`
	KOed         bool   `json:"koed"`
	FormChanged  bool   `json:"formChanged"`

	// Hand is only populated for the owning viewer.
	Hand           []CardRef `json:"hand,omitempty"`
	HandSize       int       `json:"handSize"`
	DeckCount      int       `json:"deckCount"`
	DiscardCount   int       `json:"discardCount"`
	SenseDeckCount int       `json:"senseDeckCount,omitempty"`
	DiscardTop     *CardRef  `json:"discardTop,omitempty"`

	Allies        []AllyView   `json:"allies"`
	Supports      []EntityLite `json:"supports"`
	Upgrades      []EntityLite `json:"upgrades"`
	EncounterDown int          `json:"encounterDown"`
}

type EntityLite struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Exhausted bool   `json:"exhausted"`
	Counters  int    `json:"counters,omitempty"`
	// AttachTo names the non-player host this upgrade is attached to
	// (Under Surveillance → main scheme); the board renders it beside the
	// host instead of in the owner's upgrade row.
	AttachTo string `json:"attachTo,omitempty"`
}

// AttachmentView is a card attached to (or in the case of persistent
// treacheries, associated with) a host entity.
type AttachmentView struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	// Host is the entity the card is attached to ("" = unattached).
	Host string `json:"host,omitempty"`
}

// BuildView projects the engine state for a viewer. viewerUserID empty =
// spectator. ownerUserID maps a player id to the owning user (may be empty
// while unclaimed).
func BuildView(token, name string, g *engine.Game, viewerUserID string, owners map[string]string) *GameView {
	v := &GameView{
		ID:       token,
		Name:     name,
		Scenario: g.Scenario().Name,
		Round:    g.Round,
		Over:     g.Over,
		Won:      g.Won,
		Reason:   g.Reason,
		Log:      g.Log,
	}

	for _, vil := range sortedByNum(g.Villains) {
		def := vil.EDef()
		v.Villains = append(v.Villains, VillainView{
			ID: string(vil.ID), Code: vil.Code, Name: def.Name,
			Stage: vil.Stage, StageLbl: def.StageLabel,
			HP: max(0, vil.HP()), MaxHP: vil.MaxHP,
			Scheme: vil.SchemeVal, Attack: vil.AttackVal,
			Stunned: vil.Stunned, Confused: vil.Confused, Tough: vil.Tough,
			Boosts: vil.BoostCount,
		})
	}

	if g.MainScheme != nil {
		s := g.MainScheme
		v.MainScheme = &SchemeView{
			ID: string(s.ID), Code: s.Code, Name: s.EDef().Name,
			Threat: s.Threat, MaxThreat: s.MaxThreat, Stage: s.Stage,
			Acceleration: s.AccelerationTokens,
		}
	}
	for _, s := range sortedByNum(g.SideSchemes) {
		v.SideSchemes = append(v.SideSchemes, SchemeView{
			ID: string(s.ID), Code: s.Code, Name: s.EDef().Name,
			Threat: s.Threat, MaxThreat: s.MaxThreat,
			Crisis: s.Crisis, Hazard: s.Hazard, PlayerSide: s.PlayerSide,
		})
	}
	for _, m := range sortedByNum(g.Minions) {
		def := m.EDef()
		v.Minions = append(v.Minions, MinionView{
			ID: string(m.ID), Code: m.Code, Name: def.Name,
			HP: max(0, m.HP()), MaxHP: m.MaxHP,
			Attack: m.AttackVal, Scheme: m.SchemeVal,
			Guard: m.Guard, Stunned: m.Stunned, Confused: m.Confused, Tough: m.Tough,
			EngagedWith: string(m.EngagedWith), FaceDown: m.IsDrone,
		})
	}
	for _, a := range sortedByNum(g.Attachments) {
		v.Attachments = append(v.Attachments, AttachmentView{
			ID: string(a.ID), Code: a.Code, Name: a.EDef().Name,
			Host: string(a.Target),
		})
	}
	for _, t := range sortedByNum(g.Treacheries) {
		v.Treacheries = append(v.Treacheries, AttachmentView{
			ID: string(t.ID), Code: t.Code, Name: t.EDef().Name,
			Host: string(t.Target),
		})
	}
	for _, e := range sortedByNum(g.Environments) {
		v.Environments = append(v.Environments, EntityLite{ID: string(e.ID), Code: e.Code, Name: e.EDef().Name})
	}

	for _, p := range g.Players {
		pv := PlayerView{
			ID: string(p.ID), Name: p.Name,
			Side: p.Side, HeroCode: p.HeroCode, AlterEgo: p.AlterEgoCode,
			HeroName:     engine.DB.MustLookup(p.HeroCode).Name,
			AlterEgoName: engine.DB.MustLookup(p.AlterEgoCode).Name,
			HP:           max(0, p.HP()), MaxHP: p.MaxHP,
			Exhausted: p.Exhausted, Stunned: p.Stunned, Confused: p.Confused, Tough: p.Tough > 0,
			FirstPlayer: p.FirstPlayer, KOed: p.KOed, FormChanged: p.FormChanged,
			HandSize: len(p.Hand), DeckCount: len(p.Deck), DiscardCount: len(p.Discard), EncounterDown: len(p.EncounterDown),
			SenseDeckCount: len(p.SenseDeck),
			UserID:         owners[string(p.ID)],
		}
		if viewerUserID != "" && pv.UserID == viewerUserID {
			for _, c := range p.Hand {
				pv.Hand = append(pv.Hand, CardRef{ID: c.ID, Code: c.Code, Name: c.Def().Name})
			}
		}
		if len(p.Discard) > 0 {
			top := p.Discard[len(p.Discard)-1]
			pv.DiscardTop = &CardRef{ID: top.ID, Code: top.Code, Name: top.Def().Name}
		}
		for _, id := range p.Allies {
			if a := g.Allies[id]; a != nil {
				pv.Allies = append(pv.Allies, AllyView{
					ID: string(a.ID), Code: a.Code, Name: a.EDef().Name,
					HP: max(0, a.HP()), MaxHP: a.MaxHP,
					Attack: a.AttackVal, Thwart: a.ThwartVal,
					Exhausted: a.Exhausted, Stunned: a.Stunned,
					Confused: a.Confused, Tough: a.Tough, Counters: a.Counters,
				})
			}
		}
		for _, id := range p.Supports {
			if s := g.Supports[id]; s != nil {
				pv.Supports = append(pv.Supports, EntityLite{ID: string(s.ID), Code: s.Code, Name: s.EDef().Name, Exhausted: s.Exhausted, Counters: s.Counters})
			}
		}
		for _, id := range p.Upgrades {
			if u := g.Upgrades[id]; u != nil {
				pv.Upgrades = append(pv.Upgrades, EntityLite{ID: string(u.ID), Code: u.Code, Name: u.EDef().Name, Exhausted: u.Exhausted, Counters: u.Counters, AttachTo: string(u.AttachTo)})
			}
		}
		v.Players = append(v.Players, pv)
	}

	if pq := g.Pending(); pq != nil {
		name := playerName(g, pq.Player)
		v.WaitingFor = &name
		if viewerUserID != "" && owners[string(pq.Player)] == viewerUserID {
			v.Question = pq.Question
		}
	}
	v.EncounterCount = len(g.EncounterDeck)
	v.EncounterDiscardCount = len(g.EncounterDiscard)
	if n := len(g.EncounterDiscard); n > 0 {
		top := g.EncounterDiscard[n-1]
		v.EncounterDiscardTop = &CardRef{ID: top.ID, Code: top.Code, Name: top.Def().Name}
	}
	return v
}

func playerName(g *engine.Game, id engine.PlayerID) string {
	if p := g.Player(id); p != nil {
		return p.Name
	}
	return string(id)
}

type numbered interface{ EID() engine.EntityID }

func sortedByNum[T numbered](m map[engine.EntityID]T) []T {
	out := make([]T, 0, len(m))
	for _, x := range m {
		out = append(out, x)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].EID().Num() < out[j-1].EID().Num(); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
