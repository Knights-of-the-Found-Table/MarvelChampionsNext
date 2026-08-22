// Package gamora registers the Gamora hero pack. "After you play an
// attack/thwart event" responses key off the EventPlayed announcement;
// per-attack interrupts are approximated as noted.
package gamora

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerGamora()
	registerNemesis()
}

// eventKind reports "attack"/"thwart" for an event card code, "" otherwise.
func eventKind(code string) string {
	def, ok := engine.DB.Lookup(code)
	if !ok || def.Type != "event" {
		return ""
	}
	if def.HasTrait("attack") {
		return "attack"
	}
	if def.HasTrait("thwart") {
		return "thwart"
	}
	return ""
}

func registerGamora() {
	// Gamora identity: Finesse / Precision (once per phase each).
	engine.RegisterBehavior("18001", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ep, ok := msg.(engine.EventPlayed)
			if !ok || ep.Player != e.EID() {
				return nil
			}
			switch eventKind(ep.Card.Code) {
			case "attack":
				if g.UsedThisTurn["gamora-finesse"] {
					return nil
				}
				g.UsedThisTurn["gamora-finesse"] = true
				return []engine.Message{engine.AskQuestion{Player: ep.Player, Question: engine.Ask(
					"Finesse: remove 1 threat from which scheme?", schemePicks(g, 1, ep.Player)...)}}
			case "thwart":
				if g.UsedThisTurn["gamora-precision"] {
					return nil
				}
				g.UsedThisTurn["gamora-precision"] = true
				return cardutil.ChooseEnemy("Precision: deal 1 damage to which enemy?",
					func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 1, nil })(
					g, &engine.EventCard{Code: ep.Card.Code, Owner: ep.Player})
			}
			return nil
		},
	})

	// Nebula (ally): search deck for an attack/thwart event.
	engine.RegisterBehavior("18002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			seen := map[string]bool{}
			for _, c := range p.Deck {
				k := eventKind(c.Code)
				if (k == "attack" || k == "thwart") && !seen[c.Code] {
					seen[c.Code] = true
					picks = append(picks, engine.Choice{Label: c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code}.
						Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID}, engine.ShufflePlayerDeck{Player: p.ID}))
				}
			}
			if len(picks) == 0 {
				return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Nebula: add which event to hand?", picks...)}}
		},
	})

	// Acrobatic Move: 2 damage.
	engine.RegisterBehavior("18003", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Acrobatic Move: deal 2 damage to which enemy?",
			func(g *engine.Game, e engine.Entity) (int, []engine.Message) { return 2, nil }),
	})

	// Crosscounter: prevent 3 + 1 damage + 1 threat.
	engine.RegisterBehavior("18004", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			extra := []engine.Message{engine.DamageEntity{Target: against, Damage: 1, Source: p.ID}}
			extra = append(extra, engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				"Crosscounter: remove 1 threat from which scheme?", schemePicks(g, 1, p.ID)...)})
			return engine.Defends{Defender: p.ID, Against: against, ExtraPrevent: 3}, extra, true
		},
	})

	// Set the Pace: 1 threat.
	engine.RegisterBehavior("18005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				"Set the Pace: remove 1 threat from which scheme?", schemePicks(g, 1, e.EOwner())...)}}
		},
	})

	// Decisive Blow: 4 (7 after a thwart event this turn — tracked via
	// UsedThisTurn set by Finesse-style marker below).
	engine.RegisterBehavior("18006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 4
			if g.UsedThisTurn["gamora-played-thwart"] {
				n = 7
			}
			g.UsedThisTurn["gamora-played-attack"] = true
			return cardutil.ChooseEnemy("Decisive Blow: deal damage to which enemy?",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(g, e)
		},
	})

	// Forward Momentum: 3 (5 after an attack event this turn).
	engine.RegisterBehavior("18007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 3
			if g.UsedThisTurn["gamora-played-attack"] {
				n = 5
			}
			g.UsedThisTurn["gamora-played-thwart"] = true
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				"Forward Momentum: remove threat from which scheme?", schemePicks(g, n, e.EOwner())...)}}
		},
	})

	// Conditioning Room: bottommost attack/thwart event from discard +
	// heal Gamora.
	engine.RegisterBehavior("18008", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Exhaust Conditioning Room → bottommost attack/thwart event + heal 1", Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					msgs := []engine.Message{engine.HealEntity{Target: p.ID, N: 1}}
					for i := len(p.Discard) - 1; i >= 0; i-- {
						if k := eventKind(p.Discard[i].Code); k == "attack" || k == "thwart" {
							c := p.Discard[i]
							p.Discard = append(p.Discard[:i], p.Discard[i+1:]...)
							p.Hand = append(p.Hand, c)
							g.Logf("Conditioning Room returns %s", c.Def().Name)
							break
						}
					}
					return msgs
				},
			}}
		},
	})

	// Keen Instincts: wild resource for attack/thwart events (gate
	// approximated away).
	engine.RegisterBehavior("18009", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})

	// Gamora's Sword: after an attack event, 1 damage.
	engine.RegisterBehavior("18010", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ep, ok := msg.(engine.EventPlayed)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || ep.Player != u.Owner || eventKind(ep.Card.Code) != "attack" {
				return nil
			}
			return cardutil.ChooseEnemy("Gamora's Sword: deal 1 damage to which enemy?",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 1, nil })(g, e)
		},
	})

	// Angela: search top 10 encounter cards for a minion engaged with
	// you; discard Angela otherwise.
	engine.RegisterBehavior("18011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var found *engine.Card
			n := min(10, len(g.EncounterDeck))
			for i := 0; i < n; i++ {
				c := g.EncounterDeck[i]
				if c.Def().Type == "minion" {
					found = &c
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					break
				}
			}
			g.ShuffleEncounterDeck()
			if found == nil {
				return []engine.Message{engine.DiscardControlled{Player: e.EOwner(), ID: e.EID()}}
			}
			return []engine.Message{engine.RevealEncounterCard{Player: e.EOwner(), Card: *found}}
		},
	})

	// Clobber: 3 damage; returns if first card played this round.
	engine.RegisterBehavior("18012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := cardutil.ChooseEnemy("Clobber: deal 3 damage to which enemy?",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 3, nil })(g, e)
			return msgs // the return-to-hand rider needs first-card
			// tracking that the engine does not keep
		},
	})

	// Plan of Attack: search top 4 (7 alter-ego) for an attack event.
	engine.RegisterBehavior("18013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := 4
			if !p.IsHero() {
				n = 7
			}
			var picks []engine.Choice
			for i := 0; i < n && i < len(p.Deck); i++ {
				c := p.Deck[i]
				if eventKind(c.Code) == "attack" {
					picks = append(picks, engine.Choice{Label: c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code}.
						Msgs(engine.TakeDeckCard{Player: p.ID, CardID: c.ID}, engine.ShufflePlayerDeck{Player: p.ID}))
				}
			}
			if len(picks) == 0 {
				return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Plan of Attack: add which event to hand?", picks...)}}
		},
	})

	// Uppercut: 5 damage.
	engine.RegisterBehavior("18014", &engine.Behavior{
		OnPlay: cardutil.ChooseEnemy("Uppercut: deal 5 damage to which enemy?",
			func(g *engine.Game, e engine.Entity) (int, []engine.Message) { return 5, nil }),
	})

	// First Hit: 2 damage to the villain.
	engine.RegisterBehavior("18015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for id := range g.Villains {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: e.EOwner()}}
			}
			return nil
		},
	})

	// Impede: 3 threat from the main scheme.
	engine.RegisterBehavior("18016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 3, Source: e.EOwner()}}
		},
	})

	// Combat Training reprint (alias core 01057).
	if b := engine.LookupBehavior("01057"); b != nil {
		engine.RegisterBehavior("18017", b)
	}

	// Godslayer: +2 ATK until end of phase against unique enemies
	// (per-attack window approximated).
	engine.RegisterBehavior("18018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ApplyStatBonus{Target: e.EOwner(), ATK: 2}}
		},
	})

	// Drax (ally): cannot attack minions — enemy choices for his attack
	// are not filterable per-ally today; registered with the note.
	engine.RegisterBehavior("18019", &engine.Behavior{})

	// Hit and Run: 2 damage + 2 threat.
	engine.RegisterBehavior("18020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := cardutil.ChooseEnemy("Hit and Run: deal 2 damage to which enemy?",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 2, nil })(g, e)
			msgs = append(msgs, engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				"Hit and Run: remove 2 threat from which scheme?", schemePicks(g, 2, e.EOwner())...)})
			return msgs
		},
	})

	// Basic resources.
	engine.RegisterBehavior("18021", &engine.Behavior{})
	engine.RegisterBehavior("18022", &engine.Behavior{})
	engine.RegisterBehavior("18023", &engine.Behavior{})

	// Unfulfilled Destiny obligation.
	engine.RegisterBehavior("18024", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var picks []engine.Choice
			if !p.Exhausted {
				picks = append(picks, engine.Choice{ID: "exhaust", Label: "Exhaust your alter-ego → remove from the game", Kind: engine.ChoiceLabel}.
					Msgs(engine.ExhaustEntity{ID: p.ID},
						engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			n := 0
			for _, c := range p.Hand {
				if eventKind(c.Code) != "" {
					n++
				}
			}
			if n >= 2 {
				var subs []engine.Choice
				for _, c := range p.Hand {
					if eventKind(c.Code) != "" {
						subs = append(subs, engine.Choice{Label: "Discard " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code}.
							Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}}))
					}
				}
				picks = append(picks, engine.Choice{ID: "discard", Label: "Discard 2 events", Kind: engine.ChoiceCard}.
					WithThen(engine.AskN("Discard which 2 events?", 2, subs...)))
			}
			if len(picks) == 0 {
				return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Unfulfilled Destiny:", picks...)}}
		},
	})

	// Pivotal Moment: 2 (5 with no main-scheme threat).
	engine.RegisterBehavior("18029", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 2
			if g.MainScheme != nil && g.MainScheme.Threat == 0 {
				n = 5
			}
			for id := range g.Villains {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: n, Source: e.EOwner()}}
			}
			return nil
		},
	})

	// Comms Implant: attach to a guardian ally (+1 THW, +1 HP).
	engine.RegisterBehavior("18030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a != nil && a.EDef().HasTrait("guardian") {
					picks = append(picks, engine.Choice{Label: a.EDef().Name, Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
						Msgs(engine.AttachUpgrade{ID: e.EID(), Target: a.ID, MaxHP: 1, THW: 1}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Attach Comms Implant to which guardian ally?", picks...)}}
		},
	})

	// True Grit: after defending, remove THW threat.
	engine.RegisterBehavior("18031", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, ec *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			thw := max(0, p.ThwartStat(g))
			if thw <= 0 {
				return engine.Defends{Defender: p.ID, Against: against}, nil, true
			}
			return engine.Defends{Defender: p.ID, Against: against},
				[]engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
					"True Grit: remove threat from which scheme?", schemePicks(g, thw, p.ID)...)}}, true
		},
	})

	// Enhanced Reflexes: alias the thor counters resource.
	if b := engine.LookupBehavior("06034"); b != nil {
		engine.RegisterBehavior("18032", b)
	}
}

func registerNemesis() {
	// Sibling Rivalry: Gamora-only threat removal is not enforced; the
	// villain-phase facedown card to Gamora fires each round.
	engine.RegisterBehavior("18025", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.BeginPhase); !ok {
				return nil
			}
			s := g.SideSchemes[e.EID()]
			if s == nil {
				return nil
			}
			for _, p := range g.Players {
				if p.HeroCode[:5] == "18001" && !p.KOed {
					return []engine.Message{engine.DealEncounterToPlayer{Player: p.ID}}
				}
			}
			return nil
		},
	})

	// Nebula (minion): Retaliate 2 printed; the ally-eviction interrupt
	// is approximated to a reveal check.
	engine.RegisterBehavior("18026", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			for _, p := range g.Players {
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && a.Code == "18002" {
						return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: id}}
					}
				}
			}
			return nil
		},
	})

	// In a Bind: blanking not enforced; removal via attack-event discard
	// + 1 damage.
	engine.RegisterBehavior("18027", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, p := range g.Players {
				if p.HeroCode[:5] == "18001" {
					t.Target = p.ID
					break
				}
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Discard an attack event + take 1 damage → discard In a Bind", Type: engine.AbilityAction,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Attachments[self]
					if a == nil {
						return nil
					}
					pid := a.Target
					hand := engine.CardList{}
					if q := g.Player(pid); q != nil {
						hand = q.Hand
					}
					var subs []engine.Choice
					for _, c := range hand {
						if eventKind(c.Code) == "attack" {
							subs = append(subs, engine.Choice{Label: "Discard " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code}.
								Msgs(engine.DiscardCards{Player: pid, Cards: engine.CardList{c}}))
						}
					}
					if len(subs) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: pid, Question: engine.Ask(
						"In a Bind: discard which attack event?", subs...)}}
				},
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			// Damage rider on the removal answer.
			a := g.Attachments[e.EID()]
			if a == nil {
				return nil
			}
			if dc, ok := msg.(engine.DiscardCards); ok && dc.Player == a.Target {
				g.Delete(a.ID)
				g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
				return []engine.Message{engine.DamageEntity{Target: a.Target, Damage: 1, Source: e.EID()}}
			}
			return nil
		},
	})

	// Waylay: stun + confuse Gamora; surge if either was already set.
	engine.RegisterBehavior("18028", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return []engine.Message{engine.StunEntity{Target: p.ID}, engine.ConfuseEntity{Target: p.ID}}
		},
	})
}

// ---- helpers ----

func schemePicks(g *engine.Game, n int, pid engine.PlayerID) []engine.Choice {
	return cardutil.SchemeChoices(g, func(s engine.EntityID) []engine.Message {
		return []engine.Message{engine.ThwartScheme{Scheme: s, N: n, Source: pid}}
	})
}
