// Package jessicadrew registers the Spider-Woman hero (04031) from the
// Rise of Red Skull box: the Spider-Woman / Jessica Drew identity with
// its aspect-triggered Superhuman Agility, the signature cards, the
// Uncertain Loyalties obligation and The Viper nemesis set.
//
// Note: the package is named jessicadrew because cards/spiderwoman
// already hosts the Scarlet Witch hero pack (15001).
package jessicadrew

import (
	"fmt"
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerSpiderWoman()
	registerSWSIgnatures()
	registerSWObligation()
	registerSWNemesis()
}

// registerSpiderWoman installs the Spider-Woman / Jessica Drew identity
// (04031a/b).
//
// Double Agent (choose two aspects in deck-building) is a deck-building
// rule; the engine has no deck-construction validation, so it is not
// modeled here.
func registerSpiderWoman() {
	engine.RegisterBehavior("04031", &engine.Behavior{
		// Superhuman Agility — Interrupt: when you play an aspect card,
		// Spider-Woman gets +1 THW/+1 ATK/+1 DEF until the end of the
		// round (limit once per round per aspect).
		// (Approximation: ApplyStatBonus is until-end-of-phase, slightly
		// shorter than the printed end-of-round; the per-aspect limit is
		// tracked per round. Defense events played via the defense prompt
		// are covered through PlayDefenseEvent.)
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil || !p.IsHero() {
				return nil
			}
			var defAspect string
			aspectOf := func(c engine.Card) string {
				def := c.Def()
				// Hero-set cards carry a data-layer aspect faction but are
				// not "aspect cards" for Superhuman Agility; only aspect-pool
				// cards (no card set) count.
				if def.CardSet != "" {
					return ""
				}
				return def.Aspect
			}
			switch m := msg.(type) {
			case engine.PlayCard:
				if m.Player == p.ID {
					defAspect = aspectOf(m.Card)
				}
			case engine.PlayDefenseEvent:
				if m.Player == p.ID {
					defAspect = aspectOf(m.Card)
				}
			}
			if defAspect == "" {
				return nil
			}
			key := "sw-agility-" + defAspect
			if g.UsedThisRound == nil {
				g.UsedThisRound = map[string]bool{}
			}
			if g.UsedThisRound[key] {
				return nil
			}
			g.UsedThisRound[key] = true
			g.TLogf("c.superhumanAgilityCardPlayedSpiderWomanGets111", defAspect)
			return []engine.Message{engine.ApplyStatBonus{Target: p.ID, ATK: 1, THW: 1, DEF: 1}}
		},
		// The Viper rider: while engaged with you, hand size -1.
		HandSizeBonus: func(g *engine.Game, p *engine.Player) int {
			for _, mn := range g.Minions {
				if mn != nil && mn.EngagedWith == p.ID && strings.HasPrefix(mn.Code, "04054") {
					return -1
				}
			}
			return 0
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				// Jessica Drew — Action: look at the top card of any deck
				// (limit once per round). (Approximation: the peeked card
				// is written to the game log; there is no private
				// information channel.)
				Label:        engine.Tf("c.jessicaDrewLookAtTheTopCardOfAnyDeck"),
				Type:         engine.AbilityAction,
				AlterEgoOnly: true,
				OncePerRound: true,
				Execute:      lookAtTopCard,
			}}
		},
	})
}

// lookAtTopCard runs Jessica Drew's alter-ego peek action.
func lookAtTopCard(g *engine.Game, self engine.EntityID) []engine.Message {
	p := g.Player(self)
	if p == nil {
		return nil
	}
	var choices []engine.Choice
	addDeck := func(label string, deck engine.CardList) {
		if len(deck) == 0 {
			return
		}
		top := deck[0]
		choices = append(choices, engine.Choice{
			Label: engine.S(label), Kind: engine.ChoiceLabel,
		}.Msgs(engine.AskQuestion{
			Player: p.ID,
			Question: engine.Ask(engine.Tf("c.topCard", top),
				engine.Choice{ID: "ok", Label: engine.Tf("c.ok"), Kind: engine.ChoicePass}),
		}))
	}
	for _, pl := range g.Players {
		addDeck(fmt.Sprintf("%s's deck", pl.Name), pl.Deck)
	}
	addDeck("Encounter deck", g.EncounterDeck)
	if len(choices) == 0 {
		return nil
	}
	return []engine.Message{engine.AskQuestion{
		Player:   p.ID,
		Question: engine.Ask(engine.Tf("c.jessicaDrewLookAtTheTopCardOfWhichDeck"), choices...),
	}}
}

func registerSWSIgnatures() {
	// 04032 Captain Marvel (ally): Response — after she uses a basic
	// power, draw 1 card.
	engine.RegisterBehavior("04032", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				if m.Ally == a.ID {
					return []engine.Message{engine.DrawCards{Player: a.Owner, N: 1}}
				}
			case engine.AllyThwartWindow:
				if m.Ally == a.ID {
					return []engine.Message{engine.DrawCards{Player: a.Owner, N: 1}}
				}
			case engine.WindowDefended:
				if m.Defender == a.ID {
					return []engine.Message{engine.DrawCards{Player: a.Owner, N: 1}}
				}
			}
			return nil
		},
	})

	// 04033 Finesse: Hero Resource — exhaust → wild for an aspect card.
	// (Approximation: the aspect-only restriction is not representable in
	// the payment hook.)
	engine.RegisterBehavior("04033", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true},
	})

	// 04034 Jessica Drew's Apartment: Alter-Ego Action — exhaust → search
	// the top 5 cards of your deck for an aspect card, add it to your
	// hand, shuffle. (Approximation: the unchosen cards stay on top and
	// the whole deck is shuffled after the pick.)
	engine.RegisterBehavior("04034", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:        engine.Tf("c.jessicaDrewSApartmentSearchTheTop5ForAnAspectCard"),
				Type:         engine.AbilityAction,
				AlterEgoOnly: true,
				Exhaust:      true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(e.EOwner())
					if s == nil || p == nil {
						return nil
					}
					top := p.Deck
					if len(top) > 5 {
						top = top[:5]
					}
					choices := []engine.Choice{cardutil.Skip()}
					for _, c := range top {
						def := c.Def()
						if def.Aspect == "" || def.CardSet != "" {
							continue
						}
						choices = append(choices, engine.Choice{
							Label: engine.S(c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(
							engine.TakeDeckCard{Player: p.ID, CardID: c.ID},
							engine.ShufflePlayerDeck{Player: p.ID},
						))
					}
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask(engine.Tf("c.jessicaDrewSApartmentAddAnAspectCardToYourHand"), choices...),
					}}
				},
			}}
		},
	})

	// 04035 Venom Blast: Hero Action (attack) — deal 5 damage to an enemy.
	engine.RegisterBehavior("04035", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.venomBlastDeal5DamageToAnEnemy"),
			func(g *engine.Game, e engine.Entity) (int, []engine.Message) { return 5, nil }),
	})

	// 04036 Pheromones: Hero Action — stun and confuse an enemy.
	engine.RegisterBehavior("04036", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.EnemyChoices(g, 0, pid, func(id engine.EntityID) []engine.Message {
				return []engine.Message{
					engine.StunEntity{Target: id},
					engine.ConfuseEntity{Target: id},
				}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.pheromonesStunAndConfuseAnEnemy"), choices...),
			}}
		},
	})

	// 04037 Contaminant Immunity: Hero Action — heal 3 damage from
	// Spider-Woman and give her a tough status card.
	engine.RegisterBehavior("04037", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			return []engine.Message{
				engine.HealEntity{Target: pid, N: 3},
				engine.ToughEntity{Target: pid},
			}
		},
	})

	// 04038 Inconspicuous: Hero Action (thwart) — remove a total of 3
	// threat from among schemes. (Approximation: no split-threat chooser;
	// all 3 threat comes from one scheme, the Torrential Rain precedent.)
	engine.RegisterBehavior("04038", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme(engine.Tf("c.chooseAScheme", "Inconspicuous"), func(g *engine.Game, e engine.Entity) int { return 3 }),
	})

	// 04039 Self-Propelled Glide: Hero Action — ready Spider-Woman; she
	// gains aerial until end of round. (Approximation: granted traits do
	// not expire, the Fury's Flying Car precedent.)
	engine.RegisterBehavior("04039", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			return []engine.Message{
				engine.ReadyEntity{ID: pid},
				engine.GrantTrait{Target: pid, Trait: "aerial"},
			}
		},
	})

	// 04040 Spider-Girl (ally): Response — after you play her from your
	// hand, stun and confuse a minion.
	engine.RegisterBehavior("04040", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			choices := cardutil.MinionChoices(g, func(id engine.EntityID) []engine.Message {
				return []engine.Message{
					engine.StunEntity{Target: id},
					engine.ConfuseEntity{Target: id},
				}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   e.EOwner(),
				Question: engine.Ask(engine.Tf("c.spiderGirlStunAndConfuseAMinion"), choices...),
			}}
		},
	})
}

// registerSWObligation installs Uncertain Loyalties (04053): exhaust
// Jessica Drew to remove it, or place 3 threat on the main scheme.
func registerSWObligation() {
	engine.RegisterBehavior("04053", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var penalty []engine.Message
			if g.MainScheme != nil {
				penalty = append(penalty, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 3, Source: p.ID})
			}
			return cardutil.ExhaustOrPenalty(g, p, card, engine.Tf("c.place3ThreatOnTheMainScheme"), penalty...)
		},
	})
}

// registerSWNemesis installs the Spider-Woman nemesis set (The Viper,
// The Viper's Ambition, Hydra Regular, Hail Hydra!).
func registerSWNemesis() {
	// 04054 The Viper: the hand-size rider lives on the identity's
	// HandSizeBonus hook (minion behaviors have no hand-size channel).
	engine.RegisterBehavior("04054", &engine.Behavior{})

	// 04055 The Viper's Ambition: When Revealed — place an additional
	// 1[per_hero] threat here.
	engine.RegisterBehavior("04055", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: len(g.Players), Source: e.EID()}}
		},
	})

	// 04056 Hydra Regular: Incite 1.
	engine.RegisterBehavior("04056", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()}}
		},
	})

	// 04057 Hail Hydra!: each Hydra minion engaged with you attacks you;
	// if none attacked, search the encounter deck/discard for a Hydra
	// minion and put it into play engaged with you. (Approximation: only
	// the revealing player is considered; other players' search branch is
	// not modeled.)
	engine.RegisterBehavior("04057", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn != nil && mn.EngagedWith == p.ID && mn.EDef().HasTrait("hydra") {
					msgs = append(msgs, engine.MinionActivates{MinionID: id, Player: p.ID})
				}
			}
			if len(msgs) > 0 {
				g.TLogf("c.hailHydraHydraMinionsAttack", p.Name)
				return msgs
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Def().Type == "minion" && c.Def().HasTrait("hydra") {
						zone.Remove(c.ID)
						def := c.Def()
						mn := &engine.Minion{
							ID:        g.NextEntityID(engine.KindMinion),
							Code:      c.Code,
							MaxHP:     derefInt(def.HP, 1),
							AttackVal: derefInt(def.Attack, 0),
							SchemeVal: derefInt(def.Scheme, 0),
							Tough:     def.HasKeyword("Toughness"),
							Guard:     def.HasKeyword("Guard"),
						}
						g.Minions[mn.ID] = mn
						mn.EngagedWith = p.ID
						g.TLogf("c.hailHydraEngages", def.Name, p.Name)
						return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID}}
					}
				}
			}
			return nil
		},
	})
}

func derefInt(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
