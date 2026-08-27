// Package cardutil provides shared helpers for hand-written card behavior
// packages: targeting questions, entity listings and common effect builders
// reused across packs.
package cardutil

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// SortedIDs returns the map's entity ids in stable numeric order.
func SortedIDs[T any](m map[engine.EntityID]T) []engine.EntityID {
	ids := make([]engine.EntityID, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j].Num() < ids[j-1].Num(); j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
	return ids
}

// SortedEnemyIDs returns all villains and minions sorted by entity id.
func SortedEnemyIDs(g *engine.Game) []engine.EntityID {
	ids := append(SortedIDs(g.Villains), SortedIDs(g.Minions)...)
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j].Num() < ids[j-1].Num(); j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
	return ids
}

// EnemyLabel renders an enemy with its remaining HP for choice lists.
func EnemyLabel(e engine.Entity) engine.Msg {
	switch t := e.(type) {
	case *engine.Villain:
		return engine.Tf("m.hp", t, t.HP(), t.MaxHP)
	case *engine.Minion:
		return engine.Tf("m.hp", t, t.HP(), t.MaxHP)
	}
	return engine.Tf("m.cardName", e)
}

// EnemyChoices lists all enemies as damage targets.
func EnemyChoices(g *engine.Game, dmg int, source engine.EntityID, mk func(target engine.EntityID) []engine.Message) []engine.Choice {
	var out []engine.Choice
	for _, id := range SortedEnemyIDs(g) {
		e := g.Entity(id)
		out = append(out, engine.Choice{
			Label: EnemyLabel(e), Kind: engine.ChoiceTarget,
			SourceID: id, CardCode: e.ECode(),
		}.Msgs(mk(id)...))
	}
	return out
}

// MinionChoices lists all minions as targets.
func MinionChoices(g *engine.Game, mk func(target engine.EntityID) []engine.Message) []engine.Choice {
	var out []engine.Choice
	for _, id := range SortedIDs(g.Minions) {
		mn := g.Minions[id]
		out = append(out, engine.Choice{
			Label: EnemyLabel(mn), Kind: engine.ChoiceTarget,
			SourceID: id, CardCode: mn.Code,
		}.Msgs(mk(id)...))
	}
	return out
}

// SchemeChoices lists all schemes (main first) as threat targets.
func SchemeChoices(g *engine.Game, mk func(scheme engine.EntityID) []engine.Message) []engine.Choice {
	var out []engine.Choice
	for _, id := range g.Schemes() {
		s := g.Entity(id)
		out = append(out, engine.Choice{
			Label: engine.Tf("m.cardName", s), Kind: engine.ChoiceTarget,
			SourceID: id, CardCode: s.ECode(),
		}.Msgs(mk(id)...))
	}
	return out
}

// ChooseEnemy builds an OnPlay hook that asks for an enemy and deals damage
// to it; the callback returns the damage plus optional extra messages.
func ChooseEnemy(prompt engine.Msg, f func(g *engine.Game, e engine.Entity) (int, []engine.Message)) func(g *engine.Game, e engine.Entity) []engine.Message {
	return func(g *engine.Game, e engine.Entity) []engine.Message {
		pid := e.EOwner()
		if len(g.Enemies()) == 0 {
			return nil
		}
		n, _ := f(g, e)
		var choices []engine.Choice
		for _, id := range SortedEnemyIDs(g) {
			enemy := g.Entity(id)
			choices = append(choices, engine.Choice{
				Label: EnemyLabel(enemy), Kind: engine.ChoiceTarget,
				SourceID: id, CardCode: enemy.ECode(),
			}.Msgs(engine.DamageEntity{Target: id, Damage: n, Source: pid}))
		}
		return []engine.Message{engine.AskQuestion{
			Player:   pid,
			Question: engine.Ask(prompt, choices...),
		}}
	}
}

// ChooseScheme builds an OnPlay hook that asks for a scheme and removes
// threat from it. prompt is the full question prompt (call sites pass
// engine.Tf keys such as "For Justice! — choose a scheme").
func ChooseScheme(prompt engine.Msg, amount func(g *engine.Game, e engine.Entity) int) func(g *engine.Game, e engine.Entity) []engine.Message {
	return func(g *engine.Game, e engine.Entity) []engine.Message {
		pid := e.EOwner()
		schemes := g.Schemes()
		if len(schemes) == 0 {
			return nil
		}
		var choices []engine.Choice
		for _, id := range schemes {
			s := g.Entity(id)
			choices = append(choices, engine.Choice{
				Label: engine.Tf("m.cardName", s), Kind: engine.ChoiceTarget,
				SourceID: id, CardCode: s.ECode(),
			}.Msgs(engine.ThwartScheme{Scheme: id, N: amount(g, e), Source: pid}))
		}
		return []engine.Message{engine.AskQuestion{
			Player:   pid,
			Question: engine.Ask(prompt, choices...),
		}}
	}
}

// ChooseMinion builds an OnPlay hook that deals damage to a chosen minion.
func ChooseMinion(prompt engine.Msg, dmg int) func(g *engine.Game, e engine.Entity) []engine.Message {
	return func(g *engine.Game, e engine.Entity) []engine.Message {
		pid := e.EOwner()
		choices := MinionChoices(g, func(target engine.EntityID) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: target, Damage: dmg, Source: pid}}
		})
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{
			Player:   pid,
			Question: engine.Ask(prompt, choices...),
		}}
	}
}

// Skip returns the standard pass choice.
func Skip() engine.Choice {
	return engine.Choice{ID: "skip", Label: engine.Tf("c.skip"), Kind: engine.ChoicePass}
}

// FirstPlayerID returns the first player's id.
func FirstPlayerID(g *engine.Game) engine.PlayerID {
	for _, p := range g.Players {
		if p.FirstPlayer {
			return p.ID
		}
	}
	if len(g.Players) > 0 {
		return g.Players[0].ID
	}
	return ""
}

// BoostOf returns a card's printed boost icon count.
func BoostOf(c engine.Card) int {
	if c.Def().Boost == nil {
		return 0
	}
	return *c.Def().Boost
}

// Cost returns a card's printed cost (0 when absent).
func Cost(def *data.CardDef) int {
	if def == nil || def.Cost == nil {
		return 0
	}
	return *def.Cost
}

// ExhaustOrPenalty builds the shared obligation template: "You may flip to
// alter-ego form. Choose: exhaust your identity → remove this obligation
// from the game, or apply the penalty and discard it." penaltyMsgs run after
// the obligation is discarded (surge riders etc. are safe to include).
func ExhaustOrPenalty(g *engine.Game, p *engine.Player, card engine.Card, penaltyLabel engine.Msg, penaltyMsgs ...engine.Message) []engine.Message {
	var removeMsgs []engine.Message
	if p.IsHero() && !p.FormChanged && !p.Exhausted {
		removeMsgs = append(removeMsgs, engine.ChangeForm{Player: p.ID})
	}
	removeMsgs = append(removeMsgs,
		engine.ExhaustEntity{ID: p.ID},
		engine.ObligationResolve{Player: p.ID, Card: card, Remove: true},
	)
	penalty := append([]engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}, penaltyMsgs...)
	return []engine.Message{engine.AskQuestion{
		Player: p.ID,
		Question: engine.Ask(engine.Tf("c.choose", card),
			engine.Choice{
				ID:    "remove",
				Label: engine.Tf("c.exhaustRemoveFromTheGame", p.AlterEgoDef(), card),
				Kind:  engine.ChoiceLabel,
			}.Msgs(removeMsgs...),
			engine.Choice{
				ID: "penalty", Label: penaltyLabel, Kind: engine.ChoiceLabel,
			}.Msgs(penalty...),
		),
	}}
}
