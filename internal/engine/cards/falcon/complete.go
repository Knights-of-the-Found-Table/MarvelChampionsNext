package falcon

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() { registerFalconComplete() }

func falconInt(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

// findEncounter takes the first card matching pred from the encounter deck
// or discard pile.
func findEncounter(g *engine.Game, pred func(*data.CardDef) bool) (engine.Card, bool) {
	for i, c := range g.EncounterDeck {
		if pred(c.Def()) {
			g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
			return c, true
		}
	}
	for i, c := range g.EncounterDiscard {
		if pred(c.Def()) {
			g.EncounterDiscard = append(g.EncounterDiscard[:i:i], g.EncounterDiscard[i+1:]...)
			return c, true
		}
	}
	return engine.Card{}, false
}

// engageMinionCard spawns a minion card engaged with a player.
func engageMinionCard(g *engine.Game, code string, pid engine.PlayerID) *engine.Minion {
	def := engine.Card{Code: code}.Def()
	mn := &engine.Minion{
		ID: g.NextEntityID(engine.KindMinion), Code: code,
		MaxHP: falconInt(def.HP, 1), AttackVal: falconInt(def.Attack, 0), SchemeVal: falconInt(def.Scheme, 0),
		EngagedWith: pid,
	}
	g.Minions[mn.ID] = mn
	return mn
}

func registerFalconComplete() {
	// 53014 Adam Warlock: random discard payoff.
	engine.RegisterBehavior("53014", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var used bool
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				used = m.Ally == e.EID()
			case engine.AllyThwartWindow:
				used = m.Ally == e.EID()
			}
			if !used {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil || len(p.Hand) == 0 {
				return nil
			}
			idx := g.Random(len(p.Hand))
			card := p.Hand[idx]
			p.Hand = append(p.Hand[:idx:idx], p.Hand[idx+1:]...)
			p.Discard = append(p.Discard, card)
			g.TLogf("c.adamWarlockDiscards", card)
			icon := ""
			if rs := card.Def().Resources; len(rs) > 0 {
				icon = rs[0]
			}
			switch icon {
			case "physical":
				if g.MainScheme != nil {
					return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 3, Source: e.EID()}}
				}
			case "energy":
				return []engine.Message{engine.HealEntity{Target: p.ID, N: 3}}
			default:
				for _, id := range enemyFirst(g) {
					return []engine.Message{engine.DamageEntity{Target: id, Damage: 3, Source: e.EID()}}
				}
			}
			return nil
		},
	})

	// 53015 Aero / 53016 Cloud 9: aerial team buffs.
	aeroBuff := func(stat string) *engine.Behavior {
		return &engine.Behavior{
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				return []engine.Ability{{
					Label: engine.S("Exhaust — " + map[string]string{"atk": "Aero: Aerial allies +1 ATK", "thw": "Cloud 9: Aerial allies +1 THW"}[stat]),
					Type:  engine.AbilityAction, HeroOnly: true, Exhaust: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						var out []engine.Message
						for _, p := range g.Players {
							for _, aid := range p.Allies {
								if a := g.Allies[aid]; a != nil && a.EDef().HasTrait("aerial") {
									if stat == "atk" {
										out = append(out, engine.AllyStatBonus{Ally: aid, ATK: 1})
									} else {
										out = append(out, engine.AllyStatBonus{Ally: aid, THW: 1})
									}
								}
							}
						}
						return out
					},
				}}
			},
		}
	}
	engine.RegisterBehavior("53015", aeroBuff("atk"))
	engine.RegisterBehavior("53016", aeroBuff("thw"))

	// 53017 Hugin & Munin: scout a minion and ready per boost icon.
	engine.RegisterBehavior("53017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			if card, ok := findEncounter(g, func(d *data.CardDef) bool { return d.Type == "minion" }); ok {
				mn := engageMinionCard(g, card.Code, p.ID)
				n := 1
				if mn.EDef().Boost != nil {
					n = *mn.EDef().Boost
				}
				if n > 2 {
					n = 2
				}
				var out []engine.Message
				out = append(out, engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
				if !p.Exhausted && n > 0 {
					out = append(out, engine.ReadyEntity{ID: p.ID})
					n--
				}
				for _, aid := range p.Allies {
					if n <= 0 {
						break
					}
					if a := g.Allies[aid]; a != nil && a.Exhausted {
						out = append(out, engine.ReadyEntity{ID: aid})
						n--
					}
				}
				return out
			}
			return nil
		},
	})

	// 53018 Spectrum: tuck rider approximated as a flat +2/+2.
	engine.RegisterBehavior("53018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			if a != nil {
				a.PermATK += 2
				a.PermTHW += 2
			}
			return nil
		},
	})

	// 53019 Strength in Diversity: one effect per distinct friendly trait.
	engine.RegisterBehavior("53019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			traits := map[string]bool{p.EDef().Name: true}
			count := 1
			for _, aid := range p.Allies {
				if a := g.Allies[aid]; a != nil && !traits[a.EDef().Name] {
					traits[a.EDef().Name] = true
					count++
				}
			}
			if count > 3 {
				count = 3
			}
			var out []engine.Message
			if g.MainScheme != nil {
				out = append(out, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: count, Source: e.EID()})
			}
			for _, id := range enemyFirst(g) {
				out = append(out, engine.DamageEntity{Target: id, Damage: count, Source: e.EID()})
				break
			}
			return out
		},
	})

	// 53020 Flight Squadron: ready an Aerial ally.
	engine.RegisterBehavior("53020", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.flightSquadronReadyAnAerialAlly"), Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil {
						return nil
					}
					for _, aid := range p.Allies {
						if a := g.Allies[aid]; a != nil && a.Exhausted && a.EDef().HasTrait("aerial") {
							return []engine.Message{engine.ReadyEntity{ID: aid}}
						}
					}
					return nil
				},
			}}
		},
	})

	// 53021 Resource Reserve: bank a resource card for anyone.
	engine.RegisterBehavior("53021", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || len(s.AttachedCards) > 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.resourceReserveTuckAResourceCardFromYourHand"), Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil || len(p.Hand) == 0 {
						return nil
					}
					card := p.Hand[0]
					p.Hand = p.Hand[1:]
					s.AttachedCards = append(s.AttachedCards, card)
					s.Counters = 1
					g.TLogf("c.isBankedOnResourceReserve", card)
					return nil
				},
			}}
		},
	})

	// 53022 Triskelion: ally-limit raise not modeled.
	engine.RegisterBehavior("53022", &engine.Behavior{})

	// 53023 Captain America title: fetch the shield.
	engine.RegisterBehavior("53023", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.captainAmericaSpendAResourceFetchCapSShield"), Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true, Cost: 1,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.Owner)
					if p == nil {
						return nil
					}
					for i, c := range p.Deck {
						if c.Code == "53034" {
							card := c
							p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
							p.Hand = append(p.Hand, card)
							g.TLogf("c.capSShieldAnswersTheCall")
							return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
						}
					}
					for _, c := range p.Discard {
						if c.Code == "53034" {
							return []engine.Message{engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}}
						}
					}
					return nil
				},
			}}
		},
	})

	// 53024 Wingman: consequential-damage redirect not modeled.
	engine.RegisterBehavior("53024", &engine.Behavior{})
	// 53025-53027 basic resources; 53028 Power of Flight (engine
	// powerOfBonus covers the doubling).
	engine.RegisterBehavior("53025", &engine.Behavior{})
	engine.RegisterBehavior("53026", &engine.Behavior{})
	engine.RegisterBehavior("53027", &engine.Behavior{})
	engine.RegisterBehavior("53028", &engine.Behavior{})

	// 53034 Cap's Shield.
	engine.RegisterBehavior("53034", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{DEF: 1, Retaliate: 1}
		},
	})

	// 53035 Winter Soldier: kills refund threat removal.
	engine.RegisterBehavior("53035", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			if !ok || m.Ally != e.EID() || g.MainScheme == nil {
				return nil
			}
			if mn := g.Minions[m.Target]; mn != nil && mn.HP() <= 2 {
				return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()}}
			}
			return nil
		},
	})

	// 53036 Misty Knight: boost-scaling thwart not modeled.
	engine.RegisterBehavior("53036", &engine.Behavior{})
	// 53037 Ops Room: prevention window not modeled; counters tracked.
	engine.RegisterBehavior("53037", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Supports[e.EID()].Counters = 3
			return nil
		},
	})

	// --- Techno nemesis set (53038-53042) ---
	engine.RegisterBehavior("53038", &engine.Behavior{})
	engine.RegisterBehavior("53039", &engine.Behavior{})
	engine.RegisterBehavior("53040", &engine.Behavior{})
	engine.RegisterBehavior("53041", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if card, ok := findEncounter(g, func(d *data.CardDef) bool {
				return d.Type == "attachment" && d.HasTrait("tech")
			}); ok {
				return []engine.Message{engine.RevealEncounterCard{Player: g.Players[0].ID, Card: card}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("53042", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var mn *engine.Minion
			for _, m := range g.Minions {
				if m != nil && m.Code == "53038" {
					mn = m
					break
				}
			}
			if mn == nil {
				if card, ok := findEncounter(g, func(d *data.CardDef) bool { return d.Code == "53038" }); ok {
					mn = engageMinionCard(g, card.Code, p.ID)
				}
			}
			if mn == nil {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return []engine.Message{engine.MinionActivates{MinionID: mn.ID, Player: p.ID}}
		},
	})
}

// enemyFirst lists enemy ids (villain first).
func enemyFirst(g *engine.Game) []engine.EntityID {
	var out []engine.EntityID
	for id := range g.Villains {
		out = append(out, id)
	}
	for id := range g.Minions {
		out = append(out, id)
	}
	return out
}
