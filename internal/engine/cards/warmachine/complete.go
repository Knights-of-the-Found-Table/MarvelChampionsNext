// Package warmachine registers the War Machine hero pack: the ammo
// counter economy and the Living Laser nemesis set. Ammo counters live
// on the identity via GrowthCounters (repurposed as the ammo pool).
package warmachine

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerWarMachine()
	registerNemesis()
}

// ammo reads the identity's ammo pool.
func ammo(p *engine.Player) int { return p.GrowthCounters }

func registerWarMachine() {
	// War Machine identity: Locked and Loaded — 5 ammo on entering hero
	// form.
	engine.RegisterBehavior("23001", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			cf, ok := msg.(engine.ChangeForm)
			if !ok || cf.Player != e.EID() {
				return nil
			}
			p := g.Player(cf.Player)
			if p == nil || p.IsHero() {
				return nil
			}
			// Reactions run pre-flip: pre-flip alter-ego means entering
			// hero form.
			p.GrowthCounters += 5
			g.TLogf("c.loads5AmmoCountersLockedAndLoaded", p.Name)
			return nil
		},
	})

	// Iron Man (ally): tech upgrade search.
	engine.RegisterBehavior("23002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			seen := map[string]bool{}
			for _, c := range append(append(engine.CardList{}, p.Deck...), p.Discard...) {
				def := c.Def()
				if def.Type == "upgrade" && def.HasTrait("tech") && !seen[c.Code] {
					seen[c.Code] = true
					if _, inDeck := p.Deck.Find(c.ID); inDeck {
						picks = append(picks, engine.Choice{Label: engine.S(def.Name + " (deck)"), Kind: engine.ChoiceCard, CardCode: def.Code}.
							Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID}, engine.ShufflePlayerDeck{Player: p.ID}))
					} else {
						picks = append(picks, engine.Choice{Label: engine.S(def.Name + " (discard)"), Kind: engine.ChoiceCard, CardCode: def.Code}.
							Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
					}
				}
			}
			if len(picks) == 0 {
				return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.ironManAddWhichTechUpgradeToHand"), picks...)}}
		},
	})

	// Munitions Bunker: store 2 ammo; dump them to the identity.
	engine.RegisterBehavior("23003", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil {
				return nil
			}
			ab := []engine.Ability{{
				Label: engine.Tf("c.exhaustMunitionsBunkerPlace2AmmoHere"), Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.AddEntityCounter{ID: self, N: 2}}
				},
			}}
			if s.Counters > 0 {
				ab = append(ab, engine.Ability{
					Label: engine.Tf("c.exhaustMunitionsBunkerMoveAllAmmoHereToWarMachine"), Type: engine.AbilityAction,
					Exhaust: true, HeroOnly: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						s := g.Supports[self]
						n := s.Counters
						s.Counters = 0
						p := g.Player(s.EOwner())
						if p != nil {
							p.GrowthCounters += n
							g.TLogf("c.loadsAmmoCounterS", p.Name, n)
						}
						return nil
					},
				})
			}
			return ab
		},
	})

	// Upgraded Chassis: Aerial; tough on entering hero form.
	engine.RegisterBehavior("23004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "aerial"}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			cf, ok := msg.(engine.ChangeForm)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || cf.Player != u.Owner || u.Exhausted {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil || p.IsHero() {
				return nil
			}
			return []engine.Message{
				engine.ExhaustEntity{ID: e.EID()},
				engine.ToughEntity{Target: u.Owner},
			}
		},
	})

	// Gauntlet Gun: wild resource + 1 ammo (counter accrual approximated
	// on use is not visible to the payment layer; ammo gain skipped).
	engine.RegisterBehavior("23005", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})

	// Missile Launcher: exhaust + 1 ammo → 2 damage.
	engine.RegisterBehavior("23006", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || ammo(p) <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustMissileLauncher1AmmoDeal2Damage"), Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					p := g.Player(u.EOwner())
					p.GrowthCounters--
					return cardutil.ChooseEnemy(engine.Tf("c.missileLauncherDeal2DamageToWhichEnemy"),
						func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 2, nil })(
						g, &engine.EventCard{Code: "23006", Owner: u.EOwner()})
				},
			}}
		},
	})

	// Shoulder Cannon: exhaust → 1 damage; 1 ammo to re-ready.
	engine.RegisterBehavior("23007", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			ab := []engine.Ability{{
				Label: engine.Tf("c.exhaustShoulderCannonDeal1Damage"), Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return cardutil.ChooseEnemy(engine.Tf("c.shoulderCannonDeal1DamageToWhichEnemy"),
						func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 1, nil })(
						g, g.Entity(self))
				},
			}}
			p := g.Player(e.EOwner())
			if p != nil && ammo(p) > 0 {
				ab = append(ab, engine.Ability{
					Label: engine.Tf("c.1AmmoReReadyShoulderCannon"), Type: engine.AbilityAction,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						u := g.Upgrades[self]
						if u == nil || !u.Exhausted {
							return nil
						}
						p := g.Player(u.EOwner())
						p.GrowthCounters--
						return []engine.Message{engine.ReadyEntity{ID: self}}
					},
				})
			}
			return ab
		},
	})

	// Repulsor Beam: 1 ammo → 4 damage.
	engine.RegisterBehavior("23008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || ammo(p) <= 0 {
				return nil
			}
			p.GrowthCounters--
			return cardutil.ChooseEnemy(engine.Tf("c.repulsorBeamDeal4DamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 4, nil })(g, e)
		},
	})

	// Targeted Strike: 1 ammo → 3 threat.
	engine.RegisterBehavior("23009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || ammo(p) <= 0 {
				return nil
			}
			p.GrowthCounters--
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				engine.Tf("c.targetedStrikeRemove3ThreatFromWhichScheme"), schemePicks(g, 3, e.EOwner())...)}}
		},
	})

	// Scorched Earth: 3 ammo → 3 damage to each enemy.
	engine.RegisterBehavior("23010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || ammo(p) < 3 {
				return nil
			}
			p.GrowthCounters -= 3
			var msgs []engine.Message
			for _, id := range cardutil.SortedEnemyIDs(g) {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 3, Source: p.ID})
			}
			return msgs
		},
	})

	// Full Auto: 4 ammo → 8 damage to one enemy.
	engine.RegisterBehavior("23011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || ammo(p) < 4 {
				return nil
			}
			p.GrowthCounters -= 4
			return cardutil.ChooseEnemy(engine.Tf("c.fullAutoDeal8DamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 8, nil })(g, e)
		},
	})

	// Black Panther: attach a leadership event facedown (storage).
	engine.RegisterBehavior("23012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, c := range p.Discard {
				def := c.Def()
				if def.Type == "event" && def.Aspect == "leadership" {
					picks = append(picks, engine.Choice{Label: engine.S("Attach " + def.Name), Kind: engine.ChoiceCard, CardCode: def.Code}.
						Msgs(engine.SupportStoreCard{ID: e.EID(), Card: c}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.blackPantherAttachWhichLeadershipEvent"), picks...)}}
		},
	})

	// Captain Marvel: mill 4; 3 damage per energy (stun at 2+).
	engine.RegisterBehavior("23013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := 0
			for i := 0; i < 4 && len(p.Deck) > 0; i++ {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				for _, r := range c.Def().Resources {
					if r == "energy" {
						n++
					}
				}
				p.Discard = append(p.Discard, c)
			}
			if n == 0 {
				return nil
			}
			msgs := cardutil.ChooseEnemy(engine.Tf("c.captainMarvelDeal3DamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 3, nil })(
				g, &engine.EventCard{Code: "23013", Owner: p.ID})
			if n >= 2 {
				var target engine.EntityID
				for _, id := range cardutil.SortedEnemyIDs(g) {
					target = id
					break
				}
				if target != "" {
					msgs = append(msgs, engine.StunEntity{Target: target})
				}
			}
			return msgs
		},
	})

	// Falcon: top-3 look; 1 threat per treachery.
	engine.RegisterBehavior("23014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 0
			for i := 0; i < 3 && i < len(g.EncounterDeck); i++ {
				if g.EncounterDeck[i].Def().Type == "treachery" {
					n++
				}
			}
			if n == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				engine.Tf("c.falconRemoveThreatFromWhichScheme"), schemePicks(g, n, e.EOwner())...)}}
		},
	})

	// Goliath: +4 ATK this phase, then discard (max once).
	engine.RegisterBehavior("23015", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			if g.UsedThisTurn["goliath"] {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.goliathGets4AtkThisPhaseThenDiscardHim"), Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					g.UsedThisTurn["goliath"] = true
					return []engine.Message{
						engine.AllyStatBonus{Ally: self, ATK: 4},
						engine.DiscardControlled{Player: g.Entity(self).EOwner(), ID: self},
					}
				},
			}}
		},
	})

	// Command Team: 3 counters, ready an ally.
	engine.RegisterBehavior("23016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustCommandTeamCounterReadyAnAlly"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					var picks []engine.Choice
					for _, q := range g.Players {
						for _, id := range q.Allies {
							if a := g.Allies[id]; a != nil && a.Exhausted {
								picks = append(picks, engine.Choice{Label: engine.S(a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
									Msgs(engine.AddEntityCounter{ID: self, N: -1}, engine.ReadyEntity{ID: a.ID}))
							}
						}
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: s.EOwner(),
						Question: engine.Ask(engine.Tf("c.readyWhichAlly"), picks...)}}
				},
			}}
		},
	})

	// Sneak Attack: trait-sharing ally from hand into play (end-of-phase
	// discard approximated to immediate).
	engine.RegisterBehavior("23017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			hero := p.EDef()
			var picks []engine.Choice
			for _, c := range p.Hand {
				def := c.Def()
				if def.Type != "ally" {
					continue
				}
				for _, t := range hero.Traits {
					if def.HasTrait(t) {
						picks = append(picks, engine.Choice{Label: engine.S(def.Name), Kind: engine.ChoiceCard, CardCode: def.Code}.
							Msgs(engine.PlayCard{Player: p.ID, Card: c, Paid: engine.CostPaid{}}))
						break
					}
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.sneakAttackDeployWhichAlly"), picks...)}}
		},
	})

	// Save the Day: threat equal to the sacrificed ally's cost (the
	// scheme question is delivered inside the sacrifice handler via the
	// generic N payload).
	engine.RegisterBehavior("23018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil {
					continue
				}
				cost := deref(a.EDef().Cost, 0)
				msgs := []engine.Message{engine.DiscardControlled{Player: p.ID, ID: id}}
				if cost > 0 {
					msgs = append(msgs, engine.ThwartScheme{Scheme: mainSchemeID(g), N: cost, Source: p.ID})
				}
				picks = append(picks, engine.Choice{Label: engine.S(a.EDef().Name), Kind: engine.ChoiceCard, CardCode: a.Code}.
					Msgs(msgs...))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.saveTheDaySacrificeWhichAlly"), picks...)}}
		},
	})
	engine.RegisterBehavior("23019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil {
					continue
				}
				cost := deref(a.EDef().Cost, 0)
				msgs := []engine.Message{engine.DiscardControlled{Player: p.ID, ID: id}}
				if cost > 0 {
					msgs = append(msgs, cardutil.ChooseEnemy(
						engine.Tf("c.goDownSwingingDealDamageToWhichEnemy", cost),
						func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return cost, nil })(
						g, &engine.EventCard{Code: "23019", Owner: p.ID})...)
				}
				picks = append(picks, engine.Choice{Label: engine.S(a.EDef().Name), Kind: engine.ChoiceCard, CardCode: a.Code}.
					Msgs(msgs...))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.goDownSwingingSacrificeWhichAlly"), picks...)}}
		},
	})

	// Make the Call reprint: alias core 01071.
	if b := engine.LookupBehavior("01071"); b != nil {
		engine.RegisterBehavior("23020", b)
	}

	// Innovation: spent-response — plain resource.
	engine.RegisterBehavior("23021", &engine.Behavior{})

	// Mockingbird reprint: alias core 01083.
	if b := engine.LookupBehavior("01083"); b != nil {
		engine.RegisterBehavior("23022", b)
	}

	// Quincarrier reprint: alias bkw 08023.
	if b := engine.LookupBehavior("08023"); b != nil {
		engine.RegisterBehavior("23023", b)
	}

	// Two Against the World: tech upgrade into play; ready both.
	engine.RegisterBehavior("23024", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, c := range p.Deck {
				def := c.Def()
				if def.Type == "upgrade" && def.HasTrait("tech") {
					picks = append(picks, engine.Choice{Label: engine.S(def.Name), Kind: engine.ChoiceCard, CardCode: def.Code}.
						Msgs(engine.UpgradeEnterPlay{Player: p.ID, Card: c},
							engine.ShufflePlayerDeck{Player: p.ID}))
				}
			}
			msgs := []engine.Message{engine.ReadyEntity{ID: p.ID}}
			if len(picks) == 0 {
				return append(msgs, engine.ShufflePlayerDeck{Player: p.ID})
			}
			return append(msgs, engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.putWhichTechUpgradeIntoPlay"), picks...)})
		},
	})

	// Basic resources.
	engine.RegisterBehavior("23025", &engine.Behavior{})
	engine.RegisterBehavior("23026", &engine.Behavior{})
	engine.RegisterBehavior("23027", &engine.Behavior{})

	// Equipment Malfunction obligation: exhaust-remove or dump ammo.
	engine.RegisterBehavior("23028", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var picks []engine.Choice
			if !p.Exhausted {
				picks = append(picks, engine.Choice{ID: "exhaust", Label: engine.Tf("c.exhaustJamesRhodesRemoveFromTheGame"), Kind: engine.ChoiceLabel}.
					Msgs(engine.ExhaustEntity{ID: p.ID},
						engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			picks = append(picks, engine.Choice{ID: "dump", Label: engine.Tf("c.removeAllAmmoCounters"), Kind: engine.ChoiceLabel}.
				Msgs(engine.ObligationResolve{Player: p.ID, Card: card}))
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.equipmentMalfunction"), picks...)}}
		},
	})

	// As One!: exhaust avenger+guardian → combined-ATK damage
	// (approximated to 6 flat).
	engine.RegisterBehavior("23032", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.asOneDeal6DamageToWhichEnemy"),
			func(g *engine.Game, e engine.Entity) (int, []engine.Message) { return 6, nil }),
	})

	// Vigilante Training: justice event recursion.
	engine.RegisterBehavior("23033", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 2}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustCounterShuffleAJusticeEventFromDiscard"), Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					for _, c := range p.Discard {
						def := c.Def()
						if def.Type == "event" && def.Aspect == "justice" {
							picks = append(picks, engine.Choice{Label: engine.S(def.Name), Kind: engine.ChoiceCard, CardCode: def.Code}.
								Msgs(engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}))
						}
					}
					if len(picks) == 0 {
						return nil
					}
					return append([]engine.Message{engine.AddEntityCounter{ID: self, N: -1}},
						engine.AskQuestion{Player: p.ID, Question: engine.Ask(
							engine.Tf("c.shuffleWhichJusticeEventIntoYourDeck"), picks...)})
				},
			}}
		},
	})

	// Stand Together: full prevention + counter damage (defense event).
	engine.RegisterBehavior("23034", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against, PreventAll: true},
				[]engine.Message{engine.DamageEntity{Target: against, Damage: 2, Source: p.ID}}, true
		},
	})

	// Sidearm: attach an ally +1 ATK.
	engine.RegisterBehavior("23035", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a != nil {
					picks = append(picks, engine.Choice{Label: engine.S(a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
						Msgs(engine.AttachUpgrade{ID: e.EID(), Target: a.ID, ATK: 1}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.attachSidearmToWhichAlly"), picks...)}}
		},
	})
}

func registerNemesis() {
	// Living Laser: Quickstrike printed; piercing skipped.
	engine.RegisterBehavior("23029", &engine.Behavior{})

	// Deadly Light Show: Hinder 1/hero; 1 damage to each identity on
	// defeat.
	engine.RegisterBehavior("23030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: len(g.Players), Source: e.EID()}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if d, ok := msg.(engine.SchemeDefeated); ok && d.Scheme == e.EID() {
				var msgs []engine.Message
				for _, p := range g.Players {
					msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: engine.EntityID("23030")})
				}
				return msgs
			}
			return nil
		},
	})

	// Laser Strike: discard an upgrade or surge.
	engine.RegisterBehavior("23031", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var picks []engine.Choice
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil {
					picks = append(picks, engine.Choice{Label: engine.S("Discard " + u.EDef().Name), Kind: engine.ChoiceCard, CardCode: u.Code}.
						Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
				}
			}
			if len(picks) == 0 {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.laserStrikeDiscardWhichUpgrade"), picks...)}}
		},
	})
}

// ---- helpers ----

func schemePicks(g *engine.Game, n int, pid engine.PlayerID) []engine.Choice {
	return cardutil.SchemeChoices(g, func(s engine.EntityID) []engine.Message {
		return []engine.Message{engine.ThwartScheme{Scheme: s, N: n, Source: pid}}
	})
}

func mainSchemeID(g *engine.Game) engine.EntityID {
	if g.MainScheme != nil {
		return g.MainScheme.ID
	}
	return ""
}

func deref(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
