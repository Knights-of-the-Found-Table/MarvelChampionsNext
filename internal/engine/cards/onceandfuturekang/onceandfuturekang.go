// Package onceandfuturekang registers "The Once and Future Kang"
// scenario. Officially each player fights their own Kang variant in a
// separate game area; this implementation approximates with a single
// shared area: stage 1 Kang (The Conqueror), stage 2 all four variants
// at once, stage 3 Kang (The Conqueror, final). Other approximations are
// noted inline.
package onceandfuturekang

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// Variant base codes (stage-2 Kangs) and their realm scheme codes.
var kangVariants = []string{"11002", "11003", "11004", "11005"}

// Duplicate printing ranges in the data (same villains, later codes).
var conquerorBases = []string{"11001", "11034", "11039"} // I / reprint / III
var variantPrints = map[string][]string{
	"11002": {"11035"},
	"11003": {"11036"},
	"11004": {"11037"},
	"11005": {"11038"},
}

func init() {
	registerScenario()
	registerKangs()
	registerMinions()
	registerSchemes()
	registerTreacheries()
	registerObligations()
	registerAttachments()
	registerConquerorFinal()
}

func registerScenario() {
	engine.RegisterScenario(&engine.ScenarioDef{
		ID:   "11007",
		Name: "The Once and Future Kang — Kang's Arrival",
		VillainBases: []string{
			"11001", // stage 1: Kang (The Conqueror)
		},
		MainSchemeStages: []string{"11007b", "11008b", "11013b"},
		ExtraSets:        []string{"temporal", "anachronauts", "mot", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			// Kang's Dominion only enters at stage 3; strip it from the
			// gathered encounter deck.
			var kept engine.CardList
			for _, c := range g.EncounterDeck {
				if c.Code[:5] != "11023" {
					kept = append(kept, c)
				}
			}
			g.EncounterDeck = kept
			return nil
		},
		OnMainSchemeMaxed: func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			switch s.Stage {
			case 1:
				// The Master of Time: discard side schemes, 1 acceleration
				// token each, the four variants arrive (single-area
				// approximation of per-player game areas), each player
				// reveals an encounter card.
				msgs := []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
				n := len(g.SideSchemes)
				for id := range g.SideSchemes {
					delete(g.SideSchemes, id)
				}
				if n > 0 {
					msgs = append(msgs, engine.AddAccelerationToken{Scheme: s.ID})
					g.Logf("%d side scheme(s) discarded; an acceleration token is added", n)
				}
				if kid := findKangByBase(g, "11001"); kid != "" {
					delete(g.Villains, kid) // Kang I leaves
				}
				for _, base := range kangVariants {
					spawnKang(g, base)
				}
				for _, p := range g.Players {
					msgs = append(msgs, engine.RevealNextEncounter{Player: p.ID})
				}
				return msgs
			case 2:
				// Kang's Wrath: the final Conqueror arrives with a
				// Kang's Dominion side scheme.
				msgs := []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
				spawnKang(g, "11006")
				msgs = append(msgs, spawnSchemeMsg(g, "11023")...)
				return msgs
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: "Kang conquered all of time"}}
		},
		OnVillainDefeated: func(g *engine.Game, v *engine.Villain) []engine.Message {
			base := v.Code[:5]
			delete(g.Villains, v.ID)
			g.Logf("%s is removed from the game", v.EDef().Name)
			switch base {
			case "11001", "11034":
				// Kang I defeated: the scheme advances to stage 2.
				if g.MainScheme != nil && g.MainScheme.Stage == 1 {
					return []engine.Message{engine.MainSchemeMaxed{Scheme: g.MainScheme.ID}}
				}
			case "11006", "11039":
				return []engine.Message{engine.GameOver{Won: true, Reason: "Kang was defeated across all of time"}}
			default:
				// A variant fell. When every variant is gone, the scheme
				// advances to stage 3.
				for _, other := range kangVariants {
					if findKangByBase(g, other) != "" {
						return nil
					}
				}
				if g.MainScheme != nil && g.MainScheme.Stage == 2 {
					return []engine.Message{engine.MainSchemeMaxed{Scheme: g.MainScheme.ID}}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("11007", &engine.Behavior{})
	engine.RegisterBehavior("11008", &engine.Behavior{})
	engine.RegisterBehavior("11013", &engine.Behavior{})
	// Per-player realm faces: unused in the single-area flow; markers.
	for _, code := range []string{"11009", "11010", "11011", "11012"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
}

// spawnKang constructs a Kang variant villain in place (stageCodes nil —
// defeat is intercepted by the scenario override before stage advance).
func spawnKang(g *engine.Game, base string) *engine.Villain {
	def, ok := engine.DB.Lookup(base + "b")
	if !ok {
		def, _ = engine.DB.Lookup(base)
	}
	v := &engine.Villain{
		ID:        g.NextEntityID("villain"),
		Code:      def.Code,
		Stage:     1,
		MaxHP:     deref(def.HP, 14),
		SchemeVal: deref(def.Scheme, 1),
		AttackVal: deref(def.Attack, 2),
		Tough:     def.HasKeyword("Toughness"),
	}
	g.Villains[v.ID] = v
	g.Logf("%s enters play", def.Name)
	return v
}

func findKangByBase(g *engine.Game, base string) engine.EntityID {
	for id, v := range g.Villains {
		if v.Code[:5] == base {
			return id
		}
	}
	return ""
}

func spawnSchemeMsg(g *engine.Game, code string) []engine.Message {
	def, ok := engine.DB.Lookup(code)
	if !ok {
		return nil
	}
	s := &engine.SideScheme{
		ID:        g.NextEntityID("side_scheme"),
		Code:      def.Code,
		Threat:    deref(def.BaseThreat, 2) + len(g.Players) - 1,
		MaxThreat: deref(def.Threat, 6),
	}
	g.SideSchemes[s.ID] = s
	g.Logf("%s enters play (threat %d)", def.Name, s.Threat)
	return nil
}

// registerKangs installs the villain behaviors (both printing ranges).
func registerKangs() {
	// Kang (The Conqueror) I and III: on attack, +1 threat or +2 ATK.
	for _, base := range conquerorBases {
		engine.RegisterBehavior(base, &engine.Behavior{
			VillainActivate:   conquerorActivate(),
			VillainDamageable: kangDamageable,
		})
	}
	// Immortus: no damage while a minion is in play.
	for _, base := range append([]string{"11002"}, variantPrints["11002"]...) {
		engine.RegisterBehavior(base, &engine.Behavior{
			VillainDamageable: func(g *engine.Game, v *engine.Villain, damage int) bool {
				if len(g.Minions) > 0 {
					g.Logf("%s cannot take damage while a minion is in play", v.EDef().Name)
					return false
				}
				return kangDamageable(g, v, damage)
			},
		})
	}
	// Iron Lad: Retaliate 1 printed.
	for _, base := range append([]string{"11003"}, variantPrints["11003"]...) {
		engine.RegisterBehavior(base, &engine.Behavior{VillainDamageable: kangDamageable})
	}
	// Rama-Tut: +1 ATK per obligation in play (obligations resolve
	// immediately in this engine, so the bonus is always zero; behavior
	// kept for the defeat wiring).
	for _, base := range append([]string{"11004"}, variantPrints["11004"]...) {
		engine.RegisterBehavior(base, &engine.Behavior{VillainDamageable: kangDamageable})
	}
	// Scarlet Centurion: piercing (not enforced).
	for _, base := range append([]string{"11005"}, variantPrints["11005"]...) {
		engine.RegisterBehavior(base, &engine.Behavior{VillainDamageable: kangDamageable})
	}
}

// kangDamageable consults Temporal Shield then blocks damage while
// Kang's Dominion is in play.
func kangDamageable(g *engine.Game, v *engine.Villain, damage int) bool {
	if temporalShieldCheck(g, v) {
		return false
	}
	for _, s := range g.SideSchemes {
		if s.Code[:5] == "11023" {
			g.Logf("%s cannot take damage while Kang's Dominion is in play", v.EDef().Name)
			return false
		}
	}
	return true
}

// conquerorActivate asks the attacked player: 1 threat on the main
// scheme, or Kang gets +2 ATK for this attack (via boost count).
func conquerorActivate() func(g *engine.Game, v *engine.Villain, p *engine.Player) []engine.Message {
	return func(g *engine.Game, v *engine.Villain, p *engine.Player) []engine.Message {
		if !p.IsHero() {
			// Scheme as normal.
			if v.Confused {
				v.Confused = false
				g.Logf("%s is confused; scheme canceled", v.EDef().Name)
				return nil
			}
			g.Logf("%s schemes against %s", v.EDef().Name, p.Name)
			g.Push(engine.DealBoost{Enemy: v.ID})
			g.Push(engine.RevealBoost{Enemy: v.ID})
			return []engine.Message{engine.ApplyVillainScheme{VillainID: v.ID, Player: p.ID}}
		}
		if v.Stunned {
			v.Stunned = false
			g.Logf("%s is stunned; attack canceled", v.EDef().Name)
			return nil
		}
		g.Logf("%s attacks %s", v.EDef().Name, p.Name)
		g.Push(engine.DealBoost{Enemy: v.ID})
		g.Push(engine.RevealBoost{Enemy: v.ID})
		g.Push(engine.AskQuestion{Player: p.ID, Question: engine.Ask(
			v.EDef().Name+": place 1 threat on the main scheme or take +2 ATK?",
			engine.Choice{ID: "threat", Label: "Place 1 threat on the main scheme", Kind: engine.ChoiceLabel}.
				Msgs(engine.SchemeThreat{Scheme: mainScheme(g), N: 1, Source: v.ID}),
			engine.Choice{ID: "atk", Label: "Kang gets +2 ATK for this attack", Kind: engine.ChoiceLabel}.
				Msgs(engine.BoostActivation{Enemy: v.ID, N: 2}),
		)})
		g.Push(engine.AskAttack{Enemy: v.ID, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou})
		return nil
	}
}

func mainScheme(g *engine.Game) engine.EntityID {
	if g.MainScheme != nil {
		return g.MainScheme.ID
	}
	return ""
}

// registerMinions installs the Temporal / Anachronauts minions.
func registerMinions() {
	// Macrobots: Guard + Retaliate 1 printed; boost gives Kang tough.
	engine.RegisterBehavior("11017", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			for id := range g.Villains {
				return []engine.Message{engine.ToughEntity{Target: id}}
			}
			return nil
		},
	})

	// Ancient Warrior: Quickstrike printed; boost stuns.
	engine.RegisterBehavior("11030", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.StunEntity{Target: engine.PlayerID(card.Owner)}}
		},
	})

	// Chitauri Soldier: when attacking a player, mill 1 encounter card
	// and indirect damage per boost icon.
	engine.RegisterBehavior("11031", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Source != e.EID() || !d.Target.Is(engine.KindPlayer) {
				return nil
			}
			icons := 0
			if c, ok := g.DrawEncounter(); ok {
				if b := c.Def().Boost; b != nil {
					icons = *b
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			if icons > 0 {
				return []engine.Message{engine.DamageEntity{Target: d.Target, Damage: icons, Source: e.EID()}}
			}
			return nil
		},
	})

	// Tyrannosaurus Rex: Toughness printed; piercing not enforced.
	engine.RegisterBehavior("11032", &engine.Behavior{})

	// Apocryphus: discard an ally or support on reveal; boost exhausts.
	engine.RegisterBehavior("11040", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil || mn.EngagedWith == "" {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					picks = append(picks, engine.Choice{Label: "Discard " + a.EDef().Name, Kind: engine.ChoiceCard, CardCode: a.Code}.
						Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
				}
			}
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil {
					picks = append(picks, engine.Choice{Label: "Discard " + s.EDef().Name, Kind: engine.ChoiceCard, CardCode: s.Code}.
						Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Apocryphus: discard which ally or support?", picks...)}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.ExhaustEntity{ID: engine.PlayerID(card.Owner)}}
		},
	})

	// Deathunt 9000: Toughness printed; villainous boost handling
	// approximated to the tough rider only.
	engine.RegisterBehavior("11041", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.ToughEntity{Target: boostOwner(g, card)}}
		},
	})

	// Sir Raston: Guard + Retaliate 1 printed; boost 1 damage.
	engine.RegisterBehavior("11042", &engine.Behavior{
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: engine.PlayerID(card.Owner), Damage: 1, Source: engine.EntityID("11042")}}
		},
	})

	// Terminatrix: Quickstrike printed; piercing not enforced.
	engine.RegisterBehavior("11043", &engine.Behavior{})

	// Wildrun: random discard on reveal and as boost.
	wildrun := func(g *engine.Game, card engine.Card) []engine.Message {
		p := g.Player(engine.PlayerID(card.Owner))
		if p == nil || len(p.Hand) == 0 {
			return nil
		}
		i := g.Random(len(p.Hand))
		c := p.Hand[i]
		p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
		p.Discard = append(p.Discard, c)
		g.Logf("%s discards %s at random (Wildrun)", p.Name, c.Def().Name)
		return nil
	}
	engine.RegisterBehavior("11044", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if mn := g.Minions[e.EID()]; mn != nil && mn.EngagedWith != "" {
				return wildrun(g, engine.Card{Owner: mn.EngagedWith})
			}
			return nil
		},
		Boost: wildrun,
	})

	// Kang (Master of Time): Toughness + Villainous printed; the
	// per-obligation scaling is zero under instant obligation resolution.
	engine.RegisterBehavior("11047", &engine.Behavior{})

	// Time-Displaced Soldier: Incite (1 threat) handled below; Surge
	// printed; boost deals a facedown encounter card.
	engine.RegisterBehavior("11048", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := []engine.Message{}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()})
			}
			return msgs
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			return []engine.Message{engine.DealEncounterToPlayer{Player: engine.PlayerID(card.Owner)}}
		},
	})
}

func boostOwner(g *engine.Game, card engine.Card) engine.EntityID {
	// Boost riders target the enemy being activated; the first villain
	// is the usual owner, else the first minion.
	for id := range g.Villains {
		return id
	}
	for id := range g.Minions {
		return id
	}
	return ""
}

// registerSchemes installs the Temporal side schemes.
func registerSchemes() {
	// Corrupted Timestream: random discard or 2 threat each.
	engine.RegisterBehavior("11022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var msgs []engine.Message
			for _, p := range g.Players {
				p := p
				if len(p.Hand) > 0 {
					i := g.Random(len(p.Hand))
					c := p.Hand[i]
					p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
					p.Discard = append(p.Discard, c)
					g.Logf("%s discards %s at random", p.Name, c.Def().Name)
				} else {
					msgs = append(msgs, engine.SchemeThreat{Scheme: e.EID(), N: 2, Source: p.ID})
				}
			}
			return msgs
		},
	})

	// Kang's Dominion: the damage lock lives in kangDamageable; on
	// defeat the defeating player draws an encounter card (approximated
	// to the first player).
	engine.RegisterBehavior("11023", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if d, ok := msg.(engine.SchemeDefeated); ok && d.Scheme == e.EID() {
				return []engine.Message{engine.DealEncounterToPlayer{Player: cardutil.FirstPlayerID(g)}}
			}
			return nil
		},
	})

	// Pinned Down: +2 threat per obligation in play (obligations resolve
	// instantly here, so typically zero).
	engine.RegisterBehavior("11024", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			n := 0 // obligations never persist in this engine
			if n > 0 {
				return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: 2 * n, Source: e.EID()}}
			}
			return nil
		},
	})

	// Rampage / Light of Centuries Sphere: on defeat, reveal until a
	// minion enters engaged with the defeating player (approximated to
	// the first player).
	recruitOnDefeat := func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		if d, ok := msg.(engine.SchemeDefeated); ok && d.Scheme == e.EID() {
			for len(g.EncounterDeck) > 0 {
				c, ok := g.DrawEncounter()
				if !ok {
					return nil
				}
				if c.Def().Type == "minion" {
					return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c}}
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
		}
		return nil
	}
	engine.RegisterBehavior("11025", &engine.Behavior{React: recruitOnDefeat})
	engine.RegisterBehavior("11050", &engine.Behavior{React: recruitOnDefeat})

	// Time Portal: shuffles back into the encounter deck on defeat.
	engine.RegisterBehavior("11033", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if d, ok := msg.(engine.SchemeDefeated); ok && d.Scheme == e.EID() {
				if s := g.SideSchemes[e.EID()]; s != nil {
					g.EncounterDeck = append(g.EncounterDeck, engine.Card{ID: g.NextCardID(), Code: s.Code})
					g.Logf("Time Portal shuffles back into the encounter deck")
				}
			}
			return nil
		},
	})

	// The Anachronauts: on defeat, shuffle each Temporal card from the
	// encounter discard back into the encounter deck.
	engine.RegisterBehavior("11045", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if d, ok := msg.(engine.SchemeDefeated); ok && d.Scheme == e.EID() {
				var kept engine.CardList
				for _, c := range g.EncounterDiscard {
					if c.Def().HasTrait("temporal") {
						g.EncounterDeck = append(g.EncounterDeck, c)
					} else {
						kept = append(kept, c)
					}
				}
				g.EncounterDiscard = kept
				g.Logf("Each Temporal card in the encounter discard shuffles back")
			}
			return nil
		},
	})
}

// registerTreacheries installs the treacheries.
func registerTreacheries() {
	// Energy Blast: alter-ego discards an ally/support; hero: Kang
	// attacks.
	engine.RegisterBehavior("11026", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if p.IsHero() {
				if vid := firstVillain(g); vid != "" {
					return []engine.Message{
						engine.DealBoost{Enemy: vid}, engine.RevealBoost{Enemy: vid},
						engine.AskAttack{Enemy: vid, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou},
					}
				}
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					picks = append(picks, engine.Choice{Label: "Discard " + a.EDef().Name, Kind: engine.ChoiceCard, CardCode: a.Code}.
						Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
				}
			}
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil {
					picks = append(picks, engine.Choice{Label: "Discard " + s.EDef().Name, Kind: engine.ChoiceCard, CardCode: s.Code}.
						Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
				}
			}
			if len(picks) == 0 {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Energy Blast: discard which ally or support?", picks...)}}
		},
	})

	// Manipulated Timestream: discard each event from hand.
	engine.RegisterBehavior("11027", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var kept engine.CardList
			n := 0
			for _, c := range p.Hand {
				if c.Def().Type == "event" {
					p.Discard = append(p.Discard, c)
					n++
				} else {
					kept = append(kept, c)
				}
			}
			p.Hand = kept
			if n == 0 {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return nil
		},
	})

	// Time-Travel Tactics: surge printed (engine); 1 indirect damage per
	// obligation (zero under instant resolution) — threat instead.
	engine.RegisterBehavior("11028", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return nil
		},
	})

	// Past Machinations: 1 threat + each player reveals an obligation
	// (approximated: 1 threat per player on the main scheme).
	engine.RegisterBehavior("11029", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1 + len(g.Players), Source: t.ID}}
		},
	})

	// Kang's Chosen: 1 threat + reveal until a Temporal minion.
	engine.RegisterBehavior("11046", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			msgs := []engine.Message{}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID})
			}
			for len(g.EncounterDeck) > 0 {
				c, ok := g.DrawEncounter()
				if !ok {
					break
				}
				if c.Def().Type == "minion" && c.Def().HasTrait("temporal") {
					msgs = append(msgs, engine.RevealEncounterCard{Player: p.ID, Card: c})
					break
				}
				g.EncounterDiscard = append(g.EncounterDiscard, c)
			}
			return msgs
		},
	})

	// Ancient Grudge: Kang (Master of Time) activates (fetch him if
	// needed).
	engine.RegisterBehavior("11051", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var mot engine.EntityID
			for id, mn := range g.Minions {
				if mn.Code[:5] == "11047" {
					mot = id
					break
				}
			}
			if mot == "" {
				// Search deck and discard for Master of Time.
				for i, c := range g.EncounterDeck {
					if c.Code[:5] == "11047" {
						g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
						return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
					}
				}
				for i, c := range g.EncounterDiscard {
					if c.Code[:5] == "11047" {
						g.EncounterDiscard = append(g.EncounterDiscard[:i], g.EncounterDiscard[i+1:]...)
						return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
					}
				}
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return []engine.Message{engine.AskAttack{Enemy: mot, Player: p.ID}}
		},
	})
}

// registerObligations installs the Kang obligations. The engine resolves
// obligations immediately on reveal, so each gets an instant-effect
// approximation of its lingering penalty.
func registerObligations() {
	// Weakened: 1 damage (of the basic-power penalty).
	engine.RegisterBehavior("11018", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			g.Logf("Weakened: %s takes 1 damage", p.Name)
			return []engine.Message{
				engine.DamageEntity{Target: p.ID, Damage: 1, Source: engine.EntityID("11018")},
				engine.ObligationResolve{Player: p.ID, Card: card},
			}
		},
	})

	// Stolen Memories: mill 8.
	engine.RegisterBehavior("11019", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			return []engine.Message{
				engine.MillPlayerDeck{Player: p.ID, N: 8},
				engine.ObligationResolve{Player: p.ID, Card: card},
			}
		},
	})

	// Depowered: the play restriction is not enforced; resolves clean.
	engine.RegisterBehavior("11020", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
	})

	// Time-Travel Hijinks: discard the highest-cost card you control.
	engine.RegisterBehavior("11021", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			best := ""
			bestCost := -1
			for _, id := range p.Supports {
				if s := g.Supports[id]; s != nil {
					if c := deref(s.EDef().Cost, 0); c > bestCost {
						best, bestCost = string(id), c
					}
				}
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					if c := deref(a.EDef().Cost, 0); c > bestCost {
						best, bestCost = string(id), c
					}
				}
			}
			msgs := []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
			if best != "" {
				msgs = append(msgs, engine.DiscardControlled{Player: p.ID, ID: engine.EntityID(best)})
			}
			return msgs
		},
	})

	// Fear of Kang: the attack restriction is not enforced.
	engine.RegisterBehavior("11049", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
	})
}

// registerAttachments installs the Temporal attachments.
func registerAttachments() {
	// Temporal Shield: while attached, the first damage to Kang is
	// prevented and reflects 1 to the attacker, then discards.
	engine.RegisterBehavior("11014", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				t.Target = id
				break
			}
			return nil
		},
	})

	// Future Weapon: overkill/stun riders approximated — discard after
	// the attached villain attacks.
	engine.RegisterBehavior("11015", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for id := range g.Villains {
				t.Target = id
				break
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterEnemyAttacked)
			a := g.Attachments[e.EID()]
			if !ok || a == nil || w.Enemy != a.Target {
				return nil
			}
			g.Delete(a.ID)
			g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
			g.Logf("Future Weapon is discarded after the attack")
			return nil
		},
	})

	// Frozen in Times [sic]: boost attach rider approximated away (boost
	// windows cannot attach cards to identities yet).
	engine.RegisterBehavior("11016", &engine.Behavior{})
}

func temporalShieldCheck(g *engine.Game, v *engine.Villain) bool {
	for id, a := range g.Attachments {
		if a.Code[:5] == "11014" && a.Target == v.ID {
			g.Delete(id)
			g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: a.Code})
			g.Logf("Temporal Shield prevents all damage from this attack and is discarded")
			return true
		}
	}
	return false
}

func firstVillain(g *engine.Game) engine.EntityID {
	for id := range g.Villains {
		return id
	}
	return ""
}

func deref(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

// registerConquerorFinal installs Kang the Conqueror's final stage
// (11006): the extortion choice on each attack (defeat = default win).
func registerConquerorFinal() {
	engine.RegisterBehavior("11006", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.VillainActivates)
			if !ok || m.VillainID != e.EID() {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil || !p.IsHero() {
				return nil
			}
			var opts []engine.Choice
			if g.MainScheme != nil {
				opts = append(opts, engine.Choice{
					ID: "threat", Label: "Place 1 threat on the main scheme", Kind: engine.ChoiceLabel,
				}.Msgs(engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()}))
			}
			opts = append(opts, engine.Choice{
				ID: "atk", Label: "Kang gets +2 ATK for this attack", Kind: engine.ChoiceLabel,
			}.Msgs(engine.BoostActivation{Enemy: e.EID(), N: 2}))
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Kang demands tribute: choose one", opts...)}}
		},
	})
}
