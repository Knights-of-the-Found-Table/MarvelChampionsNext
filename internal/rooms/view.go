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
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Scenario string `json:"scenario"`
	Round    int    `json:"round"`
	Over     bool   `json:"over"`
	Won      bool   `json:"won"`
	Reason   string `json:"reason,omitempty"`

	Villains    []VillainView `json:"villains"`
	MainScheme  *SchemeView   `json:"mainScheme"`
	SideSchemes []SchemeView  `json:"sideSchemes"`
	Minions     []MinionView  `json:"minions"`
	Players     []PlayerView  `json:"players"`
	Log         []string      `json:"log"`

	// Question is the pending question when it belongs to the viewer.
	Question *engine.Question `json:"question,omitempty"`
	// WaitingFor names the player being asked (public info).
	WaitingFor *string `json:"waitingFor,omitempty"`
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
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Threat    int    `json:"threat"`
	MaxThreat int    `json:"maxThreat"`
	Stage     int    `json:"stage,omitempty"`
	Crisis    bool   `json:"crisis,omitempty"`
	Hazard    int    `json:"hazard,omitempty"`
	Acceleration int `json:"acceleration,omitempty"`
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
	Tough    bool   `json:"tough"`
}

type AllyView struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"maxHp"`
	Attack   int    `json:"attack"`
	Thwart   int    `json:"thwart"`
	Exhausted bool `json:"exhausted"`
	Stunned  bool   `json:"stunned"`
}

type CardRef struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type PlayerView struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	UserID    string     `json:"userId,omitempty"`
	Side      string     `json:"side"`
	HeroCode  string     `json:"heroCode"`
	AlterEgo  string     `json:"alterEgoCode"`
	HP        int        `json:"hp"`
	MaxHP     int        `json:"maxHp"`
	Exhausted bool       `json:"exhausted"`
	Stunned   bool       `json:"stunned"`
	Confused  bool       `json:"confused"`
	Tough     bool       `json:"tough"`
	FirstPlayer bool     `json:"firstPlayer"`
	KOed      bool       `json:"koed"`
	FormChanged bool     `json:"formChanged"`

	// Hand is only populated for the owning viewer.
	Hand  []CardRef `json:"hand,omitempty"`
	HandSize int    `json:"handSize"`
	DeckCount  int  `json:"deckCount"`
	DiscardTop *CardRef `json:"discardTop,omitempty"`

	Allies   []AllyView    `json:"allies"`
	Supports []EntityLite  `json:"supports"`
	Upgrades []EntityLite  `json:"upgrades"`
	EncounterDown int      `json:"encounterDown"`
}

type EntityLite struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Exhausted bool   `json:"exhausted"`
}

// BuildView projects the engine state for a viewer. viewerUserID empty =
// spectator. ownerUserID maps a player id to the owning user (may be empty
// while unclaimed).
func BuildView(id int64, name string, g *engine.Game, viewerUserID string, owners map[string]string) *GameView {
	v := &GameView{
		ID:       id,
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
			Crisis: s.Crisis, Hazard: s.Hazard,
		})
	}
	for _, m := range sortedByNum(g.Minions) {
		def := m.EDef()
		v.Minions = append(v.Minions, MinionView{
			ID: string(m.ID), Code: m.Code, Name: def.Name,
			HP: max(0, m.HP()), MaxHP: m.MaxHP,
			Attack: m.AttackVal, Scheme: m.SchemeVal,
			Guard: m.Guard, Stunned: m.Stunned, Tough: m.Tough,
		})
	}

	for _, p := range g.Players {
		pv := PlayerView{
			ID: string(p.ID), Name: p.Name,
			Side: p.Side, HeroCode: p.HeroCode, AlterEgo: p.AlterEgoCode,
			HP: max(0, p.HP()), MaxHP: p.MaxHP,
			Exhausted: p.Exhausted, Stunned: p.Stunned, Confused: p.Confused, Tough: p.Tough,
			FirstPlayer: p.FirstPlayer, KOed: p.KOed, FormChanged: p.FormChanged,
			HandSize: len(p.Hand), DeckCount: len(p.Deck), EncounterDown: len(p.EncounterDown),
			UserID: owners[string(p.ID)],
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
				})
			}
		}
		for _, id := range p.Supports {
			if s := g.Supports[id]; s != nil {
				pv.Supports = append(pv.Supports, EntityLite{ID: string(s.ID), Code: s.Code, Name: s.EDef().Name, Exhausted: s.Exhausted})
			}
		}
		for _, id := range p.Upgrades {
			if u := g.Upgrades[id]; u != nil {
				pv.Upgrades = append(pv.Upgrades, EntityLite{ID: string(u.ID), Code: u.Code, Name: u.EDef().Name, Exhausted: u.Exhausted})
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
