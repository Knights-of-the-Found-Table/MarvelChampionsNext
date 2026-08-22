package core

// complete_player.go implements the remaining Core Set player cards.
// Approximations are noted inline where an exact rules window does not
// exist in the engine yet.

import (
	"fmt"
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func registerRemainingPlayerCards() {
	// ---- Spider-Man ----

	// Black Cat: on play, discard top 2; printed mental resources go to hand.
	engine.RegisterBehavior("01002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picked, milled engine.CardList
			for i := 0; i < 2 && len(p.Deck) > 0; i++ {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				if len(c.Def().Resources) > 0 && c.Def().Resources[0] == "mental" {
					picked = append(picked, c)
				} else {
					milled = append(milled, c)
				}
			}
			p.Hand = append(p.Hand, picked...)
			p.Discard = append(p.Discard, milled...)
			g.Logf("Black Cat: %s adds %d card(s) to hand", p.Name, len(picked))
			return nil
		},
	})

	// Backflip: defense interrupt preventing all damage.
	engine.RegisterBehavior("01003", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against, PreventAll: true}, nil, true
		},
	})

	// Enhanced Spider-Sense: cancel a revealed treachery's effects.
	engine.RegisterBehavior("01004", &engine.Behavior{
		TreacheryInterrupt: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			return []engine.Message{} // card already in the encounter discard; effects cancelled
		},
	})

	// Spider-Tracer: attach to a minion; when it is defeated remove 3 threat.
	engine.RegisterBehavior("01007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if mn := g.Minions[id]; mn != nil {
					picks = append(picks, engine.Choice{
						Label: cardutil.EnemyLabel(mn), Kind: engine.ChoiceTarget, SourceID: mn.ID, CardCode: mn.Code,
					}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: mn.ID}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   e.EOwner(),
				Question: engine.Ask("Attach Spider-Tracer to which minion?", picks...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			if d, ok := msg.(engine.MinionDefeated); ok && d.MinionID == u.AttachTo {
				return []engine.Message{
					engine.DiscardControlled{Player: u.Owner, ID: u.ID},
					engine.AskQuestion{Player: u.Owner, Question: engine.Ask("Spider-Tracer: remove 3 threat from which scheme?",
						schemePickChoices(g, 3, u.Owner)...),
					},
				}
			}
			return nil
		},
	})

	// Web-Shooter: 3 web counters, exhaust + counter → wild resource.
	engine.RegisterBehavior("01008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true, UsesCounters: true},
	})

	// Webbed Up: attach to an enemy; replaces its next attack with a stun.
	// Approximation: the stun placed here is consumed cancelling the very
	// activation it interrupts; officially it persists for the next attack.
	engine.RegisterBehavior("01009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				if enemy == nil {
					continue
				}
				picks = append(picks, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode(),
				}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id}))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   e.EOwner(),
				Question: engine.Ask("Attach Webbed Up to which enemy?", picks...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.VillainActivates:
				if m.VillainID == u.AttachTo {
					return []engine.Message{
						engine.DiscardControlled{Player: u.Owner, ID: u.ID},
						engine.StunEntity{Target: u.AttachTo},
					}
				}
			case engine.MinionActivates:
				if m.MinionID == u.AttachTo {
					return []engine.Message{
						engine.DiscardControlled{Player: u.Owner, ID: u.ID},
						engine.StunEntity{Target: u.AttachTo},
					}
				}
			}
			return nil
		},
	})

	// Spider-Woman: on play, confuse the villain.
	engine.RegisterBehavior("01011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if g.Villains[id] != nil {
					return []engine.Message{engine.ConfuseEntity{Target: id}}
				}
			}
			return nil
		},
	})

	// Crisis Interdiction: remove 2 threat (+2 elsewhere if Aerial).
	engine.RegisterBehavior("01012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			msgs := []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Crisis Interdiction: remove 2 threat from which scheme?",
					schemePickChoices(g, 2, p.ID)...)}}
			if g.EntityHasTrait(p.ID, "aerial") {
				msgs = append(msgs, engine.AskQuestion{Player: p.ID,
					Question: engine.Ask("Aerial: remove 2 threat from which other scheme?",
						schemePickChoices(g, 2, p.ID)...)})
			}
			return msgs
		},
	})

	// Photonic Blast: 5 damage; draw 1 if paid with energy.
	engine.RegisterBehavior("01013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := cardutil.ChooseEnemy("Photonic Blast: choose an enemy", func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) {
				return 5, nil
			})(g, e)
			if ec, ok := e.(*engine.EventCard); ok {
				for _, ic := range ec.Paid.Icons {
					if ic == "energy" {
						msgs = append(msgs, engine.DrawCards{Player: e.EOwner(), N: 1})
						break
					}
				}
			}
			return msgs
		},
	})

	// Alpha Flight Station: exhaust + discard 1 → draw 1 (2 for Carol).
	engine.RegisterBehavior("01015", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Alpha Flight Station + discard 1 card → draw", Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.Entity(self).EOwner())
					if p == nil || len(p.Hand) == 0 {
						return nil
					}
					n := 1
					if p.HeroCode == "01010a" {
						n = 2
					}
					var picks []engine.Choice
					for _, c := range p.Hand {
						def := c.Def()
						picks = append(picks, engine.Choice{
							Label: def.Name, Kind: engine.ChoiceCard, CardCode: def.Code,
						}.Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}},
							engine.DrawCards{Player: p.ID, N: n}))
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask(fmt.Sprintf("Discard 1 card → draw %d", n), picks...)}}
				},
			}}
		},
	})

	// Captain Marvel's Helmet: +1 DEF (+2 with Aerial).
	engine.RegisterBehavior("01016", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			b := engine.StatBonus{DEF: 1}
			// Trait check needs the game; Aerial is granted by Cosmic Flight
			// via ExtraTraits.
			if hasTrait(p, "aerial") {
				b.DEF = 2
			}
			return b
		},
	})

	// Cosmic Flight: Aerial trait + discard to prevent 3 damage.
	engine.RegisterBehavior("01017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "aerial"}}
		},
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			g.Delete(u.ID)
			p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			g.Logf("%s discards Cosmic Flight to prevent 3 damage", p.Name)
			return min(3, n), 0
		},
	})

	// Energy Channel: accumulate energy counters, spend all for 2 damage
	// each. Approximation: counters are added one per energy spent (the
	// official X-at-once spend is split into single-energy actions).
	engine.RegisterBehavior("01018", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{
				{
					Label: "Spend 1 [energy] → put an energy counter here", Type: engine.AbilityAction,
					Cost: 1, CostIcons: "energy:1",
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						return []engine.Message{engine.AddEntityCounter{ID: self, N: 1}}
					},
				},
				{
					Label: "Discard Energy Channel → deal 2 damage per counter", Type: engine.AbilityAction, HeroOnly: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						u := g.Upgrades[self]
						if u == nil || u.Counters <= 0 {
							return nil
						}
						dmg := min(10, u.Counters*2)
						owner := u.Owner
						return append([]engine.Message{engine.DiscardControlled{Player: owner, ID: self}},
							cardutil.ChooseEnemy(fmt.Sprintf("Energy Channel: %d damage to which enemy?", dmg),
								func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return dmg, nil })(g, g.Entity(self))...)
					},
				},
			}
		},
	})

	// Hellcat: return to hand.
	engine.RegisterBehavior("01020", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Return Hellcat to your hand", Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.ReturnControlled{Player: g.Entity(self).EOwner(), ID: self}}
				},
			}}
		},
	})

	// Gamma Slam: X damage = sustained damage (max 15).
	engine.RegisterBehavior("01021", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Gamma Slam: choose an enemy", func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			p := g.Player(e.EOwner())
			if p == nil {
				return 0, nil
			}
			return min(15, p.Damage), nil
		}),
	})

	// Ground Stomp: 1 damage to each enemy.
	engine.RegisterBehavior("01022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return damageEachEnemyMsgs(g, 1, e.EOwner())
		},
	})

	// Legal Practice: discard up to 5 → remove 1 threat per card.
	engine.RegisterBehavior("01023", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Hand) == 0 {
				return nil
			}
			return []engine.Message{
				engine.AskQuestion{Player: p.ID, Question: engine.Ask("Legal Practice: remove 1 threat from which scheme per discarded card?",
					schemePickChoices(g, 1, p.ID)...)},
			}
		},
	})

	// One-Two Punch: after a basic attack, ready She-Hulk.
	// Approximation: offered as a play-anytime action that readies her.
	engine.RegisterBehavior("01024", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
		},
	})

	// Split Personality: change form (bypassing the once-per-turn limit)
	// and draw up to printed hand size.
	engine.RegisterBehavior("01025", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			if p.IsHero() {
				p.Side = engine.SideAlterEgo
			} else {
				p.Side = engine.SideHero
			}
			size := 5
			if !p.IsHero() {
				size = 6
			}
			n := max(0, size-len(p.Hand))
			return []engine.Message{engine.DrawCards{Player: p.ID, N: n}}
		},
	})

	// Superhuman Law Division: exhaust + mental → remove 2 threat.
	engine.RegisterBehavior("01026", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust + spend [mental] → remove 2 threat", Type: engine.AbilityAction,
				Exhaust: true, Cost: 1, CostIcons: "mental:1", AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					if s == nil {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: s.Owner,
						Question: engine.Ask("Remove 2 threat from which scheme?",
							schemePickChoices(g, 2, s.Owner)...)}}
				},
			}}
		},
	})

	// Focused Rage: exhaust + take 1 damage → draw 1.
	engine.RegisterBehavior("01027", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Focused Rage + take 1 damage → draw 1 card", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					return []engine.Message{
						engine.DamageEntity{Target: u.Owner, Damage: 1, Source: self},
						engine.DrawCards{Player: u.Owner, N: 1},
					}
				},
			}}
		},
	})

	// Superhuman Strength: +2 ATK; after She-Hulk attacks, discard to stun.
	engine.RegisterBehavior("01028", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{ATK: 2}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			if ba, ok := msg.(engine.BasicAttack); ok && ba.Player == u.Owner {
				return []engine.Message{
					engine.DiscardControlled{Player: u.Owner, ID: u.ID},
					engine.StunEntity{Target: ba.Target},
				}
			}
			return nil
		},
	})

	// War Machine: exhaust + take 2 → 1 damage to each enemy.
	engine.RegisterBehavior("01030", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust War Machine + he takes 2 damage → deal 1 to each enemy", Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					msgs := []engine.Message{engine.DamageEntity{Target: self, Damage: 2, Source: self}}
					return append(msgs, damageEachEnemyMsgs(g, 1, g.Entity(self).EOwner())...)
				},
			}}
		},
	})

	// Repulsor Blast: 1 damage + mill 5 → +2 per printed energy.
	engine.RegisterBehavior("01031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			bonus := 0
			var milled engine.CardList
			for i := 0; i < 5 && len(p.Deck) > 0; i++ {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				for _, r := range c.Def().Resources {
					if r == "energy" {
						bonus += 2
						break
					}
				}
				milled = append(milled, c)
			}
			p.Discard = append(p.Discard, milled...)
			return cardutil.ChooseEnemy(fmt.Sprintf("Repulsor Blast (%d damage): choose an enemy", 1+bonus),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 1 + bonus, nil })(g, e)
		},
	})

	// Supersonic Punch: 4 damage (8 with Aerial).
	engine.RegisterBehavior("01032", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Supersonic Punch: choose an enemy", func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			p := g.Player(e.EOwner())
			if p != nil && g.EntityHasTrait(p.ID, "aerial") {
				return 8, nil
			}
			return 4, nil
		}),
	})

	// Pepper Potts: resource equal to the discard pile's top card.
	// Approximation: generates a wild resource.
	engine.RegisterBehavior("01033", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})

	// Stark Tower: exhaust → a player returns their topmost Tech upgrade
	// from discard to hand.
	engine.RegisterBehavior("01034", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Stark Tower → return a Tech upgrade from a discard pile", Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var picks []engine.Choice
					for _, q := range g.Players {
						for _, c := range q.Discard {
							def := c.Def()
							if def.Type == "upgrade" && def.HasTrait("tech") {
								picks = append(picks, engine.Choice{
									Label: q.Name + " — " + def.Name, Kind: engine.ChoiceCard, CardCode: def.Code,
								}.Msgs(engine.RecycleFromDiscard{Player: g.Entity(self).EOwner(), From: q.ID, CardID: c.ID}))
								break // topmost only
							}
						}
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: g.Entity(self).EOwner(),
						Question: engine.Ask("Return which Tech upgrade?", picks...)}}
				},
			}}
		},
	})

	// Arc Reactor: exhaust → ready Iron Man.
	engine.RegisterBehavior("01035", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Arc Reactor → ready your identity", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.ReadyEntity{ID: g.Entity(self).EOwner()}}
				},
			}}
		},
	})

	// Mark V Armor: +6 hit points.
	engine.RegisterBehavior("01036", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP += 6
				g.Logf("%s gets +6 hit points (Mark V Armor)", p.Name)
			}
			return nil
		},
	})

	// Mark V Helmet: exhaust → remove 1 threat (each scheme if Aerial).
	engine.RegisterBehavior("01037", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Mark V Helmet → remove 1 threat", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					if p := g.Player(u.Owner); p != nil && g.EntityHasTrait(p.ID, "aerial") {
						var msgs []engine.Message
						for _, sid := range g.Schemes() {
							msgs = append(msgs, engine.ThwartScheme{Scheme: sid, N: 1, Source: u.Owner})
						}
						return msgs
					}
					return []engine.Message{engine.AskQuestion{Player: u.Owner,
						Question: engine.Ask("Remove 1 threat from which scheme?",
							schemePickChoices(g, 1, u.Owner)...)}}
				},
			}}
		},
	})

	// Powered Gauntlets: exhaust → 1 damage (2 with Aerial).
	engine.RegisterBehavior("01038", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Powered Gauntlets → deal 1 damage", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					dmg := 1
					if p := g.Player(g.Entity(self).EOwner()); p != nil && g.EntityHasTrait(p.ID, "aerial") {
						dmg = 2
					}
					return cardutil.ChooseEnemy("Powered Gauntlets: choose an enemy",
						func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return dmg, nil })(
						g, g.Entity(self))
				},
			}}
		},
	})

	// Rocket Boots: +1 HP; exhaust + mental → Aerial until end of phase.
	engine.RegisterBehavior("01039", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP++
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Rocket Boots + spend [mental] → Aerial this phase", Type: engine.AbilityAction,
				Exhaust: true, Cost: 1, CostIcons: "mental:1", HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.GrantTrait{Target: g.Entity(self).EOwner(), Trait: "aerial"},
						engine.AddEntityCounter{ID: self, N: 1}, // marks the granted trait
					}
				},
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			// Expire the granted trait at the end of the phase.
			if _, ok := msg.(engine.EndPhase); ok {
				if u := g.Upgrades[e.EID()]; u != nil && u.Counters > 0 {
					u.Counters = 0
					if p := g.Player(u.Owner); p != nil {
						for i, t := range p.ExtraTraits {
							if t == "aerial" {
								p.ExtraTraits = append(p.ExtraTraits[:i], p.ExtraTraits[i+1:]...)
								break
							}
						}
					}
				}
			}
			return nil
		},
	})

	// Shuri: search your deck for an upgrade.
	engine.RegisterBehavior("01041", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return deckSearchQuestion(g, e.EOwner(), "upgrade", "Shuri: add which upgrade to hand?")
		},
	})

	// Ancestral Knowledge: shuffle up to 3 different discard cards into deck.
	engine.RegisterBehavior("01042", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Discard) == 0 {
				return nil
			}
			seen := map[string]bool{}
			var picks []engine.Choice
			for _, c := range p.Discard {
				if seen[c.Code] {
					continue
				}
				seen[c.Code] = true
				def := c.Def()
				picks = append(picks, engine.Choice{
					Label: def.Name, Kind: engine.ChoiceCard, CardCode: def.Code,
				}.Msgs(engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.AskN("Shuffle up to 3 different cards into your deck", 3, picks...)}}
		},
	})

	// Wakanda Forever!: resolve each Black Panther upgrade's Special.
	// Approximation: resolves in board order, the last one counts as the
	// final step of the sequence (bonus values).
	engine.RegisterBehavior("01043", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			var specials []engine.EntityID
			for _, id := range p.Upgrades {
				u := g.Upgrades[id]
				if u == nil {
					continue
				}
				def := u.EDef()
				if def.HasTrait("black_panther") && strings.Contains(def.Text, "Special") {
					specials = append(specials, id)
				}
			}
			for i, id := range specials {
				final := i == len(specials)-1
				msgs = append(msgs, wakandaSpecial(g, id, final)...)
			}
			return msgs
		},
	})

	// The Golden City: exhaust → draw 2.
	engine.RegisterBehavior("01045", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust The Golden City → draw 2 cards", Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DrawCards{Player: g.Entity(self).EOwner(), N: 2}}
				},
			}}
		},
	})

	// Interrogation Room: after you defeat a minion, exhaust it → remove
	// 1 threat. Approximation: offered to the owner whenever any minion is
	// defeated (the defeater cannot be identified from the message).
	engine.RegisterBehavior("01063", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionDefeated); !ok {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil || s.Exhausted {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: s.Owner, Question: engine.Ask(
				"Exhaust Interrogation Room → remove 1 threat?",
				engine.Choice{ID: "use", Label: "Exhaust + remove 1 threat", Kind: engine.ChoiceLabel}.
					Msgs(engine.ExhaustEntity{ID: e.EID()},
						engine.AskQuestion{Player: s.Owner, Question: engine.Ask(
							"Remove 1 threat from which scheme?", schemePickChoices(g, 1, s.Owner)...)},
					),
				engine.Choice{ID: "skip", Label: "Skip", Kind: engine.ChoicePass},
			)}}
		},
	})

	// Surveillance Team: 3 snoop counters, exhaust + counter → remove 1 threat.
	engine.RegisterBehavior("01064", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust Surveillance Team + counter → remove 1 threat", Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					if s == nil {
						return nil
					}
					return append([]engine.Message{engine.AddEntityCounter{ID: self, N: -1}},
						engine.AskQuestion{Player: s.Owner, Question: engine.Ask(
							"Remove 1 threat from which scheme?", schemePickChoices(g, 1, s.Owner)...)})
				},
			}}
		},
	})

	// Heroic Intuition: +1 THW.
	engine.RegisterBehavior("01065", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1} },
	})

	// Hawkeye: 4 arrow counters; a minion entering play can be shot for 2.
	engine.RegisterBehavior("01066", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 4}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			a := g.Allies[e.EID()]
			if !ok || a == nil || a.Counters <= 0 || a.Exhausted || m.Player != e.EOwner() {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: m.Player, Question: engine.Ask(
				"Hawkeye: remove an arrow counter to deal 2 damage to "+g.Minions[m.MinionID].EDef().Name+"?",
				engine.Choice{ID: "shoot", Label: "Shoot (2 damage)", Kind: engine.ChoiceAbility, SourceID: e.EID()}.
					Msgs(engine.AddEntityCounter{ID: e.EID(), N: -1},
						engine.DamageEntity{Target: m.MinionID, Damage: 2, Source: e.EID()}),
				engine.Choice{ID: "skip", Label: "Skip", Kind: engine.ChoicePass},
			)}}
		},
	})

	// Vision: spend [energy] → +2 THW or +2 ATK until end of phase
	// (limit once per round).
	engine.RegisterBehavior("01068", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Spend [energy] → +2 THW or +2 ATK this phase", Type: engine.AbilityAction,
				Cost: 1, CostIcons: "energy:1", OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Allies[self]
					if a == nil {
						return nil
					}
					// Ally phase bonuses expire at EndPhase.
					var picks []engine.Choice
					picks = append(picks, engine.Choice{ID: "thw", Label: "+2 THW", Kind: engine.ChoiceLabel}.
						Msgs(engine.AllyStatBonus{Ally: self, THW: 2, ATK: 0}))
					picks = append(picks, engine.Choice{ID: "atk", Label: "+2 ATK", Kind: engine.ChoiceLabel}.
						Msgs(engine.AllyStatBonus{Ally: self, THW: 0, ATK: 2}))
					return []engine.Message{engine.AskQuestion{Player: g.Entity(self).EOwner(),
						Question: engine.Ask("Vision gets +2 to which power until the end of the phase?", picks...)}}
				},
			}}
		},
	})

	registerBPUpgrades()

	// Hulk: after he attacks, discard top of deck and resolve by resource.
	engine.RegisterBehavior("01050", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyAttackWindow)
			if !ok || w.Ally != e.EID() {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			c := p.Deck[0]
			p.Deck = p.Deck[1:]
			p.Discard = append(p.Discard, c)
			var msgs []engine.Message
			physical, energy, mental := resourceFlags(c)
			if mental {
				// Discard Hulk.
				msgs = append(msgs, engine.DiscardControlled{Player: p.ID, ID: e.EID()})
				g.Logf("Hulk: mental resource discards Hulk")
				return msgs
			}
			if physical {
				msgs = append(msgs, cardutil.ChooseEnemy("Hulk: deal 2 damage to an enemy",
					func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 2, nil })(g, e)...)
			}
			if energy {
				msgs = append(msgs, damageEachEnemyMsgs(g, 1, p.ID)...)
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: e.EID()})
				for _, q := range g.Players {
					if q.ID != p.ID {
						msgs = append(msgs, engine.DamageEntity{Target: q.ID, Damage: 1, Source: e.EID()})
					}
				}
			}
			return msgs
		},
	})

	// Tigra: after she attacks a minion, heal 1.
	// Approximation: heals whenever she attacks a minion (defeat not
	// required — the defeat cannot be observed before damage resolves).
	engine.RegisterBehavior("01051", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyAttackWindow)
			if !ok || w.Ally != e.EID() {
				return nil
			}
			if mn := g.Minions[w.Target]; mn != nil {
				return []engine.Message{engine.HealEntity{Target: e.EID(), N: 1}}
			}
			return nil
		},
	})

	// Chase Them Down: remove 2 threat.
	// Approximation: playable as a plain action rather than a
	// response-after-defeat (no post-resolution window exists).
	engine.RegisterBehavior("01052", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask("Chase Them Down: remove 2 threat from which scheme?",
					schemePickChoices(g, 2, e.EOwner())...)}}
		},
	})

	// The Power of X resource cards: doubled-resource bonus is already
	// data-driven (payment validation reads the card name).
	for _, code := range []string{"01055", "01062", "01072", "01079", "01086", "01094"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// Tac Team: 3 attack counters, exhaust + counter → 2 damage.
	engine.RegisterBehavior("01056", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust Tac Team + counter → deal 2 damage", Type: engine.AbilityAction, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return append([]engine.Message{engine.AddEntityCounter{ID: self, N: -1}},
						cardutil.ChooseEnemy("Tac Team: deal 2 damage to which enemy?",
							func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 2, nil })(
							g, g.Entity(self))...)
				},
			}}
		},
	})

	// Combat Training: +1 ATK.
	engine.RegisterBehavior("01057", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} },
	})

	// Daredevil: after he thwarts, deal 1 damage to an enemy.
	engine.RegisterBehavior("01058", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyThwartWindow)
			if !ok || w.Ally != e.EID() {
				return nil
			}
			return cardutil.ChooseEnemy("Daredevil: deal 1 damage to which enemy?",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 1, nil })(g, e)
		},
	})

	// Jessica Jones: +1 THW per side scheme in play.
	engine.RegisterBehavior("01059", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			switch msg.(type) {
			case engine.RevealEncounterCard, engine.SchemeDefeated:
				if a := g.Allies[e.EID()]; a != nil {
					a.PermTHW = len(g.SideSchemes)
				}
			}
			return nil
		},
	})

	// Get Ready: ready an ally.
	engine.RegisterBehavior("01069", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, q := range g.Players {
				for _, id := range q.Allies {
					a := g.Allies[id]
					if a == nil || !a.Exhausted {
						continue
					}
					picks = append(picks, engine.Choice{
						Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code,
					}.Msgs(engine.ReadyEntity{ID: a.ID}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask("Ready which ally?", picks...)}}
		},
	})

	// Lead from the Front: +1 THW/+1 ATK to a player's characters.
	engine.RegisterBehavior("01070", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, q := range g.Players {
				q := q
				picks = append(picks, engine.Choice{
					Label: q.Name, Kind: engine.ChoiceTarget, SourceID: q.ID,
				}.Msgs(leadFromTheFront(g, q)...))
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask("Which player's characters get +1 THW / +1 ATK?", picks...)}}
		},
	})

	// Make the Call: pay an ally's printed cost from any discard pile and
	// put it into play under your control.
	engine.RegisterBehavior("01071", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, q := range g.Players {
				seen := map[string]bool{}
				for _, c := range q.Discard {
					def := c.Def()
					if def.Type != "ally" || seen[c.Code] {
						continue
					}
					seen[c.Code] = true
					cost := cardutil.Cost(def)
					if len(p.Hand) < cost {
						continue
					}
					picks = append(picks, engine.Choice{
						Label: fmt.Sprintf("%s (cost %d, %s's discard)", def.Name, cost, q.Name),
						Kind:  engine.ChoicePlay, CardCode: def.Code,
					}.WithThen(g.CustomPaymentQuestion(p, cost,
						fmt.Sprintf("Pay %d for %s (Make the Call)", cost, def.Name),
						map[string]any{
							"makeCallFrom": q.ID.String(),
							"makeCallCard": c.ID,
						})))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Put which ally into play?", picks...)}}
		},
	})

	// The Triskelion: ally limit +1 (no ally limit is enforced yet).
	engine.RegisterBehavior("01073", &engine.Behavior{})

	// Inspired: attached ally gets +1 THW and +1 ATK.
	engine.RegisterBehavior("01074", &engine.Behavior{
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
				picks = append(picks, engine.Choice{
					Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code,
				}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: a.ID, ATK: 1, THW: 1}))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Attach Inspired to which ally?", picks...)}}
		},
	})

	// Luke Cage: Toughness keyword handled at spawn.
	engine.RegisterBehavior("01076", &engine.Behavior{})

	// Counter-Punch: after your hero defends, deal ATK damage back.
	engine.RegisterBehavior("01077", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			atk := p.AttackStat(g)
			if atk < 0 {
				atk = 0
			}
			d := engine.Defends{Defender: p.ID, Against: against}
			return d, []engine.Message{engine.DamageEntity{Target: against, Damage: atk, Source: p.ID}}, true
		},
	})

	// Get Behind Me!: cancel a treachery; the villain attacks instead.
	engine.RegisterBehavior("01078", &engine.Behavior{
		TreacheryInterrupt: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if g.Villains[id] != nil {
					return []engine.Message{
						engine.DealBoost{Enemy: id},
						engine.RevealBoost{Enemy: id},
						engine.AskAttack{Enemy: id, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou},
					}
				}
			}
			return []engine.Message{}
		},
	})

	// Med Team: 3 medical counters, exhaust + counter → heal 2.
	engine.RegisterBehavior("01080", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Med Team + counter → heal 2 damage", Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					if s == nil || s.Counters <= 0 {
						return nil
					}
					var picks []engine.Choice
					for _, q := range g.Players {
						if q.Damage > 0 {
							picks = append(picks, engine.Choice{Label: q.Name + " (hero)", Kind: engine.ChoiceTarget, SourceID: q.ID}.
								Msgs(engine.AddEntityCounter{ID: self, N: -1}, engine.HealEntity{Target: q.ID, N: 2}))
						}
						for _, id := range q.Allies {
							a := g.Allies[id]
							if a != nil && a.Damage > 0 {
								picks = append(picks, engine.Choice{Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
									Msgs(engine.AddEntityCounter{ID: self, N: -1}, engine.HealEntity{Target: a.ID, N: 2}))
							}
						}
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: s.Owner,
						Question: engine.Ask("Heal 2 damage from which character?", picks...)}}
				},
			}}
		},
	})

	// Indomitable: after your hero defends, discard to ready the hero.
	engine.RegisterBehavior("01082", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			if !ok || w.Defender != e.EOwner() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			return []engine.Message{
				engine.DiscardControlled{Player: u.Owner, ID: u.ID},
				engine.ReadyEntity{ID: u.Owner},
			}
		},
	})

	// Mockingbird: on play, stun an enemy.
	engine.RegisterBehavior("01083", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				if enemy != nil {
					picks = append(picks, engine.Choice{
						Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode(),
					}.Msgs(engine.StunEntity{Target: id}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask("Stun which enemy?", picks...)}}
		},
	})

	// Haymaker: 3 damage.
	engine.RegisterBehavior("01087", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Haymaker: choose an enemy", func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			return 3, nil
		}),
	})

	// Black Widow (ally): treachery cancel via the treachery window
	// (engine-side offer; exhaust to cancel).
	engine.RegisterBehavior("01075", &engine.Behavior{})

	// Emergency: villain-scheme threat reduction via the engine's
	// ApplyVillainScheme hook.
	engine.RegisterBehavior("01085", &engine.Behavior{})

	// Basic resource cards.
	engine.RegisterBehavior("01088", &engine.Behavior{})
	engine.RegisterBehavior("01089", &engine.Behavior{})
	engine.RegisterBehavior("01090", &engine.Behavior{})

	// Avengers Mansion: exhaust → a player draws 1.
	engine.RegisterBehavior("01091", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Avengers Mansion → a player draws 1 card", Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var picks []engine.Choice
					for _, q := range g.Players {
						picks = append(picks, engine.Choice{Label: q.Name, Kind: engine.ChoiceTarget, SourceID: q.ID}.
							Msgs(engine.DrawCards{Player: q.ID, N: 1}))
					}
					return []engine.Message{engine.AskQuestion{Player: g.Entity(self).EOwner(),
						Question: engine.Ask("Which player draws 1 card?", picks...)}}
				},
			}}
		},
	})

	// Helicarrier: exhaust → a player's next card costs 1 less this phase.
	engine.RegisterBehavior("01092", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Helicarrier → a player's next card costs 1 less", Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var picks []engine.Choice
					for _, q := range g.Players {
						q := q
						picks = append(picks, engine.Choice{Label: q.Name, Kind: engine.ChoiceTarget, SourceID: q.ID}.
							Msgs(engine.CostDiscountApply{Player: q.ID, Amount: 1}))
					}
					return []engine.Message{engine.AskQuestion{Player: g.Entity(self).EOwner(),
						Question: engine.Ask("Which player gets the discount?", picks...)}}
				},
			}}
		},
	})

	// Tenacity: spend physical + discard → ready your hero.
	engine.RegisterBehavior("01093", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Spend [physical] + discard Tenacity → ready your hero", Type: engine.AbilityAction,
				Cost: 1, CostIcons: "physical:1", HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					return []engine.Message{
						engine.DiscardControlled{Player: u.Owner, ID: self},
						engine.ReadyEntity{ID: u.Owner},
					}
				},
			}}
		},
	})
}

// registerBPUpgrades installs the Black Panther upgrades' Special abilities
// (resolved via Wakanda Forever!; final step doubles the numbers).
func registerBPUpgrades() {
	// Energy Daggers: 1 damage to the villain + enemies engaged with a
	// player (2 if final).
	engine.RegisterBehavior("01046", &engine.Behavior{
		Abilities: bpSpecial(func(g *engine.Game, u *engine.Upgrade, final bool) []engine.Message {
			n := 1
			if final {
				n = 2
			}
			var picks []engine.Choice
			for _, q := range g.Players {
				q := q
				var msgs []engine.Message
				for _, id := range cardutil.SortedEnemyIDs(g) {
					if g.Villains[id] != nil || (g.Minions[id] != nil && g.Minions[id].EngagedWith == q.ID) {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: n, Source: u.ID})
					}
				}
				picks = append(picks, engine.Choice{Label: q.Name, Kind: engine.ChoiceTarget, SourceID: q.ID}.Msgs(msgs...))
			}
			return []engine.Message{engine.AskQuestion{Player: u.Owner,
				Question: engine.Ask("Energy Daggers: which player's enemies?", picks...)}}
		}),
	})

	// Panther Claws: 2 damage (4 if final).
	engine.RegisterBehavior("01047", &engine.Behavior{
		Abilities: bpSpecial(func(g *engine.Game, u *engine.Upgrade, final bool) []engine.Message {
			n := 2
			if final {
				n = 4
			}
			return cardutil.ChooseEnemy(fmt.Sprintf("Panther Claws: %d damage to which enemy?", n),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(g, g.Entity(u.ID))
		}),
	})

	// Tactical Genius: remove 1 threat (2 if final).
	engine.RegisterBehavior("01048", &engine.Behavior{
		Abilities: bpSpecial(func(g *engine.Game, u *engine.Upgrade, final bool) []engine.Message {
			n := 1
			if final {
				n = 2
			}
			return []engine.Message{engine.AskQuestion{Player: u.Owner,
				Question: engine.Ask(fmt.Sprintf("Tactical Genius: remove %d threat from which scheme?", n),
					schemePickChoices(g, n, u.Owner)...)}}
		}),
	})

	// Vibranium Suit: move 1 damage (2 if final) from your hero to an enemy.
	engine.RegisterBehavior("01049", &engine.Behavior{
		Abilities: bpSpecial(func(g *engine.Game, u *engine.Upgrade, final bool) []engine.Message {
			n := 1
			if final {
				n = 2
			}
			var picks []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				if enemy != nil {
					picks = append(picks, engine.Choice{
						Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode(),
					}.Msgs(engine.HealEntity{Target: u.Owner, N: n}, engine.DamageEntity{Target: id, Damage: n, Source: u.ID}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: u.Owner,
				Question: engine.Ask("Vibranium Suit: move damage to which enemy?", picks...)}}
		}),
	})
}

// bpSpecial wraps a Black Panther Special as a triggered "wakanda"
// ability (excluded from the action menu; resolved by Wakanda Forever!).
func bpSpecial(run func(g *engine.Game, u *engine.Upgrade, final bool) []engine.Message) func(g *engine.Game, e engine.Entity) []engine.Ability {
	return func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{
			Label: "Special (Wakanda Forever!)", Type: engine.AbilityTrigger, Trigger: "wakanda",
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				u := g.Upgrades[self]
				if u == nil {
					return nil
				}
				return run(g, u, false)
			},
		}}
	}
}

// wakandaSpecial resolves one upgrade's Special with the final-step bonus.
func wakandaSpecial(g *engine.Game, id engine.EntityID, final bool) []engine.Message {
	u := g.Upgrades[id]
	if u == nil {
		return nil
	}
	b := engine.LookupBehavior(u.Code)
	if b.Abilities == nil {
		return nil
	}
	ab := b.Abilities(g, u)
	if len(ab) == 0 {
		return nil
	}
	if !final {
		return ab[0].Execute(g, id)
	}
	// Final step: re-run with the doubled numbers by inspecting the
	// upgrade code.
	switch u.Code {
	case "01046":
		return bpFinalEnergyDaggers(g, u)
	case "01047":
		return cardutil.ChooseEnemy("Panther Claws: 4 damage to which enemy?",
			func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 4, nil })(g, u)
	case "01048":
		return []engine.Message{engine.AskQuestion{Player: u.Owner,
			Question: engine.Ask("Tactical Genius: remove 2 threat from which scheme?",
				schemePickChoices(g, 2, u.Owner)...)}}
	case "01049":
		var picks []engine.Choice
		for _, eid := range cardutil.SortedEnemyIDs(g) {
			enemy := g.Entity(eid)
			if enemy != nil {
				picks = append(picks, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: eid, CardCode: enemy.ECode(),
				}.Msgs(engine.HealEntity{Target: u.Owner, N: 2}, engine.DamageEntity{Target: eid, Damage: 2, Source: u.Owner}))
			}
		}
		if len(picks) > 0 {
			return []engine.Message{engine.AskQuestion{Player: u.Owner,
				Question: engine.Ask("Vibranium Suit: move 2 damage to which enemy?", picks...)}}
		}
	}
	return nil
}

func bpFinalEnergyDaggers(g *engine.Game, u *engine.Upgrade) []engine.Message {
	var msgs []engine.Message
	for _, id := range cardutil.SortedEnemyIDs(g) {
		if g.Villains[id] != nil {
			msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 2, Source: u.Owner})
		}
	}
	for _, mn := range g.Minions {
		msgs = append(msgs, engine.DamageEntity{Target: mn.ID, Damage: 2, Source: u.Owner})
	}
	return msgs
}

// schemePickChoices lists schemes with a flat threat-removal choice.
func schemePickChoices(g *engine.Game, n int, pid engine.PlayerID) []engine.Choice {
	return cardutil.SchemeChoices(g, func(s engine.EntityID) []engine.Message {
		return []engine.Message{engine.ThwartScheme{Scheme: s, N: n, Source: pid}}
	})
}

// damageEachEnemyMsgs deals damage to every villain and minion.
func damageEachEnemyMsgs(g *engine.Game, n int, source engine.EntityID) []engine.Message {
	var msgs []engine.Message
	for _, id := range cardutil.SortedEnemyIDs(g) {
		msgs = append(msgs, engine.DamageEntity{Target: id, Damage: n, Source: source})
	}
	return msgs
}

// deckSearchQuestion builds a whole-deck search limited to a card type.
func deckSearchQuestion(g *engine.Game, pid engine.PlayerID, typ, prompt string) []engine.Message {
	p := g.Player(pid)
	if p == nil {
		return nil
	}
	var picks []engine.Choice
	seen := map[string]bool{}
	for _, c := range p.Deck {
		def := c.Def()
		if def.Type != typ || seen[c.Code] {
			continue
		}
		seen[c.Code] = true
		picks = append(picks, engine.Choice{
			Label: def.Name, Kind: engine.ChoiceCard, CardCode: def.Code,
		}.Msgs(engine.TakeDeckCard{Player: pid, CardID: c.ID},
			engine.ShufflePlayerDeck{Player: pid}))
	}
	if len(picks) == 0 {
		return nil
	}
	return []engine.Message{engine.AskQuestion{Player: pid, Question: engine.Ask(prompt, picks...)}}
}

// resourceFlags reports a card's printed resource types.
func resourceFlags(c engine.Card) (physical, energy, mental bool) {
	for _, r := range c.Def().Resources {
		switch r {
		case "physical":
			physical = true
		case "energy":
			energy = true
		case "mental":
			mental = true
		case "wild":
			physical, energy, mental = true, true, true
		}
	}
	return
}

func hasTrait(p *engine.Player, trait string) bool {
	for _, t := range p.ExtraTraits {
		if t == trait {
			return true
		}
	}
	return false
}

// leadFromTheFront applies +1 THW / +1 ATK to a player's identity and allies.
func leadFromTheFront(g *engine.Game, q *engine.Player) []engine.Message {
	msgs := []engine.Message{engine.ApplyStatBonus{Target: q.ID, THW: 1, ATK: 1}}
	for _, id := range q.Allies {
		if a := g.Allies[id]; a != nil {
			a.BonusTHW++
			a.BonusATK++
		}
	}
	return msgs
}
