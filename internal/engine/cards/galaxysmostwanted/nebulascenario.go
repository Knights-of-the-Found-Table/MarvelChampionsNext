package galaxysmostwanted

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// techniqueCodes are Nebula's Technique attachments (16094–16098).
var techniqueCodes = map[string]bool{
	"16094": true, "16095": true, "16096": true, "16097": true, "16098": true,
}

// registerNebulaScenario installs The Art of Evasion (16088–16102):
// Nebula's Technique attachment economy and the evasion-counter chase.
func registerNebulaScenario() {
	for _, base := range []string{"16088", "16089", "16090"} {
		stage := base
		b := &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				switch m := msg.(type) {
				case engine.RevealEncounterCard:
					// The first Technique revealed each round surges.
					def := m.Card.Def()
					if def.Type == "attachment" && def.HasTrait("technique") && !g.UsedThisRound["nebula-tech-surge"] {
						g.UsedThisRound["nebula-tech-surge"] = true
						g.Logf("The first Technique of the round gains surge")
						if c, ok := g.DrawEncounter(); ok {
							return []engine.Message{engine.RevealEncounterCard{Player: m.Player, Card: c}}
						}
					}
				case engine.VillainActivates:
					if m.VillainID != e.EID() {
						return nil
					}
					p := g.Player(m.Player)
					if p == nil {
						return nil
					}
					var msgs []engine.Message
					msgs = append(msgs, resolveTechniques(g, p)...)
					switch stage {
					case "16088":
						msgs = append(msgs, discardTechniques(g, true)...)
					case "16089":
						msgs = append(msgs, discardTechniques(g, false)...)
					case "16090":
						if len(p.Deck) > 0 {
							c := p.Deck[0]
							p.Deck = p.Deck[1:]
							g.Logf("%s removes %s from the game", p.Name, c.Def().Name)
						}
						msgs = append(msgs, discardTechniques(g, false)...)
					}
					return msgs
				}
				return nil
			},
			VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
				// Cutthroat Ambition caps single-attack damage at 5.
				if techniqueAttached(g, "16094") && damage > 5 {
					v.Damage += 5
					g.Logf("Cutthroat Ambition caps the damage at 5")
					return false
				}
				return true
			},
		}
		engine.RegisterBehavior(base, b)
	}

	// 16091 The Art of Evasion (stage 1): ship + Milano into play, Power
	// Stone on Nebula, mill 2[per_hero] and attach Techniques.
	engine.RegisterBehavior("16091", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if findEnvironment(g, "16093") == "" {
				g.SpawnEnvironment("16093")
			}
			if _, mil := findMilano(g); mil == nil {
				g.SpawnSupport("16142", cardutil.FirstPlayerID(g))
			}
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "16088" {
					g.SpawnAttachment("16149", id)
					break
				}
			}
			for i := 0; i < 2*len(g.Players) && len(g.EncounterDeck) > 0; i++ {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if techniqueCodes[c.Code[:5]] {
					for id := range g.Villains {
						if v := g.Villains[id]; v != nil && v.Code[:5] == "16088" {
							g.SpawnAttachment(c.Code, id)
							break
						}
					}
				} else {
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				}
			}
			return nil
		},
	})

	// 16092 Warp Drive Initiated (stage 2): 2 evasion counters; mills per
	// counter.
	engine.RegisterBehavior("16092", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			ship := g.EnvironmentByCode("16093")
			if ship == nil {
				return nil
			}
			ship.Counters += 2
			g.Logf("Nebula's Ship gains 2 evasion counters (%d total)", ship.Counters)
			var msgs []engine.Message
			for i := 0; i < ship.Counters; i++ {
				for _, p := range g.Players {
					msgs = append(msgs, engine.MillPlayerDeck{Player: p.ID, N: 2})
				}
				msgs = append(msgs, engine.MillEncounter{N: 2})
			}
			return msgs
		},
	})

	// 16093 Nebula's Ship: evasion counter at villain phase start;
	// Milano-assisted counter removal.
	engine.RegisterBehavior("16093", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			bp, ok := msg.(engine.BeginPhase)
			if !ok || bp.Phase != engine.PhaseVillain {
				return nil
			}
			env := g.Environments[e.EID()]
			if env == nil {
				return nil
			}
			env.Counters++
			g.Logf("Nebula's Ship gains an evasion counter (%d total)", env.Counters)
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			env := g.Environments[e.EID()]
			if env == nil || env.Counters <= 0 {
				return nil
			}
			milID, mil := findMilano(g)
			if mil == nil || mil.Exhausted {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust the Milano + spend 2 resources → remove 2 evasion counters", Type: engine.AbilityAction,
				Cost: 2,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.ExhaustEntity{ID: milID},
						engine.AddEntityCounter{ID: self, N: -2},
					}
				},
			}}
		},
	})

	registerTechniques()

	// 16099 Lethal Intent: dig a Technique and reveal it.
	engine.RegisterBehavior("16099", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for len(g.EncounterDeck) > 0 {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if techniqueCodes[c.Code[:5]] {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})

	// 16100 Barrel Roll: incite 1 + evasion counter (Surge keyword is
	// data-driven).
	engine.RegisterBehavior("16100", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID})
			}
			if ship := g.EnvironmentByCode("16093"); ship != nil {
				ship.Counters++
				g.Logf("Barrel Roll: evasion counter placed (%d total)", ship.Counters)
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if ship := g.EnvironmentByCode("16093"); ship != nil {
				ship.Counters++
				g.Logf("Barrel Roll boost: evasion counter placed (%d total)", ship.Counters)
			}
			return nil
		},
	})

	// 16101 Combat Ready: dig a Technique, resolve its Special, discard.
	engine.RegisterBehavior("16101", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for len(g.EncounterDeck) > 0 {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if techniqueCodes[c.Code[:5]] {
					att := g.SpawnAttachment(c.Code, "")
					msgs := techniqueSpecial(g, p, att)
					msgs = append(msgs, engine.DiscardAttachmentMsg{ID: att.ID})
					return msgs
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})

	// 16102 Ruthless: Nebula schemes (alter-ego) or attacks (hero); an
	// evasion counter rides along either way.
	engine.RegisterBehavior("16102", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				if ship := g.EnvironmentByCode("16093"); ship != nil {
					ship.Counters++
				}
				return []engine.Message{engine.VillainActivates{VillainID: id, Player: p.ID}}
			}
			return nil
		},
	})
}

// registerTechniques installs the five Technique attachments (16094–
// 16098): attach preference, Special abilities and boost riders. The
// Special abilities themselves live in techniqueSpecial.
func registerTechniques() {
	for _, code := range []string{"16094", "16095", "16096", "16097", "16098"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				for id := range g.Villains {
					if v := g.Villains[id]; v != nil && v.Code[:5] == "16088" {
						t.Target = id
						g.Logf("%s attaches to Nebula", t.EDef().Name)
						return nil
					}
				}
				for id := range g.Villains {
					t.Target = id
					break
				}
				return nil
			},
			Boost: func(g *engine.Game, card engine.Card) []engine.Message {
				// Attach to Nebula and resolve the Special against the
				// first player.
				for id := range g.Villains {
					if v := g.Villains[id]; v != nil && v.Code[:5] == "16088" {
						att := g.SpawnAttachment(card.Code, id)
						for _, p := range g.Players {
							if p.FirstPlayer {
								return techniqueSpecial(g, p, att)
							}
						}
					}
				}
				return nil
			},
		})
	}
}

// techniqueSpecial runs one attachment's Special against the player.
func techniqueSpecial(g *engine.Game, p *engine.Player, a *engine.Attachment) []engine.Message {
	switch a.Code[:5] {
	case "16094":
		if g.MainScheme != nil {
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: a.ID}}
		}
	case "16095":
		var msgs []engine.Message
		if p.Stunned {
			for id := range g.Villains {
				msgs = append(msgs, engine.DealBoost{Enemy: id})
				break
			}
		}
		return append(msgs, engine.StunEntity{Target: p.ID})
	case "16096":
		for id := range g.Villains {
			if v := g.Villains[id]; v != nil && v.Tough {
				return []engine.Message{engine.ToughEntity{Target: id}, engine.DealBoost{Enemy: id}}
			}
			return []engine.Message{engine.ToughEntity{Target: id}}
		}
	case "16097":
		return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: a.ID}}
	case "16098":
		if len(p.Hand) > 0 {
			c := p.Hand[0]
			return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}}
		}
	}
	return nil
}

// resolveTechniques runs every attached Technique's Special against the
// player Nebula activates against.
func resolveTechniques(g *engine.Game, p *engine.Player) []engine.Message {
	var msgs []engine.Message
	for _, id := range cardutil.SortedIDs(g.Attachments) {
		a := g.Attachments[id]
		if a == nil || !techniqueCodes[a.Code[:5]] {
			continue
		}
		msgs = append(msgs, techniqueSpecial(g, p, a)...)
	}
	return msgs
}

// discardTechniques removes Technique attachments: all of them (stage I)
// or just the first (stages II/III approximation).
func discardTechniques(g *engine.Game, all bool) []engine.Message {
	var msgs []engine.Message
	dropped := 0
	for _, id := range cardutil.SortedIDs(g.Attachments) {
		a := g.Attachments[id]
		if a == nil || !techniqueCodes[a.Code[:5]] {
			continue
		}
		if !all && dropped >= 1 {
			break
		}
		msgs = append(msgs, engine.DiscardAttachmentMsg{ID: id})
		dropped++
	}
	return msgs
}

// techniqueAttached reports whether the given Technique is in play.
func techniqueAttached(g *engine.Game, base string) bool {
	for _, a := range g.Attachments {
		if a != nil && a.Code[:5] == base {
			return true
		}
	}
	return false
}
