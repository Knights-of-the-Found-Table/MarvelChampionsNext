// complete.go implements the remaining Wolverine pack cards (35013–35037):
// the shared Weapon/Skill suite, the Omega Red nemesis set and the Lady
// Deathstrike modular set.
package wolv

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerRemainingWolv()
}

func wolvVillain(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil {
			return v
		}
	}
	return nil
}

func registerRemainingWolv() {
	// 35013 Psylocke: 2 psionic counters; attack → remove 1: confuse and
	// 1 damage.
	engine.RegisterBehavior("35013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 2}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			a := g.Allies[e.EID()]
			if !ok || m.Ally != e.EID() || a == nil || a.Counters <= 0 {
				return nil
			}
			a.Counters--
			return []engine.Message{
				engine.ConfuseEntity{Target: m.Target},
				engine.DamageEntity{Target: m.Target, Damage: 1, Source: a.Owner},
			}
		},
	})

	// 35014 Sunfire: on enter, discard an enemy attachment with a Hero
	// Action/Response text (approximation: any attachment on an enemy).
	engine.RegisterBehavior("35014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Attachments) {
				t := g.Attachments[id]
				if t == nil || t.Target == "" || t.Target.Is(engine.KindPlayer) {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: engine.S("Discard " + t.EDef().Name), Kind: engine.ChoiceTarget, SourceID: id, CardCode: t.Code,
				}.Msgs(engine.DiscardAttachmentMsg{ID: id}))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.sunfireDiscardAnAttachmentSpendEnergy"), append(choices, cardutil.Skip())...),
			}}
		},
	})

	// 35015 Battle Fury: after your hero's basic attack would defeat a
	// minion, take 1 damage, discard this → ready.
	engine.RegisterBehavior("35015", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicAttack)
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			mn := g.Minions[m.Target]
			if !ok || u == nil || p == nil || mn == nil {
				return nil
			}
			if m.Player != p.ID || mn.HP() > p.AttackStat(g) {
				return nil
			}
			g.Delete(u.ID)
			p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			g.TLogf("c.battleFuryReadiesAfterTheTakedown", p.Name)
			return []engine.Message{
				engine.DamageEntity{Target: p.ID, Damage: 1, Source: p.ID},
				engine.ReadyEntity{ID: p.ID},
			}
		},
	})

	// 35016 Warrior Skill: 3 counters; basic attacks deal +1 (follow-up
	// damage).
	engine.RegisterBehavior("35016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicAttack)
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			if !ok || u == nil || p == nil || u.Exhausted || u.Counters <= 0 || m.Player != p.ID {
				return nil
			}
			u.Counters--
			return []engine.Message{
				engine.ExhaustEntity{ID: u.ID},
				engine.DamageEntity{Target: m.Target, Damage: 1, Source: p.ID},
			}
		},
	})

	// 35017 Outta My Way!: 3 damage (5 vs guard/patrol).
	engine.RegisterBehavior("35017", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.outtaMyWay"), func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			return 3, nil
		}),
	})

	// 35018 Precision Strike: 2 damage; heal 2 if it defeats the target.
	engine.RegisterBehavior("35018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				en := g.Entity(id)
				if en == nil {
					continue
				}
				msgs := []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: p.ID}}
				hp := 99
				switch t := en.(type) {
				case *engine.Minion:
					hp = t.HP()
				case *engine.Villain:
					hp = t.HP()
				}
				if hp <= 2 {
					msgs = append(msgs, engine.HealEntity{Target: p.ID, N: 2})
				}
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(en), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(msgs...))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.precisionStrike"), choices...),
			}}
		},
	})

	// 35019 Mean Swing: exhaust a Weapon upgrade → +3 ATK this phase
	// (Skilled Strike approximation).
	engine.RegisterBehavior("35019", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("weapon") && !u.Exhausted {
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
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("weapon") && !u.Exhausted {
					return []engine.Message{
						engine.ExhaustEntity{ID: u.ID},
						engine.ApplyStatBonus{Target: p.ID, ATK: 3},
					}
				}
			}
			return nil
		},
	})

	// 35020 Aggressive Energy: rider in handlePlayCard (32047 case).
	engine.RegisterBehavior("35020", &engine.Behavior{})

	// 35021 Colossus: costs 1 less for MUTANT/X-MEN identities.
	engine.RegisterBehavior("35021", &engine.Behavior{
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if g.EntityHasTrait(p.ID, "x-men") || g.EntityHasTrait(p.ID, "mutant") {
				return 1
			}
			return 0
		},
	})

	// 35022 Weapon X: Alter-Ego action — take 1 damage, mill to an
	// identity-specific card.
	engine.RegisterBehavior("35022", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "mutant")
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.weaponXTake1DamageDigForASignatureCard"), Type: engine.AbilityAction,
				AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.ActiveTurn)
					if p == nil {
						return nil
					}
					set := p.AlterEgoDef().CardSet
					for i, c := range p.Deck {
						if c.Def().CardSet == set && c.Def().Type != "hero" {
							p.Deck = append(p.Deck[:i], p.Deck[i+1:]...)
							c.Owner = p.ID
							p.Hand = append(p.Hand, c)
							g.TLogf("c.weaponXFinds", c)
							return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: p.ID}}
						}
						p.Discard = append(p.Discard, c)
					}
					p.Deck = nil
					return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: p.ID}}
				},
			}}
		},
	})

	// 35023 Fastball Special: deal the combined ATK of Colossus and
	// Wolverine (defaults to 6 when either is absent).
	engine.RegisterBehavior("35023", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			total := 0
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil {
					continue
				}
				if a.Code == "35021" || a.Code == "32048" || a.EDef().Name == "Wolverine" || a.EDef().Name == "Colossus" {
					total += a.AttackVal + a.BonusATK + a.PermATK
				}
			}
			if total == 0 {
				total = 6
			}
			choices := cardutil.EnemyChoices(g, total, p.ID, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: target, Damage: total, Source: p.ID}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.fastballSpecial"), choices...),
			}}
		},
	})

	// 35024–35026 basic resources.
	for _, code := range []string{"35024", "35025", "35026"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 35027 Past Demons: exhaust to remove, or stunned + confused.
	engine.RegisterBehavior("35027", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			choices := []engine.Choice{engine.Choice{
				ID: "exhaust", Label: engine.Tf("c.exhaustLoganRemoveFromTheGame"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true})}
			choices = append(choices, engine.Choice{
				ID: "suffer", Label: engine.Tf("c.youAreStunnedAndConfusedDiscard"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.StunEntity{Target: p.ID}, engine.ConfuseEntity{Target: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card}))
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.pastDemonsChoose"), choices...),
			}}
		},
	})

	// 35028 Omega Red: when he attacks you, 1 damage to each character
	// you control; he cannot be defeated while the Carbonadium
	// Synthesizer is in play.
	engine.RegisterBehavior("35028", &engine.Behavior{
		MinionDamageable: func(g *engine.Game, mn *engine.Minion, damage int) bool {
			for _, s := range g.SideSchemes {
				if s != nil && s.Code == "35029" {
					if mn.HP()-damage <= 0 {
						g.TLogf("c.omegaRedCannotBeDefeatedWhileTheCarbonadiumSynthesizerIsInPl")
						return false
					}
				}
			}
			return true
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			msgs := []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: e.EID()}}
			for _, id := range p.Allies {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: e.EID()})
			}
			return msgs
		},
	})

	// 35029 The Carbonadium Synthesizer: Omega Red cannot be defeated
	// while this scheme is in play (checked in his MinionDamageable hook
	// below).
	engine.RegisterBehavior("35029", &engine.Behavior{})

	// 35030 Death Factor: attach to your identity; 1 damage after your
	// turn; a basic recovery discards it instead of healing.
	engine.RegisterBehavior("35030", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, p := range g.Players {
				has := false
				for _, a := range g.Attachments {
					if a != nil && a.Code == "35030" && a.Target == p.ID {
						has = true
					}
				}
				if !has {
					t.Target = p.ID
					return nil
				}
			}
			g.Delete(t.ID)
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			t := g.Attachments[e.EID()]
			if t == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.PlayerTurnEnd:
				if engine.PlayerID(t.Target) == m.Player {
					return []engine.Message{engine.DamageEntity{Target: t.Target, Damage: 1, Source: t.ID}}
				}
			case engine.BasicRecover:
				if m.Player == engine.PlayerID(t.Target) {
					return []engine.Message{engine.DiscardAttachmentMsg{ID: t.ID}}
				}
			}
			return nil
		},
	})

	// 35031 Tentacle Strike: stunned + 1 damage (4 if already stunned).
	engine.RegisterBehavior("35031", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			dmg := 1
			if p.Stunned {
				dmg = 4
			}
			return []engine.Message{
				engine.StunEntity{Target: p.ID},
				engine.DamageEntity{Target: p.ID, Damage: dmg, Source: t.ID},
			}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			pid := cardutil.FirstPlayerID(g)
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			dmg := 1
			if p.Stunned {
				dmg = 4
			}
			return []engine.Message{engine.StunEntity{Target: pid}, engine.DamageEntity{Target: pid, Damage: dmg, Source: p.ID}}
		},
	})

	// 35032 Command Center: after an ally defeats a side scheme
	// (approximation: any side-scheme defeat), exhaust → 2 damage.
	engine.RegisterBehavior("35032", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.SchemeDefeated); !ok {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil || s.Exhausted {
				return nil
			}
			choices := cardutil.EnemyChoices(g, 2, s.Owner, func(target engine.EntityID) []engine.Message {
				return []engine.Message{engine.ExhaustEntity{ID: s.ID}, engine.DamageEntity{Target: target, Damage: 2, Source: s.Owner}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   s.Owner,
				Question: engine.Ask(engine.Tf("c.commandCenterDeal2Damage"), choices...),
			}}
		},
	})

	// 35033 Longshot: after he attacks a non-Elite minion, mill 1 — a
	// 3+ boost icon defeats it.
	engine.RegisterBehavior("35033", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			a := g.Allies[e.EID()]
			mn := g.Minions[m.Target]
			if !ok || m.Ally != e.EID() || a == nil || mn == nil || mn.EDef().HasTrait("elite") {
				return nil
			}
			if len(g.EncounterDeck) == 0 {
				return nil
			}
			top := g.EncounterDeck[0]
			g.EncounterDeck = g.EncounterDeck[1:]
			g.EncounterDiscard = append(g.EncounterDiscard, top)
			if cardutil.BoostOf(top) >= 3 {
				g.TLogf("c.longshotSLuckyHitDefeats", mn)
				return []engine.Message{engine.DamageEntity{Target: mn.ID, Damage: 99, Source: a.Owner}}
			}
			return nil
		},
	})

	// 35034 Lady Deathstrike: after she attacks and damages a character,
	// its owner discards 1 random card.
	engine.RegisterBehavior("35034", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil || len(p.Hand) == 0 {
				return nil
			}
			i := g.Random(len(p.Hand))
			c := p.Hand[i]
			p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
			p.Discard = append(p.Discard, c)
			g.TLogf("c.ladyDeathstrikeMakesDiscard", p.Name, c)
			return nil
		},
	})

	// 35035 Seeking Vengeance: reveal — activate Lady Deathstrike or
	// search her.
	engine.RegisterBehavior("35035", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, mn := range g.Minions {
				if mn != nil && mn.Code == "35034" {
					if p := g.Player(g.Players[0].ID); p != nil && p.IsHero() {
						return []engine.Message{engine.AskAttack{Enemy: mn.ID, Player: g.Players[0].ID}}
					}
					if g.MainScheme != nil {
						return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: mn.SchemeVal, Source: mn.ID}}
					}
					return nil
				}
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "35034" {
						zone.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: g.Players[0].ID, Card: c}}
					}
				}
			}
			return nil
		},
	})

	// 35036 Adamantium Upgrades: attach to an enemy without a copy;
	// spend [energy][mental][physical] to discard.
	engine.RegisterBehavior("35036", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil {
					continue
				}
				attached := false
				for _, a := range g.Attachments {
					if a != nil && a.Code == "35036" && a.Target == mn.ID {
						attached = true
					}
				}
				if attached {
					continue
				}
				t.Target = mn.ID
				mn.Attachments = append(mn.Attachments, t.ID)
				return nil
			}
			if v := wolvVillain(g); v != nil {
				t.Target = v.ID
				return nil
			}
			g.Delete(t.ID)
			return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			t := g.Attachments[e.EID()]
			if t == nil {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.adamantiumUpgradesSpendEnergyMentalPhysicalDiscard"), Type: engine.AbilityAction,
				Cost: 3, CostIcons: "energy:1 mental:1 physical:1",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
				},
			}}
		},
	})

	// 35037 Hack 'n' Slash: discard 1 random, take damage equal to its
	// printed resources.
	engine.RegisterBehavior("35037", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			n := randomDiscardWolv(g, p)
			if n > 0 {
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: n, Source: t.ID}}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			n := randomDiscardWolv(g, p)
			if n > 0 {
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: n, Source: p.ID}}
			}
			return nil
		},
	})
}

// randomDiscardWolv discards one random hand card and returns its printed
// resource count.
func randomDiscardWolv(g *engine.Game, p *engine.Player) int {
	if p == nil || len(p.Hand) == 0 {
		return 0
	}
	i := g.Random(len(p.Hand))
	c := p.Hand[i]
	p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
	p.Discard = append(p.Discard, c)
	return len(c.Def().Resources)
}
