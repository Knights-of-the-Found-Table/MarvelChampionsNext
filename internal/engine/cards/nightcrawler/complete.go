package nightcrawler

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func registerNightcrawlerExtras() {
	// 48012 Rogue: copy a friendly character (approximated: +their ATK/THW
	// for the round, once per round).
	engine.RegisterBehavior("48012", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			p := g.Player(a.Owner)
			if p == nil || len(p.Allies) < 2 {
				return nil
			}
			return []engine.Ability{{
				Label: "Rogue — absorb another friendly character", Type: engine.AbilityAction,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Allies[self]
					p := g.Player(a.Owner)
					if p == nil {
						return nil
					}
					for _, id := range p.Allies {
						if id == self {
							continue
						}
						x := g.Allies[id]
						if x == nil {
							continue
						}
						return []engine.Message{
							engine.DamageEntity{Target: id, Damage: 1, Source: p.ID},
							engine.AllyStatBonus{Ally: self, ATK: x.AttackVal, THW: x.ThwartVal},
						}
					}
					return nil
				},
			}}
		},
	})

	// 48013 Northstar: cancel boost icons for a damage.
	engine.RegisterBehavior("48013", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.RevealBoost)
			if !ok || !engine.AttackActivationPending(g, m.Enemy) {
				return nil
			}
			a := g.Allies[e.EID()]
			if a == nil || a.HP() < 2 {
				return nil
			}
			add := 0
			if v := g.Villains[m.Enemy]; v != nil {
				for _, c := range v.RevealedBoosts {
					add += cardutil.BoostOf(c)
				}
			}
			if add <= 0 {
				return nil
			}
			g.Logf("Northstar blurs the boost cards away (-%d)", add)
			return []engine.Message{
				engine.DamageEntity{Target: a.ID, Damage: 1, Source: a.Owner},
				engine.CancelBoostIcons{Enemy: m.Enemy, N: add},
			}
		},
	})

	// 48016 "Come Get Me, Bub!": summon + heal.
	engine.RegisterBehavior("48016", &engine.Behavior{
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
					return []engine.Message{
						engine.RevealEncounterCard{Player: p.ID, Card: c},
						engine.HealEntity{Target: p.ID, N: 3},
						engine.ToughEntity{Target: p.ID},
					}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})

	// 48017 Powerful Punch: 4 damage on enemy attack (defense event).
	engine.RegisterBehavior("48017", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			d := engine.Defends{Defender: p.ID, Against: against, Undefended: true}
			return d, []engine.Message{engine.DamageEntity{Target: against, Damage: 4, Source: p.ID}}, true
		},
	})

	// 48018 Riposte: +2 DEF + counter.
	engine.RegisterBehavior("48018", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			d := engine.Defends{Defender: p.ID, Against: against, DefBonus: 2}
			return d, []engine.Message{engine.DamageEntity{Target: against, Damage: 3, Source: p.ID}}, true
		},
	})

	// 48019 The Power of Protection: engine powerOfBonus.
	engine.RegisterBehavior("48019", &engine.Behavior{})

	// 48020 Astonishing X-Men: shed threat on clean defense; defeat
	// stuns/confuses.
	engine.RegisterBehavior("48020", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			if !ok || w.DamageTaken != 0 {
				return nil
			}
			s := g.SideSchemes[e.EID()]
			if s == nil || s.Threat <= 0 {
				return nil
			}
			s.Threat--
			return nil
		},
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var msgs []engine.Message
			for _, id := range cardutil.SortedEnemyIDs(g) {
				msgs = append(msgs, engine.StunEntity{Target: id}, engine.ConfuseEntity{Target: id})
			}
			return msgs
		},
	})

	// 48021 Gambit: X from a tucked boost card (approximated: tucked card
	// adds +2 ATK/THW).
	engine.RegisterBehavior("48021", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			if c, ok := g.DrawEncounter(); ok {
				g.EncounterDiscard = append(g.EncounterDiscard, c)
				b := cardutil.BoostOf(c)
				if b < 1 {
					b = 1
				}
				a.PermATK += b
				a.PermTHW += b
				g.Logf("Gambit charges up with %s (+%d)", c.Def().Name, b)
			}
			return nil
		},
	})

	// 48022 Moira MacTaggert: draw on mutant transformations.
	engine.RegisterBehavior("48022", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "Mutant")
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.ChangeForm)
			if !ok || e.EExhausted() {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil || s.Owner != m.Player {
				return nil
			}
			if p := g.Player(m.Player); p != nil && p.IsHero() {
				return []engine.Message{engine.ExhaustEntity{ID: s.ID}, engine.DrawCards{Player: p.ID, N: 1}}
			}
			return nil
		},
	})

	// 48023-48025 Energy/Genius/Strength: deckbuilding.
	for _, code := range []string{"48023", "48024", "48025"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 48027 Azazel: boost self-deals facedown.
	engine.RegisterBehavior("48027", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
				p.EncounterDown = append(p.EncounterDown, card)
			}
			return nil
		},
	})

	// 48031 Combine Forces: exhaust two to kill a minion.
	engine.RegisterBehavior("48031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var exhausted []engine.Message
			n := 0
			var targets []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn != nil && !mn.EDef().HasTrait("Elite") {
					targets = append(targets, engine.Choice{
						ID: "mn-" + id.String(), Label: mn.EDef().Name, Kind: engine.ChoiceTarget,
					}.Msgs(engine.MinionDefeated{MinionID: id}))
				}
			}
			if g.EntityHasTrait(p.ID, "X-Force") || g.EntityHasTrait(p.ID, "X-Men") {
				exhausted = append(exhausted, engine.ExhaustEntity{ID: p.ID})
				n++
			}
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil || a.Exhausted {
					continue
				}
				if (a.EDef().HasTrait("X-Force") || a.EDef().HasTrait("X-Men")) && n < 2 {
					exhausted = append(exhausted, engine.ExhaustEntity{ID: id})
					n++
				}
			}
			if n < 2 || len(targets) == 0 {
				return nil
			}
			return append([]engine.Message{engine.AskQuestion{
				Player: p.ID, Question: engine.Ask("Combine Forces — defeat:", targets...),
			}}, exhausted...)
		},
	})

	// 48032 Gunboat Diplomacy: exhaust two for split effects.
	engine.RegisterBehavior("48032", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			thw := 0
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && (a.EDef().HasTrait("X-Force") || a.EDef().HasTrait("X-Men")) && !a.Exhausted {
					thw = a.ThwartVal + a.AttackVal
					break
				}
			}
			if thw == 0 {
				return nil
			}
			var msgs []engine.Message
			if g.MainScheme != nil {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: thw, Source: p.ID})
			}
			if len(g.Enemies()) > 0 {
				msgs = append(msgs, engine.DamageEntity{Target: g.Enemies()[0], Damage: thw, Source: p.ID})
			}
			return msgs
		},
	})

	registerCrazyGang()
}

func registerCrazyGang() {
	// 48033 The Crazy Gang: minion schemes become facedown handouts.
	engine.RegisterBehavior("48033", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionActivates); !ok {
				return nil
			}
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			return nil
		},
	})

	// 48034 Queen of Hearts: fetch the scheme.
	engine.RegisterBehavior("48034", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			for _, s := range g.SideSchemes {
				if s != nil && s.Code == "48033" {
					p := g.Player(mn.EngagedWith)
					if p != nil {
						return []engine.Message{engine.DealEncounterToPlayer{Player: p.ID}}
					}
					return nil
				}
			}
			for _, c := range append(engine.CardList{}, g.EncounterDeck...) {
				if c.Code == "48033" {
					g.EncounterDeck.Remove(c.ID)
					g.ShuffleEncounterDeck()
					return []engine.Message{engine.RevealEncounterCard{Player: mn.EngagedWith, Card: c}}
				}
			}
			return nil
		},
	})

	// 48035 Jester: confuse or lose a support.
	engine.RegisterBehavior("48035", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			if p.Confused && len(p.Supports) > 0 {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Supports[0]}}
			}
			return []engine.Message{engine.ConfuseEntity{Target: p.ID}}
		},
	})

	// 48036 Executioner: attack the weakest.
	engine.RegisterBehavior("48036", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			best, bestID := 1<<30, engine.EntityID("")
			if p.HP() < best {
				best, bestID = p.HP(), p.ID
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.HP() < best {
					best, bestID = a.HP(), id
				}
			}
			if bestID != "" {
				return []engine.Message{engine.DamageEntity{Target: bestID, Damage: mn.AttackVal, Source: mn.ID}}
			}
			return nil
		},
	})

	// 48037 Tweedledope: stun or lose an upgrade.
	engine.RegisterBehavior("48037", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			if p.Stunned && len(p.Upgrades) > 0 {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[0]}}
			}
			return []engine.Message{engine.StunEntity{Target: p.ID}}
		},
	})

	// 48038 "Off with His Head!": mass minion activation.
	engine.RegisterBehavior("48038", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EngagedWith != "" {
					msgs = append(msgs, engine.MinionActivates{MinionID: id, Player: mn.EngagedWith})
				}
			}
			if len(msgs) > 0 {
				return msgs
			}
			for i := 0; i < 30; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					return nil
				}
				if c.Def().Type == "minion" {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})
}
