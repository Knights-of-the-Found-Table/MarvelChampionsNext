// Package hawkeye registers the Hawkeye hero pack (trors): the
// Hawkeye / Clint Barton identity, Hawkeye's Bow + Quiver, the five
// Arrow events, Mockingbird, Expert Marksman, the Kate Bishop ally,
// and the Crossfire nemesis set.
package hawkeye

import (
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerHawkeye()
	registerSignatures()
	registerNemesis()
	registerObligation()
}

// bowCode / quiverCode name the two signature upgrades the arrows key off.
const (
	bowCode    = "04002"
	quiverCode = "04003"
)

// readyBow returns the player's Hawkeye's Bow upgrade when it is in play
// and ready (the arrows exhaust it as part of their effect).
func readyBow(g *engine.Game, p *engine.Player) *engine.Upgrade {
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == bowCode && !u.Exhausted {
			return u
		}
	}
	return nil
}

// anyBow returns the player's Hawkeye's Bow regardless of readiness.
func anyBow(g *engine.Game, p *engine.Player) *engine.Upgrade {
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.Code == bowCode {
			return u
		}
	}
	return nil
}

// hawkeyePlayer finds the player running the Hawkeye identity.
func hawkeyePlayer(g *engine.Game) *engine.Player {
	for _, p := range g.Players {
		if strings.HasPrefix(p.HeroCode, "04001") || strings.HasPrefix(p.AlterEgoCode, "04001") {
			return p
		}
	}
	return nil
}

// registerHawkeye installs the Hawkeye / Clint Barton identity (04001a/b).
func registerHawkeye() {
	engine.RegisterBehavior("04001", &engine.Behavior{
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			var out []engine.Ability
			// "Quick Draw" — Action: Exhaust Hawkeye → ready Hawkeye's
			// Bow. Only offered while a bow sits in play exhausted.
			if bow := anyBow(g, p); bow != nil && bow.Exhausted {
				out = append(out, engine.Ability{
					Label:    engine.Tf("c.quickDrawExhaustHawkeyeReadyHawkeyeSBow"),
					Type:     engine.AbilityAction,
					Exhaust:  true,
					HeroOnly: true,
					Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
						pl := g.Player(self)
						if pl == nil {
							return nil
						}
						bow := anyBow(g, pl)
						if bow == nil {
							return nil
						}
						g.TLogf("c.quickDrawHawkeyeSBowIsReadied")
						return []engine.Message{engine.ReadyEntity{ID: bow.ID}}
					},
				})
			}
			// Weapon of Choice — Action: Spend 1 resource of any type →
			// search your deck and discard pile for Hawkeye's Bow and add
			// it to your hand. Shuffle. (Limit once per phase; the
			// once-per-turn tracker resets between phases.)
			out = append(out, engine.Ability{
				Label:        engine.Tf("c.weaponOfChoiceSpend1ResourceSearchForHawkeyeSBow"),
				Type:         engine.AbilityAction,
				AlterEgoOnly: true,
				Cost:         1,
				OncePerTurn:  true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(self)
					if pl == nil {
						return nil
					}
					var choices []engine.Choice
					for _, c := range pl.Deck {
						if c.Code == bowCode {
							choices = append(choices, engine.Choice{
								Label: engine.Tf("c.takeFromDeck", c), Kind: engine.ChoiceCard, CardCode: c.Code,
							}.Msgs(
								engine.TakeDeckCard{Player: pl.ID, CardID: c.ID},
								engine.ShufflePlayerDeck{Player: pl.ID},
							))
						}
					}
					for _, c := range pl.Discard {
						if c.Code == bowCode {
							choices = append(choices, engine.Choice{
								Label: engine.Tf("c.takeFromDiscard", c), Kind: engine.ChoiceCard, CardCode: c.Code,
							}.Msgs(
								engine.ReturnDiscardCard{Player: pl.ID, CardID: c.ID},
								engine.ShufflePlayerDeck{Player: pl.ID},
							))
						}
					}
					if len(choices) == 0 {
						g.TLogf("c.weaponOfChoiceHawkeyeSBowIsNotInDeckOrDiscardShuffling")
						return []engine.Message{engine.ShufflePlayerDeck{Player: pl.ID}}
					}
					choices = append(choices, engine.Choice{
						ID: "skip", Label: engine.Tf("c.skipStillShuffle"), Kind: engine.ChoicePass,
					}.Msgs(engine.ShufflePlayerDeck{Player: pl.ID}))
					return []engine.Message{engine.AskQuestion{
						Player:   pl.ID,
						Question: engine.Ask(engine.Tf("c.weaponOfChoiceTakeHawkeyeSBow"), choices...),
					}}
				},
			})
			return out
		},
	})
}

// registerSignatures installs Hawkeye's signature cards.
func registerSignatures() {
	registerBow()
	registerQuiver()
	registerMockingbird()
	registerArrows()
	registerExpertMarksman()
	registerKateBishop()
}

// 04002 Hawkeye's Bow: Restricted. Your hero gets +1 ATK and each of
// your Arrow attacks gain ranged. (Approximation: ranged is not modeled
// by the engine — it only interacts with retaliate, which event attacks
// already bypass.)
func registerBow() {
	engine.RegisterBehavior(bowCode, &engine.Behavior{
		IdentityStats: func(p *engine.Player) engine.StatBonus {
			return engine.StatBonus{ATK: 1}
		},
	})
}

// 04003 Hawkeye's Quiver: Hero Action — exhaust → search the top 5
// cards of your deck for an Arrow event and attach it faceup to this
// card; attached Arrows may be played as if in hand. (Approximation:
// the found Arrow goes straight to the hand, which subsumes both the
// attachment and the "play as if in hand" clause; the unchosen cards
// stay on top in their current order.)
func registerQuiver() {
	engine.RegisterBehavior(quiverCode, &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || !p.IsHero() {
				return nil
			}
			n := 5
			if len(p.Deck) < n {
				n = len(p.Deck)
			}
			found := false
			for i := 0; i < n; i++ {
				if p.Deck[i].Def().HasTrait("arrow") {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
			return []engine.Ability{{
				Label:    engine.Tf("c.hawkeyeSQuiverSearchTheTop5ForAnArrowEvent"),
				Type:     engine.AbilityAction,
				Exhaust:  true,
				HeroOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(e.EOwner())
					if pl == nil {
						return nil
					}
					n := 5
					if len(pl.Deck) < n {
						n = len(pl.Deck)
					}
					var choices []engine.Choice
					for i := 0; i < n; i++ {
						c := pl.Deck[i]
						if !c.Def().HasTrait("arrow") {
							continue
						}
						choices = append(choices, engine.Choice{
							Label: engine.Tf("c.takeName", c), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(engine.TakeDeckCard{Player: pl.ID, CardID: c.ID}))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   pl.ID,
						Question: engine.Ask(engine.Tf("c.hawkeyeSQuiverTakeAnArrowEventFromTheTop5"), choices...),
					}}
				},
			}}
		},
	})
}

// 04004 Mockingbird: Interrupt — when the villain initiates an attack
// against you, spend 1 resource of any type and return Mockingbird to
// your hand → prevent all damage from this attack. (Approximation: the
// engine's triggered interrupts cannot replace the defense prompt's
// damage resolution, so the prevention is not modeled; the spend +
// bounce is.)
func registerMockingbird() {
	engine.RegisterBehavior("04004", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Hand) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label:   engine.Tf("c.mockingbirdSpend1ResourceReturnHerToYourHandAttackDamagePrev"),
				Type:    engine.AbilityTrigger,
				Trigger: engine.TriggerVillainAttacksYou,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(e.EOwner())
					if pl == nil {
						return nil
					}
					var choices []engine.Choice
					for _, c := range pl.Hand {
						if len(c.Def().Resources) == 0 {
							continue
						}
						choices = append(choices, engine.Choice{
							Label: engine.Tf("c.spendName", c), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(
							engine.DiscardCards{Player: pl.ID, Cards: engine.CardList{c}},
							engine.ReturnControlled{Player: pl.ID, ID: self},
						))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   pl.ID,
						Question: engine.Ask(engine.Tf("c.mockingbirdSpend1ResourceToReturnHerToYourHand"), choices...),
					}}
				},
			}}
		},
	})
}

// arrowEffect is the shared skeleton of the five Arrow events: each is
// a Hero Action that exhausts Hawkeye's Bow as part of its cost.
// (Approximation: without a ready bow the event fizzles after its
// resource cost was already paid by the generic play flow.)
func arrowEffect(g *engine.Game, e engine.Entity, build func(bow *engine.Upgrade) []engine.Message) []engine.Message {
	p := g.Player(e.EOwner())
	if p == nil {
		return nil
	}
	bow := readyBow(g, p)
	if bow == nil {
		g.TLogf("c.noReadyHawkeyeSBowInPlayTheArrowFizzles", e)
		return nil
	}
	return build(bow)
}

// registerArrows installs the five Arrow events (04005-04009).
func registerArrows() {
	// 04005 Sonic Arrow: Hero Action (attack) — Exhaust Hawkeye's Bow →
	// confuse an enemy and deal 3 damage to it (5 if already confused).
	engine.RegisterBehavior("04005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return arrowEffect(g, e, func(bow *engine.Upgrade) []engine.Message {
				pid := e.EOwner()
				var choices []engine.Choice
				for _, id := range cardutil.SortedEnemyIDs(g) {
					enemy := g.Entity(id)
					dmg := 3
					if confused(enemy) {
						dmg = 5
					}
					choices = append(choices, engine.Choice{
						Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
						SourceID: id, CardCode: enemy.ECode(),
					}.Msgs(
						engine.ExhaustEntity{ID: bow.ID},
						engine.ConfuseEntity{Target: id},
						engine.DamageEntity{Target: id, Damage: dmg, Source: pid},
					))
				}
				if len(choices) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{
					Player:   pid,
					Question: engine.Ask(engine.Tf("c.sonicArrowConfuseAnEnemyAndDeal3Damage5IfAlreadyConfused"), choices...),
				}}
			})
		},
	})

	// 04006 Explosive Arrow: Hero Action — Exhaust Hawkeye's Bow and
	// choose a player → deal 3 damage to the villain and each minion
	// engaged with that player.
	engine.RegisterBehavior("04006", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return arrowEffect(g, e, func(bow *engine.Upgrade) []engine.Message {
				pid := e.EOwner()
				var choices []engine.Choice
				for _, q := range g.Players {
					msgs := []engine.Message{engine.ExhaustEntity{ID: bow.ID}}
					hit := false
					for _, id := range cardutil.SortedIDs(g.Villains) {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 3, Source: pid})
						hit = true
					}
					for _, id := range cardutil.SortedIDs(g.Minions) {
						if mn := g.Minions[id]; mn != nil && mn.EngagedWith == q.ID {
							msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 3, Source: pid})
							hit = true
						}
					}
					if !hit {
						continue
					}
					choices = append(choices, engine.Choice{
						Label: engine.S(q.Name), Kind: engine.ChoiceTarget, SourceID: q.ID,
					}.Msgs(msgs...))
				}
				if len(choices) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{
					Player:   pid,
					Question: engine.Ask(engine.Tf("c.explosiveArrowChooseAPlayer3DamageToTheVillainAndTheirEngage"), choices...),
				}}
			})
		},
	})

	// 04007 Electric Arrow: Hero Action (attack) — Exhaust Hawkeye's Bow
	// → stun an enemy and deal 3 damage to it (5 if already stunned).
	engine.RegisterBehavior("04007", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return arrowEffect(g, e, func(bow *engine.Upgrade) []engine.Message {
				pid := e.EOwner()
				var choices []engine.Choice
				for _, id := range cardutil.SortedEnemyIDs(g) {
					enemy := g.Entity(id)
					dmg := 3
					if stunned(enemy) {
						dmg = 5
					}
					choices = append(choices, engine.Choice{
						Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
						SourceID: id, CardCode: enemy.ECode(),
					}.Msgs(
						engine.ExhaustEntity{ID: bow.ID},
						engine.StunEntity{Target: id},
						engine.DamageEntity{Target: id, Damage: dmg, Source: pid},
					))
				}
				if len(choices) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{
					Player:   pid,
					Question: engine.Ask(engine.Tf("c.electricArrowStunAnEnemyAndDeal3Damage5IfAlreadyStunned"), choices...),
				}}
			})
		},
	})

	// 04008 Cable Arrow: Hero Action (thwart) — Exhaust Hawkeye's Bow →
	// remove 3 threat from a scheme, ignoring any crisis icons in play.
	// (Approximation: crisis thwart-blocking is not modeled, so there is
	// nothing to ignore.)
	engine.RegisterBehavior("04008", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return arrowEffect(g, e, func(bow *engine.Upgrade) []engine.Message {
				pid := e.EOwner()
				choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.ExhaustEntity{ID: bow.ID},
						engine.ThwartScheme{Scheme: id, N: 3, Source: pid},
					}
				})
				if len(choices) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{
					Player:   pid,
					Question: engine.Ask(engine.Tf("c.cableArrowRemove3ThreatFromAScheme"), choices...),
				}}
			})
		},
	})

	// 04009 Vibranium Arrow: Hero Action (attack) — Exhaust Hawkeye's
	// Bow → deal 6 damage to an enemy. This attack gains piercing.
	// (Piercing is not modeled by the engine.)
	engine.RegisterBehavior("04009", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			return arrowEffect(g, e, func(bow *engine.Upgrade) []engine.Message {
				pid := e.EOwner()
				choices := cardutil.EnemyChoices(g, 6, pid, func(target engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.ExhaustEntity{ID: bow.ID},
						engine.DamageEntity{Target: target, Damage: 6, Source: pid},
					}
				})
				if len(choices) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{
					Player:   pid,
					Question: engine.Ask(engine.Tf("c.vibraniumArrowDeal6DamageToAnEnemy"), choices...),
				}}
			})
		},
	})
}

// confused / stunned read the status off an enemy entity.
func confused(e engine.Entity) bool {
	switch t := e.(type) {
	case *engine.Villain:
		return t.Confused
	case *engine.Minion:
		return t.Confused
	}
	return false
}

func stunned(e engine.Entity) bool {
	switch t := e.(type) {
	case *engine.Villain:
		return t.Stunned
	case *engine.Minion:
		return t.Stunned
	}
	return false
}

// 04010 Expert Marksman: Resource — exhaust → generate a [wild]
// resource for an Arrow event. (Approximation: the Arrow-only
// restriction is relaxed to any event.)
func registerExpertMarksman() {
	engine.RegisterBehavior("04010", &engine.Behavior{
		Resource: &engine.ResourceAbility{Icon: "wild", HeroOnly: true, EventOnly: true},
	})
}

// 04011 Hawkeye (Kate Bishop) ally: Action — exhaust this ally and
// discard 1 card from your hand → deal X damage to an enemy, where X is
// the number of printed resources on that card.
func registerKateBishop() {
	engine.RegisterBehavior("04011", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || len(p.Hand) == 0 || len(g.Enemies()) == 0 {
				return nil
			}
			return []engine.Ability{{
				Label:   engine.Tf("c.hawkeyeExhaustAndDiscard1CardDealDamageEqualToItsPrintedReso"),
				Type:    engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(e.EOwner())
					if pl == nil {
						return nil
					}
					var choices []engine.Choice
					for _, c := range pl.Hand {
						x := len(c.Def().Resources)
						sub := cardutil.EnemyChoices(g, x, pl.ID, func(target engine.EntityID) []engine.Message {
							return []engine.Message{engine.DamageEntity{Target: target, Damage: x, Source: pl.ID}}
						})
						if len(sub) == 0 {
							continue
						}
						choices = append(choices, engine.Choice{
							Label: engine.Tf("c.discardXCost", c, x), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(
							engine.DiscardCards{Player: pl.ID, Cards: engine.CardList{c}},
						).WithThen(engine.Ask(engine.S("Hawkeye — deal "+itoa(x)+" damage to which enemy?"), sub...)))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   pl.ID,
						Question: engine.Ask(engine.Tf("c.hawkeyeDiscardWhichCard"), choices...),
					}}
				},
			}}
		},
	})
}

// itoa avoids pulling strconv into the label builders for one-digit ints.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [4]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// registerNemesis installs the Hawkeye nemesis encounter set:
// Crossfire, Marked for Death, Crossfire's Rifle, Sniper Shot.
func registerNemesis() {
	// 04027 Crossfire: Quickstrike; his attacks gain piercing.
	// (Quickstrike is handled by the nemesis-reveal flow; piercing is
	// not modeled. The piercing boost text is likewise skipped.)
	engine.RegisterBehavior("04027", &engine.Behavior{})

	// 04028 Marked for Death: When Revealed — the Clint Barton player
	// searches their hand, deck, discard pile, and play area for
	// Mockingbird and places her faceup beneath this card. When
	// defeated, return Mockingbird to her owner's hand. (The card is
	// parked in the scheme's StoredCards while beneath it.)
	engine.RegisterBehavior("04028", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			s, ok := e.(*engine.SideScheme)
			if !ok {
				return nil
			}
			p := hawkeyePlayer(g)
			if p == nil {
				return nil
			}
			// Play area first (an ally in play), then the card zones.
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.Code == "04004" {
					g.Delete(id)
					s.StoredCards = append(s.StoredCards, engine.Card{
						ID: g.NextCardID(), Code: "04004", Owner: p.ID,
					})
					g.TLogf("c.markedForDeathMockingbirdIsPlacedBeneathTheScheme")
					return nil
				}
			}
			zones := []*engine.CardList{&p.Hand, &p.Deck, &p.Discard}
			for _, z := range zones {
				for _, c := range *z {
					if c.Code == "04004" {
						z.Remove(c.ID)
						s.StoredCards = append(s.StoredCards, c)
						g.TLogf("c.markedForDeathMockingbirdIsPlacedBeneathTheScheme")
						return nil
					}
				}
			}
			g.TLogf("c.markedForDeathMockingbirdIsNowhereToBeFound")
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			s, ok := e.(*engine.SideScheme)
			if !ok || len(s.StoredCards) == 0 {
				return nil
			}
			for _, c := range s.StoredCards {
				if p := g.Player(c.Owner); p != nil {
					p.Hand = append(p.Hand, c)
					g.TLogf("c.markedForDeathDefeatedReturnsToSHand", c, p.Name)
				}
			}
			s.StoredCards = nil
			return nil
		},
	})

	// 04029 Crossfire's Rifle: Attach to Crossfire, otherwise the
	// villain. Attached enemy's attacks gain ranged (not modeled).
	// Hero Action: Exhaust your hero and spend a [wild] resource →
	// discard Crossfire's Rifle.
	engine.RegisterBehavior("04029", &engine.Behavior{
		OnAttach: func(g *engine.Game, t *engine.Attachment, target engine.EntityID) []engine.Message {
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.Code == "04027" {
					t.Target = id
					g.TLogf("c.crossfireSRifleAttachesToCrossfire")
					return nil
				}
			}
			for _, id := range cardutil.SortedIDs(g.Villains) {
				t.Target = id
				g.TLogf("c.crossfireSRifleAttachesToTheVillain")
				return nil
			}
			return nil
		},
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(g.ActiveTurn)
			if p == nil || !p.IsHero() || p.Exhausted {
				return nil
			}
			return []engine.Ability{{
				Label:     engine.Tf("c.crossfireSRifleExhaustYourHeroAndSpendWildDiscardThisAttachm"),
				Type:      engine.AbilityAction,
				HeroOnly:  true,
				Cost:      1,
				CostIcons: "wild:1",
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					return []engine.Message{
						engine.ExhaustEntity{ID: g.ActiveTurn},
						engine.DiscardAttachmentMsg{ID: self},
					}
				},
			}}
		},
	})

	// 04030 Sniper Shot: When Revealed (Alter-Ego) — place 3 threat on
	// the main scheme. When Revealed (Hero) — deal 3 damage to your
	// hero.
	engine.RegisterBehavior("04030", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			if p.IsHero() {
				g.TLogf("c.sniperShot3DamageTo", p.Name)
				return []engine.Message{engine.DamageEntity{Target: p.ID, Damage: 3, Source: t.ID}}
			}
			if g.MainScheme != nil {
				g.TLogf("c.sniperShot3ThreatOnTheMainScheme")
				return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: 3, Source: t.ID}}
			}
			return nil
		},
	})
}

// registerObligation installs Criminal Past (04026): You may flip to
// alter-ego form. Choose: exhaust Clint Barton → remove this card from
// the game, or discard Hawkeye's Bow from play and discard this
// obligation.
func registerObligation() {
	engine.RegisterBehavior("04026", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if p == nil {
				return nil
			}
			var penalty []engine.Message
			if bow := anyBow(g, p); bow != nil {
				penalty = append(penalty, engine.DiscardControlled{Player: p.ID, ID: bow.ID})
			}
			return cardutil.ExhaustOrPenalty(g, p, card,
				engine.Tf("c.discardHawkeyeSBowFromPlay"), penalty...)
		},
	})
}
