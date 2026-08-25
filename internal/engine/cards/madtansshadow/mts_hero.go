// Package madtansshadow registers The Mad Titan's Shadow box: campaign
// aspect cards, the Ebony Maw, Tower Defense, Thanos, Hela and Loki
// scenarios, and the box's modular sets.
package madtansshadow

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerMTSHeroCards()
	registerInfinityStones()
	registerEbonyMaw()
	registerTowerDefense()
	registerThanos()
	registerHela()
	registerLoki()
	registerCampaignSchemes()
	registerMTSScenarios()
}

// registerMTSHeroCards installs the campaign aspect cards (21011–21025,
// 21041–21065).
func registerMTSHeroCards() {
	// 21011 Captain America: cost scales with Avengers.
	engine.RegisterBehavior("21011", &engine.Behavior{})

	// 21012 Power Man: 2 chi counters → +2 ATK each.
	engine.RegisterBehavior("21012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 2}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			a := g.Allies[e.EID()]
			if a == nil || a.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.discardChiCounters2AtkEachUntilEndOfPhase"), Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if a := g.Allies[self]; a != nil {
						n := a.Counters
						a.Counters = 0
						return []engine.Message{engine.AllyStatBonus{Ally: self, ATK: 2 * n}}
					}
					return nil
				},
			}}
		},
	})

	// 21013 White Tiger: draw cards equal to villain stage.
	engine.RegisterBehavior("21013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 1
			for _, v := range g.Villains {
				if v != nil && v.Stage > n {
					n = v.Stage
				}
			}
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: n}}
		},
	})

	// 21014 Kaluu: tutor an event from the top 5.
	engine.RegisterBehavior("21014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			n := min(5, len(p.Deck))
			for i := 0; i < n; i++ {
				c := p.Deck[i]
				if c.Def().Type == "event" {
					picks = append(picks, engine.Choice{
						Label: engine.S(c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID}, engine.ShufflePlayerDeck{Player: p.ID}))
				}
			}
			if len(picks) == 0 {
				return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.kaluuAddWhichEventToHand"), picks...)}}
		},
	})

	// 21015 Mighty Avengers: all-Avenger board buff.
	engine.RegisterBehavior("21015", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			// Applied as a passive on each round start (approximation).
			return nil
		},
	})

	// 21016 Mass Attack: exhaust 3 same-trait allies → big hit.
	engine.RegisterBehavior("21016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			total := p.AttackStat(g)
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					total += a.AttackVal
				}
			}
			if total <= 0 {
				return nil
			}
			return cardutil.ChooseEnemy(engine.Tf("c.massAttackChooseAnEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) {
					return total, nil
				})(g, e)
		},
	})

	// 21017 Moxie: after changing form, +1 all stats this round.
	engine.RegisterBehavior("21017", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			cf, ok := msg.(engine.ChangeForm)
			if !ok || cf.Player != e.EOwner() {
				return nil
			}
			return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), ATK: 1, THW: 1, DEF: 1}}
		},
	})

	// 21018 Band Together: wild per ally (payment-layer approximation:
	// flat wild).
	engine.RegisterBehavior("21018", &engine.Behavior{})

	// 21019 Blade: pay or discard after each action.
	engine.RegisterBehavior("21019", &engine.Behavior{})

	// 21020 Avengers Tower: ally limit + discount.
	engine.RegisterBehavior("21020", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustAvengersTowerNextAvengerAllyThisPhaseCosts1Less"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(e.EOwner())
					if p == nil {
						return nil
					}
					return []engine.Message{engine.CostDiscountApply{Player: p.ID, Amount: 1}}
				},
			}}
		},
	})

	// 21021 Avengers Mansion: exhaust → draw.
	engine.RegisterBehavior("21021", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustAvengersMansionAPlayerDraws1Card"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var picks []engine.Choice
					for _, p := range g.Players {
						picks = append(picks, engine.Choice{
							ID: string(p.ID), Label: engine.S(p.Name), Kind: engine.ChoiceLabel,
						}.Msgs(engine.DrawCards{Player: p.ID, N: 1}))
					}
					return []engine.Message{engine.AskQuestion{Player: g.ActiveTurn,
						Question: engine.Ask(engine.Tf("c.avengersMansionWhoDraws2"), picks...)}}
				},
			}}
		},
	})

	// 21022 Ready to Rumble: after changing form, discard to ready.
	engine.RegisterBehavior("21022", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			cf, ok := msg.(engine.ChangeForm)
			if !ok || cf.Player != e.EOwner() {
				return nil
			}
			return []engine.Message{
				engine.DiscardControlled{Player: e.EOwner(), ID: e.EID()},
				engine.ReadyEntity{ID: e.EOwner()},
			}
		},
	})

	// 21023–21025 basic resources.
	engine.RegisterBehavior("21023", &engine.Behavior{})
	engine.RegisterBehavior("21024", &engine.Behavior{})
	engine.RegisterBehavior("21025", &engine.Behavior{})

	// 21041 Marvel Boy: piercing attacks (approximated flat).
	engine.RegisterBehavior("21041", &engine.Behavior{})

	// 21042/21048/21054/21060 the Eternity cycle: shuffle into the
	// encounter deck, fire on reveal.
	eternity := func(effect func(g *engine.Game, e engine.Entity) []engine.Message) *engine.Behavior {
		return &engine.Behavior{
			OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
				g.EncounterDeck = append(g.EncounterDeck, engine.Card{ID: g.NextCardID(), Code: e.ECode()})
				g.ShuffleEncounterDeck()
				return nil
			},
			ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
				g.Delete(t.ID)
				return effect(g, t)
			},
		}
	}
	engine.RegisterBehavior("21042", eternity(func(g *engine.Game, e engine.Entity) []engine.Message {
		for id := range g.Villains {
			return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: e.EID()}}
		}
		return nil
	}))
	engine.RegisterBehavior("21048", eternity(func(g *engine.Game, e engine.Entity) []engine.Message {
		if g.MainScheme != nil {
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()}}
		}
		return nil
	}))
	engine.RegisterBehavior("21054", eternity(func(g *engine.Game, e engine.Entity) []engine.Message {
		return []engine.Message{engine.DrawCards{Player: cardutil.FirstPlayerID(g), N: 1}}
	}))
	engine.RegisterBehavior("21060", eternity(func(g *engine.Game, e engine.Entity) []engine.Message {
		return []engine.Message{engine.HealEntity{Target: cardutil.FirstPlayerID(g), N: 2}}
	}))

	// 21043 Magic Attack: mill up to 5 → damage per card.
	engine.RegisterBehavior("21043", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := min(5, len(p.Deck))
			return append([]engine.Message{engine.MillPlayerDeck{Player: p.ID, N: n}},
				cardutil.ChooseEnemy(engine.Tf("c.magicAttackDealDamage"),
					func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(g, e)...)
		},
	})

	// 21044 Uppercut: 5 damage.
	engine.RegisterBehavior("21044", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.uppercutDeal5Damage2"),
			func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 5, nil }),
	})

	// 21045 Combat Training: +1 ATK.
	engine.RegisterBehavior("21045", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} },
	})

	// 21046 Audacity: after spending, 1 damage (approximated: discard
	// trigger).
	engine.RegisterBehavior("21046", &engine.Behavior{})

	// 21047 Quasar: on entering play, 1 threat off each scheme.
	engine.RegisterBehavior("21047", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			if g.MainScheme != nil {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()})
			}
			for _, id := range sortedSchemeIDs(g) {
				msgs = append(msgs, engine.ThwartScheme{Scheme: id, N: 1, Source: e.EID()})
			}
			return msgs
		},
	})

	// 21049 For Justice!: remove 3 threat.
	engine.RegisterBehavior("21049", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme(engine.Tf("c.chooseAScheme", "For Justice!"), func(g *engine.Game, s engine.Entity) int { return 3 }),
	})

	// 21050 Zone of Silence: mill up to 4 → threat removal per card.
	engine.RegisterBehavior("21050", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := min(4, len(p.Deck))
			return append([]engine.Message{engine.MillPlayerDeck{Player: p.ID, N: n}},
				cardutil.ChooseScheme(engine.Tf("c.chooseAScheme", "Zone of Silence"), func(g *engine.Game, s engine.Entity) int { return n })(g, e)...)
		},
	})

	// 21051 Heroic Intuition: +1 THW.
	engine.RegisterBehavior("21051", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1} },
	})

	// 21052 Determination: after spending, remove 1 threat.
	engine.RegisterBehavior("21052", &engine.Behavior{})

	// 21053 Major Victory: on defeat, ready a Guardian.
	engine.RegisterBehavior("21053", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.AllyDefeated)
			if !ok || d.AllyID != e.EID() {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.Exhausted && g.EntityHasTrait(id, "guardian") {
					return []engine.Message{engine.ReadyEntity{ID: id}}
				}
			}
			return nil
		},
	})

	// 21055 Summoning Spell: mill until an ally joins play.
	engine.RegisterBehavior("21055", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for len(p.Deck) > 0 {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				if c.Def().Type == "ally" {
					p.Discard = append(p.Discard, c)
					return []engine.Message{engine.PlayDiscardAlly{Player: p.ID, Card: c}}
				}
				p.Discard = append(p.Discard, c)
			}
			return nil
		},
	})

	// 21056 Make the Call: pay to revive an ally.
	engine.RegisterBehavior("21056", &engine.Behavior{})

	// 21057 Inspired: +1 THW/+1 ATK to an ally.
	engine.RegisterBehavior("21057", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if a := g.Allies[target]; a != nil {
				a.PermTHW++
				a.PermATK++
			}
			return nil
		},
	})

	// 21058 Innovation: after spending, heal an ally.
	engine.RegisterBehavior("21058", &engine.Behavior{})

	// 21059 Charlie-27: retaliate + toughness (keywords).
	engine.RegisterBehavior("21059", &engine.Behavior{})

	// 21061 Shield Spell: discard damage-worth of cards, prevent all.
	engine.RegisterBehavior("21061", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			if !p.IsHero() {
				return engine.Defends{}, nil, false
			}
			attack := 0
			switch t := g.Entity(against).(type) {
			case *engine.Villain:
				attack = t.AttackVal + t.BoostCount
			case *engine.Minion:
				attack = t.AttackVal
			}
			if len(p.Deck) < attack {
				return engine.Defends{}, nil, false
			}
			d := engine.Defends{Defender: p.ID, Against: against, PreventAll: true}
			return d, []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: attack}}, true
		},
	})

	// 21062 Counter-Punch: after defending, hit back.
	engine.RegisterBehavior("21062", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			if !ok || w.Defender != e.EOwner() {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: w.Against, Damage: p.AttackStat(g), Source: p.ID}}
		},
	})

	// 21063 Armored Vest: +1 DEF.
	engine.RegisterBehavior("21063", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{DEF: 1} },
	})

	// 21064 Preservation: after spending, heal 1.
	engine.RegisterBehavior("21064", &engine.Behavior{})

	// 21065 Martinex: Guardian discount.
	engine.RegisterBehavior("21065", &engine.Behavior{})
}

// sortedSchemeIDs lists side scheme ids in stable order.
func sortedSchemeIDs(g *engine.Game) []engine.EntityID {
	var out []engine.EntityID
	for id := range g.SideSchemes {
		out = append(out, id)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
