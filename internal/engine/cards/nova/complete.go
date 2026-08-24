// Package nova registers the Nova hero pack. Payment-icon riders use
// the event's recorded Paid icons; per-use ready windows that don't
// exist are approximated with comments.
package nova

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	// 28022 "Bring the War!": textless side scheme shell.
	engine.RegisterBehavior("28022", &engine.Behavior{})
	registerNova()
	registerNemesis()
}

// paidWith reports whether the event's payment included the icon.
func paidWith(ec *engine.EventCard, icon string) bool {
	for _, ic := range ec.Paid.Icons {
		if ic == icon {
			return true
		}
	}
	return false
}

func registerNova() {
	// Nova identity: after a basic power, ready the Supernova Helmet
	// (approximated: auto-ready on BasicAttack/Thwart/Defends).
	engine.RegisterBehavior("28001", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			// Unleash Nova Force marker (the event itself leaves play, so
			// the identity carries the reaction).
			if g.UsedThisRound["nova-force"] {
				switch msg.(type) {
				case engine.VillainDefeated, engine.MinionDefeated, engine.SchemeDefeated:
					return []engine.Message{
						engine.ReadyEntity{ID: e.EID()},
						engine.DrawCards{Player: p.ID, N: 1},
					}
				}
			}
			switch msg.(type) {
			case engine.BasicAttack, engine.BasicThwart, engine.Defends:
				for _, id := range p.Upgrades {
					if u := g.Upgrades[id]; u != nil && u.Code[:5] == "28009" && u.Exhausted {
						return []engine.Message{engine.ReadyEntity{ID: id}}
					}
				}
			}
			return nil
		},
	})

	// Ms. Marvel (ally): after an event, exhaust + 1 self-damage →
	// return it to hand.
	engine.RegisterBehavior("28002", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ep, ok := msg.(engine.EventPlayed)
			a := g.Allies[e.EID()]
			if !ok || a == nil || a.Exhausted || ep.Player != a.Owner {
				return nil
			}
			// Find the just-played event in the discard pile.
			var target *engine.Card
			for i := len(g.Player(a.Owner).Discard) - 1; i >= 0; i-- {
				if g.Player(a.Owner).Discard[i].ID == ep.Card.ID {
					target = &g.Player(a.Owner).Discard[i]
					break
				}
			}
			if target == nil {
				return nil
			}
			card := *target
			return []engine.Message{engine.AskQuestion{Player: a.Owner, Question: engine.Ask(
				"Exhaust Ms. Marvel + 1 damage → return "+card.Def().Name+" to hand?",
				engine.Choice{ID: "use", Label: "Use Ms. Marvel", Kind: engine.ChoiceAbility, SourceID: e.EID(), CardCode: "28002"}.
					Msgs(engine.ExhaustEntity{ID: e.EID()},
						engine.DamageEntity{Target: e.EID(), Damage: 1, Source: e.EID()},
						engine.ReturnDiscardCard{Player: a.Owner, CardID: card.ID}),
				engine.Choice{ID: "skip", Label: "Skip", Kind: engine.ChoicePass},
			)}}
		},
	})

	// Forcefield Projection: prevent 3 for any friendly character
	// (defense event); wild payment adds 3 damage.
	engine.RegisterBehavior("28003", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			var extra []engine.Message
			if paidWith(ec, "wild") {
				extra = append(extra, engine.DamageEntity{Target: against, Damage: 3, Source: p.ID})
			}
			return engine.Defends{Defender: p.ID, Against: against, ExtraPrevent: 3}, extra, true
		},
	})

	// Lightspeed Flight: remove 3 threat.
	engine.RegisterBehavior("28004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				"Lightspeed Flight: remove 3 threat from which scheme?", schemePicks(g, 3, e.EOwner())...)}}
		},
	})

	// Pot Shot: 4 damage.
	engine.RegisterBehavior("28005", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Pot Shot: deal 4 damage to which enemy?",
			func(g *engine.Game, e engine.Entity) (int, []engine.Message) { return 4, nil }),
	})

	// Unleash Nova Force: ready + draw on defeats until end of round —
	// approximated to a marker consumed by VillainDefeated/
	// SchemeDefeated reactions.
	engine.RegisterBehavior("28006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.UsedThisRound["nova-force"] = true
			g.Logf("Unleash Nova Force active until the end of the round")
			return nil
		},
	})

	// Connection to the Worldmind: resource that ignores hand size —
	// hand-size accounting does not track it; plain resource.
	engine.RegisterBehavior("28007", &engine.Behavior{})

	// Jesse Alexander: shuffle a Worldmind from discard, draw 1.
	engine.RegisterBehavior("28008", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Jesse Alexander → shuffle a Worldmind back, draw 1", Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					msgs := []engine.Message{engine.DrawCards{Player: p.ID, N: 1}}
					for _, c := range p.Discard {
						if c.Code[:5] == "28007" {
							msgs = append(msgs, engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID})
							break
						}
					}
					return msgs
				},
			}}
		},
	})

	// Supernova Helmet: Aerial + wild resource.
	engine.RegisterBehavior("28009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "aerial"}}
		},
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true},
	})

	// The Locust: Aggression event from discard to hand.
	engine.RegisterBehavior("28010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			seen := map[string]bool{}
			for _, c := range p.Discard {
				def := c.Def()
				if def.Type == "event" && def.Aspect == "aggression" && !seen[c.Code] {
					seen[c.Code] = true
					picks = append(picks, engine.Choice{Label: def.Name, Kind: engine.ChoiceCard, CardCode: def.Code}.
						Msgs(engine.ReturnDiscardCard{Player: p.ID, CardID: c.ID}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("The Locust: add which Aggression event to hand?", picks...)}}
		},
	})

	// Chase Them Down reprint: alias core 01052.
	if b := engine.LookupBehavior("01052"); b != nil {
		engine.RegisterBehavior("28011", b)
	}

	// Pitchback: after your hero attacks, 4 damage — response window
	// approximated to an immediate follow-up question.
	engine.RegisterBehavior("28012", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ba, ok := msg.(engine.BasicAttack)
			if !ok || ba.Player != e.EOwner() {
				return nil
			}
			return cardutil.ChooseEnemy("Pitchback: deal 4 damage to which enemy?",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 4, nil })(
				g, &engine.EventCard{Code: "28012", Owner: e.EOwner()})
		},
	})

	// No Quarter: 4 damage + excess-mill rider (approximated: always
	// mill 2, keeping Aggression cards).
	engine.RegisterBehavior("28013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := cardutil.ChooseEnemy("No Quarter: deal 4 damage to which enemy?",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 4, nil })(g, e)
			p := g.Player(e.EOwner())
			if p == nil {
				return msgs
			}
			for i := 0; i < 2 && len(p.Deck) > 0; i++ {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				if c.Def().Aspect == "aggression" {
					p.Hand = append(p.Hand, c)
				} else {
					p.Discard = append(p.Discard, c)
				}
			}
			return msgs
		},
	})

	// One by One: 2 damage; chains on defeat (approximated to a second
	// 2-damage question after the first target is a minion).
	engine.RegisterBehavior("28014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return cardutil.ChooseEnemy("One by One: deal 2 damage to which enemy?",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) {
					if mn := g.Minions[engine.EntityID(tgt.EID().String())]; mn != nil && mn.HP() <= 2 {
						second := cardutil.ChooseEnemy("One by One chains: deal 2 damage to which enemy?",
							func(g *engine.Game, t2 engine.Entity) (int, []engine.Message) { return 2, nil })(g, e)
						return 2, second
					}
					return 2, nil
				})(g, e)
		},
	})

	// The Power of Aggression reprint.
	engine.RegisterBehavior("28015", &engine.Behavior{})

	// Fluid Motion: after an Attack event, exhaust → +1 ATK this phase.
	engine.RegisterBehavior("28016", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ep, ok := msg.(engine.EventPlayed)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || u.Exhausted || ep.Player != u.Owner {
				return nil
			}
			def, okL := engine.DB.Lookup(ep.Card.Code)
			if !okL || def.Type != "event" || !def.HasTrait("attack") {
				return nil
			}
			return []engine.Message{
				engine.ExhaustEntity{ID: e.EID()},
				engine.ApplyStatBonus{Target: u.Owner, ATK: 1},
			}
		},
	})

	// Honed Technique: mental-paid Aggression Attack events deal +cost
	// damage — approximate via the EventDamageBonus on play.
	engine.RegisterBehavior("28017", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ep, ok := msg.(engine.EventPlayed)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || ep.Player != u.Owner {
				return nil
			}
			def, okL := engine.DB.Lookup(ep.Card.Code)
			if !okL || def.Type != "event" || !def.HasTrait("attack") || def.Aspect != "aggression" {
				return nil
			}
			return []engine.Message{engine.SetEventBonus{Player: u.Owner, Damage: deref(def.Cost, 0)}}
		},
	})

	// Moon Girl: draw 1 per mental paid (ally payment not recorded;
	// base 1).
	engine.RegisterBehavior("28018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
		},
	})

	// Everyday Hero: civilian rider not enforced; plain resource.
	engine.RegisterBehavior("28019", &engine.Behavior{})

	// Champions Mobile Bunker: champion identity draws 2, discards 2.
	engine.RegisterBehavior("28020", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Champions Mobile Bunker → a champion draws 2, discards 2", Type: engine.AbilityAction,
				Exhaust: true, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var picks []engine.Choice
					for _, q := range g.Players {
						if g.EntityHasTrait(q.ID, "champion") {
							picks = append(picks, engine.Choice{Label: q.Name, Kind: engine.ChoiceTarget, SourceID: q.ID}.
								Msgs(engine.DrawCards{Player: q.ID, N: 2},
									engine.BunkerDiscard{Player: q.ID}))
						}
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: g.Entity(self).EOwner(),
						Question: engine.Ask("Which champion identity?", picks...)}}
				},
			}}
		},
	})

	// Weight of the World obligation: the helmet-lock is approximated
	// away; exhaust to remove.
	engine.RegisterBehavior("28021", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if !p.Exhausted {
				return []engine.Message{
					engine.ExhaustEntity{ID: p.ID},
					engine.ObligationResolve{Player: p.ID, Card: card, Remove: true},
				}
			}
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
	})

	// Yaw and Roll: after a basic thwart, remove 3 threat.
	engine.RegisterBehavior("28026", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			bt, ok := msg.(engine.BasicThwart)
			if !ok || bt.Player != e.EOwner() {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				"Yaw and Roll: remove 3 threat from which scheme?", schemePicks(g, 3, e.EOwner())...)}}
		},
	})

	// Height Advantage: -1 damage taken while Aerial; discard at turn
	// start.
	engine.RegisterBehavior("28027", &engine.Behavior{
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			if g.EntityHasTrait(p.ID, "aerial") {
				return min(1, n), 0
			}
			return 0, 0
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ts, ok := msg.(engine.PlayerTurnStart)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || ts.Player != u.Owner {
				return nil
			}
			return []engine.Message{engine.DiscardControlled{Player: u.Owner, ID: u.ID}}
		},
	})
}

func registerNemesis() {
	// Armored Assault: tough enemies +3 ATK (synced on status changes).
	engine.RegisterBehavior("28028", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, mn := range g.Minions {
				if mn.Tough {
					mn.AttackVal += 3
				}
			}
			g.Logf("Armored Assault: each tough enemy gets +3 ATK")
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if st, ok := msg.(engine.ToughEntity); ok {
				if mn := g.Minions[st.Target]; mn != nil && g.SideSchemes[e.EID()] != nil {
					mn.AttackVal += 3
				}
			}
			return nil
		},
	})

	// Warbringer: +1 ATK per printed wild in the defender's hand for the
	// attack (approximated: attack-time boost).
	engine.RegisterBehavior("28023", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			aa, ok := msg.(engine.AskAttack)
			mn := g.Minions[e.EID()]
			if !ok || mn == nil || aa.Enemy != e.EID() {
				return nil
			}
			p := g.Player(aa.Player)
			if p == nil {
				return nil
			}
			n := 0
			for _, c := range p.Hand {
				for _, r := range c.Def().Resources {
					if r == "wild" {
						n++
					}
				}
			}
			if n > 0 {
				return []engine.Message{engine.BoostActivation{Enemy: e.EID(), N: n}}
			}
			return nil
		},
	})

	// War Delivery: pay a wild or the villain + Warbringer attack.
	engine.RegisterBehavior("28024", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			for id := range g.Villains {
				msgs = append(msgs, engine.DealBoost{Enemy: id}, engine.RevealBoost{Enemy: id},
					engine.AskAttack{Enemy: id, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou})
				break
			}
			for _, mn := range g.Minions {
				if mn.Code[:5] == "28023" {
					msgs = append(msgs, engine.AskAttack{Enemy: mn.ID, Player: p.ID})
					break
				}
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			n := 0
			if p := g.Player(engine.PlayerID(card.Owner)); p != nil {
				for _, c := range p.Hand {
					for _, r := range c.Def().Resources {
						if r == "wild" {
							n++
						}
					}
				}
			}
			if n > 0 && g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: engine.EntityID("28024")}}
			}
			return nil
		},
	})

	// "The War's Been Brought": surge printed; mill per printed wild.
	engine.RegisterBehavior("28025", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			count := func(cards engine.CardList) int {
				n := 0
				for _, c := range cards {
					for _, r := range c.Def().Resources {
						if r == "wild" {
							n++
						}
					}
				}
				return n
			}
			n := count(p.Hand) + count(p.Discard)
			for i := 0; i < n; i++ {
				if c, ok := g.DrawEncounter(); ok {
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				}
			}
			g.Logf("The War's Been Brought mills %d encounter card(s)", n)
			return nil
		},
	})

	// Armadillo: Toughness printed; regains tough after activating.
	engine.RegisterBehavior("28029", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ma, ok := msg.(engine.MinionActivates)
			if !ok || ma.MinionID != e.EID() {
				return nil
			}
			return []engine.Message{engine.ToughEntity{Target: e.EID()}}
		},
	})

	// Rollin', Rollin': fetch Armadillo if absent; attach. The
	// cannot-defend rider is approximated away.
	engine.RegisterBehavior("28030", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, mn := range g.Minions {
				if mn.Code[:5] == "28029" {
					t.Target = mn.ID
					return nil
				}
			}
			for i, c := range g.EncounterDeck {
				if c.Code[:5] == "28029" {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
				}
			}
			for i, c := range g.EncounterDiscard {
				if c.Code[:5] == "28029" {
					g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
					return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
				}
			}
			return nil
		},
	})

	// Tough and Tumble: tough enemies activate.
	engine.RegisterBehavior("28031", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			fired := false
			for id := range g.Villains {
				if g.Villains[id].Tough {
					fired = true
					if p.IsHero() {
						msgs = append(msgs, engine.DealBoost{Enemy: id}, engine.RevealBoost{Enemy: id},
							engine.AskAttack{Enemy: id, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou})
					} else {
						msgs = append(msgs, engine.ApplyVillainScheme{VillainID: id, Player: p.ID})
					}
				}
			}
			for _, mn := range g.Minions {
				if mn.Tough {
					fired = true
					if p.IsHero() {
						msgs = append(msgs, engine.AskAttack{Enemy: mn.ID, Player: p.ID})
					} else if g.MainScheme != nil {
						msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: mn.SchemeVal, Source: mn.ID})
					}
				}
			}
			if !fired {
				msgs = append(msgs, engine.RevealNextEncounter{Player: p.ID})
			}
			return msgs
		},
	})

	// Tough It Out: tough for Armadillo and the villain (surge at 1 or
	// fewer — approximated: never surges when both exist).
	engine.RegisterBehavior("28032", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := []engine.Message{}
			gave := 0
			for _, mn := range g.Minions {
				if mn.Code[:5] == "28029" && !mn.Tough {
					msgs = append(msgs, engine.ToughEntity{Target: mn.ID})
					gave++
				}
			}
			for id := range g.Villains {
				if !g.Villains[id].Tough {
					msgs = append(msgs, engine.ToughEntity{Target: id})
					gave++
				}
				break
			}
			if gave == 0 {
				msgs = append(msgs, engine.RevealNextEncounter{Player: p.ID})
			}
			return msgs
		},
	})
}

// ---- helpers ----

func schemePicks(g *engine.Game, n int, pid engine.PlayerID) []engine.Choice {
	return cardutil.SchemeChoices(g, func(s engine.EntityID) []engine.Message {
		return []engine.Message{engine.ThwartScheme{Scheme: s, N: n, Source: pid}}
	})
}

func deref(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
