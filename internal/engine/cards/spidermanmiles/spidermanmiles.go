// Package spidermanmiles registers the Miles Morales Spider-Man hero
// (27030) from the Sinister Motives box: the two identity Specials
// (Venom Blast / Spider Camouflage), the signature cards, the Keeping
// Secrets obligation and the Prowler nemesis set.
package spidermanmiles

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerMiles()
	registerMilesSignatures()
	registerMilesObligation()
	registerMilesNemesis()
}

// venomBlast resolves Miles's Venom Blast Special: deal 2 damage to an
// enemy and stun it. Shared by the identity ability and card riders.
func venomBlast(g *engine.Game, p *engine.Player) []engine.Message {
	if p == nil || !p.IsHero() {
		return nil
	}
	choices := cardutil.EnemyChoices(g, 2, p.ID, func(id engine.EntityID) []engine.Message {
		return []engine.Message{
			engine.DamageEntity{Target: id, Damage: 2, Source: p.ID},
			engine.StunEntity{Target: id},
		}
	})
	if len(choices) == 0 {
		return nil
	}
	g.TLogf("c.venomBlast2DamageAndStun")
	return []engine.Message{engine.AskQuestion{
		Player:   p.ID,
		Question: engine.Ask(engine.Tf("c.venomBlastDeal2DamageToAndStunAnEnemy"), choices...),
	}}
}

// spiderCamouflage resolves Miles's Spider Camouflage Special: give
// Spider-Man a tough status card and confuse an enemy.
func spiderCamouflage(g *engine.Game, p *engine.Player) []engine.Message {
	if p == nil || !p.IsHero() {
		return nil
	}
	msgs := []engine.Message{engine.ToughEntity{Target: p.ID}}
	choices := cardutil.EnemyChoices(g, 0, p.ID, func(id engine.EntityID) []engine.Message {
		return []engine.Message{engine.ConfuseEntity{Target: id}}
	})
	if len(choices) > 0 {
		msgs = append(msgs, engine.AskQuestion{
			Player:   p.ID,
			Question: engine.Ask(engine.Tf("c.spiderCamouflageConfuseAnEnemy"), choices...),
		})
	}
	g.TLogf("c.spiderCamouflageGainsTough", p.Name)
	return msgs
}

// basicPowerUsed reports whether the message announces Miles using one of
// his basic powers (attack / thwart / defend / recover).
func basicPowerUsed(p *engine.Player, msg engine.Message) bool {
	switch m := msg.(type) {
	case engine.BasicAttack:
		return m.Player == p.ID
	case engine.BasicThwart:
		return m.Player == p.ID
	case engine.BasicRecover:
		return m.Player == p.ID
	case engine.WindowDefended:
		return m.Defender == p.ID
	}
	return false
}

// registerMiles installs the Spider-Man / Miles Morales identity
// (27030a/b). The printed Specials are resolved by card effects and may
// each be used once per round; here they are exposed as once-per-round
// hero actions (approximation) so they stay directly usable.
func registerMiles() {
	engine.RegisterBehavior("27030", &engine.Behavior{
		// Miles Morales — Response: after you change to this form,
		// shuffle 1 Spider-Man card from your discard pile into your deck.
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			m, ok := msg.(engine.ChangeForm)
			if !ok || m.Player != p.ID || p.IsHero() {
				return nil
			}
			choices := []engine.Choice{cardutil.Skip()}
			for _, c := range p.Discard {
				if c.Def().CardSet == "spider_man_morales" {
					choices = append(choices, engine.Choice{
						Label: engine.S(c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}))
				}
			}
			if len(choices) == 1 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.milesMoralesShuffleASpiderManCardFromYourDiscardPileIntoYour"), choices...),
			}}
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{
				{
					Label: engine.Tf("c.venomBlastDeal2DamageToAndStunAnEnemy"), Type: engine.AbilityAction,
					HeroOnly: true, OncePerRound: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						return venomBlast(g, g.Player(self))
					},
				},
				{
					Label: engine.Tf("c.spiderCamouflageGainToughAndConfuseAnEnemy"), Type: engine.AbilityAction,
					HeroOnly: true, OncePerRound: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						return spiderCamouflage(g, g.Player(self))
					},
				},
			}
		},
	})
}

func registerMilesSignatures() {
	// 27031 Arachnobatics: Hero Action (attack) — deal 2 damage to an
	// enemy, +3 if it is stunned, +3 if it is confused.
	engine.RegisterBehavior("27031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				dmg := 2
				switch t := enemy.(type) {
				case *engine.Villain:
					if t.Stunned {
						dmg += 3
					}
					if t.Confused {
						dmg += 3
					}
				case *engine.Minion:
					if t.Stunned {
						dmg += 3
					}
					if t.Confused {
						dmg += 3
					}
				}
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: enemy.ECode(),
				}.Msgs(engine.DamageEntity{Target: id, Damage: dmg, Source: pid}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.arachnobaticsDealDamageToAnEnemy"), choices...),
			}}
		},
	})

	// 27032 Double Life: Action — change your form; if paid with a
	// [physical] resource, ready your identity. (Approximation: the
	// max-1-per-round limit is not enforced for hand events.)
	engine.RegisterBehavior("27032", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			msgs := []engine.Message{engine.ChangeForm{Player: pid}}
			if ec, ok := e.(*engine.EventCard); ok && ec.Paid.PaidIcon("physical") {
				msgs = append(msgs, engine.ReadyEntity{ID: pid})
			}
			return msgs
		},
	})

	// 27033 Swing In: Hero Action (thwart) — remove 4 threat from a
	// scheme; if paid with a [mental] resource, resolve Spider Camouflage.
	engine.RegisterBehavior("27033", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			paidMental := false
			if ec, ok := e.(*engine.EventCard); ok {
				paidMental = ec.Paid.PaidIcon("mental")
			}
			choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
				msgs := []engine.Message{engine.ThwartScheme{Scheme: id, N: 4, Source: pid}}
				if paidMental {
					msgs = append(msgs, spiderCamouflage(g, p)...)
				}
				return msgs
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.swingInRemove4ThreatFromAScheme"), choices...),
			}}
		},
	})

	// 27034 Web-Shot: Hero Action (attack) — deal 4 damage to an enemy;
	// if paid with a [energy] resource, resolve Venom Blast.
	engine.RegisterBehavior("27034", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			paidEnergy := false
			if ec, ok := e.(*engine.EventCard); ok {
				paidEnergy = ec.Paid.PaidIcon("energy")
			}
			choices := cardutil.EnemyChoices(g, 4, pid, func(id engine.EntityID) []engine.Message {
				msgs := []engine.Message{engine.DamageEntity{Target: id, Damage: 4, Source: pid}}
				if paidEnergy {
					msgs = append(msgs, venomBlast(g, p)...)
				}
				return msgs
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.webShotDeal4DamageToAnEnemy"), choices...),
			}}
		},
	})

	// 27035 Ganke Lee: Action — exhaust → draw 1 card; in hero form,
	// choose and discard 1 card from your hand.
	engine.RegisterBehavior("27035", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.gankeLeeDraw1CardHeroFormDiscard1"), Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(e.EOwner())
					if s == nil || p == nil {
						return nil
					}
					msgs := []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
					if p.IsHero() && len(p.Hand) > 0 {
						var choices []engine.Choice
						for _, c := range p.Hand {
							choices = append(choices, engine.Choice{
								Label: engine.S(c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
							}.Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}))
						}
						msgs = append(msgs, engine.AskQuestion{
							Player:   p.ID,
							Question: engine.Ask(engine.Tf("c.gankeLeeDiscard1Card"), choices...),
						})
					}
					return msgs
				},
			}}
		},
	})

	// 27036 Jefferson Davis: Alter-Ego Action — exhaust → remove 1 threat
	// from the scheme with the least threat.
	engine.RegisterBehavior("27036", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.jeffersonDavisRemove1ThreatFromTheLeastThreatenedScheme"), Type: engine.AbilityAction,
				AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var target engine.EntityID
					least := -1
					for _, id := range g.Schemes() {
						t := 0
						switch s := g.Entity(id).(type) {
						case *engine.MainScheme:
							t = s.Threat
						case *engine.SideScheme:
							t = s.Threat
						}
						if least < 0 || t < least {
							least, target = t, id
						}
					}
					if target == "" {
						return nil
					}
					return []engine.Message{engine.ThwartScheme{Scheme: target, N: 1, Source: e.EID()}}
				},
			}}
		},
	})

	// 27037 Power Within: Hero Response — after your hero uses a basic
	// power, discard Power Within → resolve Venom Blast.
	engine.RegisterBehavior("27037", &engine.Behavior{
		React: powerWithinReact("27037", venomBlast, "Power Within — discard to resolve Venom Blast?"),
	})

	// 27038 Defense Mechanism: Hero Response — after your hero uses a
	// basic power, discard Defense Mechanism → resolve Spider Camouflage.
	engine.RegisterBehavior("27038", &engine.Behavior{
		React: powerWithinReact("27038", spiderCamouflage, "Defense Mechanism — discard to resolve Spider Camouflage?"),
	})

	// 27039 Web-Shooter: uses (3 web counters); Hero Resource — exhaust
	// and remove 1 counter → wild.
	engine.RegisterBehavior("27039", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if u, ok := e.(*engine.Upgrade); ok {
				u.Counters = 3
			}
			return nil
		},
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true, UsesCounters: true},
	})

	// 27040 Monica Chang: Response — after she enters play, search your
	// deck, hand and discard pile for Surveillance Team and put it into
	// play; place 1 snoop counter on each Surveillance Team you control.
	engine.RegisterBehavior("27040", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			found := false
			for _, zone := range []*engine.CardList{&p.Deck, &p.Hand, &p.Discard} {
				if found {
					break
				}
				for _, c := range *zone {
					if c.Code != "27045" {
						continue
					}
					zone.Remove(c.ID)
					s := &engine.Support{ID: g.NextEntityID(engine.KindSupport), Code: c.Code, Owner: p.ID}
					g.Supports[s.ID] = s
					p.Supports = append(p.Supports, s.ID)
					g.TLogf("c.monicaChangPutsIntoPlay", s)
					if b := engine.LookupBehavior("27045"); b != nil && b.OnPlay != nil {
						g.Push(b.OnPlay(g, s)...)
					}
					found = true
					break
				}
			}
			var msgs []engine.Message
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil && s.Code == "27045" {
					msgs = append(msgs, engine.AddEntityCounter{ID: id, N: 1})
				}
			}
			return msgs
		},
	})

	// 27045 Surveillance Team (Justice pool card, Monica Chang's search
	// target): uses (3 snoop counters); Action — exhaust + remove 1
	// counter → remove 1 threat from a scheme.
	engine.RegisterBehavior("27045", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if s, ok := e.(*engine.Support); ok {
				s.Counters = 3
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Exhausted || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.surveillanceTeamRemove1SnoopCounterRemove1Threat"), Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					if s == nil || s.Counters <= 0 {
						return nil
					}
					choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
						return []engine.Message{
							engine.AddEntityCounter{ID: s.ID, N: -1},
							engine.ThwartScheme{Scheme: id, N: 1, Source: s.ID},
						}
					})
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   s.Owner,
						Question: engine.Ask(engine.Tf("c.surveillanceTeamRemove1ThreatFromAScheme"), choices...),
					}}
				},
			}}
		},
	})
}

// powerWithinReact builds the shared "after your hero uses a basic power,
// discard this upgrade to resolve a Special" response.
func powerWithinReact(code string, special func(g *engine.Game, p *engine.Player) []engine.Message, prompt string) func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
	return func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		u := g.Upgrades[e.EID()]
		if u == nil {
			return nil
		}
		p := g.Player(u.Owner)
		if p == nil || !p.IsHero() || !basicPowerUsed(p, msg) {
			return nil
		}
		skip := engine.Choice{ID: "skip", Label: engine.Tf("c.skip"), Kind: engine.ChoicePass}
		confirm := engine.Choice{
			ID: "resolve", Label: engine.S(prompt), Kind: engine.ChoiceLabel,
		}.Msgs(append([]engine.Message{engine.DiscardControlled{Player: p.ID, ID: u.ID}}, special(g, p)...)...)
		return []engine.Message{engine.AskQuestion{
			Player:   p.ID,
			Question: engine.Ask(engine.S(prompt), confirm, skip),
		}}
	}
}

// registerMilesObligation installs Keeping Secrets (27056): exhaust Miles
// to remove it, or discard Ganke Lee and Jefferson Davis from play (surge
// when neither is discarded).
func registerMilesObligation() {
	engine.RegisterBehavior("27056", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var discardMsgs []engine.Message
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil && (s.Code == "27035" || s.Code == "27036") {
					discardMsgs = append(discardMsgs, engine.DiscardControlled{Player: p.ID, ID: id})
				}
			}
			if len(discardMsgs) == 0 {
				// Surge approximation: reveal another encounter card.
				discardMsgs = append(discardMsgs, engine.RevealNextEncounter{Player: p.ID})
			}
			discardMsgs = append(discardMsgs, engine.ObligationResolve{Player: p.ID, Card: card})
			exhaust := engine.Choice{
				ID: "exhaust", Label: engine.Tf("c.exhaustMilesMoralesRemoveKeepingSecretsFromTheGame"), Kind: engine.ChoiceLabel,
			}.Msgs(
				engine.ExhaustEntity{ID: p.ID},
				engine.ObligationResolve{Player: p.ID, Card: card, Remove: true},
			)
			discard := engine.Choice{
				ID: "discard", Label: engine.Tf("c.discardGankeLeeAndJeffersonDavisFromPlay"), Kind: engine.ChoiceLabel,
			}.Msgs(discardMsgs...)
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.keepingSecretsChoose"), exhaust, discard),
			}}
		},
	})
}

// registerMilesNemesis installs the Miles nemesis set (Tracking Prey,
// Prowler, Razor Claws, Slice and Dice).
func registerMilesNemesis() {
	// 27057 Tracking Prey: When Revealed — if you are in alter-ego form,
	// place 1 acceleration token here. (Approximation: the revealing
	// player is not exposed to OnPlay; the first player stands in.)
	engine.RegisterBehavior("27057", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil || p.IsHero() {
				return nil
			}
			return []engine.Message{engine.AddAccelerationToken{Scheme: e.EID()}}
		},
	})

	// 27058 Prowler: Stalwart (data layer). When Revealed — if you are in
	// alter-ego form, give Prowler a tough status card. (Same first-player
	// approximation.)
	engine.RegisterBehavior("27058", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil || p.IsHero() {
				return nil
			}
			return []engine.Message{engine.ToughEntity{Target: e.EID()}}
		},
	})

	// 27059 Razor Claws: attach to the minion with the highest printed
	// hit points (surge if none); attached minion's attacks gain piercing
	// and it gets +2 ATK. (Approximation: piercing is not represented by
	// the combat engine; only +2 ATK is modeled.)
	engine.RegisterBehavior("27059", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best *engine.Minion
			bestHP := -1
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil {
					continue
				}
				hp := 0
				if mn.EDef().HP != nil {
					hp = *mn.EDef().HP
				}
				if hp > bestHP {
					best, bestHP = mn, hp
				}
			}
			if best == nil {
				g.Delete(t.ID)
				return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
			}
			t.Target = best.ID
			best.Attachments = append(best.Attachments, t.ID)
			g.TLogf("c.razorClawsAttachedTo", best)
			return []engine.Message{engine.BoostEnemyAttack{Enemy: best.ID, N: 2}}
		},
	})

	// 27060 Slice and Dice: Prowler attacks the player with the fewest
	// remaining hit points (even in alter-ego). (Approximation: the surge
	// rider is omitted.)
	engine.RegisterBehavior("27060", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var prowler engine.EntityID
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.Code == "27058" {
					prowler = id
					break
				}
			}
			if prowler == "" {
				return nil
			}
			var victim *engine.Player
			for _, pl := range g.Players {
				if pl.KOed {
					continue
				}
				if victim == nil || pl.HP() < victim.HP() {
					victim = pl
				}
			}
			if victim == nil {
				return nil
			}
			g.TLogf("c.sliceAndDiceProwlerAttacks", victim.Name)
			return []engine.Message{engine.MinionActivates{MinionID: prowler, Player: victim.ID}}
		},
	})
}
