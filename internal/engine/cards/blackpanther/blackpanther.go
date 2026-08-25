// Package blackpanther registers the Black Panther (Shuri) hero pack
// (bp): the Black Panther / Shuri identity, the four Black Panther
// upgrades with "Special" abilities and the cards that trigger them
// (Clawed Strike, On the Prowl, Wakanda Forever!, the T'Challa ally),
// the Wakanda supports, and the Klaw nemesis set.
package blackpanther

import (
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
)

func init() {
	registerBlackPanther()
	registerSignatures()
	registerNemesis()
	registerObligation()
}

// specialTrigger names the contextual trigger shared with the core
// Black Panther upgrades: "wakanda" abilities stay out of the action
// menu and are fired by effects that resolve Specials (Wakanda
// Forever!, Shuri's response, Clawed Strike...).
const specialTrigger = "wakanda"

// shuriPlayer finds the player running the Black Panther (Shuri)
// identity.
func shuriPlayer(g *engine.Game) *engine.Player {
	for _, p := range g.Players {
		if strings.HasPrefix(p.HeroCode, "51001") || strings.HasPrefix(p.AlterEgoCode, "51001") {
			return p
		}
	}
	return nil
}

// specialUpgrades lists the player's in-play upgrades with the Black
// Panther trait and a Special ability, in stable order.
func specialUpgrades(g *engine.Game, p *engine.Player) []*engine.Upgrade {
	var out []*engine.Upgrade
	for _, id := range p.Upgrades {
		u := g.Upgrades[id]
		if u == nil {
			continue
		}
		def := u.EDef()
		if def.HasTrait("black_panther") && strings.Contains(def.Text, "Special") {
			out = append(out, u)
		}
	}
	return out
}

// specialExecute runs one upgrade's registered Special ability.
func specialExecute(g *engine.Game, id engine.EntityID) []engine.Message {
	u := g.Upgrades[id]
	if u == nil {
		return nil
	}
	b := engine.LookupBehavior(u.Code)
	if b == nil || b.Abilities == nil {
		return nil
	}
	for _, ab := range b.Abilities(g, u) {
		if ab.Trigger == specialTrigger && ab.Execute != nil {
			return ab.Execute(g, id)
		}
	}
	return nil
}

// specialQuestion builds the "resolve the Special ability on 1 Black
// Panther upgrade you control" prompt. Returns nil when the player
// controls no eligible upgrade.
func specialQuestion(g *engine.Game, p *engine.Player) *engine.Question {
	ups := specialUpgrades(g, p)
	if len(ups) == 0 {
		return nil
	}
	var choices []engine.Choice
	for _, u := range ups {
		choices = append(choices, engine.Choice{
			Label: engine.S(u.EDef().Name), Kind: engine.ChoiceCard, CardCode: u.Code, SourceID: u.ID,
		}.Msgs(specialExecute(g, u.ID)...))
	}
	if len(choices) == 0 {
		return nil
	}
	return engine.Ask(engine.Tf("c.resolveTheSpecialAbilityOn1BlackPantherUpgrade"), choices...)
}

// registerBlackPanther installs the Black Panther / Shuri identity
// (51001a/b).
func registerBlackPanther() {
	engine.RegisterBehavior("51001", &engine.Behavior{
		// Hero side — Response: after Black Panther uses a basic power,
		// resolve the Special ability on 1 Black Panther upgrade you
		// control. Basic powers: attack, thwart, defend (recover lives
		// on the alter-ego side, where this text is inactive).
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EID())
			if p == nil || !p.IsHero() {
				return nil
			}
			used := false
			switch m := msg.(type) {
			case engine.BasicAttack:
				used = m.Player == p.ID
			case engine.BasicThwart:
				used = m.Player == p.ID
			case engine.Defends:
				used = m.Defender == p.ID && !m.Undefended && m.Via == ""
			}
			if !used {
				return nil
			}
			q := specialQuestion(g, p)
			if q == nil {
				return nil
			}
			g.TLogf("c.blackPantherSResponseResolveABlackPantherUpgradeSSpecial")
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: q}}
		},
		// Shuri — Inventor: Action: Exhaust Shuri → search your deck for
		// a Black Panther or Tech upgrade and play it, reducing its
		// resource cost by 2 (limit once per round). (Approximation: the
		// upgrade enters play for free — every Black Panther upgrade
		// costs 2, so the discount usually covers the whole cost.)
		HeroAbilities: func(g *engine.Game, p *engine.Player) []engine.Ability {
			return []engine.Ability{{
				Label:        engine.Tf("c.inventorSearchYourDeckForABlackPantherOrTechUpgradeAndPlayIt"),
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				OncePerRound: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(self)
					if pl == nil {
						return nil
					}
					var choices []engine.Choice
					for _, c := range pl.Deck {
						def := c.Def()
						if def.Type != "upgrade" {
							continue
						}
						if !def.HasTrait("black_panther") && !def.HasTrait("tech") {
							continue
						}
						choices = append(choices, engine.Choice{
							Label: engine.S("Play " + def.Name), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(
							engine.UpgradeEnterPlay{Player: pl.ID, Card: c},
							engine.ShufflePlayerDeck{Player: pl.ID},
						))
					}
					if len(choices) == 0 {
						g.TLogf("c.inventorNoBlackPantherOrTechUpgradeInDeckShuffling")
						return []engine.Message{engine.ShufflePlayerDeck{Player: pl.ID}}
					}
					choices = append(choices, engine.Choice{
						ID: "skip", Label: engine.Tf("c.skipStillShuffle"), Kind: engine.ChoicePass,
					}.Msgs(engine.ShufflePlayerDeck{Player: pl.ID}))
					return []engine.Message{engine.AskQuestion{
						Player:   pl.ID,
						Question: engine.Ask(engine.Tf("c.inventorPlayABlackPantherOrTechUpgradeFromYourDeck"), choices...),
					}}
				},
			}}
		},
	})
}

// registerSignatures installs the Black Panther (Shuri) signature cards.
func registerSignatures() {
	registerTChallaAlly()
	registerClawedStrike()
	registerOnTheProwl()
	registerWakandaForever()
	registerElephantsTrunk()
	registerQueenRamonda()
	registerAjaAdanna()
	registerKimoyoBeads()
	registerPantherClaws()
	registerSpiderBites()
	registerVibraniumSuit()
}

// 51002 T'Challa ally: Hero Response — after T'Challa uses a basic
// power, resolve the Special ability on 1 Black Panther upgrade you
// control.
func registerTChallaAlly() {
	engine.RegisterBehavior("51002", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil || !p.IsHero() {
				return nil
			}
			used := false
			switch m := msg.(type) {
			case engine.AllyAttackWindow:
				used = m.Ally == e.EID()
			case engine.AllyThwartWindow:
				used = m.Ally == e.EID()
			case engine.Defends:
				used = m.Defender == e.EID() && !m.Undefended
			}
			if !used {
				return nil
			}
			q := specialQuestion(g, p)
			if q == nil {
				return nil
			}
			g.TLogf("c.tChallaSResponseResolveABlackPantherUpgradeSSpecial")
			return []engine.Message{engine.AskQuestion{Player: p.ID, Question: q}}
		},
	})
}

// 51003 Clawed Strike: Hero Action (attack) — deal 4 damage to an
// enemy, then resolve the Special ability on 1 Black Panther upgrade
// you control.
func registerClawedStrike() {
	engine.RegisterBehavior("51003", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			special := specialQuestion(g, p)
			var choices []engine.Choice
			for _, id := range cardutil.SortedEnemyIDs(g) {
				enemy := g.Entity(id)
				ch := engine.Choice{
					Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: enemy.ECode(),
				}.Msgs(engine.DamageEntity{Target: id, Damage: 4, Source: pid})
				if special != nil {
					ch = ch.WithThen(special)
				}
				choices = append(choices, ch)
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.clawedStrikeDeal4DamageToAnEnemy"), choices...),
			}}
		},
	})
}

// 51004 On the Prowl: Hero Action (thwart) — remove 3 threat from a
// scheme, then resolve the Special ability on 1 Black Panther upgrade
// you control.
func registerOnTheProwl() {
	engine.RegisterBehavior("51004", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			pid := e.EOwner()
			p := g.Player(pid)
			if p == nil {
				return nil
			}
			special := specialQuestion(g, p)
			var choices []engine.Choice
			for _, id := range g.Schemes() {
				s := g.Entity(id)
				ch := engine.Choice{
					Label: engine.S(s.EDef().Name), Kind: engine.ChoiceTarget,
					SourceID: id, CardCode: s.ECode(),
				}.Msgs(engine.ThwartScheme{Scheme: id, N: 3, Source: pid})
				if special != nil {
					ch = ch.WithThen(special)
				}
				choices = append(choices, ch)
			}
			if len(choices) == 0 {
				return nil
			}
			return []engine.Message{engine.AskQuestion{
				Player:   pid,
				Question: engine.Ask(engine.Tf("c.onTheProwlRemove3ThreatFromAScheme"), choices...),
			}}
		},
	})
}

// 51005 Wakanda Forever!: Hero Action — resolve the Special ability on
// each Black Panther upgrade you control in any order. (Approximation:
// board order instead of a free order; unlike the core-set event of the
// same name there is no "final step" bonus.)
func registerWakandaForever() {
	engine.RegisterBehavior("51005", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			for _, u := range specialUpgrades(g, p) {
				msgs = append(msgs, specialExecute(g, u.ID)...)
			}
			if len(msgs) == 0 {
				g.TLogf("c.wakandaForeverNoBlackPantherUpgradesInPlay")
			}
			return msgs
		},
	})
}

// 51007 The Elephant's Trunk: Alter-Ego Action — exhaust this card and
// up to 2 other Wakanda allies and/or supports you control → draw 1
// card for each card exhausted this way (including this one).
func registerElephantsTrunk() {
	engine.RegisterBehavior("51007", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || p.IsHero() {
				return nil
			}
			return []engine.Ability{{
				Label:        engine.Tf("c.theElephantSTrunkExhaustItAndUpTo2OtherWakandaCardsDrawThatM"),
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(e.EOwner())
					if pl == nil {
						return nil
					}
					// The Trunk itself always contributes one draw.
					msgs := []engine.Message{engine.DrawCards{Player: pl.ID, N: 1}}
					var picks []engine.Choice
					add := func(id engine.EntityID, code, name string) {
						picks = append(picks, engine.Choice{
							Label: engine.S(name), Kind: engine.ChoiceCard, CardCode: code, SourceID: id,
						}.Msgs(
							engine.ExhaustEntity{ID: id},
							engine.DrawCards{Player: pl.ID, N: 1},
						))
					}
					for _, id := range pl.Allies {
						if a := g.Allies[id]; a != nil && !a.Exhausted && a.EDef().HasTrait("wakanda") {
							add(id, a.Code, a.EDef().Name)
						}
					}
					for _, id := range pl.Supports {
						s := g.Supports[id]
						if s == nil || s.Exhausted || s.ID == e.EID() || !s.EDef().HasTrait("wakanda") {
							continue
						}
						add(id, s.Code, s.EDef().Name)
					}
					if len(picks) == 0 {
						return msgs
					}
					msgs = append(msgs, engine.AskQuestion{
						Player:   pl.ID,
						Question: engine.AskN(engine.Tf("c.theElephantSTrunkExhaustUpTo2OtherWakandaCards"), 2, picks...),
					})
					return msgs
				},
			}}
		},
	})
}

// 51008 Queen Ramonda: Alter-Ego Action — exhaust → heal damage from an
// alter-ego with the Wakanda trait equal to that alter-ego's REC.
func registerQueenRamonda() {
	engine.RegisterBehavior("51008", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil || p.IsHero() {
				return nil
			}
			return []engine.Ability{{
				Label:        engine.Tf("c.queenRamondaHealAnAlterEgoWithTheWakandaTraitEqualToItsRec"),
				Type:         engine.AbilityAction,
				Exhaust:      true,
				AlterEgoOnly: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					var choices []engine.Choice
					for _, q := range g.Players {
						if q.IsHero() || !q.AlterEgoDef().HasTrait("wakanda") || q.Damage == 0 {
							continue
						}
						rec := q.RecoverStat(g)
						if rec < 1 {
							continue
						}
						choices = append(choices, engine.Choice{
							Label: engine.S(q.Name + " (heal " + itoa(rec) + ")"), Kind: engine.ChoiceTarget, SourceID: q.ID,
						}.Msgs(engine.HealEntity{Target: q.ID, N: rec}))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   e.EOwner(),
						Question: engine.Ask(engine.Tf("c.queenRamondaHealWhichAlterEgo"), choices...),
					}}
				},
			}}
		},
	})
}

// 51009 Aja-Adanna: Action — exhaust → shuffle 1 identity-specific card
// from your discard pile into your deck.
func registerAjaAdanna() {
	engine.RegisterBehavior("51009", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			p := g.Player(e.EOwner())
			if p == nil {
				return nil
			}
			has := false
			for _, c := range p.Discard {
				if c.Def().CardSet == "black_panther_shuri" {
					has = true
					break
				}
			}
			if !has {
				return nil
			}
			return []engine.Ability{{
				Label:   engine.Tf("c.ajaAdannaShuffle1IdentitySpecificCardFromYourDiscardIntoYour"),
				Type:    engine.AbilityAction,
				Exhaust: true,
				Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
					pl := g.Player(e.EOwner())
					if pl == nil {
						return nil
					}
					var choices []engine.Choice
					seen := map[string]bool{}
					for _, c := range pl.Discard {
						if c.Def().CardSet != "black_panther_shuri" || seen[c.Code] {
							continue
						}
						seen[c.Code] = true
						choices = append(choices, engine.Choice{
							Label: engine.S(c.Def().Name), Kind: engine.ChoiceCard, CardCode: c.Code,
						}.Msgs(engine.ShuffleIntoDeck{Player: pl.ID, CardID: c.ID}))
					}
					if len(choices) == 0 {
						return nil
					}
					return []engine.Message{engine.AskQuestion{
						Player:   pl.ID,
						Question: engine.Ask(engine.Tf("c.ajaAdannaShuffleWhichCardIntoYourDeck"), choices...),
					}}
				},
			}}
		},
	})
}

// 51010 Kimoyo Beads: Special (thwart) — remove 1 threat from a scheme.
// You may discard this card to confuse an enemy.
func registerKimoyoBeads() {
	engine.RegisterBehavior("51010", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return specialAbility(g, e, func(u *engine.Upgrade) []engine.Message {
				choices := cardutil.SchemeChoices(g, func(id engine.EntityID) []engine.Message {
					return []engine.Message{engine.ThwartScheme{Scheme: id, N: 1, Source: u.Owner}}
				})
				if len(choices) == 0 {
					return nil
				}
				var discardChoices []engine.Choice
				discardChoices = append(discardChoices, engine.Choice{
					ID: "keep", Label: engine.Tf("c.keepKimoyoBeads"), Kind: engine.ChoicePass,
				})
				for _, id := range cardutil.SortedEnemyIDs(g) {
					enemy := g.Entity(id)
					discardChoices = append(discardChoices, engine.Choice{
						Label: engine.S("Discard → confuse " + enemy.EDef().Name), Kind: engine.ChoiceTarget,
						SourceID: id, CardCode: enemy.ECode(),
					}.Msgs(
						engine.DiscardControlled{Player: u.Owner, ID: u.ID},
						engine.ConfuseEntity{Target: id},
					))
				}
				discardQ := engine.Ask(engine.Tf("c.kimoyoBeadsDiscardItToConfuseAnEnemy"), discardChoices...)
				for i := range choices {
					choices[i] = choices[i].WithThen(discardQ)
				}
				return []engine.Message{engine.AskQuestion{
					Player:   u.Owner,
					Question: engine.Ask(engine.Tf("c.kimoyoBeadsRemove1ThreatFromAScheme"), choices...),
				}}
			})
		},
	})
}

// 51011 Panther Claws: Special (attack) — deal 2 damage to an enemy.
// You may discard this card to deal 3 additional damage to that enemy
// (that attack gains piercing; piercing is not modeled).
func registerPantherClaws() {
	engine.RegisterBehavior("51011", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return specialAbility(g, e, func(u *engine.Upgrade) []engine.Message {
				var choices []engine.Choice
				for _, id := range cardutil.SortedEnemyIDs(g) {
					enemy := g.Entity(id)
					discardQ := engine.Ask(engine.Tf("c.pantherClawsDiscardItToDeal3AdditionalDamage"),
						engine.Choice{ID: "keep", Label: engine.Tf("c.keepPantherClaws"), Kind: engine.ChoicePass},
						engine.Choice{ID: "discard", Label: engine.Tf("c.discardPantherClaws3Damage"), Kind: engine.ChoiceLabel}.Msgs(
							engine.DiscardControlled{Player: u.Owner, ID: u.ID},
							engine.DamageEntity{Target: id, Damage: 3, Source: u.ID},
						),
					)
					choices = append(choices, engine.Choice{
						Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
						SourceID: id, CardCode: enemy.ECode(),
					}.Msgs(
						engine.DamageEntity{Target: id, Damage: 2, Source: u.ID},
					).WithThen(discardQ))
				}
				if len(choices) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{
					Player:   u.Owner,
					Question: engine.Ask(engine.Tf("c.pantherClawsDeal2DamageToAnEnemy"), choices...),
				}}
			})
		},
	})
}

// 51012 Spider Bites: Special — choose a player; deal 1 damage to the
// villain and each minion engaged with that player. You may discard
// this card to stun each of those enemies.
func registerSpiderBites() {
	engine.RegisterBehavior("51012", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return specialAbility(g, e, func(u *engine.Upgrade) []engine.Message {
				var choices []engine.Choice
				for _, q := range g.Players {
					var targets []engine.EntityID
					for _, id := range cardutil.SortedIDs(g.Villains) {
						targets = append(targets, id)
					}
					for _, id := range cardutil.SortedIDs(g.Minions) {
						if mn := g.Minions[id]; mn != nil && mn.EngagedWith == q.ID {
							targets = append(targets, id)
						}
					}
					if len(targets) == 0 {
						continue
					}
					var dmg, stun []engine.Message
					for _, id := range targets {
						dmg = append(dmg, engine.DamageEntity{Target: id, Damage: 1, Source: u.ID})
						stun = append(stun, engine.StunEntity{Target: id})
					}
					discardQ := engine.Ask(engine.Tf("c.spiderBitesDiscardItToStunEachOfThoseEnemies"),
						engine.Choice{ID: "keep", Label: engine.Tf("c.keepSpiderBites"), Kind: engine.ChoicePass},
						engine.Choice{ID: "discard", Label: engine.Tf("c.discardSpiderBitesStunThem"), Kind: engine.ChoiceLabel}.
							Msgs(append([]engine.Message{engine.DiscardControlled{Player: u.Owner, ID: u.ID}}, stun...)...),
					)
					choices = append(choices, engine.Choice{
						Label: engine.S(q.Name), Kind: engine.ChoiceTarget, SourceID: q.ID,
					}.Msgs(dmg...).WithThen(discardQ))
				}
				if len(choices) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{
					Player:   u.Owner,
					Question: engine.Ask(engine.Tf("c.spiderBitesChooseAPlayer1DamageToTheVillainAndTheirEngagedMi"), choices...),
				}}
			})
		},
	})
}

// 51013 Vibranium Suit: Special (attack) — move 1 damage from your hero
// to an enemy. You may discard this card to give your hero a tough
// status card. (Approximation: the "move" still deals the damage when
// the hero has no damage to move.)
func registerVibraniumSuit() {
	engine.RegisterBehavior("51013", &engine.Behavior{
		Abilities: func(g *engine.Game, e engine.Entity) []engine.Ability {
			return specialAbility(g, e, func(u *engine.Upgrade) []engine.Message {
				var choices []engine.Choice
				for _, id := range cardutil.SortedEnemyIDs(g) {
					enemy := g.Entity(id)
					discardQ := engine.Ask(engine.Tf("c.vibraniumSuitDiscardItToGiveYourHeroAToughStatusCard"),
						engine.Choice{ID: "keep", Label: engine.Tf("c.keepVibraniumSuit"), Kind: engine.ChoicePass},
						engine.Choice{ID: "discard", Label: engine.Tf("c.discardVibraniumSuitTough"), Kind: engine.ChoiceLabel}.Msgs(
							engine.DiscardControlled{Player: u.Owner, ID: u.ID},
							engine.ToughEntity{Target: u.Owner},
						),
					)
					choices = append(choices, engine.Choice{
						Label: cardutil.EnemyLabel(enemy), Kind: engine.ChoiceTarget,
						SourceID: id, CardCode: enemy.ECode(),
					}.Msgs(
						engine.HealEntity{Target: u.Owner, N: 1},
						engine.DamageEntity{Target: id, Damage: 1, Source: u.ID},
					).WithThen(discardQ))
				}
				if len(choices) == 0 {
					return nil
				}
				return []engine.Message{engine.AskQuestion{
					Player:   u.Owner,
					Question: engine.Ask(engine.Tf("c.vibraniumSuitMove1DamageFromYourHeroToAnEnemy"), choices...),
				}}
			})
		},
	})
}

// specialAbility wraps a Black Panther upgrade's Special as a triggered
// "wakanda" ability, excluded from the action menu and fired by the
// effects that resolve Specials (the same contract the core Black
// Panther upgrades use, so both Wakanda Forever! events interoperate).
func specialAbility(g *engine.Game, e engine.Entity, run func(u *engine.Upgrade) []engine.Message) []engine.Ability {
	return []engine.Ability{{
		Label: engine.Tf("c.special"), Type: engine.AbilityTrigger, Trigger: specialTrigger,
		Execute: func(g *engine.Game, self engine.EntityID) []engine.Message {
			u := g.Upgrades[self]
			if u == nil {
				return nil
			}
			return run(u)
		},
	}}
}

// itoa avoids pulling strconv into the label builders for small ints.
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

// registerNemesis installs the Black Panther (Shuri) nemesis set: Klaw,
// Manipulated M.U.S.I.C. / M.U.S.I.C., and The Scream.
func registerNemesis() {
	// 51032 Klaw: Forced Interrupt — when Klaw attacks, give him 1 boost
	// card for this activation. (Approximation: any activation of Klaw
	// gets the boost; engaged minions nearly always attack.)
	engine.RegisterBehavior("51032", &engine.Behavior{
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionActivates)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			g.TLogf("c.klawTakes1BoostCardForThisActivation")
			return []engine.Message{engine.BoostActivation{Enemy: e.EID(), N: 1}}
		},
	})

	// 51033 Manipulated M.U.S.I.C.: When Revealed — find M.U.S.I.C. and
	// put her into play engaged with you. When Defeated — discard
	// M.U.S.I.C. from play. ("Find" searches the nemesis discard, then
	// the encounter deck and discard.)
	engine.RegisterBehavior("51033", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := shuriPlayer(g)
			if p == nil {
				return nil
			}
			if card, ok := findCard(g, p, "51034"); ok {
				mn := &engine.Minion{
					ID:        g.NextEntityID(engine.KindMinion),
					Code:      "51034",
					MaxHP:     derefInt(card.Def().HP, 1),
					AttackVal: derefInt(card.Def().Attack, 0),
					SchemeVal: derefInt(card.Def().Scheme, 0),
				}
				g.Minions[mn.ID] = mn
				mn.EngagedWith = p.ID
				g.TLogf("c.manipulatedMUSICMUSICEntersPlayEngagedWith", p.Name)
				return []engine.Message{engine.MinionEntersPlay{MinionID: mn.ID, Player: p.ID}}
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.SchemeDefeated)
			if !ok || m.Scheme != e.EID() {
				return nil
			}
			for _, id := range cardutil.SortedIDs(g.Minions) {
				if mn := g.Minions[id]; mn != nil && mn.Code == "51034" {
					g.TLogf("c.manipulatedMUSICDefeatedMUSICIsDiscarded")
					g.Delete(id)
				}
			}
			return nil
		},
	})

	// 51034 M.U.S.I.C.: When Revealed — find Manipulated M.U.S.I.C. and
	// put it into play. When Defeated — move all threat from Manipulated
	// M.U.S.I.C. to the main scheme.
	engine.RegisterBehavior("51034", &engine.Behavior{
		OnPlay: func(g *engine.Game, e engine.Entity) []engine.Message {
			p := shuriPlayer(g)
			if p == nil {
				return nil
			}
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				if ss := g.SideSchemes[id]; ss != nil && ss.Code == "51033" {
					return nil // already in play
				}
			}
			if card, ok := findCard(g, p, "51033"); ok {
				s := &engine.SideScheme{
					ID:        g.NextEntityID(engine.KindSideScheme),
					Code:      "51033",
					Threat:    derefInt(card.Def().BaseThreat, 1),
					MaxThreat: derefInt(card.Def().BaseThreat, 1) * 2,
				}
				g.SideSchemes[s.ID] = s
				g.TLogf("c.mUSICManipulatedMUSICEntersPlay")
			}
			return nil
		},
		React: func(g *engine.Game, e engine.Entity, msg engine.Message) []engine.Message {
			m, ok := msg.(engine.MinionDefeated)
			if !ok || m.MinionID != e.EID() {
				return nil
			}
			for _, id := range cardutil.SortedIDs(g.SideSchemes) {
				ss := g.SideSchemes[id]
				if ss == nil || ss.Code != "51033" || ss.Threat == 0 {
					continue
				}
				n := ss.Threat
				ss.Threat = 0
				if g.MainScheme != nil {
					g.TLogf("c.mUSICDefeatedThreatMovesToTheMainScheme", n)
					return []engine.Message{engine.SchemeThreat{Scheme: g.MainScheme.ID, N: n, Source: e.EID()}}
				}
			}
			return nil
		},
	})

	// 51035 The Scream: When Revealed — stun each character you control;
	// deal 1 damage to each of those characters that was already
	// stunned.
	engine.RegisterBehavior("51035", &engine.Behavior{
		ResolveTreachery: func(g *engine.Game, t *engine.Treachery, p *engine.Player) []engine.Message {
			g.Delete(t.ID)
			var msgs []engine.Message
			if p.Stunned {
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: t.ID})
			}
			msgs = append(msgs, engine.StunEntity{Target: p.ID})
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil {
					if a.Stunned {
						msgs = append(msgs, engine.DamageEntity{Target: id, Damage: 1, Source: t.ID})
					}
					msgs = append(msgs, engine.StunEntity{Target: id})
				}
			}
			g.TLogf("c.theScreamSCharactersAreStunned", p.Name)
			return msgs
		},
		// Boost: you are stunned; if you were already stunned, take 1
		// damage. (Approximation: the boost applies to the first player;
		// the boost hook carries no attacked-player context.)
		Boost: func(g *engine.Game, card engine.Card) []engine.Message {
			p := g.Player(cardutil.FirstPlayerID(g))
			if p == nil {
				return nil
			}
			var msgs []engine.Message
			if p.Stunned {
				msgs = append(msgs, engine.DamageEntity{Target: p.ID, Damage: 1, Source: ""})
			}
			msgs = append(msgs, engine.StunEntity{Target: p.ID})
			return msgs
		},
	})
}

// findCard locates a card by code across the nemesis discard, the
// encounter deck and the encounter discard, removing it from the zone
// it was found in.
func findCard(g *engine.Game, p *engine.Player, code string) (engine.Card, bool) {
	for _, c := range p.NemesisDiscard {
		if c.Code == code {
			p.NemesisDiscard.Remove(c.ID)
			return c, true
		}
	}
	for _, c := range g.EncounterDeck {
		if c.Code == code {
			g.EncounterDeck.Remove(c.ID)
			return c, true
		}
	}
	for _, c := range g.EncounterDiscard {
		if c.Code == code {
			g.EncounterDiscard.Remove(c.ID)
			return c, true
		}
	}
	return engine.Card{}, false
}

// derefInt reads an optional printed stat.
func derefInt(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

// registerObligation installs T'Challa's Shadow (51031).
//
// The printed obligation stays in play with 4 doubt counters, raises
// the cost of each card the owner plays by 1, and sheds a counter after
// each basic power the owner uses. The engine has no persistent
// in-play obligation zone, so this is approximated as an immediate
// resolution: the obligation is discarded and the lingering cost
// increase is not modeled.
func registerObligation() {
	engine.RegisterBehavior("51031", &engine.Behavior{
		ResolveObligation: func(g *engine.Game, p *engine.Player, card engine.Card) []engine.Message {
			if p == nil {
				return nil
			}
			g.TLogf("c.tChallaSShadowTheDoubtCounters1CostEffectIsNotModeledTheObli")
			return []engine.Message{engine.ObligationResolve{Player: p.ID, Card: card}}
		},
	})
}
