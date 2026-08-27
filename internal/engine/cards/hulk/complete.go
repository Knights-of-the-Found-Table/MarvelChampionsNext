package hulk

// complete.go implements the remaining Hulk hero-pack cards. Approximations
// are noted inline.

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func registerRemaining() {
	// Hulk Smash: +10 ATK for a basic attack.
	// Approximation: playable as an action granting +10 ATK until the end
	// of the phase (the per-attack window does not exist; overkill is not
	// implemented).
	engine.RegisterBehavior("10003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), ATK: 10}}
		},
	})

	// Limitless Strength: hero-form-only resource (restriction not
	// enforced by the payment layer).
	engine.RegisterBehavior("10007", &engine.Behavior{})

	// Boundless Rage: +1 ATK; discarded after you change form.
	engine.RegisterBehavior("10009", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} },
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if cf, ok := msg.(engine.ChangeForm); ok && cf.Player == e.EOwner() {
				return []engine.Message{engine.DiscardControlled{Player: e.EOwner(), ID: e.EID()}}
			}
			return nil
		},
	})

	// Brawn: after he attacks, remove 1 threat.
	engine.RegisterBehavior("10011", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyAttackWindow)
			if !ok || w.Ally != e.EID() {
				return nil
			}
			var picks []engine.Choice
			for _, sid := range g.Schemes() {
				s := g.Entity(sid)
				picks = append(picks, engine.Choice{
					Label: engine.Tf("m.cardName", s), Kind: engine.ChoiceTarget, SourceID: sid, CardCode: s.ECode(),
				}.Msgs(engine.ThwartScheme{Scheme: sid, N: 1, Source: e.EOwner()}))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask(engine.Tf("c.brawnRemove1ThreatFromWhichScheme"), picks...)}}
		},
	})

	// Sentry: deal yourself an encounter card on entry.
	engine.RegisterBehavior("10012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.DealEncounterToPlayer{Player: e.EOwner()}}
		},
	})

	// She-Hulk: +1 ATK per damage on her.
	engine.RegisterBehavior("10013", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			switch m := msg.(type) {
			case engine.DamageEntity:
				if m.Target == e.EID() {
					if a := g.Allies[e.EID()]; a != nil {
						a.PermATK = a.Damage
					}
				}
			case engine.HealEntity:
				if m.Target == e.EID() {
					if a := g.Allies[e.EID()]; a != nil {
						a.PermATK = a.Damage
					}
				}
			}
			return nil
		},
	})

	// Drop Kick: 4 damage; physical-only payment stuns and draws.
	engine.RegisterBehavior("10014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := cardutil.ChooseEnemy(engine.Tf("c.dropKickChooseAnEnemy"), func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) {
				return 4, nil
			})(g, e)
			if ec, ok := e.(*engine.EventCard); ok && paidOnlyWith(ec, "physical") {
				var tgt engine.EntityID
				// The stun/draw riders apply to the chosen enemy; the
				// choice question carries the damage, so append riders
				// against the first enemy (single-target template).
				if ids := cardutil.SortedEnemyIDs(g); len(ids) > 0 {
					tgt = ids[0]
				}
				msgs = append(msgs, engine.StunEntity{Target: tgt}, engine.DrawCards{Player: e.EOwner(), N: 1})
			}
			return msgs
		},
	})

	// Toe to Toe: the enemy attacks you, then takes 5 damage.
	engine.RegisterBehavior("10015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				if enemy == nil {
					continue
				}
				var atk []engine.Message
				if v := g.Villains[id]; v != nil {
					atk = []engine.Message{engine.DealBoost{Enemy: id}, engine.RevealBoost{Enemy: id},
						engine.AskAttack{Enemy: id, Player: e.EOwner(), Trigger: engine.TriggerVillainAttacksYou}}
				} else {
					atk = []engine.Message{engine.AskAttack{Enemy: id, Player: e.EOwner()}}
				}
				atk = append(atk, engine.DamageEntity{Target: id, Damage: 5, Source: e.EOwner()})
				picks = append(picks, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode(),
				}.Msgs(atk...))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask(engine.Tf("c.toeToToeWhichEnemyAttacksYou"), picks...)}}
		},
	})

	// "You'll Pay for That!": after the villain attacks you, remove 1
	// threat per damage taken (max 5).
	engine.RegisterBehavior("10016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			// Approximation: resolved off the last defense window.
			p := g.Player(e.EOwner())
			n := 1
			if p != nil {
				n = min(5, max(1, p.Damage))
			}
			var picks []engine.Choice
			for _, sid := range g.Schemes() {
				s := g.Entity(sid)
				picks = append(picks, engine.Choice{
					Label: engine.Tf("m.threat", s, n), Kind: engine.ChoiceTarget, SourceID: sid, CardCode: s.ECode(),
				}.Msgs(engine.ThwartScheme{Scheme: sid, N: n, Source: e.EOwner()}))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask(engine.Tf("c.youLlPayForThatRemoveThreatFromWhichScheme", n), picks...)}}
		},
	})

	// The Power of Aggression: name-driven double resource.
	engine.RegisterBehavior("10017", &engine.Behavior{})

	// Martial Prowess: physical resource for Attack events.
	engine.RegisterBehavior("10018", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "physical", EventOnly: true},
	})

	// To the Rescue!: remove 2 threat.
	engine.RegisterBehavior("10019", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme(engine.Tf("c.chooseAScheme", "To the Rescue!"), func(g *engine.Game, e engine.Entity) int { return 2 }),
	})

	// Basic resource reprints.
	engine.RegisterBehavior("10020", &engine.Behavior{})
	engine.RegisterBehavior("10021", &engine.Behavior{})
	engine.RegisterBehavior("10022", &engine.Behavior{})

	// Avengers Mansion / Helicarrier reprints: reuse the core behaviors.
	if b := engine.LookupBehavior("01091"); b != nil {
		engine.RegisterBehavior("10023", b)
	}
	if b := engine.LookupBehavior("01092"); b != nil {
		engine.RegisterBehavior("10024", b)
	}

	// Total Destruction: threat cannot be removed while Abomination is in
	// play (enforced engine-side in removeThreat).
	engine.RegisterBehavior("10027", &engine.Behavior{})

	// Beat Cop: store threat, then spend for damage.
	engine.RegisterBehavior("10029", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil {
				return nil
			}
			ab := []engine.Ability{{
				Label: engine.Tf("c.exhaustBeatCopMove1ThreatFromASchemeToHere"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var picks []engine.Choice
					for _, sid := range g.Schemes() {
						s := g.Entity(sid)
						picks = append(picks, engine.Choice{
							Label: engine.Tf("m.cardName", s), Kind: engine.ChoiceTarget, SourceID: sid, CardCode: s.ECode(),
						}.Msgs(engine.ThwartScheme{Scheme: sid, N: 1, Source: e.EOwner()},
							engine.AddEntityCounter{ID: self, N: 1}))
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
						Question: engine.Ask(engine.Tf("c.move1ThreatFromWhichScheme"), picks...)}}
				},
			}}
			if s.Counters > 0 {
				ab = append(ab, engine.Ability{
					Label: engine.Tf("c.exhaustDiscardBeatCopDamageToAMinion1PerThreat", s.Counters),
					Type:  engine.AbilityAction, Exhaust: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						s := g.Supports[self]
						if s == nil || s.Counters <= 0 {
							return nil
						}
						dmg := s.Counters
						return append([]engine.Message{engine.DiscardControlled{Player: s.Owner, ID: self}},
							cardutil.ChooseMinion(engine.Tf("c.beatCopDamageToWhichMinion", dmg), dmg)(g, g.Entity(self))...)
					},
				})
			}
			return ab
		},
	})

	// Inspiring Presence: heal 1 damage from an ally and ready it.
	engine.RegisterBehavior("10030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, q := range g.Players {
				for _, id := range q.Allies {
					a := g.Allies[id]
					if a == nil {
						continue
					}
					picks = append(picks, engine.Choice{
						Label: engine.Tf("m.cardName", a), Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code,
					}.Msgs(engine.HealEntity{Target: a.ID, N: 1}, engine.ReadyEntity{ID: a.ID}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask(engine.Tf("c.heal1AndReadyWhichAlly"), picks...)}}
		},
	})

	// Electrostatic Armor: after you defend, 1 damage to the attacker.
	engine.RegisterBehavior("10031", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			if !ok || w.Defender != e.EOwner() {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: w.Against, Damage: 1, Source: e.EID()}}
		},
	})

	// Resourceful: discard to generate a wild resource.
	// Approximation: exhausts instead of discarding (the payment layer
	// only knows exhaust-to-generate producers).
	engine.RegisterBehavior("10032", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})
}

// paidOnlyWith reports whether the event was paid only with the icon.
func paidOnlyWith(ec *engine.EventCard, icon string) bool {
	if len(ec.Paid.Icons) == 0 {
		return false
	}
	for _, ic := range ec.Paid.Icons {
		if ic != icon {
			return false
		}
	}
	return true
}
