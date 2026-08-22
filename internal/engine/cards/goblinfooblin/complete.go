package goblinfooblin

// complete.go implements the remaining Green Goblin scenario pack cards
// (infamy/madness economy, boost riders, modular sets). The villain
// personas are registered side-keyed in goblin.go; their base codes get
// marker registrations here so the survey reflects them.

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func registerRemainingGob() {
	// Villain / main-scheme bases: the actual logic lives on the a/b
	// side registrations and the scenario definitions.
	for _, code := range []string{"02001", "02002", "02003", "02004", "02005", "02017", "02018", "02006"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// Hired Gun: villain boost card or +2 infamy.
	engine.RegisterBehavior("02007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			var villain engine.EntityID
			for id := range g.Villains {
				villain = id
			}
			_ = addInfamy // infamy option mutates on selection via nil msgs; use a marker
			picks := []engine.Choice{
				engine.Choice{ID: "boost", Label: "Give the villain 1 facedown boost card", Kind: engine.ChoiceLabel}.
					Msgs(engine.DealBoost{Enemy: villain}),
				engine.Choice{ID: "infamy", Label: "Place 2 infamy counters on Criminal Enterprise", Kind: engine.ChoiceLabel}.
					Msgs(engine.AddInfamyMsg{Env: "02006a", N: 2, OrMadness: 2}),
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask("Hired Gun:", picks...)}}
		},
		Boost: boostInfamy(1),
	})

	// Private Security Specialist: Guard + infamy boost.
	engine.RegisterBehavior("02008", &engine.Behavior{Boost: boostInfamy(1)})

	// Collapsing Bridge / Payoff: infamy boost side schemes.
	engine.RegisterBehavior("02009", &engine.Behavior{Boost: boostInfamy(1)})
	engine.RegisterBehavior("02011", &engine.Behavior{Boost: boostInfamy(1)})

	// Oscorp Manufacturing: per-hero threat (Norman side only,
	// approximated for any form).
	engine.RegisterBehavior("02010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: len(g.Players), Source: e.EID()}}
		},
	})

	// All in a Day's Work: +2 infamy (or -2 madness); boost +1.
	engine.RegisterBehavior("02012", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return addInfamy(g, 2)
		},
		Boost: boostInfamy(1),
	})

	// Mad Genius.
	engine.RegisterBehavior("02013", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var villain engine.EntityID
			for id := range g.Villains {
				villain = id
			}
			if !p.IsHero() {
				// Norman: mill 1 per infamy counter.
				n := 0
				if env := g.EnvironmentByCode("02006a"); env != nil {
					n = env.Counters
				}
				return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: n}}
			}
			// Green Goblin: attack the hero with the fewest HP.
			var target *engine.Player
			for _, q := range g.Players {
				if !q.KOed && (target == nil || q.HP() < target.HP()) {
					target = q
				}
			}
			if target == nil {
				return nil
			}
			return []engine.Message{
				engine.DealBoost{Enemy: villain}, engine.RevealBoost{Enemy: villain},
				engine.AskAttack{Enemy: villain, Player: target.ID, Trigger: engine.TriggerVillainAttacksYou},
			}
		},
	})

	// Hysteria: Green Goblin gets an extra boost card per activation.
	engine.RegisterBehavior("02020", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			a := g.Attachments[e.EID()]
			if a == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.VillainActivates:
				if m.VillainID == a.Target {
					return []engine.Message{engine.DealBoost{Enemy: a.Target}}
				}
			}
			return nil
		},
		Abilities: removalAbilityGob("Spend [mental] [mental] → discard Hysteria", "mental:2", 2),
	})

	// Goblin Thrall: Guard + "Boost: put into play" (engine-covered).
	engine.RegisterBehavior("02024", &engine.Behavior{})

	// Goblin Nation: +1 ATK to Goblin enemies; boost puts it into play
	// (engine-covered).
	engine.RegisterBehavior("02027", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, mn := range g.Minions {
				if mn.EDef().HasTrait("goblin") {
					mn.AttackVal++
				}
			}
			g.Logf("Goblin Nation: each Goblin enemy gets +1 ATK")
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if m, ok := msg.(engine.MinionEntersPlay); ok {
				if mn := g.Minions[m.MinionID]; mn != nil && mn.EDef().HasTrait("goblin") {
					mn.AttackVal++
				}
			}
			return nil
		},
	})

	// Overrun: when defeated, each player mills 2 encounter cards and
	// recruits discarded Goblin minions.
	engine.RegisterBehavior("02028", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if d, ok := msg.(engine.SchemeDefeated); ok && d.Scheme == e.EID() {
				var msgs []engine.Message
				for _, p := range g.Players {
					p := p
					for i := 0; i < 2; i++ {
						c, ok := g.DrawEncounter()
						if !ok {
							break
						}
						if c.Def().Type == "minion" && c.Def().HasTrait("goblin") {
							msgs = append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
						} else {
							g.EncounterDiscard = append(g.EncounterDiscard, c)
						}
					}
				}
				return msgs
			}
			return nil
		},
	})

	// Wicked Ambitions: mill 2×stage; per Goblin minion choose damage or
	// recruit.
	engine.RegisterBehavior("02032", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			stage := villainStage(g)
			var msgs []engine.Message
			for i := 0; i < 2*stage; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					break
				}
				if c.Def().Type == "minion" && c.Def().HasTrait("goblin") {
					msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(
						fmt.Sprintf("%s discarded: take 3 damage or put it into play engaged with you?", c.Def().Name),
						engine.Choice{ID: "dmg", Label: "Take 3 damage", Kind: engine.ChoiceLabel}.
							Msgs(engine.DamageEntity{Target: p.ID, Damage: 3, Source: t.ID}),
						engine.Choice{ID: "play", Label: "Put into play engaged with you", Kind: engine.ChoiceLabel}.
							Msgs(engine.RevealEncounterCard{Player: p.ID, Card: c}),
					)})
				} else {
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				}
			}
			return msgs
		},
	})

	// Intimidation: pay 2 or give the villain a boost card.
	engine.RegisterBehavior("02035", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var villain engine.EntityID
			for id := range g.Villains {
				villain = id
			}
			pay := g.CustomPaymentQuestion(p, 2, "Intimidation: spend 2 resources or give the villain a boost card",
				map[string]any{"player": p.ID.String()})
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				"Intimidation: pay 2 resources or boost the villain?",
				engine.Choice{ID: "pay", Label: "Pay 2 resources", Kind: engine.ChoiceLabel}.WithThen(pay),
				engine.Choice{ID: "boost", Label: "Give the villain 1 facedown boost card", Kind: engine.ChoiceLabel}.
					Msgs(engine.DealBoost{Enemy: villain}),
			)}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			var villain engine.EntityID
			for id := range g.Villains {
				villain = id
			}
			return []engine.Message{engine.DealBoost{Enemy: villain}}
		},
	})

	// Regenerative Healing: villain heals 2×stage; boost heals 2.
	engine.RegisterBehavior("02036", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			healed := healVillainsGob(g, 2*villainStage(g))
			if !healed {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			healVillainsGob(g, 2)
			return nil
		},
	})

	// A Mess of Things: +2 threat per stunned friendly character.
	engine.RegisterBehavior("02037", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 0
			for _, p := range g.Players {
				if p.Stunned {
					n++
				}
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && a.Stunned {
						n++
					}
				}
			}
			if n == 0 {
				return nil
			}
			return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: 2 * n, Source: e.EID()}}
		},
	})

	// Scorpion: Quickstrike (engine) + stun on damaging attacks.
	engine.RegisterBehavior("02038", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Source != e.EID() {
				return nil
			}
			switch d.Target.Kind() {
			case engine.KindPlayer, engine.KindAlly:
				return []engine.Message{engine.StunEntity{Target: d.Target}}
			}
			return nil
		},
	})

	// Gang-Up: alter-ego surges; hero: villain + engaged minions attack.
	engine.RegisterBehavior("02039", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			var msgs []engine.Message
			for id := range g.Villains {
				msgs = append(msgs, engine.DealBoost{Enemy: id}, engine.RevealBoost{Enemy: id},
					engine.AskAttack{Enemy: id, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou})
			}
			for _, mn := range g.Minions {
				if mn.EngagedWith == p.ID {
					msgs = append(msgs, engine.AskAttack{Enemy: mn.ID, Player: p.ID})
				}
			}
			return msgs
		},
	})

	// Tail Sweep: Scorpion attacks (or you are stunned); boost stuns.
	engine.RegisterBehavior("02040", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for _, mn := range g.Minions {
				if mn.Code == "02038" {
					return []engine.Message{engine.AskAttack{Enemy: mn.ID, Player: p.ID}}
				}
			}
			return []engine.Message{engine.StunEntity{Target: p.ID}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			// The revealer is not passed through the Boost hook; stun the
			// first player as an approximation of "you are stunned".
			return []engine.Message{engine.StunEntity{Target: card.Owner}}
		},
	})

	// Power Drain: when defeated, mill 2 and each player discards 1 card
	// per boost icon.
	engine.RegisterBehavior("02041", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if d, ok := msg.(engine.SchemeDefeated); ok && d.Scheme == e.EID() {
				icons := 0
				for i := 0; i < 2; i++ {
					c, ok := g.DrawEncounter()
					if !ok {
						break
					}
					icons += derefBoost(c)
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				}
				if icons == 0 {
					return nil
				}
				for _, q := range g.Players {
					for i := 0; i < icons && len(q.Hand) > 0; i++ {
						c := q.Hand[0]
						q.Hand = q.Hand[1:]
						q.Discard = append(q.Discard, c)
					}
				}
				g.Logf("Power Drain: each player discards %d card(s)", icons)
			}
			return nil
		},
	})

	// Electro: after attacking a player, mill 1 and indirect damage per
	// boost icon; boost mills 3.
	engine.RegisterBehavior("02042", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Source != e.EID() || !d.Target.Is(engine.KindPlayer) {
				return nil
			}
			return electroMill(g, d.Target, 1)
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			millEncounter(g, 3)
			return nil
		},
	})

	// Electromagnetic Pulse: mill 7; Electro enters or surge.
	engine.RegisterBehavior("02043", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for i := 0; i < 7; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					break
				}
				if c.Code == "02042" {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			millEncounter(g, 3)
			return nil
		},
	})

	// Lightning Bolt: mill 2, indirect damage per boost icon.
	engine.RegisterBehavior("02044", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			icons := 0
			for i := 0; i < 2; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					break
				}
				icons += derefBoost(c)
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			if icons > 0 {
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: icons, Source: t.ID}}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			millEncounter(g, 3)
			return nil
		},
	})

	// Shock Therapy: mill 1 per hero; villain heals per boost icon.
	engine.RegisterBehavior("02045", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			icons := 0
			for i := 0; i < len(g.Players); i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					break
				}
				icons += derefBoost(c)
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			if icons > 0 {
				healVillainsGob(g, icons)
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			millEncounter(g, 3)
			return nil
		},
	})

	// Running Interference: each player pays [mental][physical] or +2 threat.
	engine.RegisterBehavior("02046", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, q := range g.Players {
				pay := g.CustomPaymentQuestion(q, 2, "Running Interference: spend 2 resources or place 2 threat",
					map[string]any{"player": q.ID.String()})
				msgs = append(msgs, engine.AskQuestion{Player: q.ID, Question: engine.Ask(
					"Running Interference: pay 2 resources or place 2 threat here?",
					engine.Choice{ID: "pay", Label: "Pay 2 resources", Kind: engine.ChoiceLabel}.WithThen(pay),
					engine.Choice{ID: "threat", Label: "Place 2 threat here", Kind: engine.ChoiceLabel}.
						Msgs(engine.SchemeThreat{Scheme: e.EID(), N: 2, Source: q.ID}),
				)})
			}
			return msgs
		},
	})

	// Tombstone: after damaging a player, they discard a mental or
	// physical resource.
	engine.RegisterBehavior("02047", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Source != e.EID() || !d.Target.Is(engine.KindPlayer) {
				return nil
			}
			p := g.Player(d.Target)
			if p == nil {
				return nil
			}
			for i, c := range p.Hand {
				for _, r := range c.Def().Resources {
					if r == "mental" || r == "physical" {
						p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
						p.Discard = append(p.Discard, c)
						g.Logf("%s discards %s (Tombstone)", p.Name, c.Def().Name)
						return nil
					}
				}
			}
			return nil
		},
	})

	// All Tied Up: the ready/form lock is not enforced by the engine;
	// the removal action works.
	engine.RegisterBehavior("02048", &engine.Behavior{
		Abilities: removalAbilityGob("Spend [mental] [physical] → discard All Tied Up", "mental:1 physical:1", 2),
	})

	// Media Coverage: the doubling effect is not enforced; the removal
	// action works.
	engine.RegisterBehavior("02049", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Spend [mental] → discard Media Coverage", Type: engine.AbilityAction,
				Cost: 1, CostIcons: "mental:1", AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Attachments[self]
					if a == nil {
						return nil
					}
					g.Delete(self)
					g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
					return nil
				},
			}}
		},
	})
}

// ---- helpers ----

// boostInfamy builds a Boost hook that adds n infamy counters to
// Criminal Enterprise (or removes n madness counters from State of
// Madness).
func boostInfamy(n int) func(g *engine.Game, card engine.Card) []engine.Message {
	return func(g *engine.Game, card engine.Card) []engine.Message {
		return addInfamy(g, n)
	}
}

func addInfamy(g *engine.Game, n int) []engine.Message {
	if env := g.EnvironmentByCode("02006a"); env != nil {
		env.Counters += n
		g.Logf("Criminal Enterprise gains %d infamy counter(s) (%d total)", n, env.Counters)
		return nil
	}
	if env := g.EnvironmentByCode("02006b"); env != nil {
		env.Counters -= n
		if env.Counters < 0 {
			env.Counters = 0
		}
		g.Logf("State of Madness loses %d madness counter(s) (%d left)", n, env.Counters)
		if env.Counters == 0 {
			for id, v := range g.Villains {
				flipToNorman(g, v, env)
				_ = id
				break
			}
		}
	}
	return nil
}

func villainStage(g *engine.Game) int {
	for _, v := range g.Villains {
		return v.Stage
	}
	return 1
}

func healVillainsGob(g *engine.Game, n int) bool {
	healed := false
	for _, v := range g.Villains {
		if v.Damage > 0 {
			d := min(n, v.Damage)
			v.Damage -= d
			healed = true
			g.Logf("%s heals %d damage", v.EDef().Name, d)
		}
	}
	return healed
}

// millEncounter discards n cards off the encounter deck.
func millEncounter(g *engine.Game, n int) {
	for i := 0; i < n; i++ {
		c, ok := g.DrawEncounter()
		if !ok {
			return
		}
		g.EncounterDiscard = append(g.EncounterDiscard, c)
	}
}

// electroMill mills n encounter cards and deals the player 1 indirect
// damage per boost icon discarded (approximated as direct damage).
func electroMill(g *engine.Game, pid engine.PlayerID, n int) []engine.Message {
	icons := 0
	for i := 0; i < n; i++ {
		c, ok := g.DrawEncounter()
		if !ok {
			break
		}
		icons += derefBoost(c)
		g.EncounterDiscard = append(g.EncounterDiscard, c)
	}
	if icons > 0 {
		return []engine.Message{engine.DamageEntity{Target: pid, Damage: icons, Source: engine.EntityID("02042")}}
	}
	return nil
}

func derefBoost(c engine.Card) int {
	if b := c.Def().Boost; b != nil {
		return *b
	}
	return 0
}

func removalAbilityGob(label, icons string, cost int) func(g *engine.Game, e engine.Entity) []engine.Ability {
	return func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{
			Label: label, Type: engine.AbilityAction, Cost: cost, CostIcons: icons, HeroOnly: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				a := g.Attachments[self]
				if a == nil {
					return nil
				}
				g.Delete(self)
				g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
				g.Logf("%s is discarded", a.EDef().Name)
				return nil
			},
		}}
	}
}

var _ = cardutil.FirstPlayerID
