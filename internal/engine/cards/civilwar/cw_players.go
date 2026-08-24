package civilwar

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

func init() { registerCWPlayers() }

// deckSearchHand moves the first deck or discard card matching pred into
// the player's hand (search-class approximation), shuffling after a deck
// hit.
func cwSearchDeck(g *engine.Game, p *engine.Player, pred func(*data.CardDef) bool) []engine.Message {
	for i, c := range p.Discard {
		if pred(c.Def()) {
			card := c
			p.Discard = append(p.Discard[:i:i], p.Discard[i+1:]...)
			p.Hand = append(p.Hand, card)
			return []engine.Message{engine.ReturnDiscardCard{Player: p.ID, CardID: card.ID}}
		}
	}
	for i, c := range p.Deck {
		if pred(c.Def()) {
			card := c
			p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
			p.Hand = append(p.Hand, card)
			return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
		}
	}
	return nil
}

func registerCWPlayers() {
	// --- Tigra suite (56002-56009) ---
	// 56002 Moon Knight: drags a minion in stunned and confused.
	engine.RegisterBehavior("56002", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for i, c := range g.EncounterDeck {
				if c.Def().Type == "minion" {
					card := c
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					def := card.Def()
					mn := &engine.Minion{
						ID: g.NextEntityID(engine.KindMinion), Code: card.Code,
						MaxHP: intValue(def.HP, 1), AttackVal: intValue(def.Attack, 0), SchemeVal: intValue(def.Scheme, 0),
						EngagedWith: p.ID, Stunned: true, Confused: true,
					}
					g.Minions[mn.ID] = mn
					g.Logf("Moon Knight drags in %s — stunned and confused", def.Name)
					return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID}, engine.ShuffleEncounterDeck{}}
				}
			}
			return nil
		},
	})

	// 56003 Precinct HQ: alter-ego threat removal scaled by engaged
	// minions.
	engine.RegisterBehavior("56003", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Precinct HQ — remove 1 threat (+1 per engaged minion)", Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.Supports[self].Owner)
					if p == nil {
						return nil
					}
					n := 1
					for _, mn := range g.Minions {
						if mn != nil && mn.EngagedWith == p.ID {
							n++
						}
					}
					var out []engine.Message
					for _, sid := range g.Schemes() {
						out = append(out, engine.ThwartScheme{Scheme: sid, N: n, Source: self})
						break
					}
					return out
				},
			}}
		},
	})

	// 56004 Cat's Head Amulet: physical resource per engaged minion (max
	// 3) — approximated as a single physical resource.
	engine.RegisterBehavior("56004", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "physical"},
	})

	// 56005 Sharp Claws: +1 ATK riders approximated away.
	engine.RegisterBehavior("56005", &engine.Behavior{})

	// 56006 Cat-Like Reflexes: discard to prevent 3 damage.
	engine.RegisterBehavior("56006", &engine.Behavior{
		DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
			if n <= 0 {
				return 0, 0
			}
			g.Delete(u.ID)
			p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
			g.Logf("Cat-Like Reflexes prevent 3 damage")
			prevent := 3
			if prevent > n {
				prevent = n
			}
			return prevent, 0
		},
	})

	// 56007 Hunted: the additional-cost minion grab is approximated by
	// attaching to the first enemy in play.
	engine.RegisterBehavior("56007", &engine.Behavior{})

	// 56008 Tooth and Claw: cost scales down per engaged minion (payment
	// hook not modeled); two 4-damage strikes (two serial target asks).
	engine.RegisterBehavior("56008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			strike := cardutil.ChooseEnemy("Tooth and Claw — first 4 damage", func(g *engine.Game, en engine.Entity) (int, []engine.Message) {
				return 4, nil
			})
			strike2 := cardutil.ChooseEnemy("Tooth and Claw — second 4 damage", func(g *engine.Game, en engine.Entity) (int, []engine.Message) {
				return 4, nil
			})
			return append(strike(g, e), strike2(g, e)...)
		},
	})

	// 56009 Feline Senses: remove 3; stun a minion on the finishing blow.
	engine.RegisterBehavior("56009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var out []engine.Message
			for _, sid := range g.Schemes() {
				out = append(out, engine.ThwartScheme{Scheme: sid, N: 3, Source: e.EID()})
				break
			}
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if mn := g.Minions[id]; mn != nil {
					out = append(out, engine.StunEntity{Target: id})
					break
				}
			}
			return out
		},
	})

	// --- shared cards (56010-56022) ---
	// 56010 Two-Gun Kid: dual-target basic attacks not modeled.
	engine.RegisterBehavior("56010", &engine.Behavior{})
	// 56011 Spider-Girl: stun and confuse a minion on entry.
	engine.RegisterBehavior("56011", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, id := range cardutil.SortedEnemyIDs(g) {
				if mn := g.Minions[id]; mn != nil {
					return []engine.Message{engine.StunEntity{Target: id}, engine.ConfuseEntity{Target: id}}
				}
			}
			return nil
		},
	})
	// 56012 Air Cover: Tactic-fetch response not modeled.
	engine.RegisterBehavior("56012", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Upgrades[e.EID()].Counters = 2
			return nil
		},
	})
	// 56013 Aggressive Conditioning: +3 HP / +1 ATK under any player's
	// control (applies to owner).
	engine.RegisterBehavior("56013", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP += 3
				g.Logf("+3 max hit points")
			}
			return nil
		},
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{ATK: 1}
		},
	})
	// 56014 Suppressing Fire: heal 2 after defeating the attached minion
	// (attachment-on-minion heal window not modeled).
	engine.RegisterBehavior("56014", &engine.Behavior{})
	// 56015 "Bring It!": draw 1 per engaged minion.
	engine.RegisterBehavior("56015", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := 0
			for _, mn := range g.Minions {
				if mn != nil && mn.EngagedWith == p.ID {
					n++
				}
			}
			if n > 0 {
				return []engine.Message{engine.DrawCards{Player: p.ID, N: n}}
			}
			return nil
		},
	})
	// 56016 Coup de Grâce: defeat an upgraded non-Elite minion.
	engine.RegisterBehavior("56016", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			for _, id := range cardutil.SortedEnemyIDs(g) {
				mn := g.Minions[id]
				if mn == nil || mn.EDef().HasTrait("elite") || len(mn.Attachments) == 0 {
					continue
				}
				mn.Damage = mn.MaxHP
				return []engine.Message{engine.MinionDefeated{MinionID: id}}
			}
			return nil
		},
	})
	// 56017 Savage Strike: +6 ATK piercing basic attack — one-shot bonus
	// approximated as direct damage.
	engine.RegisterBehavior("56017", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return []engine.Message{engine.SetEventBonus{Player: p.ID, Damage: 6}}
		},
	})
	// 56018 Audacity: 1 villain damage after spend (payment riders not
	// modeled — kept as a plain resource).
	engine.RegisterBehavior("56018", &engine.Behavior{})
	// 56019 Yellow Jacket: top-5 upgrade fetch.
	engine.RegisterBehavior("56019", &engine.Behavior{
		Playable: func(g *engine.Game, p *engine.Player, def *data.CardDef) bool {
			return g.EntityHasTrait(p.ID, "avenger")
		},
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := 5
			if len(p.Deck) < n {
				n = len(p.Deck)
			}
			for i := 0; i < n; i++ {
				if p.Deck[i].Def().Type == "upgrade" {
					card := p.Deck[i]
					p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
					p.Hand = append(p.Hand, card)
					g.Logf("Yellow Jacket finds %s", card.Def().Name)
					return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
				}
			}
			return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
		},
	})
	// 56020/56021/56022 basic resources.
	engine.RegisterBehavior("56020", &engine.Behavior{})
	engine.RegisterBehavior("56021", &engine.Behavior{})
	engine.RegisterBehavior("56022", &engine.Behavior{})

	// 56023 In Too Deep obligation.
	engine.RegisterBehavior("56023", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			g.Logf("In Too Deep closes in")
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
	})

	// --- Tigra nemesis (56024-56028) ---
	// 56024 The Hood: attacks feed Criminal Underworld.
	engine.RegisterBehavior("56024", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			if s := cwSideScheme(g, "56025"); s != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 1, Source: e.EID()}}
			}
			return nil
		},
	})
	// 56025 Criminal Underworld: criminal minions immune to damage.
	engine.RegisterBehavior("56025", &engine.Behavior{})
	// 56026 The Hood's Mantle / 56027 Madame Masque / 56028 Jigsaw.
	engine.RegisterBehavior("56026", &engine.Behavior{})
	engine.RegisterBehavior("56027", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if s := cwSideScheme(g, "56025"); s != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 2, Source: e.EID()}}
			}
			return nil
		},
	})
	engine.RegisterBehavior("56028", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowAfterEnemyAttacked)
			if !ok || m.Enemy != e.EID() {
				return nil
			}
			if s := cwSideScheme(g, "56025"); s != nil {
				return []engine.Message{engine.SchemeThreat{Scheme: s.ID, N: 2, Source: e.EID()}}
			}
			return nil
		},
	})

	// --- Hulkling suite (56030-56040) ---
	// 56030 Wiccan: identity-specific event fetch.
	engine.RegisterBehavior("56030", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return cwSearchDeck(g, p, func(d *data.CardDef) bool {
				return d.Type == "event" && d.CardSet != "" &&
					(d.CardSet == "hulkling" || d.CardSet == "tigra")
			})
		},
	})
	// 56031 Altman Residence: heal Teddy, recycle a Hulkling card.
	engine.RegisterBehavior("56031", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{
				Label: "Altman Residence — heal 2 and shuffle back a Hulkling card", Type: engine.AbilityAction,
				Exhaust: true, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					p := g.Player(g.Supports[self].Owner)
					if p == nil {
						return nil
					}
					var out []engine.Message
					out = append(out, engine.HealEntity{Target: p.ID, N: 2})
					for i, c := range p.Discard {
						if c.Def().CardSet == "hulkling" {
							card := c
							p.Discard = append(p.Discard[:i:i], p.Discard[i+1:]...)
							p.Deck = append(p.Deck, card)
							out = append(out, engine.ShufflePlayerDeck{Player: p.ID})
							break
						}
					}
					return out
				},
			}}
		},
	})
	// 56032-56035 shape upgrades: stat bonuses + event-triggered actions
	// (approximated as activated abilities).
	shape := func(stat engine.StatBonus) *engine.Behavior {
		return &engine.Behavior{IdentityStats: func(p *engine.Player) engine.StatBonus { return stat }}
	}
	engine.RegisterBehavior("56032", shape(engine.StatBonus{THW: 1, ATK: 1, DEF: 1}))
	engine.RegisterBehavior("56033", shape(engine.StatBonus{ATK: 2, DEF: 1}))
	engine.RegisterBehavior("56034", shape(engine.StatBonus{ATK: 1, DEF: 2, Retaliate: 1}))
	engine.RegisterBehavior("56035", shape(engine.StatBonus{THW: 2}))
	// 56036 Alien Physiology: wild resource (double for Hulkling cards
	// not modeled).
	engine.RegisterBehavior("56036", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild"},
	})
	// 56037 Shapeshifter Strike: 5 damage, ready a Shapeshift upgrade.
	engine.RegisterBehavior("56037", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			out := cardutil.ChooseEnemy("Shapeshifter Strike — 5 damage", func(g *engine.Game, en engine.Entity) (int, []engine.Message) {
				return 5, nil
			})(g, e)
			for _, uid := range p.Upgrades {
				if u := g.Upgrades[uid]; u != nil && u.EDef().HasTrait("shapeshift") && u.Exhausted {
					out = append(out, engine.ReadyEntity{ID: uid})
					break
				}
			}
			return out
		},
	})
	// 56038 Armored Defense: defense event dealing DEF damage + tough.
	engine.RegisterBehavior("56038", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against},
				[]engine.Message{
					engine.DamageEntity{Target: against, Damage: 1, Source: p.ID},
					engine.ToughEntity{Target: p.ID},
				}, true
		},
	})
	// 56039 Impersonation: remove 4; flip rider approximated away.
	engine.RegisterBehavior("56039", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var out []engine.Message
			for _, sid := range g.Schemes() {
				out = append(out, engine.ThwartScheme{Scheme: sid, N: 4, Source: e.EID()})
				break
			}
			return out
		},
	})
	// 56040 Shapeshifter: fetch a Shapeshift upgrade.
	engine.RegisterBehavior("56040", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			return cwSearchDeck(g, p, func(d *data.CardDef) bool { return d.HasTrait("shapeshift") })
		},
	})

	// --- shared cards (56041-56053) ---
	// 56041 Patriot: tough on engage.
	engine.RegisterBehavior("56041", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.EngageMinion)
			if !ok || g.Player(m.Player) == nil || g.Player(m.Player).ID != e.EOwner() {
				return nil
			}
			return []engine.Message{engine.ToughEntity{Target: e.EID()}}
		},
	})
	// 56042 Brother Voodoo: top-5 event fetch.
	engine.RegisterBehavior("56042", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			n := 5
			if len(p.Deck) < n {
				n = len(p.Deck)
			}
			for i := 0; i < n; i++ {
				if p.Deck[i].Def().Type == "event" {
					card := p.Deck[i]
					p.Deck = append(p.Deck[:i:i], p.Deck[i+1:]...)
					p.Hand = append(p.Hand, card)
					return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
				}
			}
			return []engine.Message{engine.ShufflePlayerDeck{Player: p.ID}}
		},
	})
	// 56043 Hidden Base: tough after form change (counters).
	engine.RegisterBehavior("56043", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Supports[e.EID()].Counters = 3
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.ChangeForm)
			if !ok {
				return nil
			}
			s := g.Supports[e.EID()]
			p := g.Player(m.Player)
			if s == nil || s.Exhausted || s.Counters <= 0 || p == nil || p.ID != s.Owner {
				return nil
			}
			s.Exhausted = true
			s.Counters--
			return []engine.Message{engine.ToughEntity{Target: p.ID}}
		},
	})
	// 56044 The Night Nurse: heal + status cleanse.
	engine.RegisterBehavior("56044", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			g.Supports[e.EID()].Counters = 3
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			s := g.Supports[e.EID()]
			if s == nil || s.Counters <= 0 {
				return nil
			}
			return []engine.Ability{{
				Label: "The Night Nurse — heal 1 and clear a status", Type: engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					s := g.Supports[self]
					p := g.Player(s.Owner)
					if p == nil {
						return nil
					}
					s.Counters--
					return []engine.Message{
						engine.HealEntity{Target: p.ID, N: 1},
						engine.ClearStun{Target: p.ID},
						engine.ClearConfuse{Target: p.ID},
					}
				},
			}}
		},
	})
	// 56045 Excelsior: payback damage on defense (payment in reaction not
	// modeled — approximated as automatic 1 damage).
	engine.RegisterBehavior("56045", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.WindowDefended)
			if !ok || m.Defender != e.EOwner() || g.Upgrades[e.EID()] == nil || g.Upgrades[e.EID()].Exhausted {
				return nil
			}
			return []engine.Message{engine.DamageEntity{Target: m.Against, Damage: 1, Source: e.EID()}}
		},
	})
	// 56047 "I Can Do This All Day": defense without exhausting.
	engine.RegisterBehavior("56047", &engine.Behavior{
		DefenseEvent: func(g *engine.Game, p *engine.Player, e *engine.EventCard, against engine.EntityID) (engine.Defends, []engine.Message, bool) {
			return engine.Defends{Defender: p.ID, Against: against, NoExhaust: true}, nil, true
		},
	})
	// 56048 Taunt: villain attacks you; draw 3.
	engine.RegisterBehavior("56048", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			for id := range g.Villains {
				return []engine.Message{engine.AskAttack{Enemy: id, Player: p.ID}, engine.DrawCards{Player: p.ID, N: 3}}
			}
			return []engine.Message{engine.DrawCards{Player: p.ID, N: 3}}
		},
	})
	// 56049 Tackle: stun (+3 damage with a physical payment, not modeled).
	engine.RegisterBehavior("56049", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			var out []engine.Message
			for _, id := range cardutil.SortedEnemyIDs(g) {
				out = append(out, engine.StunEntity{Target: id})
				break
			}
			return out
		},
	})
	// 56050 Cuts Both Ways: retaliate 1 this phase (approximated as
	// nothing persistent — engine lacks retaliate grants).
	engine.RegisterBehavior("56050", &engine.Behavior{})
	// 56051 Preservation: heal rider not modeled.
	engine.RegisterBehavior("56051", &engine.Behavior{})
	// 56052 Iron Lad: power-boost interrupt not modeled.
	engine.RegisterBehavior("56052", &engine.Behavior{})
	// 56053 Assess the Situation.
	engine.RegisterBehavior("56053", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return []engine.Message{engine.TempHandSizeMsg{Player: e.EOwner(), N: 1}}
		},
	})

	// 56054 Complicated Lineage obligation.
	engine.RegisterBehavior("56054", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			g.Logf("Complicated Lineage resurfaces")
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
	})

	// --- Hulkling nemesis (56055-56058) ---
	// 56055 Super Skrull: resource-flavored attack riders approximated
	// away.
	engine.RegisterBehavior("56055", &engine.Behavior{})
	// 56056 Skrull Business: banks discarded deck cards.
	engine.RegisterBehavior("56056", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MillPlayerDeck)
			if !ok {
				return nil
			}
			s := g.SideSchemes[e.EID()]
			p := g.Player(m.Player)
			if s == nil || p == nil {
				return nil
			}
			// The milled cards already left for the discard pile; bank
			// equal faceups from the top as an approximation.
			for i := 0; i < m.N && len(p.Discard) > 0; i++ {
				c := p.Discard[len(p.Discard)-1]
				p.Discard = p.Discard[:len(p.Discard)-1]
				s.StoredCards = append(s.StoredCards, c)
			}
			g.Logf("Skrull Business confiscates %d cards", m.N)
			return nil
		},
	})
	// 56057 Fantastic Powers: finds Super Skrull.
	engine.RegisterBehavior("56057", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			t := g.Attachments[e.EID()]
			if t == nil {
				return nil
			}
			var out []engine.Message
			for i, c := range g.EncounterDeck {
				if c.Code == "56055" {
					card := c
					g.EncounterDeck = append(g.EncounterDeck[:i:i], g.EncounterDeck[i+1:]...)
					def := card.Def()
					p := g.Players[0]
					mn := &engine.Minion{
						ID: g.NextEntityID(engine.KindMinion), Code: card.Code,
						MaxHP: intValue(def.HP, 1) + 3, AttackVal: intValue(def.Attack, 0), SchemeVal: intValue(def.Scheme, 0),
						EngagedWith: p.ID,
					}
					g.Minions[mn.ID] = mn
					g.Attachments[t.ID].Target = mn.ID
					out = append(out, engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
					break
				}
			}
			return out
		},
	})
	// 56058 You're Coming With Me!: mill 3, pay per distinct icon.
	engine.RegisterBehavior("56058", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			n := 3
			if len(p.Deck) < n {
				n = len(p.Deck)
			}
			milled := append(engine.CardList(nil), p.Deck[:n]...)
			p.Deck = p.Deck[n:]
			p.Discard = append(p.Discard, milled...)
			icons := map[string]bool{}
			for _, c := range milled {
				for _, ic := range c.Def().Resources {
					icons[ic] = true
				}
			}
			out := []engine.Message{engine.DiscardCards{Player: p.ID, Cards: milled}}
			if p.IsHero() {
				out = append(out, engine.DamageEntity{Target: p.ID, Damage: len(icons), Source: t.ID})
			} else if g.MainScheme != nil {
				out = append(out, engine.SchemeThreat{Scheme: g.MainScheme.ID, N: len(icons), Source: t.ID})
			}
			return out
		},
	})
}

// intValue dereferences a numeric card field with a fallback.
func intValue(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
}

// cwSideScheme finds a side scheme by exact code.
func cwSideScheme(g *engine.Game, code string) *engine.SideScheme {
	for _, s := range g.SideSchemes {
		if s != nil && s.Code == code {
			return s
		}
	}
	return nil
}
