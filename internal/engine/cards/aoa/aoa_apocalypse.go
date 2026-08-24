package aoa

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// apocalypse finds the Apocalypse villain (either scenario family).
func apocalypse(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil {
			base := engine.BaseCodeOf(v.Code)
			if base == "45101" || base == "45184" || base == "45185" || base == "45186" {
				return v
			}
		}
	}
	return nil
}

// printedHPNumeral returns Apocalypse's printed hit point value (X in
// The Age of Apocalypse's text).
func printedHPNumeral(v *engine.Villain) int {
	if v == nil || v.MaxHP <= 0 {
		return 1
	}
	return v.MaxHP
}

func registerApocalypse() {
	// 45101/45102 Apocalypse (II)/(IV): when the main scheme completes, it
	// clears instead and the villain flips to the next stage.
	apocBehavior := &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MainSchemeMaxed); !ok {
				return nil
			}
			v := g.Villains[e.EID()]
			if v == nil || g.MainScheme == nil {
				return nil
			}
			g.MainScheme.Threat = 0
			g.Logf("%s clears the main scheme and transforms!", v.EDef().Name)
			return []engine.Message{engine.AdvanceVillainStage{VillainID: v.ID}}
		},
		VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
			// The Age of Apocalypse: a would-be defeat strips
			// attachments, heals him, and drains the scheme instead.
			if g.MainScheme == nil || engine.BaseCodeOf(g.MainScheme.Code) != "45103" {
				return true
			}
			if v.HP()-damage > 0 {
				return true
			}
			for _, a := range g.Attachments {
				if a != nil && a.Target == v.ID {
					g.Delete(a.ID)
					g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
				}
			}
			v.Damage = 0
			x := printedHPNumeral(v)
			if g.MainScheme.Threat < x {
				x = g.MainScheme.Threat
			}
			g.MainScheme.Threat -= x
			g.Logf("Apocalypse sheds his trinkets and endures! (%d threat removed from the scheme)", x)
			return false
		},
	}
	engine.RegisterBehavior("45101", apocBehavior)
	engine.RegisterBehavior("45102", apocBehavior)

	// 45103 The Age of Apocalypse: when Apocalypse would be defeated,
	// strip attachments, heal, remove X threat (approximated via the
	// villain damage gate + the scheme hook).
	engine.RegisterBehavior("45103", &engine.Behavior{})

	// 45104/45105 Heart of the Empire & The Tyrant's Throne: threat lock
	// while a Prelate lives; defeat reveals a set-aside Prelate.
	for _, code := range []string{"45104", "45105"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
				// Reveal a random set-aside Prelate minion.
				var prelims []engine.Card
				for _, c := range g.SetAside {
					if c.Def().HasTrait("Prelate") && c.Def().Type == "minion" {
						prelims = append(prelims, c)
					}
				}
				var msgs []engine.Message
				if len(prelims) > 0 {
					c := prelims[g.Random(len(prelims))]
					g.SetAside = removeCard(g.SetAside, c)
					msgs = append(msgs, engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c})
				}
				for _, p := range g.Players[1:] {
					msgs = append(msgs, engine.DealEncounterToPlayer{Player: p.ID})
				}
				return msgs
			},
		})
	}

	// 45106 Cyberpathy: after Apocalypse schemes, 1 threat on each side
	// scheme.
	engine.RegisterBehavior("45106", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := apocalypse(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.ApplyVillainScheme); !ok {
				return nil
			}
			t := g.Attachments[e.EID()]
			v := apocalypse(g)
			if t == nil || v == nil || t.Target != v.ID {
				return nil
			}
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				msgs = append(msgs, engine.ApplySchemeThreat{Scheme: id, N: 1, Source: t.ID})
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			attachToApocalypse(g, "45106")
			return nil
		},
	})

	// 45107 Biomorphing: overkill rider not modeled.
	engine.RegisterBehavior("45107", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := apocalypse(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			attachToApocalypse(g, "45107")
			return nil
		},
	})

	// 45108 Molecular Control: retaliate 1 + stalwart (retaliate via
	// engine attachment check).
	engine.RegisterBehavior("45108", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := apocalypse(g); v != nil {
				t.Target = v.ID
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			attachToApocalypse(g, "45108")
			return nil
		},
	})

	// 45109 The Fittest: buff the biggest minion.
	engine.RegisterBehavior("45109", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			best, bestID := -1, engine.EntityID("")
			for _, mn := range g.Minions {
				if mn != nil && mn.MaxHP > best {
					best, bestID = mn.MaxHP, mn.ID
				}
			}
			if bestID == "" {
				if c, ok := g.DrawEncounter(); ok {
					return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
				}
				return nil
			}
			t.Target = bestID
			if mn := g.Minions[bestID]; mn != nil {
				mn.MaxHP += 5
				mn.Tough = true
			}
			return nil
		},
	})

	// 45110 Wolf Among Sheep: the Prelate (else Apocalypse) activates.
	engine.RegisterBehavior("45110", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for _, mn := range g.Minions {
				if mn != nil && mn.EDef().HasTrait("Prelate") {
					return []engine.Message{engine.MinionActivates{MinionID: mn.ID, Player: p.ID}}
				}
			}
			if v := apocalypse(g); v != nil {
				return []engine.Message{engine.VillainActivates{VillainID: v.ID, Player: p.ID}}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for _, mn := range g.Minions {
				if mn != nil && mn.EDef().HasTrait("Prelate") {
					return []engine.Message{engine.ToughEntity{Target: mn.ID}}
				}
			}
			if v := apocalypse(g); v != nil {
				return []engine.Message{engine.ToughEntity{Target: v.ID}}
			}
			return nil
		},
	})

	// 45111 The Apocalypse Solution: defeat mills the encounter deck.
	engine.RegisterBehavior("45111", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			n := printedHPNumeral(apocalypse(g))
			for i := 0; i < n; i++ {
				if c, ok := g.DrawEncounter(); ok {
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				}
			}
			g.Logf("The Apocalypse Solution discards %d encounter cards", n)
			return nil
		},
	})
}

func attachToApocalypse(g *engine.Game, code string) {
	v := apocalypse(g)
	if v == nil {
		return
	}
	t := &engine.Attachment{ID: g.NextEntityID(engine.KindAttachment), Code: code, Target: v.ID}
	g.Attachments[t.ID] = t
	v.Attachments = append(v.Attachments, t.ID)
	g.Logf("%s attaches to Apocalypse", t.EDef().Name)
}

func removeCard(list engine.CardList, c engine.Card) engine.CardList {
	for i, x := range list {
		if x.ID == c.ID {
			return append(list[:i:i], list[i+1:]...)
		}
	}
	return list
}

// ---- En Sabah Nur (form-cycling Apocalypse) ----

// apocForms maps form traits to their villain card codes per stage.
var apocForms = map[string]string{
	"Biomorph": "45184a", "Cyberpath": "45184b", "Giant": "45184c",
}

// apocForm returns Apocalypse's current form trait.
func apocForm(g *engine.Game) string {
	v := apocalypse(g)
	if v == nil {
		return ""
	}
	if v.EDef().HasTrait("Biomorph") {
		return "Biomorph"
	}
	if v.EDef().HasTrait("Giant") {
		return "Giant"
	}
	if v.EDef().HasTrait("Cyberpath") {
		return "Cyberpath"
	}
	return ""
}

// changeApocForm flips Apocalypse's card to the requested form's card for
// his current stage.
func changeApocForm(g *engine.Game, form string) []engine.Message {
	v := apocalypse(g)
	if v == nil {
		return nil
	}
	stage := ""
	switch engine.BaseCodeOf(v.Code) {
	case "45184":
		stage = "45184"
	case "45185":
		stage = "45185"
	case "45186":
		stage = "45186"
	}
	if stage == "" {
		return nil
	}
	suffix := map[string]string{"Biomorph": "a", "Cyberpath": "b", "Giant": "c"}[form]
	code := stage + suffix
	if _, ok := engine.DB.Lookup(code); !ok {
		return nil
	}
	v.Code = code
	g.Logf("Apocalypse changes to %s form", form)
	switch form {
	case "Biomorph":
		var msgs []engine.Message
		n := map[string]int{"45184": 1, "45185": 2, "45186": 3}[stage]
		for _, p := range g.Players {
			msgs = append(msgs, engine.IndirectDamage{Player: p.ID, N: n})
		}
		return msgs
	case "Giant":
		n := map[string]int{"45184": 1, "45185": 2, "45186": 3}[stage]
		return []engine.Message{engine.HealEntity{Target: v.ID, N: n}}
	}
	return nil
}

func registerEnSabahNur() {
	// 45184/45185/45186: the form cards; base registration covers sides.
	for _, stage := range []string{"45184", "45185", "45186"} {
		engine.RegisterBehavior(stage, &engine.Behavior{})
	}

	// 45147/45148 pyramids: power counters mill for a Superpower.
	pyramid := &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BeginPhase)
			if !ok || m.Phase != engine.PhaseVillain || g.MainScheme == nil {
				return nil
			}
			if g.MainScheme.EID() != e.EID() {
				return nil
			}
			g.MainScheme.Counters++
			g.Logf("The pyramid gains a power counter (%d)", g.MainScheme.Counters)
			if g.MainScheme.Counters >= 4 {
				g.MainScheme.Counters -= 4
				p := g.Player(cardutil.FirstPlayerID(g))
				if p == nil {
					return nil
				}
				for i := 0; i < 30; i++ {
					c, ok := g.DrawEncounter()
					if !ok {
						return nil
					}
					if c.Def().HasTrait("Superpower") {
						return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
					}
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				}
			}
			return nil
		},
	}
	engine.RegisterBehavior("45147", pyramid)
	engine.RegisterBehavior("45148", pyramid)

	// 45148a The Rise of Apocalypse reveal rider: mill for a Superpower.
	// (Handled via the shared pyramid behavior's stage reveal below.)

	// 45149 Staggering Strength: attach + Giant form + stun rider.
	engine.RegisterBehavior("45149", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if v := apocalypse(g); v != nil {
				t.Target = v.ID
			}
			return changeApocForm(g, "Giant")
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok {
				return nil
			}
			t := g.Attachments[e.EID()]
			v := apocalypse(g)
			if t == nil || v == nil || m.Enemy != v.ID {
				return nil
			}
			g.Delete(t.ID)
			g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: t.Code})
			return []engine.Message{engine.StunEntity{Target: m.Player}}
		},
	})

	// 45150-45152 form treacheries: activate in form, else change + power
	// counter.
	formTreachery := func(form string) func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		return func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			v := apocalypse(g)
			if v == nil {
				return nil
			}
			if apocForm(g) == form {
				return []engine.Message{engine.VillainActivates{VillainID: v.ID, Player: p.ID}}
			}
			if g.MainScheme != nil {
				g.MainScheme.Counters++
			}
			return changeApocForm(g, form)
		}
	}
	engine.RegisterBehavior("45150", &engine.Behavior{
		ResolveTreachery: formTreachery("Biomorph"),
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return changeApocForm(g, "Biomorph")
		},
	})
	engine.RegisterBehavior("45151", &engine.Behavior{
		ResolveTreachery: formTreachery("Cyberpath"),
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return changeApocForm(g, "Cyberpath")
		},
	})
	engine.RegisterBehavior("45152", &engine.Behavior{
		ResolveTreachery: formTreachery("Giant"),
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return changeApocForm(g, "Giant")
		},
	})

	// 45153-45155 form side schemes.
	formScheme := func(form string) *engine.Behavior {
		return &engine.Behavior{
			SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
				v := apocalypse(g)
				if v == nil {
					return nil
				}
				p := g.Player(cardutil.FirstPlayerID(g))
				if apocForm(g) == form {
					return []engine.Message{engine.VillainActivates{VillainID: v.ID, Player: p.ID}}
				}
				msgs := changeApocForm(g, form)
				return append([]engine.Message{engine.ToughEntity{Target: v.ID}}, msgs...)
			},
		}
	}
	engine.RegisterBehavior("45153", formScheme("Biomorph"))
	engine.RegisterBehavior("45154", formScheme("Cyberpath"))
	engine.RegisterBehavior("45155", formScheme("Giant"))
}

func registerCelestialTech() {
	// 45156 Celestial Armor: scheme rider by milled resource.
	engine.RegisterBehavior("45156", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if id := firstVillainID(g); id != "" {
				t.Target = id
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.ApplyVillainScheme); !ok {
				return nil
			}
			t := g.Attachments[e.EID()]
			if t == nil || t.Target == "" {
				return nil
			}
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			c := p.Deck[0]
			msgs := []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 1}}
			for _, r := range c.Def().Resources {
				switch r {
				case "energy":
					msgs = append(msgs, engine.HealEntity{Target: t.Target, N: 2})
				case "mental":
					msgs = append(msgs, engine.ConfuseEntity{Target: p.ID})
					g.Delete(t.ID)
					g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: t.Code})
				case "physical":
					msgs = append(msgs, engine.ToughEntity{Target: t.Target})
				case "wild":
					msgs = append(msgs, engine.HealEntity{Target: t.Target, N: 2},
						engine.ConfuseEntity{Target: p.ID}, engine.ToughEntity{Target: t.Target})
				}
			}
			return msgs
		},
	})

	// 45157 Celestial Weapon: attack rider by milled resource.
	engine.RegisterBehavior("45157", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if id := firstVillainID(g); id != "" {
				t.Target = id
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AskAttack); !ok {
				return nil
			}
			t := g.Attachments[e.EID()]
			if t == nil || t.Target == "" {
				return nil
			}
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			c := p.Deck[0]
			msgs := []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 1}}
			for _, r := range c.Def().Resources {
				switch r {
				case "energy":
					msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.Target})
				case "mental":
					if len(p.Hand) > 0 {
						msgs = append(msgs, engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0]}})
					}
				case "physical":
					msgs = append(msgs, engine.StunEntity{Target: p.ID})
					g.Delete(t.ID)
					g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: t.Code})
				case "wild":
					msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.Target},
						engine.StunEntity{Target: p.ID})
				}
			}
			return msgs
		},
	})

	// 45158 Celestial Tech: fetch a Celestial attachment.
	engine.RegisterBehavior("45158", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for _, c := range append(engine.CardList{}, g.EncounterDeck...) {
				if c.Def().HasTrait("Celestial") && c.Def().Type == "attachment" {
					g.EncounterDeck.Remove(c.ID)
					g.ShuffleEncounterDeck()
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			for _, c := range append(engine.CardList{}, g.EncounterDiscard...) {
				if c.Def().HasTrait("Celestial") && c.Def().Type == "attachment" {
					g.EncounterDiscard.Remove(c.ID)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			return nil
		},
	})
}

func registerClanAkkaba() {
	// ancientRitual finds the Ancient Ritual side scheme.
	ancientRitual := func(g *engine.Game) *engine.SideScheme {
		for _, s := range g.SideSchemes {
			if s != nil && s.Code == "45163" {
				return s
			}
		}
		return nil
	}
	ritualThreat := func(g *engine.Game, n int) []engine.Message {
		if s := ancientRitual(g); s != nil {
			s.Threat += n
			g.Logf("Ancient Ritual gains %d threat (%d)", n, s.Threat)
			if s.Threat >= 10 {
				s.Threat -= 5
				var msgs []engine.Message
				for _, p := range g.Players {
					msgs = append(msgs, engine.DealEncounterToPlayer{Player: p.ID})
				}
				g.Logf("Ancient Ritual pays out — each player takes a facedown encounter card")
				return msgs
			}
		}
		return nil
	}

	// 45159 Ozymandias: schemes feed the ritual.
	engine.RegisterBehavior("45159", &engine.Behavior{
		MinionActivate: func(g *engine.Game, mn *engine.Minion, p *engine.Player) []engine.Message {
			// Scheme onto the ritual instead of the main scheme.
			if mn.Confused {
				mn.Confused = false
				return nil
			}
			return ritualThreat(g, mn.SchemeVal)
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return ritualThreat(g, 3)
		},
	})

	// 45160 Scarab: attacks feed the ritual.
	engine.RegisterBehavior("45160", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			return ritualThreat(g, 1)
		},
	})

	// 45161 Clan Akkaba Zealot: death feeds the ritual.
	engine.RegisterBehavior("45161", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			return ritualThreat(g, 2)
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return ritualThreat(g, 1)
		},
	})

	// 45162 Tyrant Worship: big ritual growth; boost option auto-feeds.
	engine.RegisterBehavior("45162", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return ritualThreat(g, 5)
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return ritualThreat(g, 3)
		},
	})

	// 45163 Ancient Ritual: permanent scheme; the 10-threat payoff lives
	// in ritualThreat.
	engine.RegisterBehavior("45163", &engine.Behavior{})
}
