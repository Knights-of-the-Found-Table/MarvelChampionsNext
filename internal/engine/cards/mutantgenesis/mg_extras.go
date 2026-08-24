// mg_extras.go implements the Mystique side scenario (32080–32083), the
// Future Past set (32166–32170), the Mutant Genesis campaign cards
// (32171–32175) and the X-Force aspect upgrades (32176–32195).
package mutantgenesis

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerMystique()
	registerFuturePast()
	registerCampaign()
	registerXForceAspects()
}

func registerMystique() {
	// 32080 Mystique: cannot attack the villain while she is in play; her
	// stats mirror the villain's (approximation: fixed 2/2).
	engine.RegisterBehavior("32080", &engine.Behavior{
		MinionDamageableSrc: func(g *engine.Game, mn *engine.Minion, damage int, src engine.EntityID) bool {
			return true
		},
	})

	// 32081 Metamorphic Mayhem: When Defeated — shuffle each
	// Shapeshifter card from the encounter discard into the players'
	// decks (approximation: the encounter deck).
	engine.RegisterBehavior("32081", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			shuffled := 0
			for i := 0; i < len(g.EncounterDiscard); {
				c := g.EncounterDiscard[i]
				if c.Def().HasTrait("shapeshifter") {
					g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
					g.EncounterDeck = append(g.EncounterDeck, c)
					shuffled++
					continue
				}
				i++
			}
			if shuffled > 0 {
				g.Logf("Metamorphic Mayhem — %d Shapeshifters return", shuffled)
			}
			return nil
		},
	})

	// 32082 Infiltration / 32083 Shapeshifter Surprise: shuffle into the
	// revealing player's deck (approximation: discard) with surge; the
	// enters-your-hand riders are not modeled.
	for _, code := range []string{"32082", "32083"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
				g.Delete(t.ID)
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			},
		})
	}
}

func registerFuturePast() {
	// 32166 Nimrod: the 3-damage-per-phase cap is approximated by
	// ignoring any damage beyond 3 in a single hit.
	engine.RegisterBehavior("32166", &engine.Behavior{
		MinionDamageable: func(g *engine.Game, mn *engine.Minion, damage int) bool {
			if damage > 3 {
				g.Logf("Nimrod ignores damage beyond 3")
				mn.Damage += damage - 3 // apply the capped remainder here
				return false            // the uncapped hit does not land
			}
			return true
		},
	})

	// 32167 Bastion: villainous/toughness from data; boost rider skipped.
	engine.RegisterBehavior("32167", &engine.Behavior{})

	// 32168 Nimrod's Portal: When Defeated — each player mills 2, indirect
	// damage per boost icon.
	engine.RegisterBehavior("32168", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			var msgs []engine.Message
			for _, p := range g.Players {
				n := 0
				for i := 0; i < 2 && len(p.Deck) > 0; i++ {
					top := p.Deck[0]
					p.Deck = p.Deck[1:]
					n += cardutil.BoostOf(top)
					p.Discard = append(p.Discard, top)
				}
				if n > 0 {
					msgs = append(msgs, engine.IndirectDamage{Player: p.ID, N: n})
				}
			}
			return msgs
		},
	})

	// 32169 Bastion's Machinations: each player tucks 9 deck cards here.
	engine.RegisterBehavior("32169", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			for _, p := range g.Players {
				for i := 0; i < 9 && len(p.Deck) > 0; i++ {
					top := p.Deck[0]
					p.Deck = p.Deck[1:]
					s.StoredCards = append(s.StoredCards, top)
				}
			}
			g.Logf("Bastion's Machinations — 9 cards tucked from each player")
			return nil
		},
	})

	// 32170 Nano-Sentinel Tech: attach to a minion (+4 HP, Sentinel
	// trait); the nemesis search rider is approximated.
	engine.RegisterBehavior("32170", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil {
					continue
				}
				t.Target = mn.ID
				mn.Attachments = append(mn.Attachments, t.ID)
				mn.MaxHP += 4
				return nil
			}
			g.Delete(t.ID)
			return nil
		},
	})
}

func registerCampaign() {
	// 32171–32175 campaign side schemes: each shuffles a Future Past card
	// into the encounter deck when defeated (the deck itself is not
	// modeled; the flips resolve as their b-side spawn effects).
	engine.RegisterBehavior("32171", &engine.Behavior{
		React: campaignFlip(func(g *engine.Game, pid engine.PlayerID) []engine.Message {
			// Metro P.D. joins the first player.
			a := &engine.Ally{ID: g.NextEntityID(engine.KindAlly), Code: "32171b", Owner: pid, MaxHP: 3}
			g.Allies[a.ID] = a
			if p := g.Player(pid); p != nil {
				p.Allies = append(p.Allies, a.ID)
			}
			g.Logf("Metro P.D. joins the fight")
			return nil
		}),
	})
	engine.RegisterBehavior("32172", &engine.Behavior{
		React: campaignFlip(func(g *engine.Game, pid engine.PlayerID) []engine.Message {
			// Magneto joins the players.
			a := &engine.Ally{ID: g.NextEntityID(engine.KindAlly), Code: "32172b", Owner: pid, MaxHP: 6, ThwartVal: 2, AttackVal: 2}
			g.Allies[a.ID] = a
			if p := g.Player(pid); p != nil {
				p.Allies = append(p.Allies, a.ID)
			}
			g.Logf("Magneto allies with the players!")
			return nil
		}),
	})
	// 32173 Find the Prisoners: each player tucks an ally from their deck.
	engine.RegisterBehavior("32173", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			for _, p := range g.Players {
				for i, c := range p.Deck {
					if c.Def().Type == "ally" {
						p.Deck = append(p.Deck[:i], p.Deck[i+1:]...)
						s.StoredCards = append(s.StoredCards, c)
						break
					}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("32174", &engine.Behavior{
		React: campaignFlip(func(g *engine.Game, pid engine.PlayerID) []engine.Message {
			g.Logf("Reactivate Defenses enters %s's play area", g.Player(pid).Name)
			return nil
		}),
	})
	engine.RegisterBehavior("32175", &engine.Behavior{
		React: campaignFlip(func(g *engine.Game, pid engine.PlayerID) []engine.Message {
			return nil
		}),
	})
}

// campaignFlip wraps a When-Defeated campaign flip effect.
func campaignFlip(effect func(g *engine.Game, pid engine.PlayerID) []engine.Message) func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
	return func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.SchemeDefeated)
		if !ok || m.Scheme != e.EID() {
			return nil
		}
		return effect(g, cardutil.FirstPlayerID(g))
	}
}

// registerXForceAspects installs the 20 campaign-pool aspect upgrades.
// Each is a one-shot ability that removes itself from the game.
func registerXForceAspects() {
	// Coup de Grâce (32176 brawler / 32181 commander): +3 damage on your
	// next attack this phase.
	coup := func() *engine.Behavior {
		return &engine.Behavior{
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				return []engine.Ability{{
					Label: "Coup de Grâce — your attacks this phase deal +3 damage, then remove this card", Type: engine.AbilityAction,
					Trigger: "attack",
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						u := g.Upgrades[self]
						if u == nil {
							return nil
						}
						g.EventDamageBonus[u.Owner] += 3
						g.Delete(self)
						return nil
					},
				}}
			},
		}
	}
	engine.RegisterBehavior("32176", coup())
	engine.RegisterBehavior("32181", coup())

	// Swagger (32177 brawler / 32186 defender): +3 DEF once, then ready.
	swagger := func() *engine.Behavior {
		return &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				m, ok := msg.(engine.Defends)
				u := g.Upgrades[e.EID()]
				if !ok || u == nil || m.Defender != u.Owner {
					return nil
				}
				g.Delete(u.ID)
				return []engine.Message{engine.ApplyStatBonus{Target: u.Owner, DEF: 3}, engine.ReadyEntity{ID: u.Owner}}
			},
		}
	}
	engine.RegisterBehavior("32177", swagger())
	engine.RegisterBehavior("32186", swagger())

	// Brazen Defense (32178): when defending, prevent 3 and deal 3.
	engine.RegisterBehavior("32178", &engine.Behavior{
		DefenseSubstitute: func(g *engine.Game, p *engine.Player, u *engine.Upgrade, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			g.Delete(u.ID)
			return engine.Defends{Defender: p.ID, Against: against, ExtraPrevent: 3},
				[]engine.Message{engine.DamageEntity{Target: against, Damage: 3, Source: p.ID}}, true
		},
	})

	// Ferocious Attack (32179): spend 3 → 6 damage + ready.
	engine.RegisterBehavior("32179", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Ferocious Attack — spend 3 resources → deal 6 damage and ready your hero", Type: engine.AbilityAction,
				HeroOnly: true, Cost: 3,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					g.Delete(self)
					choices := cardutil.EnemyChoices(g, 6, u.Owner, func(target engine.EntityID) []engine.Message {
						return []engine.Message{
							engine.DamageEntity{Target: target, Damage: 6, Source: u.Owner},
							engine.ReadyEntity{ID: u.Owner},
						}
					})
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   u.Owner,
						Question: engine.Ask("Ferocious Attack — deal 6 damage", choices...),
					}}
				},
			}}
		},
	})

	// War Cry (32180): [wild][wild] for Attack/Defense events + tough.
	engine.RegisterBehavior("32180", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true},
	})

	// Compassion (32182 commander / 32192 peacekeeper): after you recover,
	// heal 3 among your characters and draw 1.
	compassion := func() *engine.Behavior {
		return &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				m, ok := msg.(engine.BasicRecover)
				u := g.Upgrades[e.EID()]
				if !ok || u == nil || m.Player != u.Owner {
					return nil
				}
				g.Delete(u.ID)
				p := g.Player(u.Owner)
				if p == nil {
					return nil
				}
				msgs := []engine.Message{engine.HealEntity{Target: p.ID, N: 3}, engine.DrawCards{Player: p.ID, N: 1}}
				for _, id := range p.Allies {
					msgs = append(msgs, engine.HealEntity{Target: id, N: 3})
				}
				return msgs
			},
		}
	}
	engine.RegisterBehavior("32182", compassion())
	engine.RegisterBehavior("32192", compassion())

	// Group Assault (32183): this phase allies take no consequential
	// attack damage (approximation: heal 1 after each ally attack).
	engine.RegisterBehavior("32183", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil {
				return nil
			}
			return []engine.Message{engine.HealEntity{Target: m.Ally, N: 1}}
		},
	})

	// Shock and Awe (32184): spend 3 → 6 damage + ready each ally.
	engine.RegisterBehavior("32184", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Shock and Awe — spend 3 resources → deal 6 damage and ready each ally", Type: engine.AbilityAction,
				HeroOnly: true, Cost: 3,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					g.Delete(self)
					var msgs []engine.Message
					for _, id := range cardutil.SortedEnemyIDs(g) {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 6, Source: u.Owner})
						break
					}
					p := g.Player(u.Owner)
					if p != nil {
						for _, id := range p.Allies {
							msgs = append(msgs, engine.ReadyEntity{ID: id})
						}
					}
					return msgs
				},
			}}
		},
	})

	// Improvisation (32185): [wild][wild] + ready an ally.
	engine.RegisterBehavior("32185", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true},
	})

	// Surprise! (32187 defender / 32191 peacekeeper): after you thwart,
	// remove 3 threat and confuse an enemy.
	surprise := func() *engine.Behavior {
		return &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				m, ok := msg.(engine.BasicThwart)
				u := g.Upgrades[e.EID()]
				if !ok || u == nil || m.Player != u.Owner {
					return nil
				}
				g.Delete(u.ID)
				var msgs []engine.Message
				if g.MainScheme != nil {
					msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 3, Source: u.Owner})
				}
				if v := activeOrFirstVillain(g); v != nil {
					msgs = append(msgs, engine.ConfuseEntity{Target: v.ID})
				}
				return msgs
			},
		}
	}
	engine.RegisterBehavior("32187", surprise())
	engine.RegisterBehavior("32191", surprise())

	// Heroic Intervention (32188): spend 3 → remove 5 threat + tough.
	engine.RegisterBehavior("32188", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Heroic Intervention — spend 3 resources → remove 5 threat and gain tough", Type: engine.AbilityAction,
				HeroOnly: true, Cost: 3,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					g.Delete(self)
					var msgs []engine.Message
					if g.MainScheme != nil {
						msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 5, Source: u.Owner})
					}
					return append(msgs, engine.ToughEntity{Target: u.Owner})
				},
			}}
		},
	})

	// Determined Defense (32189): when you defend, spend 2 → the attack
	// removes 3 threat from the main scheme instead.
	engine.RegisterBehavior("32189", &engine.Behavior{
		DefenseSubstitute: func(g *engine.Game, p *engine.Player, u *engine.Upgrade, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			g.Delete(u.ID)
			var msgs []engine.Message
			if g.MainScheme != nil {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 3, Source: p.ID})
			}
			return engine.Defends{Defender: p.ID, Against: against, PreventAll: true}, msgs, true
		},
	})

	// Bodyguard (32190): [wild][wild] + draw 1.
	engine.RegisterBehavior("32190", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true},
	})

	// Rescue Operation (32193): this phase allies take no consequential
	// thwart damage (approximation as Group Assault).
	engine.RegisterBehavior("32193", engine.LookupBehavior("32183"))

	// Mentorship (32194): spend 3 → remove 5 threat + ready each ally.
	engine.RegisterBehavior("32194", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Mentorship — spend 3 resources → remove 5 threat and ready each ally", Type: engine.AbilityAction,
				HeroOnly: true, Cost: 3,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					g.Delete(self)
					var msgs []engine.Message
					if g.MainScheme != nil {
						msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 5, Source: u.Owner})
					}
					p := g.Player(u.Owner)
					if p != nil {
						for _, id := range p.Allies {
							msgs = append(msgs, engine.ReadyEntity{ID: id})
						}
					}
					return msgs
				},
			}}
		},
	})

	// Fortitude (32195): [wild][wild] + stun an enemy.
	engine.RegisterBehavior("32195", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true},
	})
}
