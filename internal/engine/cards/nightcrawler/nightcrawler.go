// Package nightcrawler registers the Nightcrawler hero pack: the Bamf!
// mechanic, signature cards and the Azazel nemesis set.
package nightcrawler

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

const bamfCode = "48006"

func init() {
	registerNightcrawler()
	registerSignatures()
	registerNemesis()
	registerObligation()
	registerNightcrawlerExtras()
}

// registerNightcrawler installs the identity (48001a/b).
func registerNightcrawler() {
	engine.RegisterBehavior("48001", &engine.Behavior{
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			var abs []engine.Ability
			// Rapid Teleportation — spend 1 resource → return a Bamf!
			// from your discard pile to your hand (once per phase,
			// approximated once per round).
			hasBamfDiscard := false
			for _, c := range p.Discard {
				if c.Code == bamfCode {
					hasBamfDiscard = true
					break
				}
			}
			if hasBamfDiscard && len(p.Hand) > 0 {
				abs = append(abs, engine.Ability{
					Label:        "Rapid Teleportation — discard 1 resource: return a Bamf! to hand",
					Type:         engine.AbilityAction,
					HeroOnly:     true,
					OncePerRound: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						p := g.Player(self)
						if p == nil {
							return nil
						}
						var bamf *engine.Card
						for i := range p.Discard {
							if p.Discard[i].Code == bamfCode {
								bamf = &p.Discard[i]
								break
							}
						}
						if bamf == nil {
							return nil
						}
						target := bamf.ID
						var choices []engine.Choice
						for _, c := range p.Hand {
							if len(c.Def().Resources) == 0 {
								continue
							}
							choices = append(choices, engine.Choice{
								Label: "Discard " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
							}.Msgs(
								engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}},
								engine.ReturnDiscardCard{Player: p.ID, CardID: target},
							))
						}
						if len(choices) == 0 {
							return nil
						}
						return []engine.Message{engine.AskQuestion{
							Player:   p.ID,
							Question: engine.Ask("Rapid Teleportation — spend 1 resource", choices...),
						}}
					},
				})
			}
			// Kurt Wagner: search your deck for a Bamf! (once per round).
			abs = append(abs, engine.Ability{
				Label:        "Kurt Wagner — search your deck for a Bamf!",
				Type:         engine.AbilityAction,
				AlterEgoOnly: true,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(self)
					if p == nil {
						return nil
					}
					for _, c := range p.Deck {
						if c.Code == bamfCode {
							return []engine.Message{
								engine.TakeDeckCard{Player: p.ID, CardID: c.ID},
								engine.ShufflePlayerDeck{Player: p.ID},
							}
						}
					}
					return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
				},
			})
			return abs
		},
	})
}

// bamfsAttachedTo lists enemies with a Bamf! attached.
func bamfsAttachedTo(g *engine.Game, target engine.EntityID) bool {
	p := firstNightcrawler(g)
	if p == nil {
		return false
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == bamfCode && u.AttachTo == target {
			return true
		}
	}
	return false
}

func firstNightcrawler(g *engine.Game) *engine.Player {
	for _, pl := range g.Players {
		if pl.HeroCode == "48001a" {
			return pl
		}
	}
	return nil
}

// bamfDiscardFrom returns the upgrade id of the Bamf! attached to target.
func bamfAttachedID(g *engine.Game, target engine.EntityID) engine.EntityID {
	for _, u := range g.Upgrades {
		if u.Code == bamfCode && u.AttachTo == target {
			return u.ID
		}
	}
	return ""
}

func registerSignatures() {
	// 48002 Daytripper: search deck+discard for a Bamf! and attach it to
	// an enemy; deal 1 damage to each enemy with a Bamf!.
	engine.RegisterBehavior("48002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			var bamf *engine.Card
			for i := range p.Deck {
				if p.Deck[i].Code == bamfCode {
					bamf = &p.Deck[i]
					break
				}
			}
			if bamf == nil {
				for i := range p.Discard {
					if p.Discard[i].Code == bamfCode {
						bamf = &p.Discard[i]
						break
					}
				}
			}
			if bamf != nil {
				msgs = append(msgs, engine.ShufflePlayerDeck{Player: pid})
				msgs = append(msgs, engine.UpgradeEnterPlay{Player: pid, Card: engine.Card{ID: bamf.ID, Code: bamfCode, Owner: pid}})
			}
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if bamfsAttachedTo(g, id) {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: pid})
				}
			}
			return msgs
		},
	})

	// 48003 Kurt's Chapel: Kurt gets +1 REC; after a basic recovery,
	// exhaust → a player draws 1.
	engine.RegisterBehavior("48003", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{REC: 1}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label:        "Kurt's Chapel — a player draws 1",
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(e.EOwner())
					if p == nil {
						return nil
					}
					if len(g.Players) == 1 {
						return []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
					}
					var choices []engine.Choice
					for _, pl := range g.Players {
						choices = append(choices, engine.Choice{
							Label: pl.Name, Kind: engine.ChoiceTarget, SourceID: pl.ID,
						}.Msgs(engine.DrawCards{Player: pl.ID, N: 1}))
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Kurt's Chapel — who draws?", choices...)}}
				},
			}}
		},
	})

	// 48004 Kurt's Cutlasses: +1 ATK, +1 DEF, retaliate 1.
	engine.RegisterBehavior("48004", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{ATK: 1, DEF: 1, Retaliate: 1}
		},
	})

	// 48005 Prehensile Tail: resource — [wild] for an event.
	engine.RegisterBehavior("48005", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild", EventOnly: true},
	})

	// 48006 Bamf!: attach to an enemy; when it attacks, Nightcrawler
	// defends without exhausting.
	engine.RegisterBehavior(bamfCode, &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if bamfsAttachedTo(g, id) || bamfAttachedID(g, id) == e.EID() {
					continue
				}
				enemy := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: enemy.ECode(),
				}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id}))
			}
			if len(choices) == 0 {
				return []engine.Message{engine.DiscardControlled{Player: pid, ID: e.EID()}}
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Bamf! — attach to an enemy", choices...),
			}}
		},
		DefenseSubstitute: func(g *engine.Game, p *engine.Player, u *engine.Upgrade, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			if u.AttachTo != against {
				return engine.Defends{}, nil, false
			}
			d := engine.Defends{Defender: p.ID, Against: against, NoExhaust: true}
			return d, []engine.Message{engine.DiscardControlled{Player: p.ID, ID: u.ID}}, true
		},
	})

	// 48007 'Port and Punch: 3 damage to an enemy; 3 to each enemy with
	// Bamf! attached.
	engine.RegisterBehavior("48007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var msgs []engine.Message
			choices := cardutil.EnemyChoices(g, 3, pid, func(t engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: t, Damage: 3, Source: pid}}
			})
			if len(choices) > 0 {
				msgs = append(msgs, engine.AskQuestion{Player: pid, Question: engine.Ask("'Port and Punch — deal 3 damage", choices...)})
			}
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if bamfsAttachedTo(g, id) {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 3, Source: pid})
				}
			}
			return msgs
		},
	})

	// 48008 Teleport Drop: discard a Bamf! from an enemy → 8 damage +
	// stun.
	engine.RegisterBehavior("48008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if !bamfsAttachedTo(g, id) {
					continue
				}
				enemy := g.Entity(id)
				bid := bamfAttachedID(g, id)
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: enemy.ECode(),
				}.Msgs(
					engine.DiscardControlled{Player: pid, ID: bid},
					engine.DamageEntity{Target: id, Damage: 8, Source: pid},
					engine.StunEntity{Target: id},
				))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: pid, Question: engine.Ask("Teleport Drop — 8 damage + stun", choices...)}}
		},
	})

	// 48009 Scout Ahead: remove 3 threat (the extra-Bamf branch is
	// approximated away).
	engine.RegisterBehavior("48009", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme("Scout Ahead", func(g *engine.Game, e engine.Entity) int { return 3 }),
	})

	// 48010 'Port Away: discard a Bamf! from hand → change form and ready.
	engine.RegisterBehavior("48010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, c := range p.Hand {
				if c.Code != bamfCode {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: "Discard a Bamf!", Kind: engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(
					engine.DiscardCards{Player: pid, Cards: engine.CardList{c}},
					engine.ChangeForm{Player: pid},
					engine.ReadyEntity{ID: pid},
				))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: pid, Question: engine.Ask("'Port Away — discard a Bamf!?", choices...)}}
		},
	})

	// 48011 Tally Ho!: after Bamf! makes Nightcrawler the defender,
	// return it to hand and deal 3 damage to the attacker.
	engine.RegisterBehavior("48011", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			if !ok || w.Via != bamfCode {
				return nil
			}
			p := g.Player(w.Defender)
			if p == nil || p.HeroCode != "48001a" {
				return nil
			}
			return []engine.Message{
				engine.DamageEntity{Target: w.Against, Damage: 3, Source: p.ID},
			}
		},
	})
}

// registerNemesis installs the Azazel set.
func registerNemesis() {
	// 48028 Brimstone Dimension: when defeated, find Azazel and deal him
	// as a facedown encounter card (approximation: reveal the nemesis
	// set).
	engine.RegisterBehavior("48028", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			s := g.SideSchemes[m.Scheme]
			if s == nil {
				return nil
			}
			for _, p := range g.Players {
				if p.HeroCode == "48001a" {
					for _, mn := range g.Minions {
						if mn.Code == "48027" {
							return nil // already in play
						}
					}
					return []engine.Message{engine.RevealNemesisSet{Player: p.ID}}
				}
			}
			return nil
		},
	})

	// 48030 Brimstone Strike: find Azazel and reveal him (approximation:
	// reveal the nemesis set).
	engine.RegisterBehavior("48030", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for _, mn := range g.Minions {
				if mn.Code == "48027" {
					return nil
				}
			}
			return []engine.Message{engine.RevealNemesisSet{Player: p.ID}}
		},
	})

	// 48029 Azazel's Sword: attach to Azazel or the villain (+1 ATK).
	engine.RegisterBehavior("48029", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id, mn := range g.Minions {
				if mn.Code == "48027" {
					t.Target = id
					return []engine.Message{engine.BoostEnemyAttack{Enemy: id, N: 1}}
				}
			}
			for id := range g.Villains {
				t.Target = id
				return []engine.Message{engine.BoostEnemyAttack{Enemy: id, N: 1}}
			}
			return nil
		},
	})
}

// registerObligation installs Crisis of Faith.
func registerObligation() {
	engine.RegisterBehavior("48026", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			// Discard each Attack and Defense event from your hand.
			var dropped []engine.Card
			kept := engine.CardList{}
			for _, c := range p.Hand {
				def := c.Def()
				if def.Type == "event" && (def.HasTrait("attack") || def.HasTrait("defense")) {
					dropped = append(dropped, c)
				} else {
					kept = append(kept, c)
				}
			}
			p.Hand = kept
			var penalty []engine.Message
			if len(dropped) > 0 {
				p.Discard = append(p.Discard, dropped...)
				g.Logf("Crisis of Faith discards %d events", len(dropped))
			}
			return cardutil.ExhaustOrPenalty(g, p, card,
				"Discard each Attack and Defense event from your hand", penalty...)
		},
	})
}
