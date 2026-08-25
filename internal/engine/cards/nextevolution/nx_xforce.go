package nextevolution

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// spawnAllyFor puts an ally card into play under the player (search
// rewards), consuming the card from wherever it currently lives is the
// caller's job.
func spawnAllyFor(g *engine.Game, p *engine.Player, c engine.Card) {
	def := c.Def()
	a := &engine.Ally{
		ID:        g.NextEntityID(engine.KindAlly),
		Code:      def.Code,
		Owner:     p.ID,
		MaxHP:     derefOr(def.HP, 1),
		AttackVal: derefOr(def.Attack, 0),
		ThwartVal: derefOr(def.Thwart, 0),
		Tough:     def.HasKeyword("Toughness"),
	}
	g.Allies[a.ID] = a
	p.Allies = append(p.Allies, a.ID)
	g.TLogf("log.putsIntoPlay", p.Name, def.Name)
	g.Push(engine.AllyEnteredPlay{Ally: a.ID, Player: p.ID})
	if b := engine.LookupBehavior(def.Code); b.OnPlay != nil {
		g.Push(b.OnPlay(g, a)...)
	}
}

// spawnUpgradeFor puts an upgrade card into play under the player.
func spawnUpgradeFor(g *engine.Game, p *engine.Player, c engine.Card) {
	def := c.Def()
	u := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: def.Code, Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)
	g.TLogf("log.putsIntoPlay", p.Name, def.Name)
	g.Push(engine.UpgradeEnterPlay{Player: p.ID, Card: c})
	if b := engine.LookupBehavior(def.Code); b.OnPlay != nil {
		g.Push(b.OnPlay(g, u)...)
	}
}

// spawnSupportFor puts a support card into play under the player.
func spawnSupportFor(g *engine.Game, p *engine.Player, c engine.Card) {
	def := c.Def()
	s := &engine.Support{ID: g.NextEntityID(engine.KindSupport), Code: def.Code, Owner: p.ID}
	g.Supports[s.ID] = s
	p.Supports = append(p.Supports, s.ID)
	g.TLogf("log.putsIntoPlay", p.Name, def.Name)
	if b := engine.LookupBehavior(def.Code); b.OnPlay != nil {
		g.Push(b.OnPlay(g, s)...)
	}
}

// searchDeckDiscard moves the first matching card from deck/discard into
// play under the owner via spawn; returns whether a card was found.
func searchDeckDiscard(g *engine.Game, p *engine.Player, pred func(*data.CardDef) bool, spawn func(*engine.Player, engine.Card)) bool {
	c, zone, ok := firstCardWhere(p, pred)
	if !ok {
		return false
	}
	takeFromZone(p, c, zone)
	spawn(p, c)
	if zone == "deck" {
		g.Push(engine.ShufflePlayerDeck{Player: p.ID})
	}
	return true
}

func registerXForceCards() {
	// 40014 Caliban: mill until an X-FACTOR/X-FORCE/X-MEN ally → to hand.
	engine.RegisterBehavior("40014", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			for i, c := range p.Deck {
				d := c.Def()
				if d.Type != "ally" {
					continue
				}
				if d.HasTrait("X-Factor") || d.HasTrait("X-Force") || d.HasTrait("X-Men") {
					p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
					p.Hand = append(p.Hand, c)
					g.TLogf("c.calibanFindsInTheDeck", d.Name)
					return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: i}}
				}
			}
			// Nothing found: mill the whole deck's worth is wrong; mill 3
			// as a bounded approximation.
			n := 3
			if len(p.Deck) < n {
				n = len(p.Deck)
			}
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: n}}
		},
	})

	// 40015 Fantomex: search for E.V.A. and put it into play.
	engine.RegisterBehavior("40015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			searchDeckDiscard(g, p, func(d *data.CardDef) bool { return d.Code == "40021" },
				func(p *engine.Player, c engine.Card) { spawnSupportFor(g, p, c) })
			return nil
		},
	})

	// 40016 Sunspot: per energy icon used to pay — payment icons for allies
	// are not recorded; flat 1 damage to the villain + engaged minions.
	engine.RegisterBehavior("40016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			for id := range g.Villains {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: p.ID})
				break
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				mn := g.Minions[id]
				if mn != nil && mn.EngagedWith == p.ID {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: p.ID})
				}
			}
			return msgs
		},
	})

	// 40017 Mission Planning: allies take no consequential damage this
	// phase (approximated: this round; flag lives in the engine's
	// consequential damage path).
	engine.RegisterBehavior("40017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.UsedThisRound["mission-planning"] = true
			g.TLogf("c.alliesTakeNoConsequentialDamageMissionPlanning")
			return nil
		},
	})

	// 40018 Call for Backup: each player fetches an ally into play
	// (auto-picked first).
	engine.RegisterBehavior("40018", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, p := range g.Players {
				searchDeckDiscard(g, p, func(d *data.CardDef) bool { return d.Type == "ally" },
					func(p *engine.Player, c engine.Card) { spawnAllyFor(g, p, c) })
			}
			return nil
		},
	})

	// 40019 Lock and Load: fetch a WEAPON upgrade with cost <= 3.
	engine.RegisterBehavior("40019", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, p := range g.Players {
				searchDeckDiscard(g, p, func(d *data.CardDef) bool {
					return d.Type == "upgrade" && d.HasTrait("Weapon") && cardutil.Cost(d) <= 3
				}, func(p *engine.Player, c engine.Card) { spawnUpgradeFor(g, p, c) })
			}
			return nil
		},
	})

	// 40021 E.V.A.: discard when Fantomex leaves; exhaust → remove 1
	// threat / deal 1 damage / heal Fantomex.
	engine.RegisterBehavior("40021", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AllyDestroyed); !ok {
				return nil
			}
			s := g.Supports[e.EID()]
			if s == nil {
				return nil
			}
			for _, id := range g.Player(s.Owner).Allies {
				if a := g.Allies[id]; a != nil && engine.BaseCodeOf(a.Code) == "40015" {
					return nil
				}
			}
			g.Delete(s.ID)
			g.TLogf("c.eVAIsDiscardedFantomexLeftPlay")
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			p := g.Player(s.Owner)
			if p == nil {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.eVARemove1ThreatDeal1DamageOrHealFantomex"), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil {
						return nil
					}
					var choices []engine.Choice
					for _, id := range g.Schemes() {
						sc := g.Entity(id)
						choices = append(choices, engine.Choice{
							ID: "thw-" + id.String(), Label: engine.S("Remove 1 threat from " + sc.EDef().Name), Kind: engine.ChoiceTarget,
						}.Msgs(engine.ThwartScheme{Scheme: id, N: 1, Source: p.ID}))
					}
					for _, id := range cardutil.SortedEnemyIDs(g) {
						enemy := g.Entity(id)
						choices = append(choices, engine.Choice{
							ID: "dmg-" + id.String(), Label: engine.S("Deal 1 damage to " + enemy.EDef().Name), Kind: engine.ChoiceTarget,
						}.Msgs(engine.DamageEntity{Target: id, Damage: 1, Source: p.ID}))
					}
					for _, id := range p.Allies {
						if a := g.Allies[id]; a != nil && engine.BaseCodeOf(a.Code) == "40015" {
							choices = append(choices, engine.Choice{
								ID: "heal", Label: engine.Tf("c.heal1DamageFromFantomex"), Kind: engine.ChoiceTarget,
							}.Msgs(engine.HealEntity{Target: id, N: 1}))
						}
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   p.ID,
						Question: engine.Ask(engine.Tf("c.eVAChoose"), choices...),
					}}
				},
			}}
		},
	})

	// 40022 Uncanny X-Force: while every character you control is X-FORCE,
	// allies get +1 THW (the consequential-damage rebate is approximated
	// away).
	engine.RegisterBehavior("40022", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return applyXForceAura(g, e)
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.AllyEnteredPlay); ok {
				return applyXForceAura(g, e)
			}
			return nil
		},
	})

	// 40023 Mission Leader: -1 cost with a SOLDIER identity; side-scheme
	// defeat → exhaust → each player draws.
	engine.RegisterBehavior("40023", &engine.Behavior{
		CardCost: func(g *engine.Game, p *engine.Player, def *data.CardDef) int {
			if def.Code != "40023" {
				return 0
			}
			if hero, ok := engine.DB.Lookup(p.HeroCode); ok && hero.HasTrait("Soldier") {
				return 1
			}
			return 0
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.SchemeDefeated); !ok || e.EExhausted() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			msgs = append(msgs, engine.ExhaustEntity{ID: u.ID})
			for _, o := range g.Players {
				msgs = append(msgs, engine.DrawCards{Player: o.ID, N: 1})
			}
			g.TLogf("c.missionLeaderEachPlayerDraws1Card")
			return msgs
		},
	})

	// 40024 Deadpool: when he would be defeated, heal 3 + acceleration
	// token instead.
	engine.RegisterBehavior("40024", &engine.Behavior{
		AllyDefeatInterrupt: func(g *engine.Game, a *engine.Ally, destroy func()) []engine.Message {
			a.Damage = 0
			if a.Damage < a.MaxHP-3 {
				a.Damage = a.MaxHP - 3
			}
			g.TLogf("c.deadpoolHeals3DamageInsteadOfBeingDefeatedAccelerationTokenA")
			return []engine.Message{engine.AddAccelerationToken{}}
		},
	})

	// 40025 Deathlok: attach a cost <= 1 upgrade from any discard pile.
	engine.RegisterBehavior("40025", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			for _, p := range g.Players {
				for _, c := range p.Discard {
					d := c.Def()
					if d.Type != "upgrade" || cardutil.Cost(d) > 1 {
						continue
					}
					if _, ok := p.Discard.Remove(c.ID); ok {
						u := &engine.Upgrade{ID: g.NextEntityID(engine.KindUpgrade), Code: c.Code, Owner: a.Owner, AttachTo: a.ID}
						g.Upgrades[u.ID] = u
						if owner := g.Player(a.Owner); owner != nil {
							owner.Upgrades = append(owner.Upgrades, u.ID)
						}
						g.TLogf("c.deathlokSalvages", d.Name)
					}
					return nil
				}
			}
			return nil
		},
	})

	// 40026 Frenemies: hurt Cable & Deadpool, remove 3 threat from two
	// schemes.
	engine.RegisterBehavior("40026", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a != nil && engine.BaseCodeOf(a.Code) == "40024" {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: p.ID})
				}
			}
			for id := range g.Villains {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: p.ID})
				break
			}
			for i, id := range g.Schemes() {
				if i >= 2 {
					break
				}
				sc := g.Entity(id)
				msgs = append(msgs, engine.ThwartScheme{Scheme: id, N: 3, Source: p.ID})
				g.TLogf("c.frenemiesRemoves3ThreatFrom", sc)
			}
			return msgs
		},
	})

	// 40027 Build Support: fetch a support with cost <= 3.
	engine.RegisterBehavior("40027", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, p := range g.Players {
				searchDeckDiscard(g, p, func(d *data.CardDef) bool {
					return d.Type == "support" && cardutil.Cost(d) <= 3
				}, func(p *engine.Player, c engine.Card) { spawnSupportFor(g, p, c) })
			}
			return nil
		},
	})

	// 40028 The Power of the Mind: doubles for PSIONIC payments — handled
	// in the engine's powerOfBonus.
	engine.RegisterBehavior("40028", &engine.Behavior{})

	// 40029 Psimitar: after you play another PSIONIC event, exhaust → 2
	// damage to an enemy.
	engine.RegisterBehavior("40029", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.EventPlayed)
			if !ok || e.EExhausted() || data.BaseCode(m.Card.Code) == "40029" {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.Owner != m.Player || !m.Card.Def().HasTrait("Psionic") {
				return nil
			}
			if len(g.Enemies()) == 0 {
				return nil
			}
			p := g.Player(u.Owner)
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				choices = append(choices, engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: enemy.ECode(),
				}.Msgs(engine.ExhaustEntity{ID: u.ID}, engine.DamageEntity{Target: id, Damage: 2, Source: p.ID}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.psimitarDeal2DamageTo"), choices...),
			}}
		},
	})

	// 40030 Sidearm: attached ally gets +1 ATK (ranged not modeled).
	engine.RegisterBehavior("40030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil || len(p.Allies) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil {
					continue
				}
				choices = append(choices, engine.Choice{
					ID: "ally-" + id.String(), Label: engine.S(a.EDef().Name), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: a.Code,
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id, ATK: 1}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.sidearmAttachTo"), choices...),
			}}
		},
	})

	// 40050 Feral: after she thwarts, discard deck top → damage the villain
	// per icon.
	engine.RegisterBehavior("40050", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyThwartWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			c, _, _ := deckTopIcons(p)
			n := iconCountOf(c)
			var msgs []engine.Message
			msgs = append(msgs, engine.MillPlayerDeck{Player: p.ID, N: 1})
			for id := range g.Villains {
				msgs = append(msgs, engine.DamageEntity{Target: id, Damage: n, Source: p.ID})
				break
			}
			g.TLogf("c.feralDiscardsDamageToTheVillain", c, n)
			return msgs
		},
	})

	// 40051 Wolfsbane: after she thwarts, name a type then discard deck
	// top; a match goes to hand.
	engine.RegisterBehavior("40051", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.AllyThwartWindow)
			if !ok || m.Ally != e.EID() {
				return nil
			}
			a := g.Allies[e.EID()]
			p := g.Player(a.Owner)
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			c, _, _ := deckTopIcons(p)
			var choices []engine.Choice
			for _, typ := range []string{"ally", "event", "support", "upgrade", "resource"} {
				choices = append(choices, engine.Choice{
					ID: "type-" + typ, Label: engine.S(typ), Kind: engine.ChoiceLabel,
				}.Msgs(engine.GuessCheck{Player: p.ID, CardCode: c.Code, Guess: typ}))
			}
			g.TLogf("c.wolfsbaneDiscards", c)
			return []engine.Message{
				engine.MillPlayerDeck{Player: p.ID, N: 1},
				engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.wolfsbaneNameACardType"), choices...)},
			}
		},
	})

	// 40052 Even the Odds: remove 1[per_hero] threat from each side scheme;
	// damage the villain per scheme defeated this way (the energy
	// requirement is approximated by Playable).
	engine.RegisterBehavior("40052", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			for _, c := range p.Hand {
				for _, r := range c.Def().Resources {
					if r == "energy" || r == "wild" {
						return true
					}
				}
			}
			return false
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := len(g.Players)
			var msgs []engine.Message
			defeated := 0
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				s := g.SideSchemes[id]
				if s == nil {
					continue
				}
				if s.Threat <= n {
					defeated++
				}
				msgs = append(msgs, engine.ThwartScheme{Scheme: id, N: n, Source: p.ID})
			}
			if defeated > 0 {
				for id := range g.Villains {
					msgs = append(msgs, engine.DamageEntity{Target: id, Damage: defeated, Source: p.ID})
					break
				}
			}
			return msgs
		},
	})

	// 40053 Team Investigation: remove 3[per_hero] threat from a side
	// scheme (Alliance group payment not modeled).
	engine.RegisterBehavior("40053", &engine.Behavior{
		OnPlay: cardutil.ChooseScheme(engine.Tf("c.teamInvestigationChooseAScheme"), func(g *engine.Game, e engine.Entity) int {
			return 3 * len(g.Players)
		}),
	})

	// 40054 Take Out the Guards: each player discards 1 non-ELITE minion
	// (auto-picks their first).
	engine.RegisterBehavior("40054", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, p := range g.Players {
				for _, id := range cardutil.SortedIDs(g.Minions) {
					mn := g.Minions[id]
					if mn != nil && mn.EngagedWith == p.ID && !mn.EDef().HasTrait("Elite") {
						g.Logf("%s discards %s", p.Name, mn.EDef().Name)
						g.Delete(id)
						g.EncounterDiscard = append(g.EncounterDiscard, engine.Card{ID: g.NextCardID(), Code: mn.Code})
						break
					}
				}
			}
			return nil
		},
	})

	// 40055 Overwatch: attach to a scheme; a thwart on the attached scheme
	// copies the removal onto another scheme.
	engine.RegisterBehavior("40055", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil || len(g.Schemes()) == 0 {
				return nil
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					ID: "sch-" + id.String(), Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.AttachUpgrade{ID: u.ID, Target: id}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.overwatchAttachToWhichScheme"), choices...),
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.ThwartScheme)
			if !ok {
				return nil
			}
			u := g.Upgrades[e.EID()]
			if u == nil || u.AttachTo == "" || u.AttachTo != m.Scheme {
				return nil
			}
			if m.N <= 0 || len(g.Schemes()) < 2 {
				return nil
			}
			p := g.Player(u.Owner)
			if p == nil {
				return nil
			}
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				if id == u.AttachTo {
					continue
				}
				s := g.Entity(id)
				choices = append(choices, engine.Choice{
					ID: "sch-" + id.String(), Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.DiscardControlled{Player: u.Owner, ID: u.ID},
					engine.ThwartScheme{Scheme: id, N: m.N, Source: p.ID}))
			}
			return []engine.Message{engine.AskQuestion{
				Player:   p.ID,
				Question: engine.Ask(engine.Tf("c.overwatchCopyTheThreatRemovalOnto", m.N), choices...),
			}}
		},
	})

	// 40056 Atlas Bear: look at a player deck's top card; events may be
	// bought to hand for 1 damage.
	engine.RegisterBehavior("40056", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			a := g.Allies[e.EID()]
			if a == nil {
				return nil
			}
			p := g.Player(a.Owner)
			if p == nil || len(p.Deck) == 0 {
				return nil
			}
			top := p.Deck[0]
			if top.Def().Type != "event" {
				return nil
			}
			return []engine.Ability{{
				Label: engine.S("Atlas Bear — peek and fetch " + top.Def().Name), Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					a := g.Allies[self]
					p := g.Player(a.Owner)
					if p == nil || len(p.Deck) == 0 {
						return nil
					}
					c := p.Deck[0]
					if c.Def().Type != "event" {
						return nil
					}
					p.Deck = p.Deck[1:]
					p.Hand = append(p.Hand, c)
					g.TLogf("c.atlasBearFetchesToHand1Damage", c)
					return []engine.Message{engine.DamageEntity{Target: a.ID, Damage: 1, Source: p.ID}}
				},
			}}
		},
	})

	// 40057 White Fox: after being discarded from the deck top, enter play.
	engine.RegisterBehavior("40057", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DeckTopDiscarded)
			if !ok || data.BaseCode(m.Card.Code) != "40057" {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			if _, ok := p.Discard.Remove(m.Card.ID); ok {
				spawnAllyFor(g, p, m.Card)
				g.TLogf("c.whiteFoxLeapsOutOfTheDiscardPile")
			}
			return nil
		},
	})

	// 40058 The Posse: heal 1 and ready each POSSE character.
	engine.RegisterBehavior("40058", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			n := 0
			if g.EntityHasTrait(engine.EntityID(p.ID), "Posse") {
				n++
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.EDef().HasTrait("Posse") {
					n++
				}
			}
			return n >= 3
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			for _, id := range p.Allies {
				a := g.Allies[id]
				if a == nil || !a.EDef().HasTrait("Posse") {
					continue
				}
				msgs = append(msgs, engine.HealEntity{Target: id, N: 1}, engine.ReadyEntity{ID: id})
			}
			return msgs
		},
	})

	// 40059 Superpower Training: each player fetches an identity-specific
	// upgrade (same card set as their hero).
	engine.RegisterBehavior("40059", &engine.Behavior{
		SideSchemeDefeated: func(g *engine.Game, s *engine.SideScheme) []engine.Message {
			for _, p := range g.Players {
				heroSet := ""
				if d, ok := engine.DB.Lookup(p.HeroCode); ok {
					heroSet = d.CardSet
				}
				if heroSet == "" {
					continue
				}
				set := heroSet
				searchDeckDiscard(g, p, func(d *data.CardDef) bool {
					return d.Type == "upgrade" && d.CardSet == set
				}, func(p *engine.Player, c engine.Card) { spawnUpgradeFor(g, p, c) })
			}
			return nil
		},
	})

	// 40060 Digging Deep: after discarded from the deck top, back to hand.
	engine.RegisterBehavior("40060", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.DeckTopDiscarded)
			if !ok || data.BaseCode(m.Card.Code) != "40060" {
				return nil
			}
			p := g.Player(m.Player)
			if p == nil {
				return nil
			}
			if _, ok := p.Discard.Remove(m.Card.ID); ok {
				p.Hand = append(p.Hand, m.Card)
				g.TLogf("c.diggingDeepClawsItsWayBackToHand")
			}
			return nil
		},
	})

	// 40061-40063 Energy/Genius/Strength: "Max 1 per deck" is deckbuilding.
	for _, code := range []string{"40061", "40062", "40063"} {
		engine.RegisterBehavior(code, &engine.Behavior{})
	}

	// 40064 Sharpshooter: ranged attacks get luck-boosted damage
	// (approximation: boosts the owner's next basic attack).
	engine.RegisterBehavior("40064", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.BasicAttack)
			if !ok || e.EExhausted() {
				return nil
			}
			u := g.Upgrades[e.EID()]
			p := g.Player(u.Owner)
			if p == nil || m.Player != p.ID || len(p.Deck) == 0 {
				return nil
			}
			c, _, _ := deckTopIcons(p)
			n := iconCountOf(c)
			if n <= 0 {
				return nil
			}
			g.EventDamageBonus[p.ID] += n
			g.TLogf("c.sharpshooterDiscardsDamageOnThisAttack", c, n)
			return []engine.Message{engine.MillPlayerDeck{Player: p.ID, N: 1}}
		},
	})
}

// applyXForceAura grants controlled allies +1 THW while every character
// the owner controls has X-FORCE.
func applyXForceAura(g *engine.Game, e engine.Entity) []engine.Message {
	s := g.Supports[e.EID()]
	if s == nil {
		return nil
	}
	p := g.Player(s.Owner)
	if p == nil {
		return nil
	}
	if !g.EntityHasTrait(engine.EntityID(p.ID), "X-Force") {
		return nil
	}
	for _, id := range p.Allies {
		a := g.Allies[id]
		if a == nil || !a.EDef().HasTrait("X-Force") {
			return nil
		}
	}
	for _, id := range p.Allies {
		a := g.Allies[id]
		if a != nil && a.PermTHW == 0 {
			a.PermTHW = 1
		}
	}
	g.TLogf("c.uncannyXForceAlliesGet1Thw")
	return nil
}
