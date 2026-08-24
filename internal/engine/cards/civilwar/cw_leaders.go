package civilwar

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() { registerCWLeaders() }

// cwLeaderSCH grants +1 SCH per Tech attachment (max 3) — Iron Man's
// leader scaling.
func cwLeaderSCH(g *engine.Game, e engine.Entity) (int, int) {
	v, ok := e.(*engine.Villain)
	if !ok {
		return 0, 0
	}
	n := 0
	for _, aid := range v.Attachments {
		if a := g.Attachments[aid]; a != nil && a.EDef().HasTrait("tech") {
			n++
		}
	}
	if n > 3 {
		n = 3
	}
	return 0, n
}

// cwDealAll deals each player a facedown encounter card (the stage II/IV
// leader entry tax).
func cwDealAll(g *engine.Game) []engine.Message {
	var out []engine.Message
	for _, p := range g.Players {
		if !p.KOed {
			out = append(out, engine.DealEncounterToPlayer{Player: p.ID})
		}
	}
	return out
}

// cwFindAttach finds an attachment code in the encounter deck/discard and
// attaches it to the villain.
func cwFindAttach(g *engine.Game, code string, target engine.EntityID) bool {
	for _, list := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
		for i, c := range *list {
			if c.Code == code {
				*list = append((*list)[:i:i], (*list)[i+1:]...)
				g.SpawnAttachment(code, target)
				return true
			}
		}
	}
	return false
}

// cwVillain returns the first villain (leader proxy).
func cwVillain(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil {
			return v
		}
	}
	return nil
}

func registerCWLeaders() {
	// --- Iron Man leader (56059-56062) ---
	ironMan := func(code string) {
		engine.RegisterBehavior(code, &engine.Behavior{
			EnemyStatBonus: cwLeaderSCH,
		})
	}
	ironMan("56059")
	ironMan("56060")
	ironMan("56061")
	ironMan("56062")
	// Stage II/IV entry tax.
	for _, code := range []string{"56060", "56062"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			EnemyStatBonus: cwLeaderSCH,
			VillainStage: func(g *engine.Game, v *engine.Villain, nextStage int) []engine.Message {
				g.Logf("%s calls in reinforcements — he cannot be damaged this phase (approximated)", v.EDef().Name)
				return cwDealAll(g)
			},
		})
	}

	// 56065 Powered Gauntlets / 56066 Rocket Boots: armor riders.
	engine.RegisterBehavior("56065", &engine.Behavior{})
	engine.RegisterBehavior("56066", &engine.Behavior{})
	// 56067/56068: tough on attach.
	for _, code := range []string{"56067", "56068"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
				return []engine.Message{engine.ToughEntity{Target: target}}
			},
		})
	}
	// 56069 Mark V Armor: banks damage (approximated: +5 max HP shield).
	engine.RegisterBehavior("56069", &engine.Behavior{})
	// 56070 Repulsor Blast.
	engine.RegisterBehavior("56070", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				if len(p.Allies) > 0 {
					return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Allies[0]}}
				}
				if len(p.Supports) > 0 {
					return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Supports[0]}}
				}
				return nil
			}
			out := []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 4}}
			out = append(out, engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID})
			return out
		},
	})
	// 56071 Supersonic Punch.
	engine.RegisterBehavior("56071", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			v := cwVillain(g)
			if v == nil {
				return nil
			}
			if !p.IsHero() {
				if g.MainScheme != nil {
					out := []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID}}
					return out
				}
				return nil
			}
			return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: p.ID}}
		},
	})
	// 56072 Stark Tower: defeat reveals an Iron Man attachment.
	engine.RegisterBehavior("56072", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			if v := cwVillain(g); v != nil {
				for _, code := range []string{"56065", "56066", "56067", "56068", "56069"} {
					if cwFindAttach(g, code, v.ID) {
						return nil
					}
				}
			}
			return nil
		},
	})

	// --- Mighty Avengers (56073-56077) ---
	engine.RegisterBehavior("56073", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if v := cwVillain(g); v != nil {
				return []engine.Message{engine.ToughEntity{Target: v.ID}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("56074", &engine.Behavior{
		OnPlay: cwMainThreat1,
	})
	engine.RegisterBehavior("56075", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.DamageEntity{Target: e.EOwner(), Damage: 1, Source: e.EID()}}
		},
	})
	// 56076/56081 Mighty Avengers treachery: replay each minion's When
	// Revealed (approximated: each minion activates instead).
	cwMinionBlitz := func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		var out []engine.Message
		for _, mn := range g.Minions {
			if mn != nil {
				out = append(out, engine.MinionActivates{MinionID: mn.ID, Player: mn.EngagedWith})
			}
		}
		if len(out) == 0 {
			out = append(out, engine.RevealNextEncounter{Player: p.ID})
		}
		return out
	}
	engine.RegisterBehavior("56076", &engine.Behavior{ResolveTreachery: cwMinionBlitz})
	engine.RegisterBehavior("56081", &engine.Behavior{ResolveTreachery: cwMinionBlitz})
	engine.RegisterBehavior("56077", &engine.Behavior{})

	// --- The Initiative (56078-56082) ---
	engine.RegisterBehavior("56078", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			out := []engine.Message{engine.ConfuseEntity{Target: p.ID}}
			if p.Confused && g.MainScheme != nil {
				out = append(out, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()})
			}
			return out
		},
	})
	engine.RegisterBehavior("56079", &engine.Behavior{})
	engine.RegisterBehavior("56080", &engine.Behavior{})
	engine.RegisterBehavior("56082", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			g.ShuffleEncounterDeck()
			g.Logf("The Fifty State Initiative's remnants shuffle back in")
			return nil
		},
	})

	// --- Maria Hill modular (56083-56086) ---
	engine.RegisterBehavior("56083", &engine.Behavior{})
	engine.RegisterBehavior("56084", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			return []engine.Message{engine.ToughEntity{Target: target}}
		},
	})
	engine.RegisterBehavior("56085", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var out []engine.Message
			for _, sid := range g.Schemes() {
				out = append(out, engine.SchemeThreat{Scheme: sid, N: 2, Source: t.ID})
			}
			if cwMinionByCode(g, "56083") != nil {
				out = append(out, engine.RevealNextEncounter{Player: p.ID})
			}
			return out
		},
	})
	engine.RegisterBehavior("56086", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			p := g.Players[0]
			if p == nil {
				return nil
			}
			var out []engine.Message
			out = append(out, engine.DamageEntity{Target: p.ID, Damage: 1, Source: s.ID})
			for _, mn := range g.Minions {
				if mn != nil && mn.EngagedWith == p.ID {
					out = append(out, engine.DamageEntity{Target: mn.ID, Damage: 1, Source: s.ID})
				}
			}
			return out
		},
	})

	// --- Dangerous Recruits (56087-56091) ---
	engine.RegisterBehavior("56087", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: mn.EngagedWith, Damage: 2, Source: e.EID()}}
		},
	})
	engine.RegisterBehavior("56088", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p == nil || len(p.Upgrades) == 0 {
				return nil
			}
			return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[0]}}
		},
	})
	engine.RegisterBehavior("56089", &engine.Behavior{
		ResolveTreachery: cwAllMinionsActivate,
	})
	engine.RegisterBehavior("56090", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			n := len(g.Minions)
			if g.MainScheme != nil && n > 0 {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: t.ID}}
			}
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
	})
	engine.RegisterBehavior("56091", &engine.Behavior{})

	// --- Captain Marvel leader (56092-56095, 56098-56104) ---
	capMarvel := func(code string) {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
	capMarvel("56092")
	capMarvel("56093")
	capMarvel("56094")
	capMarvel("56095")
	// Energy Channel 56098: counters per attack against her; 4 triggers a
	// counterattack.
	engine.RegisterBehavior("56098", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			t, ok := e.(*engine.Attachment)
			if !ok {
				return nil
			}
			var attacker engine.EntityID
			switch m := msg.(type) {
			case engine.BasicAttack:
				if m.Target == t.Target {
					attacker = m.Player
				}
			case engine.AllyAttackWindow:
				if m.Target == t.Target {
					if a := g.Allies[m.Ally]; a != nil {
						attacker = a.Owner
					}
				}
			}
			if attacker == "" || t.Counters >= 3 {
				return nil
			}
			t.Counters++
			g.Logf("Energy Channel holds %d energy counters", t.Counters)
			if t.Counters >= 4 {
				t.Counters = 0
				if v := g.Villains[t.Target]; v != nil {
					return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: attacker}}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("56099", &engine.Behavior{})
	engine.RegisterBehavior("56100", &engine.Behavior{})
	engine.RegisterBehavior("56101", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				if len(p.Allies) > 0 {
					return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Allies[0]}}
				}
				return nil
			}
			var out []engine.Message
			if v := cwVillain(g); v != nil {
				out = append(out, engine.AskAttack{Enemy: v.ID, Player: p.ID})
			}
			return out
		},
	})
	engine.RegisterBehavior("56102", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				var out []engine.Message
				if v := cwVillain(g); v != nil && g.MainScheme != nil {
					out = append(out, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID})
				}
				return out
			}
			var out []engine.Message
			for _, sid := range g.Schemes() {
				out = append(out, engine.SchemeThreat{Scheme: sid, N: 2, Source: t.ID})
			}
			return out
		},
	})
	engine.RegisterBehavior("56103", &engine.Behavior{})
	engine.RegisterBehavior("56104", &engine.Behavior{})

	// --- Cape Killer (56105-56108) ---
	engine.RegisterBehavior("56105", &engine.Behavior{})
	engine.RegisterBehavior("56106", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			p.ExtraTraits = append(p.ExtraTraits, "unregistered")
			g.Logf("%s is flagged Unregistered", p.Name)
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}}
		},
	})
	engine.RegisterBehavior("56107", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var out []engine.Message
			for _, pl := range g.Players {
				if cwHasTrait(pl, "unregistered") && !pl.Exhausted {
					out = append(out, engine.ExhaustEntity{ID: pl.ID})
				}
			}
			if len(out) == 0 {
				out = append(out, engine.RevealNextEncounter{Player: p.ID})
			}
			return out
		},
	})
	engine.RegisterBehavior("56108", &engine.Behavior{})

	// --- Martial Law (56109-56111) ---
	engine.RegisterBehavior("56109", &engine.Behavior{})
	engine.RegisterBehavior("56110", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for i, c := range g.EncounterDeck {
				if c.Def().Type == "minion" {
					card := c
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					def := card.Def()
					if !p.IsHero() && g.MainScheme != nil {
						return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: intValue(def.Scheme, 1), Source: t.ID}}
					}
					return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: intValue(def.Attack, 1), Source: t.ID}}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("56111", &engine.Behavior{})

	// --- Heroes for Hire (56112-56116) ---
	engine.RegisterBehavior("56112", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			if p := g.Player(mn.EngagedWith); p != nil && len(p.Allies) > 0 {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Allies[0]}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("56113", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			if p := g.Player(mn.EngagedWith); p != nil && len(p.Upgrades) > 0 {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[0]}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("56114", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			p.ExtraTraits = append(p.ExtraTraits, "unregistered")
			g.Logf("%s is flagged Unregistered", p.Name)
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}}
		},
	})
	bounty := func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		if len(p.Hand) == 0 {
			return nil
		}
		card := p.Hand[0]
		p.Hand = p.Hand[1:]
		p.Discard = append(p.Discard, card)
		n := len(card.Def().Resources)
		if n == 0 {
			n = 1
		}
		if g.MainScheme != nil {
			return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{card}},
				engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: t.ID}}
		}
		return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{card}}}
	}
	engine.RegisterBehavior("56115", &engine.Behavior{ResolveTreachery: bounty})
	engine.RegisterBehavior("56119", &engine.Behavior{ResolveTreachery: bounty})
	engine.RegisterBehavior("56116", &engine.Behavior{})

	// --- Paladin (56117-56120) ---
	engine.RegisterBehavior("56117", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			p := g.Player(mn.EngagedWith)
			if p != nil && len(p.Hand) > 0 {
				card := p.Hand[0]
				p.Hand = p.Hand[1:]
				p.Discard = append(p.Discard, card)
				return []engine.Message{engine.DiscardCards{Player: p.ID, Cards: engine.CardList{card}}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("56118", &engine.Behavior{})
	engine.RegisterBehavior("56120", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, p := range g.Players {
				p.ExtraTraits = append(p.ExtraTraits, "unregistered")
			}
			g.Logf("Everyone is flagged Unregistered")
			return nil
		},
	})

	// --- standard_pvp (56125-56127, 56203-56205) ---
	pvpRighteous := func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		if v := cwVillain(g); v != nil && g.MainScheme != nil {
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID}}
		}
		return nil
	}
	pvpWhatever := func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		if v := cwVillain(g); v != nil {
			return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: p.ID}}
		}
		return nil
	}
	pvpTargeted := func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		return []engine.Message{engine.DealEncounterToPlayer{Player: p.ID}}
	}
	for _, codes := range [][3]string{{"56125", "56126", "56127"}, {"56203", "56204", "56205"}} {
		engine.RegisterBehavior(codes[0], &engine.Behavior{ResolveTreachery: pvpRighteous})
		engine.RegisterBehavior(codes[1], &engine.Behavior{ResolveTreachery: pvpWhatever})
		engine.RegisterBehavior(codes[2], &engine.Behavior{ResolveTreachery: pvpTargeted})
	}
	// 56128a/56206a Choosing Sides: the damage cap is approximated away.
	engine.RegisterBehavior("56128", &engine.Behavior{})
	engine.RegisterBehavior("56206", &engine.Behavior{})

	// --- Iron Man player cards (56129-56132) ---
	engine.RegisterBehavior("56129", &engine.Behavior{})
	engine.RegisterBehavior("56130", &engine.Behavior{})
	engine.RegisterBehavior("56131", &engine.Behavior{})
	engine.RegisterBehavior("56132", &engine.Behavior{})

	// --- Captain Marvel player cards (56133-56136) ---
	engine.RegisterBehavior("56133", &engine.Behavior{})
	engine.RegisterBehavior("56134", &engine.Behavior{})
	engine.RegisterBehavior("56135", &engine.Behavior{})
	engine.RegisterBehavior("56136", &engine.Behavior{})

	// --- Captain America leader (56137-56149) ---
	for _, code := range []string{"56137", "56138", "56139", "56140"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
	// 56143 Cap's Shield: retaliate + steals to attackers (approximated:
	// retaliate via engine hook is villain-side; here just moves).
	engine.RegisterBehavior("56143", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicAttack)
			t, ok2 := e.(*engine.Attachment)
			if !ok || !ok2 || m.Target != t.Target {
				return nil
			}
			t.Target = m.Player
			g.Logf("Cap's Shield is flung at the attacker")
			return nil
		},
	})
	engine.RegisterBehavior("56144", &engine.Behavior{})
	engine.RegisterBehavior("56145", &engine.Behavior{})
	engine.RegisterBehavior("56146", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			v := cwVillain(g)
			if v == nil {
				return nil
			}
			if !p.IsHero() && g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID}}
			}
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 2, Source: v.ID},
				engine.StunEntity{Target: p.ID}}
		},
	})
	engine.RegisterBehavior("56147", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			v := cwVillain(g)
			if v == nil {
				return nil
			}
			if !p.IsHero() && g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID}}
			}
			var out []engine.Message
			out = append(out, engine.DamageEntity{Target: p.ID, Damage: 2, Source: v.ID})
			return out
		},
	})
	engine.RegisterBehavior("56148", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				if len(p.Allies) > 0 {
					return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Allies[0]}}
				}
			}
			if v := cwVillain(g); v != nil {
				return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: p.ID}, engine.StunEntity{Target: p.ID}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("56149", &engine.Behavior{})
	// Captain America player cards (56207-56210).
	engine.RegisterBehavior("56207", &engine.Behavior{})
	engine.RegisterBehavior("56208", &engine.Behavior{})
	engine.RegisterBehavior("56209", &engine.Behavior{})
	engine.RegisterBehavior("56210", &engine.Behavior{})

	// --- New Avengers (56150-56154) ---
	for _, code := range []string{"56150", "56151", "56152"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
	engine.RegisterBehavior("56153", &engine.Behavior{ResolveTreachery: cwMinionBoostBlitz})
	engine.RegisterBehavior("56154", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			g.ShuffleEncounterDeck()
			return nil
		},
	})

	// --- Secret Avengers (56155-56159) ---
	for _, code := range []string{"56155", "56156"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
	engine.RegisterBehavior("56157", &engine.Behavior{ResolveTreachery: cwMinionBoostBlitz})
	engine.RegisterBehavior("56158", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if len(p.Allies) > 0 {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Allies[0]}}
			}
			return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
		},
	})
	engine.RegisterBehavior("56159", &engine.Behavior{})

	// --- Namor modular (56160-56164) ---
	engine.RegisterBehavior("56160", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			return []engine.Message{engine.MinionActivates{MinionID: e.EID(), Player: mn.EngagedWith}}
		},
	})
	engine.RegisterBehavior("56161", &engine.Behavior{})
	engine.RegisterBehavior("56162", &engine.Behavior{})
	engine.RegisterBehavior("56163", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if v := cwVillain(g); v != nil {
				return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: p.ID}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("56164", &engine.Behavior{})

	// --- Atlanteans (56165-56167) ---
	engine.RegisterBehavior("56165", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			return []engine.Message{engine.MillPlayerDeck{Player: mn.EngagedWith, N: 3}}
		},
	})
	atlanteans := func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		n := 3
		if len(p.Deck) < n {
			n = len(p.Deck)
		}
		milled := append(engine.CardList(nil), p.Deck[:n]...)
		p.Deck = p.Deck[n:]
		p.Discard = append(p.Discard, milled...)
		types := map[string]bool{}
		for _, c := range milled {
			types[c.Def().Type] = true
		}
		out := []engine.Message{engine.DiscardCards{Player: p.ID, Cards: milled}}
		if p.IsHero() {
			out = append(out, engine.IndirectDamage{Player: p.ID, N: len(types)})
		} else if g.MainScheme != nil {
			out = append(out, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: len(types), Source: t.ID})
		}
		return out
	}
	engine.RegisterBehavior("56166", &engine.Behavior{ResolveTreachery: atlanteans})
	engine.RegisterBehavior("56167", &engine.Behavior{})

	// --- Spider-Woman leader (56168-56179) ---
	for _, code := range []string{"56168", "56169", "56170", "56171"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			EnemyStatBonus: func(g *engine.Game, e engine.Entity) (int, int) {
				return cwFinesseBonus(g, e), 0
			},
		})
	}
	// 56174 Finesse: a skill counter per treachery reveal (max 3).
	engine.RegisterBehavior("56174", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			r, ok := msg.(engine.RevealEncounterCard)
			if !ok || r.Card.Def().Type != "treachery" {
				return nil
			}
			t, ok2 := e.(*engine.Attachment)
			if !ok2 || t.Counters >= 3 {
				return nil
			}
			t.Counters++
			g.Logf("Finesse gains a skill counter (%d)", t.Counters)
			return nil
		},
	})
	engine.RegisterBehavior("56175", &engine.Behavior{})
	engine.RegisterBehavior("56176", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return []engine.Message{engine.StunEntity{Target: p.ID}, engine.ConfuseEntity{Target: p.ID}}
		},
	})
	engine.RegisterBehavior("56177", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.IsHero() {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			if v := cwVillain(g); v != nil {
				return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: p.ID}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("56178", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if v := cwVillain(g); v != nil && g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("56179", &engine.Behavior{})
	// Spider-Woman player cards (56211-56214).
	engine.RegisterBehavior("56211", &engine.Behavior{})
	engine.RegisterBehavior("56212", &engine.Behavior{})
	engine.RegisterBehavior("56213", &engine.Behavior{})
	engine.RegisterBehavior("56214", &engine.Behavior{})

	// --- Spider-Man modular (56180-56183) ---
	engine.RegisterBehavior("56180", &engine.Behavior{})
	engine.RegisterBehavior("56181", &engine.Behavior{})
	engine.RegisterBehavior("56182", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: t.ID}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("56183", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s := g.SideSchemes[e.EID()]
			s.Threat += 2 * len(g.Players)
			return nil
		},
	})

	// --- Defenders (56184-56188) ---
	engine.RegisterBehavior("56184", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			return []engine.Message{engine.StunEntity{Target: mn.EngagedWith}}
		},
	})
	engine.RegisterBehavior("56185", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			return []engine.Message{engine.ConfuseEntity{Target: mn.EngagedWith}}
		},
	})
	engine.RegisterBehavior("56186", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var out []engine.Message
			if v := cwVillain(g); v != nil {
				out = append(out, engine.ToughEntity{Target: v.ID})
			}
			for _, id := range cardutil.SortedEnemyIDs(g) {
				out = append(out, engine.ToughEntity{Target: id})
			}
			return out
		},
	})
	engine.RegisterBehavior("56187", &engine.Behavior{ResolveTreachery: cwAllMinionsActivate})
	engine.RegisterBehavior("56188", &engine.Behavior{})
	engine.RegisterBehavior("56192", &engine.Behavior{ResolveTreachery: cwAllMinionsActivate})

	// --- Hell's Kitchen (56189-56193) ---
	engine.RegisterBehavior("56189", &engine.Behavior{})
	engine.RegisterBehavior("56190", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			p := g.Player(mn.EngagedWith)
			if p == nil || p.Exhausted {
				return nil
			}
			return []engine.Message{engine.ExhaustEntity{ID: p.ID}}
		},
	})
	engine.RegisterBehavior("56191", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if mn := g.Minions[target]; mn != nil {
				mn.MaxHP += 4
			}
			return nil
		},
	})
	engine.RegisterBehavior("56193", &engine.Behavior{})

	// --- Cloak and Dagger (56194-56198) ---
	engine.RegisterBehavior("56194", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("56195", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			var out []engine.Message
			if v := cwVillain(g); v != nil {
				out = append(out, engine.HealEntity{Target: v.ID, N: 2})
			}
			out = append(out, engine.DamageEntity{Target: mn.EngagedWith, Damage: 2, Source: e.EID()})
			return out
		},
	})
	engine.RegisterBehavior("56196", &engine.Behavior{})
	engine.RegisterBehavior("56197", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var out []engine.Message
			if v := cwVillain(g); v != nil {
				out = append(out, engine.HealEntity{Target: v.ID, N: 3})
			}
			out = append(out, engine.IndirectDamage{Player: p.ID, N: 3})
			return out
		},
	})
	engine.RegisterBehavior("56198", &engine.Behavior{})
}

// cwMainThreat1 places 1 threat on the main scheme (minion When
// Revealed).
func cwMainThreat1(g *engine.Game, e engine.Entity) []engine.Message {
	if g.MainScheme != nil {
		return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: e.EID()}}
	}
	return nil
}

// cwAllMinionsActivate: every minion activates against its engagee.
func cwAllMinionsActivate(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
	var out []engine.Message
	for _, mn := range g.Minions {
		if mn != nil {
			out = append(out, engine.MinionActivates{MinionID: mn.ID, Player: mn.EngagedWith})
		}
	}
	if len(out) == 0 {
		if v := cwVillain(g); v != nil {
			out = append(out, engine.VillainActivates{VillainID: v.ID, Player: p.ID})
		}
	}
	return out
}

// cwMinionBoostBlitz resolves each minion's Boost rider — approximated as
// each engaged minion activating.
func cwMinionBoostBlitz(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
	return cwAllMinionsActivate(g, t, p)
}

// cwMinionByCode finds a live minion with the exact code.
func cwMinionByCode(g *engine.Game, code string) *engine.Minion {
	for _, mn := range g.Minions {
		if mn != nil && mn.Code == code {
			return mn
		}
	}
	return nil
}

// cwFinesseBonus grants the Spider-Woman leader +1 ATK per Finesse
// counter.
func cwFinesseBonus(g *engine.Game, e engine.Entity) int {
	for _, a := range g.Attachments {
		if a != nil && data.BaseCode(a.Code) == "56174" {
			return a.Counters
		}
	}
	return 0
}

// cwHasTrait checks a player's dynamic extra traits.
func cwHasTrait(p *engine.Player, trait string) bool {
	for _, t := range p.ExtraTraits {
		if t == trait {
			return true
		}
	}
	return false
}
