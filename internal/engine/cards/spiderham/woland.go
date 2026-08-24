// woland.go implements the Web of Life and Destiny set (30012–30038): the
// shared Web-Warrior player cards, the Hunting the Spider-Totems side
// scheme and the Inheritor minions that hunt the spider-totems.
package spiderham

import (
	"fmt"
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerWoLANDPlayers()
	registerWoLANDScheme()
	registerInheritors()
}

// webWarriorCardsControlled counts the Web-Warrior cards a player
// controls (identity + allies, supports and upgrades).
func webWarriorCardsControlled(g *engine.Game, p *engine.Player) int {
	n := 0
	if g.EntityHasTrait(p.ID, "web-warrior") {
		n++
	}
	for _, id := range p.Allies {
		if a := g.Allies[id]; a != nil && a.EDef().HasTrait("web-warrior") {
			n++
		}
	}
	for _, id := range p.Supports {
		if s := g.Supports[id]; s != nil && s.EDef().HasTrait("web-warrior") {
			n++
		}
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("web-warrior") {
			n++
		}
	}
	return n
}

// webWarriorCharacterInPlay reports whether any player identity (in either
// form) or ally with the Web-Warrior trait is in play.
func webWarriorCharacterInPlay(g *engine.Game) bool {
	for _, p := range g.Players {
		if g.EntityHasTrait(p.ID, "web-warrior") {
			return true
		}
	}
	for _, a := range g.Allies {
		if a != nil && a.EDef().HasTrait("web-warrior") {
			return true
		}
	}
	return false
}

func registerWoLANDPlayers() {
	// 30012 Lady Spider: Response — after she thwarts and removes threat
	// from a scheme, if you control another Web-Warrior card, remove an
	// equal amount of threat from a different scheme.
	engine.RegisterBehavior("30012", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyThwartWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil || webWarriorCardsControlled(g, p) < 2 {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			n := a.ThwartVal + a.BonusTHW + a.PermTHW
			if n <= 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				if id == m.Scheme {
					continue
				}
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: "Remove " + fmt.Sprint(n) + " threat from " + s.EDef().Name,
					Kind:  engine.ChoiceTarget, SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.ThwartScheme{Scheme: id, N: n, Source: p.ID}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Lady Spider — remove "+fmt.Sprint(n)+" threat from a different scheme", append(choices, cardutil.Skip())...),
			}}
		},
	})

	// 30013 Spider-Man: Response — after he enters play, remove 1 threat
	// from a scheme for each Web-Warrior card you control (including
	// Spider-Man).
	engine.RegisterBehavior("30013", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme("Spider-Man", func(g *engine.Game, e engine.Entity) int {
			p := g.Player(e.EOwner())
			if p == nil {
				return 0
			}
			return webWarriorCardsControlled(g, p)
		}),
	})

	// 30014 Even the Odds: Hero Action (thwart) — remove 1 threat [per
	// hero] from each side scheme; deal 1 damage to the villain for each
	// side scheme defeated this way. The Requirement ([energy]) is not
	// modeled.
	engine.RegisterBehavior("30014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			perHero := len(g.Players)
			defeated := 0
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				s := g.SideSchemes[id]
				if s == nil {
					continue
				}
				if s.Threat <= perHero {
					defeated++
				}
				msgs = append(msgs, engine.ThwartScheme{Scheme: id, N: perHero, Source: p.ID})
			}
			if defeated > 0 {
				for _, id := range cardutil.SortedIDs(g.Villains) {
					if v := g.Villains[id]; v != nil {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: defeated, Source: p.ID})
						break
					}
				}
			}
			return msgs
		},
	})

	// 30015 Great Responsibility: reprint of core 01061 — resolved by the
	// interrupt window in handle(SchemeThreat).
	engine.RegisterBehavior("30015", &engine.Behavior{})

	// 30016 Making an Entrance: Hero Interrupt — your hero's basic thwart
	// gets +2 THW (until end of phase, the Skilled Strike approximation);
	// after a basic thwart that removes all threat from a scheme, heal 2.
	engine.RegisterBehavior("30016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), THW: 2}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicThwart)
			if !ok || m.Player != e.EOwner() {
				return nil
			}
			p := g.Player(e.EOwner())
			s := g.Entity(m.Target)
			// Fires only while the event's +2 THW bonus is active
			// (approximation of the "that thwart" window).
			if p == nil || p.BonusTHW <= 0 || s == nil {
				return nil
			}
			if schemeThreatOf(g, m.Target) <= m.N {
				return []engine.Message{engine.HealEntity{Target: p.ID, N: 2}}
			}
			return nil
		},
	})

	// 30017 One Way or Another: Hero Action (max 1 per round) — search the
	// encounter deck for a side scheme, reveal it, draw 3 cards, shuffle
	// the encounter deck.
	engine.RegisterBehavior("30017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			if g.UsedThisRound["30017"] {
				return nil
			}
			g.UsedThisRound["30017"] = true
			var choices []engine.Choice
			for _, c := range g.EncounterDeck {
				if c.Def().Type != "side_scheme" {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: "Reveal " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(
					engine.EncounterTakeCard{CardID: c.ID},
					engine.RevealEncounterCard{Player: pid, Card: c},
					engine.DrawCards{Player: pid, N: 3},
					engine.ShuffleEncounterDeck{},
				))
			}
			if len(choices) == 0 {
				g.Logf("One Way or Another — no side scheme in the encounter deck")
				return []engine.Message{engine.DrawCards{Player: pid, N: 3}, engine.ShuffleEncounterDeck{}}
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("One Way or Another — reveal a side scheme from the encounter deck (draw 3)", choices...),
			}}
		},
	})

	// 30018 Followed: attach to a side scheme (max 1 per scheme);
	// Interrupt — when the attached scheme is defeated, deal 4 damage to
	// an enemy.
	engine.RegisterBehavior("30018", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return len(g.SideSchemes) > 0
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			pid := e.EOwner()
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				s := g.SideSchemes[id]
				if s == nil || s.PlayerSide {
					continue
				}
				if upgradeAttachedToScheme(g, "30018", id) {
					continue // max 1 per scheme
				}
				attach := id
				upgrade := u
				choices = append(choices, engine.Choice{
					Label: "Attach to " + s.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.Code,
				}.Msgs(engine.AttachUpgrade{ID: upgrade.ID, Target: attach}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Followed — attach to a side scheme", choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || m.Scheme != u.AttachTo {
				return nil
			}
			p := g.Player(u.Owner)
			owner := p
			g.Delete(u.ID)
			if owner != nil {
				owner.Discard = append(owner.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: u.Owner})
			}
			choices := cardutil.EnemyChoices(g, 4, u.ID, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: 4, Source: u.ID}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   u.Owner,
				Question: engine.Ask("Followed — deal 4 damage to an enemy", choices...),
			}}
		},
	})

	// 30019 Overwatch: attach to a scheme (max 1 per scheme); Hero
	// Interrupt — when threat is removed from the attached scheme by a
	// thwart, discard this card → remove an equal amount of threat from a
	// different scheme.
	engine.RegisterBehavior("30019", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return len(g.Schemes()) > 0
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			pid := e.EOwner()
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				if upgradeAttachedToScheme(g, "30019", id) {
					continue // max 1 per scheme
				}
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: "Attach to " + s.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Overwatch — attach to a scheme", choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.ThwartScheme)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || m.Scheme != u.AttachTo {
				return nil
			}
			amount := m.N
			if t := schemeThreatOf(g, m.Scheme); t < amount {
				amount = t
			}
			if amount <= 0 {
				return nil
			}
			p := g.Player(u.Owner)
			owner := p
			g.Delete(u.ID)
			if owner != nil {
				owner.Discard = append(owner.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: u.Owner})
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				if id == m.Scheme {
					continue
				}
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: "Remove " + fmt.Sprint(amount) + " threat from " + s.EDef().Name,
					Kind:  engine.ChoiceTarget, SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.ThwartScheme{Scheme: id, N: amount, Source: u.Owner}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   u.Owner,
				Question: engine.Ask("Overwatch — remove "+fmt.Sprint(amount)+" threat from a different scheme", choices...),
			}}
		},
	})

	// 30020 Scarlet Spider: play only if you control a Web-Warrior card.
	// Interrupt — when you would reveal an encounter card, name a card
	// type: on a match Scarlet Spider takes 1 damage and you draw 1.
	// (Approximation: the guess question resolves while the reveal is
	// processing; players are trusted to name blind.)
	engine.RegisterBehavior("30020", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return webWarriorCardsControlled(g, p) >= 1
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.RevealEncounterCard)
			if !ok || m.Player != e.EOwner() {
				return nil
			}
			if a := g.Allies[e.EID()]; a == nil {
				return nil
			}
			var choices []engine.Choice
			for _, t := range []string{"minion", "treachery", "side_scheme", "environment", "attachment", "obligation"} {
				choices = append(choices, engine.Choice{
					ID: "guess-" + t, Label: t, Kind: engine.ChoiceLabel,
				}.Msgs(engine.GuessCheck{Player: m.Player, CardCode: m.Card.Code, Guess: t, Penalty: e.EID()}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   m.Player,
				Question: engine.Ask("Scarlet Spider — name a card type (look at that card)", choices...),
			}}
		},
	})

	// 30021 SP//dr: play only if you control a Web-Warrior card; when
	// defeated by consequential damage she returns to your hand instead.
	engine.RegisterBehavior("30021", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return webWarriorCardsControlled(g, p) >= 1
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			if !ok || m.Target != e.EID() || m.Source != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil || a.Damage+m.Damage < a.MaxHP {
				return nil
			}
			// Lethal consequential damage (ally-sourced): to hand.
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			code := a.Code
			g.Delete(a.ID)
			p.Hand = append(p.Hand, engine.Card{ID: g.NextCardID(), Code: code, Owner: p.ID})
			g.Logf("SP//dr returns to %s's hand (excess consequential damage)", p.Name)
			return nil
		},
	})

	// 30022 Team-Building Exercise: Hero Action — exhaust → the next card
	// you play this phase that shares a trait with your hero costs 1 less
	// (approximation: the discount applies to the next card; the chosen
	// card should be played via the normal play flow).
	engine.RegisterBehavior("30022", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Team-Building Exercise — your next card this phase costs 1 less", Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					if s == nil {
						return nil
					}
					return []engine.Message{engine.CostDiscountApply{Player: s.Owner, Amount: 1}}
				},
			}}
		},
	})

	// 30023 Web of Life and Destiny: free for Web-Warrior identities;
	// Response — after a Web-Warrior ally leaves play, choose a player →
	// that player draws 1 card.
	engine.RegisterBehavior("30023", &engine.Behavior{
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if g.EntityHasTrait(p.ID, "web-warrior") && def.Cost != nil {
				return *def.Cost
			}
			return 0
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyDefeated)
			if !ok {
				return nil
			}
			a := g.Allies[m.AllyID]
			if a == nil || !a.EDef().HasTrait("web-warrior") {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil {
				return nil
			}
			var choices []engine.Choice
			for _, tp := range g.Players {
				choices = append(choices, engine.Choice{
					Label: tp.Name + " draws 1", Kind: engine.ChoiceLabel,
				}.Msgs(engine.DrawCards{Player: tp.ID, N: 1}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   s.Owner,
				Question: engine.Ask("Web of Life and Destiny — a Web-Warrior ally left play: choose a player to draw 1", choices...),
			}}
		},
	})

	// 30029 Warrior of the Great Web: attach to a character with "Spider"
	// in its title; attached character gains Web-Warrior; Response — after
	// a Web-Warrior ally leaves play, attached character gets +1 ATK
	// until end of phase.
	engine.RegisterBehavior("30029", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			hasSpiderTitle := func(name string) bool {
				return strings.Contains(strings.ToLower(name), "spider")
			}
			var choices []engine.Choice
			if hasSpiderTitle(p.HeroDef().Name) || hasSpiderTitle(p.AlterEgoDef().Name) {
				if !warriorAttachedTo(g, p.ID) {
					choices = append(choices, engine.Choice{
						Label: "Attach to " + p.HeroDef().Name, Kind: engine.ChoiceTarget, SourceID: p.ID,
					}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: p.ID, GrantTrait: "web-warrior"}))
				}
			}
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil || !hasSpiderTitle(a.EDef().Name) || warriorAttachedTo(g, id) {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: "Attach to " + a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id, GrantTrait: "web-warrior"}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Warrior of the Great Web — attach to a character with \"Spider\" in its title", choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyDefeated)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil {
				return nil
			}
			a := g.Allies[m.AllyID]
			if a == nil || !a.EDef().HasTrait("web-warrior") {
				return nil
			}
			if u.AttachTo == "" {
				return nil
			}
			if u.AttachTo.Is(engine.KindPlayer) {
				return []engine.Message{engine.ApplyStatBonus{Target: u.AttachTo, ATK: 1}}
			}
			return []engine.Message{engine.AllyStatBonus{Ally: u.AttachTo, ATK: 1}}
		},
	})
}

func registerWoLANDScheme() {
	// 30030 Hunting the Spider-Totems: Forced Interrupt — when the villain
	// phase begins, discard the top 3 cards of the encounter deck; each
	// Inheritor minion discarded this way enters play engaged with a
	// player who controls a Web-Warrior character, if able (otherwise the
	// first player).
	engine.RegisterBehavior("30030", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			bp, ok := msg.(engine.BeginPhase)
			if !ok || bp.Phase != engine.PhaseVillain {
				return nil
			}
			var msgs []engine.Message
			for i := 0; i < 3; i++ {
				if len(g.EncounterDeck) == 0 {
					if len(g.EncounterDiscard) == 0 {
						break
					}
					g.EncounterDeck = g.EncounterDiscard
					g.EncounterDiscard = nil
					for j := len(g.EncounterDeck) - 1; j > 0; j-- {
						k := g.Random(j + 1)
						g.EncounterDeck[j], g.EncounterDeck[k] = g.EncounterDeck[k], g.EncounterDeck[j]
					}
				}
				top := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				def := top.Def()
				if def.Type == "minion" && def.HasTrait("inheritor") {
					pid := cardutil.FirstPlayerID(g)
					for _, p := range g.Players {
						if webWarriorCardsControlled(g, p) > 0 {
							pid = p.ID
							break
						}
					}
					hp := 1
					if def.HP != nil {
						hp = *def.HP
					}
					atk, sch := 0, 0
					if def.Attack != nil {
						atk = *def.Attack
					}
					if def.Scheme != nil {
						sch = *def.Scheme
					}
					mn := &engine.Minion{
						ID: g.NextEntityID(engine.KindMinion), Code: def.Code,
						MaxHP: hp, AttackVal: atk, SchemeVal: sch, EngagedWith: pid,
					}
					g.Minions[mn.ID] = mn
					g.Logf("Hunting the Spider-Totems — %s enters play engaged with %s", def.Name, g.Player(pid).Name)
					msgs = append(msgs, engine.MinionEntersPlay{MinionID: mn.ID, Player: pid})
				} else {
					g.EncounterDiscard = append(g.EncounterDiscard, top)
				}
			}
			return msgs
		},
	})
}

// registerInheritors installs the eight Inheritor minions (30031–30038).
// The "each Inheritor minion gains X" auras resolve through engine checks
// (thwartBlockerName, guardBlocksVillain, retaliateOf, minion activation);
// stalwart and overkill/piercing auras are not modeled.
func registerInheritors() {
	// inheritorReveal builds the common "When Revealed: if a Web-Warrior
	// character is in play, …" guard.
	inheritorReveal := func(g *engine.Game, e engine.Entity, msg engine.Message, effect func(g *engine.Game, pid engine.PlayerID) []engine.Message) []engine.Message {
		m, ok := msg.(engine.MinionEntersPlay)
		if !ok || m.MinionID != e.EID() {
			return nil
		}
		if !webWarriorCharacterInPlay(g) {
			g.Logf("%s finds no spider-totem to hunt (no Web-Warrior in play)", e.EDef().Name)
			return nil
		}
		return effect(g, m.Player)
	}

	// 30031 Bora: each Inheritor minion gains 1 acceleration icon (the
	// villain-phase threat step adds 1 per Inheritor, approximated here).
	engine.RegisterBehavior("30031", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			switch m := msg.(type) {
			case engine.BeginPhase:
				if m.Phase != engine.PhaseVillain || g.MainScheme == nil {
					return nil
				}
				n := 0
				for _, mn := range g.Minions {
					if mn != nil && mn.EDef().HasTrait("inheritor") {
						n++
					}
				}
				if n > 0 {
					return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: e.EID()}}
				}
			case engine.MinionEntersPlay:
				return inheritorReveal(g, e, msg, func(g *engine.Game, pid engine.PlayerID) []engine.Message {
					var msgs []engine.Message
					for _, id := range g.Schemes() {
						msgs = append(msgs, engine.SchemeThreat{Scheme: id, N: 1, Source: e.EID()})
					}
					return msgs
				})
			}
			return nil
		},
	})

	// 30032 Brix: each Inheritor minion gains patrol
	// (thwartBlockerName); When Revealed — 2 threat on the main scheme.
	engine.RegisterBehavior("30032", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionEntersPlay); ok {
				return inheritorReveal(g, e, msg, func(g *engine.Game, pid engine.PlayerID) []engine.Message {
					if g.MainScheme == nil {
						return nil
					}
					return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()}}
				})
			}
			return nil
		},
	})

	// 30033 Daemos: each Inheritor minion gains stalwart (not modeled);
	// When Revealed — stun a character the revealing player controls.
	engine.RegisterBehavior("30033", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			return inheritorReveal(g, e, msg, func(g *engine.Game, pid engine.PlayerID) []engine.Message {
				p := g.Player(pid)
				if p == nil {
					return nil
				}
				var choices []engine.Choice
				choices = append(choices, engine.Choice{
					Label: "Stun " + p.Name, Kind: engine.ChoiceTarget, SourceID: p.ID,
				}.Msgs(engine.StunEntity{Target: p.ID}))
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil {
						choices = append(choices, engine.Choice{
							Label: "Stun " + a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
						}.Msgs(engine.StunEntity{Target: id}))
					}
				}
				return []engine.Message{engine.AskQuestion{
					Player:   pid,
					Question: engine.Ask("Daemos — stun a character you control", choices...),
				}}
			})
		},
	})

	// 30034 Jennix: each Inheritor minion gains guard
	// (guardBlocksVillain); When Revealed — Jennix gains a tough status
	// card.
	engine.RegisterBehavior("30034", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionEntersPlay); ok {
				return inheritorReveal(g, e, msg, func(g *engine.Game, pid engine.PlayerID) []engine.Message {
					return []engine.Message{engine.ToughEntity{Target: e.EID()}}
				})
			}
			return nil
		},
	})

	// 30035 Karn: each Inheritor minion's attacks gain overkill and
	// piercing (not modeled); When Revealed — discard an upgrade or
	// support the revealing player controls.
	engine.RegisterBehavior("30035", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			return inheritorReveal(g, e, msg, func(g *engine.Game, pid engine.PlayerID) []engine.Message {
				p := g.Player(pid)
				if p == nil {
					return nil
				}
				var choices []engine.Choice
				for _, id := range cardutil.SortedIDs(g.Supports) {
					if s := g.Supports[id]; s != nil && s.Owner == pid {
						choices = append(choices, engine.Choice{
							Label: "Discard " + s.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.Code,
						}.Msgs(engine.DiscardControlled{Player: pid, ID: id}))
					}
				}
				for _, id := range cardutil.SortedIDs(g.Upgrades) {
					if u := g.Upgrades[id]; u != nil && u.Owner == pid {
						choices = append(choices, engine.Choice{
							Label: "Discard " + u.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: u.Code,
						}.Msgs(engine.DiscardControlled{Player: pid, ID: id}))
					}
				}
				if len(choices) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{
					Player:   pid,
					Question: engine.Ask("Karn — discard an upgrade or support you control", choices...),
				}}
			})
		},
	})

	// 30036 Morlun: each Inheritor minion gets +1 ATK (applied on entry
	// while Morlun is in play, BoostEnemyAttack); When Revealed — take 2
	// damage.
	engine.RegisterBehavior("30036", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			switch m := msg.(type) {
			case engine.MinionEntersPlay:
				if m.MinionID == e.EID() {
					return inheritorReveal(g, e, msg, func(g *engine.Game, pid engine.PlayerID) []engine.Message {
						var msgs []engine.Message
						for _, id := range cardutil.SortedIDs(g.Minions) {
							mn := g.Minions[id]
							if mn != nil && mn.ID != e.EID() && mn.EDef().HasTrait("inheritor") {
								msgs = append(msgs, engine.BoostEnemyAttack{Enemy: id, N: 1})
							}
						}
						return append(msgs, engine.DamageEntity{Target: pid, Damage: 2, Source: e.EID()})
					})
				}
				mn := g.Minions[m.MinionID]
				if mn != nil && mn.EDef().HasTrait("inheritor") {
					return []engine.Message{engine.BoostEnemyAttack{Enemy: m.MinionID, N: 1}}
				}
			}
			return nil
		},
	})

	// 30037 Solus: each Inheritor minion gains villainous (minion
	// activation); When Revealed — give Solus a facedown boost card
	// (approximation: +1 boost icon on his next attack).
	engine.RegisterBehavior("30037", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionEntersPlay); ok {
				return inheritorReveal(g, e, msg, func(g *engine.Game, pid engine.PlayerID) []engine.Message {
					if mn := g.Minions[e.EID()]; mn != nil {
						mn.BoostCount++
						g.Logf("Solus gains a facedown boost card")
					}
					return nil
				})
			}
			return nil
		},
	})

	// 30038 Verna: each Inheritor minion gains retaliate 1 (retaliateOf);
	// When Revealed — deal 1 damage to each character the revealing
	// player controls.
	engine.RegisterBehavior("30038", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			return inheritorReveal(g, e, msg, func(g *engine.Game, pid engine.PlayerID) []engine.Message {
				p := g.Player(pid)
				if p == nil {
					return nil
				}
				var msgs []engine.Message
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: e.EID()})
				for _, id := range p.Allies {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: e.EID()})
				}
				return msgs
			})
		},
	})
}

// schemeThreatOf reads a scheme's current threat (0 for unknown ids).
func schemeThreatOf(g *engine.Game, id engine.EntityID) int {
	if s := g.SideSchemes[id]; s != nil {
		return s.Threat
	}
	if g.MainScheme != nil && g.MainScheme.ID == id {
		return g.MainScheme.Threat
	}
	return 0
}

// upgradeAttachedToScheme reports whether an upgrade with the given code
// is attached to the given scheme.
func upgradeAttachedToScheme(g *engine.Game, code string, scheme engine.EntityID) bool {
	for _, u := range g.Upgrades {
		if u != nil && u.Code == code && u.AttachTo == scheme {
			return true
		}
	}
	return false
}

// warriorAttachedTo reports whether a Warrior of the Great Web (30029) is
// attached to the character.
func warriorAttachedTo(g *engine.Game, target engine.EntityID) bool {
	for _, u := range g.Upgrades {
		if u != nil && u.Code == "30029" && u.AttachTo == target {
			return true
		}
	}
	return false
}
