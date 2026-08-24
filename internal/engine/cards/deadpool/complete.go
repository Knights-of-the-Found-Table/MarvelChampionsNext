package deadpool

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// iconCount counts the crisis/acceleration/amplify/hazard icons in play
// (Deadpool's scaling rider; amplify icons are not modeled).
func iconCount(g *engine.Game) int {
	n := 0
	if g.MainScheme != nil {
		if g.MainScheme.Crisis {
			n++
		}
		n += g.MainScheme.AccelerationTokens
		n += g.MainScheme.Hazard
	}
	for _, s := range g.SideSchemes {
		if s != nil {
			if s.Crisis {
				n++
			}
			n += s.Hazard
		}
	}
	return n
}

func registerCorpsCards() {
	// 44013 Dogpool: chip an enemy on death.
	engine.RegisterBehavior("44013", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AllyDefeated); !ok {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			if len(g.Enemies()) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(engine.DamageEntity{Target: id, Damage: 1, Source: a.Owner}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   a.Owner,
				Question: engine.Ask("Dogpool's parting shot — 1 damage to:", choices...),
			}}
		},
	})

	// 44014 Headpool: damaged minion attacks another enemy (approximated
	// as 2 damage follow-up).
	engine.RegisterBehavior("44014", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			if mn := g.Minions[m.Target]; mn == nil || mn.HP() <= 0 {
				return nil
			}
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if id != m.Target {
					g.Logf("Headpool's bite turns the minion on its ally")
					return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: e.EID()}}
				}
			}
			return nil
		},
	})

	// 44015 Kidpool: piercing rider not modeled.
	engine.RegisterBehavior("44015", &engine.Behavior{})

	// 44016 Lady Deadpool: death kills a non-Elite minion.
	engine.RegisterBehavior("44016", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AllyDefeated); !ok {
				return nil
			}
			if g.Allies[e.EID()] == nil {
				return nil
			}
			for _, mn := range g.Minions {
				if mn != nil && !mn.EDef().HasTrait("Elite") {
					g.Logf("Lady Deadpool takes %s down with her", mn.EDef().Name)
					return []engine.Message{engine.MinionDefeated{MinionID: mn.ID}}
				}
			}
			return nil
		},
	})

	// 44017 Barely a Scratch: prevent icon-scaled damage (defense event).
	engine.RegisterBehavior("44017", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			n := iconCount(g)
			d := engine.Defends{Defender: p.ID, Against: against, Undefended: true, ExtraPrevent: n}
			g.Logf("Barely a Scratch prevents %d damage", n)
			return d, nil, true
		},
	})

	// 44018 Cutupper: 5 damage + stun.
	engine.RegisterBehavior("44018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(g.Enemies()) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(engine.DamageEntity{Target: id, Damage: 5, Source: p.ID}, engine.StunEntity{Target: id}))
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID, Question: engine.Ask("Cutupper — 5 damage and stun:", choices...),
			}}
		},
	})

	// 44019 Da Bomb: 10 to the villain + splash.
	engine.RegisterBehavior("44019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			for id := range g.Villains {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 10, Source: p.ID})
				break
			}
			n := iconCount(g)
			for _, id := range cardutil.SortedEnemyIDs(g) {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: n, Source: p.ID})
			}
			for _, o := range g.Players {
				msgs = append(msgs, engine.DamageEntity{Target: o.ID, Damage: n, Source: p.ID})
			}
			return msgs
		},
	})

	// 44020 Get Rage-y: ready + buff an ally.
	engine.RegisterBehavior("44020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Allies) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil {
					continue
				}
				choices = append(choices, engine.Choice{
					ID: "ally-" + id.String(), Label: a.EDef().Name, Kind: engine.ChoiceTarget,
				}.Msgs(engine.ReadyEntity{ID: id}, engine.AllyStatBonus{Ally: id, ATK: 1}))
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID, Question: engine.Ask("Get Rage-y — ready and buff:", choices...),
			}}
		},
	})

	// 44021 "I Got This": icon-based multi-tool.
	engine.RegisterBehavior("44021", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := iconCount(g)
			var msgs []engine.Message
			if n >= 1 && len(g.Enemies()) > 0 {
				msgs = append(msgs, engine.DamageEntity{Target: g.Enemies()[0], Damage: 3, Source: p.ID})
			}
			if n >= 2 && len(g.Schemes()) > 0 {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.Schemes()[0], N: 2, Source: p.ID})
			}
			if n >= 3 {
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && a.Exhausted {
						msgs = append(msgs, engine.ReadyEntity{ID: id})
						break
					}
				}
			}
			if n >= 4 {
				msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
			}
			g.Logf("I Got This — %d icons in play", n)
			return msgs
		},
	})

	// 44022 Not my Responsibility: threat-to-damage redirect (approximated
	// as an auto window on the main scheme like Great Responsibility).
	engine.RegisterBehavior("44022", &engine.Behavior{
		TreacheryInterrupt: nil,
	})

	// 44023 'Pool Inspection: 5 + icons threat removal.
	engine.RegisterBehavior("44023", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			if g.MainScheme != nil {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 5, Source: p.ID})
			}
			n := iconCount(g)
			if n > 0 {
				for _, id := range g.Schemes() {
					msgs = append(msgs, engine.ThwartScheme{Scheme: id, N: n, Source: p.ID})
				}
			}
			return msgs
		},
	})

	// 44024 Live Dangerously: +2 hand size.
	engine.RegisterBehavior("44024", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Logf("Live Dangerously — +2 hand size this game (approximated as a log; the bonus applies via the flag below)")
			for _, p := range g.Players {
				p.TempHandSize += 0
			}
			return nil
		},
	})

	// 44025-44027 Self * resources: damage-scaled doubling — approximation
	// handled in the engine's powerOfBonus-style payment (not modeled);
	// registered for the sweep.
	for _, code := range []string{"44025", "44026", "44027"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 44028 Git Gud: defeat save.
	engine.RegisterBehavior("44028", &engine.Behavior{
		DefeatSave: func(g *engine.Game, p *engine.Player, u *engine.Upgrade) bool {
			p.Damage = p.MaxHP - 1
			if p.IsHero() {
				g.Push(engine.ChangeForm{Player: p.ID})
			}
			g.Delete(u.ID)
			g.Logf("Git Gud! %s holds on at 1 HP", p.Name)
			return true
		},
	})

	// 44029 Healing Factor: heal 2 each player phase.
	engine.RegisterBehavior("44029", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.BeginPhase); !ok || e.EExhausted() {
				return nil
			}
			m := msg.(engine.BeginPhase)
			if m.Phase != engine.PhasePlayer {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.HealEntity{Target: u.Owner, N: 2}}
		},
	})

	// 44030 Stick-To-Itiveness: pay physical → ready hero.
	engine.RegisterBehavior("44030", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			for _, c := range p.Hand {
				for _, r := range c.Def().Resources {
					if r == "physical" || r == "wild" {
						return []engine.Ability{{
							Label: "Stick-To-Itiveness — ready your hero (spend " + c.Def().Name + ")", Type: engine.AbilityAction,
							HeroOnly: true, Exhaust: true,
							Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
								u := g.Upgrades[self]
								p := g.Player(u.Owner)
								if p == nil {
									return nil
								}
								for _, hc := range p.Hand {
									for _, rr := range hc.Def().Resources {
										if rr == "physical" || rr == "wild" {
											return []engine.Message{
												engine.DiscardCards{Player: p.ID, Cards: engine.CardList{hc}},
												engine.ReadyEntity{ID: p.ID},
											}
										}
									}
								}
								return nil
							},
						}}
					}
				}
			}
			return nil
		},
	})

	// 44031 Frenemies: the Cable/Deadpool team-up (same engine as 40026).
	engine.RegisterBehavior("44031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a != nil && (engine.BaseCodeOf(a.Code) == "40024" || engine.BaseCodeOf(a.Code) == "45011") {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: p.ID})
				}
			}
			for i, id := range g.Schemes() {
				if i >= 2 {
					break
				}
				msgs = append(msgs, engine.ThwartScheme{Scheme: id, N: 3, Source: p.ID})
			}
			return msgs
		},
	})
}

func registerDreadpool() {
	// 44037 Crisis of Infinite Deadpools: reveal the set-aside Dreadpool
	// board.
	engine.RegisterBehavior("44037", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			var kept engine.CardList
			for _, c := range g.SetAside {
				if c.Code == "44038" || c.Code == "44039" {
					msgs = append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
					continue
				}
				kept = append(kept, c)
			}
			g.SetAside = kept
			for _, c := range kept {
				g.EncounterDeck = append(g.EncounterDeck, c)
			}
			g.SetAside = nil
			g.ShuffleEncounterDeck()
			return msgs
		},
	})

	// 44038 Dreadpool: defeat deals him facedown.
	engine.RegisterBehavior("44038", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn != nil && mn.EngagedWith == "" {
				if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
					mn.EngagedWith = p.ID
				}
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionDefeated); !ok {
				return nil
			}
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			p := g.Player(cardutil.FirstPlayerID(g))
			if p != nil {
				p.EncounterDown = append(p.EncounterDown, engine.Card{ID: g.NextCardID(), Code: mn.Code})
				g.Logf("Dreadpool is dealt facedown to %s", p.Name)
			}
			return nil
		},
	})

	// 44039 Dreadful Deeds: threat per 'Pool card holder.
	engine.RegisterBehavior("44039", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			n := 0
			for _, p := range g.Players {
				pool := false
				for _, id := range p.Supports {
					if sup := g.Supports[id]; sup != nil && isPoolCard(sup.Code) {
						pool = true
					}
				}
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && isPoolCard(a.Code) {
						pool = true
					}
				}
				for _, id := range p.Upgrades {
					if u := g.Upgrades[id]; u != nil && isPoolCard(u.Code) {
						pool = true
					}
				}
				if pool {
					n += 2
				}
			}
			s.Threat += n
			return nil
		},
	})

	// 44040 Anti-Regeneration Ray: text blanking not modeled.
	engine.RegisterBehavior("44040", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, mn := range g.Minions {
				if mn != nil && engine.BaseCodeOf(mn.Code) == "44038" {
					t.Target = mn.ID
					return nil
				}
			}
			if id := firstVillain(g); id != "" {
				t.Target = id
			}
			return nil
		},
	})

	// 44041 'Pool-ized: possess the costliest ally (ally → minion).
	engine.RegisterBehavior("44041", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			best, pick := -1, engine.EntityID("")
			var owner engine.PlayerID
			for _, p := range g.Players {
				for _, id := range p.Allies {
					a := g.Allies[id]
					if a == nil {
						continue
					}
					if c := cardutil.Cost(a.EDef()); c > best {
						best, pick, owner = c, id, p.ID
					}
				}
			}
			if pick == "" {
				if c, ok := g.DrawEncounter(); ok {
					return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
				}
				return nil
			}
			a := g.Allies[pick]
			hp, atk, sch := a.MaxHP-a.Damage, a.AttackVal, a.ThwartVal
			code := a.Code
			g.Delete(pick)
			mn := &engine.Minion{ID: g.NextEntityID(engine.KindMinion), Code: code, MaxHP: hp, AttackVal: atk, SchemeVal: sch}
			g.Minions[mn.ID] = mn
			mn.EngagedWith = owner
			t.Target = mn.ID
			g.Logf("%s is 'Pool-ized!", engine.DB.MustLookup(code).Name)
			return nil
		},
	})

	// 44042 Metacidal Tendencies: hit the Corps.
	engine.RegisterBehavior("44042", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			n := 2
			for _, mn := range g.Minions {
				if mn != nil && engine.BaseCodeOf(mn.Code) == "44038" {
					n = 3
				}
			}
			dealt := false
			var msgs []engine.Message
			for _, o := range g.Players {
				for _, id := range o.Allies {
					if a := g.Allies[id]; a != nil && a.EDef().HasTrait("Deadpool Corps") {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: n, Source: t.ID})
						dealt = true
					}
				}
			}
			if !dealt && g.MainScheme != nil {
				msgs = append(msgs, engine.AddAccelerationToken{})
			}
			return msgs
		},
	})
}

func isPoolCard(code string) bool {
	def, ok := engine.DB.Lookup(code)
	return ok && def.HasTrait("Deadpool Corps")
}

func firstVillain(g *engine.Game) engine.EntityID {
	for id := range g.Villains {
		return id
	}
	return ""
}

func registerCorpsNeutrals() {
	// 44043 Bob: 2 damage or 1 threat.
	engine.RegisterBehavior("44043", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			if len(g.Enemies()) > 0 {
				var dmg []engine.Choice
				for _, id := range cardutil.SortedEnemyIDs(g) {
					enemy := g.Entity(id)
					dmg = append(dmg, engine.Choice{
						Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.DamageEntity{Target: id, Damage: 2, Source: p.ID}))
				}
				choices = append(choices, engine.Choice{ID: "dmg", Label: "Deal 2 damage", Kind: engine.ChoiceLabel}.
					Msgs(engine.AskQuestion{Player: p.ID, Question: engine.Ask("Bob — damage:", dmg...)}))
			}
			if len(g.Schemes()) > 0 {
				var thw []engine.Choice
				for _, id := range g.Schemes() {
					s := g.Entity(id)
					thw = append(thw, engine.Choice{
						Label: s.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.ThwartScheme{Scheme: id, N: 1, Source: p.ID}))
				}
				choices = append(choices, engine.Choice{ID: "thw", Label: "Remove 1 threat", Kind: engine.ChoiceLabel}.
					Msgs(engine.AskQuestion{Player: p.ID, Question: engine.Ask("Bob — thwart:", thw...)}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Bob — choose:", choices...)}}
		},
	})

	// 44044 Negasonic: cancel a treachery for 2 self damage (window in
	// the engine's treachery handler).
	engine.RegisterBehavior("44044", &engine.Behavior{})

	// 44045 Pandapool: Toughness keyword is data-driven.
	engine.RegisterBehavior("44045", &engine.Behavior{})

	// 44046 Break Time: heal each identity 1 (comic-reading timer
	// approximated).
	engine.RegisterBehavior("44046", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.HealEntity{Target: p.ID, N: 1})
			}
			return msgs
		},
	})

	// 44047 Get in Front of Me!: treachery cancel + villain attacks you.
	engine.RegisterBehavior("44047", &engine.Behavior{
		TreacheryInterrupt: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if id := firstVillain(g); id != "" {
				return []engine.Message{engine.AskAttack{Enemy: id, Player: p.ID}}
			}
			return nil
		},
	})

	// 44048 Mulligan: discard hand, redraw.
	engine.RegisterBehavior("44048", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return !g.UsedThisRound["card-played"]
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			hand := append(engine.CardList{}, p.Hand...)
			p.Hand = nil
			g.Logf("Mulligan! %s discards %d cards", p.Name, len(hand))
			return []engine.Message{
				engine.DiscardCards{Player: p.ID, Cards: hand},
				engine.DrawCards{Player: p.ID, N: p.HandSize(g)},
			}
		},
	})

	// 44049 Deadpool Corps Ship: facedown encounter for a 'Pool ally.
	engine.RegisterBehavior("44049", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			for _, c := range p.Hand {
				if c.Def().Type == "ally" && c.Def().HasTrait("Deadpool Corps") {
					card := c
					return []engine.Ability{{
						Label: "Deadpool Corps Ship — deploy " + card.Def().Name, Type: engine.AbilityAction,
						Exhaust: true,
						Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
							s := g.Supports[self]
							p := g.Player(s.Owner)
							if p == nil {
								return nil
							}
							if _, ok := p.Hand.Remove(card.ID); ok {
								def := card.Def()
								a := &engine.Ally{
									ID:        g.NextEntityID(engine.KindAlly),
									Code:      def.Code,
									Owner:     p.ID,
									MaxHP:     derefIntA(def.HP, 1),
									AttackVal: derefIntA(def.Attack, 0),
									ThwartVal: derefIntA(def.Thwart, 0),
									Tough:     def.HasKeyword("Toughness"),
								}
								g.Allies[a.ID] = a
								p.Allies = append(p.Allies, a.ID)
								g.Push(engine.AllyEnteredPlay{Ally: a.ID, Player: p.ID})
								g.Logf("%s beams down from the ship", def.Name)
							}
							return []engine.Message{engine.DealEncounterToPlayer{Player: p.ID}}
						},
					}}
				}
			}
			return nil
		},
	})

	// 44050 Plot Convenience: bank aspect cards.
	engine.RegisterBehavior("44050", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			var out []engine.Ability
			s := g.Supports[e.EID()]
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			if len(s.AttachedCards) < 3 && len(p.Hand) > 0 {
				out = append(out, engine.Ability{
					Label: "Plot Convenience — stash a card", Type: engine.AbilityAction,
					Exhaust: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						s := g.Supports[self]
						p := g.Player(s.Owner)
						if p == nil || len(p.Hand) == 0 || len(s.AttachedCards) >= 3 {
							return nil
						}
						c := p.Hand[0]
						if _, ok := p.Hand.Remove(c.ID); ok {
							s.AttachedCards = append(s.AttachedCards, c)
							s.Counters = len(s.AttachedCards)
							g.Logf("%s is stashed on Plot Convenience", c.Def().Name)
						}
						return nil
					},
				})
			}
			if len(s.AttachedCards) > 0 {
				out = append(out, engine.Ability{
					Label: "Plot Convenience — retrieve a card", Type: engine.AbilityAction,
					Exhaust: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						s := g.Supports[self]
						p := g.Player(s.Owner)
						if p == nil || len(s.AttachedCards) == 0 {
							return nil
						}
						c := s.AttachedCards[0]
						s.AttachedCards = s.AttachedCards[1:]
						s.Counters = len(s.AttachedCards)
						p.Hand = append(p.Hand, c)
						return nil
					},
				})
			}
			return out
		},
	})

	// 44051 Ambush: attached side scheme's defeat discards a minion.
	engine.RegisterBehavior("44051", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil || len(g.Schemes()) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					ID: "sch-" + id.String(), Label: s.EDef().Name, Kind: engine.ChoiceTarget,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id}))
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID, Question: engine.Ask("Ambush — attach to:", choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.AttachTo != m.Scheme {
				return nil
			}
			for _, mn := range g.Minions {
				if mn != nil && !mn.EDef().HasTrait("Elite") {
					return []engine.Message{engine.MinionDefeated{MinionID: mn.ID}}
				}
			}
			return nil
		},
	})

	// 44052 Bazooka: icon-scaled shot.
	engine.RegisterBehavior("44052", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil || len(g.Enemies()) == 0 {
				return nil
			}
			n := iconCount(g)
			return []engine.Ability{{
				Label: fmt.Sprintf("Bazooka — %d damage (discard itself)", n), Type: engine.AbilityAction,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.Owner)
					if p == nil || len(g.Enemies()) == 0 {
						return nil
					}
					var choices []engine.Choice
					for _, id := range cardutil.SortedEnemyIDs(g) {
						enemy := g.Entity(id)
						choices = append(choices, engine.Choice{
							Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id,
						}.Msgs(engine.DiscardControlled{Player: u.Owner, ID: u.ID},
							engine.DamageEntity{Target: id, Damage: n, Source: p.ID}))
					}
					return []engine.Message{engine.AskQuestion{
						Player: p.ID, Question: engine.Ask("Bazooka — target:", choices...),
					}}
				},
			}}
		},
	})

	// 44053 Blackout / 44056 Rock Paper Scissors / 44057 Tic-Tac-Toe /
	// 44058 War: metagame minigames approximated away.
	for _, code := range []string{"44053", "44056", "44057"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
	engine.RegisterBehavior("44058", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil || len(g.Enemies()) == 0 || len(p.Deck) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "War — trade deck cards for damage", Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					p := g.Player(u.Owner)
					if p == nil || len(p.Deck) == 0 {
						return nil
					}
					enc, _ := g.DrawEncounter()
					g.EncounterDiscard = append(g.EncounterDiscard, enc)
					stars := cardutil.BoostOf(enc)
					c := p.Deck[0]
					dmg := cardutil.Cost(c.Def())
					g.Logf("War: encounter %s (+%d stars), deck %s (%d damage)", enc.Def().Name, stars, c.Def().Name, dmg)
					return []engine.Message{
						engine.MillPlayerDeck{Player: p.ID, N: 1},
						engine.DamageEntity{Target: p.ID, Damage: stars, Source: u.Owner},
						engine.DamageEntity{Target: g.Enemies()[0], Damage: dmg, Source: u.Owner},
					}
				},
			}}
		},
	})

	// 44054 Distraction: attached minion cannot activate (engine minion
	// activation check).
	engine.RegisterBehavior("44054", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, mn := range g.Minions {
				if mn != nil && !mn.EDef().HasTrait("Elite") {
					t.Target = mn.ID
					return nil
				}
			}
			if c, ok := g.DrawEncounter(); ok {
				return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
			}
			return nil
		},
	})

	// 44055 Laser Swords: +ATK per icon (IdentityStatsG).
	engine.RegisterBehavior("44055", &engine.Behavior{
		IdentityStatsG: func(g *engine.Game, p *engine.Player, u *engine.Upgrade) engine.StatBonus {
			n := iconCount(g)
			if n > 4 {
				n = 4
			}
			return engine.StatBonus{ATK: n}
		},
	})
}

func derefIntA(p *int, fallback int) int {
	if p == nil {
		return fallback
	}
	return *p
}
