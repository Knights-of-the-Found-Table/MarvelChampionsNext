// Package ant registers the Ant-Man hero pack. Ant-Man's identity is
// triple-sided in print; this implementation models Tiny/Giant as traits
// granted on the player whenever they change to hero form (the flip asks
// which form), and form-dependent riders are composed into the same
// answer so they resolve in order.
package ant

import (
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerAntMan()
	registerPackCards()
	registerNemesis()
}

func stripForm(p *engine.Player) {
	var kept []string
	for _, t := range p.ExtraTraits {
		if t != "giant" && t != "tiny" {
			kept = append(kept, t)
		}
	}
	p.ExtraTraits = kept
}

// InForm reports the player's current Ant-Man form ("" when neither).
func InForm(p *engine.Player) string {
	for _, t := range p.ExtraTraits {
		if t == "giant" || t == "tiny" {
			return t
		}
	}
	return ""
}

func registerAntMan() {
	engine.RegisterBehavior("12001", &engine.Behavior{
		// Changing to hero form asks Tiny or Giant; changing back strips
		// the form traits.
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var pid engine.PlayerID
			switch m := msg.(type) {
			case engine.ChangeForm:
				pid = m.Player
			case engine.ChangeFormAgain:
				pid = m.Player
			default:
				return nil
			}
			if pid != e.EID() {
				return nil
			}
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			// Reactions run before the flip applies: pre-flip alter-ego
			// means the player is changing TO a hero form.
			if !p.IsHero() {
				giant := engine.Choice{ID: "giant", Label: engine.Tf("c.giantForm"), Kind: engine.ChoiceForm}
				tiny := engine.Choice{ID: "tiny", Label: engine.Tf("c.tinyForm"), Kind: engine.ChoiceForm}
				gMsgs := withForm(g, p, "giant")
				tMsgs := withForm(g, p, "tiny")
				giant = giant.Msgs(gMsgs...)
				tiny = tiny.Msgs(tMsgs...)
				return []engine.Message{engine.AskQuestion{Player: p.ID,
					Question: engine.Ask(engine.Tf("c.whichHeroForm"), giant, tiny)}}
			}
			return []engine.Message{engine.SetAntForm{Player: p.ID, Form: ""}}
		},
		// Setup: search deck and discard for Ant-Man's Helmet.
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			var picks []engine.Choice
			seen := map[string]bool{}
			for _, c := range append(append(engine.CardList{}, p.Deck...), p.Discard...) {
				if c.Code != "12008" || seen[c.ID] {
					continue
				}
				seen[c.ID] = true
				picks = append(picks, engine.Choice{Label: engine.Tf("c.antManSHelmet"), Kind: engine.ChoiceCard, CardCode: c.Code}.
					Msgs(engine.UpgradeEnterPlay{Player: p.ID, Card: c}))
			}
			if len(picks) == 0 {
				return nil
			}
			picks = append(picks, engine.Choice{ID: "skip", Label: engine.Tf("c.skip"), Kind: engine.ChoicePass})
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.setupPutAntManSHelmetIntoPlay"), picks...)}}
		},
	})
}

// withForm snapshots setForm's messages without mutating yet (the choice
// answer applies them; the mutation itself is repeated inside via a
// marker message so it happens on resolution).
func withForm(g *engine.Game, p *engine.Player, form string) []engine.Message {
	msgs := []engine.Message{engine.SetAntForm{Player: p.ID, Form: form}}
	msgs = append(msgs, formRiders(g, p, form)...)
	return msgs
}

// formRiders builds the rider effects for a form without mutating.
func formRiders(g *engine.Game, p *engine.Player, form string) []engine.Message {
	var msgs []engine.Message
	switch form {
	case "giant":
		for _, id := range p.Upgrades {
			if u := g.Upgrades[id]; u != nil {
				switch u.Code {
				case "12008":
					msgs = append(msgs, engine.HealEntity{Target: p.ID, N: 2})
				case "12009":
					msgs = append(msgs, engine.ApplyStatBonus{Target: p.ID, ATK: 1})
				case "12016":
					msgs = append(msgs, engine.ApplyStatBonus{Target: p.ID, THW: 1, ATK: 1, DEF: 1})
				}
			}
		}
		msgs = append(msgs, cardutil.ChooseEnemy(engine.Tf("c.giantNuisanceDeal1DamageToWhichEnemy"),
			func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 1, nil })(
			g, engine.Entity(&engine.EventCard{Code: "12001", Owner: p.ID}))...)
	case "tiny":
		for _, id := range p.Upgrades {
			if u := g.Upgrades[id]; u != nil {
				switch u.Code {
				case "12008":
					msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
				case "12016":
					msgs = append(msgs, engine.ApplyStatBonus{Target: p.ID, THW: 1, ATK: 1, DEF: 1})
				}
			}
		}
	}
	return msgs
}

func registerPackCards() {
	// Wasp: on play, giant → 2 damage / tiny → remove 2 threat.
	engine.RegisterBehavior("12002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			switch InForm(p) {
			case "giant":
				return cardutil.ChooseEnemy(engine.Tf("c.waspDeal2DamageToWhichEnemy"),
					func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 2, nil })(g, e)
			case "tiny":
				return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
					engine.Tf("c.waspRemove2ThreatFromWhichScheme"), schemePicks(g, 2, p.ID)...)}}
			}
			return nil
		},
	})

	// Giant Stomp: 1 to each minion + 8 to an enemy (form restriction
	// approximated as always playable).
	engine.RegisterBehavior("12003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, id := range cardutil.SortedIDs(g.Minions) {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: e.EOwner()})
			}
			msgs = append(msgs, cardutil.ChooseEnemy(engine.Tf("c.giantStompDeal8DamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 8, nil })(g, e)...)
			return msgs
		},
	})

	// Hive Mind: 2 threat + 1 per Army of Ants.
	engine.RegisterBehavior("12004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 2
			p := g.Player(e.EOwner())
			if p != nil {
				for _, id := range p.Supports {
					if s := g.Supports[id]; s != nil && s.Code == "12007" {
						n++
					}
				}
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				engine.Tf("c.hiveMindRemoveThreatFromWhichScheme"), schemePicks(g, n, e.EOwner())...)}}
		},
	})

	// Resize: flip form + draw 1.
	engine.RegisterBehavior("12005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			msgs := []engine.Message{engine.ChangeFormAgain{Player: p.ID}}
			return append(msgs, engine.DrawCards{Player: p.ID, N: 1})
		},
	})

	// Pym Particles: spent-response; the payment layer has no
	// spent-card hook, registered as a plain resource.
	engine.RegisterBehavior("12006", &engine.Behavior{})

	// Army of Ants: exhaust → 1 damage (tiny form).
	engine.RegisterBehavior("12007", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || InForm(p) != "tiny" {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustArmyOfAntsDeal1Damage"), Type: engine.AbilityAction, Exhaust: true,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return cardutil.ChooseEnemy(engine.Tf("c.armyOfAntsDeal1DamageToWhichEnemy"),
						func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 1, nil })(
						g, g.Entity(self))
				},
			}}
		},
	})

	// Ant-Man's Helmet: riders composed into the form flip.
	engine.RegisterBehavior("12008", &engine.Behavior{})

	// Giant Strength: rider composed into the form flip.
	engine.RegisterBehavior("12009", &engine.Behavior{})

	// Wrist Gauntlets: form-dependent stun/confuse.
	engine.RegisterBehavior("12010", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var ab []engine.Ability
			switch InForm(p) {
			case "giant":
				ab = append(ab, engine.Ability{
					Label: engine.Tf("c.exhaustSpendPhysicalPhysicalStunAnEnemy"), Type: engine.AbilityAction,
					Exhaust: true, Cost: 2, CostIcons: "physical:2", HeroOnly: true,
					Execute: stunChoice("Wrist Gauntlets: stun which enemy?"),
				})
			case "tiny":
				ab = append(ab, engine.Ability{
					Label: engine.Tf("c.exhaustSpendEnergyEnergyConfuseAnEnemy"), Type: engine.AbilityAction,
					Exhaust: true, Cost: 2, CostIcons: "energy:2", HeroOnly: true,
					Execute: confuseChoice("Wrist Gauntlets: confuse which enemy?"),
				})
			}
			return ab
		},
	})

	// Ant-Man (ally): pym counters via overpay are not tracked by the
	// payment layer; enters with no counters.
	engine.RegisterBehavior("12011", &engine.Behavior{})

	// Giant-Man: +2 ATK at 3+ remaining HP.
	engine.RegisterBehavior("12012", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			switch msg.(type) {
			case engine.DamageEntity, engine.HealEntity:
				want := 0
				if a.HP() >= 3 {
					want = 2
				}
				a.PermATK = want
			}
			return nil
		},
	})

	// Ronin: +1 THW/+1 ATK while an upgrade is attached.
	engine.RegisterBehavior("12013", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			switch msg.(type) {
			case engine.AttachUpgrade, engine.DiscardControlled, engine.AllyAttackWindow, engine.AllyThwartWindow:
				want := 0
				p := g.Player(a.Owner)
				if p != nil {
					for _, id := range p.Upgrades {
						if u := g.Upgrades[id]; u != nil && u.AttachTo == a.ID {
							want = 1
							break
						}
					}
				}
				a.PermATK = want
				a.PermTHW = want
			}
			return nil
		},
	})

	// Stinger: no ally limit is enforced by the engine.
	engine.RegisterBehavior("12014", &engine.Behavior{})

	// Call for Aid: mill until an Avenger ally joins the hand.
	engine.RegisterBehavior("12015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for len(p.Deck) > 0 {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				if c.Def().Type == "ally" && c.Def().HasTrait("avenger") {
					p.Hand = append(p.Hand, c)
					g.TLogf("c.findsCallForAid", p.Name, c)
					return nil
				}
				p.Discard = append(p.Discard, c)
			}
			return nil
		},
	})

	// Moxie: rider composed into the form flip.
	engine.RegisterBehavior("12016", &engine.Behavior{})

	// Power Gloves: after the attached ally attacks or thwarts, 1 damage.
	engine.RegisterBehavior("12017", &engine.Behavior{
		OnPlay: attachToAllyPick("Power Gloves"),
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			var ally engine.EntityID
			switch w := msg.(type) {
			case engine.AllyAttackWindow:
				ally = w.Ally
			case engine.AllyThwartWindow:
				ally = w.Ally
			default:
				return nil
			}
			if ally != u.AttachTo {
				return nil
			}
			return cardutil.ChooseEnemy(engine.Tf("c.powerGlovesDeal1DamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 1, nil })(g, e)
		},
	})

	// Reinforced Suit: attached ally +2 HP.
	engine.RegisterBehavior("12018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if m := attachPickMsg(g, e, "Reinforced Suit", 2); m != nil {
				return []engine.Message{m}
			}
			return nil
		},
	})

	// First Aid: heal 2 from any character.
	engine.RegisterBehavior("12019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, q := range g.Players {
				if q.Damage > 0 {
					picks = append(picks, engine.Choice{Label: engine.S(q.Name), Kind: engine.ChoiceTarget, SourceID: q.ID}.
						Msgs(engine.HealEntity{Target: q.ID, N: 2}))
				}
				for _, id := range q.Allies {
					if a := g.Allies[id]; a != nil && a.Damage > 0 {
						picks = append(picks, engine.Choice{Label: engine.S(a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
							Msgs(engine.HealEntity{Target: a.ID, N: 2}))
					}
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask(engine.Tf("c.firstAidHealWhichCharacter"), picks...)}}
		},
	})

	// Swarm Tactics: change form + ready.
	engine.RegisterBehavior("12020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{
				engine.ChangeFormAgain{Player: e.EOwner()},
				engine.ReadyEntity{ID: e.EOwner()},
			}
		},
	})

	// Basic resources.
	engine.RegisterBehavior("12021", &engine.Behavior{})
	engine.RegisterBehavior("12022", &engine.Behavior{})
	engine.RegisterBehavior("12023", &engine.Behavior{})

	// Team-Building Exercise: exhaust → next trait-sharing card -1.
	engine.RegisterBehavior("12024", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustTeamBuildingExerciseNextTraitSharingCardCosts1Less"), Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.Entity(self).EOwner())
					if p == nil {
						return nil
					}
					hero := p.HeroDef()
					var picks []engine.Choice
					seen := map[string]bool{}
					for _, c := range p.Hand {
						def := c.Def()
						if seen[c.Code] || def.Cost == nil || *def.Cost <= 0 {
							continue
						}
						match := ""
						for _, t := range hero.Traits {
							if def.HasTrait(t) {
								match = t
								break
							}
						}
						if match == "" {
							continue
						}
						seen[c.Code] = true
						picks = append(picks, engine.Choice{Label: engine.S(def.Name + " (" + match + ")"), Kind: engine.ChoiceCard, CardCode: def.Code}.
							Msgs(engine.CostDiscountApply{Player: p.ID, Amount: 1}))
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask(engine.Tf("c.discountWhichCardAnyOfItsSharedTraits"), picks...)}}
				},
			}}
		},
	})

	// Care for Cassie: exhaust to remove from game, or discard a card.
	engine.RegisterBehavior("12025", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var picks []engine.Choice
			if !p.Exhausted {
				picks = append(picks, engine.Choice{ID: "exhaust", Label: engine.Tf("c.exhaustScottLangRemoveFromTheGame"), Kind: engine.ChoiceLabel}.
					Msgs(engine.ExhaustEntity{ID: p.ID},
						engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			if len(p.Hand) > 0 {
				var subs []engine.Choice
				for _, c := range p.Hand {
					subs = append(subs, engine.Choice{Label: engine.Tf("m.discardCard", c), Kind: engine.ChoiceCard, CardCode: c.Code}.
						Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}},
							engine.ObligationResolve{Player: p.ID, Card: card}))
				}
				picks = append(picks, engine.Choice{ID: "discard", Label: engine.Tf("c.discard1CardFromHand"), Kind: engine.ChoiceCard}.
					WithThen(engine.Ask(engine.Tf("c.discardWhichCard"), subs...)))
			}
			if len(picks) == 0 {
				return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.careForCassie"), picks...)}}
		},
	})

	// Tech Theft: blanking tech text boxes is not enforceable; the
	// scheme itself works via threat.
	engine.RegisterBehavior("12026", &engine.Behavior{})

	// Moment of Triumph: no after-defeat window exists; approximated to
	// healing 1 on play.
	engine.RegisterBehavior("12030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.HealEntity{Target: e.EOwner(), N: 1}}
		},
	})

	// Lay Down the Law: no change-form response window; action form
	// removes 3 threat.
	engine.RegisterBehavior("12031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				engine.Tf("c.layDownTheLawRemove3ThreatFromWhichScheme"), schemePicks(g, 3, e.EOwner())...)}}
		},
	})

	// Muster Courage: tough to up to X characters (X = villain stage).
	engine.RegisterBehavior("12032", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			x := 1
			for _, v := range g.Villains {
				x = v.Stage
			}
			x = min(3, x)
			var picks []engine.Choice
			for _, q := range g.Players {
				picks = append(picks, engine.Choice{Label: engine.S(q.Name), Kind: engine.ChoiceTarget, SourceID: q.ID}.
					Msgs(engine.ToughEntity{Target: q.ID}))
				for _, id := range q.Allies {
					if a := g.Allies[id]; a != nil {
						picks = append(picks, engine.Choice{Label: engine.S(a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
							Msgs(engine.ToughEntity{Target: a.ID}))
					}
				}
			}
			if len(picks) == 0 {
				return nil
			}
			q := engine.AskN(engine.Tf("c.musterCourageGiveToughToWhichCharacters"), x, picks...)
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: q}}
		},
	})

	// Assess the Situation: +1 hand size until end of phase.
	engine.RegisterBehavior("12033", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.TempHandSizeMsg{Player: e.EOwner(), N: 1}}
		},
	})
}

func registerNemesis() {
	// Yellowjacket: form-dependent stats (giant → retaliate 1 approximated
	// to nothing; tiny → +1 ATK).
	engine.RegisterBehavior("12027", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			if _, ok := msg.(engine.ChangeForm); !ok {
				return nil
			}
			for _, p := range g.Players {
				switch InForm(p) {
				case "tiny":
					mn.AttackVal = baseMnATK(mn) + 1
				default:
					mn.AttackVal = baseMnATK(mn)
				}
			}
			return nil
		},
	})

	// Size Increase: 3 counters; one is removed after the attached enemy
	// activates; discarded when empty.
	engine.RegisterBehavior("12028", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Minions {
				if g.Minions[id].Code[:5] == "12027" {
					t.Target = id
					t.Counters = 3
					return nil
				}
			}
			for id := range g.Villains {
				t.Target = id
				t.Counters = 3
				return nil
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			a := g.Attachments[e.EID()]
			if a == nil {
				return nil
			}
			var hit bool
			switch m := msg.(type) {
			case engine.VillainActivates:
				hit = m.VillainID == a.Target
			case engine.MinionActivates:
				hit = m.MinionID == a.Target
			}
			if !hit {
				return nil
			}
			a.Counters--
			if a.Counters <= 0 {
				g.Delete(a.ID)
				g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
				g.TLogf("c.sizeIncreaseIsDiscarded")
			}
			return nil
		},
	})

	// Yellowjacket's Plan: discard until a nemesis-set card, reveal it.
	engine.RegisterBehavior("12029", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for len(g.EncounterDeck) > 0 {
				c, ok := g.DrawEncounter()
				if !ok {
					return nil
				}
				base := c.Code[:5]
				if base == "12027" || base == "12028" || base == "12029" {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return nil
		},
	})
}

// ---- helpers ----

func schemePicks(g *engine.Game, n int, pid engine.PlayerID) []engine.Choice {
	return cardutil.SchemeChoices(g, func(s engine.EntityID) []engine.Message {
		return []engine.Message{engine.ThwartScheme{Scheme: s, N: n, Source: pid}}
	})
}

func stunChoice(prompt string) func(g *engine.Game, self engine.EntityID) []engine.Message {
	return func(g *engine.Game, self engine.EntityID) []engine.Message {
		var picks []engine.Choice
		for _, id := range cardutil.SortedEnemyIDs(g) {
			enemy := g.Entity(id)
			if enemy != nil {
				picks = append(picks, engine.Choice{Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
					Msgs(engine.StunEntity{Target: id}))
			}
		}
		if len(picks) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: g.Entity(self).EOwner(), Question: engine.Ask(engine.S(prompt), picks...)}}
	}
}

func confuseChoice(prompt string) func(g *engine.Game, self engine.EntityID) []engine.Message {
	return func(g *engine.Game, self engine.EntityID) []engine.Message {
		var picks []engine.Choice
		for _, id := range cardutil.SortedEnemyIDs(g) {
			enemy := g.Entity(id)
			if enemy != nil {
				picks = append(picks, engine.Choice{Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
					Msgs(engine.ConfuseEntity{Target: id}))
			}
		}
		if len(picks) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: g.Entity(self).EOwner(), Question: engine.Ask(engine.S(prompt), picks...)}}
	}
}

func attachToAllyPick(name string) func(g *engine.Game, e engine.Entity) []engine.Message {
	return func(g *engine.Game, e engine.Entity) []engine.Message {
		return []engine.Message{attachPickMsg(g, e, name, 0)}
	}
}

// attachPickMsg builds one attach-target question whose leaves carry the
// AttachUpgrade message; the composite is delivered via a carrier
// question attached to the upgrade itself.
func attachPickMsg(g *engine.Game, e engine.Entity, name string, hp int) engine.Message {
	p := g.Player(e.EOwner())
	if p == nil {
		return nil
	}
	var picks []engine.Choice
	for _, id := range p.Allies {
		a := g.Allies[id]
		if a == nil {
			continue
		}
		if name == "Power Gloves" && !hasTrait(a, "avenger") {
			continue
		}
		picks = append(picks, engine.Choice{Label: engine.S(a.EDef().Name), Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
			Msgs(engine.AttachUpgrade{ID: e.EID(), Target: a.ID, MaxHP: hp}))
	}
	if len(picks) == 0 {
		return nil
	}
	return engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.S("Attach "+name+" to which ally?"), picks...)}
}

func hasTrait(e engine.Entity, want string) bool {
	for _, t := range e.EDef().Traits {
		if strings.EqualFold(t, want) {
			return true
		}
	}
	if a, ok := e.(*engine.Ally); ok {
		for _, t := range a.ExtraTraits {
			if strings.EqualFold(t, want) {
				return true
			}
		}
	}
	return false
}

func baseMnATK(mn *engine.Minion) int {
	if d := mn.EDef().Attack; d != nil {
		return *d
	}
	return mn.AttackVal
}
