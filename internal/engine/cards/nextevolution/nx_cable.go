package nextevolution

import (
	"fmt"
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func registerCableCards() {
	// 40002 Bodyslide: change form; each other player may follow.
	engine.RegisterBehavior("40002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			msgs := []engine.Message{engine.ChangeForm{Player: p.ID}}
			for _, o := range g.Players {
				if o.ID == p.ID || o.KOed {
					continue
				}
				msgs = append(msgs, engine.AskQuestion{
					Player: o.ID,
					Question: engine.Ask(engine.Tf("c.bodyslideChangeToTheSameFormAs", o.Name, p.Name),
						engine.Choice{ID: "yes", Label: engine.Tf("c.changeForm"), Kind: engine.ChoiceLabel}.Msgs(engine.ChangeForm{Player: o.ID}),
						engine.Choice{ID: "no", Label: engine.Tf("c.stay"), Kind: engine.ChoicePass}),
				})
			}
			return msgs
		},
	})

	// 40003 Mind Scan: remove 3 threat +1 per side scheme in the victory
	// display.
	engine.RegisterBehavior("40003", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme(engine.Tf("c.mindScanChooseAScheme"), func(g *engine.Game, e engine.Entity) int {
			return 3 + victorySideSchemes(g)
		}),
	})

	// 40004 Precognition: look at the top X encounter cards (X = victory
	// display side schemes); may discard 1 of them.
	engine.RegisterBehavior("40004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			x := victorySideSchemes(g)
			if x == 0 {
				g.TLogf("c.precognitionNoSideSchemesInTheVictoryDisplay")
				return nil
			}
			var names []string
			for i := 0; i < x && i < len(g.EncounterDeck); i++ {
				names = append(names, g.EncounterDeck[i].Def().Name)
			}
			var choices []engine.Choice
			for i := 0; i < x && i < len(g.EncounterDeck); i++ {
				c := g.EncounterDeck[i]
				choices = append(choices, engine.Choice{
					ID: fmt.Sprintf("drop-%d", i), Label: engine.Tf("m.discardCard", c), Kind: engine.ChoiceLabel,
				}.Msgs(engine.EncounterTakeCard{CardID: c.ID}))
			}
			choices = append(choices, cardutil.Skip())
			g.TLogf("c.precognitionSees", names)
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.precognitionDiscardOneOfTheRevealedEncounterCards"), choices...),
			}}
		},
	})

	// 40005 Telekinetic Blast: 6 damage +1 per victory display side scheme.
	engine.RegisterBehavior("40005", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy(engine.Tf("c.telekineticBlast"), func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
			return 6 + victorySideSchemes(g), nil
		}),
	})

	// 40006 Technovirus Purge: player side scheme; when defeated it joins
	// the victory display. The ongoing PSIONIC/stat rider while in the
	// victory display is not modeled (no hook).
	engine.RegisterBehavior("40006", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			// The generic defeat handler already moves "Victory" schemes to
			// the display; this hook only logs the ongoing rider.
			g.TLogf("c.technovirusPurgeJoinsTheVictoryDisplayPsionicRiderNotModeled")
			return nil
		},
	})

	// 40007 Graymalkin: readies after a side scheme is defeated; resource
	// ability for an energy icon.
	engine.RegisterBehavior("40007", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "energy"},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.SchemeDefeated); !ok {
				return nil
			}
			s := e.(*engine.Support)
			if s.Exhausted {
				s.Exhausted = false
				g.TLogf("c.graymalkinReadies")
			}
			return nil
		},
	})

	// 40008 Professor: alter-ego action — draw 1 or fetch a player side
	// scheme to hand.
	engine.RegisterBehavior("40008", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.professorDraw1CardOrFetchAPlayerSideScheme"), Type: engine.AbilityAction,
				AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil {
						return nil
					}
					var fetch []engine.Message
					if c, zone, ok := firstCardWhere(p, func(d *data.CardDef) bool {
						return d.Type == "player_side_scheme"
					}); ok {
						takeFromZone(p, c, zone)
						p.Hand = append(p.Hand, c)
						g.TLogf("c.fetchesFromTheir", p.Name, c, zone)
						if zone == "deck" {
							fetch = append(fetch, engine.ShufflePlayerDeck{Player: p.ID})
						}
					}
					return []engine.Message{engine.AskQuestion{
						Player: p.ID,
						Question: engine.Ask(engine.Tf("c.professorChoose"),
							engine.Choice{ID: "draw", Label: engine.Tf("c.draw1Card"), Kind: engine.ChoiceLabel}.
								Msgs(engine.DrawCards{Player: p.ID, N: 1}),
							engine.Choice{ID: "fetch", Label: engine.Tf("c.searchForAPlayerSideScheme"), Kind: engine.ChoiceLabel}.
								Msgs(fetch...),
						)},
					}
				},
			}}
		},
	})

	// 40009 Askani'son: after you defend, exhaust + energy → remove THW
	// threat from a scheme (the energy payment auto-spends the first
	// matching card in hand).
	engine.RegisterBehavior("40009", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowDefended)
			if !ok || w.Defender != e.EOwner() || e.EExhausted() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			for _, c := range p.Hand {
				for _, r := range c.Def().Resources {
					if r != "energy" && r != "wild" {
						continue
					}
					n := p.ThwartStat(g)
					var choices []engine.Choice
					for _, id := range g.Schemes() {
						s := g.Entity(id)
						choices = append(choices, engine.Choice{
							Label: engine.Tf("m.cardName", s), Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.ECode(),
						}.Msgs(engine.ThwartScheme{Scheme: id, N: n, Source: p.ID}))
					}
					return []engine.Message{
						engine.ExhaustEntity{ID: u.ID},
						engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}},
						engine.AskQuestion{Player: p.ID, Question: engine.Ask(
							engine.Tf("c.askaniSonSpendForEnergyAndRemoveThreatFrom", c, n), choices...)},
					}
				}
			}
			return nil
		},
	})

	// 40010 Forced Amnesia: after a non-permanent side scheme is defeated,
	// add it and this card to the victory display.
	engine.RegisterBehavior("40010", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || g.SideSchemes[m.Scheme] == nil || e.EExhausted() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			if !strings.Contains(g.SideSchemes[m.Scheme].EDef().Text, "Victory") {
				g.VictoryDisplay = append(g.VictoryDisplay, engine.Card{ID: g.NextCardID(), Code: g.SideSchemes[m.Scheme].Code})
			}
			g.VictoryDisplay = append(g.VictoryDisplay, engine.Card{ID: g.NextCardID(), Code: u.Code})
			g.Delete(u.ID)
			g.TLogf("c.forcedAmnesiaAddsTheSchemeAndItselfToTheVictoryDisplay")
			return nil
		},
	})

	// 40011 Plasma Rifle: exhaust + energy → 1 damage per victory display
	// side scheme (max 4).
	engine.RegisterBehavior("40011", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			n := victorySideSchemes(g)
			if n > 4 {
				n = 4
			}
			return []engine.Ability{{
				Label: engine.Tf("c.plasmaRifleDealDamageToAnEnemy", n), Type: engine.AbilityAction,
				HeroOnly: true, Exhaust: true, CostIcons: "energy:1",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					_ = self
					return cardutil.ChooseEnemy(engine.Tf("c.plasmaRifle"), func(g *engine.Game, e engine.Entity) (int, []engine.Message) {
						return n, nil
					})(g, e)
				},
			}}
		},
	})

	// 40012 Telekinetic Force Field: discard → prevent all damage to a
	// friendly character (auto-used; approximation).
	engine.RegisterBehavior("40012", &engine.Behavior{
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (prevented, reflect int) {
			g.Push(engine.DiscardControlled{Player: u.Owner, ID: u.ID})
			g.TLogf("c.telekineticForceFieldPreventsDamage", n)
			return n, 0
		},
	})

	// 40013 Temporal Leap: when the main scheme would complete, remove from
	// game + return a victory display side scheme → move 4 threat to it.
	// (Approximation: the first victory display side scheme returns
	// automatically.)
	engine.RegisterBehavior("40013", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MainSchemeMaxed)
			if !ok || g.MainScheme == nil || m.Scheme != g.MainScheme.ID {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if g.Player(u.Owner) == nil {
				return nil
			}
			for i, c := range g.VictoryDisplay {
				if c.Def().Type != "side_scheme" && c.Def().Type != "player_side_scheme" {
					continue
				}
				g.VictoryDisplay = append(g.VictoryDisplay[:i:i], g.VictoryDisplay[i+1:]...)
				g.Delete(u.ID)
				move := 4
				if g.MainScheme.Threat < move {
					move = g.MainScheme.Threat
				}
				g.MainScheme.Threat -= move
				spawnSideSchemeCard(g, c.Code, move)
				g.TLogf("c.temporalLeapRemovesItselfFromTheGameAndPullsBackOutThreatMov", c, move)
				return nil
			}
			return nil
		},
	})

	// 40031 Technovirus Resurgence: fetch Technovirus Purge into play;
	// otherwise discard and take a facedown encounter card.
	engine.RegisterBehavior("40031", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if c, zone, ok := firstCardOf(p, "40006"); ok {
				takeFromZone(p, c, zone)
				p.Hand = append(p.Hand, c)
				g.TLogf("c.pullsTechnovirusPurgeFromTheir", p.Name, zone)
				var extra []engine.Message
				if zone == "deck" {
					extra = append(extra, engine.ShufflePlayerDeck{Player: p.ID})
				}
				return append([]engine.Message{engine.PlayCard{Player: p.ID, Card: c}}, extra...)
			}
			g.TLogf("c.technovirusPurgeNotFoundTakesAFacedownEncounterCard", p.Name)
			return []engine.Message{
				engine.ObligationResolve{Player: p.ID, Card: card},
				engine.DealEncounterToPlayer{Player: p.ID},
			}
		},
	})

	// 40032 Stryfe (Cable's nemesis minion): when a player plays a PSIONIC
	// event, Stryfe takes 1 damage (the "cancel the event" half is not
	// modeled).
	engine.RegisterBehavior("40032", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.EventPlayed)
			if !ok {
				return nil
			}
			if !m.Card.Def().HasTrait("Psionic") {
				return nil
			}
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			g.TLogf("c.stryfeAbsorbsThePsionicBacklash1Damage")
			return []engine.Message{engine.DamageEntity{Target: mn.ID, Damage: 1, Source: m.Player}}
		},
	})

	// 40033 Back to the Future: threat locks live in the engine's
	// removeThreat; the damage locks are approximated away.
	engine.RegisterBehavior("40033", &engine.Behavior{})

	// 40034 Telekinetic Force Field (Stryfe's attachment): damage
	// absorption lives in the engine's damage().
	engine.RegisterBehavior("40034", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, mn := range g.Minions {
				if mn != nil && engine.BaseCodeOf(mn.Code) == "40032" {
					t.Target = mn.ID
					g.TLogf("c.telekineticForceFieldAttachesToStryfe")
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
			// [star] Boost: attach this card to the activating enemy
			// (approximation: the first villain).
			if id := boostEnemy(g); id != "" {
				t := &engine.Attachment{ID: g.NextEntityID(engine.KindAttachment), Code: "40034", Target: id}
				g.Attachments[t.ID] = t
				g.TLogf("c.telekineticForceFieldAttachesToTheVillain")
			}
			return nil
		},
	})

	// 40035 Mind Scan (treachery): 2 threat +1 per victory display side
	// scheme; boost: confused.
	engine.RegisterBehavior("40035", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if g.MainScheme == nil {
				return nil
			}
			n := 2 + victorySideSchemes(g)
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: t.ID}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return confuseBoost(g, card)
		},
	})

	// 40036 Telekinetic Blast (treachery): 2 damage +1 per victory display
	// side scheme; boost: stunned.
	engine.RegisterBehavior("40036", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			n := 2 + victorySideSchemes(g)
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: n, Source: t.ID}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return stunBoost(g, card)
		},
	})
}
