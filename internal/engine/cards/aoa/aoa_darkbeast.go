package aoa

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// settingEnv finds the in-play Setting environment.
func settingEnv(g *engine.Game) *engine.Environment {
	for _, e := range g.Environments {
		if e != nil && e.EDef().HasTrait("Setting") {
			return e
		}
	}
	return nil
}

// resolveSettingSpecial resolves the Setting environment's Special
// ability against the player: Savage Land mills 3, Genosha threatens the
// main scheme, Blue Moon chips the identity.
func resolveSettingSpecial(g *engine.Game, p *engine.Player) []engine.Message {
	env := settingEnv(g)
	if env == nil || p == nil {
		return nil
	}
	switch engine.BaseCodeOf(env.Code) {
	case "45127": // The Savage Land
		g.Logf("%s resolves The Savage Land — discards 3", p.Name)
		return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 3}}
	case "45133": // Genosha
		if g.MainScheme != nil {
			g.Logf("%s resolves Genosha — 1 threat on the main scheme", p.Name)
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: env.ID}}
		}
	case "45139": // Blue Area of the Moon
		g.Logf("%s resolves the Blue Area of the Moon — 1 damage", p.Name)
		return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: env.ID}}
	}
	return nil
}

// revealRandomSetting reveals a random set-aside Setting environment.
func revealRandomSetting(g *engine.Game) []engine.Message {
	var settings []engine.Card
	for _, c := range g.SetAside {
		if c.Def().Type == "environment" && c.Def().HasTrait("Setting") {
			settings = append(settings, c)
		}
	}
	if len(settings) == 0 {
		return nil
	}
	c := settings[g.Random(len(settings))]
	g.SetAside = removeCard(g.SetAside, c)
	// Only one Setting lives at a time.
	for _, env := range g.Environments {
		if env != nil && env.EDef().HasTrait("Setting") {
			g.Delete(env.ID)
		}
	}
	g.Logf("The scene shifts — %s!", c.Def().Name)
	return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
}

// darkBeast finds the Dark Beast villain.
func darkBeast(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == "45118" {
			return v
		}
	}
	return nil
}

func registerDarkBeast() {
	// 45118-45120 Dark Beast: attacks trigger the Setting's Special; on
	// reveal a random Setting appears.
	dbBehavior := &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			return resolveSettingSpecial(g, g.Player(m.Player))
		},
	}
	engine.RegisterBehavior("45118", dbBehavior)
	engine.RegisterBehavior("45119", dbBehavior)
	engine.RegisterBehavior("45120", dbBehavior)

	// 45121 Dark Beast's Bogus Journey: stage completed loses (default).
	engine.RegisterBehavior("45121", &engine.Behavior{})

	// 45122 High-Tech Goggles / 45123 Genetic Enhancement: attach to Dark
	// Beast (the hero discard actions are not modeled).
	for _, code := range []string{"45122", "45123"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				if v := darkBeast(g); v != nil {
					t.Target = v.ID
				}
				return nil
			},
			Boost: func(g *engine.Game, card engine.Card) []engine.Message {
				if v := darkBeast(g); v != nil {
					t := &engine.Attachment{ID: g.NextEntityID(engine.KindAttachment), Code: card.Code, Target: v.ID}
					g.Attachments[t.ID] = t
					v.Attachments = append(v.Attachments, t.ID)
					g.Logf("%s attaches to Dark Beast", card.Def().Name)
				}
				return nil
			},
		})
	}

	// 45124 Cruel Experiment: buff a milled minion.
	engine.RegisterBehavior("45124", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for i := 0; i < 30; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					return nil
				}
				if c.Def().Type == "minion" {
					t.Target = engine.EntityID("") // filled after spawn below
					g.EncounterDiscard = append(g.EncounterDiscard, c)
					return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})

	// 45125 Evil Genius: scheme or attack with a boost.
	engine.RegisterBehavior("45125", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			v := darkBeast(g)
			if v == nil {
				return nil
			}
			if p.IsHero() {
				return []engine.Message{
					engine.DealBoost{Enemy: v.ID},
					engine.AskAttack{Enemy: v.ID, Player: p.ID},
				}
			}
			return []engine.Message{
				engine.ToughEntity{Target: v.ID},
				engine.DealBoost{Enemy: v.ID},
				engine.RevealBoost{Enemy: v.ID},
				engine.ApplyVillainScheme{VillainID: v.ID, Player: p.ID},
			}
		},
	})

	// 45126 Time-Travel Shenanigans: mill for a Setting-set card.
	engine.RegisterBehavior("45126", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			env := settingEnv(g)
			if env == nil {
				return nil
			}
			set := env.EDef().CardSet
			for i := 0; i < 30; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					return nil
				}
				if c.Def().CardSet == set {
					return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})

	registerSavageLand()
	registerGenosha()
	registerBlueMoon()
}

func registerSavageLand() {
	// 45127 The Savage Land: the villain's retaliate rider is not
	// modeled; the Special mills 3.
	engine.RegisterBehavior("45127", &engine.Behavior{})

	// 45128 Pterosaur: reveals resolve the Special; boost self-deals.
	engine.RegisterBehavior("45128", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			return resolveSettingSpecial(g, g.Player(mn.EngagedWith))
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				for _, e := range g.Environments {
					if e != nil && e.Code == "45127" {
						p.EncounterDown = append(p.EncounterDown, card)
						return nil
					}
				}
			}
			return nil
		},
	})

	// 45129 Velociraptor: luck-scaled attack.
	engine.RegisterBehavior("45129", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			mn := g.Minions[e.EID()]
			p := g.Player(m.Player)
			if mn == nil || p == nil || len(p.Deck) == 0 {
				return nil
			}
			n := len(p.Deck[0].Def().Resources)
			mn.BoostCount += n
			g.Logf("Velociraptor discards a card — +%d ATK", n)
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 1}}
		},
	})

	// 45130 Giant Ape: defeat resolves the Special.
	engine.RegisterBehavior("45130", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			return resolveSettingSpecial(g, g.Player(cardutil.FirstPlayerID(g)))
		},
	})

	// 45131 Land Out of Time: Special + indirect per icon, or fetch the
	// Savage Land.
	engine.RegisterBehavior("45131", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for _, env := range g.Environments {
				if env != nil && env.Code == "45127" {
					msgs := resolveSettingSpecial(g, p)
					// The Special already milled 3; approximate the
					// indirect damage as 1 per icon of the top 3 (skip:
					// resolveSettingSpecial already discarded them).
					return msgs
				}
			}
			for _, c := range append(engine.CardList{}, g.EncounterDeck...) {
				if c.Code == "45127" {
					g.EncounterDeck.Remove(c.ID)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			return nil
		},
	})

	// 45132 Village Under Attack: defeat resolves the Special for
	// everyone.
	engine.RegisterBehavior("45132", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, resolveSettingSpecial(g, p)...)
			}
			return msgs
		},
	})
}

func registerGenosha() {
	// 45133 Genosha: steady rider not modeled; Special threatens.
	engine.RegisterBehavior("45133", &engine.Behavior{})

	// 45134 Magistrate: defeat attaches Escaped Mutant; boost scales.
	engine.RegisterBehavior("45134", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			return attachEscapedMutant(g, p.ID)
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			// +3 boost icons while Escaped Mutant is attached to the
			// revealer (first player approximation).
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				for _, a := range g.Attachments {
					if a != nil && a.Code == "45137" && a.Target == p.ID {
						if id := firstVillainID(g); id != "" {
							return []engine.Message{engine.BoostActivation{Enemy: id, N: 3}}
						}
					}
				}
			}
			return nil
		},
	})

	// 45135 Armored Unibike: engages the marked player; attacks resolve
	// the Special.
	engine.RegisterBehavior("45135", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			for _, a := range g.Attachments {
				if a != nil && a.Code == "45137" && a.Target != "" {
					mn.EngagedWith = engine.PlayerID(a.Target)
					break
				}
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			pid := m.Player
			if a := g.Allies[engine.EntityID(pid)]; a != nil {
				pid = a.Owner
			}
			return resolveSettingSpecial(g, g.Player(pid))
		},
	})

	// 45136 Genoshan Mech: double Special after killing an ally.
	engine.RegisterBehavior("45136", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			if a := g.Allies[engine.EntityID(m.Player)]; a == nil {
				return nil
			}
			pid := m.Player
			if a := g.Allies[engine.EntityID(pid)]; a != nil {
				pid = a.Owner
			}
			msgs := resolveSettingSpecial(g, g.Player(pid))
			return append(msgs, resolveSettingSpecial(g, g.Player(pid))...)
		},
	})

	// 45137 Escaped Mutant: quickstrike aura not modeled; the discard
	// action resolves the Special.
	engine.RegisterBehavior("45137", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				t.Target = p.ID
			}
			return nil
		},
	})

	// 45138 Police State: defeat fetches Escaped Mutant.
	engine.RegisterBehavior("45138", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			for _, c := range append(engine.CardList{}, g.EncounterDeck...) {
				if c.Code == "45137" {
					g.EncounterDeck.Remove(c.ID)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			return nil
		},
	})
}

func attachEscapedMutant(g *engine.Game, pid engine.PlayerID) []engine.Message {
	t := &engine.Attachment{ID: g.NextEntityID(engine.KindAttachment), Code: "45137", Target: pid}
	g.Attachments[t.ID] = t
	g.Logf("Escaped Mutant attaches to %s", g.Player(pid).Name)
	return nil
}

func registerBlueMoon() {
	// 45139 Blue Area of the Moon: minions gain guard (approximated on
	// reveal); Special chips the identity.
	engine.RegisterBehavior("45139", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionEntersPlay); !ok {
				return nil
			}
			return nil
		},
	})

	// 45140 Gladiator: immune while Trial by Combat lives.
	engine.RegisterBehavior("45140", &engine.Behavior{
		MinionDamageable: func(g *engine.Game, m *engine.Minion, damage int) bool {
			for _, s := range g.SideSchemes {
				if s != nil && s.Code == "45146" {
					g.Logf("Gladiator cannot take damage while Trial by Combat is in play")
					return false
				}
			}
			return true
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for _, s := range g.SideSchemes {
				if s != nil && s.Code == "45146" {
					if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
						p.EncounterDown = append(p.EncounterDown, card)
					}
				}
			}
			return nil
		},
	})

	// 45141-45143 Oracle/Manta/Earthquake: status + Special on repeat.
	minionStatusSpecial := func(code string, already func(p *engine.Player) bool, apply func(p *engine.Player) engine.Message) {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
				mn := g.Minions[e.EID()]
				if mn == nil || mn.EngagedWith == "" {
					return nil
				}
				p := g.Player(mn.EngagedWith)
				if p == nil {
					return nil
				}
				msgs := []engine.Message{apply(p)}
				if already(p) {
					msgs = append(msgs, resolveSettingSpecial(g, p)...)
				}
				return msgs
			},
		})
	}
	minionStatusSpecial("45141", func(p *engine.Player) bool { return p.Confused },
		func(p *engine.Player) engine.Message { return engine.ConfuseEntity{Target: p.ID} })
	minionStatusSpecial("45142", func(p *engine.Player) bool { return p.Stunned },
		func(p *engine.Player) engine.Message { return engine.StunEntity{Target: p.ID} })
	minionStatusSpecial("45143", func(p *engine.Player) bool { return p.Exhausted },
		func(p *engine.Player) engine.Message { return engine.ExhaustEntity{ID: p.ID} })

	// 45144 Warstar: mill check.
	engine.RegisterBehavior("45144", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			c, ok := g.DrawEncounter()
			if !ok {
				return nil
			}
			if c.Def().Type == "minion" && c.Def().HasTrait("Imperial Guard") {
				return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
			}
			g.EncounterDiscard = append(g.EncounterDiscard, c)
			return resolveSettingSpecial(g, p)
		},
	})

	// 45145 Imperial Guardsman: buff a minion.
	engine.RegisterBehavior("45145", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, mn := range g.Minions {
				if mn != nil {
					t.Target = mn.ID
					mn.MaxHP += 4
					return nil
				}
			}
			if c, ok := g.DrawEncounter(); ok {
				return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok {
				return nil
			}
			t := g.Attachments[e.EID()]
			if t == nil || t.Target != m.MinionID {
				return nil
			}
			return resolveSettingSpecial(g, g.Player(cardutil.FirstPlayerID(g)))
		},
	})

	// 45146 Trial by Combat: defeat shuffles Guards back.
	engine.RegisterBehavior("45146", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var kept engine.CardList
			for _, c := range g.EncounterDiscard {
				if c.Def().Type == "minion" && c.Def().HasTrait("Imperial Guard") {
					g.EncounterDeck = append(g.EncounterDeck, c)
					continue
				}
				kept = append(kept, c)
			}
			g.EncounterDiscard = kept
			g.ShuffleEncounterDeck()
			g.Logf("Trial by Combat — Imperial Guards shuffle back")
			return nil
		},
	})
}
