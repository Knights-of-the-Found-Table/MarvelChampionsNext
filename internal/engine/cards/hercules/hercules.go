// Package hercules registers Hercules, his Labor and Gift decks, signature
// cards, obligation, and Ares nemesis set.
package hercules

import (
	"fmt"
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

var laborCodes = []string{"59002", "59003", "59004"}
var giftCodes = []string{"59005", "59006", "59007"}

func init() {
	registerHercules()
	registerLabors()
	registerGifts()
	registerSignatures()
	registerObligation()
	registerNemesis()
}

func shuffleCards(g *engine.Game, cards engine.CardList) {
	for i := len(cards) - 1; i > 0; i-- {
		j := g.Random(i + 1)
		cards[i], cards[j] = cards[j], cards[i]
	}
}

// Hercules uses the existing serialized side-deck fields as two independent
// piles: SenseDeck is the Labor deck and SideDiscard is the Gift deck. Gift
// cards never enter the side discard through normal play, so the two zones do
// not collide for this identity.
func setupSideDecks(g *engine.Game, p *engine.Player) {
	p.SenseDeck = nil
	p.SideDiscard = nil
	for _, code := range laborCodes {
		p.SenseDeck = append(p.SenseDeck, engine.Card{ID: g.NextCardID(), Code: code, Owner: p.ID})
	}
	for _, code := range giftCodes {
		p.SideDiscard = append(p.SideDiscard, engine.Card{ID: g.NextCardID(), Code: code, Owner: p.ID})
	}
	shuffleCards(g, p.SenseDeck)
	shuffleCards(g, p.SideDiscard)
	g.TLogf("c.beginsWithACardLaborDeckAndACardGiftDeck", p.Name, len(p.SenseDeck), len(p.SideDiscard))
}

func isLaborCode(code string) bool {
	for _, labor := range laborCodes {
		if code == labor {
			return true
		}
	}
	return false
}

func laborInPlay(g *engine.Game, p *engine.Player) bool {
	for _, a := range g.Attachments {
		if a != nil && (a.Code == "59002" || a.Code == "59003") {
			return true
		}
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == "59004" {
			return true
		}
	}
	return false
}

func giftCount(g *engine.Game, p *engine.Player) int {
	if p == nil {
		return 0
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == "59035" {
			return 0
		}
	}
	n := 0
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && (u.Code == "59005" || u.Code == "59006" || u.Code == "59007") {
			n++
		}
	}
	return n
}

func revealTopLabor(g *engine.Game, p *engine.Player) []engine.Message {
	if p == nil || laborInPlay(g, p) || len(p.SenseDeck) == 0 {
		return nil
	}
	card := p.SenseDeck[0]
	p.SenseDeck = p.SenseDeck[1:]
	g.TLogf("c.revealsFromTheLaborDeck", p.Name, card)
	return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: card}}
}

func removeEncounterCard(zone *engine.CardList, code string) {
	for _, c := range *zone {
		if c.Code == code {
			zone.Remove(c.ID)
			return
		}
	}
}

func removeAttachmentFromTarget(g *engine.Game, a *engine.Attachment) {
	if a == nil {
		return
	}
	if mn := g.Minions[a.Target]; mn != nil {
		kept := mn.Attachments[:0]
		for _, id := range mn.Attachments {
			if id != a.ID {
				kept = append(kept, id)
			}
		}
		mn.Attachments = kept
	}
}

func putTopGiftIntoPlay(g *engine.Game, p *engine.Player) []engine.Message {
	if p == nil || len(p.SideDiscard) == 0 {
		return nil
	}
	card := p.SideDiscard[0]
	p.SideDiscard = p.SideDiscard[1:]
	u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: card.Code, Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)
	g.TLogf("c.putsIntoPlayFromTheGiftDeck", p.Name, card)
	if b := engine.LookupBehavior(card.Code); b != nil && b.OnPlay != nil {
		return b.OnPlay(g, u)
	}
	return nil
}

// completeLabor uses ObligationRemoved as the engine's serialized victory
// display proxy. The engine has no global victory-display zone; keeping the
// completed Labor here prevents it from re-entering any encounter zone and
// preserves the completed-card record across saves.
func completeLabor(g *engine.Game, p *engine.Player, code string, entity engine.EntityID) []engine.Message {
	if p == nil || !isLaborCode(code) {
		return nil
	}
	key := fmt.Sprintf("hercules-atonement-%d-%s", g.Round, g.Phase)
	if g.UsedThisRound[key] {
		return nil
	}
	g.UsedThisRound[key] = true
	if a := g.Attachments[entity]; a != nil {
		removeAttachmentFromTarget(g, a)
	}
	g.Delete(entity)
	removeEncounterCard(&g.EncounterDiscard, code)
	p.ObligationRemoved = append(p.ObligationRemoved, engine.Card{ID: g.NextCardID(), Code: code, Owner: p.ID})
	msgs := putTopGiftIntoPlay(g, p)
	msgs = append(msgs, engine.ReadyEntity{ID: p.ID})
	if p.IsHero() {
		msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(
			engine.Tf("c.atonementFlipToAlterEgoForm"),
			engine.Choice{ID: "flip", Label: engine.Tf("c.flipToAlterEgo"), Kind: engine.ChoiceLabel}.Msgs(engine.ChangeForm{Player: p.ID}),
			engine.Choice{ID: "stay", Label: engine.Tf("c.remainInHeroForm"), Kind: engine.ChoicePass},
		)})
	}
	return msgs
}

func registerHercules() {
	engine.RegisterBehavior("59001", &engine.Behavior{
		HeroSetup: func(g *engine.Game, p *engine.Player) []engine.Message {
			setupSideDecks(g, p)
			return nil
		},
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			if p == nil || p.IsHero() || laborInPlay(g, p) || len(p.SenseDeck) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label: engine.Tf("c.newLaborsOfHerculesRevealTheTopLabor"), Type: engine.AbilityAction, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return revealTopLabor(g, g.Player(self))
				},
			}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil {
				return nil
			}
			switch m := msg.(type) {
			case engine.MinionDefeated:
				for _, a := range g.Attachments {
					if a != nil && a.Code == "59002" && a.Target == m.MinionID {
						return completeLabor(g, p, a.Code, a.ID)
					}
				}
			case engine.SchemeDefeated:
				for _, a := range g.Attachments {
					if a != nil && a.Code == "59003" && a.Target == m.Scheme {
						return completeLabor(g, p, a.Code, a.ID)
					}
				}
			}
			return nil
		},
	})
}

func popEncounterCard(g *engine.Game, match func(*data.CardDef) bool) (engine.Card, bool) {
	for _, zone := range []*engine.CardList{&g.EncounterDeck, &g.EncounterDiscard} {
		for _, c := range *zone {
			if match(c.Def()) {
				zone.Remove(c.ID)
				return c, true
			}
		}
	}
	return engine.Card{}, false
}

func registerLabors() {
	// Defeat the Hydra. The target receives the printed +6 HP. Dynamic Elite
	// and the "only Hercules attacks" damage gate cannot be expressed for an
	// arbitrary minion code, so those two restrictions are documented here.
	engine.RegisterBehavior("59002", &engine.Behavior{
		OnAttach: func(g *engine.Game, a *engine.Attachment, _ engine.EntityID) []engine.Message {
			removeEncounterCard(&g.EncounterDiscard, a.Code)
			card, ok := popEncounterCard(g, func(def *data.CardDef) bool {
				return def.Type == "minion" && def.HP != nil && *def.HP >= 6 && !def.HasTrait("elite")
			})
			if !ok {
				g.Delete(a.ID)
				g.TLogf("c.defeatTheHydraFoundNoEligibleMinion")
				return nil
			}
			// A newly revealed minion is identified by the following
			// MinionEntersPlay message and attached there. Encounter cards do
			// not carry an owner, so reveal it to the deterministic first player.
			a.Counters = 1
			return []engine.Message{engine.RevealEncounterCard{Player: cardutil.FirstPlayerID(g), Card: card}}
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			a := g.Attachments[e.EID()]
			m, ok := msg.(engine.MinionEntersPlay)
			if !ok || a == nil || a.Target != "" || a.Counters != 1 {
				return nil
			}
			mn := g.Minions[m.MinionID]
			if mn == nil || mn.EDef().HP == nil || *mn.EDef().HP < 6 || mn.EDef().HasTrait("elite") {
				return nil
			}
			a.Target = mn.ID
			a.Counters = 0
			mn.MaxHP += 6
			mn.Attachments = append(mn.Attachments, a.ID)
			return nil
		},
	})

	// Embody Pathos directly creates the chosen out-of-play encounter side
	// scheme because the generic reveal message exposes no "entered play"
	// hook to the Labor attachment. Per-player symbols are treated as 1, as
	// printed; the engine has no assault flag, and the Hercules-only thwart
	// restriction cannot be attached dynamically to an arbitrary scheme.
	engine.RegisterBehavior("59003", &engine.Behavior{OnAttach: func(g *engine.Game, a *engine.Attachment, _ engine.EntityID) []engine.Message {
		removeEncounterCard(&g.EncounterDiscard, a.Code)
		card, ok := popEncounterCard(g, func(def *data.CardDef) bool { return def.Type == "side_scheme" })
		if !ok {
			g.Delete(a.ID)
			g.TLogf("c.embodyPathosFoundNoEncounterSideScheme")
			return nil
		}
		def := card.Def()
		base := 1
		if def.BaseThreat != nil {
			base = *def.BaseThreat
		}
		s := &engine.SideScheme{
			ID: g.NextEntityID("side_scheme"), Code: card.Code,
			Threat: base + 6, MaxThreat: base + 6,
			Crisis: strings.Contains(strings.ToLower(def.Text), "crisis"), Hazard: def.Hazards,
		}
		g.SideSchemes[s.ID] = s
		a.Target = s.ID
		g.TLogf("c.entersPlayWithEmbodyPathosAttachedThreat", def.Name, s.Threat)
		if b := engine.LookupBehavior(card.Code); b != nil && b.OnPlay != nil {
			return b.OnPlay(g, s)
		}
		return nil
	}})

	// Protect Humanity is represented by an upgrade with three counters so
	// it can remain in play and react to defended villain attacks. The engine
	// cannot redirect an attack to an ally and then let Hercules defend that
	// ally; each villain attack Hercules defends removes one counter instead.
	engine.RegisterBehavior("59004", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: card.Code, Owner: p.ID, Counters: 3}
			g.Upgrades[u.ID] = u
			p.Upgrades = append(p.Upgrades, u.ID)
			for _, zone := range []*engine.CardList{&p.Hand, &p.Deck, &p.Discard} {
				for _, c := range *zone {
					if c.Code != "59008" {
						continue
					}
					zone.Remove(c.ID)
					def := c.Def()
					hp, atk, thw := 1, 0, 0
					if def.HP != nil {
						hp = *def.HP
					}
					if def.Attack != nil {
						atk = *def.Attack
					}
					if def.Thwart != nil {
						thw = *def.Thwart
					}
					a := &engine.Ally{ID: g.NextEntityID("ally"), Code: c.Code, Owner: p.ID, MaxHP: hp, AttackVal: atk, ThwartVal: thw}
					g.Allies[a.ID] = a
					p.Allies = append(p.Allies, a.ID)
					g.TLogf("c.protectHumanityPutsAmadeusChoIntoPlay")
					if b := engine.LookupBehavior(c.Code); b != nil && b.OnPlay != nil {
						return b.OnPlay(g, a)
					}
					return nil
				}
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			u := g.Upgrades[e.EID()]
			m, ok := msg.(engine.WindowDefended)
			if !ok || u == nil || m.Defender != u.Owner || g.Villains[m.Against] == nil {
				return nil
			}
			u.Counters--
			if u.Counters <= 0 {
				return completeLabor(g, g.Player(u.Owner), u.Code, u.ID)
			}
			return nil
		},
	})
}

func registerGifts() {
	engine.RegisterBehavior("59005", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		if p := g.Player(e.EOwner()); p != nil {
			p.MaxHP += 2
		}
		// Steady has no identity stat slot in the engine.
		return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 4}}
	}})
	engine.RegisterBehavior("59006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			if p := g.Player(e.EOwner()); p != nil {
				p.MaxHP++
			}
			return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 4}}
		},
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{Retaliate: 1} },
	})
	engine.RegisterBehavior("59007", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		if p := g.Player(e.EOwner()); p != nil {
			p.MaxHP++
		}
		// Piercing on basic attacks is not represented by identity stats.
		return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 4}}
	}})
}

func giftDiscount(g *engine.Game, p *engine.Player, def *data.CardDef) int {
	return min(giftCount(g, p), cardutil.Cost(def))
}

func registerSignatures() {
	engine.RegisterBehavior("59008", &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{Label: engine.Tf("c.amadeusChoDraw1Card"), Type: engine.AbilityAction, Exhaust: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				return []engine.Message{engine.DrawCards{Player: e.EOwner(), N: 1}}
			},
		}}
		// Redirecting a minion attack to this ally is outside the current
		// defense-substitution channel.
	}})

	engine.RegisterBehavior("59009", &engine.Behavior{
		CardCost: giftDiscount,
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			choices := cardutil.EnemyChoices(g, 5, e.EOwner(), func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.DamageEntity{Target: id, Damage: 5, Source: e.EOwner()}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(engine.Tf("c.theGiftOfBattleChooseAnEnemy"), choices...)}}
		},
	})

	engine.RegisterBehavior("59010", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		p := g.Player(e.EOwner())
		if p == nil {
			return nil
		}
		n := giftCount(g, p)
		msgs := []engine.Message{engine.ReadyEntity{ID: p.ID}}
		if n >= 2 {
			msgs = append(msgs, engine.ToughEntity{Target: p.ID})
		}
		if n >= 3 {
			msgs = append(msgs, engine.DrawCards{Player: p.ID, N: 1})
		}
		if n >= 1 {
			var choices []engine.Choice
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.Exhausted && u.EDef().CardSet == "hercules" {
					choices = append(choices, engine.Choice{Label: engine.S("Ready " + u.EDef().Name), Kind: engine.ChoiceCard, CardCode: u.Code, SourceID: u.ID}.Msgs(engine.ReadyEntity{ID: u.ID}))
				}
			}
			if len(choices) > 0 {
				msgs = append(msgs, engine.AskQuestion{Player: p.ID, Question: engine.Ask(engine.Tf("c.sonOfZeusReadyAnUpgrade"), choices...)})
			}
		}
		return msgs
	}})

	engine.RegisterBehavior("59011", &engine.Behavior{
		CardCost: giftDiscount,
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
				return []engine.Message{engine.ThwartScheme{Scheme: id, N: 4, Source: e.EOwner()}}
			})
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{Player: e.EOwner(), Question: engine.Ask(engine.Tf("c.wisdomOfAthenaChooseAScheme"), choices...)}}
		},
	})

	// ResourceAbility can emit only one icon. Olympus therefore provides one
	// wild resource rather than one for each Gift when multiple Gifts are in play.
	engine.RegisterBehavior("59012", &engine.Behavior{Resource: &engine.ResourceAbility{Icon: "wild"}})

	engine.RegisterBehavior("59013", &engine.Behavior{Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
		return []engine.Ability{{
			Label: engine.Tf("c.gauntletsOfHerculesRetaliateForEachGift"), Type: engine.AbilityTrigger,
			Trigger: engine.TriggerWhenDefended, Exhaust: true, HeroOnly: true,
			Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
				p := g.Player(e.EOwner())
				if p == nil {
					return nil
				}
				// The trigger payload is not passed to Execute; choose the active
				// villain, or the first enemy, as the attacker approximation.
				target := g.ActiveVillain
				if target == "" {
					ids := cardutil.SortedEnemyIDs(g)
					if len(ids) > 0 {
						target = ids[0]
					}
				}
				if target == "" {
					return nil
				}
				return []engine.Message{engine.DamageEntity{Target: target, Damage: giftCount(g, p), Source: p.ID}}
			},
		}}
	}})

	engine.RegisterBehavior("59014", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		u := g.Upgrades[e.EID()]
		m, ok := msg.(engine.BasicAttack)
		if !ok || u == nil || u.Exhausted || m.Player != u.Owner {
			return nil
		}
		n := giftCount(g, g.Player(u.Owner))
		if n == 0 {
			return nil
		}
		// BasicAttack already contains its calculated ATK, so the bonus is
		// applied as a second damage packet. This auto-uses the ready Mace;
		// overkill is not exposed by the combat message.
		return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.DamageEntity{Target: m.Target, Damage: n, Source: u.Owner}}
	}})

	engine.RegisterBehavior("59015", &engine.Behavior{DamagePrevention: func(g *engine.Game, u *engine.Upgrade, p *engine.Player, n int) (int, int) {
		if u.Exhausted {
			return 0, 0
		}
		u.Exhausted = true
		// DamagePrevention does not expose attack attribution; this prevents
		// the next damage instance rather than villain-attack damage only.
		return min(1, n), 0
	}})

	engine.RegisterBehavior("59016", &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus { return engine.StatBonus{THW: 1, ATK: 1, DEF: 1} },
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			if _, ok := msg.(engine.EndRound); !ok {
				return nil
			}
			return []engine.Message{engine.DiscardControlled{Player: e.EOwner(), ID: e.EID()}}
		},
	})

	engine.RegisterBehavior("59017", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		u := g.Upgrades[e.EID()]
		m, ok := msg.(engine.DamageEntity)
		if !ok || u == nil || u.Exhausted || m.Source != u.Owner {
			return nil
		}
		mn := g.Minions[m.Target]
		if mn == nil || m.Damage <= mn.HP() {
			return nil
		}
		// DamageEntity does not identify attack vs event damage. Excess damage
		// from any Hercules-sourced lethal hit is treated as an attack.
		return []engine.Message{engine.ExhaustEntity{ID: u.ID}, engine.HealEntity{Target: u.Owner, N: m.Damage - mn.HP()}}
	}})
}

func registerObligation() {
	// Appeal to Athena remains in play as a pseudo-upgrade so giftCount can
	// enforce "considered to control none". The alternative double-mental
	// payment is omitted; alter-ego Hercules may exhaust to remove it.
	engine.RegisterBehavior("59035", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			u := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: card.Code, Owner: p.ID}
			g.Upgrades[u.ID] = u
			p.Upgrades = append(p.Upgrades, u.ID)
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return []engine.Ability{{Label: engine.Tf("c.appealToAthenaExhaustHerculesAndRemove"), Type: engine.AbilityAction, AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					u := g.Upgrades[self]
					if u == nil {
						return nil
					}
					g.Delete(u.ID)
					if p := g.Player(u.Owner); p != nil {
						p.ObligationRemoved = append(p.ObligationRemoved, engine.Card{ID: g.NextCardID(), Code: u.Code, Owner: p.ID})
						return []engine.Message{engine.ExhaustEntity{ID: p.ID}}
					}
					return nil
				},
			}}
		},
	})
}

func registerNemesis() {
	engine.RegisterBehavior("59036", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.MinionActivates)
		if !ok || m.MinionID != e.EID() {
			return nil
		}
		p := g.Player(m.Player)
		if p != nil && !p.IsHero() {
			return []engine.Message{engine.DealEncounterToPlayer{Player: p.ID}}
		}
		return nil
	}})

	engine.RegisterBehavior("59037", &engine.Behavior{React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
		m, ok := msg.(engine.DamageEntity)
		mn := g.Minions[e.EID()]
		if !ok || mn == nil || m.Target != mn.ID || m.Damage <= 0 || g.Entity(m.Source) == nil {
			return nil
		}
		owner := g.Player(g.Entity(m.Source).EOwner())
		if owner == nil {
			return nil
		}
		var choices []engine.Choice
		for _, c := range owner.Hand {
			for _, r := range c.Def().Resources {
				if r == "physical" || r == "wild" {
					choices = append(choices, engine.Choice{Label: engine.S("Spend " + c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code}.Msgs(engine.DiscardCards{Player: owner.ID, Cards: engine.CardList{c}}))
					break
				}
			}
		}
		choices = append(choices, engine.Choice{ID: "heal", Label: engine.Tf("c.letLerneanHydraHeal2"), Kind: engine.ChoiceLabel}.Msgs(engine.HealEntity{Target: mn.ID, N: 2}))
		return []engine.Message{engine.AskQuestion{Player: owner.ID, Question: engine.Ask(engine.Tf("c.lerneanHydraSpendAPhysicalResource"), choices...)}}
	}})

	engine.RegisterBehavior("59038", &engine.Behavior{OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
		n := 0
		for _, entity := range g.AllEntities() {
			if entity.EID() != e.EID() && entity.EDef().HasTrait("olympus") {
				n++
			}
		}
		if n == 0 {
			return nil
		}
		return []engine.Message{engine.SchemeThreat{Scheme: e.EID(), N: n, Source: e.EID()}}
	}})

	engine.RegisterBehavior("59039", &engine.Behavior{OnAttach: func(g *engine.Game, a *engine.Attachment, _ engine.EntityID) []engine.Message {
		for _, id := range cardutil.SortedIDs(g.Minions) {
			if mn := g.Minions[id]; mn != nil && mn.Code == "59036" {
				a.Target = id
				mn.AttackVal += 2
				mn.Attachments = append(mn.Attachments, a.ID)
				return nil
			}
		}
		for _, id := range cardutil.SortedIDs(g.Villains) {
			if v := g.Villains[id]; v != nil {
				a.Target = id
				v.AttackVal += 2
				return nil
			}
		}
		return nil
		// Whether a friendly character took damage is absent from
		// WindowAfterEnemyAttacked, so the conditional self-discard is omitted.
	}})

	engine.RegisterBehavior("59040", &engine.Behavior{ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
		g.Delete(t.ID)
		var msgs []engine.Message
		for _, id := range cardutil.SortedIDs(g.Minions) {
			if mn := g.Minions[id]; mn != nil {
				msgs = append(msgs, engine.AskAttack{Enemy: id, Player: mn.EngagedWith})
			}
		}
		if len(msgs) > 0 {
			return msgs
		}
		var discarded engine.CardList
		for len(g.EncounterDeck) > 0 {
			c := g.EncounterDeck[0]
			g.EncounterDeck = g.EncounterDeck[1:]
			if c.Def().Type == "minion" {
				g.EncounterDiscard = append(g.EncounterDiscard, discarded...)
				return []engine.Message{engine.RevealEncounterCard{Player: p.ID, Card: c}}
			}
			discarded = append(discarded, c)
		}
		g.EncounterDiscard = append(g.EncounterDiscard, discarded...)
		return nil
	}})
}
