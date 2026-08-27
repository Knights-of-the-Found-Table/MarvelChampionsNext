// Package riseofredskull registers The Rise of Red Skull campaign box:
// Crossbones, Absorbing Man, Taskmaster, Zola and Red Skull scenarios
// plus the campaign aspect cards and modular sets.
package riseofredskull

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerTRORSAspect()
	registerCrossbones()
	registerAbsorbingMan()
	registerTaskmaster()
	registerZola()
	registerRedSkull()
	registerHydraModulars()
	registerCampaignExtras()
	registerTRORSScenarios()
}

// registerTRORSAspect installs the campaign aspect cards (04012–04025,
// 04041–04052).
func registerTRORSAspect() {
	// 04012 Black Knight / 04014 U.S. Agent: keyword allies.
	engine.RegisterBehavior("04012", &engine.Behavior{})
	engine.RegisterBehavior("04014", &engine.Behavior{})

	// 04013 Goliath: +4 ATK burst, then gone.
	engine.RegisterBehavior("04013", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.goliathGets4AtkThisPhaseThenDiscardsHimself"), Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.AllyStatBonus{Ally: self, ATK: 4},
						engine.AllyDestroyed{AllyID: self},
					}
				},
			}}
		},
	})

	// 04015 Sky Cycle: ready the attached Avenger ally.
	engine.RegisterBehavior("04015", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if a := g.Allies[target]; a != nil {
				a.ExtraTraits = append(a.ExtraTraits, "aerial")
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			u := g.Upgrades[e.EID()]
			if u == nil || u.AttachTo == "" {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustSkyCycleReadyTheAttachedAlly"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if u := g.Upgrades[self]; u != nil {
						return []engine.Message{engine.ReadyEntity{ID: u.AttachTo}}
					}
					return nil
				},
			}}
		},
	})

	// 04016 Team Training: allies get +1 HP (applied as each ally
	// enters play; existing allies on play of the support).
	engine.RegisterBehavior("04016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					a.MaxHP++
				}
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyEntersPlayFree)
			_ = m
			if !ok {
				return nil
			}
			return nil
		},
	})

	// 04017 Ready for Action: tough an ally.
	engine.RegisterBehavior("04017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					picks = append(picks, engine.Choice{
						Label: engine.Tf("m.cardName", a), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.ToughEntity{Target: id}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.readyForActionToughWhichAlly"), picks...)}}
		},
	})

	// 04018 Lead from the Front: team-wide +1 THW/+1 ATK.
	engine.RegisterBehavior("04018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				p.BonusTHW++
				p.BonusATK++
				for _, id := range p.Allies {
					msgs = append(msgs, engine.AllyStatBonus{Ally: id, ATK: 1, THW: 1})
				}
			}
			g.TLogf("c.leadFromTheFrontTheTeamGets1ThwAnd1AtkThisPhase")
			return msgs
		},
	})

	// 04019 Power of Leadership: data-driven doubling.
	engine.RegisterBehavior("04019", &engine.Behavior{})

	// 04020 War Machine: keyword ally.
	engine.RegisterBehavior("04020", &engine.Behavior{})

	// 04021 Avengers Tower: discount ability.
	engine.RegisterBehavior("04021", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustAvengersTowerNextAvengerAllyThisPhaseCosts1Less"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.CostDiscountApply{Player: e.EOwner(), Amount: 1}}
				},
			}}
		},
	})

	// 04022 Earth's Mightiest Heroes: ready an Avenger.
	engine.RegisterBehavior("04022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.Exhausted && g.EntityHasTrait(id, "avenger") {
					picks = append(picks, engine.Choice{
						Label: engine.Tf("m.cardName", a), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.ReadyEntity{ID: id}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.earthSMightiestHeroesReadyWhichAvenger"), picks...)}}
		},
	})

	// 04023–04025 / 04050–04052 basic resources.
	for _, code := range []string{"04023", "04024", "04025", "04050", "04051", "04052"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 04041 Combat Training / 04046 Heroic Intuition.
	engine.RegisterBehavior("04041", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{ATK: 1} },
	})
	engine.RegisterBehavior("04046", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1} },
	})

	// 04042 Tac Team: alias core 01056-style (counter-powered pings).
	engine.RegisterBehavior("04042", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustTacTeam1CounterDeal2Damage"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					if s := g.Supports[self]; s != nil {
						s.Counters--
					}
					return cardutil.ChooseEnemy(engine.Tf("c.tacTeamDeal2Damage"),
						func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 2, nil })(g, g.Entity(self))
				},
			}}
		},
	})

	// 04043 Press the Advantage: 2 damage, draw if target is statused.
	engine.RegisterBehavior("04043", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.pressTheAdvantageDeal2Damage"),
			func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) {
				var extra []engine.Message
				switch t := tgt.(type) {
				case *engine.Villain:
					if t.Stunned || t.Confused {
						extra = append(extra, engine.DrawCards{Player: g.ActiveTurn, N: 1})
					}
				case *engine.Minion:
					if t.Stunned || t.Confused {
						extra = append(extra, engine.DrawCards{Player: g.ActiveTurn, N: 1})
					}
				}
				return 2, extra
			}),
	})

	// 04044 Piercing Strike: 3 damage.
	engine.RegisterBehavior("04044", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.piercingStrikeDeal3Damage"),
			func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 3, nil }),
	})

	// 04045 Spider-Man: 3[per_hero] threat off a side scheme.
	engine.RegisterBehavior("04045", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, id := range sortedSchemeIDs(g) {
				if s := g.SideSchemes[id]; s != nil && s.Threat > 0 {
					picks = append(picks, engine.Choice{
						Label: engine.Tf("m.cardName", s), Kind: engine.ChoiceTarget, SourceID: id,
					}.Msgs(engine.ThwartScheme{Scheme: id, N: 3 * len(g.Players), Source: e.EID()}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask(engine.Tf("c.spiderManRemoveThreatFromWhichSideScheme"), picks...)}}
		},
	})

	// 04047 Skilled Investigator: draw on scheme defeat.
	engine.RegisterBehavior("04047", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.SchemeDefeated); !ok {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.Exhausted {
				return nil
			}
			u.Exhausted = true
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
		},
	})

	// 04048 Interrogation Room: minion kills remove threat.
	engine.RegisterBehavior("04048", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionDefeated); !ok {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil || s.Exhausted || g.MainScheme == nil {
				return nil
			}
			s.Exhausted = true
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()}}
		},
	})

	// 04049 Clear the Area: 2 threat, draw on the kill.
	engine.RegisterBehavior("04049", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := cardutil.ChooseScheme(engine.Tf("c.clearTheAreaChooseAScheme"), func(g *engine.Game, s engine.Entity) int { return 2 })(g, e)
			return append(msgs, engine.DrawCards{Player: e.EOwner(), N: 1})
		},
	})
}

// registerCrossbones installs the Crossbones scenario (04058–04075). The
// Experimental Weapons deck is approximated by the set-aside pool.
func registerCrossbones() {
	for _, base := range []string{"04058", "04059", "04060"} {
		b := &engine.Behavior{}
		if base == "04059" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				for i, c := range g.EncounterDeck {
					if c.Code[:5] == "04064" {
						g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
						g.ShuffleEncounterDeck()
						return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
					}
				}
				return nil
			}
		}
		if base == "04060" {
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				return revealExperimentalWeapon(g)
			}
		}
		engine.RegisterBehavior(base, b)
	}

	// 04061–04063 main schemes.
	engine.RegisterBehavior("04061", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			for _, code := range []string{"04072", "04073", "04074", "04075"} {
				g.SetAside = append(g.SetAside, engine.Card{ID: g.NextCardID(), Code: code})
			}
			return nil
		},
	})
	engine.RegisterBehavior("04062", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			return revealExperimentalWeapon(g)
		},
	})
	engine.RegisterBehavior("04063", &engine.Behavior{
		MainSchemeRevealed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			return revealExperimentalWeapon(g)
		},
	})

	// 04064 Machine Gun: ammo-powered strafing.
	engine.RegisterBehavior("04064", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Counters = 2 * len(g.Players)
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "04058" {
					t.Target = id
					return nil
				}
			}
			for id := range g.Villains {
				t.Target = id
				break
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.VillainActivates)
			if !ok {
				return nil
			}
			a := g.Attachments[e.EID()]
			p := g.Player(m.Player)
			if a == nil || a.Counters <= 0 || p == nil || !p.IsHero() {
				return nil
			}
			a.Counters--
			stars := 0
			if len(g.EncounterDeck) > 0 {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if b := c.Def().Boost; b != nil {
					stars = *b
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			if stars > 0 {
				return []engine.Message{engine.IndirectDamage{Player: p.ID, N: stars}}
			}
			return nil
		},
	})

	// 04065 Crossbones' Armor: soaks up to 5.
	engine.RegisterBehavior("04065", &engine.Behavior{})

	// 04066 Hydra Bomber.
	engine.RegisterBehavior("04066", &engine.Behavior{})

	// 04067 Full Auto.
	engine.RegisterBehavior("04067", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if !p.IsHero() {
				if c, ok := g.DrawEncounter(); ok {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				return nil
			}
			x := 0
			for _, v := range g.Villains {
				if v != nil && v.Code[:5] == "04058" {
					x = v.AttackVal
				}
			}
			stars := 0
			for i := 0; i < x && len(g.EncounterDeck) > 0; i++ {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if b := c.Def().Boost; b != nil {
					stars += *b
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			if stars > 0 {
				return []engine.Message{engine.IndirectDamage{Player: p.ID, N: stars}}
			}
			return nil
		},
	})

	// 04068 Hard as Nails.
	engine.RegisterBehavior("04068", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for _, v := range g.Villains {
				if v == nil {
					continue
				}
				if v.Tough {
					v.Damage = max(0, v.Damage-3)
				} else {
					v.Tough = true
				}
				return nil
			}
			return nil
		},
	})

	// 04069 Raid the Armory.
	engine.RegisterBehavior("04069", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if g.MainScheme != nil {
				g.Push(engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID})
			}
			for len(g.EncounterDeck) > 0 {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if c.Def().Type == "attachment" && c.Def().HasTrait("weapon") {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})

	// 04070/04071 side schemes.
	engine.RegisterBehavior("04070", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for id := range g.Villains {
				return []engine.Message{engine.VillainActivates{VillainID: id, Player: g.ActiveTurn}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("04071", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			stars := 0
			for i := 0; i < len(g.Players) && len(g.EncounterDeck) > 0; i++ {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if b := c.Def().Boost; b != nil {
					stars += *b
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			if s := g.SideSchemes[e.EID()]; s != nil {
				s.Threat += stars
			}
			return nil
		},
	})

	// 04072–04075 experimental weapons.
	for _, spec := range []struct {
		code, icons string
	}{
		{"04072", "energy:1 physical:1"},
		{"04073", "energy:1 mental:1"},
		{"04074", "mental:1 physical:1"},
		{"04075", "energy:1 mental:1 physical:1"},
	} {
		engine.RegisterBehavior(spec.code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				for id := range g.Villains {
					t.Target = id
					break
				}
				return nil
			},
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				return []engine.Ability{{
					Label: engine.Tf("c.spendResourcesDiscardThisWeapon"), Type: engine.AbilityAction,
					CostIcons: spec.icons,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
					},
				}}
			},
		})
	}
}

// revealExperimentalWeapon pulls a weapon from the pool onto the villain.
func revealExperimentalWeapon(g *engine.Game) []engine.Message {
	if len(g.SetAside) == 0 {
		return nil
	}
	idx := g.Random(len(g.SetAside))
	pick := g.SetAside[idx]
	g.SetAside = append(g.SetAside[:idx], g.SetAside[idx+1:]...)
	for id := range g.Villains {
		g.SpawnAttachment(pick.Code, id)
		break
	}
	g.TLogMajorf("c.experimentalWeaponAttachesToTheVillain", pick)
	return nil
}

// sortedSchemeIDs lists side scheme ids in stable order.
func sortedSchemeIDs(g *engine.Game) []engine.EntityID {
	var out []engine.EntityID
	for id := range g.SideSchemes {
		out = append(out, id)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
