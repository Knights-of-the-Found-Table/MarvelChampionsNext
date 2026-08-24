// complete.go implements the remaining Storm pack cards (36014–36039):
// the shared X-Men leadership suite, the Callisto nemesis set and the
// Shadow King modular set.
package storm

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerRemainingStorm()
}

func registerRemainingStorm() {
	// 36014 Havok: when he attacks, mill 1 — +1 damage per boost icon
	// (extra consequential damage not modeled).
	engine.RegisterBehavior("36014", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyAttackWindow)
			a := g.Allies[e.EID()]
			if !ok || m.Ally != e.EID() || a == nil || len(g.EncounterDeck) == 0 {
				return nil
			}
			top := g.EncounterDeck[0]
			g.EncounterDeck = g.EncounterDeck[1:]
			g.EncounterDiscard = append(g.EncounterDiscard, top)
			boost := cardutil.BoostOf(top)
			if boost > 0 {
				g.Logf("Havok overcharges (+%d)", boost)
				return []engine.Message{engine.DamageEntity{Target: m.Target, Damage: boost, Source: a.Owner}}
			}
			return nil
		},
	})

	// 36015 Mirage: on enter — stun an enemy with SCH below her THW.
	engine.RegisterBehavior("36015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(e.EOwner())
			if a == nil || p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn != nil && mn.SchemeVal < a.ThwartVal {
					choices = append(choices, engine.Choice{
						Label: "Stun " + cardutil.EnemyLabel(mn), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.StunEntity{Target: id}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Mirage — stun an enemy with lower SCH", choices...),
			}}
		},
	})

	// 36016 Gentle: the villain-attack consequential rider is not
	// modeled.
	engine.RegisterBehavior("36016", &engine.Behavior{})

	// 36017 Pixie: on enter — return an X-Men ally from discard to hand.
	engine.RegisterBehavior("36017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, c := range p.Discard {
				if c.Def().Type == "ally" && c.Def().HasTrait("x-men") {
					choices = append(choices, engine.Choice{
						Label: "Return " + c.Def().Name + " to hand", Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Pixie — return an X-Men ally to hand", choices...),
			}}
		},
	})

	// 36018 Uncanny X-Men: X-Men allies get +1 HP (applied on entry; the
	// discount rider is not modeled).
	engine.RegisterBehavior("36018", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyEnteredPlay)
			s := g.Supports[e.EID()]
			if !ok || s == nil {
				return nil
			}
			a := g.Allies[m.Ally]
			if a != nil && a.Owner == s.Owner && a.EDef().HasTrait("x-men") {
				a.MaxHP++
			}
			return nil
		},
	})

	// 36019 Leadership Skill: 3 counters; an ally's basic attack/thwart
	// gets +1.
	engine.RegisterBehavior("36019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil || u.Exhausted || u.Counters <= 0 {
				return nil
			}
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				if m.Ally != u.AttachTo && !alliedToOwner(g, m.Ally, u.Owner) {
					return nil
				}
				u.Counters--
				return []engine.Message{engine.DamageEntity{Target: m.Target, Damage: 1, Source: u.Owner}}
			case engine.AllyThwartWindow:
				if m.Ally != u.AttachTo && !alliedToOwner(g, m.Ally, u.Owner) {
					return nil
				}
				u.Counters--
				return []engine.Message{engine.ThwartScheme{Scheme: m.Scheme, N: 1, Source: u.Owner}}
			}
			return nil
		},
	})

	// 36020 "To Me, My X-Men!": search the top 5 for an X-Men ally and
	// put it into play (approximation: added to hand).
	engine.RegisterBehavior("36020", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "x-men")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for i := 0; i < 5 && i < len(p.Deck); i++ {
				c := p.Deck[i]
				if c.Def().Type == "ally" && c.Def().HasTrait("x-men") {
					p.Deck = append(p.Deck[:i], p.Deck[i+1:]...)
					c.Owner = p.ID
					p.Hand = append(p.Hand, c)
					g.Logf("\"To Me, My X-Men!\" — %s answers the call", c.Def().Name)
					return nil
				}
			}
			return nil
		},
	})

	// 36021 Effective Leadership: rider in handlePlayCard.
	engine.RegisterBehavior("36021", &engine.Behavior{})

	// 36022 Forge: on enter — take an X-Men/X-Force support to hand.
	engine.RegisterBehavior("36022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, c := range p.Deck {
				if c.Def().Type == "support" && (c.Def().HasTrait("x-men") || c.Def().HasTrait("x-force")) {
					choices = append(choices, engine.Choice{
						Label: "Take " + c.Def().Name + " (deck)", Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID}))
					break
				}
			}
			for _, c := range p.Discard {
				if c.Def().Type == "support" && (c.Def().HasTrait("x-men") || c.Def().HasTrait("x-force")) {
					choices = append(choices, engine.Choice{
						Label: "Take " + c.Def().Name + " (discard)", Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
					break
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Forge — take a support card", choices...),
			}}
		},
	})

	// 36023 The X-Jet / 36025 X-Mansion: reprints.
	engine.RegisterBehavior("36023", &engine.Behavior{Resource: &engine.ResourceAbility{Icon: "wild"}})
	engine.RegisterBehavior("36025", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "X-Mansion — heal 1 damage from a MUTANT or X-Men character", Type: engine.AbilityAction,
				AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.ActiveTurn)
					if p == nil {
						return nil
					}
					var choices []engine.Choice
					add := func(id engine.EntityID, label string, ok bool) {
						if ok {
							choices = append(choices, engine.Choice{
								Label: "Heal " + label, Kind: engine.ChoiceTarget, SourceID: id,
							}.Msgs(engine.HealEntity{Target: id, N: 1}))
						}
					}
					add(p.ID, p.Name, g.EntityHasTrait(p.ID, "x-men") || g.EntityHasTrait(p.ID, "mutant"))
					for _, id := range p.Allies {
						if a := g.Allies[id]; a != nil {
							add(id, a.EDef().Name, a.EDef().HasTrait("x-men") || a.EDef().HasTrait("mutant"))
						}
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask("X-Mansion — heal 1 damage", choices...),
					}}
				},
			}}
		},
	})

	// 36024 Utopia: reprint of the Cyclops implementation.
	engine.RegisterBehavior("36024", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyEnteredPlay)
			s := g.Supports[e.EID()]
			if !ok || s == nil || s.Exhausted {
				return nil
			}
			a := g.Allies[m.Ally]
			if a == nil || !a.EDef().HasTrait("x-men") || a.Owner != s.Owner {
				return nil
			}
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			if g.EntityHasTrait(p.ID, "x-men") {
				choices = append(choices, engine.Choice{
					Label: "Ready " + p.Name, Kind: engine.ChoiceTarget, SourceID: p.ID,
				}.Msgs(engine.ExhaustEntity{ID: s.ID}, engine.ReadyEntity{ID: p.ID}))
			}
			for _, id := range p.Allies {
				if x := g.Allies[id]; x != nil && x.EDef().HasTrait("x-men") {
					choices = append(choices, engine.Choice{
						Label: "Ready " + x.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.ExhaustEntity{ID: s.ID}, engine.ReadyEntity{ID: id}))
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Utopia — ready an X-Men character", choices...),
			}}
		},
	})

	// 36026 Endurance: +3 max hit points.
	engine.RegisterBehavior("36026", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP += 3
				g.Logf("%s gets +3 hit points (Endurance)", p.Name)
			}
			return nil
		},
	})

	// 36027–36029 basic resources.
	for _, code := range []string{"36027", "36028", "36029"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 36030 Claustrophobia: alter-ego exhaust removes it.
	engine.RegisterBehavior("36030", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			choices := []engine.Choice{engine.Choice{
				ID: "keep", Label: "Keep Claustrophobia in play (no form change)", Kind: engine.ChoiceLabel,
			}.Msgs(engine.ObligationResolve{Player: p.ID, Card: card})}
			if !p.IsHero() && !p.Exhausted {
				choices = append(choices, engine.Choice{
					ID: "exhaust", Label: "Exhaust Ororo Munroe → remove from the game", Kind: engine.ChoiceLabel,
				}.Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			} else if p.IsHero() && !p.FormChanged && !p.Exhausted {
				choices = append(choices, engine.Choice{
					ID: "flip", Label: "Flip to alter-ego form (exhaust to remove next)", Kind: engine.ChoiceLabel,
				}.Msgs(engine.ChangeForm{Player: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Claustrophobia — choose:", choices...),
			}}
		},
	})

	// 36031 Callisto: when Knife Fight is revealed, she gains tough.
	engine.RegisterBehavior("36031", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.RevealEncounterCard)
			if !ok || m.Card.Code != "36034" {
				return nil
			}
			if mn := g.Minions[e.EID()]; mn != nil {
				return []engine.Message{engine.ToughEntity{Target: mn.ID}}
			}
			return nil
		},
	})

	// 36032 Leader of the Morlocks: When Defeated — reveal Knife Fight.
	engine.RegisterBehavior("36032", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "36034" {
						zone.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
					}
				}
			}
			return nil
		},
	})

	// 36033 Switchblade: attach to the highest-ATK minion (piercing
	// cosmetic).
	engine.RegisterBehavior("36033", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best *engine.Minion
			for _, mn := range g.Minions {
				if mn == nil {
					continue
				}
				if best == nil || mn.AttackVal > best.AttackVal {
					best = mn
				}
			}
			if best == nil {
				g.Delete(t.ID)
				return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
			}
			t.Target = best.ID
			best.Attachments = append(best.Attachments, t.ID)
			return nil
		},
	})

	// 36034 Knife Fight: alter-ego — surge; hero — clash with the
	// highest-ATK enemy.
	engine.RegisterBehavior("36034", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if !p.IsHero() {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			var target engine.EntityID
			best := -1
			for _, id := range cardutil.SortedIDs(g.Villains) {
				if v := g.Villains[id]; v != nil && v.AttackVal > best {
					best, target = v.AttackVal, id
				}
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.AttackVal > best {
					best, target = mn.AttackVal, id
				}
			}
			if target == "" {
				return nil
			}
			return []engine.Message{
				engine.DamageEntity{Target: p.ID, Damage: best, Source: t.ID},
				engine.DamageEntity{Target: target, Damage: p.AttackStat(g), Source: p.ID},
			}
		},
	})

	// 36035 Hangar Bay: after an ally defends and survives, exhaust →
	// ready it.
	engine.RegisterBehavior("36035", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowDefended)
			s := g.Supports[e.EID()]
			if !ok || s == nil || s.Exhausted || !m.Defender.Is(engine.KindAlly) {
				return nil
			}
			if a := g.Allies[m.Defender]; a != nil && a.Owner == s.Owner {
				return []engine.Message{engine.ExhaustEntity{ID: s.ID}, engine.ReadyEntity{ID: a.ID}}
			}
			return nil
		},
	})

	// 36036 The Shadow King: reveal — search Possessed; the
	// Controlled-minion damage shield is approximated away.
	engine.RegisterBehavior("36036", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "36038" {
						zone.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: m.Player, Card: c}}
					}
				}
			}
			return nil
		},
	})

	// 36037 Ruler of the Astral Plane: When Defeated — discard a
	// Possessed from play.
	engine.RegisterBehavior("36037", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			for _, id := range cardutil.SortedIDs(g.Attachments) {
				if a := g.Attachments[id]; a != nil && a.Code == "36038" {
					return []engine.Message{engine.DiscardAttachmentMsg{ID: id}}
				}
			}
			return nil
		},
	})

	// 36038 Possessed: attach to the lowest-THW ally and convert it into
	// an enemy minion (blank text; SCH = printed THW).
	engine.RegisterBehavior("36038", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best *engine.Ally
			for _, p := range g.Players {
				for _, id := range p.Allies {
					a := g.Allies[id]
					if a == nil {
						continue
					}
					possessed := false
					for _, x := range g.Attachments {
						if x != nil && x.Code == "36038" && x.Target == a.ID {
							possessed = true
						}
					}
					if possessed {
						continue
					}
					if best == nil || a.ThwartVal < best.ThwartVal {
						best = a
					}
				}
			}
			if best == nil {
				g.Delete(t.ID)
				return nil
			}
			owner := best.Owner
			code, thw, atk := best.Code, best.ThwartVal, best.AttackVal
			hp := best.MaxHP - best.Damage
			pid := owner
			g.Delete(best.ID)
			mn := &engine.Minion{
				ID: g.NextEntityID(engine.KindMinion), Code: code,
				MaxHP: hp, AttackVal: atk, SchemeVal: thw, EngagedWith: pid,
			}
			g.Minions[mn.ID] = mn
			g.Logf("Possessed — %s turns against %s!", best.EDef().Name, g.Player(pid).Name)
			return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: pid}}
		},
	})

	// 36039 Astral Attack: each engaged minion activates; none → surge;
	// boost shuffles Shadow King cards back.
	engine.RegisterBehavior("36039", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EngagedWith == p.ID {
					msgs = append(msgs, engine.MinionActivates{MinionID: id, Player: p.ID})
				}
			}
			if len(msgs) == 0 {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			shuffled := 0
			for i := 0; i < len(g.EncounterDiscard); {
				c := g.EncounterDiscard[i]
				if c.Def().CardSet == "shadow_king" {
					g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
					g.EncounterDeck = append(g.EncounterDeck, c)
					shuffled++
					continue
				}
				i++
			}
			if shuffled > 0 {
				g.Logf("The Shadow King reclaims %d cards", shuffled)
			}
			return nil
		},
	})
}

// alliedToOwner reports whether the ally belongs to the player.
func alliedToOwner(g *engine.Game, ally engine.EntityID, pid engine.PlayerID) bool {
	a := g.Allies[ally]
	return a != nil && a.Owner == pid
}
