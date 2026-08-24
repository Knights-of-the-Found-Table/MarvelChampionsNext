package nextevolution

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// sinister finds the Mister Sinister villain.
func sinister(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == "40136" {
			return v
		}
	}
	return nil
}

// sinisterHasTrait checks Sinister's printed traits plus the superpower
// attachments' grants (Flight/Brute/Psionic).
func sinisterHasTrait(g *engine.Game, trait string) bool {
	v := sinister(g)
	if v == nil {
		return false
	}
	if v.EDef().HasTrait(trait) {
		return true
	}
	grants := map[string]string{"40151": "Aerial", "40155": "Brute", "40159": "Psionic"}
	for _, aid := range v.Attachments {
		if a := g.Attachments[aid]; a != nil && grants[a.Code] == trait {
			return true
		}
	}
	return false
}

// sinisterSuperpowers counts SUPERPOWER attachments on Sinister.
func sinisterSuperpowers(g *engine.Game) int {
	v := sinister(g)
	if v == nil {
		return 0
	}
	n := 0
	for _, aid := range v.Attachments {
		if a := g.Attachments[aid]; a != nil && a.EDef().HasTrait("Superpower") {
			n++
		}
	}
	return n
}

// attachToSinister pins an attachment to Mister Sinister.
func attachToSinister(g *engine.Game, code string) *engine.Attachment {
	v := sinister(g)
	if v == nil {
		return nil
	}
	t := &engine.Attachment{ID: g.NextEntityID(engine.KindAttachment), Code: code, Target: v.ID}
	g.Attachments[t.ID] = t
	v.Attachments = append(v.Attachments, t.ID)
	g.Logf("%s attaches to Mister Sinister", t.EDef().Name)
	return t
}

func registerSinister() {
	// 40136-40138 Mister Sinister stages.
	engine.RegisterBehavior("40136", &engine.Behavior{
		// Forced Response: a status card placed on him adds main-scheme
		// threat (1/2/3 by stage).
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var target engine.EntityID
			switch m := msg.(type) {
			case engine.StunEntity:
				target = m.Target
			case engine.ConfuseEntity:
				target = m.Target
			case engine.ToughEntity:
				target = m.Target
			default:
				return nil
			}
			if target != e.EID() || g.MainScheme == nil {
				return nil
			}
			n := map[string]int{"40136": 1, "40137": 2, "40138": 3}[e.ECode()]
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: e.EID()}}
		},
		// Teleported Away (40146): Sinister cannot take damage and schemes
		// instead of attacking; Sinister Disguise (40144) redirects damage
		// unless two mental resources are spent.
		VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
			if g.SideSchemeInPlay("40146") {
				g.Logf("Mister Sinister cannot take damage (Teleported Away)")
				return false
			}
			for _, aid := range v.Attachments {
				if a := g.Attachments[aid]; a != nil && a.Code == "40144" {
					// Approximation: auto-spend a pair of mental resources
					// when one exists; otherwise the damage fizzles onto
					// the weakest ally.
					for _, p := range g.Players {
						var mental []engine.Card
						for _, c := range p.Hand {
							for _, r := range c.Def().Resources {
								if r == "mental" || r == "wild" {
									mental = append(mental, c)
									break
								}
							}
							if len(mental) >= 2 {
								break
							}
						}
						if len(mental) >= 2 {
							g.Delete(aid)
							g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
							g.Push(engine.DiscardCards{Player: p.ID, Cards: mental[:2]})
							g.Logf("Sinister Disguise: %s spends two mental resources; the attachment discards", p.Name)
							return true
						}
					}
					g.Delete(aid)
					g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
					var weakest engine.EntityID
					best := 1 << 30
					for _, p := range g.Players {
						for _, id := range p.Allies {
							if a2 := g.Allies[id]; a2 != nil && a2.HP() < best {
								best, weakest = a2.HP(), id
							}
						}
					}
					if weakest != "" {
						g.Logf("Sinister Disguise redirects %d damage", damage)
						g.Push(engine.DamageEntity{Target: weakest, Damage: damage, Source: v.ID})
					}
					return false
				}
			}
			return true
		},
		VillainActivate: func(g *engine.Game, v *engine.Villain, p *engine.Player) []engine.Message {
			if g.SideSchemeInPlay("40146") {
				g.Logf("Mister Sinister schemes instead of attacking (Teleported Away)")
				return []engine.Message{
					engine.DealBoost{Enemy: v.ID},
					engine.RevealBoost{Enemy: v.ID},
					engine.ApplyVillainScheme{VillainID: v.ID, Player: p.ID},
				}
			}
			if p.IsHero() {
				if v.Stunned {
					v.Stunned = false
					g.Logf("Mister Sinister is stunned; attack canceled")
					return nil
				}
				return []engine.Message{
					engine.DealBoost{Enemy: v.ID},
					engine.RevealBoost{Enemy: v.ID},
					engine.AskAttack{Enemy: v.ID, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou},
				}
			}
			if v.Confused {
				v.Confused = false
				return nil
			}
			return []engine.Message{
				engine.DealBoost{Enemy: v.ID},
				engine.RevealBoost{Enemy: v.ID},
				engine.ApplyVillainScheme{VillainID: v.ID, Player: p.ID},
			}
		},
		// Stage II/III reveal: extra main-scheme threat.
		VillainStage: func(g *engine.Game, v *engine.Villain, nextStage int) []engine.Message {
			if g.MainScheme == nil {
				return nil
			}
			base := 1
			if v.Code == "40138" {
				base = 2
			}
			if sinisterSuperpowers(g) < 2 {
				base++
			}
			n := base * len(g.Players)
			g.Logf("Mister Sinister reveals stage %s — %d threat", v.EDef().StageLabel, n)
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: v.ID}}
		},
	})

	// Stages II and III share the base registration.
	engine.RegisterBehavior("40137", engine.LookupBehavior("40136"))
	engine.RegisterBehavior("40138", engine.LookupBehavior("40136"))

	// 40139 Sinister Intent: remove one random stage 2 from the game at
	// reveal (handled here); see the scenario hooks for stage flow.
	engine.RegisterBehavior("40139", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage != 1 || len(s.StageCodes) != 5 {
				return nil
			}
			i := 1 + g.Random(3)
			g.Logf("Sinister Intent removes %s from the game", engine.DB.MustLookup(s.StageCodes[i]).Name)
			s.StageCodes = append(s.StageCodes[:i:i], s.StageCodes[i+1:]...)
			return nil
		},
	})

	// 40140-40142 stage 2s: attach the matching superpower on reveal.
	stageAttach := map[string]string{"40140": "40151", "40141": "40155", "40142": "40159"}
	for stage, attach := range stageAttach {
		code := stage
		att := attach
		engine.RegisterBehavior(code, &engine.Behavior{
			MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
				attachToSinister(g, att)
				return nil
			},
		})
	}

	// 40143 Sinister Ends: deal everyone a facedown encounter card; the
	// redirect of Sinister's attacks to Hope is not modeled.
	engine.RegisterBehavior("40143", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.DealEncounterToPlayer{Player: p.ID})
			}
			return msgs
		},
	})

	// 40144 Sinister Disguise: logic lives on the villain's damage gate.
	engine.RegisterBehavior("40144", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := sinister(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
	})

	// 40145 Sinister Soldier: scales with Sinister's superpowers.
	engine.RegisterBehavior("40145", &engine.Behavior{
		EnemyStatBonus: func(g *engine.Game, e engine.Entity) (atk, sch int) {
			n := sinisterSuperpowers(g)
			return n, n
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if n := sinisterSuperpowers(g); n > 0 {
				if v := sinister(g); v != nil {
					return []engine.Message{engine.BoostActivation{Enemy: v.ID, N: n}}
				}
			}
			return nil
		},
	})

	// 40146 Teleported Away: effects live on the villain's gates.
	engine.RegisterBehavior("40146", &engine.Behavior{})

	// 40147 Genetic Mastery: punish Sinister's traits.
	engine.RegisterBehavior("40147", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			if sinisterHasTrait(g, "Aerial") {
				msgs = append(msgs, engine.IndirectDamage{Player: p.ID, N: 2})
			}
			if sinisterHasTrait(g, "Brute") {
				msgs = append(msgs, engine.ExhaustEntity{ID: p.ID})
			}
			if sinisterHasTrait(g, "Psionic") && g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID})
			}
			return msgs
		},
	})

	// 40148 Molecular Control: tough (+ heal if Brute); boost: tough.
	engine.RegisterBehavior("40148", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			v := sinister(g)
			if v == nil {
				return nil
			}
			msgs := []engine.Message{engine.ToughEntity{Target: v.ID}}
			if sinisterHasTrait(g, "Brute") {
				msgs = append(msgs, engine.HealEntity{Target: v.ID, N: 4})
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if v := sinister(g); v != nil {
				return []engine.Message{engine.ToughEntity{Target: v.ID}}
			}
			return nil
		},
	})

	// 40149 Sinister Schemes: Sinister schemes; Psionic confuses.
	engine.RegisterBehavior("40149", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			v := sinister(g)
			if v == nil {
				return nil
			}
			msgs := []engine.Message{engine.ApplyVillainScheme{VillainID: v.ID, Player: p.ID}}
			if sinisterHasTrait(g, "Psionic") {
				msgs = append(msgs, engine.ConfuseEntity{Target: p.ID})
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return confuseBoost(g, card)
		},
	})

	// 40150 Sinister Strike: alter-ego surge / hero attacked.
	engine.RegisterBehavior("40150", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			v := sinister(g)
			if !p.IsHero() {
				if c, ok := g.DrawEncounter(); ok {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				return nil
			}
			if v == nil {
				return nil
			}
			msgs := []engine.Message{engine.AskAttack{Enemy: v.ID, Player: p.ID}}
			if sinisterHasTrait(g, "Aerial") {
				msgs = append(msgs, engine.StunEntity{Target: p.ID})
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return stunBoost(g, card)
		},
	})

	// 40151 Flight: permanent Aerial grant (overkill rider not modeled).
	engine.RegisterBehavior("40151", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := sinister(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
	})

	// 40152 Aerial Bombardment: +1 ATK (engine); retaliate-ignoring and
	// the discard action not modeled.
	engine.RegisterBehavior("40152", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := sinister(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
	})

	// 40153 Out of Reach: only Aerial heroes can damage Sinister
	// (approximation; the discard action is not modeled).
	engine.RegisterBehavior("40153", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := sinister(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
	})

	// 40154 High Ground: strip toughs + group indirect.
	engine.RegisterBehavior("40154", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for _, o := range g.Players {
				if o.Tough {
					o.Tough = false
				}
				for _, id := range o.Allies {
					if a := g.Allies[id]; a != nil {
						a.Tough = false
					}
				}
			}
			n := 2
			if sinisterHasTrait(g, "Brute") {
				n = 4
			}
			per := n / len(g.Players)
			if per < 1 {
				per = 1
			}
			var msgs []engine.Message
			for _, o := range g.Players {
				msgs = append(msgs, engine.IndirectDamage{Player: o.ID, N: per})
			}
			return msgs
		},
	})

	// 40155 Super Strength: permanent Brute grant (steady not modeled).
	engine.RegisterBehavior("40155", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := sinister(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
	})

	// 40156 Impervious: -1 damage per hit (engine).
	engine.RegisterBehavior("40156", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := sinister(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
	})

	// 40157 Thrown Object: ranged rider not modeled; discard after the
	// villain attacks (ClearBoosts marks the activation's end).
	engine.RegisterBehavior("40157", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := sinister(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.ClearBoosts)
			if !ok {
				return nil
			}
			t := g.Attachments[e.EID()]
			if t == nil || m.Enemy != t.Target {
				return nil
			}
			g.Delete(t.ID)
			g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: t.Code})
			g.Logf("Thrown Object is discarded after the attack")
			return nil
		},
	})

	// 40158 "I'll Take That": lose an upgrade.
	engine.RegisterBehavior("40158", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if len(p.Upgrades) == 0 {
				if c, ok := g.DrawEncounter(); ok {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				return nil
			}
			pickHighest := sinisterHasTrait(g, "Psionic")
			best := -1
			if pickHighest {
				best = 1 << 30
			}
			var pick engine.EntityID
			for _, id := range p.Upgrades {
				u := g.Upgrades[id]
				if u == nil {
					continue
				}
				c := cardutil.Cost(u.EDef())
				if (pickHighest && c < best) || (!pickHighest && c > best) {
					best, pick = c, id
				}
			}
			if pick != "" {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: pick}}
			}
			return nil
		},
	})

	// 40159 Telepathy: permanent Psionic grant + retaliate 1 (engine).
	engine.RegisterBehavior("40159", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := sinister(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
	})

	// 40160 Manufactured Drama: exhaust each support you control (the
	// cannot-ready rider is approximated away).
	engine.RegisterBehavior("40160", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var msgs []engine.Message
			exhausted := 0
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil && !s.Exhausted {
					msgs = append(msgs, engine.ExhaustEntity{ID: id})
					exhausted++
				}
			}
			if exhausted == 0 {
				if c, ok := g.DrawEncounter(); ok {
					msgs = append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
				}
			}
			return append([]engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}, msgs...)
		},
	})

	// 40161 Sowing Discord: exhaust each ally you control.
	engine.RegisterBehavior("40161", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var msgs []engine.Message
			exhausted := 0
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && !a.Exhausted {
					msgs = append(msgs, engine.ExhaustEntity{ID: id})
					exhausted++
				}
			}
			if exhausted == 0 {
				if c, ok := g.DrawEncounter(); ok {
					msgs = append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
				}
			}
			return append([]engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}, msgs...)
		},
	})

	// 40162 One Step Ahead: discard random cards; boost overkill not
	// modeled.
	engine.RegisterBehavior("40162", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			n := 1
			if sinisterHasTrait(g, "Aerial") {
				n = 2
			}
			var discarded engine.CardList
			for i := 0; i < n && len(p.Hand) > 0; i++ {
				idx := g.Random(len(p.Hand))
				discarded = append(discarded, p.Hand[idx])
				p.Hand = append(p.Hand[:idx:idx], p.Hand[idx+1:]...)
			}
			if len(discarded) > 0 {
				g.Logf("One Step Ahead discards %d random card(s)", len(discarded))
				return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: discarded}}
			}
			return nil
		},
	})
}
