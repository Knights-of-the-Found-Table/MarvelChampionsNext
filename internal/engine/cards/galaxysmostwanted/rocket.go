package galaxysmostwanted

import (
	"strconv"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerRocketCards installs Rocket Raccoon's signature deck (16019,
// 16030–16039), his aspect reprints (16040–16046), the shared Groot team-up
// cards (16020/16048), basic resources, obligation and nemesis set.
func registerRocketCards() {
	// 16019 Rocket Raccoon (ally): +3 ATK against minions.
	engine.RegisterBehavior("16019", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyAttackWindow)
			if !ok || w.Ally != e.EID() || g.Minions[w.Target] == nil {
				return nil
			}
			g.Logf("Rocket Raccoon: +3 ATK against %s", cardutil.EnemyLabel(g.Entity(w.Target)))
			return []engine.Message{engine.DamageEntity{Target: w.Target, Damage: 3, Source: e.EID()}}
		},
	})

	// 16030 I've Got a Plan: after a basic thwart, ready Rocket and +1 THW
	// for the phase.
	engine.RegisterBehavior("16030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), THW: 1}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			bt, ok := msg.(engine.BasicThwart)
			if !ok || bt.Player != e.EOwner() {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil || !p.IsHero() {
				return nil
			}
			g.Logf("I've Got a Plan: Rocket Raccoon readies")
			return []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
		},
	})

	// 16031 Reload: ready each tech upgrade.
	engine.RegisterBehavior("16031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.Exhausted && u.EDef().HasTrait("tech") {
					msgs = append(msgs, engine.ReadyEntity{ID: id})
				}
			}
			return msgs
		},
	})

	// 16032 Schadenfreude: until end of turn, heal 2 each time Rocket
	// damages an enemy (round-scoped flag; cleanup rides UsedThisRound).
	engine.RegisterBehavior("16032", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.UsedThisRound["16032"] = true
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Source != e.EOwner() {
				return nil
			}
			if !g.UsedThisRound["16032"] {
				return nil
			}
			tgt := g.Entity(d.Target)
			if tgt == nil || !(tgt.EID().Is(engine.KindVillain) || tgt.EID().Is(engine.KindMinion)) {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil || p.Damage == 0 {
				return nil
			}
			return []engine.Message{engine.HealEntity{Target: p.ID, N: 2}}
		},
	})

	// 16033 Salvage: after spending, tech upgrade from discard → top of
	// deck. Payment-time triggers are approximated to discard-time.
	engine.RegisterBehavior("16033", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DiscardCards)
			if !ok || d.Player != e.EOwner() {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for _, c := range d.Cards {
				if c.Code[:5] == "16033" {
					var picks []engine.Choice
					for _, dc := range p.Discard {
						if dc.Def().Type == "upgrade" && dc.Def().HasTrait("tech") {
							picks = append(picks, engine.Choice{
								Label: dc.Def().Name, Kind: engine.ChoiceCard, CardCode: dc.Code,
							}.Msgs(engine.DiscardToBottom{Player: p.ID, CardID: dc.ID}))
						}
					}
					if len(picks) > 0 {
						return []engine.Message{engine.AskQuestion{Player: p.ID,
							Question: engine.Ask("Salvage: put which tech upgrade on top of your deck?", picks...)}}
					}
					return nil
				}
			}
			return nil
		},
	})

	// 16034 Battery Pack: 2 charges; move one to another tech upgrade.
	engine.RegisterBehavior("16034", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 2}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			if u == nil || p == nil || u.Counters <= 0 {
				return nil
			}
			var targets []engine.Choice
			for _, id := range p.Upgrades {
				if o := g.Upgrades[id]; o != nil && o.ID != u.ID && o.EDef().HasTrait("tech") {
					targets = append(targets, engine.Choice{
						Label: o.EDef().Name, Kind: engine.ChoiceTarget, SourceID: o.ID,
					}.Msgs(engine.AddEntityCounter{ID: o.ID, N: 1}))
				}
			}
			if len(targets) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust Battery Pack → move 1 charge to a tech upgrade", Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if o := g.Upgrades[self]; o != nil {
						o.Counters--
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask("Battery Pack: add 1 charge to which tech upgrade?", targets...)}}
				},
			}}
		},
	})

	// 16035 Cybernetic Skeleton: +3 max HP; +1 ATK in hero form.
	engine.RegisterBehavior("16035", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p != nil {
				p.MaxHP += 3
				g.Logf("%s gets +3 hit points", p.Name)
			}
			return nil
		},
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			if p.IsHero() {
				return engine.StatBonus{ATK: 1}
			}
			return engine.StatBonus{}
		},
	})

	// 16036 Particle Cannon: 2 charges; 4 damage overkill ranged.
	engine.RegisterBehavior("16036", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 2}}
		},
		Abilities: chargeCannonAbility("16036", "Particle Cannon: deal 4 damage to which enemy?", 4),
	})

	// 16037 Rocket Launcher: 2 charges; 2 damage to the villain and each
	// minion engaged with a chosen player.
	engine.RegisterBehavior("16037", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 2}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u := g.Upgrades[e.EID()]
			if u == nil || u.Counters <= 0 || u.Exhausted {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust Rocket Launcher + 1 charge → 2 damage to the villain and a player's minions", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if o := g.Upgrades[self]; o != nil {
						o.Counters--
					}
					owner := g.Player(e.EOwner())
					var picks []engine.Choice
					for _, p := range g.Players {
						var msgs []engine.Message
						for vid := range g.Villains {
							msgs = append(msgs, engine.DamageEntity{Target: vid, Damage: 2, Source: self})
						}
						for _, mid := range cardutil.SortedIDs(g.Minions) {
							if m := g.Minions[mid]; m != nil && m.EngagedWith == p.ID {
								msgs = append(msgs, engine.DamageEntity{Target: mid, Damage: 2, Source: self})
							}
						}
						picks = append(picks, engine.Choice{
							Label: p.Name + "'s engagement", Kind: engine.ChoiceLabel,
						}.Msgs(msgs...))
					}
					return []engine.Message{engine.AskQuestion{Player: owner.ID,
						Question: engine.Ask("Rocket Launcher: hit the villain and which player's minions?", picks...)}}
				},
			}}
		},
	})

	// 16038 Rocket's Pistol: 3 charges; 2 damage.
	engine.RegisterBehavior("16038", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		Abilities: chargeCannonAbility("16038", "Rocket's Pistol: deal 2 damage to which enemy?", 2),
	})

	// 16039 Thruster Boots: +1 THW and aerial in hero form.
	engine.RegisterBehavior("16039", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "aerial"}}
		},
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			if p.IsHero() {
				return engine.StatBonus{THW: 1}
			}
			return engine.StatBonus{}
		},
	})

	registerRocketBasics()
	registerRocketObligation()
	registerRocketNemesis()
}

// chargeCannonAbility builds the shared "exhaust + 1 charge → N damage"
// hero ability used by Particle Cannon and Rocket's Pistol.
func chargeCannonAbility(code, prompt string, dmg int) func(g *engine.Game, e engine.Entity) []engine.Ability {
	return func(g *engine.Game, e engine.Entity) []engine.Ability {
		u := g.Upgrades[e.EID()]
		if u == nil || u.Counters <= 0 {
			return nil
		}
		return []engine.Ability{{
			Label:    "Exhaust + 1 charge → deal " + itoa(dmg) + " damage (overkill, ranged)",
			Type:     engine.AbilityAction,
			Exhaust:  true,
			HeroOnly: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				if o := g.Upgrades[self]; o != nil {
					o.Counters--
				}
				return cardutil.ChooseEnemy(prompt,
					func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return dmg, nil })(
					g, g.Entity(self))
			},
		}}
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

// registerRocketBasics installs Rocket's aspect reprints (16040–16052) and
// the shared team-up cards.
func registerRocketBasics() {
	// 16020/16048 Flora and Fauna: 2 growth counters + ready Groot, or 2
	// charges + ready a Rocket upgrade. The Groot half is the only branch
	// for Groot players, so its counters are added eagerly.
	floraFauna := &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			if p.HeroCode[:5] == "16001" {
				addGrowth(p, 2)
				picks = append(picks, engine.Choice{
					ID: "groot", Label: "Place 2 growth counters on Groot and ready him", Kind: engine.ChoiceLabel,
				}.Msgs(engine.ReadyEntity{ID: p.ID}))
			} else {
				for _, id := range p.Upgrades {
					if u := g.Upgrades[id]; u != nil {
						switch u.Code[:5] {
						case "16034", "16036", "16037", "16038":
							picks = append(picks, engine.Choice{
								Label: "2 charges + ready " + u.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id,
							}.Msgs(engine.AddEntityCounter{ID: id, N: 2}, engine.ReadyEntity{ID: id}))
						}
					}
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Flora and Fauna: which mode?", picks...)}}
		},
	}
	engine.RegisterBehavior("16020", floraFauna)
	engine.RegisterBehavior("16048", floraFauna)

	// 16040 Bug: after your hero's basic attack, heal 1.
	engine.RegisterBehavior("16040", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ba, ok := msg.(engine.BasicAttack)
			if !ok || ba.Player != e.EOwner() {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil || a.Damage == 0 {
				return nil
			}
			return []engine.Message{engine.HealEntity{Target: e.EID(), N: 1}}
		},
	})

	// 16041 Chase Them Down: alias core 01052.
	if b := engine.LookupBehavior("01052"); b != nil {
		engine.RegisterBehavior("16041", b)
	}

	// 16042 Into the Fray: alias Wasp 13013.
	if b := engine.LookupBehavior("13013"); b != nil {
		engine.RegisterBehavior("16042", b)
	}

	// 16043 Looking for Trouble: reveal a minion engaged with you, remove
	// 3 threat from the main scheme.
	engine.RegisterBehavior("16043", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := discardUntilMinionEngaged(g, e.EOwner())
			if g.MainScheme != nil {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 3, Source: e.EID()})
			}
			return msgs
		},
	})

	// 16044 Relentless Assault: alias core 01053.
	if b := engine.LookupBehavior("01053"); b != nil {
		engine.RegisterBehavior("16044", b)
	}

	// 16045 Follow Through: excess damage +1 — no excess window; the
	// damage rider is approximated as +1 whenever the hero's attack
	// targets a minion (where excess is common).
	engine.RegisterBehavior("16045", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ba, ok := msg.(engine.BasicAttack)
			if !ok || ba.Player != e.EOwner() || g.Minions[ba.Target] == nil {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: ba.Target, Damage: 1, Source: e.EID()}}
		},
	})

	// 16046 Hand Cannon: +2 ATK for basic attacks, charge-powered.
	engine.RegisterBehavior("16046", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ba, ok := msg.(engine.BasicAttack)
			if !ok || ba.Player != e.EOwner() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			if u == nil || u.Exhausted || p == nil || u.Counters <= 0 || !p.IsHero() {
				return nil
			}
			u.Exhausted = true
			u.Counters--
			g.Logf("Hand Cannon: +2 damage for this attack")
			return []engine.Message{engine.DamageEntity{Target: ba.Target, Damage: 2, Source: e.EID()}}
		},
	})

	// 16047 Groot (ally): after defending, heal 2.
	engine.RegisterBehavior("16047", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			if !ok || w.Defender != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil || a.Damage == 0 {
				return nil
			}
			return []engine.Message{engine.HealEntity{Target: e.EID(), N: 2}}
		},
	})

	// 16049–16051 basic resources; 16052 Booster Boots below.
	engine.RegisterBehavior("16049", &engine.Behavior{})
	engine.RegisterBehavior("16050", &engine.Behavior{})
	engine.RegisterBehavior("16051", &engine.Behavior{})

	// 16052 Booster Boots: exhaust + discard top card → prevent 1 damage
	// from an attack (auto-used on any damage, Energy Barrier style).
	engine.RegisterBehavior("16052", &engine.Behavior{
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			if u.Exhausted {
				return 0, 0
			}
			u.Exhausted = true
			g.Push(engine.MillPlayerDeck{Player: p.ID, N: 1})
			g.Logf("Booster Boots: 1 damage prevented")
			return 1, 0
		},
	})
}

// registerRocketObligation installs Crisis on Halfworld (16053).
func registerRocketObligation() {
	engine.RegisterBehavior("16053", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			exhaust := engine.Choice{
				ID: "exhaust", Label: "Exhaust your alter-ego → remove Crisis on Halfworld from the game", Kind: engine.ChoiceLabel,
			}.Msgs(
				engine.ExhaustEntity{ID: p.ID},
				engine.ObligationResolve{Player: p.ID, Card: card, Remove: true})
			var strip []engine.Choice
			strip = append(strip, exhaust)
			// Discard the highest-cost upgrade.
			best, bestCost := -1, -1
			for i, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil {
					c := cardutil.Cost(u.EDef())
					if c > bestCost {
						best, bestCost = i, c
					}
				}
			}
			if best >= 0 {
				strip = append(strip, engine.Choice{
					ID: "strip", Label: "Discard your highest-cost upgrade", Kind: engine.ChoiceLabel,
				}.Msgs(engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[best]},
					engine.ObligationResolve{Player: p.ID, Card: card}))
			} else {
				strip = append(strip, engine.Choice{
					ID: "surge", Label: "No upgrades to discard (surge)", Kind: engine.ChoiceLabel, Disabled: true,
				})
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Crisis on Halfworld: choose an option", strip...)}}
		},
	})
}

// registerRocketNemesis installs Blackjack O'Hare and his bazooka
// (16055–16056). Quickstrike and Villainous are engine-driven keywords.
func registerRocketNemesis() {
	// 16055 Blackjack O'Hare: Quickstrike + Villainous.
	engine.RegisterBehavior("16055", &engine.Behavior{})

	// 16056 Blackjack's Bazooka: attach to Blackjack O'Hare (or the
	// villain); hero action to buy it off.
	engine.RegisterBehavior("16056", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			// Prefer the Blackjack O'Hare minion.
			for id := range g.Minions {
				if m := g.Minions[id]; m != nil && m.Code[:5] == "16055" {
					t.Target = id
					g.Logf("Blackjack's Bazooka attaches to Blackjack O'Hare")
					return nil
				}
			}
			for id := range g.Villains {
				t.Target = id
				g.Logf("Blackjack's Bazooka attaches to the villain")
				return nil
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Spend 3 mental resources → discard Blackjack's Bazooka", Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
				},
			}}
		},
	})
}

// discardUntilMinionEngaged mills the encounter deck until a minion shows
// up and reveals it engaged with the given player.
func discardUntilMinionEngaged(g *engine.Game, pid engine.PlayerID) []engine.Message {
	for len(g.EncounterDeck) > 0 {
		c := g.EncounterDeck[0]
		g.EncounterDeck = g.EncounterDeck[1:]
		if c.Def().Type == "minion" {
			return []engine.Message{engine.RevealEncounterCard{Player: pid, Card: c}}
		}
		g.EncounterDiscard = append(g.EncounterDiscard, c)
	}
	return nil
}
