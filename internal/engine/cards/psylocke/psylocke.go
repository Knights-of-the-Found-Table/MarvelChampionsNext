// Package psylocke registers the Psylocke hero pack (41001): the
// Psylocke / Betsy Braddock identity with the double-sided Psi-Knife /
// Psi-Katana upgrades (Psi-Energy Control flips them), the signature
// cards, the Body Swapped obligation and the Chimera nemesis set.
//
// The engine has no generic double-sided-upgrade message; the aos
// pack's convention is reused: a neutral AddEntityCounter{N: 0} on the
// upgrade is the package-local flip signal, and each side's React
// swaps the entity code to the other side.
package psylocke

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerPsylocke()
	registerPsiUpgrades()
	registerSignatures()
	registerNemesis()
	registerObligation()
}

// psiUpgrades lists the player's in-play Psi-Energy upgrades
// (Psi-Knife 41002a and Psi-Katana 41002b share base code 41002).
func psiUpgrades(g *engine.Game, p *engine.Player) []*engine.Upgrade {
	var out []*engine.Upgrade
	for _, id := range p.Upgrades {
		u := g.Upgrades[id]
		if u == nil || len(u.Code) < 5 || u.Code[:5] != "41002" {
			continue
		}
		out = append(out, u)
	}
	return out
}

// psiCount counts the player's in-play upgrades on one side
// ("a" = Psi-Knife, "b" = Psi-Katana).
func psiCount(g *engine.Game, p *engine.Player, side string) int {
	n := 0
	for _, u := range psiUpgrades(g, p) {
		if u.Code == "41002"+side {
			n++
		}
	}
	return n
}

// flipSignal emits the package-local flip signal for one upgrade.
func flipSignal(u *engine.Upgrade) engine.Message {
	return engine.AddEntityCounter{ID: u.ID, N: 0}
}

// registerPsylocke installs the Psylocke / Betsy Braddock identity
// (41001a/b).
func registerPsylocke() {
	engine.RegisterBehavior("41001", &engine.Behavior{
		// Psi-Energy Control — Interrupt: when you use one of
		// Psylocke's basic powers (THW, ATK, or DEF), flip 1
		// [psi-energy] upgrade.
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil || !p.IsHero() {
				return nil
			}
			used := false
			switch m := msg.(type) {
			case engine.BasicThwart:
				used = m.Player == p.ID
			case engine.BasicAttack:
				used = m.Player == p.ID
			case engine.Defends:
				used = m.Defender == p.ID && !m.Undefended && m.Via == ""
			}
			if !used {
				return nil
			}
			ups := psiUpgrades(g, p)
			if len(ups) == 0 {
				return nil
			}
			if len(ups) == 1 {
				g.Logf("Psi-Energy Control — %s flips", ups[0].EDef().Name)
				return []engine.Message{flipSignal(ups[0])}
			}
			var choices []engine.Choice
			for _, u := range ups {
				choices = append(choices, engine.Choice{
					Label: "Flip " + u.EDef().Name, Kind: engine.ChoiceCard, CardCode: u.Code, SourceID: u.ID,
				}.Msgs(flipSignal(u)))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Psi-Energy Control — flip 1 Psi-Energy upgrade", choices...),
			}}
		},
	})
}

// registerPsiUpgrades installs both sides of the permanent double-sided
// Psi-Knife / Psi-Katana upgrade. The printed "Hero Resource: exhaust →
// generate a resource; you may flip this card" is approximated: the
// resource is generated via the declarative Resource hook (the payment
// channel carries no per-use notification), so the optional flip on
// resource use is not modeled — flips happen through Psi-Energy
// Control and Body Swapped.
func registerPsiUpgrades() {
	// 41002a Psi-Knife: +1 THW; hero resource [mental].
	engine.RegisterBehavior("41002a", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{THW: 1}
		},
		Resource: &engine.ResourceAbility{Icon: "mental", HeroOnly: true},
		React:    psiFlipReact("41002a", "41002b"),
	})
	// 41002b Psi-Katana: +1 ATK and piercing on basic attacks
	// (piercing is not modeled); hero resource [physical].
	engine.RegisterBehavior("41002b", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{ATK: 1}
		},
		Resource: &engine.ResourceAbility{Icon: "physical", HeroOnly: true},
		React:    psiFlipReact("41002b", "41002a"),
	})
}

// psiFlipReact swaps the upgrade's code to the other side when the
// flip signal lands on it.
func psiFlipReact(self, other string) func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
	return func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.AddEntityCounter)
		if !ok || m.ID != e.EID() || m.N != 0 {
			return nil
		}
		u := g.Upgrades[e.EID()]
		if u == nil || u.Code != self {
			return nil
		}
		u.Code = other
		g.Logf("%s flips to %s", u.EDef().Name, other)
		return nil
	}
}

// registerSignatures installs Psylocke's signature cards.
func registerSignatures() {
	registerAngelAlly()
	registerFlurryOfBlades()
	registerMentalDetection()
	registerPsionicRedirect()
	registerTelepathicSuggestion()
	registerTrainingRegimen()
	registerMartialArtsTraining()
	registerPsionicTraining()
	registerWeaponsTraining()
}

// 41003 Angel (ally): Response — after you play Angel from your hand,
// ready your identity. (Approximation: any enters-play triggers it.)
func registerAngelAlly() {
	engine.RegisterBehavior("41003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
		},
	})
}

// 41004 Flurry of Blades: Hero Action (attack) — deal 2 damage to an
// enemy; for each Psi-Knife confuse an enemy; for each Psi-Katana deal
// 2 damage to an enemy. (Approximation: the per-katana damage is one
// combined hit of 2 per katana on a single chosen enemy.)
func registerFlurryOfBlades() {
	engine.RegisterBehavior("41004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			knives := psiCount(g, p, "a")
			katanas := psiCount(g, p, "b")

			// Last step: per-katana damage.
			var katanaQ *engine.Question
			if katanas > 0 {
				dmgChoices := cardutil.EnemyChoices(g, 2*katanas, pid, func(id engine.EntityID) []engine.Message {
					return []engine.Message{engine.DamageEntity{Target: id, Damage: 2 * katanas, Source: pid}}
				})
				if len(dmgChoices) > 0 {
					katanaQ = engine.Ask("Flurry of Blades — Psi-Katana damage to which enemy?", dmgChoices...)
				}
			}
			// Middle step: per-knife confuses (choose up to N enemies).
			var knifeQ *engine.Question
			if knives > 0 {
				confChoices := cardutil.EnemyChoices(g, 0, pid, func(id engine.EntityID) []engine.Message {
					return []engine.Message{engine.ConfuseEntity{Target: id}}
				})
				if len(confChoices) > 0 {
					if katanaQ != nil {
						for i := range confChoices {
							confChoices[i] = confChoices[i].WithThen(katanaQ)
						}
					}
					knifeQ = engine.AskN("Flurry of Blades — confuse enemies (one per Psi-Knife)", knives, confChoices...)
				}
			}
			// First step: 2 damage to an enemy.
			choices := cardutil.EnemyChoices(g, 2, pid, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: 2, Source: pid}}
			})
			if len(choices) == 0 {
				return nil
			}
			next := knifeQ
			if next == nil {
				next = katanaQ
			}
			if next != nil {
				for i := range choices {
					choices[i] = choices[i].WithThen(next)
				}
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Flurry of Blades — deal 2 damage to an enemy", choices...),
			}}
		},
	})
}

// 41005 Mental Detection: Hero Action (thwart) — remove 1 threat from
// a scheme, +2 per Psi-Knife; draw 1 card per Psi-Katana.
func registerMentalDetection() {
	engine.RegisterBehavior("41005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			n := 1 + 2*psiCount(g, p, "a")
			draw := psiCount(g, p, "b")
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				msgs := []engine.Message{engine.ThwartScheme{Scheme: id, N: n, Source: pid}}
				if draw > 0 {
					msgs = append(msgs, engine.DrawCards{Player: pid, N: draw})
				}
				choices = append(choices, engine.Choice{
					Label: s.EDef().Name, Kind: engine.ChoiceTarget, SourceID: id, CardCode: s.ECode(),
				}.Msgs(msgs...))
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask("Mental Detection — remove threat from a scheme", choices...),
			}}
		},
	})
}

// 41006 Psionic Redirect: Hero Interrupt (defense) — prevent 2 damage
// from an enemy attack, +2 per Psi-Katana; confuse that enemy if you
// control a Psi-Knife. (One confuse total, not per knife.)
func registerPsionicRedirect() {
	engine.RegisterBehavior("41006", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			if !p.IsHero() {
				return engine.Defends{}, nil, false
			}
			prevent := 2 + 2*psiCount(g, p, "b")
			var extra []engine.Message
			if psiCount(g, p, "a") > 0 {
				extra = append(extra, engine.ConfuseEntity{Target: against})
			}
			d := engine.Defends{Defender: p.ID, Against: against, Undefended: true, ExtraPrevent: prevent}
			return d, extra, true
		},
	})
}

// 41007 Telepathic Suggestion: Hero Interrupt — cancel a revealed
// encounter card's When Revealed; per Psi-Katana deal 2 damage to an
// enemy; per Psi-Knife remove 1 threat from a scheme. (Approximation:
// hooks the treachery interrupt window, so only treacheries can be
// cancelled; the per-side riders are one combined choice each.)
func registerTelepathicSuggestion() {
	engine.RegisterBehavior("41007", &engine.Behavior{
		TreacheryInterrupt: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if !p.IsHero() {
				return nil
			}
			var msgs []engine.Message
			if katanas := psiCount(g, p, "b"); katanas > 0 {
				dmgChoices := cardutil.EnemyChoices(g, 2*katanas, p.ID, func(id engine.EntityID) []engine.Message {
					return []engine.Message{engine.DamageEntity{Target: id, Damage: 2 * katanas, Source: p.ID}}
				})
				if len(dmgChoices) > 0 {
					msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(
						"Telepathic Suggestion — Psi-Katana damage to which enemy?", dmgChoices...)})
				}
			}
			if knives := psiCount(g, p, "a"); knives > 0 {
				thwChoices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
					return []engine.Message{engine.ThwartScheme{Scheme: id, N: knives, Source: p.ID}}
				})
				if len(thwChoices) > 0 {
					msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(
						"Telepathic Suggestion — remove threat from which scheme?", thwChoices...)})
				}
			}
			if msgs == nil {
				msgs = []engine.Message{} // cancel only
			}
			return msgs
		},
	})
}

// 41008 Training Regimen: Action — exhaust → search your deck for a
// [skill] card and add it to your hand (shuffle); in hero form,
// discard 1 card from your hand.
func registerTrainingRegimen() {
	engine.RegisterBehavior("41008", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil {
				return nil
			}
			return []engine.Ability{{
				Label:   "Training Regimen — search your deck for a Skill card",
				Type:    engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(s.Owner)
					if p == nil {
						return nil
					}
					var choices []engine.Choice
					seen := map[string]bool{}
					for _, c := range p.Deck {
						if !c.Def().HasTrait("skill") || seen[c.Code] {
							continue
						}
						seen[c.Code] = true
						take := []engine.Message{
							engine.TakeDeckCard{Player: p.ID, CardID: c.ID},
							engine.ShufflePlayerDeck{Player: p.ID},
						}
						ch := engine.Choice{
							Label: "Take " + c.Def().Name, Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(take...)
						if p.IsHero() && len(p.Hand) > 0 {
							var discards []engine.Choice
							for _, h := range p.Hand {
								discards = append(discards, engine.Choice{
									Label: "Discard " + h.Def().Name, Kind: engine.ChoiceCard, CardCode: h.Code,
								}.Msgs(engine.DiscardCards{Player: p.ID, Cards: engine.CardList{h}}))
							}
							ch = ch.WithThen(engine.Ask("Training Regimen — discard 1 card from your hand", discards...))
						}
						choices = append(choices, ch)
					}
					if len(choices) == 0 {
						return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
					}
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask("Training Regimen — take a Skill card from your deck", choices...),
					}}
				},
			}}
		},
	})
}

// 41009 Martial Arts Training: +1 DEF. Hero Response — after Psylocke
// defends against an attack, discard this → ready Psylocke.
func registerMartialArtsTraining() {
	engine.RegisterBehavior("41009", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{DEF: 1}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.Defends)
			if !ok {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil || !p.IsHero() || m.Defender != p.ID || m.Undefended || m.Via != "" {
				return nil
			}
			g.Logf("Martial Arts Training — %s readies", p.Name)
			return []engine.Message{
				engine.DiscardControlled{Player: p.ID, ID: u.ID},
				engine.ReadyEntity{ID: p.ID},
			}
		},
	})
}

// 41010 Psionic Training: Psylocke ignores guard and patrol (not
// modeled). Hero Response — after Psylocke thwarts, discard this →
// confuse an enemy.
func registerPsionicTraining() {
	engine.RegisterBehavior("41010", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.WindowAfterThwarted)
			if !ok {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil || !p.IsHero() || w.Player != p.ID {
				return nil
			}
			choices := cardutil.EnemyChoices(g, 0, p.ID, func(id engine.EntityID) []engine.Message {
				return []engine.Message{
					engine.DiscardControlled{Player: p.ID, ID: u.ID},
					engine.ConfuseEntity{Target: id},
				}
			})
			if len(choices) == 0 {
				return nil
			}
			g.Logf("Psionic Training — confuse an enemy")
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Psionic Training — discard it to confuse an enemy", choices...),
			}}
		},
	})
}

// 41011 Weapons Training: retaliate 1. Hero Response — after Psylocke
// attacks, discard this → ready each [weapon] upgrade you control.
func registerWeaponsTraining() {
	engine.RegisterBehavior("41011", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{Retaliate: 1}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicAttack)
			if !ok {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil || !p.IsHero() || m.Player != p.ID {
				return nil
			}
			msgs := []engine.Message{engine.DiscardControlled{Player: p.ID, ID: u.ID}}
			for _, id := range p.Upgrades {
				w := g.Upgrades[id]
				if w != nil && w.ID != u.ID && w.EDef().HasTrait("weapon") && w.Exhausted {
					msgs = append(msgs, engine.ReadyEntity{ID: w.ID})
				}
			}
			g.Logf("Weapons Training — weapon upgrades ready")
			return msgs
		},
	})
}

// registerNemesis installs the Psylocke nemesis set
// (psylocke_nemesis): Chimera, Interdimensional Plunder, Psionic
// Illusion and Telekinetic Dragon.
func registerNemesis() {
	// 41026 Chimera: Forced Interrupt — when Chimera activates against
	// you, she gets +X SCH and +X ATK for this activation, X = the
	// number of [mental] resources on cards you control.
	// (Approximation: the bump is applied to the printed stats and is
	// not reverted after the activation.)
	engine.RegisterBehavior("41026", &engine.Behavior{
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
			x := mentalResourcesControlled(g, p)
			if x <= 0 {
				return nil
			}
			mn.AttackVal += x
			mn.SchemeVal += x
			g.Logf("Chimera — +%d ATK/+x SCH for this activation", x)
			return nil
		},
	})

	// 41027 Interdimensional Plunder: When Revealed — place 1 threat
	// here for each upgrade in play.
	engine.RegisterBehavior("41027", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s, ok := e.(*engine.SideScheme)
			if !ok {
				return nil
			}
			n := 0
			for _, p := range g.Players {
				n += len(p.Upgrades)
			}
			if n > 0 {
				s.Threat += n
				if s.Threat > s.MaxThreat {
					s.Threat = s.MaxThreat
				}
				g.Logf("Interdimensional Plunder — +%d threat (upgrades in play)", n)
			}
			return nil
		},
	})

	// 41028 Psionic Illusion: attach to your identity; the attack
	// redirection guessing game is not modeled.
	engine.RegisterBehavior("41028", &engine.Behavior{})

	// 41029 Telekinetic Dragon: When Revealed — take X indirect damage,
	// X = [mental] resources on cards you control; surge when X is 0.
	// (Approximation: indirect damage lands on the identity. Boost:
	// "spend [mental] or confuse your identity" — only the confuse
	// branch is modeled, against the first player.)
	engine.RegisterBehavior("41029", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			x := mentalResourcesControlled(g, p)
			if x == 0 {
				g.Logf("Telekinetic Dragon — no mental resources; surge")
				return []engine.Message{engine.RevealNextEncounter{Player: p.ID}}
			}
			g.Logf("Telekinetic Dragon — %d damage to %s", x, p.Name)
			return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: x, Source: t.ID}}
		},
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			return []engine.Message{engine.ConfuseEntity{Target: p.ID}}
		},
	})
}

// mentalResourcesControlled counts printed [mental] resource icons on a
// player's in-play cards (allies, supports, upgrades).
func mentalResourcesControlled(g *engine.Game, p *engine.Player) int {
	n := 0
	count := func(code string) {
		for _, r := range engine.DB.MustLookup(code).Resources {
			if r == "mental" {
				n++
			}
		}
	}
	for _, id := range p.Allies {
		if a := g.Allies[id]; a != nil {
			count(a.Code)
		}
	}
	for _, id := range p.Supports {
		if s := g.Supports[id]; s != nil {
			count(s.Code)
		}
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil {
			count(u.Code)
		}
	}
	return n
}

// registerObligation installs Body Swapped (41025): when revealed, flip
// each psi-energy upgrade to its Psi-Katana side and exhaust it. The
// lingering "cannot flip Psi-Katana upgrades" is not modeled
// (obligations resolve immediately); the removal cost is a [psionic]
// card from hand instead of the printed alter-ego action.
func registerObligation() {
	engine.RegisterBehavior("41025", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var msgs []engine.Message
			for _, id := range p.Upgrades {
				u := g.Upgrades[id]
				if u == nil || len(u.Code) < 5 || u.Code[:5] != "41002" {
					continue
				}
				u.Code = "41002b"
				msgs = append(msgs, engine.ExhaustEntity{ID: u.ID})
			}
			var choices []engine.Choice
			for _, c := range p.Hand {
				if !c.Def().HasTrait("psionic") {
					continue
				}
				choices = append(choices, engine.Choice{
					Label: "Discard " + c.Def().Name + " → remove Body Swapped from the game",
					Kind:  engine.ChoiceCard, CardCode: c.Code,
				}.Msgs(
					engine.DiscardCards{Player: p.ID, Cards: engine.CardList{c}},
					engine.ObligationResolve{Player: p.ID, Card: card, Remove: true},
				))
			}
			choices = append(choices, engine.Choice{
				ID: "discard", Label: "Discard Body Swapped", Kind: engine.ChoicePass,
			}.Msgs(engine.ObligationResolve{Player: p.ID, Card: card}))
			msgs = append(msgs, engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask("Body Swapped — discard a Psionic card to remove it, or discard it", choices...),
			})
			return msgs
		},
	})
}
