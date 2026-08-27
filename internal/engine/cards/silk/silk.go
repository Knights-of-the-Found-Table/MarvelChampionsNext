// Package silk registers Silk, her tucked encounter-card suite, obligation,
// and Morlun nemesis set. Tucked cards use the identity's side-deck slot.
package silk

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() { registerIdentity(); registerSignatures(); registerObligation(); registerNemesis() }

func tuck(g *engine.Game, p *engine.Player, c engine.Card) {
	if p == nil {
		return
	}
	if c.ID == "" {
		c.ID = g.NextCardID()
	}
	c.Owner = p.ID
	p.SenseDeck = append(p.SenseDeck, c)
	for len(p.SenseDeck) > 4 {
		p.SenseDeck = p.SenseDeck[1:]
	}
}

func sameSet(a, b engine.Card) bool {
	return a.Def().CardSet != "" && a.Def().CardSet == b.Def().CardSet
}
func matchingTucked(p *engine.Player, code string) (int, int) {
	probe := engine.Card{Code: code}
	for i, c := range p.SenseDeck {
		if sameSet(c, probe) {
			return i, 1
		}
	}
	return -1, 0
}
func discardTucked(p *engine.Player, i int) engine.Card {
	c := p.SenseDeck[i]
	p.SenseDeck = append(p.SenseDeck[:i], p.SenseDeck[i+1:]...)
	return c
}
func countMatching(p *engine.Player, code string) int {
	n := 0
	probe := engine.Card{Code: code}
	for _, c := range p.SenseDeck {
		if sameSet(c, probe) {
			n++
		}
	}
	return n
}

func registerIdentity() {
	engine.RegisterBehavior("52001", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		p := g.Player(e.EID())
		if p == nil {
			return nil
		}
		switch m := msg.(type) {
		case engine.MinionDefeated:
			if mn := g.Minions[m.MinionID]; mn != nil {
				tuck(g, p, engine.Card{Code: mn.Code})
			}
		case engine.SchemeDefeated:
			if s := g.SideSchemes[m.Scheme]; s != nil {
				tuck(g, p, engine.Card{Code: s.Code})
			}
		case engine.RevealEncounterCard:
			if m.Player == p.ID && m.Card.Def().Type == "treachery" {
				// No post-treachery window exists. Tuck a tracked copy at reveal;
				// the resolving physical card still follows generic discard flow.
				tuck(g, p, engine.Card{Code: m.Card.Code})
			}
		}
		return nil
	}})
}

func registerSignatures() {
	engine.RegisterBehavior("52002", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		if len(g.EncounterDeck) == 0 {
			return nil
		}
		// A target choice cannot feed its encounter set into a later mill loop.
		// Use the active villain's set, otherwise the main scheme's set.
		set := ""
		if v := g.Villains[g.ActiveVillain]; v != nil {
			set = v.EDef().CardSet
		} else if g.MainScheme != nil {
			set = g.MainScheme.EDef().CardSet
		}
		n := 0
		var found engine.Card
		for _, c := range g.EncounterDeck {
			n++
			if c.Def().CardSet == set {
				found = c
				break
			}
		}
		if found.Code == "" {
			n = len(g.EncounterDeck)
			found = g.EncounterDeck[n-1]
		}
		g.EncounterDeck = g.EncounterDeck[n:]
		tuck(g, p, found)
		return nil
	}})
	engine.RegisterBehavior("52003", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		var choices []engine.Choice
		for _, id := range cardutil.SortedEnemyIDs(g) {
			enemy := g.Entity(id)
			n := 7
			if i, ok := matchingTucked(p, enemy.ECode()); ok > 0 {
				discardTucked(p, i)
				n = 9
			}
			choices = append(choices, engine.Choice{Label: engine.Tf("m.cardName", enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
				Msgs(engine.DamageEntity{Target: id, Damage: n, Source: p.ID}))
		}
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.swingingSilkKickChooseAnEnemy"), choices...)}}
	}})
	engine.RegisterBehavior("52004", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			n := 2
			if i, ok := matchingTucked(p, g.Entity(id).ECode()); ok > 0 {
				discardTucked(p, i)
				n = 5
			}
			return []engine.Message{engine.ThwartScheme{Scheme: id, N: n, Source: p.ID}}
		})
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.wallcrawlChooseAScheme"), choices...)}}
	}})
	engine.RegisterBehavior("52005", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.SchemeDefeated)
		if !ok || m.Scheme != e.EID() {
			return nil
		}
		p := g.Player(e.EOwner())
		n := min(2, len(g.EncounterDeck))
		var choices []engine.Choice
		for _, c := range g.EncounterDeck[:n] {
			choices = append(choices, engine.Choice{Label: engine.Tf("m.cardName", c), Kind: engine.ChoiceCard, CardCode: c.Code}.
				Msgs(engine.DiscardEncounterCard{Card: c}))
		}
		if len(choices) == 0 {
			return nil
		}
		// DiscardEncounterCard is the movement approximation; the identity's
		// tuck store cannot be addressed from a serialized choice.
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.getTheScoopTuckOneOfTheTopCards"), choices...)}}
	}})
	engine.RegisterBehavior("52006", &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{
			Label: engine.Tf("c.albertMoonTuckEncounterCardOrHeal"), Type: engine.AbilityAction, AlterEgoOnly: true, Exhaust: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				s := g.Supports[self]
				p := g.Player(s.Owner)
				var choices []engine.Choice
				if len(g.EncounterDeck) > 0 {
					c := g.EncounterDeck[0]
					choices = append(choices, engine.Choice{ID: "tuck", Label: engine.Tf("c.tuckName", c), Kind: engine.ChoiceCard, CardCode: c.Code}.
						Msgs(engine.DiscardEncounterCard{Card: c}))
				}
				choices = append(choices, engine.Choice{ID: "heal", Label: engine.Tf("c.healForTuckedCards"), Kind: engine.ChoiceLabel}.
					Msgs(engine.HealEntity{Target: p.ID, N: len(p.SenseDeck)}))
				return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.albertMoonChoose"), choices...)}}
			},
		}}
	}})
	engine.RegisterBehavior("52007", &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{
			Label: engine.Tf("c.jJonahJamesonRemove2Threat"), Type: engine.AbilityAction, Exhaust: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				s := g.Supports[self]
				p := g.Player(s.Owner)
				choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
					return []engine.Message{engine.ThwartScheme{Scheme: id, N: 2, Source: s.ID}}
				})
				if len(choices) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.jJonahJamesonChooseASideScheme"), choices...)}}
			},
		}}
	}})
	// Eidetic Memory's reveal replacement is not representable by the immutable RevealEncounterCard message.
	engine.RegisterBehavior("52008", &engine.Behavior{})
	engine.RegisterBehavior("52009", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1} },
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if len(p.SenseDeck) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.organicWebbingDiscardTuckedCardAndReadySilk"), Type: engine.AbilityAction, HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.Owner)
					discardTucked(p, 0)
					return []engine.Message{engine.ReadyEntity{ID: p.ID}, engine.GrantTrait{Target: p.ID, Trait: "aerial"}}
				},
			}}
		},
	})
	engine.RegisterBehavior("52010", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.BasicThwart)
		u := e.(*engine.Upgrade)
		if !ok || m.Player != u.Owner || u.Exhausted {
			return nil
		}
		n := countMatching(g.Player(u.Owner), g.Entity(m.Target).ECode())
		if n == 0 {
			return nil
		}
		return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.ThwartScheme{Scheme: m.Target, N: n, Source: u.Owner}}
	}})
	engine.RegisterBehavior("52011", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.BasicAttack)
		u := e.(*engine.Upgrade)
		if !ok || m.Player != u.Owner || u.Exhausted {
			return nil
		}
		n := countMatching(g.Player(u.Owner), g.Entity(m.Target).ECode())
		if n == 0 {
			return nil
		}
		return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.DamageEntity{Target: m.Target, Damage: n, Source: u.Owner}}
	}})
	// Defense timing lacks the attacking enemy on Defends; tuck the top encounter-discard card after any defense.
	engine.RegisterBehavior("52012", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.Defends)
		u := e.(*engine.Upgrade)
		if !ok || m.Defender != u.Owner || u.Exhausted || len(g.EncounterDiscard) == 0 {
			return nil
		}
		tuck(g, g.Player(u.Owner), g.EncounterDiscard[len(g.EncounterDiscard)-1])
		return []engine.Message{engine.ExhaustEntity{ID: u.ID}}
	}})
}

func registerObligation() {
	engine.RegisterBehavior("52028", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		// Obligations cannot host tucked cards. Resolve by discarding up to two identity-tucked cards.
		wasFull := len(p.SenseDeck) >= 4
		for i := 0; i < min(2, len(p.SenseDeck)); i++ {
			discardTucked(p, 0)
		}
		return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card, Remove: wasFull}}
	}})
}

func registerNemesis() {
	engine.RegisterBehavior("52029", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.MinionActivates)
		mn := g.Minions[e.EID()]
		if !ok || mn == nil || m.MinionID != mn.ID {
			return nil
		}
		n := 0
		for _, p := range g.Players {
			n += len(p.SenseDeck)
		}
		// Recomputed activation bonuses cannot be scoped, so this persistent update is approximate.
		mn.AttackVal += n
		mn.SchemeVal += n
		return nil
	}})
	engine.RegisterBehavior("52030", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		n := 0
		for _, p := range g.Players {
			n += len(p.SenseDeck)
		}
		if n == 0 {
			return nil
		}
		return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: n, Source: e.EID()}}
	}})
	engine.RegisterBehavior("52031", &engine.Behavior{ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		if len(p.SenseDeck) >= 4 {
			discardTucked(p, 0)
		}
		tuck(g, p, engine.Card{Code: t.Code})
		g.Delete(t.ID)
		return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
	}})
}
