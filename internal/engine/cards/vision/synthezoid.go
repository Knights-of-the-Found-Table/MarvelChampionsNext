package vision

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() { registerSynthezoid() }

// synVillain returns the first villain (leader proxy).
func synVillain(g *engine.Game) *engine.Villain {
	for _, v := range g.Villains {
		if v != nil {
			return v
		}
	}
	return nil
}

// synFindAttach pulls an attachment from the encounter piles and attaches
// it to the villain.
func synFindAttach(g *engine.Game, code string, target engine.EntityID) bool {
	for _, list := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
		for i, c := range *list {
			if data.BaseCode(c.Code) == code {
				*list = append((*list)[:i:i], (*list)[i+1:]...)
				g.SpawnAttachment(c.Code, target)
				return true
			}
		}
	}
	return false
}

// synEngagee returns the player a revealed minion affects.
func synEngagee(e engine.Entity) engine.PlayerID {
	if mn := gMinion(e); mn != nil {
		return mn.EngagedWith
	}
	return ""
}

func gMinion(e engine.Entity) *engine.Minion {
	if mn, ok := e.(*engine.Minion); ok {
		return mn
	}
	return nil
}

func registerSynthezoid() {
	// --- She-Hulk leader (57001-57004) ---
	for _, code := range []string{"57001", "57002", "57003", "57004"} {
		engine.RegisterBehavior(code, &engine.Behavior{
			React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
				// Forced Response: after an alter-ego changes to hero
				// form, 1 damage to that hero.
				if _, ok := msg.(engine.ChangeForm); !ok {
					return nil
				}
				v, ok := e.(*engine.Villain)
				if !ok {
					return nil
				}
				var out []engine.Message
				for _, p := range g.Players {
					if p.IsHero() && !p.KOed {
						out = append(out, engine.DamageEntity{Target: p.ID, Damage: 1, Source: v.ID})
					}
				}
				return out
			},
		})
	}
	engine.RegisterBehavior("57007", &engine.Behavior{})
	engine.RegisterBehavior("57008", &engine.Behavior{})
	engine.RegisterBehavior("57009", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			v := synVillain(g)
			if v == nil {
				return nil
			}
			var out []engine.Message
			if p.IsHero() {
				out = append(out, engine.AskAttack{Enemy: v.ID, Player: p.ID})
			} else if g.MainScheme != nil {
				out = append(out, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID})
			}
			out = append(out, engine.ToughEntity{Target: v.ID})
			return out
		},
	})
	engine.RegisterBehavior("57010", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var out []engine.Message
			if !p.Exhausted {
				out = append(out, engine.ExhaustEntity{ID: p.ID})
			}
			for _, aid := range p.Allies {
				if a := g.Allies[aid]; a != nil && !a.Exhausted {
					out = append(out, engine.ExhaustEntity{ID: aid})
				}
			}
			if len(out) == 0 {
				out = append(out, engine.RevealNextEncounter{Player: p.ID})
			}
			return out
		},
	})
	engine.RegisterBehavior("57011", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			// Discard the highest-cost ally/support; its cost becomes
			// main-scheme threat.
			best, bestCost := engine.EntityID(""), -1
			for _, aid := range p.Allies {
				if a := g.Allies[aid]; a != nil && cardCostOf(a.EDef()) > bestCost {
					best, bestCost = aid, cardCostOf(a.EDef())
				}
			}
			for _, sid := range p.Supports {
				if s := g.Supports[sid]; s != nil && cardCostOf(s.EDef()) > bestCost {
					best, bestCost = sid, cardCostOf(s.EDef())
				}
			}
			if best == "" || g.MainScheme == nil {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return []engine.Message{
				engine.DiscardControlled{Player: p.ID, ID: best},
				engine.SchemeThreat{Scheme: g.MainScheme.ID, N: bestCost, Source: t.ID},
			}
		},
	})
	engine.RegisterBehavior("57012", &engine.Behavior{})
	// She-Hulk player cards (57032-57035).
	engine.RegisterBehavior("57032", &engine.Behavior{})
	engine.RegisterBehavior("57033", &engine.Behavior{})
	engine.RegisterBehavior("57034", &engine.Behavior{})
	engine.RegisterBehavior("57035", &engine.Behavior{})

	// --- SHIELD Ops (57013-57016) ---
	engine.RegisterBehavior("57013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("57014", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var out []engine.Message
			if v := synVillain(g); v != nil {
				out = append(out, engine.ToughEntity{Target: v.ID})
			}
			for _, mn := range g.Minions {
				if mn != nil {
					out = append(out, engine.ToughEntity{Target: mn.ID})
				}
			}
			return out
		},
	})
	engine.RegisterBehavior("57015", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for i, c := range g.EncounterDeck {
				if c.Def().Type == "minion" {
					card := c
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					def := card.Def()
					if !p.IsHero() && g.MainScheme != nil {
						return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: synInt(def.Scheme, 1), Source: t.ID}}
					}
					return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: synInt(def.Attack, 1), Source: t.ID}}
				}
			}
			return nil
		},
	})
	engine.RegisterBehavior("57016", &engine.Behavior{})

	// --- Thunderbolts modular (57017-57021) ---
	engine.RegisterBehavior("57017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := gMinion(e)
			if mn == nil {
				return nil
			}
			if v := synVillain(g); v != nil {
				return []engine.Message{engine.StunEntity{Target: v.ID}}
			}
			return []engine.Message{engine.StunEntity{Target: mn.EngagedWith}}
		},
	})
	engine.RegisterBehavior("57018", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := gMinion(e)
			if mn == nil {
				return nil
			}
			if v := synVillain(g); v != nil {
				return []engine.Message{engine.ConfuseEntity{Target: v.ID}}
			}
			return []engine.Message{engine.ConfuseEntity{Target: mn.EngagedWith}}
		},
	})
	engine.RegisterBehavior("57019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := gMinion(e)
			if mn == nil {
				return nil
			}
			if v := synVillain(g); v != nil {
				return []engine.Message{engine.DamageEntity{Target: v.ID, Damage: 2, Source: e.EID()}}
			}
			return []engine.Message{engine.DamageEntity{Target: mn.EngagedWith, Damage: 2, Source: e.EID()}}
		},
	})
	engine.RegisterBehavior("57020", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var out []engine.Message
			for _, mn := range g.Minions {
				if mn != nil {
					out = append(out, engine.MinionActivates{MinionID: mn.ID, Player: mn.EngagedWith})
				}
			}
			if len(out) == 0 {
				if v := synVillain(g); v != nil {
					out = append(out, engine.VillainActivates{VillainID: v.ID, Player: p.ID})
				}
			}
			return out
		},
	})
	engine.RegisterBehavior("57021", &engine.Behavior{})

	// --- Taskmaster modular (57022-57026) ---
	engine.RegisterBehavior("57022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.MinionActivates{MinionID: e.EID(), Player: synEngagee(e)}}
		},
	})
	engine.RegisterBehavior("57023", &engine.Behavior{})
	engine.RegisterBehavior("57024", &engine.Behavior{})
	engine.RegisterBehavior("57025", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			var out []engine.Message
			if v := synVillain(g); v != nil {
				out = append(out, engine.ToughEntity{Target: v.ID})
			}
			return out
		},
	})
	engine.RegisterBehavior("57026", &engine.Behavior{})

	// --- Deadly Duo (57027-57031) ---
	millTypes := func(g *engine.Game, t *engine.Treachery, p *engine.Player, heroDamage bool) []engine.Message {
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
		if heroDamage {
			out = append(out, engine.IndirectDamage{Player: p.ID, N: len(types)})
		} else if g.MainScheme != nil {
			out = append(out, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: len(types), Source: t.ID})
		}
		return out
	}
	engine.RegisterBehavior("57027", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(synEngagee(e))
			if p == nil {
				return nil
			}
			return millTypes(g, nil, p, true)
		},
	})
	engine.RegisterBehavior("57028", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(synEngagee(e))
			if p == nil {
				return nil
			}
			return millTypes(g, nil, p, false)
		},
	})
	engine.RegisterBehavior("57029", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			if mn := g.Minions[target]; mn != nil {
				mn.MaxHP += 4
			}
			return nil
		},
	})
	engine.RegisterBehavior("57030", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			best, bestCost := engine.EntityID(""), -1
			for _, aid := range p.Allies {
				if a := g.Allies[aid]; a != nil && cardCostOf(a.EDef()) > bestCost {
					best, bestCost = aid, cardCostOf(a.EDef())
				}
			}
			for _, sid := range p.Supports {
				if s := g.Supports[sid]; s != nil && cardCostOf(s.EDef()) > bestCost {
					best, bestCost = sid, cardCostOf(s.EDef())
				}
			}
			if best == "" {
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: best}}
		},
	})
	engine.RegisterBehavior("57031", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			var out []engine.Message
			for _, p := range g.Players {
				out = append(out, engine.MillPlayerDeck{Player: p.ID, N: 8})
			}
			return out
		},
	})

	// --- standard_pvp (57036-57038, 57078-57080) ---
	pvpScheme := func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		if v := synVillain(g); v != nil && g.MainScheme != nil {
			return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID}}
		}
		return nil
	}
	pvpAttack := func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		if v := synVillain(g); v != nil {
			return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: p.ID}}
		}
		return nil
	}
	pvpDeal := func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		return []engine.Message{engine.DealEncounterToPlayer{Player: p.ID}}
	}
	for _, codes := range [][3]string{{"57036", "57037", "57038"}, {"57078", "57079", "57080"}} {
		engine.RegisterBehavior(codes[0], &engine.Behavior{ResolveTreachery: pvpScheme})
		engine.RegisterBehavior(codes[1], &engine.Behavior{ResolveTreachery: pvpAttack})
		engine.RegisterBehavior(codes[2], &engine.Behavior{ResolveTreachery: pvpDeal})
	}
	engine.RegisterBehavior("57039", &engine.Behavior{})
	engine.RegisterBehavior("57081", &engine.Behavior{})

	// --- Vision leader (57040-57054, 57074-57077) ---
	for _, code := range []string{"57040", "57041", "57042", "57043"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}
	// 57046a Dense / (linked Intangible) mass form attachments.
	engine.RegisterBehavior("57046", &engine.Behavior{})
	engine.RegisterBehavior("57047", &engine.Behavior{})
	engine.RegisterBehavior("57048", &engine.Behavior{})
	engine.RegisterBehavior("57049", &engine.Behavior{})
	engine.RegisterBehavior("57050", &engine.Behavior{})
	engine.RegisterBehavior("57051", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			v := synVillain(g)
			if v == nil {
				return nil
			}
			if p.IsHero() {
				return []engine.Message{engine.AskAttack{Enemy: v.ID, Player: p.ID}}
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: v.SchemeVal, Source: v.ID}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("57052", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if !p.Exhausted {
				return []engine.Message{engine.ExhaustEntity{ID: p.ID}, engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
			}
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
		},
	})
	engine.RegisterBehavior("57053", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if len(p.Upgrades) > 0 {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[0]}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("57054", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			g.Logf("Just Passing Through disorients its conqueror")
			return nil
		},
	})
	// Vision player cards.
	engine.RegisterBehavior("57074", &engine.Behavior{})
	engine.RegisterBehavior("57075", &engine.Behavior{})
	engine.RegisterBehavior("57076", &engine.Behavior{})
	engine.RegisterBehavior("57077", &engine.Behavior{})

	// --- Young Avengers (57055-57059) ---
	engine.RegisterBehavior("57055", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if v := synVillain(g); v != nil {
				return []engine.Message{engine.ToughEntity{Target: v.ID}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("57056", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(synEngagee(e))
			if p != nil && len(p.Upgrades) > 0 {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Upgrades[0]}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("57057", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(synEngagee(e))
			if p != nil && len(p.Supports) > 0 {
				return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: p.Supports[0]}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("57058", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
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
		},
	})
	engine.RegisterBehavior("57059", &engine.Behavior{})

	// --- Scarlet Twins (57060-57064) ---
	engine.RegisterBehavior("57060", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := gMinion(e)
			if mn == nil {
				return nil
			}
			if mn.EngagedWith != "" {
				return []engine.Message{engine.StunEntity{Target: mn.EngagedWith}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("57061", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			mn := gMinion(e)
			if mn == nil {
				return nil
			}
			if g.MainScheme != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 2, Source: e.EID()}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("57062", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if len(p.Hand) == 0 {
				return nil
			}
			idx := g.Random(len(p.Hand))
			card := p.Hand[idx]
			p.Hand = append(p.Hand[:idx:idx], p.Hand[idx+1:]...)
			p.Discard = append(p.Discard, card)
			return []engine.Message{
				engine.DiscardCards{Player: p.ID, Cards: engine.CardList{card}},
				engine.IndirectDamage{Player: p.ID, N: cardCostOf(card.Def())},
			}
		},
	})
	engine.RegisterBehavior("57063", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			g.Logf("Spellcasting fizzles the next card")
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
	})
	engine.RegisterBehavior("57064", &engine.Behavior{})

	// --- Moon Knight modular (57065-57068) ---
	engine.RegisterBehavior("57065", &engine.Behavior{})
	engine.RegisterBehavior("57066", &engine.Behavior{})
	engine.RegisterBehavior("57067", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			return []engine.Message{engine.ExhaustEntity{ID: p.ID}, engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
		},
	})
	engine.RegisterBehavior("57068", &engine.Behavior{})

	// --- Royal Guard (57069-57073) ---
	royalGuard := func(code string) {
		engine.RegisterBehavior(code, &engine.Behavior{
			OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
				mn := gMinion(e)
				if mn == nil {
					return nil
				}
				return []engine.Message{engine.MillPlayerDeck{Player: mn.EngagedWith, N: 2}}
			},
		})
	}
	royalGuard("57069")
	royalGuard("57070")
	engine.RegisterBehavior("57071", &engine.Behavior{
		ResolveTreachery: millTreathery,
	})
	engine.RegisterBehavior("57072", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			p.ExtraTraits = append(p.ExtraTraits, "hunted")
			g.Logf("%s is Hunted", p.Name)
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
	})
	engine.RegisterBehavior("57073", &engine.Behavior{})

	// Main scheme bases (single-face snapshot).
	for _, code := range []string{"57005", "57006", "57044", "57045"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	registerSynthezoidScenarios()
}

func millTreathery(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
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

// cardCostOf reads a card's printed cost (0 default).
func cardCostOf(def *data.CardDef) int {
	if def == nil || def.Cost == nil {
		return 0
	}
	return *def.Cost
}

// synInt dereferences a numeric card field with a fallback.
func synInt(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

func registerSynthezoidScenarios() {
	advance := func(reason string) func(g *engine.Game, s *engine.MainScheme) []engine.Message {
		return func(g *engine.Game, s *engine.MainScheme) []engine.Message {
			if s.Stage < len(s.StageCodes) {
				return []engine.Message{engine.ReplaceMainScheme{Scheme: s.ID}}
			}
			return []engine.Message{engine.GameOver{Won: false, Reason: reason}}
		}
	}

	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "57005b",
		Name:             "Synthezoid Smackdown — Registration: She-Hulk",
		VillainBases:     []string{"57001"},
		MainSchemeStages: []string{"57005b", "57006b"},
		ExtraSets:        []string{"shield_ops", "thunderbolts_modular", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			for _, v := range g.Villains {
				if g.Difficulty == "expert" {
					v.SetVillainStages([]string{"57003", "57004"})
				} else {
					v.SetVillainStages([]string{"57001", "57002"})
				}
				synFindAttach(g, "57007", v.ID)
			}
			return nil
		},
		OnMainSchemeMaxed: advance("The Registration Act completed"),
	})

	engine.RegisterScenario(&engine.ScenarioDef{
		ID:               "57044b",
		Name:             "Synthezoid Smackdown — Resistance: Vision",
		VillainBases:     []string{"57040"},
		MainSchemeStages: []string{"57044b", "57045b"},
		ExtraSets:        []string{"young_avengers", "scarlet_twins", "standard"},
		Setup: func(g *engine.Game) []engine.Message {
			for _, v := range g.Villains {
				if g.Difficulty == "expert" {
					v.SetVillainStages([]string{"57042", "57043"})
				} else {
					v.SetVillainStages([]string{"57040", "57041"})
				}
				if !synFindAttach(g, "57046", v.ID) {
					// Double-sided setup attachments never enter the
					// encounter deck; spawn the a-side directly.
					g.SpawnAttachment("57046a", v.ID)
				}
			}
			return nil
		},
		OnMainSchemeMaxed: advance("The Resistance was crushed"),
	})
}
