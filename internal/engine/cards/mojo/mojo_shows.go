// mojo_shows.go implements the six genre Show modular sets (crime,
// fantasy, horror, sci-fi, sitcom, western) and the Longshot ally.
package mojo

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerShowEnvironments()
	registerCrimeSet()
	registerFantasySet()
	registerHorrorSet()
	registerSciFiSet()
	registerSitcomSet()
	registerWesternSet()
}

// showReveal returns the shared When Revealed effects: discard other
// Setting environments (+ surge when revealed from the deck, handled by
// the caller passing surge=true).
func showReveal(g *engine.Game, e *engine.Environment) []engine.Message {
	var msgs []engine.Message
	for _, id := range cardutil.SortedIDs(g.Environments) {
		other := g.Environments[id]
		if other != nil && other.ID != e.ID && other.EDef().HasTrait("setting") {
			g.Delete(other.ID)
			g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: other.Code})
			g.TLogf("c.replaces", e, other)
		}
	}
	return msgs
}

// showBehavior registers a Show environment: shared replace-on-reveal and
// any set-specific aura (most auras are approximated away).
func showBehavior(code string, onReveal func(g *engine.Game, e *engine.Environment) []engine.Message) {
	engine.RegisterBehavior(code, &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			env := g.Environments[e.EID()]
			if env == nil {
				return nil
			}
			msgs := showReveal(g, env)
			if onReveal != nil {
				msgs = append(msgs, onReveal(g, env)...)
			}
			return msgs
		},
	})
}

func registerShowEnvironments() {
	// 39035 Dial M for Mojo (crime), 39041 A Game of Mojo's (fantasy),
	// 39047 The Mojo Files (horror), 39053 Mojo Runner (sci-fi),
	// 39060 Mojo in the Middle (sitcom), 39066 Wild Wild Mojo (western):
	// global auras are approximated away; the Setting replacement runs.
	for _, code := range []string{"39035", "39041", "39047", "39053", "39060", "39066"} {
		showBehavior(code, nil)
	}
	// The b-sides reuse the same behavior via base-code lookup.
}

func registerCrimeSet() {
	// 39036 Build the Case: clue counters per defeated side scheme; at 3
	// it discards.
	engine.RegisterBehavior("39036", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.SchemeDefeated); !ok {
				return nil
			}
			return nil // clue counters tracked by the owner's count below
		},
	})

	// 39037/39038/39039: typed-resource thwart actions; the global
	// debuffs (threat lock, -2 ATK, villain immunity) are approximated by
	// the Dragnet villain lock only.
	typedScheme := func(code, icon string) {
		engine.RegisterBehavior(code, &engine.Behavior{
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				return []engine.Ability{{
					Label: engine.S("Spend X [" + icon + "] resources → remove X threat"), Type: engine.AbilityAction,
					HeroOnly: true, Cost: 1, CostIcons: icon + ":1",
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						return []engine.Message{engine.ThwartScheme{Scheme: self, N: 1, Source: g.ActiveTurn}}
					},
				}}
			},
		})
	}
	typedScheme("39037", "mental")
	typedScheme("39038", "energy")
	typedScheme("39039", "physical")

	// 39040 Elementary, My Dear Mojo: move a side scheme's threat to the
	// main scheme (defeating it) or reveal a scheme.
	engine.RegisterBehavior("39040", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var choices []engine.Choice
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				s := g.SideSchemes[id]
				if s == nil || s.PlayerSide || g.MainScheme == nil {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: engine.Tf("c.moveThreatToMain", s), Kind: engine.ChoiceTarget, SourceID: id,
				}.Msgs(engine.SchemeThreat{Scheme: g.MainScheme.ID, N: s.Threat, Source: t.ID},
					engine.ThwartScheme{Scheme: id, N: s.Threat, Source: t.ID}))
			}
			if len(choices) > 0 {
				return []engine.Message{engine.AskQuestion{
					Player:   p.ID,
					Question: engine.Ask(engine.Tf("c.elementaryMyDearMojoChoose"), choices...),
				}}
			}
			for guards := 0; guards < 40; guards++ {
				if len(g.EncounterDeck) == 0 {
					return nil
				}
				top := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if top.Def().Type == "side_scheme" {
					return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: top}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, top)
			}
			return nil
		},
	})
}

func registerFantasySet() {
	// 39042 Dragon: defeated → each player draws 4.
	engine.RegisterBehavior("39042", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionDefeated); !ok {
				return nil
			}
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 4})
			}
			return msgs
		},
	})

	// 39043 Goblin: physical-only damage (approximated); defeated →
	// remove 2 threat.
	engine.RegisterBehavior("39043", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() || g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()}}
		},
	})

	// 39044 Troll: defeated → the defeater returns an ally from discard
	// (first player approximation).
	engine.RegisterBehavior("39044", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, c := range p.Discard {
				if c.Def().Type == "ally" {
					choices = append(choices, engine.Choice{
						Label: engine.Tf("c.putIntoPlay", c), Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(engine.AllyEntersPlayFree{Player: p.ID, Card: c}))
					break
				}
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.trollDefeatedReturnAnAllyToPlay"), choices...),
			}}
		},
	})

	// 39045 Fetch Quest: defeated → each player draws 1 (the free-play
	// search approximated).
	engine.RegisterBehavior("39045", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			var msgs []engine.Message
			for _, p := range g.Players {
				msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
			}
			return msgs
		},
	})

	// 39046 Mana Drain: each player discards a random card (the
	// type-wide discard approximated).
	engine.RegisterBehavior("39046", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			for _, tp := range g.Players {
				if len(tp.Hand) > 0 {
					i := g.Random(len(tp.Hand))
					c := tp.Hand[i]
					tp.Hand = append(tp.Hand[:i], tp.Hand[i+1:]...)
					tp.Discard = append(tp.Discard, c)
				}
			}
			return nil
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil || len(p.Hand) == 0 {
				return nil
			}
			i := g.Random(len(p.Hand))
			c := p.Hand[i]
			p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
			p.Discard = append(p.Discard, c)
			return nil
		},
	})
}

func registerHorrorSet() {
	// 39048 Bandolier of Stakes: attach to your identity for a resource;
	// +1 ATK riders per stake counter (approximated as a flat +1).
	engine.RegisterBehavior("39048", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 3}}
		},
	})

	// 39049 Cultist: after activating, fetch The Kraken and discard
	// himself.
	engine.RegisterBehavior("39049", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionActivates)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			var msgs []engine.Message
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "39050" {
						zone.Remove(c.ID)
						msgs = append(msgs, engine.RevealEncounterCard{Player: m.Player, Card: c})
						break
					}
				}
				if len(msgs) > 0 {
					break
				}
			}
			if mn := g.Minions[e.EID()]; mn != nil {
				g.Delete(mn.ID)
				g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: mn.Code})
			}
			return msgs
		},
	})

	// 39050 The Kraken: after activating, 1 damage to each other
	// character; defeated → friendly characters heal 1.
	engine.RegisterBehavior("39050", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			switch m := msg.(type) {
			case engine.MinionActivates:
				if m.MinionID != e.EID() {
					return nil
				}
				var msgs []engine.Message
				for _, p := range g.Players {
					msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: e.EID()})
				}
				return msgs
			case engine.MinionDefeated:
				if m.MinionID != e.EID() {
					return nil
				}
				var msgs []engine.Message
				for _, p := range g.Players {
					msgs = append(msgs, engine.HealEntity{Target: p.ID, N: 1})
				}
				return msgs
			}
			return nil
		},
	})

	// 39051 Vampire: after attacking and damaging, heal all.
	engine.RegisterBehavior("39051", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AskAttack); !ok {
				return nil
			}
			if mn := g.Minions[e.EID()]; mn != nil {
				mn.Damage = 0
				g.TLogf("c.theVampireFeedsAndHealsFully")
			}
			return nil
		},
	})

	// 39052 Werewolf Pack: reveal — defeat an ally (tucked under it);
	// a would-be defeat discards the tucked ally and heals instead.
	engine.RegisterBehavior("39052", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.MinionDefeated); !ok {
				return nil
			}
			return nil
		},
	})
}

func registerSciFiSet() {
	// 39054 Avalanche 9.0: on engage — exhaust a character (identity
	// approximation).
	engine.RegisterBehavior("39054", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			return []engine.Message{engine.ExhaustEntity{ID: p.ID}, engine.DamageEntity{Target: p.ID, Damage: 1, Source: e.EID()}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			pid := cardutil.FirstPlayerID(g)
			return []engine.Message{engine.ExhaustEntity{ID: pid}}
		},
	})

	// 39055 Blob 3.14: max 2 damage per hit (remainder discarded).
	engine.RegisterBehavior("39055", &engine.Behavior{
		MinionDamageable: func(g *engine.Game, mn *engine.Minion, damage int) bool {
			if damage > 2 {
				g.TLogf("c.blob314IgnoresDamageBeyond2")
				mn.Damage += 2
				return false
			}
			return true
		},
	})

	// 39056 Magneto 2.6: after activating, discard 1 card per magnetic
	// counter on him (counters approximated by TuckedCards length).
	engine.RegisterBehavior("39056", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionActivates)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			mn := g.Minions[e.EID()]
			p := g.Player(m.Player)
			if mn == nil || p == nil {
				return nil
			}
			mn.TuckedCards = append(mn.TuckedCards, engine.Card{ID: g.NextCardID(), Code: "magnet"})
			for i := 0; i < len(mn.TuckedCards) && len(p.Hand) > 0; i++ {
				j := g.Random(len(p.Hand))
				c := p.Hand[j]
				p.Hand = append(p.Hand[:j], p.Hand[j+1:]...)
				p.Discard = append(p.Discard, c)
			}
			return nil
		},
	})

	// 39057 Pyro 4.0: when he attacks, take 2 indirect damage.
	engine.RegisterBehavior("39057", &engine.Behavior{
		MinionActivate: func(g *engine.Game, mn *engine.Minion, p *engine.Player) []engine.Message {
			var msgs []engine.Message
			msgs = append(msgs, engine.IndirectDamage{Player: p.ID, N: 2})
			if p.IsHero() {
				msgs = append(msgs, engine.AskAttack{Enemy: mn.ID, Player: p.ID})
			} else if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: mn.SchemeVal, Source: mn.ID})
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.IndirectDamage{Player: cardutil.FirstPlayerID(g), N: 2}}
		},
	})

	// 39058 Toad 2.0: on engage — tuck one of the player's upgrades;
	// defeated → return the tucked cards.
	engine.RegisterBehavior("39058", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			mn := g.Minions[e.EID()]
			if !ok || m.MinionID != e.EID() || mn == nil {
				return nil
			}
			for _, c := range mn.TuckedCards {
				p := g.Player(c.Owner)
				if p != nil {
					cc := c
					cc.Owner = p.ID
					p.Hand = append(p.Hand, cc)
				}
			}
			mn.TuckedCards = nil
			return nil
		},
	})

	// 39059 ICE-Teroid M: minions gain guard (enforced in
	// guardBlocksVillain); at round end the first player reveals a
	// minion.
	engine.RegisterBehavior("39059", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.EndRound); !ok {
				return nil
			}
			pid := cardutil.FirstPlayerID(g)
			for guards := 0; guards < 40; guards++ {
				if len(g.EncounterDeck) == 0 {
					return nil
				}
				top := g.EncounterDeck[0]
				g.EncounterDeck = g.EncounterDeck[1:]
				if top.Def().Type == "minion" {
					return []engine.Message{engine.RevealEncounterCard{Player: pid, Card: top}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, top)
			}
			return nil
		},
	})
}

func registerSitcomSet() {
	// 39061 Family Matters: supports blanked; alter-go exhaust-all
	// removes it (approximation: exhaust identity).
	engine.RegisterBehavior("39061", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			choices := []engine.Choice{engine.Choice{
				ID: "keep", Label: engine.Tf("c.keepFamilyMattersSupportsBlanked"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.ObligationResolve{Player: p.ID, Card: card})}
			if !p.IsHero() && !p.Exhausted {
				choices = append(choices, engine.Choice{
					ID: "exhaust", Label: engine.Tf("c.exhaustYourIdentityAndSupportsDiscard"), Kind: engine.ChoiceLabel,
				}.Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.familyMattersChoose"), choices...),
			}}
		},
	})

	// 39062 Growing Pains: upgrades cost +2; alter-ego discard one.
	engine.RegisterBehavior("39062", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			choices := []engine.Choice{engine.Choice{
				ID: "keep", Label: engine.Tf("c.keepGrowingPainsUpgradesCost2"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.ObligationResolve{Player: p.ID, Card: card})}
			if !p.IsHero() && len(p.Hand) > 0 {
				c := p.Hand[0]
				choices = append(choices, engine.Choice{
					ID: "discard", Label: engine.Tf("c.discardUpgradeObl", c), Kind: engine.ChoiceLabel,
				}.Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.growingPainsChoose"), choices...),
			}}
		},
	})

	// 39063 The Odd Couple: ally limit −2 (no limit counter; the rider is
	// player-enforced); alter-ego shuffle an ally away.
	engine.RegisterBehavior("39063", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			choices := []engine.Choice{engine.Choice{
				ID: "keep", Label: engine.Tf("c.keepTheOddCouple"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.ObligationResolve{Player: p.ID, Card: card})}
			for _, c := range p.Discard {
				if c.Def().Type == "ally" {
					choices = append(choices, engine.Choice{
						ID: "shuffle", Label: engine.Tf("c.shuffleInObl", c), Kind: engine.ChoiceLabel,
					}.Msgs(engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
					break
				}
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.theOddCoupleChoose"), choices...),
			}}
		},
	})

	// 39064 The One with the Breakup: discard 3 → a player draws 1.
	engine.RegisterBehavior("39064", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			choices := []engine.Choice{engine.Choice{
				ID: "keep", Label: engine.Tf("c.keepTheOneWithTheBreakup"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.ObligationResolve{Player: p.ID, Card: card})}
			if len(p.Hand) >= 3 {
				choices = append(choices, engine.Choice{
					ID: "discard", Label: engine.Tf("c.discard3CardsAPlayerDraws1DiscardThisObligation"), Kind: engine.ChoiceLabel,
				}.Msgs(engine.DiscardCards{Player: p.ID, Cards: handTake(p, 3)},
					engine.DrawCards{Player: p.ID, N: 1},
					engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.theOneWithTheBreakupChoose"), choices...),
			}}
		},
	})

	// 39065 Watch Me Play: alter-go exhaust + confused removal.
	engine.RegisterBehavior("39065", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			choices := []engine.Choice{engine.Choice{
				ID: "keep", Label: engine.Tf("c.keepWatchMePlay"), Kind: engine.ChoiceLabel,
			}.Msgs(engine.ObligationResolve{Player: p.ID, Card: card})}
			if !p.IsHero() && !p.Exhausted && p.Confused {
				choices = append(choices, engine.Choice{
					ID: "remove", Label: engine.Tf("c.exhaustYourIdentityDiscardTheConfusionDiscardThisObligation"), Kind: engine.ChoiceLabel,
				}.Msgs(engine.ExhaustEntity{ID: p.ID}, engine.ClearConfuse{Target: p.ID}, engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.watchMePlayChoose"), choices...),
			}}
		},
	})
}

func registerWesternSet() {
	// 39067 Dead or Alive: attach to the biggest minion (+3/hero HP);
	// defeated → each player recovers a discard card.
	engine.RegisterBehavior("39067", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			var best *engine.Minion
			for _, mn := range g.Minions {
				if mn == nil {
					continue
				}
				if best == nil || mn.MaxHP > best.MaxHP {
					best = mn
				}
			}
			if best == nil {
				g.Delete(t.ID)
				return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
			}
			t.Target = best.ID
			best.Attachments = append(best.Attachments, t.ID)
			best.MaxHP += 3 * len(g.Players)
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			t := g.Attachments[e.EID()]
			if !ok || t == nil || m.MinionID != t.Target {
				return nil
			}
			var msgs []engine.Message
			for _, p := range g.Players {
				if len(p.Discard) > 0 {
					c := p.Discard[len(p.Discard)-1]
					p.Discard = p.Discard[:len(p.Discard)-1]
					c.Owner = p.ID
					p.Hand = append(p.Hand, c)
				}
			}
			g.Delete(t.ID)
			return msgs
		},
	})

	// 39068 Card Shark: after attacking, mill 3 and take indirect damage
	// per distinct resource type.
	engine.RegisterBehavior("39068", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AskAttack)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			types := map[string]bool{}
			for i := 0; i < 3 && len(p.Deck) > 0; i++ {
				top := p.Deck[0]
				p.Deck = p.Deck[1:]
				p.Discard = append(p.Discard, top)
				for _, r := range top.Def().Resources {
					types[r] = true
				}
			}
			if len(types) > 0 {
				return []engine.Message{engine.IndirectDamage{Player: p.ID, N: len(types)}}
			}
			return nil
		},
	})

	// 39069 Gunslinger: quickstrike from data; the engage payoff is
	// approximated away.
	engine.RegisterBehavior("39069", &engine.Behavior{})

	// 39070 A Game of Cards: draw 5, discard 5, lose a support/upgrade
	// per distinct type (approximated to one card).
	engine.RegisterBehavior("39070", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			msgs := []engine.Message{engine.DrawCards{Player: p.ID, N: 5}}
			discard := handTake(p, 5)
			for _, c := range discard {
				p.Discard = append(p.Discard, c)
			}
			if len(discard) == 5 {
				p.Hand = p.Hand[len(discard):]
			}
			return msgs
		},
	})

	// 39071 Longshot: When Revealed — into play under the revealer +
	// surge (the ally reveal path lives in revealEncounterCard).
	engine.RegisterBehavior("39071", &engine.Behavior{})
}
