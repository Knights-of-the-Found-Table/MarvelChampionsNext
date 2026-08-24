// Package trickster registers Trickster Takeover: the Enchantress and
// Loki, God of Lies scenarios plus the Trickster Magic modular set.
package trickster

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerEnchantress()
	registerLoki()
	registerTricksterMagic()
	registerScenarios()
}

// ttVillain returns the first villain in play.
func ttVillain(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil {
			return v
		}
	}
	return nil
}

// gazeOf finds the Hypnotic Gaze attachment on a player (any of
// 55007a-55011a or their Trance flips).
func gazeOf(g *engine.Game, pid engine.PlayerID) *engine.Attachment {
	for _, a := range g.Attachments {
		if a == nil || a.Target == "" || engine.PlayerID(a.Target) != pid {
			continue
		}
		base := data.BaseCode(a.Code)
		if base >= "55007" && base <= "55011" {
			return a
		}
	}
	return nil
}

// addCharm places a charm counter on the player's Enchantment, flipping
// it to the Trance side at five counters.
func addCharm(g *engine.Game, pid engine.PlayerID, n int) {
	a := gazeOf(g, pid)
	if a == nil {
		return
	}
	a.Counters += n
	g.Logf("%s's Enchantment holds %d charm counters", g.Player(pid).Name, a.Counters)
	if a.Counters >= 5 && data.BaseCode(a.Code) == a.Code {
		a.Code = a.Code + "b"
		a.Counters = 0
		p := g.Player(pid)
		p.ExtraTraits = append(p.ExtraTraits, "enthralled")
		g.LogMajorf("%s falls into a Trance — Enthralled!", p.Name)
	}
}

// charmOnAttack bumps the attacking player's Enchantment after an enemy
// attack resolves.
func charmOnAttack(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
	m, ok := msg.(engine.WindowAfterEnemyAttacked)
	if !ok || m.Enemy != e.EID() {
		return nil
	}
	addCharm(g, m.Player, 1)
	return nil
}

func registerEnchantress() {
	// Enchantress I-III: attack taxing + immunity from Future of Despair.
	for _, code := range []string{"55001", "55002", "55003"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			React: charmOnAttack,
			VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
				if g.SideSchemeInPlay("55006") {
					g.Logf("Enchantress is shielded by Future of Despair")
					return false
				}
				return true
			},
		})
	}
	// Stage II/III: Future of Despair joins with bonus threat.
	engine.RegisterBehavior("55002", &engine.Behavior{
		React:        charmOnAttack,
		VillainStage: ttRevealDespair(3),
	})
	engine.RegisterBehavior("55003", &engine.Behavior{
		React:        charmOnAttack,
		VillainStage: ttRevealDespair(4),
	})

	// Main schemes.
	engine.RegisterBehavior("55004", &engine.Behavior{})
	engine.RegisterBehavior("55005", &engine.Behavior{})

	// 55006 Future of Despair: charm on entry.
	engine.RegisterBehavior("55006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, p := range g.Players {
				addCharm(g, p.ID, 1)
			}
			return nil
		},
	})

	// 55007a-55011a Hypnotic Gazes (Defiant until the Trance flip, which
	// addCharm performs).
	for _, code := range []string{"55007", "55008", "55009", "55010", "55011"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 55012-55014 Temptations: forced-action riders approximated as
	// alter-ego discard actions.
	for _, code := range []string{"55012", "55013", "55014"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
	// 55015 Seduced.
	engine.RegisterBehavior("55015", &engine.Behavior{})
	// 55016 Crown of the Enchantress: schemes add charm; group exhaust to
	// shed it.
	engine.RegisterBehavior("55016", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeThreat)
			if !ok || m.Source == "" || g.Villains[m.Source] == nil {
				return nil
			}
			for _, p := range g.Players {
				addCharm(g, p.ID, 1)
			}
			return nil
		},
	})

	// 55017 Enthralled Lackey: attacks add charm.
	engine.RegisterBehavior("55017", &engine.Behavior{React: charmOnAttack})
	// 55018 Enthralled Brute.
	engine.RegisterBehavior("55018", &engine.Behavior{})
	// 55019 Sindr: activation keyed to Defiant/Enthralled.
	engine.RegisterBehavior("55019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				if mn := g.Minions[e.EID()]; mn != nil {
					p = g.Player(mn.EngagedWith)
				}
			}
			if p == nil {
				return nil
			}
			return []engine.Message{engine.MinionActivates{MinionID: e.EID(), Player: p.ID}}
		},
	})
	// 55020 Ulik: avenges Enchantress damage.
	engine.RegisterBehavior("55020", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			if !ok || m.Damage <= 0 {
				return nil
			}
			mn := g.Minions[e.EID()]
			if mn == nil || g.Villains[m.Target] == nil {
				return nil
			}
			g.Logf("Ulik lashes out at %s", g.Player(mn.EngagedWith).Name)
			return []engine.Message{engine.DamageEntity{Target: mn.EngagedWith, Damage: mn.AttackVal, Source: e.EID()}}
		},
	})

	// 55021 Law of Attraction: find an Enthralled minion.
	engine.RegisterBehavior("55021", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for i, c := range g.EncounterDeck {
				if c.Def().HasTrait("enthralled") && c.Def().Type == "minion" {
					card := c
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					def := card.Def()
					mn := &engine.Minion{
						ID: g.NextEntityID(engine.KindMinion), Code: card.Code,
						MaxHP: ttInt(def.HP, 1), AttackVal: ttInt(def.Attack, 0), SchemeVal: ttInt(def.Scheme, 0),
						EngagedWith: g.Players[0].ID,
					}
					g.Minions[mn.ID] = mn
					return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: g.Players[0].ID}}
				}
			}
			return nil
		},
	})
	// 55022 Spellbound.
	engine.RegisterBehavior("55022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, p := range g.Players {
				addCharm(g, p.ID, 1)
			}
			return nil
		},
	})
	// 55023-55026 treacheries.
	engine.RegisterBehavior("55023", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			// Reveal the first minion or side scheme found.
			for i, c := range g.EncounterDeck {
				if c.Def().Type == "minion" || c.Def().Type == "side_scheme" {
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("55024", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			addCharm(g, p.ID, 1)
			return []engine.Message{engine.StunEntity{Target: p.ID}, engine.ConfuseEntity{Target: p.ID}}
		},
	})
	engine.RegisterBehavior("55025", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			v := ttVillain(g)
			if v == nil {
				return nil
			}
			if p.IsHero() {
				return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: p.ID}}
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("55026", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			addCharm(g, p.ID, 1)
			return []engine.Message{engine.IndirectDamage{Player: p.ID, N: 3}}
		},
	})
}

// ttRevealDespair puts the set-aside Future of Despair into play with n
// extra threat per hero.
func ttRevealDespair(perHero int) func(g *engine.Game, v *engine.Villain, nextStage int) []engine.Message {
	return func(g *engine.Game, v *engine.Villain, nextStage int) []engine.Message {
		for i, c := range g.SetAside {
			if c.Code == "55006" {
				g.SetAside = append(g.SetAside[:i:i], g.SetAside[i+1:]...)
				s := &engine.SideScheme{
					ID: g.NextEntityID(engine.KindSideScheme), Code: "55006",
					Threat: 2 + perHero*len(g.Players), MaxThreat: 6 + 2*len(g.Players),
				}
				g.SideSchemes[s.ID] = s
				g.LogMajorf("Future of Despair enters play")
				return nil
			}
		}
		return nil
	}
}

// ttInt dereferences a numeric card field.
func ttInt(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}
