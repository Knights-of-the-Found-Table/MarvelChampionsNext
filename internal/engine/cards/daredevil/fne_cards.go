package daredevil

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// registerFNECards installs the remaining Fear No Evil box cards
// (60019-60032 and the basic resources 60057-60059).
func registerFNECards() {
	registerBlindspot()
	registerCloak()
	registerDagger()
	registerGhostRider()
	registerKnowYourEnemy()
	registerDeEscalation()
	registerChanceEncounter()
	registerLegalTrouble()
	registerMoveInShadow()
	registerStealthTraining()
	registerStick()
	registerDanceWithDevil()
	registerSensoryOverload()
	registerFNEResources()
}

// 60019 Blindspot: after Blindspot thwarts, confuse an enemy with an
// upgrade attached.
func registerBlindspot() {
	engine.RegisterBehavior("60019", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyThwartWindow)
			a, ok2 := e.(*engine.Ally)
			if !ok || !ok2 || m.Ally != a.ID {
				return nil
			}
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if upgradesAttachedTo(g, id) == 0 {
					continue
				}
				enemy := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: enemy.ECode(),
				}.Msgs(engine.ConfuseEntity{Target: id}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.blindspotConfuseAnEnemyWithAnUpgradeAttached"), append(choices, cardutil.Skip())...),
			}}
		},
	})
}

// 60020 Cloak: action — exhaust and spend 2 [energy] resources → find
// Dagger and put her into play.
func registerCloak() {
	engine.RegisterBehavior("60020", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			a, ok := e.(*engine.Ally)
			if !ok {
				return nil
			}
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			hasDagger := false
			for _, c := range append(append(engine.CardList{}, p.Deck...), p.Discard...) {
				if c.Code == "60021" {
					hasDagger = true
					break
				}
			}
			if !hasDagger {
				return nil
			}
			return []engine.Ability{{
				Label:     engine.Tf("c.cloakExhaustAndSpend2EnergyFindDagger"),
				Type:      engine.AbilityAction,
				Exhaust:   true,
				Cost:      2,
				CostIcons: "energy:2",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(a.Owner)
					if p == nil {
						return nil
					}
					// Move Dagger from deck/discard to hand, then field her.
					for i, c := range p.Deck {
						if c.Code != "60021" {
							continue
						}
						p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
						p.Hand = append(p.Hand, c)
						return []engine.Message{
							engine.ShufflePlayerDeck{Player: p.ID},
							engine.AllyEntersPlayFree{Player: p.ID, Card: c},
						}
					}
					for _, c := range p.Discard {
						if c.Code == "60021" {
							return []engine.Message{engine.AllyEntersPlayFree{Player: p.ID, Card: c, FromOwner: p.ID}}
						}
					}
					return nil
				},
			}}
		},
	})
}

// 60021 Dagger: the Cloak riders are print-only in this data (Cloak has no
// acceleration icon; the -1 consequential with Cloak in play is not
// modeled).
func registerDagger() {
	engine.RegisterBehavior("60021", &engine.Behavior{})
}

// 60022 Ghost Rider: when he attacks an enemy, spend a [energy] resource
// (2 if the target is the villain) → confuse that enemy (approximation:
// the energy icon is checked in hand, the discard happens on confirm).
func registerGhostRider() {
	engine.RegisterBehavior("60022", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			a, ok2 := e.(*engine.Ally)
			if !ok || !ok2 || m.Ally != a.ID {
				return nil
			}
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			need := 1
			target := "enemy"
			if g.Villains[m.Target] != nil {
				need = 2
				target = "villain"
			}
			var choices []engine.Choice
			for _, c := range p.Hand {
				hasEnergy := false
				for _, r := range c.Def().Resources {
					if r == "energy" || r == "wild" {
						hasEnergy = true
					}
				}
				if !hasEnergy {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: engine.Tf("c.spendEnergyNeeded", c, need),
					Kind:  engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(
					engine.ConsumeHandCard{Player: p.ID, CardID: c.ID},
					engine.ConfuseEntity{Target: m.Target},
				))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask(engine.Tf("c.ghostRiderSpendEnergyToConfuseThe", need, target),
					append(choices, cardutil.Skip())...),
			}}
		},
	})
}

// traitDiscount is the "Discount 1 ([[Trait]])" printed rider.
func traitDiscount(traits ...string) func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
	return func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
		for _, t := range traits {
			if g.EntityHasTrait(p.ID, t) {
				return 1
			}
		}
		return 0
	}
}

// 60023 Know Your Enemy: Discount 1 (Martial Artist); remove 1 threat from
// a scheme, twice.
func registerKnowYourEnemy() {
	schemePick := func(g *engine.Game, e engine.Entity) []engine.Message {
		pid := e.EOwner()
		var choices []engine.Choice
		for _, id := range g.Schemes() {
			s := g.Entity(id)
			choices = append(choices, engine.Choice{
				Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget,
				SourceID: id, CardCode: s.ECode(),
			}.Msgs(engine.ThwartScheme{Scheme: id, N: 1, Source: pid}))
		}
		if len(choices) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: pid, Question: engine.Ask(engine.Tf("c.knowYourEnemyRemove1Threat"), choices...)}}
	}
	engine.RegisterBehavior("60023", &engine.Behavior{
		CardCost: traitDiscount("martial artist"),
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			first := schemePick(g, e)
			if first == nil {
				return nil
			}
			// second pick runs after the first resolves
			second := schemePick(g, e)
			return append(first, second...)
		},
	})
}

// 60024 De-escalation: when defeated, remove an acceleration token from
// play (tokens live on the main scheme in this engine).
func registerDeEscalation() {
	engine.RegisterBehavior("60024", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			if g.MainScheme != nil && g.MainScheme.AccelerationTokens > 0 {
				g.MainScheme.AccelerationTokens--
				g.TLogf("c.deEscalationRemovesAnAccelerationTokenFrom", g.MainScheme)
			} else {
				g.TLogf("c.deEscalationFindsNoAccelerationTokenInPlay")
			}
			return nil
		},
	})
}

// 60025 Chance Encounter: attach to a side scheme; when it is defeated,
// search your deck and discard pile for an ally → your hand (shuffle).
func registerChanceEncounter() {
	engine.RegisterBehavior("60025", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				if g.MainScheme != nil && id == g.MainScheme.ID {
					continue // side schemes only
				}
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.chanceEncounterAttachToWhichSideScheme"), choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			u, ok2 := e.(*engine.Upgrade)
			if !ok || !ok2 || m.Scheme != u.AttachTo {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			// Search approximation (established convention): auto-take
			// the first ally from the discard pile, else the first from
			// the deck, then shuffle.
			var out []engine.Message
			found := false
			for _, c := range p.Discard {
				if c.Def().Type == "ally" {
					out = append(out, engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID})
					found = true
					break
				}
			}
			if !found {
				for i, c := range p.Deck {
					if c.Def().Type == "ally" {
						p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
						p.Hand = append(p.Hand, c)
						out = append(out, engine.ShufflePlayerDeck{Player: p.ID})
						found = true
						break
					}
				}
			}
			if found {
				g.TLogf("c.chanceEncounterTakesAnAllyToHand", p.Name)
			}
			return append(out, engine.DiscardControlled{Player: u.Owner, ID: u.ID})
		},
	})
}

// 60026 Legal Trouble: Discount 1 (Attorney or Police); attach to a
// minion, which gets -2 SCH.
func registerLegalTrouble() {
	engine.RegisterBehavior("60026", &engine.Behavior{
		CardCost:               traitDiscount("attorney", "police"),
		AttachedEnemySchemeMod: -2,
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if g.Minions[id] == nil {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(g.Entity(id)), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: g.Entity(id).ECode(),
				}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.legalTroubleAttachToWhichMinion"), choices...),
			}}
		},
	})
}

// 60027 Move in Shadow: Discount 1 (Martial Artist or Spy); Temporary;
// response — after you play a card (including this one), remove 1 threat
// from a scheme (approximation: rides the EventPlayed, AllyEnteredPlay and
// effect-fielded UpgradeEnterPlay windows; normally-played upgrades and
// supports have no announce message; the free threat removal auto-targets
// the first threatened scheme).
func registerMoveInShadow() {
	removeOne := func(g *engine.Game, pid engine.PlayerID) {
		for _, id := range g.Schemes() {
			s := g.SideSchemes[id]
			if s != nil && s.Threat > 0 && !s.PlayerSide {
				g.Push(engine.ThwartScheme{Scheme: id, N: 1, Source: pid})
				return
			}
			if g.MainScheme != nil && id == g.MainScheme.ID && g.MainScheme.Threat > 0 {
				g.Push(engine.ThwartScheme{Scheme: id, N: 1, Source: pid})
				return
			}
		}
	}
	engine.RegisterBehavior("60027", &engine.Behavior{
		CardCost: traitDiscount("martial artist", "spy"),
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u, ok := e.(*engine.Upgrade)
			if !ok {
				return nil
			}
			switch m := msg.(type) {
			case engine.EventPlayed:
				if m.Player != u.Owner {
					return nil
				}
			case engine.UpgradeEnterPlay:
				if m.Player != u.Owner {
					return nil
				}
			case engine.AllyEnteredPlay:
				if m.Player != u.Owner {
					return nil
				}
			case engine.EndRound:
				return []engine.Message{engine.ReturnControlled{Player: u.Owner, ID: u.ID}}
			default:
				return nil
			}
			removeOne(g, u.Owner)
			return nil
		},
	})
}

// 60028 Stealth Training: play under any player's control (approximation:
// owner only, like Heroic Conditioning); response — after you thwart and
// exactly defeat a side scheme, exhaust → stun an enemy.
func registerStealthTraining() {
	engine.RegisterBehavior("60028", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			u, ok2 := e.(*engine.Upgrade)
			if !ok || !ok2 || u.Exhausted {
				return nil
			}
			if s := g.SideSchemes[m.Scheme]; s == nil || s.PlayerSide {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			choices := cardutil.EnemyChoices(g, 0, p.ID, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.StunEntity{Target: target}}
			})
			if len(choices) == 0 {
				return nil
			}
			use := engine.Choice{
				ID: "use", Label: engine.Tf("c.exhaustStealthTrainingStunAnEnemy"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.ExhaustEntity{ID: u.ID}).WithThen(
				engine.Ask(engine.Tf("c.stealthTrainingStunWhichEnemy"), choices...))
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.stealthTrainingStunAnEnemy"), use, cardutil.Skip()),
			}}
		},
	})
}

// 60029 Stick: when a friendly Martial Artist character uses a basic power,
// exhaust Stick → +1 to that power (approximation: rides the thwart and
// ally-attack windows; the -1-for-ready trade is not modeled).
func registerStick() {
	engine.RegisterBehavior("60029", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			s, ok := e.(*engine.Support)
			if !ok || s.Exhausted {
				return nil
			}
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			var extra []engine.Message
			switch m := msg.(type) {
			case engine.WindowAfterThwarted:
				if m.Player != s.Owner || !g.EntityHasTrait(p.ID, "martial artist") {
					return nil
				}
				extra = []engine.Message{engine.ThwartScheme{Scheme: m.Scheme, N: 1, Source: s.Owner}}
			case engine.AllyThwartWindow:
				a := g.Allies[m.Ally]
				if a == nil || a.Owner != s.Owner || !a.EDef().HasTrait("martial artist") {
					return nil
				}
				extra = []engine.Message{engine.ThwartScheme{Scheme: m.Scheme, N: 1, Source: s.Owner}}
			case engine.AllyAttackWindow:
				a := g.Allies[m.Ally]
				if a == nil || a.Owner != s.Owner || !a.EDef().HasTrait("martial artist") {
					return nil
				}
				extra = []engine.Message{engine.DamageEntity{Target: m.Target, Damage: 1, Source: s.Owner}}
			default:
				return nil
			}
			use := engine.Choice{
				ID: "use", Label: engine.Tf("c.exhaustStick1ToThatPower"), Kind: engine.ChoiceLabel,
			}.Msgs(append([]engine.Message{engine.ExhaustEntity{ID: s.ID}}, extra...)...)
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.stick1ToThatBasicPower"), use, cardutil.Skip()),
			}}
		},
	})
}

// 60031 Dance with the Devil: attach to an enemy; hero action (attack) —
// discard → deal 3 damage to attached enemy.
func registerDanceWithDevil() {
	engine.RegisterBehavior("60031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(g.Entity(id)), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: g.Entity(id).ECode(),
				}.Msgs(engine.AttachUpgrade{ID: e.EID(), Target: id}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.danceWithTheDevilAttachToWhichEnemy"), choices...),
			}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u, ok := e.(*engine.Upgrade)
			if !ok || u.AttachTo == "" {
				return nil
			}
			return []engine.Ability{{
				Label:    engine.Tf("c.danceWithTheDevilDiscardDeal3DamageToAttachedEnemy"),
				Type:     engine.AbilityAction,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.DiscardControlled{Player: u.Owner, ID: u.ID},
						engine.DamageEntity{Target: u.AttachTo, Damage: 3, Source: u.Owner},
					}
				},
			}}
		},
	})
}

// 60032 Sensory Overload: obligation — after a Sense upgrade enters play,
// take 1 damage (approximation: obligations cannot persist in play, so the
// damage is dealt once per Sense upgrade in play when revealed; the
// alter-ego recover discard rider is not modeled).
func registerSensoryOverload() {
	engine.RegisterBehavior("60032", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			n := 0
			for _, uid := range p.Upgrades {
				if u := g.Upgrades[uid]; u != nil && u.EDef().HasTrait("sense") {
					n++
				}
			}
			msgs := []engine.Message{}
			if n > 0 {
				g.TLogf("c.sensoryOverloadTakesDamageSenseUpgradesInPlay", p.Name, n, n)
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: n, Source: engine.EntityID("obligation")})
			}
			msgs = append(msgs, engine.ObligationResolve{Player: p.ID, Card: card})
			return msgs
		},
	})
}

// 60057-60059 basic resources (deckbuilding limit only).
func registerFNEResources() {
	engine.RegisterBehavior("60057", &engine.Behavior{})
	engine.RegisterBehavior("60058", &engine.Behavior{})
	engine.RegisterBehavior("60059", &engine.Behavior{})
}
