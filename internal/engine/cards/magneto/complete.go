package magneto

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func registerMagnetoExtras() {
	// 49012 M: kill a smaller minion.
	engine.RegisterBehavior("49012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			for _, mn := range g.Minions {
				if mn != nil && mn.HP() < a.MaxHP {
					g.TLogf("c.mObliterates", mn)
					return []engine.Message{engine.MinionDefeated{MinionID: mn.ID}}
				}
			}
			return nil
		},
	})

	// 49013 Kid Omega: pay energy or mental for a wave.
	engine.RegisterBehavior("49013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			var energy, mental *engine.Card
			for i := range p.Hand {
				for _, r := range p.Hand[i].Def().Resources {
					if r == "energy" || r == "wild" && energy == nil {
						energy = &p.Hand[i]
					}
					if r == "mental" || r == "wild" && mental == nil {
						mental = &p.Hand[i]
					}
				}
			}
			var choices []engine.Choice
			if energy != nil {
				var msgs []engine.Message
				msgs = append(msgs, engine.DiscardCards{Player: p.ID, Cards: engine.CardList{*energy}})
				for _, id := range cardutil.SortedEnemyIDs(g) {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: p.ID})
				}
				choices = append(choices, engine.Choice{ID: "energy", Label: engine.Tf("c.spendEnergyStrike", energy), Kind: engine.ChoiceLabel}.Msgs(msgs...))
			}
			if mental != nil {
				var msgs []engine.Message
				msgs = append(msgs, engine.DiscardCards{Player: p.ID, Cards: engine.CardList{*mental}})
				for _, id := range g.Schemes() {
					msgs = append(msgs, engine.ThwartScheme{Scheme: id, N: 1, Source: p.ID})
				}
				choices = append(choices, engine.Choice{ID: "mental", Label: engine.Tf("c.spendSchemeThreat", mental), Kind: engine.ChoiceLabel}.Msgs(msgs...))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.kidOmegaChoose"), choices...)}}
		},
	})

	// 49014 Phoenix: ready + heal an X-Men ally.
	engine.RegisterBehavior("49014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			for _, id := range p.Allies {
				x := g.Allies[id]
				if x != nil && x.EDef().HasTrait("X-Men") {
					return []engine.Message{engine.ReadyEntity{ID: id}, engine.HealEntity{Target: id, N: 1}}
				}
			}
			return nil
		},
	})

	// 49015 Cyclops: mark a villain for +1 damage taken (approximation:
	// event bonus pool).
	engine.RegisterBehavior("49015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			g.EventDamageBonus[p.ID] += 1
			g.TLogf("c.cyclopsPaintsATarget1DamageThisPhase")
			return nil
		},
	})

	// 49016 Won't Stay Down: fetch a fallen mutant.
	engine.RegisterBehavior("49016", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Force") || g.EntityHasTrait(p.ID, "X-Men")
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			for _, c := range p.Discard {
				d := c.Def()
				if d.Type == "ally" && (d.HasTrait("X-Force") || d.HasTrait("X-Men")) {
					card := c
					return []engine.Ability{{
						Label: engine.Tf("c.wontStayDownReturn", card), Type: engine.AbilityAction,
						AlterEgoOnly: true,
						Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
							s := g.Supports[self]
							p := g.Player(s.Owner)
							if p == nil {
								return nil
							}
							if _, ok := p.Discard.Remove(card.ID); ok {
								p.Hand = append(p.Hand, card)
							}
							g.Delete(s.ID)
							g.TLogf("c.wonTStayDownIsDiscarded")
							return nil
						},
					}}
				}
			}
			return nil
		},
	})

	// 49017 Squared Off: summon a minion, discount an ally.
	engine.RegisterBehavior("49017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for i := 0; i < 30; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					return nil
				}
				if c.Def().Type == "minion" {
					g.TLogf("c.squaredOffDragsIntoTheFight", c)
					return []engine.Message{
						engine.RevealEncounterCard{Player: p.ID, Card: c},
						engine.CostDiscountApply{Player: p.ID, Amount: 3},
					}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})

	// 49018 Noble Sacrifice: trade an ally for healing.
	engine.RegisterBehavior("49018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Allies) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil {
					continue
				}
				choices = append(choices, engine.Choice{
					ID: "ally-" + id.String(), Label: engine.Tf("m.cardName", a), Kind: engine.ChoiceTarget,
				}.Msgs(engine.AllyDestroyed{AllyID: id},
					engine.HealEntity{Target: p.ID, N: a.MaxHP},
					engine.ToughEntity{Target: p.ID}))
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID, Question: engine.Ask(engine.Tf("c.nobleSacrificeGiveUp"), choices...),
			}}
		},
	})

	// 49019 "You Got This!": after a basic power, trade an ally to ready.
	engine.RegisterBehavior("49019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Allies) == 0 || !p.Exhausted {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil {
					continue
				}
				choices = append(choices, engine.Choice{
					ID: "ally-" + id.String(), Label: engine.Tf("m.cardName", a), Kind: engine.ChoiceTarget,
				}.Msgs(engine.AllyDestroyed{AllyID: id}, engine.ReadyEntity{ID: p.ID}))
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID, Question: engine.Ask(engine.Tf("c.youGotThisSacrifice"), choices...),
			}}
		},
	})

	// 49020 New Recruits: defeat grabs a New ally.
	engine.RegisterBehavior("49020", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Men")
		},
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, p := range g.Players {
				if c, zone, ok := firstWhere(p, func(d *data.CardDef) bool {
					return d.Type == "ally" && d.HasTrait("New")
				}); ok {
					take(p, c, zone)
					p.Hand = append(p.Hand, c)
					g.TLogf("c.recruits", p.Name, c)
				}
			}
			return nil
		},
	})

	// 49021 White Queen: shed a status card.
	engine.RegisterBehavior("49021", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "X-Force") || g.EntityHasTrait(p.ID, "X-Men")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, p := range g.Players {
				if p.Stunned {
					p.Stunned = false
					g.TLogf("c.whiteQueenClearsAStun")
					return nil
				}
				if p.Confused {
					p.Confused = false
					g.TLogf("c.whiteQueenClearsAConfuse")
					return nil
				}
				for _, mn := range g.Minions {
					if mn != nil && (mn.Stunned || mn.Confused || mn.Tough) {
						mn.Stunned, mn.Confused, mn.Tough = false, false, false
						g.TLogf("c.whiteQueenClearsSStatus", mn)
						return nil
					}
				}
			}
			return nil
		},
	})

	// 49022 Face the Past: nemesis hunt (the attack lockout is
	// approximated away).
	engine.RegisterBehavior("49022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			// The nemesis minion ships in the nemesis deck; pull the top
			// minion from it.
			for i, c := range p.NemesisDeck {
				if c.Def().Type == "minion" {
					p.NemesisDeck = append(p.NemesisDeck[:i:i], p.NemesisDeck[i+1:]...)
					return []engine.Message{
						engine.RevealNemesisSet{Player: p.ID},
						engine.ReadyEntity{ID: p.ID},
						engine.DrawCards{Player: p.ID, N: 3},
					}
				}
			}
			return []engine.Message{engine.ReadyEntity{ID: p.ID}, engine.DrawCards{Player: p.ID, N: 3}}
		},
	})

	// 49023 Deft Focus: discount the next superpower.
	engine.RegisterBehavior("49023", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.deftFocusNextSuperpowerCosts1Less"), Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					return []engine.Message{engine.CostDiscountApply{Player: u.Owner, Amount: 1}}
				},
			}}
		},
	})

	// 49024-49026 Energy/Genius/Strength: deckbuilding limits only.
	for _, code := range []string{"49024", "49025", "49026"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 49033-49036 New allies: trait-scaled bonuses applied on entry.
	newAlly := func(code string, apply func(a *engine.Ally, p *engine.Player, g *engine.Game)) {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
				a := g.Allies[e.EID()]
				if a == nil {
					return nil
				}
				if p := g.Player(a.Owner); p != nil && (g.EntityHasTrait(p.ID, "Mutant") || g.EntityHasTrait(p.ID, "X-Men")) {
					apply(a, p, g)
				}
				return nil
			},
		})
	}
	newAlly("49033", func(a *engine.Ally, p *engine.Player, g *engine.Game) { a.PermATK++ })
	newAlly("49034", func(a *engine.Ally, p *engine.Player, g *engine.Game) { a.PermTHW++ })
	newAlly("49035", func(a *engine.Ally, p *engine.Player, g *engine.Game) { a.Tough = true })
	newAlly("49036", func(a *engine.Ally, p *engine.Player, g *engine.Game) { a.MaxHP += 2 })

	// 49037 Children of the Atom: cross-grant X-traits.
	engine.RegisterBehavior("49037", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.Supports[e.EID()]
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			grant := func(traits []string, has []string) []string {
				for _, t := range has {
					found := false
					for _, x := range traits {
						if x == t {
							found = true
						}
					}
					if !found {
						traits = append(traits, t)
					}
				}
				return traits
			}
			all := []string{"X-Factor", "X-Force", "X-Men"}
			p.ExtraTraits = grant(p.ExtraTraits, all)
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					if a.EDef().HasTrait("X-Factor") || a.EDef().HasTrait("X-Force") || a.EDef().HasTrait("X-Men") {
						a.ExtraTraits = grant(a.ExtraTraits, all)
					}
				}
			}
			g.TLogf("c.childrenOfTheAtomUnitesTheXFamily")
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AllyEnteredPlay); !ok {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil {
				return nil
			}
			if p := g.Player(s.Owner); p != nil {
				found := false
				for _, t := range p.ExtraTraits {
					if t == "X-Men" {
						found = true
					}
				}
				if !found {
					p.ExtraTraits = append(p.ExtraTraits, "X-Factor", "X-Force", "X-Men")
				}
			}
			return nil
		},
	})
}

func registerHellfire() {
	// 49038 Sebastian Shaw: facedown boost per attack (approximated: +1
	// ATK counter per attack against him).
	engine.RegisterBehavior("49038", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.BasicAttack); !ok {
				return nil
			}
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			mn.Counters++
			g.TLogf("c.sebastianShawAbsorbsTheBlowAtk", mn.Counters)
			return nil
		},
		EnemyStatBonus: func(g *engine.Game, e engine.Entity) (atk, sch int) {
			if mn := g.Minions[e.EID()]; mn != nil {
				return mn.Counters, 0
			}
			return 0, 0
		},
	})

	// 49039 Selene: allies cannot attack her (engine ally attack filter
	// approximated); boost sacrifices an ally.
	engine.RegisterBehavior("49039", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil && len(p.Allies) > 0 {
				return []engine.Message{engine.AllyDestroyed{AllyID: p.Allies[0]}}
			}
			return nil
		},
	})

	// 49040 Hellfire Pawn: boost self-deploys.
	engine.RegisterBehavior("49040", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: card}}
		},
	})

	// 49041 The Inner Circle: threat per Hellfire card.
	engine.RegisterBehavior("49041", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			n := 0
			for _, mn := range g.Minions {
				if mn != nil && mn.EDef().HasTrait("Hellfire") {
					n++
				}
			}
			s.Threat += 2 * n
			return nil
		},
	})

	// 49042 Power and Decadence: tough + extra activation.
	engine.RegisterBehavior("49042", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			if id := firstVillainID(g); id != "" {
				msgs = append(msgs, engine.ToughEntity{Target: id})
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if id := firstVillainID(g); id != "" {
				p := g.Player(cardutil.FirstPlayerID(g))
				if p != nil {
					return []engine.Message{engine.VillainActivates{VillainID: id, Player: p.ID}}
				}
			}
			return nil
		},
	})
}

func firstWhere(p *engine.Player, pred func(*data.CardDef) bool) (engine.Card, string, bool) {
	for _, c := range p.Deck {
		if pred(c.Def()) {
			return c, "deck", true
		}
	}
	for _, c := range p.Discard {
		if pred(c.Def()) {
			return c, "discard", true
		}
	}
	return engine.Card{}, "", false
}

func take(p *engine.Player, c engine.Card, zone string) bool {
	switch zone {
	case "deck":
		_, ok := p.Deck.Remove(c.ID)
		return ok
	case "discard":
		_, ok := p.Discard.Remove(c.ID)
		return ok
	}
	return false
}

func firstVillainID(g *engine.Game) engine.EntityID {
	for id := range g.Villains {
		return id
	}
	return ""
}
