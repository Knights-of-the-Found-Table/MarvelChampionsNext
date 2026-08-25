// complete.go implements the remaining Cyclops pack cards (33011–33026,
// 33032–33035): the shared X-Men pool expansion (allies, locations,
// tactics, training upgrades and basic resources).
package cyclops

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerRemainingCyclops()
}

// attachToMinion builds an OnPlay that attaches the upgrade to a minion
// without a copy of the same card (max 1 per minion).
func attachToMinion(code string) func(g *engine.Game, e engine.Entity) []engine.Message {
	return func(g *engine.Game, e engine.Entity) []engine.Message {
		u := g.Upgrades[e.EID()]
		if u == nil {
			return nil
		}
		var choices []engine.Choice
		for _, mn := range g.Minions {
			if mn == nil {
				continue
			}
			attached := false
			for _, x := range g.Upgrades {
				if x != nil && x.Code == code && x.AttachTo == mn.ID {
					attached = true
				}
			}
			if attached {
				continue
			}
			choices = append(choices, engine.Choice{
				Label: engine.S("Attach to " + mn.EDef().Name), Kind: engine.ChoiceTarget, SourceID: mn.ID, CardCode: mn.Code,
			}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: mn.ID}))
		}
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{
			Player:   e.EOwner(),
			Question: engine.Ask(engine.S(u.EDef().Name+" — attach to a minion"), choices...),
		}}
	}
}

// trainingAttached reports whether the ally already has a Training
// upgrade.
func trainingAttached(g *engine.Game, ally engine.EntityID) bool {
	for _, u := range g.Upgrades {
		if u != nil && u.AttachTo == ally && u.EDef().HasTrait("training") {
			return true
		}
	}
	return false
}

// attachToXMenAlly builds an OnPlay that attaches the upgrade to one of
// the owner's X-Men allies (max 1 Training per ally).
func attachToXMenAlly(bonus func() (thw, atk, hp int)) func(g *engine.Game, e engine.Entity) []engine.Message {
	return func(g *engine.Game, e engine.Entity) []engine.Message {
		u := g.Upgrades[e.EID()]
		p := g.Player(e.EOwner())
		if u == nil || p == nil {
			return nil
		}
		thw, atk, hp := bonus()
		var choices []engine.Choice
		for _, id := range p.Allies {
			a := g.Allies[id]
			if a == nil || !a.EDef().HasTrait("x-men") || trainingAttached(g, id) {
				continue
			}
			choices = append(choices, engine.Choice{
				Label: engine.S("Attach to " + a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
			}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id, THW: thw, ATK: atk, MaxHP: hp}))
		}
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{
			Player:   p.ID,
			Question: engine.Ask(engine.S(u.EDef().Name+" — attach to an X-Men ally"), choices...),
		}}
	}
}

func registerRemainingCyclops() {
	// 33011 Beast: on enter — take a resource card from deck or discard.
	engine.RegisterBehavior("33011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, c := range p.Deck {
				if c.Def().Type == "resource" {
					choices = append(choices, engine.Choice{
						Label: engine.S("Take " + c.Def().Name + " (deck)"), Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID}))
					break
				}
			}
			for _, c := range p.Discard {
				if c.Def().Type == "resource" {
					choices = append(choices, engine.Choice{
						Label: engine.S("Take " + c.Def().Name + " (discard)"), Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
					break
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.beastTakeAResourceCard"), choices...),
			}}
		},
	})

	// 33012 Dust: when she attacks a minion, she attacks each minion in
	// play (approximation: splash damage to the other minions; the +1
	// consequential rider is not modeled).
	engine.RegisterBehavior("33012", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			a := g.Allies[e.EID()]
			if !ok || m.Ally != e.EID() || a == nil {
				return nil
			}
			if mn := g.Minions[m.Target]; mn == nil {
				return nil
			}
			dmg := a.AttackVal + a.BonusATK + a.PermATK
			var msgs []engine.Message
			for _, mn := range g.Minions {
				if mn != nil && mn.ID != m.Target {
					msgs = append(msgs, engine.DamageEntity{Target: mn.ID, Damage: dmg, Source: a.Owner})
				}
			}
			if len(msgs) > 0 {
				g.TLogf("c.dustSSandstormHitsEveryMinion")
			}
			return msgs
		},
	})

	// 33013 Rockslide: Retaliate 1 from data.
	engine.RegisterBehavior("33013", &engine.Behavior{})

	// 33014 Blindfold: on enter — look at the top 5 encounter cards and
	// discard one (approximation: the top card is discarded).
	engine.RegisterBehavior("33014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if len(g.EncounterDeck) == 0 {
				return nil
			}
			top := g.EncounterDeck[0]
			g.EncounterDeck = g.EncounterDeck[1:]
			g.EncounterDiscard = append(g.EncounterDiscard, top)
			g.TLogf("c.blindfoldDiscardsFromTheTopOfTheEncounterDeck", top)
			return nil
		},
	})

	// 33015 Danger Room Training: +1 THW / +1 ATK / +1 HP to an X-Men
	// ally.
	engine.RegisterBehavior("33015", &engine.Behavior{
		OnPlay: attachToXMenAlly(func() (int, int, int) { return 1, 1, 1 }),
	})

	// 33016 Coordinated Attack: attach to a minion (the -1 consequential
	// rider is not modeled).
	engine.RegisterBehavior("33016", &engine.Behavior{
		OnPlay: attachToMinion("33016"),
	})

	// 33017 Teamwork: alias the Thor printing (06032) approximation.
	if b := engine.LookupBehavior("06032"); b != nil {
		engine.RegisterBehavior("33017", b)
	}

	// 33018 Effective Leadership: the rider resolves in handlePlayCard.
	engine.RegisterBehavior("33018", &engine.Behavior{})

	// 33019 Angel: costs 1 less for MUTANT/X-MEN identities.
	engine.RegisterBehavior("33019", &engine.Behavior{
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if g.EntityHasTrait(p.ID, "x-men") || g.EntityHasTrait(p.ID, "mutant") {
				return 1
			}
			return 0
		},
	})

	// 33020 Utopia: the +1 ally limit is moot without a limit counter;
	// Response — after an X-Men ally enters play, exhaust → ready an
	// X-Men character.
	engine.RegisterBehavior("33020", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyEnteredPlay)
			s := g.Supports[e.EID()]
			if !ok || s == nil || s.Exhausted {
				return nil
			}
			a := g.Allies[m.Ally]
			if a == nil || !a.EDef().HasTrait("x-men") {
				return nil
			}
			var choices []engine.Choice
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			if g.EntityHasTrait(p.ID, "x-men") {
				choices = append(choices, engine.Choice{
					Label: engine.S("Ready " + p.Name), Kind: engine.ChoiceTarget, SourceID: p.ID,
				}.Msgs(engine.ExhaustEntity{ID: s.ID}, engine.ReadyEntity{ID: p.ID}))
			}
			for _, id := range p.Allies {
				if x := g.Allies[id]; x != nil && x.EDef().HasTrait("x-men") {
					choices = append(choices, engine.Choice{
						Label: engine.S("Ready " + x.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.ExhaustEntity{ID: s.ID}, engine.ReadyEntity{ID: id}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.utopiaReadyAnXMenCharacter"), choices...),
			}}
		},
	})

	// 33021 Danger Room: Alter-Ego Response — after an X-Men ally enters
	// play, exhaust → attach a Training upgrade from deck/discard to it.
	engine.RegisterBehavior("33021", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyEnteredPlay)
			s := g.Supports[e.EID()]
			if !ok || s == nil || s.Exhausted {
				return nil
			}
			a := g.Allies[m.Ally]
			if a == nil || !a.EDef().HasTrait("x-men") {
				return nil
			}
			// Find a Training upgrade in the owner's zones (approximation:
			// auto-attach the first found).
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			var found *engine.Card
			var from string
			for i := range p.Deck {
				if p.Deck[i].Def().HasTrait("training") {
					c := p.Deck[i]
					found = &c
					from = "deck"
					break
				}
			}
			if found == nil {
				for i := range p.Discard {
					if p.Discard[i].Def().HasTrait("training") {
						c := p.Discard[i]
						found = &c
						from = "discard"
						break
					}
				}
			}
			if found == nil {
				return nil
			}
			if from == "deck" {
				p.Deck.Remove(found.ID)
			} else {
				p.Discard.Remove(found.ID)
			}
			u := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: found.Code, Owner: p.ID}
			g.Upgrades[u.ID] = u
			p.Upgrades = append(p.Upgrades, u.ID)
			s.Exhausted = true
			g.TLogf("c.dangerRoomAttachesTo", u, a)
			return []engine.Message{engine.AttachUpgrade{ID: u.ID, Target: a.ID, MaxHP: 1, ATK: 1, THW: 1}}
		},
	})

	// 33022 Game Time: ready an ally with a Training upgrade and heal 1.
	engine.RegisterBehavior("33022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil || !trainingAttached(g, id) {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: engine.S("Ready " + a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
				}.Msgs(engine.ReadyEntity{ID: id}, engine.HealEntity{Target: id, N: 1}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.gameTimeReadyAnAllyWithATrainingUpgrade"), choices...),
			}}
		},
	})

	// 33023 Psychic Rapport: ready Cyclops and Phoenix; return a Cyclops
	// card from discard to hand (the Phoenix Force counter option is
	// approximated as a draw).
	engine.RegisterBehavior("33023", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			base := []engine.Message{engine.ReadyEntity{ID: p.ID}}
			var choices []engine.Choice
			for _, c := range p.Discard {
				if c.Def().CardSet == "cyclops" {
					choices = append(choices, engine.Choice{
						Label: engine.S("Return " + c.Def().Name + " to hand"), Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
				}
			}
			choices = append(choices, engine.Choice{
				ID: "draw", Label: engine.Tf("c.draw1CardPhoenixForceCountersApproximated"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.DrawCards{Player: p.ID, N: 1}))
			return append(base, engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.psychicRapportChoose2"), choices...),
			})
		},
	})

	// 33024–33026 basic resources.
	for _, code := range []string{"33024", "33025", "33026"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 33032 Marked: attach to a minion; the overkill rider is cosmetic.
	engine.RegisterBehavior("33032", &engine.Behavior{
		OnPlay: attachToMinion("33032"),
	})

	// 33033 Befuddle: attach to a minion; the THW-for-ATK swap has no
	// attack window.
	engine.RegisterBehavior("33033", &engine.Behavior{
		OnPlay: attachToMinion("33033"),
	})

	// 33034 Pinned Down: attach to a minion; it gets −2 ATK (applied as a
	// permanent attack reduction).
	engine.RegisterBehavior("33034", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			var choices []engine.Choice
			for _, mn := range g.Minions {
				if mn == nil {
					continue
				}
				attached := false
				for _, uid := range g.Upgrades {
					if uid != nil && uid.Code == "33034" && uid.AttachTo == mn.ID {
						attached = true
					}
				}
				if attached {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: engine.S("Pin down " + mn.EDef().Name + " (−2 ATK)"), Kind: engine.ChoiceTarget, SourceID: mn.ID,
				}.Msgs(
					engine.AttachUpgrade{ID: u.ID, Target: mn.ID},
					engine.BoostEnemyAttack{Enemy: mn.ID, N: -2},
				))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   e.EOwner(),
				Question: engine.Ask(engine.Tf("c.pinnedDownAttachToAMinion"), choices...),
			}}
		},
	})

	// 33035 Honorary X-Men: attach to a friendly character (+1 HP, gains
	// X-Men).
	engine.RegisterBehavior("33035", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "x-men")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			if u == nil || p == nil {
				return nil
			}
			var choices []engine.Choice
			hasHonorary := func(id engine.EntityID) bool {
				for _, x := range g.Upgrades {
					if x != nil && x.Code == "33035" && x.AttachTo == id {
						return true
					}
				}
				return false
			}
			if !hasHonorary(p.ID) {
				choices = append(choices, engine.Choice{
					Label: engine.S("Attach to " + p.Name), Kind: engine.ChoiceTarget, SourceID: p.ID,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: p.ID, MaxHP: 1, GrantTrait: "x-men"}))
			}
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil || hasHonorary(id) {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: engine.S("Attach to " + a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id, MaxHP: 1, GrantTrait: "x-men"}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.honoraryXMenAttachToAFriendlyCharacter"), choices...),
			}}
		},
	})
}
