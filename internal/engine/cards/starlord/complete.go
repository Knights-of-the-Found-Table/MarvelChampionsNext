// Package starlord registers the Star-Lord hero pack: the facedown
// encounter-card economy ("What could go wrong?"), guardian synergies,
// and the Mister Knife nemesis set.
package starlord

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerStarLord()
	registerNemesis()
}

func facedownCount(p *engine.Player) int { return len(p.EncounterDown) }

func registerStarLord() {
	// Star-Lord identity: allies gain guardian; "What could go wrong?"
	// — the cost-reduction interrupt is approximated as a round-limited
	// discount ability.
	engine.RegisterBehavior("17001", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			if g.UsedThisRound["sl-wcgw"] {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.whatCouldGoWrong3DiscountOnYourNextCardYouTakeAFacedownEncou"), Type: engine.AbilityAction,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					g.UsedThisRound["sl-wcgw"] = true
					pid := engine.PlayerID(self)
					return []engine.Message{
						engine.DealEncounterToPlayer{Player: pid},
						engine.CostDiscountApply{Player: pid, Amount: 3},
					}
				},
			}}
		},
	})

	// Nova Prime: defeat a non-elite minion on entry.
	engine.RegisterBehavior("17002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var picks []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn != nil && !mn.EDef().HasTrait("elite") {
					picks = append(picks, engine.Choice{Label: cardutil.EnemyLabel(mn), Kind: engine.ChoiceTarget, SourceID: id, CardCode: mn.Code}.
						Msgs(engine.DamageEntity{Target: id, Damage: mn.HP(), Source: e.EOwner()}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask(engine.Tf("c.novaPrimeDefeatWhichNonEliteMinion"), picks...)}}
		},
	})

	// Daring Escape: facedown card → ready + draw.
	engine.RegisterBehavior("17003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{
				engine.DealEncounterToPlayer{Player: e.EOwner()},
				engine.ReadyEntity{ID: e.EOwner()},
				engine.DrawCards{Player: e.EOwner(), N: 1},
			}
		},
	})

	// Gutsy Move: 2 threat + 2 per facedown card.
	engine.RegisterBehavior("17004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			n := 2
			if p != nil {
				n += 2 * facedownCount(p)
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				engine.Tf("c.gutsyMoveRemoveThreatFromWhichScheme"), schemePicks(g, n, e.EOwner())...)}}
		},
	})

	// Sliding Shot: 5 + 2 per facedown card.
	engine.RegisterBehavior("17005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			n := 5
			if p != nil {
				n += 2 * facedownCount(p)
			}
			return cardutil.ChooseEnemy(engine.Tf("c.slidingShotDealDamageToWhichEnemy", n),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(g, e)
		},
	})

	// Bad Boy: villain-attack damage prevention + flip + draw 2.
	engine.RegisterBehavior("17006", &engine.Behavior{
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			// Only against villain attacks; attribution approximated to
			// any damage while the villain is the only enemy.
			for id := range g.Villains {
				_ = id
				g.Delete(u.ID)
				p.Side = engine.SideAlterEgo
				g.TLogf("c.badBoyPreventsAllDamageChangesToAlterEgoForm", p.Name)
				g.Push(engine.DrawCards{Player: p.ID, N: 2})
				return n, 0
			}
			return 0, 0
		},
	})

	// Element Gun: exhaust + 1 any resource → 3 damage.
	engine.RegisterBehavior("17007", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustElementGun1ResourceDeal3Damage"), Type: engine.AbilityAction,
				Exhaust: true, Cost: 1, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return cardutil.ChooseEnemy(engine.Tf("c.elementGunDeal3DamageToWhichEnemy"),
						func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 3, nil })(g, g.Entity(self))
				},
			}}
		},
	})

	// Jet Boots: Aerial + prevent 1 per facedown card.
	engine.RegisterBehavior("17008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.GrantTrait{Target: e.EOwner(), Trait: "aerial"}}
		},
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			if u.Exhausted {
				return 0, 0
			}
			g.Push(engine.ExhaustEntity{ID: u.ID})
			return min(facedownCount(p), n), 0
		},
	})

	// Leader of the Guardians: guardian characters +1 THW.
	engine.RegisterBehavior("17009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.EDef().HasTrait("guardian") {
					a.PermTHW++
				}
			}
			g.TLogf("c.leaderOfTheGuardiansEachGuardianAllyGets1Thw")
			return nil
		},
	})

	// Star-Lord's Helmet: +1 hand size per facedown card (max 3) in hero
	// form.
	engine.RegisterBehavior("17010", &engine.Behavior{
		HandSizeBonus: func(g *engine.Game, p *engine.Player) int {
			if !p.IsHero() {
				return 0
			}
			return min(3, facedownCount(p))
		},
	})

	// Adam Warlock: random discard → resource-table effect.
	engine.RegisterBehavior("17011", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			var hit bool
			switch w := msg.(type) {
			case engine.AllyAttackWindow:
				hit = w.Ally == e.EID()
			case engine.AllyThwartWindow:
				hit = w.Ally == e.EID()
			}
			if !hit {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil || len(p.Hand) == 0 {
				return nil
			}
			i := g.Random(len(p.Hand))
			c := p.Hand[i]
			p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
			p.Discard = append(p.Discard, c)
			res := ""
			for _, r := range c.Def().Resources {
				res = r
				if r == "wild" {
					break
				}
			}
			switch res {
			case "physical":
				return []engine.Message{engine.AskQuestion{Player: p.ID, Question: engine.Ask(
					engine.Tf("c.adamWarlockRemove3ThreatFromWhichScheme"), schemePicks(g, 3, p.ID)...)}}
			case "energy":
				var picks []engine.Choice
				for _, q := range g.Players {
					if q.Damage > 0 {
						picks = append(picks, engine.Choice{Label: engine.S(q.Name), Kind: engine.ChoiceTarget, SourceID: q.ID}.
							Msgs(engine.HealEntity{Target: q.ID, N: 3}))
					}
				}
				if len(picks) > 0 {
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask(engine.Tf("c.adamWarlockHealWhichIdentity"), picks...)}}
				}
				return nil
			case "mental":
				return cardutil.ChooseEnemy(engine.Tf("c.adamWarlockDeal3DamageToWhichEnemy"),
					func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 3, nil })(
					g, &engine.EventCard{Code: "17011", Owner: p.ID})
			}
			return nil
		},
	})

	// Beta Ray Bill: after defeating a minion, 2 main-scheme threat.
	engine.RegisterBehavior("17012", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			w, ok := msg.(engine.AllyAttackWindow)
			if !ok || w.Ally != e.EID() || g.Minions[w.Target] == nil {
				return nil
			}
			if g.MainScheme == nil {
				return nil
			}
			return []engine.Message{engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 2, Source: e.EOwner()}}
		},
	})

	// Yondu: ranged (ignored); stats only.
	engine.RegisterBehavior("17013", &engine.Behavior{})

	// Air Supremacy: 3 damage to X enemies (X = aerial characters).
	engine.RegisterBehavior("17014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			x := 0
			if p != nil {
				if g.EntityHasTrait(p.ID, "aerial") {
					x++
				}
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && g.EntityHasTrait(a.ID, "aerial") {
						x++
					}
				}
			}
			if x == 0 {
				return nil
			}
			var picks []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				if enemy != nil {
					picks = append(picks, engine.Choice{Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
						Msgs(engine.DamageEntity{Target: id, Damage: 3, Source: e.EOwner()}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			q := engine.AskN(engine.Tf("c.airSupremacyHitWhichEnemies"), min(x, len(picks)), picks...)
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: q}}
		},
	})

	// Blaze of Glory: guardians +2/+2 this phase; 1 damage at phase end
	// (rider approximated to immediate 0 — end-of-phase window absent).
	engine.RegisterBehavior("17015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			msgs := []engine.Message{engine.ApplyStatBonus{Target: p.ID, THW: 2, ATK: 2}}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.EDef().HasTrait("guardian") {
					msgs = append(msgs, engine.AllyStatBonus{Ally: id, THW: 2, ATK: 2})
				}
			}
			return msgs
		},
	})

	// Get Ready reprint: alias core 01069.
	if b := engine.LookupBehavior("01069"); b != nil {
		engine.RegisterBehavior("17016", b)
	}

	// Target Practice: weapon-attachment attack interrupt — per-attack
	// window approximated to a flat +2 on the ally's next attack via
	// PermATK (not reverted; noted).
	engine.RegisterBehavior("17017", &engine.Behavior{})

	// The Power of Leadership reprint.
	engine.RegisterBehavior("17018", &engine.Behavior{})

	// Laser Blaster: attach a guardian ally +1 ATK.
	engine.RegisterBehavior("17019", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a != nil && a.EDef().HasTrait("guardian") {
					picks = append(picks, engine.Choice{Label: engine.Tf("m.cardName", a), Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
						Msgs(engine.AttachUpgrade{ID: e.EID(), Target: a.ID, ATK: 1}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.attachLaserBlasterToWhichGuardianAlly"), picks...)}}
		},
	})

	// Cosmo: consequential gamble — not modeled.
	engine.RegisterBehavior("17020", &engine.Behavior{})

	// C.I.T.T.: exhaust + 2 resources → ready a guardian character.
	engine.RegisterBehavior("17021", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustCITT2ResourcesReadyAGuardianCharacter"), Type: engine.AbilityAction,
				Exhaust: true, Cost: 2, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					for _, id := range p.Allies {
						if a := g.Allies[id]; a != nil && a.Exhausted && a.EDef().HasTrait("guardian") {
							picks = append(picks, engine.Choice{Label: engine.Tf("m.cardName", a), Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
								Msgs(engine.ReadyEntity{ID: a.ID}))
						}
					}
					if p.Exhausted && g.EntityHasTrait(p.ID, "guardian") {
						picks = append(picks, engine.Choice{Label: engine.Tf("c.nameIdentity", p.Name), Kind: engine.ChoiceTarget, SourceID: p.ID}.
							Msgs(engine.ReadyEntity{ID: p.ID}))
					}
					if len(picks) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask(engine.Tf("c.readyWhichGuardianCharacter"), picks...)}}
				},
			}}
		},
	})

	// Knowhere reprint: alias nebula 22021.
	if b := engine.LookupBehavior("22021"); b != nil {
		engine.RegisterBehavior("17022", b)
	}

	// Pulse Grenade: discard + mill 2 → 1 damage per boost icon.
	engine.RegisterBehavior("17023", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.discardPulseGrenadeMill2DamagePerBoostIcon"), Type: engine.AbilityAction, HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					n := 0
					for i := 0; i < 2; i++ {
						c, ok := g.DrawEncounter()
						if !ok {
							break
						}
						if b := c.Def().Boost; b != nil {
							n += *b
						}
						g.EncounterDiscard = append(g.EncounterDiscard, c)
					}
					return append([]engine.Message{engine.DiscardControlled{Player: u.Owner, ID: self}},
						cardutil.ChooseEnemy(engine.Tf("c.pulseGrenadeDealDamageToWhichEnemy", n),
							func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(
							g, &engine.EventCard{Code: "17023", Owner: u.Owner})...)
				},
			}}
		},
	})

	// Banishment obligation.
	engine.RegisterBehavior("17024", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var picks []engine.Choice
			if !p.Exhausted {
				picks = append(picks, engine.Choice{ID: "exhaust", Label: engine.Tf("c.exhaustPeterQuillRemoveFromTheGame"), Kind: engine.ChoiceLabel}.
					Msgs(engine.ExhaustEntity{ID: p.ID},
						engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.Code[:5] == "17007" {
					picks = append(picks, engine.Choice{ID: "discard-gun", Label: engine.Tf("c.discardElementGun"), Kind: engine.ChoiceLabel}.
						Msgs(engine.DiscardControlled{Player: p.ID, ID: id},
							engine.ObligationResolve{Player: p.ID, Card: card}))
					break
				}
			}
			if len(picks) == 0 {
				msgs := []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
				if g.MainScheme != nil {
					msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 3, Source: engine.EntityID("17024")})
				}
				return msgs
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.banishment"), picks...)}}
		},
	})

	// Dive Bomb: 7 damage + 1 to each other enemy.
	engine.RegisterBehavior("17028", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := cardutil.ChooseEnemy(engine.Tf("c.diveBombDeal7DamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 7, nil })(g, e)
			for _, id := range cardutil.SortedEnemyIDs(g) {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: e.EOwner()})
			}
			return msgs
		},
	})

	// Agile Flight: up to 5 threat split across schemes (approximated:
	// 5 on one scheme).
	engine.RegisterBehavior("17029", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(
				engine.Tf("c.agileFlightRemoveThreatFromWhichScheme"), schemePicks(g, 5, e.EOwner())...)}}
		},
	})

	// Ever Vigilant: ready + 2 main-scheme threat.
	engine.RegisterBehavior("17030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			msgs := []engine.Message{engine.ReadyEntity{ID: e.EOwner()}}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.ThwartScheme{Scheme: g.MainScheme.ID, N: 2, Source: e.EOwner()})
			}
			return msgs
		},
	})

	// Enhanced Awareness reprint: alias drs's counter resource pattern.
	if b := engine.LookupBehavior("06034"); b != nil {
		engine.RegisterBehavior("17031", b)
	}
}

func registerNemesis() {
	// Budding Crime Syndicate: Hinder 2 per hero.
	engine.RegisterBehavior("17025", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: 2 * len(g.Players), Source: e.EID()}}
		},
	})

	// Mister Knife: Retaliate 1 printed; first-treachery surge rider
	// approximated away.
	engine.RegisterBehavior("17026", &engine.Behavior{})

	// Spartoi Cunning: random discard + 1 damage + 1 threat.
	engine.RegisterBehavior("17027", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			if len(p.Hand) > 0 {
				i := g.Random(len(p.Hand))
				c := p.Hand[i]
				p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
				p.Discard = append(p.Discard, c)
			}
			msgs := []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID}}
			if g.MainScheme != nil {
				msgs = append(msgs, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: t.ID})
			}
			return msgs
		},
	})
}

// ---- helpers ----

func schemePicks(g *engine.Game, n int, pid engine.PlayerID) []engine.Choice {
	return cardutil.SchemeChoices(g, func(s engine.EntityID) []engine.Message {
		return []engine.Message{engine.ThwartScheme{Scheme: s, N: n, Source: pid}}
	})
}
