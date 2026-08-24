package trickster

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// registerTricksterMagic implements the Wrecking Crew-style modular: each
// Elite minion flips into its linked Defiant ally on defeat.
func registerTricksterMagic() {
	flips := map[string]string{
		"55056": "55063", // Absorbing Man
		"55057": "55064", // Titania
		"55058": "55065", // Whirlwind
		"55059": "55066", // Zzzax
	}
	for minion, ally := range flips {
		code := minion
		allyCode := ally
		engine.RegisterBehavior(code, &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				m, ok := msg.(engine.MinionDefeated)
				if !ok || m.MinionID != e.EID() {
					return nil
				}
				mn := g.Minions[e.EID()]
				if mn == nil {
					return nil
				}
				p := g.Player(mn.EngagedWith)
				if p == nil {
					p = g.Players[0]
				}
				if p == nil {
					return nil
				}
				card := engine.Card{Code: allyCode}
				def := card.Def()
				a := &engine.Ally{
					ID: g.NextEntityID(engine.KindAlly), Code: allyCode,
					Owner: p.ID, MaxHP: ttInt(def.HP, 3),
					AttackVal: ttInt(def.Attack, 1), ThwartVal: ttInt(def.Thwart, 1),
				}
				g.AddAlly(a, p.ID)
				g.Logf("%s breaks free of the enchantment and joins %s!", def.Name, p.Name)
				return []engine.Message{engine.AllyEnteredPlay{Ally: a.ID, Player: p.ID}}
			},
		})
	}

	// 55060 The Trickster Tango / 55061 Puppet Master.
	engine.RegisterBehavior("55060", &engine.Behavior{})
	engine.RegisterBehavior("55061", &engine.Behavior{})
	// 55062 Love Triangle.
	engine.RegisterBehavior("55062", &engine.Behavior{})
	// 55063-55066 Defiant allies.
	engine.RegisterBehavior("55063", &engine.Behavior{})
	engine.RegisterBehavior("55064", &engine.Behavior{})
	engine.RegisterBehavior("55065", &engine.Behavior{})
	engine.RegisterBehavior("55066", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := 5
			if len(p.Deck) < n {
				n = len(p.Deck)
			}
			for i := 0; i < n; i++ {
				if len(p.Deck[i].Def().Resources) > 0 && p.Deck[i].Def().Resources[0] == "energy" {
					card := p.Deck[i]
					p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
					p.Hand = append(p.Hand, card)
					return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
				}
			}
			return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
		},
	})
}

func registerScenarios() {
	// Prime Real Estate — Enchantress across two scheme stages.
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "55004b",
		Name:             "Enchantress — Prime Real Estate",
		VillainBases:     []string{"55001"},
		MainSchemeStages: []string{"55004b", "55005b"},
		ExtraSets:        []string{"trickster_magic", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			for _, v := range g.Villains {
				if g.Difficulty == "expert" {
					v.SetVillainStages([]string{"55002", "55003"})
					v.Code = "55002"
					def := engine.Card{Code: "55002"}.Def()
					v.MaxHP = ttInt(def.HP, v.MaxHP)
					v.AttackVal = ttInt(def.Attack, v.AttackVal)
					v.SchemeVal = ttInt(def.Scheme, v.SchemeVal)
				} else {
					v.SetVillainStages([]string{"55001", "55002"})
				}
			}
			// Future of Despair waits aside; each identity wears a random
			// Hypnotic Gaze (the double-sided gazes never enter the
			// encounter deck, so they are attached directly).
			var kept engine.CardList
			for _, c := range g.EncounterDeck {
				if c.Code == "55006" {
					g.SetAside = append(g.SetAside, c)
					continue
				}
				kept = append(kept, c)
			}
			g.EncounterDeck = kept
			gazeCodes := []string{"55007a", "55008a", "55009a", "55010a", "55011a"}
			for i := range gazeCodes {
				j := i + g.Random(len(gazeCodes)-i)
				gazeCodes[i], gazeCodes[j] = gazeCodes[j], gazeCodes[i]
			}
			for i, p := range g.Players {
				if i < len(gazeCodes) {
					g.SpawnAttachment(gazeCodes[i], p.ID)
					g.Logf("%s is caught by a Hypnotic Gaze", p.Name)
				}
			}
			return nil
		},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage < len(s.StageCodes) {
				return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: engine.Tf("reason.enchantressEstate")}}
		},
	})

	// Worlds Collide — the Avatars of Loki (pod play collapsed to one
	// table: a random avatar starts, the rest wait set-aside).
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "55028b",
		Name:             "Loki, God of Lies — Worlds Collide",
		VillainBases:     []string{"55029a"},
		MainSchemeStages: []string{"55033b"},
		ExtraSets:        []string{"trickster_magic", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// One random avatar starts in play; the rest are set aside.
			for id := range g.Villains {
				g.Delete(id)
			}
			kept := g.Random(len(avatarCodes))
			for i, code := range avatarCodes {
				if i == kept {
					if v := g.SpawnVillainFromCard(data.BaseCode(code)); v != nil {
						g.Logf("%s steps out of the shadows", v.EDef().Name)
					}
				} else {
					g.SetAside = append(g.SetAside, engine.Card{ID: g.NextCardID(), Code: code})
				}
			}
			// The Synergy environments join the board.
			g.SpawnEnvironment("55052")
			g.SpawnEnvironment("55053")
			g.SpawnEnvironment("55054")
			g.SpawnEnvironment("55055")
			return nil
		},
		OnVillainDefeated: func(g *engine.Game, v *engine.Villain) []engine.Message {
			// A shattered avatar or the revealed god falling ends it.
			return []engine.Message{engine.GameOver{Won: true, Reason: engine.Tf("reason.lokiIllusions")}}
		},
	})
}
