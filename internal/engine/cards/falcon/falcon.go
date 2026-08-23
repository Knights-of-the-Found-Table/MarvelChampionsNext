// Package falcon registers Falcon, his Aerial/encounter-top suite,
// obligation, and Viper nemesis set.
package falcon

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

const eagleSignalBase = -53010

func init() { registerIdentity(); registerSignatures(); registerObligation(); registerNemesis() }

func boostIcons(c engine.Card) int {
	// CardDef exposes ordinary boost icons but not separate star icons.
	if c.Def().Boost != nil {
		return *c.Def().Boost
	}
	return 0
}
func topIcons(g *engine.Game) int {
	if len(g.EncounterDeck) == 0 {
		return 0
	}
	return boostIcons(g.EncounterDeck[0])
}

func registerIdentity() {
	engine.RegisterBehavior("53001", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		p := g.Player(e.EID())
		m, ok := msg.(engine.PlayCard)
		if !ok || p == nil || m.Player != p.ID || !m.Card.Def().HasTrait("aerial") {
			return nil
		}
		n := topIcons(g)
		return []engine.Message{engine.MillEncounter{N: 1}, engine.AddEntityCounter{ID: p.ID, N: eagleSignalBase - n}}
	}})
}

func registerSignatures() {
	engine.RegisterBehavior("53002", &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{Label: "Redwing — return to hand and scout", Type: engine.AbilityAction, HeroOnly: true, Exhaust: true, Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
			a := g.Allies[self]
			p := g.Player(a.Owner)
			n := topIcons(g)
			var ch []engine.Choice
			ch = append(ch, cardutil.EnemyChoices(g, n, p.ID, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: n, Source: a.ID}}
			})...)
			ch = append(ch, cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.ThwartScheme{Scheme: id, N: n, Source: a.ID}}
			})...)
			msgs := []engine.Message{engine.ReturnControlled{Player: p.ID, ID: a.ID}, engine.MillEncounter{N: 1}}
			if len(ch) > 0 {
				msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask("Redwing — damage or thwart", ch...)})
			}
			return msgs
		}}}
	}})
	engine.RegisterBehavior("53003", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		n := 4 + topIcons(g)
		ch := cardutil.EnemyChoices(g, n, p.ID, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.MillEncounter{N: 1}, engine.DamageEntity{Target: id, Damage: n, Source: p.ID}}
		})
		if len(ch) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Bird of Prey — choose an enemy", ch...)}}
	}})
	engine.RegisterBehavior("53004", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		n := 3 + topIcons(g)
		ch := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
			return []engine.Message{engine.MillEncounter{N: 1}, engine.ThwartScheme{Scheme: id, N: n, Source: p.ID}}
		})
		if len(ch) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Bird's-Eye View — choose a scheme", ch...)}}
	}})
	engine.RegisterBehavior("53005", &engine.Behavior{DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
		n := topIcons(g)
		return engine.Defends{Defender: p.ID, Against: against}, []engine.Message{engine.DrawCards{Player: p.ID, N: n}}, true
	}})
	engine.RegisterBehavior("53006", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 5}}
	}, Resource: &engine.ResourceAbility{Icon: "energy", UsesCounters: true}})
	engine.RegisterBehavior("53007", &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{Label: "Soup Kitchen — heal and discount", Type: engine.AbilityAction, AlterEgoOnly: true, Exhaust: true, Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
			s := g.Supports[self]
			p := g.Player(s.Owner)
			return []engine.Message{engine.ExhaustEntity{ID: p.ID}, engine.HealEntity{Target: p.ID, N: p.RecoverStat(g)}, engine.CostDiscountApply{Player: p.ID, Amount: 2}}
		}}}
	}})
	// DamagePrevention can protect Falcon, not another friendly character;
	// use owner-only prevention and force the form change when consumed.
	engine.RegisterBehavior("53008", &engine.Behavior{DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
		g.Delete(u.ID)
		g.Push(engine.ChangeForm{Player: p.ID})
		return n, 0
	}})
	engine.RegisterBehavior("53009", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Aerial Recon — take an encounter card for a recon counter", Type: engine.AbilityAction, HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					return []engine.Message{engine.DealEncounterToPlayer{Player: u.Owner}, engine.AddEntityCounter{ID: u.ID, N: 1}}
				},
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DealEncounterToPlayer)
			u := e.(*engine.Upgrade)
			if !ok || m.Player != u.Owner || u.Counters < 1 {
				return nil
			}
			// The deal message itself cannot be canceled by a reaction; consuming
			// the counter is exact, prevention is the hook limitation.
			u.Counters--
			return nil
		},
	})
	engine.RegisterBehavior("53010", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		u := e.(*engine.Upgrade)
		if u.Exhausted {
			return nil
		}
		n := topIcons(g)
		switch m := msg.(type) {
		case engine.BasicAttack:
			if m.Player == u.Owner {
				return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.MillEncounter{N: 1}, engine.DamageEntity{Target: m.Target, Damage: n, Source: u.Owner}}
			}
		case engine.BasicThwart:
			if m.Player == u.Owner {
				return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.MillEncounter{N: 1}, engine.ThwartScheme{Scheme: m.Target, N: n, Source: u.Owner}}
			}
		}
		return nil
	}})
	// Draw Their Fire's no-exhaust defense flag has no phase aura hook.
	engine.RegisterBehavior("53011", &engine.Behavior{})
	engine.RegisterBehavior("53012", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.AddEntityCounter)
		u := e.(*engine.Upgrade)
		if !ok || m.ID != u.Owner || m.N > eagleSignalBase {
			return nil
		}
		icons := eagleSignalBase - m.N
		if icons <= 0 {
			return nil
		}
		var msgs []engine.Message
		for _, id := range g.Player(u.Owner).Allies {
			if len(msgs) >= icons {
				break
			}
			msgs = append(msgs, engine.ReadyEntity{ID: id})
		}
		msgs = append(msgs, engine.DiscardControlled{Player: u.Owner, ID: u.ID})
		return msgs
	}})
	engine.RegisterBehavior("53013", &engine.Behavior{IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{DEF: 1} }, DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
		if u.Exhausted {
			return 0, 0
		}
		u.Exhausted = true
		return min(1, n), 1
	}})
}

func registerObligation() {
	engine.RegisterBehavior("53029", &engine.Behavior{ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
		// Persistent emergency counters have no obligation play area. Model the
		// three payments as discarding up to three cards, then remove it.
		n := min(3, len(p.Hand))
		return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: append(engine.CardList(nil), p.Hand[:n]...)}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}}
	}})
}
func registerNemesis() {
	engine.RegisterBehavior("53030", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.MinionActivates)
		if !ok || m.MinionID != e.EID() {
			return nil
		}
		return []engine.Message{engine.MillEncounter{N: 5}}
	}})
	// Serpent Solutions' dealt-card interception cannot identify which card a
	// bulk MillEncounter discarded, so the side scheme is registered as approximate.
	engine.RegisterBehavior("53031", &engine.Behavior{})
	engine.RegisterBehavior("53032", &engine.Behavior{Boost: func(g *engine.Game, card engine.Card) []engine.Message {
		if v := g.Villains[g.ActiveVillain]; v != nil {
			return []engine.Message{engine.DealBoost{Enemy: v.ID}}
		}
		return nil
	}})
	engine.RegisterBehavior("53033", &engine.Behavior{ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		g.Delete(t.ID)
		for i := len(g.EncounterDiscard) - 1; i >= 0; i-- {
			c := g.EncounterDiscard[i]
			if c.Def().HasTrait("serpent society") {
				g.EncounterDiscard.Remove(c.ID)
				g.EncounterDeck = append(g.EncounterDeck, c)
			}
		}
		g.ShuffleEncounterDeck()
		return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
	}})
}
