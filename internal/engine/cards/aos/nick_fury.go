package aos

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() {
	registerNickFury()
	registerSuitForms()
	registerNickSignatures()
	registerNickNemesis()
	registerNickObligation()
}

func suitUpgrade(g *engine.Game, pid engine.PlayerID) *engine.Upgrade {
	p := g.Player(pid)
	if p == nil {
		return nil
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && len(u.Code) >= 5 && u.Code[:5] == "50035" {
			return u
		}
	}
	return nil
}

func suitForm(g *engine.Game, pid engine.PlayerID) string {
	u := suitUpgrade(g, pid)
	if u == nil {
		return ""
	}
	if u.Code == "50035b" {
		return "stealth"
	}
	return "assault"
}

func setSuitForm(g *engine.Game, pid engine.PlayerID, form string) {
	u := suitUpgrade(g, pid)
	if u == nil {
		return
	}
	code := "50035a"
	if form == "stealth" {
		code = "50035b"
	}
	if u.Code != code {
		u.Code = code
		g.Logf("Nick Fury changes to %s suit form", form)
	}
}

// stealthSignal uses the engine's neutral counter adjustment as a package-
// local flip message. The core engine has no generic double-sided-upgrade
// message; both suit faces recognize N=0 as "change to Stealth".
func stealthSignal(u *engine.Upgrade) engine.Message {
	return engine.AddEntityCounter{ID: u.ID, N: 0}
}

func assaultBonus(u *engine.Upgrade) int {
	if u == nil || u.Code != "50035a" || u.Counters <= 0 {
		return 0
	}
	n := min(3, u.Counters)
	u.Counters -= n
	return n
}

// registerNickFury installs Nick Fury (50034a/b).
func registerNickFury() {
	engine.RegisterBehavior("50034", &engine.Behavior{
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			for _, zone := range []engine.CardList{p.Hand, p.Deck, p.Discard} {
				for _, c := range zone {
					if len(c.Code) >= 5 && c.Code[:5] == "50035" {
						return []engine.Message{engine.UpgradeEnterPlay{Player: p.ID, Card: c}}
					}
				}
			}
			return nil
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			if suitUpgrade(g, p.ID) == nil {
				return nil
			}
			return []engine.Ability{{
				Label: "Infiltrate — change to Stealth suit form",
				Type:  engine.AbilityAction, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := suitUpgrade(g, self)
					if u == nil {
						return nil
					}
					return []engine.Message{stealthSignal(u)}
				},
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.BasicThwart:
				if m.Player == p.ID && !p.Confused {
					if u := suitUpgrade(g, p.ID); u != nil {
						return []engine.Message{engine.AddEntityCounter{ID: u.ID, N: 1}}
					}
				}
			case engine.BasicAttack:
				if m.Player == p.ID && !p.Stunned {
					// Break Cover is forced and happens before the suit's
					// Assault interrupt is collected.
					setSuitForm(g, p.ID, "assault")
				}
			}
			return nil
		},
	})
}

func registerSuitForms() {
	assaultReact := func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		u := g.Upgrades[e.EID()]
		if u == nil {
			return nil
		}
		if m, ok := msg.(engine.AddEntityCounter); ok && m.ID == u.ID && m.N == 0 {
			setSuitForm(g, u.Owner, "stealth")
			return nil
		}
		m, ok := msg.(engine.BasicAttack)
		p := g.Player(u.Owner)
		if !ok || m.Player != u.Owner || u.Code != "50035a" || p == nil || p.Stunned {
			return nil
		}
		n := assaultBonus(u)
		if n == 0 {
			return nil
		}
		return []engine.Message{engine.DamageEntity{Target: m.Target, Damage: n, Source: u.Owner}}
	}

	stealthReact := func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		u := g.Upgrades[e.EID()]
		if u == nil {
			return nil
		}
		if m, ok := msg.(engine.AddEntityCounter); ok && m.ID == u.ID && m.N == 0 {
			setSuitForm(g, u.Owner, "stealth")
			return nil
		}
		p := g.Player(u.Owner)
		if p == nil || !p.IsHero() || u.Code != "50035b" {
			return nil
		}
		hit := false
		switch m := msg.(type) {
		case engine.VillainActivates:
			hit = m.Player == p.ID
		case engine.MinionActivates:
			hit = m.Player == p.ID
		}
		if !hit || u.Counters > 5 {
			return nil
		}
		// Approximation: activation replacement is not cancellable through a
		// Behavior hook. The enemy still attacks, but the 1 redirected threat
		// is stored on Stealth so the suit economy remains operational.
		return []engine.Message{engine.AddEntityCounter{ID: u.ID, N: 1}}
	}

	assaultBehavior := &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if u, ok := e.(*engine.Upgrade); ok {
				u.Code = "50035a"
			}
			return nil
		},
		React: assaultReact,
	}
	// Register the base for Implemented(), plus exact faces so behavior()
	// can dispatch after the upgrade's Code flips.
	engine.RegisterBehavior("50035", assaultBehavior)
	engine.RegisterBehavior("50035a", assaultBehavior)
	engine.RegisterBehavior("50035b", &engine.Behavior{React: stealthReact})
}

func registerNickSignatures() {
	// 50036 Maria Hill: place threat equal to her printed THW on the suit.
	// Actual removed threat can be lower, which the current ally window does
	// not report.
	engine.RegisterBehavior("50036", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyThwartWindow)
			a := g.Allies[e.EID()]
			if !ok || a == nil || m.Ally != a.ID {
				return nil
			}
			u := suitUpgrade(g, a.Owner)
			if u == nil {
				return nil
			}
			return []engine.Message{engine.AddEntityCounter{ID: u.ID, N: max(0, a.ThwartVal+a.PermTHW+a.BonusTHW)}}
		},
	})

	// 50037 Concentrated Fire. If the selected enemy is already within the
	// attack's damage range, the defeat rider is offered as a nested choice.
	engine.RegisterBehavior("50037", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			setSuitForm(g, pid, "assault")
			u := suitUpgrade(g, pid)
			damage := 4 + assaultBonus(u)
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				target := g.Entity(id)
				if target == nil {
					continue
				}
				choice := engine.Choice{
					Label: fmt.Sprintf("%s — %d ranged damage", cardutil.EnemyLabel(target), damage),
					Kind:  engine.ChoiceTarget, SourceID: id, CardCode: target.ECode(),
				}.Msgs(engine.DamageEntity{Target: id, Damage: damage, Source: pid})
				lethal, scheme := false, 0
				switch t := target.(type) {
				case *engine.Minion:
					lethal, scheme = t.HP() <= damage, t.SchemeVal
				case *engine.Villain:
					lethal, scheme = t.HP() <= damage, t.SchemeVal
				}
				if lethal && u != nil {
					rider := engine.Ask("Concentrated Fire — enemy defeated; choose:",
						engine.Choice{ID: "intel", Label: fmt.Sprintf("Place %d threat on the suit", scheme), Kind: engine.ChoiceLabel}.
							Msgs(engine.AddEntityCounter{ID: u.ID, N: scheme}),
						engine.Choice{ID: "stealth", Label: "Change to Stealth suit form", Kind: engine.ChoiceLabel}.
							Msgs(stealthSignal(u)),
					)
					choice = choice.WithThen(rider)
				}
				choices = append(choices, choice)
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: pid,
				Question: engine.Ask("Concentrated Fire — choose an enemy", choices...)}}
		},
	})

	// 50038 Covert Surveillance.
	engine.RegisterBehavior("50038", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			u := suitUpgrade(g, pid)
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				if s == nil {
					continue
				}
				base := []engine.Message{engine.ThwartScheme{Scheme: id, N: 2, Source: pid}}
				if u != nil {
					if suitForm(g, pid) == "stealth" {
						base = append(base, engine.AddEntityCounter{ID: u.ID, N: 2})
					} else {
						base = append(base, stealthSignal(u))
					}
				}
				choices = append(choices, engine.Choice{
					Label: "Remove 2 threat from " + s.EDef().Name + " and use the suit rider",
					Kind:  engine.ChoiceTarget, SourceID: id, CardCode: s.ECode(),
				}.Msgs(base...))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: pid,
				Question: engine.Ask("Covert Surveillance — choose a scheme", choices...)}}
		},
	})

	// 50039 Spray Fire.
	engine.RegisterBehavior("50039", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			setSuitForm(g, pid, "assault")
			damage := 3 + assaultBonus(suitUpgrade(g, pid))
			var choices []engine.Choice
			for _, targetPlayer := range g.Players {
				var msgs []engine.Message
				for _, id := range cardutil.SortedIDs(g.Villains) {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: damage, Source: pid})
				}
				for _, id := range cardutil.SortedIDs(g.Minions) {
					if mn := g.Minions[id]; mn != nil && mn.EngagedWith == targetPlayer.ID {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: damage, Source: pid})
					}
				}
				choices = append(choices, engine.Choice{
					Label: fmt.Sprintf("%s's enemies — %d ranged damage", targetPlayer.Name, damage),
					Kind:  engine.ChoiceTarget, SourceID: targetPlayer.ID,
				}.Msgs(msgs...))
			}
			return []engine.Message{engine.AskQuestion{Player: pid,
				Question: engine.Ask("Spray Fire — choose a player", choices...)}}
		},
	})

	// 50040 Fury's Flying Car. Aerial cannot currently expire at end of
	// round, so the granted trait persists.
	engine.RegisterBehavior("50040", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			u := suitUpgrade(g, e.EOwner())
			if s == nil || u == nil || u.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "Fury's Flying Car — remove 1 suit threat and ready Nick Fury",
				Type:  engine.AbilityAction, HeroOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					if s == nil {
						return nil
					}
					u := suitUpgrade(g, s.Owner)
					if u == nil || u.Counters <= 0 {
						return nil
					}
					return []engine.Message{
						engine.AddEntityCounter{ID: u.ID, N: -1},
						engine.ReadyEntity{ID: s.Owner},
						engine.GrantTrait{Target: s.Owner, Trait: "aerial"},
					}
				},
			}}
		},
	})

	// 50041 Safe House #221.
	engine.RegisterBehavior("50041", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Safe House #221 — heal 2 or place 1 suit threat",
				Type:  engine.AbilityAction, AlterEgoOnly: true, Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					if s == nil {
						return nil
					}
					choices := []engine.Choice{engine.Choice{
						ID: "heal", Label: "Heal 2 damage from Nick Fury", Kind: engine.ChoiceLabel,
					}.Msgs(engine.HealEntity{Target: s.Owner, N: 2})}
					if u := suitUpgrade(g, s.Owner); u != nil {
						choices = append(choices, engine.Choice{
							ID: "intel", Label: "Place 1 threat on the suit", Kind: engine.ChoiceLabel,
						}.Msgs(engine.AddEntityCounter{ID: u.ID, N: 1}))
					}
					return []engine.Message{engine.AskQuestion{Player: s.Owner,
						Question: engine.Ask("Safe House #221 — choose", choices...)}}
				},
			}}
		},
	})

	// 50042 EM Shield. Attack attribution is unavailable to prevention
	// hooks, so it prevents the next damage instance and discards.
	engine.RegisterBehavior("50042", &engine.Behavior{
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			g.Delete(u.ID)
			p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			return n, 0
		},
	})

	// 50043 Eyepatch Camera. The interrupt is automatic rather than
	// optional; matching threat is removed again after placement, producing
	// the same net main-scheme threat.
	engine.RegisterBehavior("50043", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeThreat)
			u := g.Upgrades[e.EID()]
			p := g.Player(e.EOwner())
			if !ok || u == nil || p == nil || !p.IsHero() || g.MainScheme == nil || m.Scheme != g.MainScheme.ID || m.N <= 0 {
				return nil
			}
			n := min(3, m.N)
			suit := suitUpgrade(g, p.ID)
			if suit == nil {
				return nil
			}
			return []engine.Message{
				engine.DiscardControlled{Player: p.ID, ID: u.ID},
				engine.ThwartScheme{Scheme: m.Scheme, N: n, Source: u.ID},
				engine.AddEntityCounter{ID: suit.ID, N: n},
			}
		},
	})

	// 50044 Fury's Watch. ResourceAbility can consume one counter per use;
	// generating two resources in one exhaustion is not representable.
	engine.RegisterBehavior("50044", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "mental", UsesCounters: true},
	})

	// 50045 Intelligence Analysis: the treachery interrupt channel only
	// scans events in hand, not upgrades in play.
	engine.RegisterBehavior("50045", &engine.Behavior{})

	// 50046 Secret Agent: the engine has no preparation-resolved window.
	engine.RegisterBehavior("50046", &engine.Behavior{})
}

func registerNickObligation() {
	engine.RegisterBehavior("50059", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			setSuitForm(g, p.ID, "assault")
			u := suitUpgrade(g, p.ID)
			if u == nil || u.Counters == 0 {
				return []engine.Message{
					engine.ObligationResolve{Player: p.ID, Card: card},
					engine.DealEncounterToPlayer{Player: p.ID},
				}
			}
			n := u.Counters
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask("Discovered — choose:",
					engine.Choice{ID: "damage", Label: fmt.Sprintf("Take %d damage", n), Kind: engine.ChoiceLabel}.
						Msgs(engine.DamageEntity{Target: p.ID, Damage: n, Source: u.ID},
							engine.ObligationResolve{Player: p.ID, Card: card}),
					engine.Choice{ID: "clear", Label: fmt.Sprintf("Remove all %d threat", n), Kind: engine.ChoiceLabel}.
						Msgs(engine.AddEntityCounter{ID: u.ID, N: -n},
							engine.ObligationResolve{Player: p.ID, Card: card}),
				)}}
		},
	})
}

func intValue(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

func takeEncounterBy(g *engine.Game, match func(*data.CardDef) bool) (engine.Card, bool) {
	for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
		for _, c := range *zone {
			if match(c.Def()) {
				return zone.Remove(c.ID)
			}
		}
	}
	return engine.Card{}, false
}

func engageEncounterMinion(g *engine.Game, p *engine.Player, match func(*data.CardDef) bool) (*engine.Minion, bool) {
	card, ok := takeEncounterBy(g, match)
	if !ok {
		return nil, false
	}
	def := card.Def()
	mn := &engine.Minion{
		ID: g.NextEntityID("minion"), Code: card.Code,
		MaxHP: intValue(def.HP, 1), AttackVal: intValue(def.Attack, 0), SchemeVal: intValue(def.Scheme, 0),
		Tough: def.HasKeyword("Toughness"), Guard: def.HasKeyword("Guard"), EngagedWith: p.ID,
	}
	g.Minions[mn.ID] = mn
	return mn, true
}

func registerNickNemesis() {
	// 50060 Orion. Multiple tough cards collapse to the engine's boolean
	// tough state. A neutral counter message delays the new tough status
	// until after the triggering damage resolves.
	engine.RegisterBehavior("50060", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			mn := g.Minions[e.EID()]
			if mn == nil {
				return nil
			}
			if m, ok := msg.(engine.AddEntityCounter); ok && m.ID == mn.ID && m.N == 0 {
				mn.Tough = true
				return nil
			}
			m, ok := msg.(engine.DamageEntity)
			if !ok || m.Target != mn.ID || m.Damage <= 0 || mn.Tough {
				return nil
			}
			return []engine.Message{engine.AddEntityCounter{ID: mn.ID, N: 0}}
		},
	})

	// 50061 Acquire Infinity Formula.
	engine.RegisterBehavior("50061", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DamageEntity)
			if !ok || m.Damage <= 0 {
				return nil
			}
			p := g.Player(m.Target)
			if p == nil || p.HeroCode[:5] != "50034" {
				return nil
			}
			for _, mn := range g.Minions {
				if mn != nil && len(mn.Code) >= 5 && mn.Code[:5] == "50060" {
					return []engine.Message{engine.AddEntityCounter{ID: mn.ID, N: 0}}
				}
			}
			mn, found := engageEncounterMinion(g, p, func(def *data.CardDef) bool { return def.Code == "50060" })
			if !found {
				return nil
			}
			return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID}}
		},
	})

	// 50062 Leviathan Soldier.
	engine.RegisterBehavior("50062", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionActivates)
			mn := g.Minions[e.EID()]
			if !ok || mn == nil || m.MinionID != mn.ID {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil || p.IsHero() {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: mn.ID}}
		},
	})

	// 50063 Cold Storage. Encounter-deck shuffle after the search is omitted
	// because shuffle is not exposed as a generic encounter message.
	engine.RegisterBehavior("50063", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			mn, found := engageEncounterMinion(g, p, func(def *data.CardDef) bool {
				return def.Type == "minion" && def.HasTrait("leviathan")
			})
			if !found {
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 2, Source: t.ID}}
			}
			return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			if len(g.Players) == 0 {
				return nil
			}
			// Boost hooks do not expose the activation target; use the first
			// player as a deterministic approximation.
			return []engine.Message{engine.DamageEntity{Target: g.Players[0].ID, Damage: 1}}
		},
	})
}
