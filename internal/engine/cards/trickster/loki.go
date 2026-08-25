package trickster

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// avatarCodes are the four Avatar of Loki villain cards.
var avatarCodes = []string{"55029a", "55030a", "55031a", "55032a"}

// avatarBases are the registry keys for the avatar behaviors.
var avatarBases = []string{"55029", "55030", "55031", "55032"}

// ttAvatar finds the Avatar of Loki villain in play.
func ttAvatar(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && v.EDef().HasTrait("avatar of loki") {
			return v
		}
	}
	return nil
}

// ttShatter adds shatter counters to the Avatar of Loki.
func ttShatter(g *engine.Game, n int) {
	if v := ttAvatar(g); v != nil {
		v.Counters += n
		g.TLogf("c.holdsShatterCounters", v, v.Counters)
	}
}

// synergyEnv finds a Synergy environment by base code.
func synergyEnv(g *engine.Game, base string) *engine.Environment {
	for _, env := range g.Environments {
		if env != nil && data.BaseCode(env.Code) == base {
			return env
		}
	}
	return nil
}

func registerLoki() {
	// The Avatars: when one would be defeated, it is swapped for another
	// illusion instead; the god himself falls when the deck runs dry.
	for _, code := range avatarBases {
		engine.RegisterBehavior(code, &engine.Behavior{
			VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
				if v.Damage+damage < v.MaxHP {
					return true
				}
				if v.Counters > 0 {
					// Shattered: this illusion collapses for good.
					return true
				}
				v.Damage = 0
				g.TLogf("c.wasAnIllusionAnotherTakesItsPlace", v)
				ttSwapAvatar(g, v)
				return false
			},
		})
	}
	// 55027 Loki, God of Lies: reveals his true form at low health.
	engine.RegisterBehavior("55027", &engine.Behavior{
		VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
			if data.BaseCode(v.Code) != "55027" {
				return true
			}
			v.Damage += damage
			if v.MaxHP-v.Damage <= 10 && v.Code == "55027a" {
				v.Code = "55027b"
				g.TLogMajorf("c.lokiGodOfLiesShedsHisDisguise")
			}
			return false
		},
	})

	// Main schemes.
	engine.RegisterBehavior("55028", &engine.Behavior{})
	engine.RegisterBehavior("55033", &engine.Behavior{})

	// 55034 Intense Focus.
	engine.RegisterBehavior("55034", &engine.Behavior{})
	// 55035 Wrapped in Chains.
	engine.RegisterBehavior("55035", &engine.Behavior{})
	// 55036 Dark Scepter.
	engine.RegisterBehavior("55036", &engine.Behavior{})

	// 55037 Draugr Buddy.
	engine.RegisterBehavior("55037", &engine.Behavior{})
	// 55038 Grendell: defeat shatters 3.
	engine.RegisterBehavior("55038", &engine.Behavior{React: shatterOnDefeat(3)})
	// 55039 Malekith.
	engine.RegisterBehavior("55039", &engine.Behavior{React: shatterOnDefeat(2)})
	// 55040 Minotaur.
	engine.RegisterBehavior("55040", &engine.Behavior{React: shatterOnDefeat(2)})
	// 55041 The Mangog: cross-pod defeat (approximated: 3 shatter).
	engine.RegisterBehavior("55041", &engine.Behavior{React: shatterOnDefeat(3)})
	// 55042 Fenris Wolf.
	engine.RegisterBehavior("55042", &engine.Behavior{React: shatterOnDefeat(3)})
	// 55043 Hraesvelgr.
	engine.RegisterBehavior("55043", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for i := 0; i < 3; i++ {
				if c, ok := g.DrawEncounter(); ok {
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				}
			}
			return nil
		},
		React: shatterOnDefeat(3),
	})
	// 55044 Laufey.
	engine.RegisterBehavior("55044", &engine.Behavior{React: shatterOnDefeat(3)})

	// 55045 Aura of Stasis.
	engine.RegisterBehavior("55045", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for i := 0; i < 2; i++ {
				if c, ok := g.DrawEncounter(); ok {
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				}
			}
			return nil
		},
	})
	// 55046 Door Between Worlds / 55047 Lofty Goals / 55048 New Jotunheim.
	engine.RegisterBehavior("55046", &engine.Behavior{React: shatterOnDefeat(3)})
	engine.RegisterBehavior("55047", &engine.Behavior{React: shatterOnDefeat(2)})
	engine.RegisterBehavior("55048", &engine.Behavior{React: shatterOnDefeat(3)})

	// 55049 Dark Arts.
	engine.RegisterBehavior("55049", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for i := len(g.EncounterDiscard) - 1; i >= 0; i-- {
				if g.EncounterDiscard[i].Def().Type == "minion" {
					c := g.EncounterDiscard[i]
					g.EncounterDiscard = append(g.EncounterDiscard[:i:i], g.EncounterDiscard[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 3, Source: t.ID}}
			}
			return nil
		},
	})
	// 55050 Dirty Trick.
	engine.RegisterBehavior("55050", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if v := ttAvatar(g); v != nil && v.Counters > 0 {
				v.Counters--
			}
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
		},
	})
	// 55051 Stories and Lies: swap the avatar for a random set-aside one.
	engine.RegisterBehavior("55051", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if v := ttAvatar(g); v != nil {
				ttSwapAvatar(g, v)
				if v2 := ttAvatar(g); v2 != nil {
					if p.IsHero() && v2.AttackVal >= v2.SchemeVal {
						return []engine.Message{engine.AskAttack{Enemy: v2.ID, Player: p.ID}}
					}
					if g.MainScheme != nil {
						return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v2.SchemeVal, Source: v2.ID}}
					}
				}
			}
			return nil
		},
	})

	// 55052-55055 Synergy environments: spend a synergy counter for a
	// one-shot boost.
	synergy := func(base string, label engine.Msg) {
		engine.RegisterBehavior(base, &engine.Behavior{
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				env := g.Environments[e.EID()]
				if env == nil || env.Counters <= 0 {
					return nil
				}
				return []engine.Ability{{
					Label: label, Type: engine.AbilityAction,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						env := g.Environments[self]
						if env == nil || env.Counters <= 0 {
							return nil
						}
						env.Counters--
						p := g.Players[0]
						if p == nil {
							return nil
						}
						switch data.BaseCode(env.Code) {
						case "55052": // Domineering Force: +4 attack damage
							ids := enemyIDs(g)
							if len(ids) > 0 {
								return []engine.Message{engine.DamageEntity{Target: ids[0], Damage: 4, Source: self}}
							}
						case "55053": // Feigned Retreat: prevent 4
							return []engine.Message{engine.ToughEntity{Target: p.ID}}
						case "55054": // Mounting Resistance: remove 4 threat
							if g.MainScheme != nil {
								return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 4, Source: self}}
							}
						case "55055": // Unified Front: draw 2 (double wild)
							return []engine.Message{engine.DrawCards{Player: p.ID, N: 2}}
						}
						return nil
					},
				}}
			},
		})
	}
	synergy("55052", engine.Tf("c.domineeringForceSpendASynergyCounterDeal4Damage"))
	synergy("55053", engine.Tf("c.feignedRetreatSpendASynergyCounterToughStatusCard"))
	synergy("55054", engine.Tf("c.mountingResistanceSpendASynergyCounterRemove4Threat"))
	synergy("55055", engine.Tf("c.unifiedFrontSpendASynergyCounterDraw2Cards"))
}

// shatterOnDefeat adds shatter counters when the entity is defeated.
func shatterOnDefeat(n int) func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
	return func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.MinionDefeated)
		if !ok || m.MinionID != e.EID() {
			return nil
		}
		ttShatter(g, n)
		return nil
	}
}

// enemyIDs lists enemy entity ids.
func enemyIDs(g *engine.Game) []engine.EntityID {
	var out []engine.EntityID
	for id := range g.Villains {
		out = append(out, id)
	}
	for id := range g.Minions {
		out = append(out, id)
	}
	return out
}

// ttSwapAvatar replaces the avatar in play with a random set-aside one;
// when none remain the real Loki reveals himself.
func ttSwapAvatar(g *engine.Game, v *engine.Villain) {
	var bench []engine.Card
	for _, c := range g.SetAside {
		if c.Def().Type == "villain" && c.Def().HasTrait("avatar of loki") {
			bench = append(bench, c)
		}
	}
	old := v.Code
	g.Delete(v.ID)
	if len(bench) == 0 {
		// The illusions are spent: Loki, God of Lies takes the field.
		nv := g.SpawnVillainFromCard("55027")
		if nv != nil {
			g.TLogf("c.theIllusionsBurnAwayLokiGodOfLiesStandsRevealed")
		}
		return
	}
	pick := bench[g.Random(len(bench))]
	var kept engine.CardList
	for _, c := range g.SetAside {
		if c.ID != pick.ID {
			kept = append(kept, c)
		}
	}
	g.SetAside = kept
	if nv := g.SpawnVillainFromCard(data.BaseCode(pick.Code)); nv != nil {
		g.TLogf("c.aNewAvatarOfLokiEmerges")
	}
	_ = old
}
