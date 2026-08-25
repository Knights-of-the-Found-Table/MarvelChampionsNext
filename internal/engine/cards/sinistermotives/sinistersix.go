package sinistermotives

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// sixBases are the Sinister Six member codes with their attack riders.
var sixBases = []string{"27094", "27095", "27096", "27097", "27098", "27099"}

// registerSinisterSix installs the Sinister Six scenario (27094–27112):
// the active counter rotates through rotating members.
func registerSinisterSix() {
	// Members: after attacking you, the rider fires and the counter
	// moves on.
	riders := map[string]func(g *engine.Game, p *engine.Player, src engine.EntityID) []engine.Message{
		"27094": func(g *engine.Game, p *engine.Player, src engine.EntityID) []engine.Message {
			var msgs []engine.Message
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: src})
			}
			for id := range g.SideSchemes {
				msgs = append(msgs, engine.SchemeThreat{Scheme: id, N: 1, Source: src})
			}
			return msgs
		},
		"27095": func(g *engine.Game, p *engine.Player, src engine.EntityID) []engine.Message {
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 7}}
		},
		"27096": func(g *engine.Game, p *engine.Player, src engine.EntityID) []engine.Message {
			return []engine.Message{engine.IndirectDamage{Player: p.ID, N: 2}}
		},
		"27097": func(g *engine.Game, p *engine.Player, src engine.EntityID) []engine.Message {
			// Discard a support or upgrade (most recent).
			if len(p.Upgrades) > 0 {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[len(p.Upgrades)-1]}}
			}
			if len(p.Supports) > 0 {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Supports[len(p.Supports)-1]}}
			}
			return nil
		},
		"27098": func(g *engine.Game, p *engine.Player, src engine.EntityID) []engine.Message {
			return []engine.Message{engine.StunEntity{Target: p.ID}}
		},
		"27099": func(g *engine.Game, p *engine.Player, src engine.EntityID) []engine.Message {
			if len(p.Hand) > 0 {
				return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0]}}}
			}
			return nil
		},
	}

	for _, base := range sixBases {
		rider := riders[base]
		engine.RegisterBehavior(base, &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				w, ok := msg.(engine.WindowAfterEnemyAttacked)
				if !ok || w.Enemy != e.EID() {
					return nil
				}
				p := g.Player(w.Player)
				if p == nil {
					return nil
				}
				msgs := rider(g, p, e.EID())
				msgs = append(msgs, advanceSixCounter(g, e.EID())...)
				return msgs
			},
		})
	}

	// 27100 Sinister Synchronization (stage 1): bench the extras.
	engine.RegisterBehavior("27100", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			want := len(g.Players) + 1
			var inPlay []engine.EntityID
			for _, id := range cardutil.SortedIDs(g.Villains) {
				if v := g.Villains[id]; v != nil {
					inPlay = append(inPlay, id)
				}
			}
			for _, id := range inPlay[want:] {
				v := g.Villains[id]
				if v == nil {
					continue
				}
				base := engine.BaseCodeOf(v.ECode())
				g.SetAside = append(g.SetAside, engine.Card{ID: g.NextCardID(), Code: base})
				delete(g.Villains, id)
				g.TLogf("c.waitsOnTheBench", engine.DB.MustLookup(base).Name)
			}
			// Active counter on the first member.
			for _, id := range cardutil.SortedIDs(g.Villains) {
				g.ActiveVillain = id
				break
			}
			return nil
		},
	})

	// 27101 Sinister Beatdown (stage 2): ambush from the bench.
	engine.RegisterBehavior("27101", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if msgs := sixAmbush(g); len(msgs) == 0 {
				return []engine.Message{engine.DealEncounterToPlayer{Player: cardutil.FirstPlayerID(g)}}
			}
			return nil
		},
	})

	// 27102 Light at the End: the escape hatch.
	engine.RegisterBehavior("27102", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			tw, ok := msg.(engine.ThwartScheme)
			if !ok || tw.Scheme != e.EID() {
				return nil
			}
			s := g.SideSchemes[e.EID()]
			if s == nil || s.Threat > tw.N {
				return nil
			}
			// Last threat removed: ambush, then escape if possible.
			var msgs []engine.Message
			msgs = append(msgs, sixAmbush(g)...)
			if len(g.Villains) == 0 && len(g.SetAside) == 0 {
				msgs = append(msgs, engine.GameOver{Won: true, Reason: engine.Tf("reason.escapedSinisterSix")})
			} else {
				// Stay in play with fresh threat.
				s.Threat = tw.N + 3
				g.TLogf("c.lightAtTheEndFlaresBackToLife")
			}
			return nil
		},
	})

	// 27103–27106 leadership attachments: attach preferences only (the
	// aura effects are approximated away).
	engine.RegisterBehavior("27103", &engine.Behavior{})
	engine.RegisterBehavior("27104", &engine.Behavior{})
	engine.RegisterBehavior("27105", &engine.Behavior{})
	engine.RegisterBehavior("27106", &engine.Behavior{})

	// 27107 Brute Force Barricade: locks other side schemes (the lock is
	// approximated as a plain hinder).
	engine.RegisterBehavior("27107", &engine.Behavior{})

	// 27108–27110 summon treacheries: pull the benched members.
	for _, spec := range []struct {
		code          string
		first, second string
	}{
		{"27108", "27096", "27099"},
		{"27109", "27095", "27097"},
		{"27110", "27094", "27098"},
	} {
		engine.RegisterBehavior(spec.code, &engine.Behavior{
			ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
				g.Delete(t.ID)
				return []engine.Message{engine.SummonSix{Cards: []string{spec.first, spec.second}}}
			},
		})
	}

	// 27111 Partnership of Pain: gang-stats activation.
	engine.RegisterBehavior("27111", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			total, id := 0, engine.EntityID("")
			for vid := range g.Villains {
				if v := g.Villains[vid]; v != nil {
					if p.IsHero() {
						total += v.AttackVal
					} else {
						total += v.SchemeVal
					}
					id = vid
				}
			}
			if id == "" {
				return nil
			}
			return []engine.Message{
				engine.BoostActivation{Enemy: id, N: total},
				engine.VillainActivates{VillainID: id, Player: p.ID},
			}
		},
	})

	// 27112 Surprise!: resolve Ambush!.
	engine.RegisterBehavior("27112", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			msgs := sixAmbush(g)
			if len(msgs) == 0 {
				for id := range g.SideSchemes {
					if g.SideSchemes[id] != nil && g.SideSchemes[id].Code[:5] == "27102" {
						return []engine.Message{engine.SchemeThreat{Scheme: id, N: 3, Source: t.ID}}
					}
				}
			}
			return msgs
		},
	})
}

// advanceSixCounter moves the active counter to the next member.
func advanceSixCounter(g *engine.Game, current engine.EntityID) []engine.Message {
	ids := cardutil.SortedIDs(g.Villains)
	for i, id := range ids {
		if id == current && i+1 < len(ids) {
			g.ActiveVillain = ids[i+1]
			g.TLogf("c.theActiveCounterMovesTo", g.Villains[ids[i+1]].EDef().Name)
			return nil
		}
	}
	if len(ids) > 0 {
		g.ActiveVillain = ids[0]
	}
	return nil
}

// sixAmbush pulls a random benched member into play with the counter.
func sixAmbush(g *engine.Game) []engine.Message {
	bench := []string{}
	for _, c := range g.SetAside {
		base := engine.BaseCodeOf(c.Code)
		if base[:2] == "27" && containsStr(sixBases, base) {
			bench = append(bench, base)
		}
	}
	if len(bench) == 0 {
		return nil
	}
	pick := bench[g.Random(len(bench))]
	var kept engine.CardList
	for _, c := range g.SetAside {
		if engine.BaseCodeOf(c.Code) != pick {
			kept = append(kept, c)
		}
	}
	g.SetAside = kept
	v := g.SpawnVillainFromCard(pick)
	if v != nil {
		g.ActiveVillain = v.ID
		g.TLogMajorf("c.ambushJoinsTheSinisterSix", v)
	}
	return nil
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
