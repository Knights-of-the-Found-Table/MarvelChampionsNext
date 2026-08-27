// ironspider.go implements the remaining SP//dr pack cards: the shared
// Web-Warrior player cards (31014–31024, 31029) and the Sinister Six /
// Iron Spider encounter set (31030–31037).
package spdr

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerIronSpiderPlayers()
	registerIronSpiderEncounter()
}

// printedHP reports the hero side's printed hit points.
func printedHP(p *engine.Player) int {
	if d := p.HeroDef(); d != nil && d.HP != nil {
		return *d.HP
	}
	return 0
}

func registerIronSpiderPlayers() {
	// 31014 Daredevil: Response — after he defends against an attack, move
	// 1 damage from him to the attacking enemy.
	engine.RegisterBehavior("31014", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowDefended)
			if !ok || m.Defender != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil || a.Damage <= 0 {
				return nil
			}
			a.Damage--
			g.TLogf("c.daredevilMoves1DamageToTheAttacker")
			return []engine.Message{engine.DamageEntity{Target: m.Against, Damage: 1, Source: a.ID}}
		},
	})

	// 31015 Spider-Man Noir: X equals the facedown cards attached here (max
	// 3); after you resolve a treachery, if you control another
	// Web-Warrior card, attach it facedown (X raises ATK/THW).
	engine.RegisterBehavior("31015", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.TreacheryResolve)
			if !ok || m.Cancelled {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil || m.Player != e.EOwner() || a.Counters >= 3 {
				return nil
			}
			another := false
			for _, id := range g.Player(e.EOwner()).Allies {
				if x := g.Allies[id]; x != nil && x.ID != a.ID && x.EDef().HasTrait("web-warrior") {
					another = true
				}
			}
			if g.EntityHasTrait(e.EOwner(), "web-warrior") {
				another = true
			}
			if !another {
				return nil
			}
			a.Counters++
			a.BonusATK++
			a.BonusTHW++
			g.TLogf("c.spiderManNoirTucksFacedownX", m.Card, a.Counters)
			return nil
		},
	})

	// 31016 Repurpose: Hero Action — discard a Tech upgrade you control →
	// ready your hero and give it +X (the upgrade's printed cost) to THW,
	// ATK or DEF until end of round (approximation: until end of phase,
	// the ApplyStatBonus window).
	engine.RegisterBehavior("31016", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			if !p.IsHero() {
				return false
			}
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("tech") {
					return true
				}
			}
			return false
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Upgrades {
				u := g.Upgrades[id]
				if u == nil || !u.EDef().HasTrait("tech") {
					continue
				}
				x := 0
				if d := u.EDef(); d.Cost != nil {
					x = *d.Cost
				}
				stats := engine.Ask(engine.S("Repurpose — choose a power (+"+fmt.Sprint(x)+" until end of round)"),
					engine.Choice{Label: engine.S("+THW"), Kind: engine.ChoiceLabel}.Msgs(engine.ApplyStatBonus{Target: p.ID, THW: x}),
					engine.Choice{Label: engine.S("+ATK"), Kind: engine.ChoiceLabel}.Msgs(engine.ApplyStatBonus{Target: p.ID, ATK: x}),
					engine.Choice{Label: engine.S("+DEF"), Kind: engine.ChoiceLabel}.Msgs(engine.ApplyStatBonus{Target: p.ID, DEF: x}),
				)
				choices = append(choices, engine.Choice{
					Label: engine.Tf("c.discardBonus", u, x), Kind: engine.ChoiceTarget,
					SourceID: u.ID, CardCode: u.Code,
				}.Msgs(engine.DiscardControlled{Player: p.ID, ID: u.ID}, engine.ReadyEntity{ID: p.ID}).WithThen(stats))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.repurposeDiscardATechUpgrade"), choices...),
			}}
		},
	})

	// 31017 Thwip Thwip!: deal 1 damage to a Web-Warrior character you
	// control → place a total of 2 stun status cards on up to 2 enemies.
	engine.RegisterBehavior("31017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			stunPicks := func() []engine.Choice {
				var picks []engine.Choice
				for _, id := range cardutil.SortedEnemyIDs(g) {
					picks = append(picks, engine.Choice{
						Label: engine.Tf("c.stun2", cardutil.EnemyLabel(g.Entity(id))), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.StunEntity{Target: id}))
				}
				return append(picks, cardutil.Skip())
			}
			second := engine.Ask(engine.Tf("c.thwipThwipStunASecondEnemy"), stunPicks()...)
			var choices []engine.Choice
			addPick := func(label string, pid engine.EntityID) {
				choices = append(choices, engine.Choice{
					Label: engine.S("Deal 1 damage to " + label), Kind: engine.ChoiceTarget, SourceID: pid,
				}.Msgs(engine.DamageEntity{Target: pid, Damage: 1, Source: p.ID}).WithThen(
					engine.Ask(engine.Tf("c.thwipThwipStunTheFirstEnemy"), stunPicks()...)).WithThen(second))
			}
			if p.IsHero() && g.EntityHasTrait(p.ID, "web-warrior") {
				addPick(p.Name, p.ID)
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.EDef().HasTrait("web-warrior") {
					addPick(a.EDef().Name, a.ID)
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.thwipThwipDeal1DamageToAWebWarriorYouControl"), choices...),
			}}
		},
	})

	// 31018 Energy Barrier: 3 reflection counters; when you would take
	// damage, remove 1 → prevent 1 and deal 1 damage to an enemy (the
	// engine's reflection auto-targets the attacker).
	engine.RegisterBehavior("31018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			if u.Counters <= 0 {
				return 0, 0
			}
			u.Counters--
			return 1, 1
		},
	})

	// 31019 Forcefield Generator: 6 energy counters, max 1 per player;
	// forced prevention of up to N damage, one counter per point.
	engine.RegisterBehavior("31019", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.Code == "31019" {
					return false
				}
			}
			return true
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 6}}
		},
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			if u.Counters <= 0 {
				return 0, 0
			}
			pv := n
			if pv > u.Counters {
				pv = u.Counters
			}
			u.Counters -= pv
			return pv, 0
		},
	})

	// 31020 Spider-Tingle: when you would reveal an encounter card, deal 1
	// damage to a Web-Warrior you control → if it is a treachery, cancel
	// its When Revealed and discard Spider-Tingle. Implemented through
	// the treachery interrupt window (upgrade scan); the damage auto-hits
	// the first eligible Web-Warrior.
	engine.RegisterBehavior("31020", &engine.Behavior{
		TreacheryInterrupt: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var target engine.EntityID
			if p.IsHero() && g.EntityHasTrait(p.ID, "web-warrior") {
				target = p.ID
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.EDef().HasTrait("web-warrior") {
					target = a.ID
					break
				}
			}
			if target == "" {
				return nil
			}
			// The upgrade discards itself when the interrupt fires; find
			// it by code among the player's upgrades.
			for _, id := range p.Upgrades {
				if up := g.Upgrades[id]; up != nil && up.Code == "31020" {
					return []engine.Message{
						engine.DamageEntity{Target: target, Damage: 1, Source: p.ID},
						engine.DiscardControlled{Player: p.ID, ID: up.ID},
						engine.TreacheryResolve{Player: p.ID, Card: card, Cancelled: true},
					}
				}
			}
			return nil
		},
	})

	// 31021 Spider-Ham: play only if you control a Web-Warrior card;
	// after he attacks or thwarts, discard the top encounter card — 1
	// damage to him per boost icon discarded.
	engine.RegisterBehavior("31021", &engine.Behavior{
		Playable: webWarriorRequired,
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var hit bool
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				hit = m.Ally == e.EID()
			case engine.AllyThwartWindow:
				hit = m.Ally == e.EID()
			}
			if !hit || len(g.EncounterDeck) == 0 {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			top := g.EncounterDeck[0]
			g.EncounterDeck = g.EncounterDeck[1:]
			g.EncounterDiscard = append(g.EncounterDiscard, top)
			boost := cardutil.BoostOf(top)
			if boost > 0 {
				g.TLogf("c.spiderHamDiscardsBoostAndTakesDamage", top, boost, boost)
				return []engine.Message{engine.DamageEntity{Target: a.ID, Damage: boost, Source: a.ID}}
			}
			g.TLogf("c.spiderHamDiscardsNoBoostIcons", top)
			return nil
		},
	})

	// 31022 Spider-Man: play only if you control a Web-Warrior card;
	// after he enters play, ready an upgrade you control (draw 1 if it is
	// Tech).
	engine.RegisterBehavior("31022", &engine.Behavior{
		Playable: webWarriorRequired,
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Upgrades {
				u := g.Upgrades[id]
				if u == nil {
					continue
				}
				msgs := []engine.Message{engine.ReadyEntity{ID: u.ID}}
				if u.EDef().HasTrait("tech") {
					msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
				}
				choices = append(choices, engine.Choice{
					Label: engine.Tf("c.readyName", u), Kind: engine.ChoiceTarget, SourceID: u.ID, CardCode: u.Code,
				}.Msgs(msgs...))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.spiderManReadyAnUpgradeYouControl"), choices...),
			}}
		},
	})

	// 31023 Limitless Stamina: play only with 14+ printed hit points;
	// Hero Action — ready your hero.
	engine.RegisterBehavior("31023", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return printedHP(p) >= 14
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
		},
	})

	// 31024 Unshakable: play only with 14+ printed hit points; your
	// identity gains steady (stun/confuse are cancelled by a chained
	// clear).
	engine.RegisterBehavior("31024", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return printedHP(p) >= 14
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.StunEntity:
				if m.Target == p.ID {
					g.TLogf("c.isSteadyTheStunIsCancelled", p.Name)
					return []engine.Message{engine.ClearStun{Target: p.ID}}
				}
			case engine.ConfuseEntity:
				if m.Target == p.ID {
					g.TLogf("c.isSteadyTheConfusionIsCancelled", p.Name)
					return []engine.Message{engine.ClearConfuse{Target: p.ID}}
				}
			}
			return nil
		},
	})

	// 31029 Clarity of Purpose: attach to a friendly character; Hero
	// Resource — exhaust and deal 1 damage to attached character →
	// generate a [wild] resource.
	engine.RegisterBehavior("31029", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			if u == nil || p == nil {
				return nil
			}
			var choices []engine.Choice
			if !clarityAttachedTo(g, p.ID) {
				choices = append(choices, engine.Choice{
					Label: engine.Tf("c.attachToName", p), Kind: engine.ChoiceTarget, SourceID: p.ID,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: p.ID}))
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && !clarityAttachedTo(g, id) {
					choices = append(choices, engine.Choice{
						Label: engine.Tf("c.attachToName", a), Kind: engine.ChoiceTarget, SourceID: id, CardCode: a.Code,
					}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.clarityOfPurposeAttachToAFriendlyCharacter"), choices...),
			}}
		},
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true, DamageAttached: 1},
	})
}

func registerIronSpiderEncounter() {
	// 31030 Grand Larceny: threat cannot be removed while a Criminal
	// minion is in play (enforced in removeThreat).
	engine.RegisterBehavior("31030", &engine.Behavior{})

	// 31031 Bombshell: her attack's damage splitting among the attacked
	// player's characters is not modeled; Boost — 1 indirect damage to
	// each player.
	engine.RegisterBehavior("31031", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.IndirectDamage{Player: p.ID, N: 1})
			}
			return msgs
		},
	})

	// 31032 Electro: after she engages you or activates against you,
	// attach a printed [energy] card from your hand (each gives +1 max
	// HP).
	engine.RegisterBehavior("31032", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			pid := engine.PlayerID("")
			switch m := msg.(type) {
			case engine.MinionEntersPlay:
				if m.MinionID != e.EID() {
					return nil
				}
				pid = m.Player
			case engine.MinionActivates:
				if m.MinionID != e.EID() {
					return nil
				}
				pid = m.Player
			default:
				return nil
			}
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, c := range p.Hand {
				hasEnergy := false
				for _, icon := range c.Def().Resources {
					if icon == "energy" {
						hasEnergy = true
					}
				}
				if !hasEnergy {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: engine.Tf("c.attachName", c), Kind: engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(engine.AttachHandCard{Player: p.ID, CardID: c.ID, Enemy: e.EID()}))
			}
			if len(choices) == 0 {
				g.TLogf("c.electroFindsNoEnergyCardInSHand", p.Name)
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.electroAttachACardWithAPrintedEnergyResource"), choices...),
			}}
		},
	})

	// 31033 Hobgoblin: when he would attack you, discard encounter cards
	// equal to his ATK instead; take 1 indirect damage per boost icon
	// discarded.
	engine.RegisterBehavior("31033", &engine.Behavior{
		MinionActivate: func(g *engine.Game, mn *engine.Minion, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				// Schemes normally while the target is in alter-ego form.
				if g.MainScheme != nil {
					return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: mn.SchemeVal, Source: mn.ID}}
				}
				return nil
			}
			boosts := 0
			for i := 0; i < mn.AttackVal; i++ {
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
				g.EncounterDiscard = append(g.EncounterDiscard, top)
				boosts += cardutil.BoostOf(top)
			}
			g.TLogf("c.hobgoblinDiscardsEncounterCardsTakesIndirectDamage", mn.AttackVal, p.Name, boosts)
			if boosts <= 0 {
				return nil
			}
			return []engine.Message{engine.IndirectDamage{Player: p.ID, N: boosts}}
		},
	})

	// 31034 Iron Spider: Guard / Retaliate 1 / Toughness are data
	// keywords; Patrol and overkill are not modeled.
	engine.RegisterBehavior("31034", &engine.Behavior{})

	// 31035 Sandman: after he takes damage from an attack, discard the top
	// 7 encounter cards; Boost — the same mill.
	engine.RegisterBehavior("31035", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			if !ok || m.Target != e.EID() || !m.Source.Is(engine.KindPlayer) {
				return nil
			}
			g.TLogf("c.sandmanDiscardingTheTop7EncounterCards")
			for i := 0; i < 7 && len(g.EncounterDeck) > 0; i++ {
				top := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				g.EncounterDiscard = append(g.EncounterDiscard, top)
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			g.TLogf("c.sandmanDiscardingTheTop7EncounterCards")
			for i := 0; i < 7 && len(g.EncounterDeck) > 0; i++ {
				top := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				g.EncounterDiscard = append(g.EncounterDiscard, top)
			}
			return nil
		},
	})

	// 31036 Spot: When Defeated — shuffle him into the encounter deck
	// (excess-damage condition not tracked); Boost — put him into play
	// engaged with the first player.
	engine.RegisterBehavior("31036", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			for i, c := range g.EncounterDiscard {
				if c.Code == e.ECode() {
					g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
					g.EncounterDeck = append(g.EncounterDeck, c)
					g.TLogf("c.spotShufflesBackIntoTheEncounterDeck")
					break
				}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: card}}
		},
	})

	// 31037 Surge in Crime: the "each Criminal gains surge" aura is not
	// modeled; Hero Action — with no Criminal in play, spend 2 resources
	// → discard this scheme.
	engine.RegisterBehavior("31037", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			for _, mn := range g.Minions {
				if mn != nil && mn.EDef().HasTrait("criminal") {
					return nil
				}
			}
			return []engine.Ability{{
				Label: engine.Tf("c.surgeInCrimeSpend2ResourcesDiscardThisScheme"), Type: engine.AbilityAction,
				HeroOnly: true, Cost: 2,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.SideSchemes[self]
					if s == nil {
						return nil
					}
					g.Delete(s.ID)
					g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: s.Code})
					g.TLogf("log.discarded", s)
					return nil
				},
			}}
		},
	})
}

// webWarriorRequired gates "play only if you control a Web-Warrior card".
func webWarriorRequired(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
	if g.EntityHasTrait(p.ID, "web-warrior") {
		return true
	}
	for _, id := range p.Allies {
		if a := g.Allies[id]; a != nil && a.EDef().HasTrait("web-warrior") {
			return true
		}
	}
	return false
}

// clarityAttachedTo reports whether a Clarity of Purpose (31029) is
// already attached to the character.
func clarityAttachedTo(g *engine.Game, target engine.EntityID) bool {
	for _, u := range g.Upgrades {
		if u != nil && u.Code == "31029" && u.AttachTo == target {
			return true
		}
	}
	return false
}
