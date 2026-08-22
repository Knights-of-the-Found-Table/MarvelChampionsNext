// Package wasp registers the Wasp hero pack: the Giant/Tiny form riders
// (reusing the ant package's form traits) and the Beetle nemesis set.
package wasp

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerWasp()
	registerNemesis()
}

// form reports the player's Giant/Tiny form (shared traits with the ant
// package's SetAntForm message).
func form(p *engine.Player) string {
	for _, t := range p.ExtraTraits {
		if t == "giant" || t == "tiny" {
			return t
		}
	}
	return ""
}

func registerWasp() {
	// Wasp identity: basic-power splitting is approximated away (single
	// targets); the form-flip flow is shared with Ant-Man via SetAntForm.
	engine.RegisterBehavior("13001", &engine.Behavior{
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
			// Reactions run pre-flip: alter-ego → hero asks the form.
			if !p.IsHero() {
				giant := engine.Choice{ID: "giant", Label: "Giant form", Kind: engine.ChoiceForm}
				tiny := engine.Choice{ID: "tiny", Label: "Tiny form", Kind: engine.ChoiceForm}
				giant = giant.Msgs(engine.SetAntForm{Player: pid, Form: "giant"})
				tiny = tiny.Msgs(engine.SetAntForm{Player: pid, Form: "tiny"})
				return []engine.Message{engine.AskQuestion{Player: pid,
					Question: engine.Ask("Which hero form?", giant, tiny)}}
			}
			return []engine.Message{engine.SetAntForm{Player: pid, Form: ""}}
		},
	})

	// Ant-Man (ally): form-scaled stats (synced on form changes).
	engine.RegisterBehavior("13002", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.SetAntForm); !ok {
				return nil
			}
			a := g.Allies[e.EID()]
			owner := g.Player(e.EOwner())
			if a == nil || owner == nil {
				return nil
			}
			switch form(owner) {
			case "giant":
				a.PermTHW, a.PermATK = 0, 1
			case "tiny":
				a.PermTHW, a.PermATK = 1, 0
			default:
				a.PermTHW, a.PermATK = 0, 0
			}
			return nil
		},
	})

	// Giant Help: 3 threat (4 total split in giant — approximated to one
	// scheme).
	engine.RegisterBehavior("13003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			n := 3
			if p != nil && form(p) == "giant" {
				n = 4
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				"Giant Help: remove threat from which scheme?", schemePicks(g, n, e.EOwner())...)}}
		},
	})

	// Pinpoint Strike: 7 damage (+1 tiny).
	engine.RegisterBehavior("13004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			n := 7
			if p != nil && form(p) == "tiny" {
				n = 8
			}
			return cardutil.ChooseEnemy("Pinpoint Strike: deal damage to which enemy?",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(g, e)
		},
	})

	// Rapid Growth: per-use window approximated to a form set + phase
	// bonus.
	engine.RegisterBehavior("13005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{
				engine.SetAntForm{Player: e.EOwner(), Form: "giant"},
				engine.ApplyStatBonus{Target: e.EOwner(), THW: 2, ATK: 2, DEF: 2},
			}
		},
	})

	// Wasp Sting: giant 4 split / tiny 5 single.
	engine.RegisterBehavior("13006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			if form(p) == "tiny" {
				return cardutil.ChooseEnemy("Wasp Sting (tiny): deal 5 damage to which enemy?",
					func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 5, nil })(g, e)
			}
			if form(p) == "giant" {
				return cardutil.ChooseEnemy("Wasp Sting (giant): deal 4 damage to which enemy?",
					func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 4, nil })(g, e)
			}
			return nil
		},
	})

	// Pym Particles: spent-response — plain resource.
	engine.RegisterBehavior("13007", &engine.Behavior{})

	// Red Room Training: giant retaliate 1.
	engine.RegisterBehavior("13008", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			if form(p) == "giant" {
				return engine.StatBonus{Retaliate: 1}
			}
			return engine.StatBonus{}
		},
	})

	// Bio-Synthetic Wings: Aerial; tiny prevents 1.
	engine.RegisterBehavior("13009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "aerial"}}
		},
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			if form(p) == "tiny" && !u.Exhausted {
				g.Push(engine.ExhaustEntity{ID: u.ID})
				return min(1, n), 0
			}
			return 0, 0
		},
	})

	// Wasp's Helmet: giant +1 THW / tiny +1 ATK.
	engine.RegisterBehavior("13010", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			switch form(p) {
			case "giant":
				return engine.StatBonus{THW: 1}
			case "tiny":
				return engine.StatBonus{ATK: 1}
			}
			return engine.StatBonus{}
		},
	})

	// Thor (ally): 2 damage to the villain on entry (3 on physical
	// payment — ally payment not recorded).
	engine.RegisterBehavior("13011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for id := range g.Villains {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: e.EOwner()}}
			}
			return nil
		},
	})

	// Wasp (ally): pym-counter HP scaling — overpay not tracked; base
	// registration.
	engine.RegisterBehavior("13012", &engine.Behavior{})

	// Into the Fray: 6 damage to a minion; excess removes main threat.
	engine.RegisterBehavior("13013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				msgs := []engine.Message{engine.DamageEntity{Target: id, Damage: 6, Source: e.EOwner()}}
				if g.MainScheme != nil && mn.HP() < 6 {
					msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 6 - mn.HP(), Source: e.EOwner()})
				}
				picks = append(picks, engine.Choice{Label: cardutil.EnemyLabel(mn), Kind: engine.ChoiceTarget, SourceID: id, CardCode: mn.Code}.
					Msgs(msgs...))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask("Into the Fray: damage which minion?", picks...)}}
		},
	})

	// Surprise Attack: after a form change, 3 damage (response window
	// approximated to play-time when a form is set).
	engine.RegisterBehavior("13014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return cardutil.ChooseEnemy("Surprise Attack: deal 3 damage to which enemy?",
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 3, nil })(g, e)
		},
	})

	// The Power of Aggression reprint + basics + The Power in All of Us.
	engine.RegisterBehavior("13015", &engine.Behavior{})
	engine.RegisterBehavior("13021", &engine.Behavior{})
	engine.RegisterBehavior("13022", &engine.Behavior{})
	engine.RegisterBehavior("13023", &engine.Behavior{})
	engine.RegisterBehavior("13024", &engine.Behavior{})

	// Boot Camp: each ally you control +1 ATK.
	engine.RegisterBehavior("13016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					a.PermATK++
				}
			}
			g.Logf("Boot Camp: each ally gets +1 ATK")
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if pc, ok := msg.(engine.PlayCard); ok {
				if s := g.Supports[e.EID()]; s != nil && pc.Player == s.EOwner() {
					def, okL := engine.DB.Lookup(pc.Card.Code)
					if okL && def.Type == "ally" {
						if a := newestAllyOf(g, pc.Player); a != nil {
							a.PermATK++
						}
					}
				}
			}
			return nil
		},
	})

	// Lie in Wait: discard on minion engagement → 3 damage.
	engine.RegisterBehavior("13017", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil || m.Player != u.Owner {
				return nil
			}
			return []engine.Message{
				engine.DiscardControlled{Player: u.Owner, ID: u.ID},
				engine.DamageEntity{Target: m.MinionID, Damage: 3, Source: u.Owner},
			}
		},
	})

	// Ironheart: draw 1 on entry.
	engine.RegisterBehavior("13018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
		},
	})

	// Spider-Man: choose THW or ATK +2 this phase.
	engine.RegisterBehavior("13019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				"Spider-Man gets +2 to which power this phase?",
				engine.Choice{ID: "thw", Label: "+2 THW", Kind: engine.ChoiceLabel}.
					Msgs(engine.AllyStatBonus{Ally: e.EID(), THW: 2}),
				engine.Choice{ID: "atk", Label: "+2 ATK", Kind: engine.ChoiceLabel}.
					Msgs(engine.AllyStatBonus{Ally: e.EID(), ATK: 2}),
			)}}
		},
	})

	// Swarm Tactics: change form + ready (ChangeFormAgain re-asks the
	// form via the identity React).
	engine.RegisterBehavior("13020", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{
				engine.ChangeFormAgain{Player: e.EOwner()},
				engine.ReadyEntity{ID: e.EOwner()},
			}
		},
	})

	// Quincarrier reprint: alias bkw 08023.
	if b := engine.LookupBehavior("08023"); b != nil {
		engine.RegisterBehavior("13025", b)
	}

	// Red Dreams obligation.
	engine.RegisterBehavior("13026", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var picks []engine.Choice
			if !p.Exhausted {
				picks = append(picks, engine.Choice{ID: "exhaust", Label: "Exhaust Nadia → remove from the game", Kind: engine.ChoiceLabel}.
					Msgs(engine.ExhaustEntity{ID: p.ID},
						engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			picks = append(picks, engine.Choice{ID: "discard", Label: "Discard mental cards + take 1 damage", Kind: engine.ChoiceLabel}.
				Msgs(engine.ObligationResolve{Player: p.ID, Card: card}))
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Red Dreams:", picks...)}}
		},
	})

	// Running Interference: 2 + stage (max 3) main-scheme threat.
	engine.RegisterBehavior("13031", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if g.MainScheme == nil {
				return nil
			}
			n := 2
			for _, v := range g.Villains {
				n += min(3, v.Stage)
			}
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: n, Source: e.EOwner()}}
		},
	})

	// All for One: 3 damage + 1 per exhausted Avenger.
	engine.RegisterBehavior("13032", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var avengers []engine.Choice
			for _, q := range g.Players {
				if !q.Exhausted && g.EntityHasTrait(q.ID, "avenger") {
					avengers = append(avengers, engine.Choice{Label: "Exhaust " + q.Name + " (identity)", Kind: engine.ChoiceTarget, SourceID: q.ID}.
						Msgs(engine.ExhaustEntity{ID: q.ID}))
				}
			}
			for _, q := range g.Players {
				for _, id := range q.Allies {
					a := g.Allies[id]
					if a != nil && !a.Exhausted && a.EDef().HasTrait("avenger") {
						avengers = append(avengers, engine.Choice{Label: "Exhaust " + a.EDef().Name, Kind: engine.ChoiceCard, CardCode: a.Code}.
							Msgs(engine.ExhaustEntity{ID: id}))
					}
				}
			}
			msgs := []engine.Message{}
			if len(avengers) > 0 {
				q := engine.AskN("All for One: exhaust which Avengers?", len(avengers), avengers...)
				msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: q})
			}
			msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(
				"All for One: deal 3 (+1 per exhausted Avenger) to which enemy?",
				cardutil.EnemyChoices(g, 3, p.ID, func(t engine.EntityID) []engine.Message {
					return []engine.Message{engine.AllForOneDamage{Player: p.ID, Target: t}}
				})...)})
			return msgs
		},
	})

	// Perseverance: tough after a form change (approximated to on-play).
	engine.RegisterBehavior("13033", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ToughEntity{Target: e.EOwner()}}
		},
	})

	// Athletic Conditioning reprint: alias drax 19021.
	if b := engine.LookupBehavior("19021"); b != nil {
		engine.RegisterBehavior("13034", b)
	}
}

func registerNemesis() {
	// Mother's Orders: basic attacks cost 1 more — the surcharge is not
	// enforced; threat-based scheme only.
	engine.RegisterBehavior("13027", &engine.Behavior{})

	// Beetle: Guard printed; on defeat, spend physical or shuffle back.
	engine.RegisterBehavior("13028", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if d, ok := msg.(engine.MinionDefeated); ok && d.MinionID == e.EID() {
				if mn := g.Minions[e.EID()]; mn != nil && mn.EngagedWith != "" {
					// Default: shuffle back into the encounter deck.
					code := mn.Code
					g.Delete(e.EID())
					g.EncounterDeck = append(g.EncounterDeck, engine.Card{ID: g.NextCardID(), Code: code})
					g.Logf("Beetle shuffles back into the encounter deck")
				}
			}
			return nil
		},
	})

	// Beetle Armor MK IV: +4 HP to the attached character.
	engine.RegisterBehavior("13029", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, mn := range g.Minions {
				if mn.Code[:5] == "13028" {
					t.Target = mn.ID
					mn.MaxHP += 4
					return nil
				}
			}
			for id := range g.Villains {
				t.Target = id
				g.Villains[id].MaxHP += 4
				return nil
			}
			return nil
		},
	})

	// Beetle Mania: hero — Beetle attacks +1; alter-ego surge.
	engine.RegisterBehavior("13030", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			for _, mn := range g.Minions {
				if mn.Code[:5] == "13028" {
					mn.AttackVal++
					return []engine.Message{engine.AskAttack{Enemy: mn.ID, Player: p.ID}}
				}
			}
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
	})
}

// ---- helpers ----

func schemePicks(g *engine.Game, n int, pid engine.PlayerID) []engine.Choice {
	return cardutil.SchemeChoices(g, func(s engine.EntityID) []engine.Message {
		return []engine.Message{engine.ThwartScheme{Scheme: s, N: n, Source: pid}}
	})
}

// newestAllyOf returns the most recently spawned ally of a player.
func newestAllyOf(g *engine.Game, pid engine.PlayerID) *engine.Ally {
	p := g.Player(pid)
	if p == nil || len(p.Allies) == 0 {
		return nil
	}
	return g.Allies[p.Allies[len(p.Allies)-1]]
}

var _ = fmt.Sprintf
