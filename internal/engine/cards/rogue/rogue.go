// Package rogue registers the Rogue hero pack (38001): the
// Rogue / Anna Marie identity built around the Touched upgrade
// (Skin Contact), the signature cards, the Deadly Touch obligation
// and the Mystique nemesis set.
//
// The engine has no set-aside zone; Touched's set-aside state is
// modeled with the player's side-deck slot (SenseDeck), the same slot
// Echo uses for tucked cards.
package rogue

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerRogue()
	registerTouched()
	registerSignatures()
	registerNemesis()
	registerObligation()
}

// touchedCode is the Touched upgrade's card code.
const touchedCode = "38002"

// touchedInPlay returns Rogue's in-play Touched upgrade, if any.
func touchedInPlay(g *engine.Game, p *engine.Player) *engine.Upgrade {
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == touchedCode {
			return u
		}
	}
	return nil
}

// bringTouchedIntoPlay finds Touched in the set-aside slot, hand, deck
// or discard pile and puts it into play under Rogue's control (direct
// state mutation: UpgradeEnterPlay does not know the set-aside slot).
func bringTouchedIntoPlay(g *engine.Game, p *engine.Player) *engine.Upgrade {
	if u := touchedInPlay(g, p); u != nil {
		return u
	}
	zones := []*engine.CardList{&p.SenseDeck, &p.Hand, &p.Deck, &p.Discard}
	for _, z := range zones {
		for _, c := range *z {
			if c.Code != touchedCode {
				continue
			}
			z.Remove(c.ID)
			u := &engine.Upgrade{
				ID:    g.NextEntityID(engine.KindUpgrade),
				Code:  touchedCode,
				Owner: p.ID,
			}
			g.Upgrades[u.ID] = u
			p.Upgrades = append(p.Upgrades, u.ID)
			g.TLogf("c.bringsTouchedIntoPlay", p.Name)
			return u
		}
	}
	return nil
}

// setTouchedAside finds Touched anywhere (play, hand, deck, discard)
// and moves it to the set-aside slot, clearing the copied traits.
func setTouchedAside(g *engine.Game, p *engine.Player) {
	if u := touchedInPlay(g, p); u != nil {
		g.Delete(u.ID)
		p.SenseDeck = append(p.SenseDeck, engine.Card{ID: g.NextCardID(), Code: touchedCode, Owner: p.ID})
		p.ExtraTraits = nil
		g.TLogf("c.touchedIsSetAside")
		return
	}
	for _, z := range []*engine.CardList{&p.Hand, &p.Deck, &p.Discard} {
		for _, c := range *z {
			if c.Code == touchedCode {
				z.Remove(c.ID)
				p.SenseDeck = append(p.SenseDeck, c)
				p.ExtraTraits = nil
				g.TLogf("c.touchedIsSetAside")
				return
			}
		}
	}
}

// syncTouchTraits copies the traits of Touched's attached character to
// Rogue (Skin Contact / Energy Transfer: "you gain each of the
// attached character's traits"). Recomputed on each attach; cleared
// when Touched leaves play.
func syncTouchTraits(g *engine.Game, p *engine.Player) {
	p.ExtraTraits = nil
	u := touchedInPlay(g, p)
	if u == nil || u.AttachTo == "" {
		return
	}
	tgt := g.Entity(u.AttachTo)
	if tgt == nil {
		return
	}
	p.ExtraTraits = append(p.ExtraTraits, tgt.EDef().Traits...)
}

// touchTargetKind reports "villain" | "minion" | "ally" | "hero" for
// Touched's attached character, "" when unattached.
func touchTargetKind(g *engine.Game, p *engine.Player) string {
	u := touchedInPlay(g, p)
	if u == nil || u.AttachTo == "" {
		return ""
	}
	switch g.Entity(u.AttachTo).(type) {
	case *engine.Villain:
		return "villain"
	case *engine.Minion:
		return "minion"
	case *engine.Ally:
		return "ally"
	case *engine.Player:
		return "hero"
	}
	return ""
}

// attachQuestion builds the "attach Touched to another character"
// prompt over every character in play except Rogue herself.
func attachQuestion(g *engine.Game, p *engine.Player, u *engine.Upgrade, extra ...engine.Message) *engine.Question {
	var choices []engine.Choice
	add := func(id engine.EntityID, name, code string) {
		msgs := append([]engine.Message{engine.AttachUpgrade{ID: u.ID, Target: id}}, extra...)
		choices = append(choices, engine.Choice{
			Label: engine.S(name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: code,
		}.Msgs(msgs...))
	}
	for _, id := range cardutil.SortedIDs(g.Villains) {
		if v := g.Villains[id]; v != nil {
			add(id, v.EDef().Name, v.Code)
		}
	}
	for _, id := range cardutil.SortedIDs(g.Minions) {
		if mn := g.Minions[id]; mn != nil {
			add(id, mn.EDef().Name, mn.Code)
		}
	}
	for _, q := range g.Players {
		for _, id := range q.Allies {
			if a := g.Allies[id]; a != nil {
				add(id, a.EDef().Name+" ("+q.Name+")", a.Code)
			}
		}
		if q.ID != p.ID {
			add(q.ID, q.Name+" (identity)", q.ECode())
		}
	}
	if len(choices) == 0 {
		return nil
	}
	return engine.Ask(engine.Tf("c.attachTouchedToWhichCharacter"), choices...)
}

// registerRogue installs the Rogue / Anna Marie identity (38001a/b).
func registerRogue() {
	engine.RegisterBehavior("38001", &engine.Behavior{
		// Setup: set your Touched upgrade aside.
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			setTouchedAside(g, p)
			return nil
		},
		// Forced Response — after the player phase begins, find Touched
		// and set it aside. Also clears the copied traits at the end of
		// the round (Skin Contact lasts until then).
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.BeginPhase:
				if m.Phase == engine.PhasePlayer {
					setTouchedAside(g, p)
				}
			case engine.EndRound:
				// Skin Contact's trait copy expires; Touched's own
				// set-aside runs at the next player phase.
				p.ExtraTraits = nil
				if u := touchedInPlay(g, p); u != nil && u.AttachTo != "" {
					syncTouchTraits(g, p)
				}
			}
			return nil
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				// Skin Contact — Action: attach Touched to another
				// character; gain its traits until the end of the round.
				Label:        engine.Tf("c.skinContactAttachTouchedToAnotherCharacter"),
				Type:         engine.AbilityAction,
				HeroOnly:     true,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(self)
					if pl == nil {
						return nil
					}
					u := bringTouchedIntoPlay(g, pl)
					if u == nil {
						g.TLogf("c.skinContactTouchedIsNowhereToBeFound")
						return nil
					}
					q := attachQuestion(g, pl, u)
					if q == nil {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: pl.ID, Question: q}}
				},
			}}
		},
	})
}

// registerTouched installs the Touched upgrade (38002). The printed
// bonuses by attached type: minion → Rogue's attacks gain overkill
// (not modeled); villain → retaliate 1 (modeled via IdentityStats);
// ally → the Aerial trait and hero → stalwart are read by the
// conditional signature events (Goin' Rogue, Southern Cross) directly
// from the attached target's kind.
func registerTouched() {
	engine.RegisterBehavior(touchedCode, &engine.Behavior{
		// Rogue gains retaliate 1 while Touched is attached to the
		// villain (game-state-aware stat hook).
		IdentityStatsG: func(g *engine.Game, p *engine.Player, u *engine.Upgrade) engine.StatBonus {
			if u.AttachTo == "" {
				return engine.StatBonus{}
			}
			if v, ok := g.Entity(u.AttachTo).(*engine.Villain); ok && v != nil {
				return engine.StatBonus{Retaliate: 1}
			}
			return engine.StatBonus{}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AttachUpgrade)
			if !ok || m.ID != e.EID() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			if p := g.Player(u.Owner); p != nil {
				syncTouchTraits(g, p)
			}
			return nil
		},
	})
}

// registerSignatures installs Rogue's signature cards.
func registerSignatures() {
	registerGambitAlly()
	registerRoguesJacket()
	registerGoinRogue()
	registerSouthernCross()
	registerEnergyTransfer()
	registerBulletproofBelle()
	registerSuperpowerAdaptation()
}

// 38003 Gambit (ally): enters play with 3 charge counters. Interrupt —
// when Gambit attacks, remove 1 charge counter → deal 1 damage to an
// enemy. (The interrupt auto-fires at the attack window while counters
// remain; target = the attacked enemy.)
func registerGambitAlly() {
	engine.RegisterBehavior("38003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyAttackWindow)
			if !ok || w.Ally != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil || a.Counters <= 0 {
				return nil
			}
			g.TLogf("c.gambitRemovesAChargeCounter1Damage")
			return []engine.Message{
				engine.AddEntityCounter{ID: a.ID, N: -1},
				engine.DamageEntity{Target: w.Target, Damage: 1, Source: a.ID},
			}
		},
	})
}

// 38004 Rogue's Jacket: while Touched is attached to a friendly
// character, +1 THW; to an enemy character, +1 ATK.
func registerRoguesJacket() {
	engine.RegisterBehavior("38004", &engine.Behavior{
		IdentityStatsG: func(g *engine.Game, p *engine.Player, u *engine.Upgrade) engine.StatBonus {
			touch := touchedInPlay(g, p)
			if touch == nil || touch.AttachTo == "" {
				return engine.StatBonus{}
			}
			switch g.Entity(touch.AttachTo).(type) {
			case *engine.Player, *engine.Ally:
				return engine.StatBonus{THW: 1}
			case *engine.Villain, *engine.Minion:
				return engine.StatBonus{ATK: 1}
			}
			return engine.StatBonus{}
		},
	})
}

// 38005 Goin' Rogue: Hero Action (thwart) — remove 3 threat from a
// scheme. Aerial (Touched on an ally): +2 threat. Retaliate (Touched
// on the villain): confuse an enemy. Stalwart (Touched on a hero):
// draw 1 card.
func registerGoinRogue() {
	engine.RegisterBehavior("38005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			kind := touchTargetKind(g, p)
			n := 3
			if kind == "ally" {
				n = 5
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				msgs := []engine.Message{engine.ThwartScheme{Scheme: id, N: n, Source: pid}}
				if kind == "hero" {
					msgs = append(msgs, engine.DrawCards{Player: pid, N: 1})
				}
				ch := engine.Choice{
					Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.ECode(),
				}.Msgs(msgs...)
				if kind == "villain" {
					confuse := cardutil.EnemyChoices(g, 0, pid, func(t engine.EntityID) []engine.Message {
						return []engine.Message{engine.ConfuseEntity{Target: t}}
					})
					if len(confuse) > 0 {
						ch = ch.WithThen(engine.Ask(engine.Tf("c.goinRogueConfuseAnEnemy"), confuse...))
					}
				}
				choices = append(choices, ch)
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.goinRogueRemoveThreatFromAScheme"), choices...),
			}}
		},
	})
}

// 38006 Southern Cross: Hero Action (attack) — deal 6 damage to an
// enemy. Aerial: +2 damage. Retaliate: stun that enemy. Stalwart:
// draw 1 card.
func registerSouthernCross() {
	engine.RegisterBehavior("38006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			kind := touchTargetKind(g, p)
			n := 6
			if kind == "ally" {
				n = 8
			}
			choices := cardutil.EnemyChoices(g, n, pid, func(id engine.EntityID) []engine.Message {
				msgs := []engine.Message{engine.DamageEntity{Target: id, Damage: n, Source: pid}}
				if kind == "villain" {
					msgs = append(msgs, engine.StunEntity{Target: id})
				}
				if kind == "hero" {
					msgs = append(msgs, engine.DrawCards{Player: pid, N: 1})
				}
				return msgs
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.southernCrossDealDamageToAnEnemy"), choices...),
			}}
		},
	})
}

// 38007 Energy Transfer: Hero Action — attach Touched to a character
// other than Rogue and deal 2 damage to it → heal 2 damage from Rogue
// and ready her; gain its traits until the end of the round.
func registerEnergyTransfer() {
	engine.RegisterBehavior("38007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			u := bringTouchedIntoPlay(g, p)
			if u == nil {
				g.TLogf("c.energyTransferTouchedIsNowhereToBeFound")
				return nil
			}
			q := attachQuestion(g, p, u)
			if q == nil {
				return nil
			}
			// The 2 damage to the touched character rides each choice.
			for i := range q.Choices {
				c := q.Choices[i]
				tgt := c.SourceID
				q.Choices[i] = engine.Choice{
					Label: engine.Tf("c.takes2Damage", c.Label), Kind: c.Kind, SourceID: tgt, CardCode: c.CardCode,
				}.Msgs(
					engine.AttachUpgrade{ID: u.ID, Target: tgt},
					engine.DamageEntity{Target: tgt, Damage: 2, Source: pid},
					engine.HealEntity{Target: pid, N: 2},
					engine.ReadyEntity{ID: pid},
				)
			}
			return []engine.Message{engine.AskQuestion{Player: pid, Question: q}}
		},
	})
}

// 38008 Bulletproof Belle: Hero Interrupt (defense) — when an enemy
// with Touched attached attacks, prevent all damage from that attack
// and gain a tough status card.
func registerBulletproofBelle() {
	engine.RegisterBehavior("38008", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			if !p.IsHero() {
				return engine.Defends{}, nil, false
			}
			u := touchedInPlay(g, p)
			if u == nil || u.AttachTo != against {
				return engine.Defends{}, nil, false
			}
			d := engine.Defends{Defender: p.ID, Against: against, Undefended: true, PreventAll: true}
			return d, []engine.Message{engine.ToughEntity{Target: p.ID}}, true
		},
	})
}

// 38009 Superpower Adaptation: Hero Action — if Touched is attached to
// a friendly character, search its owner's discard pile for an event
// of the same classification (identity-specific vs aspect/basic) and
// add it to your hand.
func registerSuperpowerAdaptation() {
	engine.RegisterBehavior("38009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			u := touchedInPlay(g, p)
			if u == nil || u.AttachTo == "" {
				return nil
			}
			var owner *engine.Player
			identitySpecific := false
			switch t := g.Entity(u.AttachTo).(type) {
			case *engine.Player:
				owner = t
				identitySpecific = true
			case *engine.Ally:
				owner = g.Player(t.Owner)
			default:
				return nil // enemy target: no friendly classification
			}
			if owner == nil {
				return nil
			}
			var choices []engine.Choice
			seen := map[string]bool{}
			for _, c := range owner.Discard {
				def := c.Def()
				if def.Type != "event" || seen[c.Code] {
					continue
				}
				if (def.Aspect == "") != identitySpecific {
					continue
				}
				seen[c.Code] = true
				choices = append(choices, engine.Choice{
					Label: engine.S("Take " + def.Name), Kind: engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(engine.RecycleFromDiscard{Player: pid, From: owner.ID, CardID: c.ID}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.superpowerAdaptationTakeAnEventFromTheDiscardPile"), choices...),
			}}
		},
	})
}

// registerNemesis installs the Rogue nemesis set (rogue_nemesis):
// Mystique, Mystique's Manipulations and Misled.
func registerNemesis() {
	// 38025 Mystique: Toughness + Villainous from the data layer.
	// Forced Response — after Mystique engages you, find Misled and
	// shuffle it into your deck.
	engine.RegisterBehavior("38025", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var pid engine.PlayerID
			switch m := msg.(type) {
			case engine.MinionEntersPlay:
				if m.MinionID == e.EID() {
					pid = m.Player
				}
			case engine.EngageMinion:
				if m.MinionID == e.EID() {
					pid = m.Player
				}
			}
			if pid == "" {
				return nil
			}
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			card, ok := findMisled(g)
			if !ok {
				return nil
			}
			g.TLogf("c.mystiqueMisledIsShuffledIntoSDeck", p.Name)
			p.Deck = append(p.Deck, card)
			return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
		},
	})

	// 38026 Mystique's Manipulations: When Defeated — search the
	// encounter deck and discard pile for Misled and shuffle it into
	// your deck. (The defeating player is not on the message; the
	// nemesis-owning Rogue player stands in.)
	engine.RegisterBehavior("38026", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			p := roguePlayer(g)
			if p == nil {
				return nil
			}
			card, ok := findMisled(g)
			if !ok {
				return nil
			}
			g.TLogf("c.mystiqueSManipulationsDefeatedMisledIsShuffledIntoSDeck", p.Name)
			p.Deck = append(p.Deck, card)
			return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
		},
	})

	// 38027 Misled: When Revealed — shuffle this card into your deck;
	// it gains surge. (The "after this enters your hand, place 2
	// threat" response is not modeled.)
	engine.RegisterBehavior("38027", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			p.Deck = append(p.Deck, engine.Card{ID: g.NextCardID(), Code: "38027", Owner: p.ID})
			g.TLogf("c.misledShufflesIntoSDeckSurge", p.Name)
			return []engine.Message{
				engine.ShufflePlayerDeck{Player: p.ID},
				engine.RevealNextEncounter{Player: p.ID},
			}
		},
	})
}

// roguePlayer finds the player running the Rogue identity.
func roguePlayer(g *engine.Game) *engine.Player {
	for _, p := range g.Players {
		if p.HeroCode == "38001a" || p.AlterEgoCode == "38001b" {
			return p
		}
	}
	return nil
}

// findMisled pulls Misled out of the encounter deck or discard pile.
func findMisled(g *engine.Game) (engine.Card, bool) {
	for _, c := range g.EncounterDeck {
		if c.Code == "38027" {
			g.EncounterDeck.Remove(c.ID)
			return c, true
		}
	}
	for _, c := range g.EncounterDiscard {
		if c.Code == "38027" {
			g.EncounterDiscard.Remove(c.ID)
			return c, true
		}
	}
	return engine.Card{}, false
}

// registerObligation installs Deadly Touch (38024): if Touched is
// attached to a friendly character, deal 2 damage to it; otherwise
// place 2 threat on the main scheme. Discard either way.
func registerObligation() {
	engine.RegisterBehavior("38024", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			msgs := []engine.Message{}
			if u := touchedInPlay(g, p); u != nil && u.AttachTo != "" {
				switch g.Entity(u.AttachTo).(type) {
				case *engine.Player, *engine.Ally:
					msgs = append(msgs, engine.DamageEntity{Target: u.AttachTo, Damage: 2, Source: ""})
					g.TLogf("c.deadlyTouch2DamageToTheTouchedCharacter")
					msgs = append(msgs, engine.ObligationResolve{Player: p.ID, Card: card})
					return msgs
				}
			}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: ""})
			}
			g.TLogf("c.deadlyTouch2ThreatOnTheMainScheme")
			return append(msgs, engine.ObligationResolve{Player: p.ID, Card: card})
		},
	})
}
