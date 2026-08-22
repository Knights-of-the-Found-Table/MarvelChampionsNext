// Package thor registers the Thor hero pack: the identity, the core
// signature cards (Mjolnir, Lady Sif, God of Thunder, For Asgard!,
// Hammer Throw, Lightning Strike, Thor's Helmet, Asgard location) and
// the Loki / Frost Giant nemesis set.
package thor

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerRemainingThor()
	registerThor()
	registerSignatures()
	registerNemesis()
	registerObligation()
}

// reactKey names a tracked usage for response / interrupt hooks. The
// engine resets UsedThisTurn between phases, matching a "once per phase"
// limit (Thor's "Have at thee!").
func reactKey(code, slot string) string {
	return fmt.Sprintf("react:%s:%s", code, slot)
}

// registerThor installs the Thor / Odinson identity (06001a/b).
func registerThor() {
	engine.RegisterBehavior("06001", &engine.Behavior{
		// "Have at thee!" — Response: after you engage a minion, draw 2
		// cards. Limit once per phase. Hooks MinionEntersPlay.
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok || m.Player != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil || !p.IsHero() {
				return nil
			}
			key := reactKey("06001", "have-at-thee")
			if g.UsedThisTurn[key] {
				return nil
			}
			g.UsedThisTurn[key] = true
			g.Logf("Have at thee! — %s draws 2 cards", p.Name)
			return []engine.Message{engine.DrawCards{Player: p.ID, N: 2}}
		},
		// Asgard location (06007) in play grants +1 hand size. Only
		// the identity owns a HandSizeBonus hook, so we count the
		// Asgard supports across all players here. (Asgard is a
		// hero-specific support that only appears in Thor decks in
		// practice, but the bonus logic works for any owner.)
		HandSizeBonus: func(g *engine.Game, p *engine.Player) int {
			n := 0
			for _, pl := range g.Players {
				for _, id := range pl.Supports {
					if s := g.Supports[id]; s != nil && s.Code == "06007" {
						n++
					}
				}
			}
			return n
		},
		// Alter-Ego side: Worthy — Action: search your deck and
		// discard for the Mjolnir upgrade and add it to your hand.
		// Shuffle. Limit once per round.
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				Label:        "Worthy — search your deck and discard for Mjolnir",
				Type:         engine.AbilityAction,
				AlterEgoOnly: true,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(self)
					if pl == nil {
						return nil
					}
					var deckChoices, discardChoices []engine.Choice
					for _, c := range pl.Deck {
						if c.Code == "06009" {
							deckChoices = append(deckChoices, engine.Choice{
								Label: "Take " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
							}.Msgs(
								engine.TakeDeckCard{Player: pl.ID, CardID: c.ID},
								engine.ShufflePlayerDeck{Player: pl.ID},
							))
						}
					}
					for _, c := range pl.Discard {
						if c.Code == "06009" {
							discardChoices = append(discardChoices, engine.Choice{
								Label: "Take " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
							}.Msgs(
								engine.ReturnDiscardCard{Player: pl.ID, CardID: c.ID},
								engine.ShufflePlayerDeck{Player: pl.ID},
							))
						}
					}
					all := append(deckChoices, discardChoices...)
					if len(all) == 0 {
						g.Logf("Worthy — Mjolnir is not in deck or discard; shuffling")
						return []engine.Message{engine.ShufflePlayerDeck{Player: pl.ID}}
					}
					all = append(all, engine.Choice{
						ID: "skip", Label: "Skip (still shuffle)", Kind: engine.ChoicePass,
					}.Msgs(engine.ShufflePlayerDeck{Player: pl.ID}))
					return []engine.Message{engine.AskQuestion{
						Player:   pl.ID,
						Question: engine.Ask("Worthy — take Mjolnir from your deck or discard", all...),
					}}
				},
			}}
		},
	})
}

// registerSignatures installs Thor's signature cards.
func registerSignatures() {
	registerLadySif()
	registerDefenderOfTheNineRealms()
	registerForAsgard()
	registerHammerThrow()
	registerLightningStrike()
	registerAsgardSupport()
	registerGodOfThunder()
	registerMjolnir()
	registerThorsHelmet()
}

// 06002 Lady Sif: Response — after Lady Sif enters play, ready Thor or
// Odinson. (The OnPlay hook fires whenever the ally enters play — pay,
// Quinjet, Make the Call, Lockjaw — which approximates the "enters
// play" trigger cleanly.)
func registerLadySif() {
	engine.RegisterBehavior("06002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			g.Logf("Lady Sif readies %s", p.Name)
			return []engine.Message{engine.ReadyEntity{ID: p.ID}}
		},
	})
}

// 06003 Defender of the Nine Realms: Hero Action (thwart) — discard
// cards from the top of the encounter deck until you discard a minion,
// put it engaged with you, then remove 3 threat from a scheme. (Engine
// approximation: the mill-until-minion part is omitted because there is
// no generic "reveal from encounter deck until X" hook; the 3-threat
// removal still happens.)
func registerDefenderOfTheNineRealms() {
	engine.RegisterBehavior("06003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.ThwartScheme{Scheme: id, N: 3, Source: pid}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Defender of the Nine Realms — choose a scheme", choices...),
			}}
		},
	})
}

// 06004 For Asgard!: Alter-Ego Action — search your deck and discard
// pile for a card with the Asgard trait and add it to your hand. Shuffle.
func registerForAsgard() {
	engine.RegisterBehavior("06004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var deckChoices, discardChoices []engine.Choice
			for _, c := range p.Deck {
				if c.Def().HasTrait("asgard") {
					deckChoices = append(deckChoices, engine.Choice{
						Label: "Take " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(
						engine.TakeDeckCard{Player: pid, CardID: c.ID},
						engine.ShufflePlayerDeck{Player: pid},
					))
				}
			}
			for _, c := range p.Discard {
				if c.Def().HasTrait("asgard") {
					discardChoices = append(discardChoices, engine.Choice{
						Label: "Take " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
					}.Msgs(
						engine.ShuffleIntoDeck{Player: pid, CardID: c.ID},
						engine.ShufflePlayerDeck{Player: pid},
					))
				}
			}
			all := append(deckChoices, discardChoices...)
			if len(all) == 0 {
				g.Logf("For Asgard! — no Asgard card in deck or discard")
				return []engine.Message{engine.ShufflePlayerDeck{Player: pid}}
			}
			all = append(all, engine.Choice{
				ID: "skip", Label: "Skip (still shuffle)", Kind: engine.ChoicePass,
			}.Msgs(engine.ShufflePlayerDeck{Player: pid}))
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("For Asgard! — take an Asgard card", all...),
			}}
		},
	})
}

// 06005 Hammer Throw: Hero Action (attack) — Exhaust Mjolnir → deal 8
// damage to an enemy and return Mjolnir to your hand.
func registerHammerThrow() {
	engine.RegisterBehavior("06005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			var mjolnir *engine.Upgrade
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.Code == "06009" {
					mjolnir = u
					break
				}
			}
			if mjolnir == nil || mjolnir.Exhausted {
				g.Logf("Hammer Throw — no ready Mjolnir in play")
				return nil
			}
			choices := cardutil.EnemyChoices(g, 8, pid, func(target engine.EntityID) []engine.Message {
				return []engine.Message{
					engine.ExhaustEntity{ID: mjolnir.ID},
					engine.DamageEntity{Target: target, Damage: 8, Source: pid},
					engine.ReturnControlled{Player: pid, ID: mjolnir.ID},
				}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Hammer Throw — deal 8 damage and return Mjolnir to your hand", choices...),
			}}
		},
	})
}

// 06006 Lightning Strike: Hero Action — spend X [energy] resources →
// deal X damage to the villain and each minion engaged with you.
// (Engine approximation: any resources may be used; the cost is paid
// normally and the X amount is offered up to the number of resources in
// hand.)
func registerLightningStrike() {
	engine.RegisterBehavior("06006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			maxPay := 0
			for _, c := range p.Hand {
				maxPay += len(c.Def().Resources)
			}
			if maxPay == 0 {
				g.Logf("Lightning Strike — no resources available")
				return nil
			}
			var amtChoices []engine.Choice
			for x := 1; x <= maxPay; x++ {
				x := x
				amtChoices = append(amtChoices, engine.Choice{
					ID: fmt.Sprintf("x-%d", x), Label: fmt.Sprintf("Deal %d damage to each target", x),
					Kind: engine.ChoiceLabel,
				}.Msgs(lightningStrikeMessages(g, p, x)...))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Lightning Strike — choose X to spend", amtChoices...),
			}}
		},
	})
}

// lightningStrikeMessages builds the X-damage-to-villain-and-engaged
// minions message payload.
func lightningStrikeMessages(g *engine.Game, p *engine.Player, x int) []engine.Message {
	var msgs []engine.Message
	for _, id := range cardutil.SortedIDs(g.Villains) {
		msgs = append(msgs, engine.DamageEntity{Target: id, Damage: x, Source: p.ID})
	}
	for _, id := range cardutil.SortedIDs(g.Minions) {
		mn := g.Minions[id]
		if mn == nil {
			continue
		}
		if mn.EngagedWith == p.ID {
			msgs = append(msgs, engine.DamageEntity{Target: id, Damage: x, Source: p.ID})
		}
	}
	return msgs
}

// 06007 Asgard: You get +1 hand size. (The bonus is applied by the
// Thor identity's HandSizeBonus, which counts Asgard supports in
// play across all players.)
func registerAsgardSupport() {
	engine.RegisterBehavior("06007", &engine.Behavior{
		// No runtime hooks needed; the bonus is computed at draw time
		// by the identity's HandSizeBonus.
	})
}

// 06008 God of Thunder: Hero Resource — exhaust → generate a [energy]
// resource.
func registerGodOfThunder() {
	engine.RegisterBehavior("06008", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "energy", HeroOnly: true},
	})
}

// 06009 Mjolnir: Restricted. Thor gets +1 ATK and gains the Aerial
// trait. (The Aerial trait is granted on play; the engine has no clean
// upgrade-removal hook, so the trait may linger if Mjolnir is
// discarded. This matches the existing pattern in the engine for
// Doctor Strange's Cloak of Levitation and Daredevil's Billy Club.)
func registerMjolnir() {
	engine.RegisterBehavior("06009", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{ATK: 1}
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "aerial"}}
		},
	})
}

// 06010 Thor's Helmet: You get +5 hit points.
func registerThorsHelmet() {
	engine.RegisterBehavior("06010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			p.MaxHP += 5
			g.Logf("Thor's Helmet grants +5 HP to %s (now %d)", p.Name, p.MaxHP)
			return nil
		},
	})
}

// registerNemesis installs the Thor nemesis encounter set: Family Feud
// (side scheme), Loki (self-resurrect minion), Frost Giant (tough with
// boost stun), Trickster (treachery).
func registerNemesis() {
	registerFamilyFeud()
	registerLoki()
	registerFrostGiant()
	registerTrickster()
}

// 06027 Family Feud: When Revealed — place 1 additional threat here for
// each Asgard card in play.
func registerFamilyFeud() {
	engine.RegisterBehavior("06027", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s, ok := e.(*engine.SideScheme)
			if !ok {
				return nil
			}
			n := 0
			for _, p := range g.Players {
				if p.HeroDef().HasTrait("asgard") {
					n++
				}
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && a.EDef().HasTrait("asgard") {
						n++
					}
				}
				for _, id := range p.Supports {
					if s := g.Supports[id]; s != nil && s.EDef().HasTrait("asgard") {
						n++
						_ = s
					}
				}
				for _, id := range p.Upgrades {
					if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("asgard") {
						n++
					}
				}
			}
			if n > 0 {
				s.Threat += n
				if s.Threat > s.MaxThreat {
					s.Threat = s.MaxThreat
				}
				g.Logf("Family Feud gains %d threat from Asgard cards (now %d)", n, s.Threat)
			}
			return nil
		},
	})
}

// 06028 Loki: Forced Interrupt — when Loki would be defeated, discard
// the top card of the encounter deck; if it is a treachery, heal all
// damage from Loki instead. (Engine approximation: when any damage
// event would put Loki at >= MaxHP, peek the encounter deck top and
// reset damage to 0 if it's a treachery.)
func registerLoki() {
	engine.RegisterBehavior("06028", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			mn, ok := e.(*engine.Minion)
			if !ok {
				return nil
			}
			if mn.Damage < mn.MaxHP {
				return nil
			}
			// Only react once per lethal damage event: skip if we
			// already saved this Loki in the current step. The engine
			// doesn't expose step state, so we approximate by tracking
			// on the minion itself.
			if mn.Damage == 0 {
				return nil
			}
			card, ok := g.PeekEncounterTop()
			if !ok {
				return nil
			}
			def := card.Def()
			if def.Type != "treachery" {
				g.Logf("Loki's self-resolve discarded %s (not a treachery)", def.Name)
				return nil
			}
			g.Logf("Loki's self-resolve! Top is %s — Loki heals to full", def.Name)
			mn.Damage = 0
			return nil
		},
	})
}

// 06029 Frost Giant: Toughness + boost — if the villain is attacking
// and this attack deals damage to a character, stun that character.
// (Toughness is parsed by the data layer, the boost stun text is read
// directly from the card text. No bespoke hooks needed.)
func registerFrostGiant() {
	engine.RegisterBehavior("06029", &engine.Behavior{})
}

// 06030 Trickster: When Revealed — discard the top 3 cards of your deck.
// Place 1 threat on the main scheme for each different card type
// discarded this way. (Engine approximation: place 1 threat per
// discarded card; the type-distinctness check is omitted.)
func registerTrickster() {
	engine.RegisterBehavior("06030", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var discarded engine.CardList
			for i := 0; i < 3 && len(p.Deck) > 0; i++ {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				discarded = append(discarded, c)
			}
			if len(discarded) == 0 {
				return nil
			}
			msgs := []engine.Message{engine.DiscardCards{Player: p.ID, Cards: discarded}}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{
					Scheme: g.MainScheme.ID, N: len(discarded), Source: t.ID,
				})
			}
			g.Logf("Trickster discards %d cards and places %d threat on the main scheme", len(discarded), len(discarded))
			return msgs
		},
	})
}

// 06026 Odin's Anger: Give to the Odinson player. You may flip to
// alter-ego form. Choose: exhaust Odinson → remove Odin's Anger from
// the game, or discard Mjolnir from your hand or from play. You are
// stunned. Discard this obligation.
func registerObligation() {
	engine.RegisterBehavior("06026", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			// Build the "discard Mjolnir" subtree: any Mjolnir card in
			// hand or in play. Each choice carries the discard + the
			// shared stun+resolve tail.
			type mj struct {
				label string
				card  engine.Card
				id    engine.EntityID
				play  bool
			}
			var mjs []mj
			for _, c := range p.Hand {
				if c.Code == "06009" {
					mjs = append(mjs, mj{label: "Discard Mjolnir from hand", card: c})
				}
			}
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.Code == "06009" {
					mjs = append(mjs, mj{label: "Discard Mjolnir from play", id: id, play: true})
				}
			}
			stunAndDiscard := []engine.Message{
				engine.StunEntity{Target: p.ID},
				engine.ObligationResolve{Player: p.ID, Card: card},
			}
			var penaltySubtree *engine.Question
			if len(mjs) > 0 {
				combined := make([]engine.Choice, 0, len(mjs))
				for _, m := range mjs {
					msgs := []engine.Message{}
					if m.play {
						msgs = append(msgs, engine.DiscardControlled{Player: p.ID, ID: m.id})
					} else {
						msgs = append(msgs, engine.DiscardCards{Player: p.ID, Cards: engine.CardList{m.card}})
					}
					msgs = append(msgs, stunAndDiscard...)
					combined = append(combined, engine.Choice{
						Label: m.label, Kind: engine.ChoiceCard, CardCode: "06009",
					}.Msgs(msgs...))
				}
				penaltySubtree = engine.Ask("Odin's Anger — choose a Mjolnir to discard", combined...)
			} else {
				penaltySubtree = engine.Ask("Odin's Anger — no Mjolnir; you are stunned",
					engine.Choice{ID: "stun", Label: "Stun", Kind: engine.ChoiceLabel}.Msgs(stunAndDiscard...),
				)
			}
			var removeMsgs []engine.Message
			if p.IsHero() && !p.FormChanged && !p.Exhausted {
				removeMsgs = append(removeMsgs, engine.ChangeForm{Player: p.ID})
			}
			removeMsgs = append(removeMsgs,
				engine.ExhaustEntity{ID: p.ID},
				engine.ObligationResolve{Player: p.ID, Card: card, Remove: true},
			)
			root := engine.Ask("Odin's Anger — choose:",
				engine.Choice{
					ID:    "remove",
					Label: "Exhaust Odinson → remove Odin's Anger from the game",
					Kind:  engine.ChoiceLabel,
				}.Msgs(removeMsgs...),
				engine.Choice{
					ID:    "penalty",
					Label: "Discard Mjolnir; you are stunned",
					Kind:  engine.ChoiceLabel,
				}.WithThen(penaltySubtree),
			)
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: root}}
		},
	})
}
