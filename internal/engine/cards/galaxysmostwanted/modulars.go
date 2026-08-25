package galaxysmostwanted

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerModularSets installs the modular encounter sets (plus the
// textless Vendetta side scheme shell, 16054).: Band of Badoon
// (16117–16121), Galactic Artifacts (16122–16130), Kree Militant
// (16131–16134), Menagerie Medley (16135–16137), Space Pirates (16138–
// 16141), Ship Command (16142–16148) and the Power Stone (16149).
func registerModularSets() {
	// 16054 Vendetta: textless side scheme shell.
	engine.RegisterBehavior("16054", &engine.Behavior{})
	registerBandOfBadoon()
	registerGalacticArtifacts()
	registerKreeMilitant()
	registerMenagerie()
	registerSpacePirates()
	registerShipCommand()
	registerPowerStone()
}

func registerBandOfBadoon() {
	// 16117 Badoon Assassin: attacks the hero it engages with +2 ATK.
	engine.RegisterBehavior("16117", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			em, ok := msg.(engine.EngageMinion)
			if !ok || em.MinionID != e.EID() {
				return nil
			}
			p := g.Player(em.Player)
			if p == nil || !p.IsHero() {
				return nil
			}
			m := g.Minions[e.EID()]
			if m == nil {
				return nil
			}
			g.TLogf("c.badoonAssassinStrikesWith2Atk", p.Name)
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: m.AttackVal + 2, Source: e.EID()}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.BoostActivation{N: 0}}
		},
	})

	// 16118 Badoon Grunt: when it engages with no other minions around,
	// take a facedown encounter card; boost spawns itself (engine-side).
	engine.RegisterBehavior("16118", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			em, ok := msg.(engine.EngageMinion)
			if !ok || em.MinionID != e.EID() {
				return nil
			}
			others := 0
			for _, m := range g.Minions {
				if m != nil && m.ID != e.EID() && m.EngagedWith == em.Player {
					others++
				}
			}
			if others == 0 {
				return []engine.Message{engine.DealEncounterToPlayer{Player: em.Player}}
			}
			return nil
		},
	})

	// 16119 Badoon Lieutenant: Patrol (data keyword; the thwart block is
	// engine-side via thwartBlocked).
	engine.RegisterBehavior("16119", &engine.Behavior{})

	// 16120 Badoon Sentry: Retaliate 1 (data keyword); boost gives the
	// villain tough (or +2 boost icons).
	engine.RegisterBehavior("16120", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil {
					if v.Tough {
						return []engine.Message{engine.BoostActivation{Enemy: id, N: 2}}
					}
					v.Tough = true
					g.TLogf("c.gainsAToughStatusCard", v)
					return nil
				}
			}
			return nil
		},
	})

	// 16121 Badoon Warlord: attacks gain overkill (approximation: +2
	// damage instead of spill); boost +2 on attacks.
	engine.RegisterBehavior("16121", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.BoostActivation{N: 2}}
		},
	})
}

func registerGalacticArtifacts() {
	// 16122 Cloak of Hercules: -1 ATK on the weakest enemy.
	engine.RegisterBehavior("16122", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Target = lowestATKEnemy(g, true)
			g.TLogf("c.cloakOfHerculesAttachesTo", g.Entity(t.Target).EDef().Name)
			return nil
		},
		Abilities: artifactBuyoff("16122", "Spend [physical][physical][physical] → discard Cloak of Hercules", "physical:3"),
	})

	// 16123 Obedience Potion: -1 THW/-1 ATK/-1 DEF on your identity.
	engine.RegisterBehavior("16123", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			// Revealed against a player: attach to them.
			if p := g.Player(target); p != nil {
				t.Target = p.ID
			} else {
				t.Target = cardutil.FirstPlayerID(g)
			}
			p := g.Player(t.Target)
			if p != nil {
				p.BonusTHW--
				p.BonusATK--
				p.BonusDEF--
				g.TLogf("c.obediencePotionWeakens", p.Name)
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.take1DamageSpendMentalMentalDiscardObediencePotion"), Type: engine.AbilityAction,
				CostIcons: "mental:2",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Attachments[self]
					owner := cardutil.FirstPlayerID(g)
					if a != nil && a.Target.Is(engine.KindPlayer) {
						owner = a.Target
					}
					return []engine.Message{
						engine.DamageEntity{Target: owner, Damage: 1, Source: self},
						engine.DiscardAttachmentMsg{ID: self},
					}
				},
			}}
		},
	})

	// 16124 The Beyonder's Blazer: 2 main-scheme threat to buy off.
	engine.RegisterBehavior("16124", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Target = highestSCHEnemy(g)
			g.TLogf("c.theBeyonderSBlazerAttachesTo", g.Entity(t.Target).EDef().Name)
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.place2ThreatOnTheMainSchemeSpend2ResourcesDiscardTheBeyonder"), Type: engine.AbilityAction,
				Cost: 2,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					msgs := []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
					if g.MainScheme != nil {
						msgs = append([]engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: self}}, msgs...)
					}
					return msgs
				},
			}}
		},
	})

	// 16125 The Poison: 1 damage per poison counter each turn.
	engine.RegisterBehavior("16125", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if p := g.Player(target); p != nil {
				t.Target = p.ID
			} else {
				t.Target = cardutil.FirstPlayerID(g)
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ts, ok := msg.(engine.PlayerTurnStart)
			if !ok {
				return nil
			}
			a := g.Attachments[e.EID()]
			if a == nil || a.Target != ts.Player {
				return nil
			}
			a.Counters++
			g.TLogf("c.thePoisonSCounters", a.Counters)
			return []engine.Message{engine.DamageEntity{Target: ts.Player, Damage: a.Counters, Source: e.EID()}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.spend3ResourcesOfDifferentTypesDiscardThePoison"), Type: engine.AbilityAction,
				Cost: 3,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
				},
			}}
		},
	})

	// 16126 Vandarian Power Stone: sits on the weakest schemer.
	engine.RegisterBehavior("16126", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Target = lowestSCHEnemy(g)
			g.TLogf("c.vandarianPowerStoneAttachesTo", g.Entity(t.Target).EDef().Name)
			return nil
		},
		Abilities: artifactBuyoff("16126", "Spend [energy][energy][energy] → discard Vandarian Power Stone", "energy:3"),
	})

	// 16127–16130 trophy side schemes: victory rewards on defeat.
	engine.RegisterBehavior("16127", &engine.Behavior{
		SideSchemeDefeated: readyIdentityReward,
	})
	engine.RegisterBehavior("16128", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			return []engine.Message{engine.HealEntity{Target: g.ActiveTurn, N: 4}}
		},
	})
	engine.RegisterBehavior("16129", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			return []engine.Message{engine.DrawCards{Player: g.ActiveTurn, N: 2}}
		},
	})
	engine.RegisterBehavior("16130", &engine.Behavior{
		// Play a card from hand at -3 cost: approximated as a 3-cost
		// discount on the next card.
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			return []engine.Message{engine.CostDiscountApply{Player: g.ActiveTurn, Amount: 3}}
		},
	})
}

func registerKreeMilitant() {
	// 16131 Kree Combat Armor: -1 damage per attack on the strongest.
	engine.RegisterBehavior("16131", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			t.Target = highestATKEnemy(g, false)
			g.TLogf("c.kreeCombatArmorAttachesTo", g.Entity(t.Target).EDef().Name)
			return nil
		},
		Abilities: artifactBuyoff("16131", "Spend 3 resources of the same type → discard Kree Combat Armor", ""),
	})

	// 16132 Kree Commando: Patrol; boost piercing on attacks.
	engine.RegisterBehavior("16132", &engine.Behavior{})

	// 16133 Kree Lieutenant: Guard + Stalwart; boost +3 on attacks.
	engine.RegisterBehavior("16133", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.BoostActivation{N: 3}}
		},
	})

	// 16134 Kree Private: Quickstrike; boost overkill.
	engine.RegisterBehavior("16134", &engine.Behavior{})
}

func registerMenagerie() {
	// 16135 Psionic Ghost: confuse rider; boost spawns itself
	// (engine-side).
	engine.RegisterBehavior("16135", &engine.Behavior{})

	// 16136 Servant Bot: Guard + Patrol.
	engine.RegisterBehavior("16136", &engine.Behavior{})

	// 16137 Starshark: Quickstrike; indirect damage attacks; boost hits
	// your characters.
	engine.RegisterBehavior("16137", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1})
			}
			return msgs
		},
	})
}

func registerSpacePirates() {
	// 16138 Pirate Commander: Quickstrike; steals a random card; boost
	// feeds the villain a boost card.
	engine.RegisterBehavior("16138", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || w.Enemy != e.EID() {
				return nil
			}
			p := g.Player(w.Player)
			if p == nil || len(p.Hand) == 0 {
				return nil
			}
			c := p.Hand[0]
			p.Hand.Remove(c.ID)
			g.TLogf("c.pirateCommanderRemovesFromTheGame", c)
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for id := range g.Villains {
				return []engine.Message{engine.DealBoost{Enemy: id}}
			}
			return nil
		},
	})

	// 16139 Pirate Lackey: Quickstrike; mills your deck into oblivion.
	engine.RegisterBehavior("16139", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || w.Enemy != e.EID() {
				return nil
			}
			p := g.Player(w.Player)
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			c := p.Deck[0]
			p.Deck = p.Deck[1:]
			g.TLogf("c.pirateLackeyRemovesFromTheGame", c)
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for id := range g.Villains {
				return []engine.Message{engine.DealBoost{Enemy: id}}
			}
			return nil
		},
	})

	// 16140 Sound the Alarms: +1 ATK to each enemy; boost reveals itself
	// (engine handles "Reveal this card" boosts).
	engine.RegisterBehavior("16140", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, id := range cardutil.SortedEnemyIDs(g) {
				switch t := g.Entity(id).(type) {
				case *engine.Villain:
					t.AttackVal++
				case *engine.Minion:
					t.AttackVal++
				}
			}
			g.TLogf("c.eachEnemyGets1AtkWhileSoundTheAlarmsIsInPlay")
			return nil
		},
	})

	// 16141 Honor Among Thieves: dig a Criminal minion, tough + villain
	// boost card.
	engine.RegisterBehavior("16141", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for len(g.EncounterDeck) > 0 {
				c := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if c.Def().Type == "minion" && c.Def().HasTrait("criminal") {
					var msgs []engine.Message
					msgs = append(msgs, engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c})
					for id := range g.Villains {
						msgs = append(msgs, engine.DealBoost{Enemy: id})
					}
					return msgs
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})
}

func registerShipCommand() {
	// 16142 Milano: Piloting — exhaust for a wild resource.
	engine.RegisterBehavior("16142", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})

	// 16143 Rogue Vessel: end-of-villain-phase chip damage; Milano
	// removal.
	engine.RegisterBehavior("16143", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ep, ok := msg.(engine.EndPhase)
			if !ok || ep.Phase != engine.PhaseVillain {
				return nil
			}
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: e.EID()})
			}
			return msgs
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			milID, mil := findMilano(g)
			if mil == nil || mil.Exhausted {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustTheMilanoSpend2ResourcesDiscardRogueVessel"), Type: engine.AbilityAction,
				Cost: 2,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					g.Delete(self)
					g.TLogf("c.rogueVesselIsDiscarded")
					return []engine.Message{engine.ExhaustEntity{ID: milID}}
				},
			}}
		},
	})

	// 16144 Cannonade: Milano removal.
	engine.RegisterBehavior("16144", milanoSchemeBehavior(3))

	// 16145–16147 peril taxes: exhaust the Milano, pay resources, or the
	// first player suffers.
	peril := func(label string, penalty func(g *engine.Game, p *engine.Player) []engine.Message) *engine.Behavior {
		return &engine.Behavior{
			ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
				g.Delete(t.ID)
				var opts []engine.Choice
				if milID, mil := findMilano(g); mil != nil && !mil.Exhausted {
					opts = append(opts, engine.Choice{
						ID: "milano", Label: engine.Tf("c.exhaustTheMilano"), Kind: engine.ChoiceLabel,
					}.Msgs(engine.ExhaustEntity{ID: milID}))
				}
				if len(p.Hand) > 0 {
					opts = append(opts, engine.Choice{
						ID: "pay", Label: engine.S(label), Kind: engine.ChoiceLabel,
					}.Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0]}}))
				}
				opts = append(opts, engine.Choice{
					ID: "penalty", Label: engine.Tf("c.sufferThePenalty"), Kind: engine.ChoiceLabel,
				}.Msgs(penalty(g, p)...))
				return []engine.Message{engine.AskQuestion{Player: p.ID,
					Question: engine.Ask(engine.Tf("c.perilChooseOneTheFirstPlayerSuffersOtherwise"), opts...)}}
			},
		}
	}
	engine.RegisterBehavior("16145", peril(
		"Discard a card (approximates [physical][physical])",
		func(g *engine.Game, p *engine.Player) []engine.Message {
			return []engine.Message{engine.StunEntity{Target: cardutil.FirstPlayerID(g)}}
		}))
	engine.RegisterBehavior("16146", peril(
		"Discard a card (approximates [mental][mental])",
		func(g *engine.Game, p *engine.Player) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: cardutil.FirstPlayerID(g), Damage: 3}}
		}))
	engine.RegisterBehavior("16147", peril(
		"Discard a card (approximates [energy][energy])",
		func(g *engine.Game, p *engine.Player) []engine.Message {
			fp := g.Player(cardutil.FirstPlayerID(g))
			if fp != nil && len(fp.Hand) > 0 {
				return []engine.Message{engine.DiscardCards{Player: fp.ID, Cards: engine.CardList{fp.Hand[0]}}}
			}
			return nil
		}))

	// 16148 Special Delivery: exhaust the Milano or boost the villain's
	// move.
	engine.RegisterBehavior("16148", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			milID, mil := findMilano(g)
			if mil != nil && !mil.Exhausted {
				return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
					engine.Tf("c.specialDeliveryExhaustTheMilano"),
					engine.Choice{
						ID: "yes", Label: engine.Tf("c.exhaustTheMilano"), Kind: engine.ChoiceLabel,
					}.Msgs(engine.ExhaustEntity{ID: milID}),
					engine.Choice{
						ID: "no", Label: engine.Tf("c.theVillainSchemesAttacksWith1"), Kind: engine.ChoiceLabel,
					}.Msgs(engine.BoostActivation{N: 1}),
				)}}
			}
			for id := range g.Villains {
				return []engine.Message{
					engine.BoostActivation{Enemy: id, N: 1},
					engine.VillainActivates{VillainID: id, Player: p.ID},
				}
			}
			return nil
		},
	})
}

func registerPowerStone() {
	// 16149 Power Stone: hops to whoever deals 3+ damage in one attack.
	engine.RegisterBehavior("16149", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			// Setup: attach to the villain by default.
			if target == "" {
				for id := range g.Villains {
					t.Target = id
					break
				}
			}
			g.TLogf("c.thePowerStoneHumsWithEnergy")
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Damage < 3 {
				return nil
			}
			a := g.Attachments[e.EID()]
			if a == nil || a.Locked || d.Target != a.Target {
				return nil
			}
			if !(d.Source.Is(engine.KindPlayer) || d.Source.Is(engine.KindVillain) || d.Source.Is(engine.KindMinion)) {
				return nil
			}
			if d.Source.Is(engine.KindMinion) {
				return nil // minions never claim the stone
			}
			a.Target = d.Source
			g.TLogMajorf("c.thePowerStoneFliesTo", g.Entity(d.Source).EDef().Name)
			return nil
		},
	})
}

// artifactBuyoff builds a generic "spend resources → discard this
// attachment" hero ability.
func artifactBuyoff(code, label, costIcons string) func(g *engine.Game, e engine.Entity) []engine.Ability {
	return func(g *engine.Game, e engine.Entity) []engine.Ability {
		ab := engine.Ability{
			Label: engine.S(label), Type: engine.AbilityAction,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
			},
		}
		if costIcons != "" {
			ab.CostIcons = costIcons
		} else {
			ab.Cost = 3
		}
		return []engine.Ability{ab}
	}
}

// readyIdentityReward readies the defeating player's identity.
var readyIdentityReward = func(g *engine.Game, s *engine.SideScheme) []engine.Message {
	return []engine.Message{engine.ReadyEntity{ID: g.ActiveTurn}}
}

// lowestATKEnemy finds the enemy with the lowest ATK (villains first for
// unique targets).
func lowestATKEnemy(g *engine.Game, villainsOnly bool) engine.EntityID {
	best, bestATK := engine.EntityID(""), 1<<30
	for _, id := range cardutil.SortedEnemyIDs(g) {
		atk := 1 << 30
		switch t := g.Entity(id).(type) {
		case *engine.Villain:
			atk = t.AttackVal
		case *engine.Minion:
			if villainsOnly {
				continue
			}
			atk = t.AttackVal
		}
		if atk < bestATK {
			best, bestATK = id, atk
		}
	}
	if best == "" {
		for id := range g.Villains {
			return id
		}
	}
	return best
}

// highestATKEnemy finds the enemy with the highest ATK.
func highestATKEnemy(g *engine.Game, villainsOnly bool) engine.EntityID {
	best, bestATK := engine.EntityID(""), -1
	for _, id := range cardutil.SortedEnemyIDs(g) {
		atk := -1
		switch t := g.Entity(id).(type) {
		case *engine.Villain:
			atk = t.AttackVal
		case *engine.Minion:
			if villainsOnly {
				continue
			}
			atk = t.AttackVal
		}
		if atk > bestATK {
			best, bestATK = id, atk
		}
	}
	if best == "" {
		for id := range g.Villains {
			return id
		}
	}
	return best
}

// highestSCHEnemy finds the villain with the highest scheme.
func highestSCHEnemy(g *engine.Game) engine.EntityID {
	best, bestSCH := engine.EntityID(""), -1
	for id := range g.Villains {
		if v := g.Villains[id]; v != nil && v.SchemeVal > bestSCH {
			best, bestSCH = id, v.SchemeVal
		}
	}
	return best
}

// lowestSCHEnemy finds the villain with the lowest scheme.
func lowestSCHEnemy(g *engine.Game) engine.EntityID {
	best, bestSCH := engine.EntityID(""), 1<<30
	for id := range g.Villains {
		if v := g.Villains[id]; v != nil && v.SchemeVal < bestSCH {
			best, bestSCH = id, v.SchemeVal
		}
	}
	return best
}
