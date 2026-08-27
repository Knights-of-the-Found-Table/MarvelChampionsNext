// mg_sabretooth.go implements the Stalked by Sabretooth scenario content
// (32060–32072): the Sabretooth villain, his schemes, the Robert Kelly
// protection objective and the sabretooth modular set.
package mutantgenesis

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerSabretooth()
}

// robertKelly finds the Robert Kelly ally in play.
func robertKelly(g *engine.Game) *engine.Ally {
	for _, a := range g.Allies {
		if a != nil && a.Code == "32066" {
			return a
		}
	}
	return nil
}

// sabretoothVillain returns the Sabretooth villain.
func sabretoothVillain(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == "32060" {
			return v
		}
	}
	return nil
}

func registerSabretooth() {
	// 32060–32062 Sabretooth stages: after activating, discard the top
	// encounter card and heal per boost icon.
	for _, code := range []string{"32060", "32061", "32062"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				m, ok := msg.(engine.VillainActivates)
				if !ok || m.VillainID != e.EID() || len(g.EncounterDeck) == 0 {
					return nil
				}
				v := g.Villains[e.EID()]
				if v == nil {
					return nil
				}
				top := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				g.EncounterDiscard = append(g.EncounterDiscard, top)
				heal := cardutil.BoostOf(top)
				if heal > 0 && v.Damage > 0 {
					v.Damage -= heal
					if v.Damage < 0 {
						v.Damage = 0
					}
					g.TLogf("c.sabretoothHealsBoostIconsOn", heal, top)
				}
				return nil
			},
		})
	}

	// 32063 Stalked by Sabretooth: after villain-phase step 1, deal 2
	// damage to Robert Kelly (3 at 6+/hero threat). Kelly's defeat loses
	// the game (see 32066).
	engine.RegisterBehavior("32063", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			bp, ok := msg.(engine.BeginPhase)
			if !ok || bp.Phase != engine.PhaseVillain {
				return nil
			}
			s := g.MainScheme
			kelly := robertKelly(g)
			if s == nil || kelly == nil {
				return nil
			}
			n := 2
			if s.Threat >= 6*len(g.Players) {
				n = 3
			}
			return []engine.Message{engine.DamageEntity{Target: kelly.ID, Damage: n, Source: s.ID}}
		},
	})

	// 32064 The Injured Senator: completing it defeats Robert Kelly (the
	// loss routes through his defeat response).
	engine.RegisterBehavior("32064", &engine.Behavior{})

	// 32066 Robert Kelly: if he leaves play the players lose.
	engine.RegisterBehavior("32066", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AllyDefeated); !ok {
				return nil
			}
			if a := g.Allies[e.EID()]; a != nil && a.Code == "32066" {
				return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.kellyDefeated")}}
			}
			return nil
		},
	})

	// 32065 Find the Senator: When Defeated — the first player takes
	// control of Robert Kelly (he already is in play) and the main scheme
	// advances (approximation of the flip-next-to-scheme rider).
	engine.RegisterBehavior("32065", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() || g.MainScheme == nil {
				return nil
			}
			g.TLogf("c.findTheSenatorRobertKellyIsSafeTheMainSchemeAdvances")
			if g.MainScheme.Stage == 1 {
				return []engine.Message{engine.ReplaceMainScheme{Scheme: g.MainScheme.ID}}
			}
			return nil
		},
	})

	// 32067 Adamantium Claws / 32068 Animal Ferocity: attach to
	// Sabretooth (piercing/stalwart cosmetic); spend one of each basic
	// resource to discard.
	for _, code := range []string{"32067", "32068"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				if v := sabretoothVillain(g); v != nil {
					t.Target = v.ID
				}
				return nil
			},
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				t := g.Attachments[e.EID()]
				if t == nil {
					return nil
				}
				return []engine.Ability{{
					Label: engine.Tf("c.discardSpend", t, "[energy][mental][physical]"), Type: engine.AbilityAction,
					Cost: 3, CostIcons: "energy:1 mental:1 physical:1",
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
					},
				}}
			},
		})
	}

	// 32069 Sabretooth Strikes: 1 damage to Robert Kelly; you may exhaust
	// your hero to prevent it.
	engine.RegisterBehavior("32069", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			kelly := robertKelly(g)
			if kelly == nil {
				return nil
			}
			choices := []engine.Choice{engine.Choice{
				ID: "take", Label: engine.Tf("c.deal1DamageToRobertKelly"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.DamageEntity{Target: kelly.ID, Damage: 1, Source: t.ID})}
			if p.IsHero() && !p.Exhausted {
				choices = append(choices, engine.Choice{
					ID: "prevent", Label: engine.Tf("c.exhaustYourHeroPrevent"), Kind: engine.ChoiceLabel,
				}.Msgs(engine.ExhaustEntity{ID: p.ID}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.sabretoothStrikesChoose"), choices...),
			}}
		},
	})

	// 32070 Unrelenting Savage: Sabretooth activates against you (+1
	// ATK/SCH while undamaged).
	engine.RegisterBehavior("32070", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			v := sabretoothVillain(g)
			if v == nil {
				return nil
			}
			if !p.IsHero() && g.MainScheme != nil {
				n := v.SchemeVal
				if v.Damage == 0 {
					n++
				}
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: v.ID}}
			}
			msgs := []engine.Message{}
			if v.Damage == 0 {
				msgs = append(msgs, engine.BoostEnemyAttack{Enemy: v.ID, N: 1})
			}
			return append(msgs, engine.AskAttack{Enemy: v.ID, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou})
		},
	})

	// 32071 Medical Emergency: Hinder 2/hero; When Defeated — heal 2
	// damage from Robert Kelly.
	engine.RegisterBehavior("32071", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			if kelly := robertKelly(g); kelly != nil {
				return []engine.Message{engine.HealEntity{Target: kelly.ID, N: 2}}
			}
			return nil
		},
	})

	// 32072 Feral Rage: When Defeated — Sabretooth attacks the defeater
	// (first player approximation).
	engine.RegisterBehavior("32072", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			v := sabretoothVillain(g)
			if v == nil {
				return nil
			}
			return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: cardutil.FirstPlayerID(g)}}
		},
	})
}
