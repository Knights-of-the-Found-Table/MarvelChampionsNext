package sinistermotives

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// registerVenomGoblin installs the Venom Goblin scenario (27113–27126)
// on the three-Manhattan board. The engine holds one main scheme, so
// Lower and Upper Manhattan ride as high-threat side-scheme surrogates
// and the glider counter points at whichever scheme currently hosts it.
func registerVenomGoblin() {
	// Venom Goblin I–III (27113–27115): after activating, the glider
	// moves to the emptiest scheme.
	for _, base := range []string{"27113", "27114", "27115"} {
		b := &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				w, ok := msg.(engine.WindowAfterEnemyAttacked)
				if !ok || w.Enemy != e.EID() {
					return nil
				}
				return gliderMove(g, true)
			},
		}
		if base != "27113" {
			n := 2
			if base == "27115" {
				n = 3
			}
			b.VillainStage = func(g *engine.Game, v *engine.Villain, stage int) []engine.Message {
				var msgs []engine.Message
				for _, p := range g.Players {
					for i := 0; i < n; i++ {
						msgs = append(msgs, engine.DealEncounterToPlayer{Player: p.ID})
					}
				}
				return msgs
			}
		}
		engine.RegisterBehavior(base, b)
	}

	// 27116 Skies Over New York: build the board (called via the
	// scenario setup since this card flips itself away).
	engine.RegisterBehavior("27116", &engine.Behavior{})

	// 27117–27119 Manhattan specials (side-scheme surrogates for Lower
	// and Upper; Midtown is the main scheme).
	engine.RegisterBehavior("27117", &engine.Behavior{})
	engine.RegisterBehavior("27118", &engine.Behavior{})
	engine.RegisterBehavior("27119", &engine.Behavior{})

	// 27120 We Are One: buy-off.
	engine.RegisterBehavior("27120", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil && v.Code[:5] == "27113" {
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
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Spend [energy][mental][physical] → discard We Are One", Type: engine.AbilityAction,
				CostIcons: "energy:1 mental:1 physical:1",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
				},
			}}
		},
	})

	// 27121–27123 symbiote minions: boost riders move the glider.
	engine.RegisterBehavior("27121", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return gliderMove(g, false)
		},
	})
	engine.RegisterBehavior("27122", &engine.Behavior{})
	engine.RegisterBehavior("27123", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			var msgs []engine.Message
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1})
			}
			return msgs
		},
	})

	// 27124 Festering Mass / 27125 Joy Ride.
	engine.RegisterBehavior("27124", &engine.Behavior{})
	engine.RegisterBehavior("27125", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return gliderMove(g, false)
		},
	})

	// 27126 Spreading Panic: resolve the glider scheme's special.
	engine.RegisterBehavior("27126", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			return resolveManhattan(g)
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return resolveManhattan(g)
		},
	})
}

// gliderMove hops the glider counter to the emptiest (least=true) or
// fullest (least=false) Manhattan and fires its special.
func gliderMove(g *engine.Game, least bool) []engine.Message {
	schemes := manhattanIDs(g)
	if len(schemes) == 0 {
		return nil
	}
	pick := schemes[0]
	for _, id := range schemes {
		a, b := threatOf(g, id), threatOf(g, pick)
		if least && a < b {
			pick = id
		}
		if !least && a > b {
			pick = id
		}
	}
	g.GliderCounter = pick
	g.Logf("The glider swoops over %s", manhattanName(g, pick))
	return resolveManhattanOn(g, pick)
}

// resolveManhattan fires the special of the current glider scheme.
func resolveManhattan(g *engine.Game) []engine.Message {
	if g.GliderCounter == "" {
		if g.MainScheme != nil {
			g.GliderCounter = g.MainScheme.ID
		} else {
			return nil
		}
	}
	return resolveManhattanOn(g, g.GliderCounter)
}

// resolveManhattanOn runs a Manhattan district special (approximated:
// the symbiote-environment bonus riders are folded in when any
// symbiote-natured card is in play).
func resolveManhattanOn(g *engine.Game, id engine.EntityID) []engine.Message {
	symbiote := false
	for _, env := range g.Environments {
		if env != nil && env.EDef().HasTrait("symbiote") {
			symbiote = true
		}
	}
	for _, s := range g.SideSchemes {
		if s != nil && s.EDef().HasTrait("symbiote") {
			symbiote = true
		}
	}
	which := ""
	if g.MainScheme != nil && g.MainScheme.ID == id {
		which = engine.BaseCodeOf(g.MainScheme.ECode())
	} else if s := g.SideSchemes[id]; s != nil {
		which = engine.BaseCodeOf(s.ECode())
	}
	switch which {
	case "27117": // Lower Manhattan: threat on each scheme.
		var msgs []engine.Message
		if g.MainScheme != nil {
			msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1})
		}
		for sid := range g.SideSchemes {
			msgs = append(msgs, engine.SchemeThreat{Scheme: sid, N: 1})
		}
		if symbiote && g.MainScheme != nil && g.MainScheme.ID == id {
			msgs = append(msgs, engine.SchemeThreat{Scheme: id, N: 1})
		}
		return msgs
	case "27118": // Midtown: indirect damage.
		n := 2
		if symbiote {
			n = 3
		}
		var msgs []engine.Message
		for _, p := range g.Players {
			msgs = append(msgs, engine.IndirectDamage{Player: p.ID, N: n})
		}
		return msgs
	case "27119": // Upper Manhattan: discard.
		var msgs []engine.Message
		for _, p := range g.Players {
			if len(p.Hand) > 0 {
				msgs = append(msgs, engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0]}})
			}
			if symbiote {
				msgs = append(msgs, engine.MillPlayerDeck{Player: p.ID, N: 4})
			}
		}
		return msgs
	}
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

// manhattanIDs lists the three districts (main + surrogates).
func manhattanIDs(g *engine.Game) []engine.EntityID {
	var out []engine.EntityID
	if g.MainScheme != nil {
		out = append(out, g.MainScheme.ID)
	}
	for _, id := range sortedSchemeIDs(g) {
		if s := g.SideSchemes[id]; s != nil {
			switch engine.BaseCodeOf(s.ECode()) {
			case "27117", "27119":
				out = append(out, id)
			}
		}
	}
	return out
}

func threatOf(g *engine.Game, id engine.EntityID) int {
	if g.MainScheme != nil && g.MainScheme.ID == id {
		return g.MainScheme.Threat
	}
	if s := g.SideSchemes[id]; s != nil {
		return s.Threat
	}
	return 1 << 30
}

func manhattanName(g *engine.Game, id engine.EntityID) string {
	if e := g.Entity(id); e != nil {
		return e.EDef().Name
	}
	return "the city"
}

// registerSMModulars installs the box's modular encounter sets (27127–
// 27189): City in Chaos, Down to Earth, Goblin Gear, Guerrilla Tactics,
// Osborn Tech, Personal Nightmare, Sinister Assault, Symbiotic Strength,
// Whispers of Paranoia, Bad Publicity, Community Service, Snitches Get
// Stitches and S.H.I.E.L.D. Tech.
func registerSMModulars() {
	// 27127 Panic in the Streets: blanks location/persona supports
	// (approximation: none modeled).
	engine.RegisterBehavior("27127", &engine.Behavior{})

	// 27128 Rhino: overkill/piercing minion.
	engine.RegisterBehavior("27128", &engine.Behavior{})

	// 27129 Calling in Favors: Rhino schemes or joins.
	engine.RegisterBehavior("27129", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Minions {
				if m := g.Minions[id]; m != nil && m.Code[:5] == "27128" {
					for vid := range g.Villains {
						return []engine.Message{
							engine.BoostActivation{Enemy: vid, N: 2},
							engine.VillainActivates{VillainID: vid, Player: p.ID},
						}
					}
				}
			}
			for i, c := range g.EncounterDeck {
				if c.Code[:5] == "27128" {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					g.ShuffleEncounterDeck()
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			return nil
		},
	})

	// 27130 Now or Never: peril tax.
	engine.RegisterBehavior("27130", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var opts []engine.Choice
			if g.MainScheme != nil {
				opts = append(opts, engine.Choice{
					ID: "accel", Label: "Place 1 acceleration token on the main scheme", Kind: engine.ChoiceLabel,
				}.Msgs(engine.AddAccelerationToken{Scheme: g.MainScheme.ID}))
			}
			if !p.Exhausted {
				opts = append(opts, engine.Choice{
					ID: "exhaust", Label: "Exhaust your identity + spend 1 resource", Kind: engine.ChoiceLabel,
				}.Msgs(engine.ExhaustEntity{ID: p.ID}))
			}
			if len(opts) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Now or Never: choose one", opts...)}}
		},
	})

	// 27131 Common Criminal: alter-ego action to buy it off.
	engine.RegisterBehavior("27131", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			m := g.Minions[e.EID()]
			if m == nil {
				return nil
			}
			return []engine.Ability{{
				Label: "Spend 1 resource → deal 3 damage to Common Criminal (reward on kill)", Type: engine.AbilityAction,
				AlterEgoOnly: true, Cost: 1,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return append([]engine.Message{
						engine.DamageEntity{Target: self, Damage: 3, Source: g.ActiveTurn},
					}, commonCriminalReward(g)...)
				},
			}}
		},
	})

	// 27132 Friends and Family: hero events cost +1; discard an
	// identity card to drop it.
	engine.RegisterBehavior("27132", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			for _, c := range p.Hand {
				if c.Def().PackCode == "sm" || c.Code[:2] == "27" {
					return []engine.Message{
						engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}},
						engine.ObligationResolve{Player: p.ID, Card: card, Remove: true},
					}
				}
			}
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
	})

	// 27133 Volunteer Work: alter-ego REC thwarts.
	engine.RegisterBehavior("27133", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.SideSchemes[e.EID()]
			if s == nil || s.Threat <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Spend 2 resources → remove threat equal to your REC", Type: engine.AbilityAction,
				AlterEgoOnly: true, Cost: 2,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.ActiveTurn)
					if p == nil {
						return nil
					}
					n := p.RecoverStat(g)
					msgs := []engine.Message{engine.ThwartScheme{Scheme: self, N: n, Source: g.ActiveTurn}}
					if g.EntityHasTrait(p.ID, "civilian") {
						msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
					}
					return msgs
				},
			}}
		},
	})

	// 27134 "Threat or Menace?": form matters.
	engine.RegisterBehavior("27134", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if p.IsHero() && g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID}}
			}
			p.FormChanged = true // denies the next form change
			return nil
		},
	})

	// 27135 Loose Ends: fish out an obligation (approximated: surge).
	engine.RegisterBehavior("27135", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for i, c := range p.ObligationDeck {
				_ = i
				g.Logf("Loose Ends dredges up %s", c.Def().Name)
				p.ObligationDeck = engine.CardList{}
				return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
			}
			if c, ok := g.DrawEncounter(); ok {
				return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
			}
			return nil
		},
	})

	// 27136 Advanced Glider: the villain activates twice.
	engine.RegisterBehavior("27136", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				t.Target = id
				break
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok {
				return nil
			}
			a := g.Attachments[e.EID()]
			if a == nil || a.Target != w.Enemy || g.UsedThisRound["27136"] {
				return nil
			}
			g.UsedThisRound["27136"] = true
			g.Logf("Advanced Glider: %s strikes again", g.Villains[w.Enemy].EDef().Name)
			return []engine.Message{engine.VillainActivates{VillainID: w.Enemy, Player: w.Player}}
		},
	})

	// 27137–27139 bomb attachments.
	bombs := func(label string, penalty func(g *engine.Game, p *engine.Player) []engine.Message) *engine.Behavior {
		return &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				t.Counters = 2
				for id := range g.Villains {
					t.Target = id
					break
				}
				return nil
			},
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				w, ok := msg.(engine.WindowAfterEnemyAttacked)
				if !ok {
					return nil
				}
				a := g.Attachments[e.EID()]
				if a == nil || a.Target != w.Enemy || a.Counters <= 0 {
					return nil
				}
				a.Counters--
				p := g.Player(w.Player)
				if p == nil {
					return nil
				}
				return penalty(g, p)
			},
		}
	}
	engine.RegisterBehavior("27137", bombs("exhaust", func(g *engine.Game, p *engine.Player) []engine.Message {
		var msgs []engine.Message
		if len(p.Upgrades) > 0 {
			msgs = append(msgs, engine.ExhaustEntity{ID: p.Upgrades[0]})
		}
		if len(p.Supports) > 0 {
			msgs = append(msgs, engine.ExhaustEntity{ID: p.Supports[0]})
		}
		return msgs
	}))
	engine.RegisterBehavior("27138", bombs("indirect", func(g *engine.Game, p *engine.Player) []engine.Message {
		return []engine.Message{engine.IndirectDamage{Player: p.ID, N: 2}}
	}))
	engine.RegisterBehavior("27139", bombs("discard", func(g *engine.Game, p *engine.Player) []engine.Message {
		if len(p.Hand) > 0 {
			return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0]}}}
		}
		return nil
	}))

	// 27140 Limitless Supply / 27141 Remote Navigation.
	engine.RegisterBehavior("27140", &engine.Behavior{})
	engine.RegisterBehavior("27141", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for _, a := range g.Attachments {
				if a != nil && a.Code[:5] == "27136" {
					for id := range g.Villains {
						return []engine.Message{engine.VillainActivates{VillainID: id, Player: p.ID}}
					}
				}
			}
			for i, c := range g.EncounterDeck {
				if c.Code[:5] == "27136" {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for id := range g.Villains {
				return []engine.Message{engine.DealBoost{Enemy: id}, engine.DealBoost{Enemy: id}}
			}
			return nil
		},
	})

	// 27142 Life-Size Decoy: no side-scheme thwarts (approximation:
	// patrol-like main-scheme block registered via keyword handling).
	engine.RegisterBehavior("27142", &engine.Behavior{})

	// 27143–27146 Guerrilla Tactics schemes.
	engine.RegisterBehavior("27143", &engine.Behavior{})
	engine.RegisterBehavior("27144", &engine.Behavior{})
	engine.RegisterBehavior("27145", &engine.Behavior{})
	engine.RegisterBehavior("27146", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			n := 1
			for range g.Minions {
				n++
			}
			for range g.Villains {
				n++
			}
			if g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n - 1, Source: t.ID}}
		},
	})

	// 27147–27152 Osborn Tech attachments: villain auras with buy-offs.
	villainAttach := func() func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
		return func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				t.Target = id
				break
			}
			return nil
		}
	}
	engine.RegisterBehavior("27147", &engine.Behavior{
		OnAttach: villainAttach(),
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Discard your highest-cost upgrade → discard Arm Cannon", Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.ActiveTurn)
					if p == nil || len(p.Upgrades) == 0 {
						return nil
					}
					_, u := highestCostUpgrade(g, p)
					return []engine.Message{
						engine.DiscardControlled{Player: p.ID, ID: u},
						engine.DiscardAttachmentMsg{ID: self},
					}
				},
			}}
		},
	})
	engine.RegisterBehavior("27148", &engine.Behavior{OnAttach: villainAttach()})
	engine.RegisterBehavior("27149", &engine.Behavior{OnAttach: villainAttach()})
	engine.RegisterBehavior("27150", &engine.Behavior{OnAttach: villainAttach()})
	engine.RegisterBehavior("27151", &engine.Behavior{OnAttach: villainAttach()})
	engine.RegisterBehavior("27152", &engine.Behavior{OnAttach: villainAttach()})

	// 27153 Induced Panic: identity blanking (approximation: AE action
	// to strip it).
	engine.RegisterBehavior("27153", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if p := g.Player(target); p != nil {
				t.Target = p.ID
			} else {
				t.Target = cardutil.FirstPlayerID(g)
			}
			return nil
		},
	})

	// 27154 Evil Doppelgänger: scales with identity cards in hand.
	engine.RegisterBehavior("27154", &engine.Behavior{})

	// 27155 Fool's Paradise: +2 hand size aura.
	engine.RegisterBehavior("27155", &engine.Behavior{})

	// 27156 Weakness from Within: threat per hand card.
	engine.RegisterBehavior("27156", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if s := g.SideSchemes[e.EID()]; s != nil {
				p := g.Player(g.EOwnerIfPlayer())
				if p == nil {
					p = g.Players[0]
				}
				s.Threat += len(p.Hand)
			}
			return nil
		},
	})

	// 27157 Deepest Fears: mill your deck size.
	engine.RegisterBehavior("27157", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: len(p.Hand)}}
		},
	})

	// 27158–27163 Sinister Assault minions (Villainous riders).
	for _, base := range []string{"27158", "27159", "27160", "27161", "27162", "27163"} {
		engine.RegisterBehavior(base, &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				w, ok := msg.(engine.WindowAfterEnemyAttacked)
				if !ok || w.Enemy != e.EID() {
					return nil
				}
				p := g.Player(w.Player)
				if p == nil {
					return nil
				}
				switch base {
				case "27158":
					var msgs []engine.Message
					if g.MainScheme != nil {
						msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()})
					}
					return msgs
				case "27160":
					return []engine.Message{engine.IndirectDamage{Player: p.ID, N: 2}}
				case "27161":
					if len(p.Upgrades) > 0 {
						return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[len(p.Upgrades)-1]}}
					}
					if len(p.Supports) > 0 {
						return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Supports[len(p.Supports)-1]}}
					}
				case "27162":
					return []engine.Message{engine.StunEntity{Target: p.ID}}
				case "27163":
					if len(p.Hand) > 0 {
						return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{p.Hand[0]}}}
					}
				}
				return nil
			},
		})
	}

	// 27164–27169 Symbiotic Strength.
	engine.RegisterBehavior("27164", &engine.Behavior{OnAttach: villainAttach()})
	engine.RegisterBehavior("27165", &engine.Behavior{OnAttach: villainAttach()})
	engine.RegisterBehavior("27166", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if p := g.Player(target); p != nil {
				t.Target = p.ID
			} else {
				t.Target = cardutil.FirstPlayerID(g)
			}
			return nil
		},
	})
	engine.RegisterBehavior("27167", &engine.Behavior{})
	engine.RegisterBehavior("27168", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				return []engine.Message{
					engine.DealBoost{Enemy: id},
					engine.VillainActivates{VillainID: id, Player: p.ID},
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("27169", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for id := range g.Villains {
				return []engine.Message{engine.DealBoost{Enemy: id}}
			}
			return nil
		},
	})

	// 27170–27173 Whispers of Paranoia.
	engine.RegisterBehavior("27170", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if p := g.Player(target); p != nil {
				t.Target = p.ID
			} else {
				t.Target = cardutil.FirstPlayerID(g)
			}
			return nil
		},
	})
	engine.RegisterBehavior("27171", &engine.Behavior{})
	engine.RegisterBehavior("27172", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1})
			}
			return msgs
		},
	})
	engine.RegisterBehavior("27173", &engine.Behavior{})

	// 27174–27175 Bad Publicity.
	engine.RegisterBehavior("27174", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if s := g.Supports[e.EID()]; s != nil {
				// Environment-typed: handled below.
				_ = s
			}
			if env := g.Environments[e.EID()]; env != nil {
				env.Counters = 2 * len(g.Players)
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			switch msg.(type) {
			case engine.MinionDefeated:
				if env := g.Environments[e.EID()]; env != nil && env.Counters > 0 {
					env.Counters--
					g.Logf("Public Outcry: %d notoriety counters left", env.Counters)
				}
			case engine.SchemeDefeated:
				if msg.(engine.SchemeDefeated).Scheme != e.EID() {
					if env := g.Environments[e.EID()]; env != nil && env.Counters > 0 {
						env.Counters--
						g.Logf("Public Outcry: %d notoriety counters left", env.Counters)
					}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("27175", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for _, env := range g.Environments {
				if env != nil && env.Code[:5] == "27174" {
					env.Counters += 2
					return nil
				}
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID}}
			}
			return nil
		},
	})

	// 27176–27180 Community Service victory schemes.
	engine.RegisterBehavior("27176", &engine.Behavior{})
	engine.RegisterBehavior("27177", &engine.Behavior{})
	engine.RegisterBehavior("27178", &engine.Behavior{})
	engine.RegisterBehavior("27179", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			bp, ok := msg.(engine.BeginPhase)
			if !ok || bp.Phase != engine.PhaseVillain {
				return nil
			}
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			s.Counters++
			if s.Counters >= 2 {
				g.Delete(e.EID())
				var msgs []engine.Message
				for _, p := range g.Players {
					msgs = append(msgs, engine.MillPlayerDeck{Player: p.ID, N: 3})
				}
				return msgs
			}
			return nil
		},
	})
	engine.RegisterBehavior("27180", &engine.Behavior{})

	// 27181 Snitches Get Stitches: attaches to the Venom ally.
	engine.RegisterBehavior("27181", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Allies {
				if a := g.Allies[id]; a != nil && a.Code[:5] == "27190" {
					t.Target = id
					g.Logf("Snitches Get Stitches attaches to Venom")
					return nil
				}
			}
			return []engine.Message{engine.DiscardAttachmentMsg{ID: t.ID}}
		},
	})

	// 27182–27189 S.H.I.E.L.D. Tech permanent upgrades.
	engine.RegisterBehavior("27182", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ba, ok := msg.(engine.BasicAttack)
			if !ok || ba.Player != e.EOwner() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.Counters <= 0 {
				return nil
			}
			u.Counters--
			return cardutil.ChooseEnemy("Compact Darts: deal 1 damage",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 1, nil })(g, e)
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Spend 1 resource → place 3 dart counters", Type: engine.AbilityAction,
				AlterEgoOnly: true, Cost: 1, OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.AddEntityCounter{ID: self, N: 3}}
				},
			}}
		},
	})
	engine.RegisterBehavior("27183", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP += 2
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Spend 1 resource → -1 damage from each enemy attack this phase", Type: engine.AbilityAction,
				HeroOnly: true, Cost: 1,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), DEF: 1}}
				},
			}}
		},
	})
	engine.RegisterBehavior("27184", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{ATK: 1, THW: -1}
		},
	})
	engine.RegisterBehavior("27185", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Propulsion Gauntlet + take 2 indirect damage → ready your hero", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.IndirectDamage{Player: e.EOwner(), N: 2},
						engine.ReadyEntity{ID: e.EOwner()},
					}
				},
			}}
		},
	})
	engine.RegisterBehavior("27186", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{THW: 1}
		},
	})
	engine.RegisterBehavior("27187", &engine.Behavior{})
	engine.RegisterBehavior("27188", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{ATK: -1, DEF: 1, Retaliate: 1}
		},
	})
	engine.RegisterBehavior("27189", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			switch m := msg.(type) {
			case engine.MinionEntersPlay:
				if u := g.Upgrades[e.EID()]; u != nil {
					u.AttachTo = m.MinionID
				}
			case engine.MinionDefeated:
				if u := g.Upgrades[e.EID()]; u != nil && u.AttachTo == m.MinionID {
					u.AttachTo = ""
					return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
				}
			}
			return nil
		},
	})
}

// commonCriminalReward draws or thwarts on the kill.
func commonCriminalReward(g *engine.Game) []engine.Message {
	return []engine.Message{engine.DrawCards{Player: g.ActiveTurn, N: 1}}
}

// highestCostUpgrade finds the priciest upgrade of a player.
func highestCostUpgrade(g *engine.Game, p *engine.Player) (int, engine.EntityID) {
	best, bestCost := engine.EntityID(""), -1
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil {
			c := cardutil.Cost(u.EDef())
			if c > bestCost {
				best, bestCost = id, c
			}
		}
	}
	return bestCost, best
}
