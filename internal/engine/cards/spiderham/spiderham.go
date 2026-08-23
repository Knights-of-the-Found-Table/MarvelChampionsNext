// Package spiderham registers the Spider-Ham hero (30001): the
// Spider-Ham / Peter Porker identity with its toon-counter economy
// (counters double as wild resources through the identity-level
// ResourceAbility extension), the Cartoon signature cards, the
// "I Really Want a Hot Dog!" obligation and The Green Gobbler nemesis
// set.
package spiderham

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerSpiderHam()
	registerHamSignatures()
	registerHamObligation()
	registerHamNemesis()
}

// registerSpiderHam installs the Spider-Ham / Peter Porker identity
// (30001a/b). Toon counters are tracked on the identity (p.Counters) and
// are spendable as wild resources in either form through the identity
// Resource hook (see entity.go ResourceAbility.NoExhaust).
func registerSpiderHam() {
	engine.RegisterBehavior("30001", &engine.Behavior{
		// Each toon counter on Spider-Ham can be spent as a wild resource.
		Resource: &engine.ResourceAbility{Icon: "wild", UsesCounters: true, NoExhaust: true},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			switch m := msg.(type) {
			// Spider-Nonsense — Response: after Spider-Ham takes any
			// amount of damage, place 1 toon counter on him.
			case engine.DamageEntity:
				if m.Target == p.ID && m.Damage > 0 && p.IsHero() {
					g.Logf("Spider-Nonsense — Spider-Ham gains a toon counter")
					return []engine.Message{engine.AddEntityCounter{ID: p.ID, N: 1}}
				}
			// Cartoon Power — Response: after Peter Porker makes a basic
			// recovery, place 1 toon counter on him.
			case engine.BasicRecover:
				if m.Player == p.ID && !p.IsHero() {
					g.Logf("Cartoon Power — Peter Porker gains a toon counter")
					return []engine.Message{engine.AddEntityCounter{ID: p.ID, N: 1}}
				}
			}
			return nil
		},
	})
}

func registerHamSignatures() {
	// 30002 Captain Americat: Response — after he enters play, place 1
	// toon counter on your identity and shuffle 1 Spider-Ham card from
	// your discard pile into your deck.
	engine.RegisterBehavior("30002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			msgs := []engine.Message{engine.AddEntityCounter{ID: p.ID, N: 1}}
			choices := []engine.Choice{cardutil.Skip()}
			for _, c := range p.Discard {
				if c.Def().CardSet == "spider_ham" {
					choices = append(choices, engine.Choice{
						Label: c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}))
				}
			}
			if len(choices) > 1 {
				msgs = append(msgs, engine.AskQuestion{
					Player:   p.ID,
					Question: engine.Ask("Captain Americat — shuffle a Spider-Ham card into your deck", choices...),
				})
			}
			return msgs
		},
	})

	// 30003 Ham It Up: Hero Action (thwart) — remove 1 threat from a
	// scheme for each toon counter on Spider-Ham.
	engine.RegisterBehavior("30003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || p.Counters <= 0 {
				return nil
			}
			return cardutil.ChooseScheme("Ham It Up", func(g *engine.Game, e engine.Entity) int {
				return p.Counters
			})(g, e)
		},
	})

	// 30004 Hogwashed: Hero Action — remove 1 toon counter → deal 5
	// damage to a minion or remove 5 threat from a side scheme.
	engine.RegisterBehavior("30004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || p.Counters <= 0 {
				return nil
			}
			pay := engine.AddEntityCounter{ID: p.ID, N: -1}
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				choices = append(choices, engine.Choice{
					Label: "Deal 5 damage to " + cardutil.EnemyLabel(mn), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: mn.Code,
				}.Msgs(pay, engine.DamageEntity{Target: id, Damage: 5, Source: p.ID}))
			}
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				s := g.SideSchemes[id]
				choices = append(choices, engine.Choice{
					Label: "Remove 5 threat from " + s.EDef().Name, Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: s.Code,
				}.Msgs(pay, engine.ThwartScheme{Scheme: id, N: 5, Source: p.ID}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Hogwashed — remove 1 toon counter → choose:", choices...),
			}}
		},
	})

	// 30005 "I Don't Think So!": Hero Interrupt — when you reveal a card
	// from the encounter deck, remove 1 toon counter → cancel its effects
	// and discard it. (Approximation: covers treachery reveals through
	// the treachery interrupt window; other card types' reveal windows
	// are not exposed.)
	engine.RegisterBehavior("30005", &engine.Behavior{
		TreacheryInterrupt: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if !p.IsHero() || p.Counters <= 0 {
				return nil
			}
			return []engine.Message{
				engine.AddEntityCounter{ID: p.ID, N: -1},
				engine.TreacheryResolve{Player: p.ID, Card: card, Cancelled: true},
			}
		},
	})

	// 30006 Petulant Pig: Hero Action — the villain attacks you; draw 3
	// cards.
	engine.RegisterBehavior("30006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			villain := g.ActiveVillain
			if villain == "" || g.Villains[villain] == nil {
				for _, id := range cardutil.SortedIDs(g.Villains) {
					villain = id
					break
				}
			}
			if villain == "" {
				return []engine.Message{engine.DrawCards{Player: pid, N: 3}}
			}
			return []engine.Message{
				engine.AskAttack{Enemy: villain, Player: pid, Trigger: engine.TriggerVillainAttacksYou},
				engine.DrawCards{Player: pid, N: 3},
			}
		},
	})

	// 30007 Swinging Web Pig: Hero Action (attack) — deal 6 damage to an
	// enemy and confuse it.
	engine.RegisterBehavior("30007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.EnemyChoices(g, 6, pid, func(id engine.EntityID) []engine.Message {
				return []engine.Message{
					engine.DamageEntity{Target: id, Damage: 6, Source: pid},
					engine.ConfuseEntity{Target: id},
				}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Swinging Web Pig — deal 6 damage to and confuse an enemy", choices...),
			}}
		},
	})

	// 30008 The Daily Beagle: Alter-Ego Action — exhaust → place 1 toon
	// counter on Peter Porker.
	engine.RegisterBehavior("30008", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "The Daily Beagle — place 1 toon counter on Peter Porker", Type: engine.AbilityAction,
				AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.AddEntityCounter{ID: e.EOwner(), N: 1}}
				},
			}}
		},
	})

	// 30009 Cartoon Physics: Interrupt — when your identity would take
	// any amount of damage, discard this card → prevent all but 1 of it.
	engine.RegisterBehavior("30009", &engine.Behavior{
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			g.Logf("Cartoon Physics — %s wiggles out of the damage", p.Name)
			g.Delete(u.ID)
			p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			if n <= 1 {
				return 0, 0
			}
			return n - 1, 0
		},
	})

	// 30010 Huge Wooden Hammer: Spider-Ham gets +1 ATK; Hero Interrupt —
	// when Spider-Ham makes a basic attack, exhaust + remove 1 toon
	// counter → +2 ATK for that attack and overkill. (Approximation: the
	// +2 lands as follow-up damage on the same target, the Nick Fury
	// assault-suit precedent; overkill is not modeled.)
	engine.RegisterBehavior("30010", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} },
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicAttack)
			if !ok {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.Exhausted {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil || !p.IsHero() || m.Player != p.ID || p.Counters <= 0 {
				return nil
			}
			g.Logf("Huge Wooden Hammer — +2 damage to the attack")
			return []engine.Message{
				engine.ExhaustEntity{ID: u.ID},
				engine.AddEntityCounter{ID: p.ID, N: -1},
				engine.DamageEntity{Target: m.Target, Damage: 2, Source: u.ID},
			}
		},
	})

	// 30011 Organic Webbing: Spider-Ham gets +1 THW; Hero Action —
	// exhaust + remove 1 toon counter → ready Spider-Ham and he gains
	// aerial until end of phase. (Approximation: granted traits do not
	// expire.)
	engine.RegisterBehavior("30011", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1} },
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || p.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Organic Webbing — remove 1 toon counter → ready Spider-Ham", Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil || p.Counters <= 0 {
						return nil
					}
					return []engine.Message{
						engine.AddEntityCounter{ID: p.ID, N: -1},
						engine.ReadyEntity{ID: p.ID},
						engine.GrantTrait{Target: p.ID, Trait: "aerial"},
					}
				},
			}}
		},
	})
}

// registerHamObligation installs "I Really Want a Hot Dog!" (30024):
// exhaust Peter Porker and remove 1 toon counter to remove it, or be
// stunned (surge when already stunned).
func registerHamObligation() {
	engine.RegisterBehavior("30024", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var choices []engine.Choice
			if p.Counters > 0 {
				removeMsgs := []engine.Message{}
				if p.IsHero() && !p.FormChanged && !p.Exhausted {
					removeMsgs = append(removeMsgs, engine.ChangeForm{Player: p.ID})
				}
				removeMsgs = append(removeMsgs,
					engine.ExhaustEntity{ID: p.ID},
					engine.AddEntityCounter{ID: p.ID, N: -1},
					engine.ObligationResolve{Player: p.ID, Card: card, Remove: true},
				)
				choices = append(choices, engine.Choice{
					ID: "exhaust", Label: "Exhaust Peter Porker and remove 1 toon counter → remove from the game", Kind: engine.ChoiceLabel,
				}.Msgs(removeMsgs...))
			}
			penalty := []engine.Message{engine.StunEntity{Target: p.ID}}
			if p.Stunned {
				// Surge approximation: reveal another encounter card.
				penalty = append(penalty, engine.RevealNextEncounter{Player: p.ID})
			}
			penalty = append(penalty, engine.ObligationResolve{Player: p.ID, Card: card})
			choices = append(choices, engine.Choice{
				ID: "stun", Label: "You are stunned → discard the obligation", Kind: engine.ChoiceLabel,
			}.Msgs(penalty...))
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("\"I Really Want a Hot Dog!\" — choose:", choices...),
			}}
		},
	})
}

// registerHamNemesis installs the Spider-Ham nemesis set (Nefarious Trap,
// The Green Gobbler, Gobbler Glider, "Feast on This!").
func registerHamNemesis() {
	// 30025 Nefarious Trap: When Defeated — The Green Gobbler attacks the
	// player who defeated this scheme; if he is not in play, search the
	// encounter deck and discard pile for him and put him into play
	// engaged with that player. (Approximation: the defeater is not
	// reported by SchemeDefeated; the first player stands in.)
	engine.RegisterBehavior("30025", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			pid := cardutil.FirstPlayerID(g)
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.Code == "30026" {
					return []engine.Message{engine.MinionActivates{MinionID: id, Player: pid}}
				}
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "30026" {
						zone.Remove(c.ID)
						mn := &engine.Minion{
							ID: g.NextEntityID(engine.KindMinion), Code: c.Code,
							MaxHP: 4, AttackVal: 2, SchemeVal: 1, EngagedWith: pid,
						}
						g.Minions[mn.ID] = mn
						g.Logf("Nefarious Trap — The Green Gobbler engages %s", pid)
						return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: pid}}
					}
				}
			}
			return nil
		},
	})

	// 30026 The Green Gobbler: Forced Response — after he engages you,
	// discard all counters from each card you control.
	engine.RegisterBehavior("30026", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			mn := g.Minions[e.EID()]
			if !ok || mn == nil || m.MinionID != mn.ID {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			g.Logf("The Green Gobbler — %s loses all counters", p.Name)
			p.Counters = 0
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					a.Counters = 0
				}
			}
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil {
					s.Counters = 0
				}
			}
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil {
					u.Counters = 0
				}
			}
			return nil
		},
	})

	// 30027 Gobbler Glider: Surge (data layer). Attach to The Green
	// Gobbler, otherwise to a minion; attached minion gains aerial and
	// +1 ATK. (Approximation: the aerial trait is cosmetic; +1 ATK is
	// modeled.)
	engine.RegisterBehavior("30027", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var gobbler, first *engine.Minion
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil {
					continue
				}
				if first == nil {
					first = mn
				}
				if mn.Code == "30026" {
					gobbler = mn
				}
			}
			best := gobbler
			if best == nil {
				best = first
			}
			if best == nil {
				g.Delete(t.ID)
				return nil
			}
			t.Target = best.ID
			best.Attachments = append(best.Attachments, t.ID)
			g.Logf("Gobbler Glider — attached to %s", best.EDef().Name)
			return []engine.Message{engine.BoostEnemyAttack{Enemy: best.ID, N: 1}}
		},
	})

	// 30028 "Feast on This!": When Revealed — take 2 damage and you are
	// confused; if already confused, surge.
	engine.RegisterBehavior("30028", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			msgs := []engine.Message{
				engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID},
				engine.ConfuseEntity{Target: p.ID},
			}
			if p.Confused {
				msgs = append(msgs, engine.RevealNextEncounter{Player: p.ID})
			}
			return msgs
		},
	})
}
