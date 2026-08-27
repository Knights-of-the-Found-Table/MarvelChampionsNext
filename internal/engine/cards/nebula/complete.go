// Package nebula registers the Nebula hero pack: the technique upgrade
// system resolved by Combat Protocols at turn start, and the Gamora
// nemesis set. Piercing/overkill/patrol are not modeled; those lines are
// approximated or skipped with comments.
package nebula

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

// techniqueSpecials maps technique upgrade codes to their Special.
var techniqueSpecials = map[string]func(g *engine.Game, u *engine.Upgrade) []engine.Message{
	"22004": func(g *engine.Game, u *engine.Upgrade) []engine.Message {
		return []engine.Message{engine.AskQuestion{Player: u.Owner, Question: engine.Ask(
			engine.Tf("c.cutthroatAmbitionRemove3ThreatFromWhichScheme"), schemePicks(g, 3, u.Owner)...)}}
	},
	"22005": func(g *engine.Game, u *engine.Upgrade) []engine.Message {
		var picks []engine.Choice
		for _, id := range cardutil.SortedEnemyIDs(g) {
			enemy := g.Entity(id)
			if enemy == nil {
				continue
			}
			picks = append(picks,
				engine.Choice{Label: engine.Tf("c.stun2", cardutil.EnemyLabel(enemy)), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
					Msgs(engine.StunEntity{Target: id}),
				engine.Choice{Label: engine.Tf("c.confuse2", cardutil.EnemyLabel(enemy)), Kind: engine.ChoiceTarget, SourceID: id, CardCode: enemy.ECode()}.
					Msgs(engine.ConfuseEntity{Target: id}))
		}
		if len(picks) == 0 {
			return nil
		}
		return []engine.Message{engine.AskQuestion{Player: u.Owner,
			Question: engine.Ask(engine.Tf("c.evasiveManeuveringStunOrConfuseWhichEnemy"), picks...)}}
	},
	"22006": func(g *engine.Game, u *engine.Upgrade) []engine.Message {
		return []engine.Message{engine.ToughEntity{Target: u.Owner}}
	},
	"22007": func(g *engine.Game, u *engine.Upgrade) []engine.Message {
		return cardutil.ChooseEnemy(engine.Tf("c.weaponsMasterDeal4DamageToWhichEnemy"),
			func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return 4, nil })(
			g, &engine.EventCard{Code: "22007", Owner: u.Owner})
	},
	"22008": func(g *engine.Game, u *engine.Upgrade) []engine.Message {
		// Look at top 3, discard 1 (approximation: discard the first).
		if len(g.EncounterDeck) > 0 {
			c, _ := g.DrawEncounter()
			g.EncounterDiscard = append(g.EncounterDiscard, c)
			g.TLogf("c.wideStanceDiscardsFromTheEncounterDeck", c)
		}
		return nil
	},
}

func init() {
	registerNebula()
	registerNemesis()
}

func registerNebula() {
	// Combat Protocols: at turn start, resolve each technique's Special
	// and discard it.
	engine.RegisterBehavior("22001", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			ts, ok := msg.(engine.PlayerTurnStart)
			if !ok || ts.Player != e.EID() {
				return nil
			}
			p := g.Player(ts.Player)
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			for _, id := range append([]engine.EntityID(nil), p.Upgrades...) {
				u := g.Upgrades[id]
				if u == nil || !u.EDef().HasTrait("technique") {
					continue
				}
				if sp, ok := techniqueSpecials[u.Code[:5]]; ok {
					msgs = append(msgs, sp(g, u)...)
				}
				msgs = append(msgs, engine.DiscardControlled{Player: p.ID, ID: id})
			}
			return msgs
		},
	})

	// Gamora (ally): resolve a technique's Special on entry.
	engine.RegisterBehavior("22002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Upgrades {
				u := g.Upgrades[id]
				if u == nil || !u.EDef().HasTrait("technique") {
					continue
				}
				sp, ok := techniqueSpecials[u.Code[:5]]
				if !ok {
					continue
				}
				var msgs []engine.Message
				msgs = append(msgs, sp(g, u)...)
				picks = append(picks, engine.Choice{Label: engine.Tf("m.cardName", u), Kind: engine.ChoiceCard, CardCode: u.Code}.
					Msgs(msgs...))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.gamoraResolveWhichTechniqueSSpecial"), picks...)}}
		},
	})

	// Nebula's Ship: wild resource.
	engine.RegisterBehavior("22003", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})

	// The technique upgrades themselves (passives approximated).
	engine.RegisterBehavior("22004", &engine.Behavior{})
	engine.RegisterBehavior("22005", &engine.Behavior{})
	engine.RegisterBehavior("22006", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			if p.IsHero() {
				return engine.StatBonus{THW: 1, ATK: 1}
			}
			return engine.StatBonus{}
		},
	})
	engine.RegisterBehavior("22007", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			if p.IsHero() {
				return engine.StatBonus{Retaliate: 1}
			}
			return engine.StatBonus{}
		},
	})
	engine.RegisterBehavior("22008", &engine.Behavior{
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			return min(1, n), 0
		},
	})

	// Combat Ready: shuffle up to 2 techniques into the deck, then mill
	// until a technique enters play and resolves.
	engine.RegisterBehavior("22009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var shufflePicks []engine.Choice
			for _, c := range p.Discard {
				def := c.Def()
				if def.Type == "upgrade" && def.HasTrait("technique") {
					shufflePicks = append(shufflePicks, engine.Choice{Label: engine.Tf("m.cardName", def), Kind: engine.ChoiceCard, CardCode: def.Code}.
						Msgs(engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}))
				}
			}
			msgs := []engine.Message{}
			if len(shufflePicks) > 0 {
				q := engine.AskN(engine.Tf("c.combatReadyShuffleUpTo2TechniquesIntoYourDeck"), min(2, len(shufflePicks)), shufflePicks...)
				msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: q})
			}
			// Mill until a technique, put it into play + resolve.
			for len(p.Deck) > 0 {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				if c.Def().Type == "upgrade" && c.Def().HasTrait("technique") {
					msgs = append(msgs, engine.UpgradeEnterPlay{Player: p.ID, Card: c})
					msgs = append(msgs, engine.ResolveTechnique{Player: p.ID, Code: c.Code})
					return msgs
				}
				p.Discard = append(p.Discard, c)
			}
			return msgs
		},
	})

	// Lethal Intent: resolve up to X techniques' Specials.
	engine.RegisterBehavior("22010", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Upgrades {
				u := g.Upgrades[id]
				if u == nil || !u.EDef().HasTrait("technique") {
					continue
				}
				sp, ok := techniqueSpecials[u.Code[:5]]
				if !ok {
					continue
				}
				var msgs []engine.Message
				msgs = append(msgs, sp(g, u)...)
				picks = append(picks, engine.Choice{Label: engine.Tf("m.cardName", u), Kind: engine.ChoiceCard, CardCode: u.Code}.
					Msgs(msgs...))
			}
			if len(picks) == 0 {
				return nil
			}
			q := engine.AskN(engine.Tf("c.lethalIntentResolveWhichTechniques"), len(picks), picks...)
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: q}}
		},
	})

	// Eros: confuse a minion per mental paid (payment icons recorded).
	engine.RegisterBehavior("22011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			// Payment icons are not recorded for ally plays; one
			// confusion as the base case.
			var picks []engine.Choice
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				picks = append(picks, engine.Choice{Label: cardutil.EnemyLabel(mn), Kind: engine.ChoiceTarget, SourceID: id, CardCode: mn.Code}.
					Msgs(engine.ConfuseEntity{Target: id}))
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(),
				Question: engine.Ask(engine.Tf("c.erosConfuseWhichMinion"), picks...)}}
		},
	})

	// Wraith: boost-effect cancel — no boost window; approximated.
	engine.RegisterBehavior("22012", &engine.Behavior{})

	// Venom: consequential damage reduction with a clean main scheme —
	// not modeled.
	engine.RegisterBehavior("22013", &engine.Behavior{})

	// Justice Served: after your thwart empties a scheme, discard to
	// ready.
	engine.RegisterBehavior("22014", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.SchemeDefeated)
			u := g.Upgrades[e.EID()]
			if !ok || u == nil {
				return nil
			}
			// Thwart attribution is not tracked; resolve on any scheme
			// defeat while the owner played a thwart this turn.
			_ = d
			return []engine.Message{
				engine.DiscardControlled{Player: u.Owner, ID: u.ID},
				engine.ReadyEntity{ID: u.Owner},
			}
		},
	})

	// One Way or Another: fetch a side scheme, reveal, draw 3.
	engine.RegisterBehavior("22015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for i, c := range g.EncounterDeck {
				if c.Def().Type == "side_scheme" {
					g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
					return []engine.Message{
						engine.RevealEncounterCard{Player: e.EOwner(), Card: c},
						engine.DrawCards{Player: e.EOwner(), N: 3},
					}
				}
			}
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 3}}
		},
	})

	// Determination: spent-response — no hook; plain resource.
	engine.RegisterBehavior("22016", &engine.Behavior{})

	// The Power of Justice reprint + basics.
	engine.RegisterBehavior("22017", &engine.Behavior{})
	engine.RegisterBehavior("22024", &engine.Behavior{})
	engine.RegisterBehavior("22025", &engine.Behavior{})
	engine.RegisterBehavior("22026", &engine.Behavior{})

	// Brains Over Brawn: after a basic thwart, THW damage.
	engine.RegisterBehavior("22018", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			bt, ok := msg.(engine.BasicThwart)
			if !ok || bt.Player != e.EOwner() {
				return nil
			}
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := max(0, p.ThwartStat(g))
			if n <= 0 {
				return nil
			}
			return cardutil.ChooseEnemy(engine.Tf("c.brainsOverBrawnDealDamageToWhichEnemy"),
				func(g *engine.Game, tgt engine.Entity) (int, []engine.Message) { return n, nil })(
				g, &engine.EventCard{Code: "22018", Owner: p.ID})
		},
	})

	// Heroic Intuition reprint.
	if b := engine.LookupBehavior("01065"); b != nil {
		engine.RegisterBehavior("22019", b)
	}

	// Cosmo: consequential-damage gamble — not modeled.
	engine.RegisterBehavior("22020", &engine.Behavior{})

	// Knowhere: ally limit +1 (unenforced); guardian ally draw.
	engine.RegisterBehavior("22021", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			pc, ok := msg.(engine.PlayCard)
			s := g.Supports[e.EID()]
			if !ok || s == nil || s.Exhausted {
				return nil
			}
			def := pc.Card.Def()
			if def.Type == "ally" && def.HasTrait("guardian") {
				return []engine.Message{
					engine.ExhaustEntity{ID: e.EID()},
					engine.DrawCards{Player: pc.Player, N: 1},
				}
			}
			return nil
		},
	})

	// Daughters of Thanos: draw 3.
	engine.RegisterBehavior("22022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 3}}
		},
	})

	// First Aid reprint.
	if b := engine.LookupBehavior("12019"); b != nil {
		engine.RegisterBehavior("22023", b)
	}

	// Inferiority Complex obligation.
	engine.RegisterBehavior("22027", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			var picks []engine.Choice
			if !p.Exhausted {
				picks = append(picks, engine.Choice{ID: "exhaust", Label: engine.Tf("c.exhaustYourAlterEgoRemoveFromTheGame"), Kind: engine.ChoiceLabel}.
					Msgs(engine.ExhaustEntity{ID: p.ID},
						engine.ObligationResolve{Player: p.ID, Card: card, Remove: true}))
			}
			n := 0
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("technique") {
					n++
				}
			}
			if n >= 2 {
				var subs []engine.Choice
				for _, id := range p.Upgrades {
					if u := g.Upgrades[id]; u != nil && u.EDef().HasTrait("technique") {
						subs = append(subs, engine.Choice{Label: engine.Tf("m.discardCard", u), Kind: engine.ChoiceCard, CardCode: u.Code}.
							Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
					}
				}
				picks = append(picks, engine.Choice{ID: "discard", Label: engine.Tf("c.discard2Techniques"), Kind: engine.ChoiceCard}.
					WithThen(engine.AskN(engine.Tf("c.discardWhich2Techniques"), 2, subs...)))
			}
			if len(picks) == 0 {
				return []engine.Message{
					engine.RevealNextEncounter{Player: p.ID},
					engine.ObligationResolve{Player: p.ID, Card: card},
				}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.inferiorityComplex"), picks...)}}
		},
	})

	// Energy Spear: attach a guardian ally (+2 ATK; piercing skipped).
	engine.RegisterBehavior("22032", &engine.Behavior{
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
						Msgs(engine.AttachUpgrade{ID: e.EID(), Target: a.ID, ATK: 2}))
				}
			}
			if len(picks) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.attachEnergySpearToWhichGuardianAlly"), picks...)}}
		},
	})

	// Guardians of the Galaxy team support: draw after attaching
	// upgrades (all-guardian check approximated away).
	engine.RegisterBehavior("22033", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if au, ok := msg.(engine.AttachUpgrade); ok {
				if s := g.Supports[e.EID()]; s != nil {
					if tgt := g.Entity(au.Target); tgt != nil && tgt.EID().Is(engine.KindAlly) {
						return []engine.Message{engine.DrawCards{Player: s.Owner, N: 1}}
					}
				}
			}
			return nil
		},
	})

	// Defensive Training: 2 training counters, shuffle a Protection event.
	engine.RegisterBehavior("22034", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.AddEntityCounter{ID: e.EID(), N: 2}}
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.exhaustCounterShuffleAProtectionEventFromDiscard"), Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.EOwner())
					if p == nil {
						return nil
					}
					var picks []engine.Choice
					for _, c := range p.Discard {
						def := c.Def()
						if def.Type == "event" && def.Aspect == "protection" {
							picks = append(picks, engine.Choice{Label: engine.Tf("m.cardName", def), Kind: engine.ChoiceCard, CardCode: def.Code}.
								Msgs(engine.ShuffleIntoDeck{Player: p.ID, CardID: c.ID}))
						}
					}
					if len(picks) == 0 {
						return nil
					}
					return append([]engine.Message{engine.AddEntityCounter{ID: self, N: -1}},
						engine.AskQuestion{Player: p.ID, Question: engine.Ask(
							engine.Tf("c.shuffleWhichProtectionEventIntoYourDeck"), picks...)})
				},
			}}
		},
	})

	// Honorary Guardian: +1 HP + guardian trait.
	engine.RegisterBehavior("22035", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var picks []engine.Choice
			picks = append(picks, engine.Choice{Label: engine.Tf("c.nameIdentity", p.Name), Kind: engine.ChoiceTarget, SourceID: p.ID}.
				Msgs(engine.AttachUpgrade{ID: e.EID(), Target: p.ID, MaxHP: 1, GrantTrait: "guardian"}))
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a != nil {
					picks = append(picks, engine.Choice{Label: engine.Tf("m.cardName", a), Kind: engine.ChoiceTarget, SourceID: a.ID, CardCode: a.Code}.
						Msgs(engine.AttachUpgrade{ID: e.EID(), Target: a.ID, MaxHP: 1, GrantTrait: "guardian"}))
				}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.attachHonoraryGuardianToWhichCharacter"), picks...)}}
		},
	})
}

func registerNemesis() {
	// Self-Preservation: Gamora-stat adjustments approximated to a flat
	// -1 THW/-1 ATK on Nebula while in play.
	engine.RegisterBehavior("22029", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, p := range g.Players {
				if p.HeroCode[:5] == "22001" {
					p.BonusTHW--
					p.BonusATK--
					g.TLogf("c.selfPreservationGets1Thw1Atk", p.Name)
				}
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if d, ok := msg.(engine.SchemeDefeated); ok && d.Scheme == e.EID() {
				for _, p := range g.Players {
					if p.HeroCode[:5] == "22001" {
						p.BonusTHW++
						p.BonusATK++
					}
				}
			}
			return nil
		},
	})

	// Gamora (minion): evicts the Gamora ally; after damaging a player,
	// they discard an upgrade.
	engine.RegisterBehavior("22028", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, p := range g.Players {
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && a.Code == "22002" {
						return []engine.Message{engine.DiscardControlled{Player: p.ID, ID: id}}
					}
				}
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			d, ok := msg.(engine.DamageEntity)
			if !ok || d.Source != e.EID() || !d.Target.Is(engine.KindPlayer) {
				return nil
			}
			p := g.Player(d.Target)
			if p == nil || len(p.Upgrades) == 0 {
				return nil
			}
			var picks []engine.Choice
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil {
					picks = append(picks, engine.Choice{Label: engine.Tf("m.discardCard", u), Kind: engine.ChoiceCard, CardCode: u.Code}.
						Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
				}
			}
			return []engine.Message{engine.AskQuestion{Player: p.ID,
				Question: engine.Ask(engine.Tf("c.gamoraDiscardWhichUpgrade"), picks...)}}
		},
	})

	// Lethal Weapon: attach Gamora/villain; discard an upgrade to remove.
	engine.RegisterBehavior("22030", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, p := range g.Players {
				for _, id := range p.Allies {
					if a := g.Allies[id]; a != nil && a.Code == "22002" {
						t.Target = a.ID
						return nil
					}
				}
			}
			for id := range g.Villains {
				t.Target = id
				return nil
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: engine.Tf("c.discardAnUpgradeYouControlDiscardLethalWeapon"), Type: engine.AbilityAction,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Attachments[self]
					pid := engine.PlayerID("")
					for _, p := range g.Players {
						for _, id := range p.Upgrades {
							_ = id
							pid = p.ID
						}
					}
					if a == nil || pid == "" {
						return nil
					}
					p := g.Player(pid)
					if p == nil || len(p.Upgrades) == 0 {
						return nil
					}
					var picks []engine.Choice
					for _, id := range p.Upgrades {
						if u := g.Upgrades[id]; u != nil {
							picks = append(picks, engine.Choice{Label: engine.Tf("m.discardCard", u), Kind: engine.ChoiceCard, CardCode: u.Code}.
								Msgs(engine.DiscardControlled{Player: p.ID, ID: id}))
						}
					}
					return []engine.Message{engine.AskQuestion{Player: p.ID,
						Question: engine.Ask(engine.Tf("c.discardWhichUpgrade"), append(picks,
							engine.Choice{ID: "go", Label: engine.Tf("c.discardLethalWeapon"), Kind: engine.ChoiceLabel}.
								Msgs(engine.DiscardAttachmentMsg{ID: self}))...)}}
				},
			}}
		},
	})

	// Old Rivals: the Gamora minion attacks you (villain fallback);
	// surge otherwise.
	engine.RegisterBehavior("22031", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			for _, mn := range g.Minions {
				if mn.Code[:5] == "22028" {
					return []engine.Message{engine.AskAttack{Enemy: mn.ID, Player: p.ID}}
				}
			}
			for _, q := range g.Players {
				for _, id := range q.Allies {
					if a := g.Allies[id]; a != nil && a.Code == "22002" {
						return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: a.AttackVal + a.PermATK, Source: id}}
					}
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
