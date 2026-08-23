package aos

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

const shieldTrait = "s.h.i.e.l.d."

// hasShieldTrait works around the legacy trait parser splitting dotted
// acronyms into ["s", "h", "i", "e", "l", "d"].
func hasShieldTrait(def *data.CardDef) bool {
	if def == nil {
		return false
	}
	if def.HasTrait(shieldTrait) {
		return true
	}
	want := []string{"s", "h", "i", "e", "l", "d"}
	for i := 0; i+len(want) <= len(def.Traits); i++ {
		match := true
		for j := range want {
			if def.Traits[i+j] != want[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func shieldSupports(g *engine.Game, p *engine.Player) []*engine.Support {
	if p == nil {
		return nil
	}
	var out []*engine.Support
	for _, id := range p.Supports {
		if s := g.Supports[id]; s != nil && hasShieldTrait(s.EDef()) {
			out = append(out, s)
		}
	}
	return out
}

func supportCounterChoices(g *engine.Game, p *engine.Player, prompt string, n int) []engine.Message {
	var choices []engine.Choice
	for _, s := range shieldSupports(g, p) {
		choices = append(choices, engine.Choice{
			Label: s.EDef().Name,
			Kind:  engine.ChoiceTarget, SourceID: s.ID, CardCode: s.Code,
		}.Msgs(engine.AddEntityCounter{ID: s.ID, N: n}))
	}
	if len(choices) == 0 {
		return nil
	}
	return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(prompt, choices...)}}
}

func allEnemyDamage(g *engine.Game, source engine.EntityID, n int) []engine.Message {
	var msgs []engine.Message
	for _, id := range cardutil.SortedEnemyIDs(g) {
		msgs = append(msgs, engine.DamageEntity{Target: id, Damage: n, Source: source})
	}
	return msgs
}

func addTraitOnce(a *engine.Ally, trait string) {
	for _, existing := range a.ExtraTraits {
		if existing == trait {
			return
		}
	}
	a.ExtraTraits = append(a.ExtraTraits, trait)
}

func controlledMinion(g *engine.Game, p *engine.Player) *engine.Minion {
	if p == nil || len(p.Deck) == 0 {
		return nil
	}
	card := p.Deck[0]
	p.Deck = p.Deck[1:]
	// The engine has no generic facedown-minion entity. Controller is used
	// as the visual proxy while Source preserves the actual player card.
	mn := &engine.Minion{
		ID: g.NextEntityID("minion"), Code: "50030", MaxHP: 1,
		AttackVal: 1, SchemeVal: 1, EngagedWith: p.ID,
		Source: &card, BlankText: true,
	}
	g.Minions[mn.ID] = mn
	g.Logf("%s is put into play facedown as a Controlled minion", card.Def().Name)
	return mn
}

func removeCounterFromFirstSupport(g *engine.Game, p *engine.Player) bool {
	if p == nil {
		return false
	}
	for _, id := range p.Supports {
		if s := g.Supports[id]; s != nil && s.Counters > 0 {
			s.Counters--
			g.Logf("%s loses 1 all-purpose counter", s.EDef().Name)
			return true
		}
	}
	return false
}

func controlledInnocentsInPlay(g *engine.Game) bool {
	for _, env := range g.Environments {
		if env != nil && env.Code == "50032" {
			return true
		}
	}
	return false
}
