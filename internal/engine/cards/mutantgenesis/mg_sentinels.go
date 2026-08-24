// mg_sentinels.go implements the Night of the Sentinels scenario content:
// the Sentinel villain (32084–32088), the Project Wideawake set
// (32089–32100), Zero Tolerance (32101–32104) and the Sentinels modular
// (32105–32108).
package mutantgenesis

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerSentinelVillain()
	registerWideawake()
	registerZeroTolerance()
	registerSentinelsModular()
}

// oztScheme finds Operation Zero Tolerance.
func oztScheme(g *engine.Game) *engine.SideScheme {
	for _, s := range g.SideSchemes {
		if s != nil && s.Code == "32104" {
			return s
		}
	}
	return nil
}

// tuckUnderOZT mills a player's top card facedown under Operation Zero
// Tolerance.
func tuckUnderOZT(g *engine.Game, pid engine.PlayerID) {
	s := oztScheme(g)
	p := g.Player(pid)
	if s == nil || p == nil || len(p.Deck) == 0 {
		return
	}
	top := p.Deck[0]
	p.Deck = p.Deck[1:]
	s.StoredCards = append(s.StoredCards, top)
	g.Logf("%s's %s is placed facedown under Operation Zero Tolerance", p.Name, top.Def().Name)
}

// sentinelVillain returns the Sentinel villain (Night of the Sentinels).
func sentinelVillain(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil && engine.BaseCodeOf(v.Code) == "32084" {
			return v
		}
	}
	return nil
}

func registerSentinelVillain() {
	// 32084–32086 Sentinel stages: "When Revealed" — search Abduction
	// Protocols (approximation: revealed once via the scenario setup and
	// on each stage advance); stages II+ deal each other player a facedown
	// encounter card.
	for i, code := range []string{"32084", "32085", "32086"} {
		stage := i + 1
		engine.RegisterBehavior(code, &engine.Behavior{
			VillainStage: func(g *engine.Game, v *engine.Villain, nextStage int) []engine.Message {
				var msgs []engine.Message
				for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
					for _, c := range *zone {
						if c.Code == "32100" {
							zone.Remove(c.ID)
							msgs = append(msgs, engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c})
							break
						}
					}
					if len(msgs) > 0 {
						break
					}
				}
				if stage >= 2 {
					first := cardutil.FirstPlayerID(g)
					for _, p := range g.Players {
						if p.ID != first {
							msgs = append(msgs, engine.DealEncounterToPlayer{Player: p.ID})
						}
					}
				}
				if len(msgs) > 0 {
					g.Logf("Sentinel stage %d comes online", stage)
				}
				return msgs
			},
		})
	}

	// 32087 Night of the Sentinels: after threat lands here, at 5+/hero
	// the first player tucks their top card under OZT and 5+/hero threat
	// is removed.
	engine.RegisterBehavior("32087", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeThreat)
			s := g.MainScheme
			if !ok || s == nil || s.ID != e.EID() || m.Scheme != s.ID {
				return nil
			}
			per := 5 * len(g.Players)
			if s.Threat+m.N < per {
				return nil
			}
			first := cardutil.FirstPlayerID(g)
			tuckUnderOZT(g, first)
			return []engine.Message{engine.ThwartScheme{Scheme: s.ID, N: per, Source: s.ID}}
		},
	})

	// 32088 Mutants at the Mall (stage 2 / side scheme): When Defeated —
	// reveal a Sentinel minion and put Jubilee into play.
	engine.RegisterBehavior("32088", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			var msgs []engine.Message
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Def().Type == "minion" && c.Def().HasTrait("sentinel") {
						zone.Remove(c.ID)
						msgs = append(msgs, engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: c})
						break
					}
				}
				if len(msgs) > 0 {
					break
				}
			}
			// Jubilee (the b side) joins the first player.
			pid := cardutil.FirstPlayerID(g)
			a := &engine.Ally{
				ID: g.NextEntityID(engine.KindAlly), Code: "32088b", Owner: pid,
				MaxHP: 3, ThwartVal: 1, AttackVal: 1,
			}
			g.Allies[a.ID] = a
			if p := g.Player(pid); p != nil {
				p.Allies = append(p.Allies, a.ID)
			}
			g.Logf("Jubilee joins the fight!")
			return msgs
		},
	})
}

func registerWideawake() {
	// 32093 Sentinel Mark IV: guard/patrol from data; the boost
	// self-spawn matches the generic text scan.
	engine.RegisterBehavior("32093", &engine.Behavior{})

	// 32094 Gauntlet Beam / 32095 Learning A.I. / 32096 Adaptive Armor:
	// villain attachments with cosmetic riders; spend 3 same-type
	// resources to discard.
	villainAttachment := func(code, icons string) {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				if v := sentinelVillain(g); v != nil {
					t.Target = v.ID
				} else if mv := activeOrFirstVillain(g); mv != nil {
					t.Target = mv.ID
				}
				if code == "32096" {
					// +8 hit points.
					if mv := sentinelVillain(g); mv != nil {
						mv.MaxHP += 8
					}
				}
				return nil
			},
			Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
				t := g.Attachments[e.EID()]
				if t == nil {
					return nil
				}
				return []engine.Ability{{
					Label: "Discard " + t.EDef().Name + " — spend " + icons, Type: engine.AbilityAction,
					Cost: 3, CostIcons: icons,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
					},
				}}
			},
		})
	}
	villainAttachment("32094", "physical:3")
	villainAttachment("32095", "mental:3")
	villainAttachment("32096", "energy:3")

	// 32097 Self-Repair: clear the villain's statuses, tough, heal 5.
	engine.RegisterBehavior("32097", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			v := sentinelVillain(g)
			if v == nil {
				v = activeOrFirstVillain(g)
			}
			if v == nil {
				return nil
			}
			v.Stunned, v.Confused, v.Tough = false, false, false
			if v.Damage > 0 {
				v.Damage -= 5
				if v.Damage < 0 {
					v.Damage = 0
				}
			}
			g.Logf("%s self-repairs (heals 5)", v.EDef().Name)
			return []engine.Message{engine.ToughEntity{Target: v.ID}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if v := activeOrFirstVillain(g); v != nil {
				return []engine.Message{engine.ToughEntity{Target: v.ID}}
			}
			return nil
		},
	})

	// 32098 Mutant Detected: tuck your top card under OZT, or the villain
	// and each engaged minion attacks you.
	engine.RegisterBehavior("32098", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			attack := []engine.Message{}
			if v := sentinelVillain(g); v != nil {
				attack = append(attack, engine.AskAttack{Enemy: v.ID, Player: p.ID, Trigger: engine.TriggerVillainAttacksYou})
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.EngagedWith == p.ID {
					attack = append(attack, engine.MinionActivates{MinionID: id, Player: p.ID})
				}
			}
			return []engine.Message{engine.AskQuestion{
				Player: p.ID,
				Question: engine.Ask("Mutant Detected — choose:",
					engine.Choice{
						ID: "tuck", Label: "Place your top card facedown under Operation Zero Tolerance", Kind: engine.ChoiceLabel,
					}.Msgs(engine.TuckCardUnderOZT{Player: p.ID}),
					engine.Choice{
						ID: "fight", Label: "The villain and each engaged minion attacks you", Kind: engine.ChoiceLabel,
					}.Msgs(attack...),
				),
			}}
		},
	})

	// 32100 Abduction Protocols: Hinder 2/hero; When Defeated — a random
	// set-aside Captive ally enters play.
	engine.RegisterBehavior("32100", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			for i, c := range g.SetAside {
				if c.Def().HasTrait("captive") {
					g.SetAside = append(g.SetAside[:i], g.SetAside[i+1:]...)
					pid := cardutil.FirstPlayerID(g)
					return []engine.Message{engine.RevealEncounterCard{Player: pid, Card: c}}
				}
			}
			return nil
		},
	})
}

func registerZeroTolerance() {
	// 32101 Sentinel Mark II: When Revealed — surge if OZT is in play,
	// otherwise search OZT and reveal it.
	engine.RegisterBehavior("32101", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			if oztScheme(g) != nil {
				return []engine.Message{engine.RevealNextEncounter{Player: m.Player}}
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "32104" {
						zone.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: m.Player, Card: c}}
					}
				}
			}
			return nil
		},
	})

	// 32102 Sentinel Mark III: When Revealed — attach the Energy Barrier
	// from the encounter piles.
	engine.RegisterBehavior("32102", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "32103" {
						zone.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: m.Player, Card: c}}
					}
				}
			}
			return nil
		},
	})

	// 32103 Energy Barrier: attach to a Sentinel minion; it gains tough
	// after attacking.
	engine.RegisterBehavior("32103", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn == nil || !mn.EDef().HasTrait("sentinel") {
					continue
				}
				attached := false
				for _, aid := range mn.Attachments {
					if a := g.Attachments[aid]; a != nil && a.Code == "32103" {
						attached = true
					}
				}
				if attached {
					continue
				}
				t.Target = mn.ID
				mn.Attachments = append(mn.Attachments, t.ID)
				return []engine.Message{engine.ToughEntity{Target: mn.ID}}
			}
			g.Delete(t.ID)
			return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
		},
	})

	// 32104 Operation Zero Tolerance: defeated allies get tucked faceup;
	// at players+3 tucked cards the players lose.
	engine.RegisterBehavior("32104", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyDefeated)
			s := g.SideSchemes[e.EID()]
			if !ok || s == nil {
				return nil
			}
			if a := g.Allies[m.AllyID]; a != nil {
				s.StoredCards = append(s.StoredCards, engine.Card{ID: g.NextCardID(), Code: a.Code})
				g.Logf("%s is placed under Operation Zero Tolerance", a.EDef().Name)
			}
			if len(s.StoredCards) >= len(g.Players)+3 {
				return []engine.Message{engine.GameOver{Won: false, Reason: "Operation Zero Tolerance captured too many allies"}}
			}
			return nil
		},
	})
}

func registerSentinelsModular() {
	// 32105 Sentinel Mark V: When Revealed — attacks if targeted, else
	// search Targeted for Elimination.
	engine.RegisterBehavior("32105", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			for _, a := range g.Attachments {
				if a != nil && a.Code == "32107" && a.Target.Is(engine.KindPlayer) {
					return []engine.Message{engine.AskAttack{Enemy: e.EID(), Player: m.Player}}
				}
			}
			for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				for _, c := range *zone {
					if c.Code == "32107" {
						zone.Remove(c.ID)
						return []engine.Message{engine.RevealEncounterCard{Player: m.Player, Card: c}}
					}
				}
			}
			return nil
		},
	})

	// 32106 Sentinel Mark VI: quickstrike from data; boost rider skipped.
	engine.RegisterBehavior("32106", &engine.Behavior{})

	// 32107 Targeted for Elimination: attach to an identity without one;
	// action — exhaust to discard.
	engine.RegisterBehavior("32107", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, p := range g.Players {
				has := false
				for _, a := range g.Attachments {
					if a != nil && a.Code == "32107" && a.Target == p.ID {
						has = true
					}
				}
				if !has {
					t.Target = p.ID
					g.Logf("Targeted for Elimination attaches to %s", p.Name)
					return nil
				}
			}
			g.Delete(t.ID)
			return []engine.Message{engine.RevealNextEncounter{Player: cardutil.FirstPlayerID(g)}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			t := g.Attachments[e.EID()]
			if t == nil {
				return nil
			}
			return []engine.Ability{{
				Label: "Exhaust your identity → discard Targeted for Elimination", Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					t := g.Attachments[self]
					if t == nil {
						return nil
					}
					if p := g.Player(engine.PlayerID(t.Target)); p != nil && p.Exhausted {
						return []engine.Message{engine.DiscardAttachmentMsg{ID: self}}
					}
					return nil
				},
			}}
		},
	})

	// 32108 Relentless Robots: players engaged with a Sentinel cannot
	// thwart this scheme (enforced via thwartBlocker-style gate; the
	// engine lacks per-scheme gates, so the scheme exposes no extra
	// behavior).
	engine.RegisterBehavior("32108", &engine.Behavior{})
}
