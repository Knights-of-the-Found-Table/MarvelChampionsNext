package engine

import (
	"fmt"
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// handle implements the core semantics for each message type. Entities have
// already reacted (interrupts) before these run.
func (g *Game) handle(msg Message) {
	switch m := msg.(type) {
	case StartGame:
		g.handleStartGame()

	case BeginRound:
		g.Round++
		g.UsedThisRound = map[string]bool{}
		for _, p := range g.Players {
			p.AllyPlayedThisRound = false
		}
		g.tlogMajorf("log.round", g.Round)
		g.Push(BeginPhase{Phase: PhasePlayer})

	case BeginPhase:
		g.Phase = m.Phase
		g.handleBeginPhase(m.Phase)

	case EndPhase:
		switch m.Phase {
		case PhaseResource:
			// Legacy saves only: games persisted with the old
			// resource→player→villain round structure.
			g.expirePhaseEffects()
			g.Push(BeginPhase{Phase: PhasePlayer})
		case PhasePlayer:
			// End of player phase, official steps 1–2: each player in
			// player order may discard any number of cards (and must
			// discard down to hand size); then all players draw up and
			// ready (FinishPlayerPhase) before the phase's effects expire.
			for _, p := range g.playerOrder() {
				if !p.KOed {
					g.Push(DiscardToHandSize{Player: p.ID})
				}
			}
			g.Push(FinishPlayerPhase{})
		case PhaseVillain:
			// Official villain phase steps 5–6: the first player token
			// passes clockwise, then the phase/round's until-end effects
			// expire (handled when EndRound processes).
			g.Push(PassFirstPlayerToken{})
			g.Push(EndRound{})
		}

	case FinishPlayerPhase:
		// End of player phase steps 2–4: draw up to hand size, ready all
		// cards (including exhausted encounter cards), then expire
		// until-end-of-player-phase effects.
		for _, p := range g.playerOrder() {
			if p.KOed {
				continue
			}
			g.Push(DrawCards{Player: p.ID, N: max(0, p.HandSize(g)-len(p.Hand))})
			g.Push(ReadyAll{Player: p.ID})
		}
		for _, id := range sortedIDs(g.Environments) {
			if e := g.Environments[id]; e != nil {
				e.Exhausted = false
			}
		}
		g.expirePhaseEffects()
		g.tlogf("log.playerPhaseEnds")
		g.Push(BeginPhase{Phase: PhaseVillain})

	case PassFirstPlayerToken:
		if len(g.Players) < 2 {
			return
		}
		for _, p := range g.Players {
			if !p.FirstPlayer {
				continue
			}
			if next := g.nextActivePlayer(p.ID); next != nil && next.ID != p.ID {
				p.FirstPlayer = false
				next.FirstPlayer = true
				g.tlogf("log.firstPlayerPasses", next.Name)
			}
			return
		}

	case DiscardToHandSize:
		g.askDiscardToHandSize(m.Player)

	case ResolveMulligan:
		g.askMulligan(m.Player)

	case MulliganCard:
		p := g.Player(m.Player)
		if p == nil {
			return
		}
		if c, ok := p.Hand.Remove(m.CardID); ok {
			p.Discard = append(p.Discard, c)
			g.tlogf("log.mulligansAway", p.Name, c)
			g.Push(DrawCards{Player: p.ID, N: 1})
		}

	case EndRound:
		// End-of-round: until-end-of-villain-phase/round effects expire,
		// then the next round begins.
		g.expirePhaseEffects()
		g.Push(BeginRound{})

	case PlayerTurnStart:
		p := g.Player(m.Player)
		if p == nil {
			return
		}
		p.FormChanged = false
		p.EndedTurn = false
		g.ActiveTurn = p.ID
		g.tlogf("log.turnBegins", p.Name)

	case PlayerTurnEnd:
		p := g.Player(m.Player)
		if p != nil {
			p.EndedTurn = true
		}
		g.ActiveTurn = ""
		// next player who hasn't taken a turn this round
		for i := range g.Players {
			q := g.Players[(g.TurnIndex+i)%len(g.Players)]
			if !q.EndedTurn && !q.KOed {
				g.TurnIndex = (g.TurnIndex + i) % len(g.Players)
				g.Push(PlayerTurnStart{Player: q.ID})
				return
			}
		}
		g.Push(EndPhase{Phase: PhasePlayer})

	case ReadyAll:
		p := g.Player(m.Player)
		if p == nil {
			return
		}
		p.Exhausted = false
		for _, id := range p.Allies {
			if a := g.Allies[id]; a != nil {
				// Captive Hope (40131): Hope Summers cannot ready.
				if engineHopeLocked(g, a) {
					continue
				}
				a.Exhausted = false
			}
		}
		for _, id := range p.Supports {
			if s := g.Supports[id]; s != nil {
				s.Exhausted = false
			}
		}
		for _, id := range p.Upgrades {
			if u := g.Upgrades[id]; u != nil {
				u.Exhausted = false
			}
		}

	case ReadyEntity:
		if a, ok := g.Allies[m.ID]; ok && engineHopeLocked(g, a) {
			g.tlogf("log.hopeCannotReady")
			return
		}
		if e := g.Entity(m.ID); e != nil {
			g.setExhausted(m.ID, false)
		}

	case ExhaustEntity:
		g.setExhausted(m.ID, true)

	case DrawCards:
		p := g.Player(m.Player)
		if p == nil {
			return
		}
		for i := 0; i < m.N; i++ {
			if len(p.Deck) == 0 {
				if len(p.Discard) == 0 {
					g.tlogf("log.noCardsToDraw", p.Name)
					break
				}
				// Player deck emptied: shuffle the discard pile into a new
				// deck and deal self one facedown encounter card.
				p.Deck = p.Discard
				p.Discard = nil
				g.shuffle(&p.Deck)
				g.tlogMajorf("log.shufflesNewDeck", p.Name)
				if c, ok := g.drawEncounter(); ok {
					p.EncounterDown = append(p.EncounterDown, c)
					g.tlogf("log.dealsFacedown", p.Name)
				}
			}
			card := p.Deck[0]
			p.Deck = p.Deck[1:]
			p.Hand = append(p.Hand, card)
		}

	case ShufflePlayerDeck:
		if p := g.Player(m.Player); p != nil {
			g.shuffle(&p.Deck)
			g.tlogf("log.shufflesDeck", p.Name)
		}

	case DiscardCards:
		p := g.Player(m.Player)
		if p == nil {
			return
		}
		for _, c := range m.Cards {
			p.Hand.Remove(c.ID)
			p.Discard = append(p.Discard, c)
		}

	case ChangeForm:
		p := g.Player(m.Player)
		if p == nil || p.FormChanged {
			return
		}
		p.FormChanged = true
		if p.IsHero() {
			p.Side = SideAlterEgo
			g.tlogf("log.changesTo", p.Name, p.AlterEgoDef())
		} else {
			p.Side = SideHero
			g.tlogf("log.changesTo", p.Name, p.HeroDef())
		}

	case PlayCard:
		g.handlePlayCard(m)

	case ResourcePay:
		p := g.Player(m.Player)
		if p == nil {
			return
		}
		for _, c := range m.Cards {
			if _, ok := p.Hand.Remove(c.ID); ok {
				p.Discard = append(p.Discard, c)
			}
		}

	case BasicThwart:
		p := g.Player(m.Player)
		if p == nil {
			return
		}
		if p.Confused {
			p.Confused = false
			g.tlogf("log.confusedCannotThwart", p.Name)
			return
		}
		if g.thwartBlocked(p) {
			g.tlogf("log.cannotThwartEngaged", p.Name, g.thwartBlockerName(p))
			return
		}
		p.Exhausted = true
		g.Push(ThwartScheme{Scheme: m.Target, N: m.N, Source: p.ID})
		g.Push(WindowAfterThwarted{Player: p.ID, Scheme: m.Target})

	case BasicAttack:
		p := g.Player(m.Player)
		if p == nil {
			return
		}
		if p.Stunned {
			p.Stunned = false
			g.tlogf("log.stunnedCannotAttack", p.Name)
			return
		}
		p.Exhausted = true
		g.Push(DamageEntity{Target: m.Target, Damage: m.N, Source: p.ID})
		// Retaliate
		if e := g.Entity(m.Target); e != nil {
			if v := retaliateOf(g, e); v > 0 {
				g.Push(DamageEntity{Target: p.ID, Damage: v, Source: m.Target})
			}
		}

	case BasicRecover:
		p := g.Player(m.Player)
		if p == nil {
			return
		}
		p.Exhausted = true
		g.Push(HealEntity{Target: p.ID, N: p.RecoverStat(g)})

	case VillainActivates:
		g.handleVillainActivates(m)

	case MinionActivates:
		g.handleMinionActivates(m)

	case MinionActivations:
		g.beginMinionActivations(m.Player)

	case AskMinionOrder:
		g.handleAskMinionOrder(m)

	case AskAttack:
		g.handleAskAttack(m)

	case OtherDefenders:
		g.handleOtherDefenders(m)

	case AskOtherAction:
		g.handleAskOtherAction(m)

	case SchemeThreat:
		// Hinder keyword: while side schemes with Hinder X are in play, X
		// fewer threat is placed on the main scheme (each placement).
		if m.N > 0 && g.MainScheme != nil && m.Scheme == g.MainScheme.ID {
			if h := g.hinderTotal(); h > 0 {
				if h >= m.N {
					g.tlogf("log.hinderPreventsAll", m.N, g.MainScheme)
					return
				}
				g.tlogf("log.hinderPrevents", h, g.MainScheme)
				m.N -= h
			}
		}
		// Not my Responsibility (44022): a player holding the event may
		// redirect the threat to themselves or an ally as damage.
		if m.N > 0 {
			if p, card, ok := g.handCardHolding("44022"); ok {
				var nmrChoices []Choice
				nmrChoices = append(nmrChoices,
					Choice{ID: "nmr-self", Label: Tf("c.playNotMyResponsibilityTakesDamage", p.Name, m.N), Kind: ChoicePlay, CardCode: card.Code}.
						Msgs(ConsumeHandCard{Player: p.ID, CardID: card.ID},
							DamageEntity{Target: p.ID, Damage: m.N, Source: p.ID}))
				for _, aid := range p.Allies {
					if a := g.Allies[aid]; a != nil && a.HP() > m.N {
						nmrChoices = append(nmrChoices,
							Choice{ID: "nmr-ally-" + aid.String(), Label: Tf("c.takesTheDamageInstead", a, m.N), Kind: ChoicePlay, CardCode: card.Code}.
								Msgs(ConsumeHandCard{Player: p.ID, CardID: card.ID},
									DamageEntity{Target: aid, Damage: m.N, Source: p.ID}))
					}
				}
				nmrChoices = append(nmrChoices,
					Choice{ID: "nmr-pass", Label: Tf("m.pass"), Kind: ChoicePass}.
						Msgs(ApplySchemeThreat{Scheme: m.Scheme, N: m.N, Source: m.Source}))
				g.Push(AskQuestion{Player: p.ID, Question: Ask(
					Tf("q.nmrPrompt", p.Name, m.N), nmrChoices...)})
				return
			}
		}
		// Great Responsibility interrupt window (01061/30015): a player
		// holding the event may take the threat as damage instead.
		if m.N > 0 {
			if p, card, ok := g.handCardHolding("01061", "30015"); ok {
				g.Push(AskQuestion{Player: p.ID, Question: Ask(
					Tf("q.grPrompt", p.Name, m.N),
					Choice{ID: "gr-play", Label: Tf("m.playGR", m.N), Kind: ChoicePlay, CardCode: card.Code}.
						Msgs(ConsumeHandCard{Player: p.ID, CardID: card.ID},
							DamageEntity{Target: p.ID, Damage: m.N, Source: p.ID}),
					Choice{ID: "gr-pass", Label: Tf("m.pass"), Kind: ChoicePass}.
						Msgs(ApplySchemeThreat{Scheme: m.Scheme, N: m.N, Source: m.Source}),
				)})
				return
			}
		}
		g.addThreat(m.Scheme, m.N, m.Source)

	case ApplySchemeThreat:
		g.addThreat(m.Scheme, m.N, m.Source)

	case ThwartScheme:
		g.removeThreat(m.Scheme, m.N, m.Source)

	case DamageEntity:
		// Warning interrupt window (09021): the damaged hero's player may
		// play Warning to reduce the damage by 1 (approximation: only the
		// damaged player's own copy triggers).
		if m.Damage > 0 && m.Target.Is(KindPlayer) && !m.Unpreventable {
			p := g.Player(PlayerID(m.Target))
			if p != nil && !p.KOed {
				var warn Card
				found := false
				for _, hc := range p.Hand {
					if data.BaseCode(hc.Code) == "09021" {
						warn, found = hc, true
						break
					}
				}
				if found {
					card := warn
					g.Push(AskQuestion{Player: p.ID, Question: Ask(
						Tf("q.warningPrompt", p.Name, m.Damage, m.Damage-1),
						Choice{ID: "warn-play", Label: Tf("m.playWarning"), Kind: ChoicePlay, CardCode: card.Code}.
							Msgs(ConsumeHandCard{Player: p.ID, CardID: card.ID},
								ApplyDamage{Target: m.Target, Damage: m.Damage - 1, Source: m.Source}),
						Choice{ID: "warn-pass", Label: Tf("m.pass"), Kind: ChoicePass}.
							Msgs(ApplyDamage{Target: m.Target, Damage: m.Damage, Source: m.Source}),
					)})
					return
				}
			}
		}
		g.damage(m.Target, m.Damage, m.Source, m.Unpreventable)

	case ApplyDamage:
		g.damage(m.Target, m.Damage, m.Source, m.Unpreventable)

	case CancelBoostIcons:
		if v := g.Villains[m.Enemy]; v != nil && m.N > 0 {
			v.BoostCount -= m.N
			if v.BoostCount < 0 {
				v.BoostCount = 0
			}
			g.tlogf("log.boostCancelled", m.N, v.BoostCount)
		}

	case GuessCheck:
		if p := g.Player(m.Player); p != nil {
			if def, ok := DB.Lookup(m.CardCode); ok {
				if def.Type == m.Guess {
					g.Logf("%s guessed right — draws 1 card", p.Name)
					g.Push(DamageEntity{Target: m.Penalty, Damage: 1, Source: p.ID}, DrawCards{Player: p.ID, N: 1})
				} else {
					g.Logf("%s guessed wrong (%s is not a %s)", p.Name, def.Name, m.Guess)
				}
			}
		}

	case EncounterTakeCard:
		g.EncounterDeck.Remove(m.CardID)

	case ShuffleEncounterDeck:
		for i := len(g.EncounterDeck) - 1; i > 0; i-- {
			j := g.Random(i + 1)
			g.EncounterDeck[i], g.EncounterDeck[j] = g.EncounterDeck[j], g.EncounterDeck[i]
		}
		g.tlogf("log.encounterShuffled")

	case ConvertMinionToAlly:
		mn := g.Minions[m.MinionID]
		p := g.Player(m.Owner)
		if mn == nil || p == nil {
			return
		}
		code, sch, atk := mn.Code, mn.SchemeVal, mn.AttackVal
		hp := mn.MaxHP - mn.Damage
		g.Delete(mn.ID)
		a := &Ally{
			ID: g.nextEntityID(KindAlly), Code: code, Owner: p.ID,
			MaxHP: hp + m.Consequential, Damage: m.Consequential,
			ThwartVal: sch, AttackVal: atk,
		}
		g.Allies[a.ID] = a
		p.Allies = append(p.Allies, a.ID)
		g.tlogMajorf("log.takesControlBlank", p.Name, a)
		g.Push(AllyEnteredPlay{Ally: a.ID, Player: p.ID})

	case AddMagnetCounter:
		if s := g.MainScheme; s != nil && s.ID == m.Scheme {
			s.Counters++
			g.Logf("A magnet counter is placed on %s (%d)", s.EDef().Name, s.Counters)
			if s.Counters >= 3 {
				s.Counters -= 3
				for guards := 0; guards < 40; guards++ {
					if len(g.EncounterDeck) == 0 {
						if len(g.EncounterDiscard) == 0 {
							return
						}
						g.EncounterDeck = g.EncounterDiscard
						g.EncounterDiscard = nil
						for i := len(g.EncounterDeck) - 1; i > 0; i-- {
							j := g.Random(i + 1)
							g.EncounterDeck[i], g.EncounterDeck[j] = g.EncounterDeck[j], g.EncounterDeck[i]
						}
					}
					top := g.EncounterDeck[0]
					g.EncounterDeck = g.EncounterDeck[1:]
					if top.Def().HasTrait("magnetic") {
						g.Push(RevealEncounterCard{Player: PlayerID(g.playerOrder()[0].ID), Card: top})
						return
					}
					g.EncounterDiscard = append(g.EncounterDiscard, top)
				}
			}
		}

	case TuckCardUnderOZT:
		// Mill the player's top deck card facedown under the Operation
		// Zero Tolerance side scheme (32104).
		for _, s := range g.SideSchemes {
			if s == nil || s.Code != "32104" {
				continue
			}
			if p := g.Player(m.Player); p != nil && len(p.Deck) > 0 {
				top := p.Deck[0]
				p.Deck = p.Deck[1:]
				s.StoredCards = append(s.StoredCards, top)
				g.Logf("%s's %s is placed facedown under Operation Zero Tolerance", p.Name, top.Def().Name)
			}
		}

	case ShuffleMinionIntoDeck:
		if mn := g.Minions[m.MinionID]; mn != nil {
			code := mn.Code
			g.Delete(mn.ID)
			g.EncounterDeck = append(g.EncounterDeck, Card{ID: g.nextCardID(), Code: code})
			for i := len(g.EncounterDeck) - 1; i > 0; i-- {
				j := g.Random(i + 1)
				g.EncounterDeck[i], g.EncounterDeck[j] = g.EncounterDeck[j], g.EncounterDeck[i]
			}
			g.tlogf("log.shuffledIntoEncounter", DB.MustLookup(code))
		}

	case AttachHandCard:
		if p := g.Player(m.Player); p != nil {
			if c, ok := p.Hand.Remove(m.CardID); ok {
				if mn := g.Minions[m.Enemy]; mn != nil {
					n := 0
					for _, icon := range c.Def().Resources {
						if icon == "energy" {
							n++
						}
					}
					mn.MaxHP += n
					mn.TuckedCards = append(mn.TuckedCards, c)
					g.tlogf("log.attachesFacedown", mn, c, n)
				}
			}
		}

	case HealEntity:
		g.heal(m.Target, m.N)

	case StunEntity:
		g.setStatus(m.Target, "stunned", true)
	case ConfuseEntity:
		g.setStatus(m.Target, "confused", true)
	case ToughEntity:
		g.setStatus(m.Target, "tough", true)
	case ClearStun:
		g.setStatus(m.Target, "stunned", false)
	case ClearConfuse:
		g.setStatus(m.Target, "confused", false)
	case ClearTough:
		g.setStatus(m.Target, "tough", false)
	case ClearAllTough:
		g.discardAllTough(m.Target)
	case ToughDiscarded:
		// Window only: entities react via React on this message.

	case Defends:
		g.handleDefends(m)

	case DealBoost:
		if v := g.Villains[m.Enemy]; v != nil {
			if card, ok := g.drawEncounter(); ok {
				card.FaceDown = true
				v.BoostCards = append(v.BoostCards, card)
			}
		}

	case RevealBoost:
		if v := g.Villains[m.Enemy]; v != nil {
			var stillFacedown CardList
			for _, c := range v.BoostCards {
				if c.FaceDown {
					c.FaceDown = false
					def := c.Def()
					if def.BoostEntersPlay {
						// "Boost: put this card into play" — spawn it
						// instead of contributing boost icons.
						g.tlogMajorf("log.boostEnters", def)
						g.Push(RevealEncounterCard{Player: g.boostSpawnTarget(v), Card: c})
						continue
					}
					add := deref(def.Boost, 0)
					v.BoostCount += add
					g.tlogf("log.boostRevealed", def, add)
					v.RevealedBoosts = append(v.RevealedBoosts, c)
					// Foiled! interrupt window (09038): when the boost card
					// is turned faceup during a scheme activation, a player
					// holding the event may cancel its boost icons. The
					// pending ApplyVillainScheme in the queue marks this as a
					// scheme activation (attack activations queue AskAttack
					// instead).
					scheme := g.schemeActivationPending(v.ID)
					if add > 0 && scheme {
						if p, card, ok := g.handCardHolding("09038"); ok {
							g.Push(AskQuestion{Player: p.ID, Question: Ask(
								Tf("q.foiledPrompt", p.Name, add),
								Choice{ID: "foiled-play", Label: Tf("m.playFoiled", add), Kind: ChoicePlay, CardCode: card.Code}.
									Msgs(ConsumeHandCard{Player: p.ID, CardID: card.ID},
										CancelBoostIcons{Enemy: v.ID, N: add}),
								Choice{ID: "foiled-pass", Label: Tf("m.pass"), Kind: ChoicePass},
							)})
						}
					}
					// Preemptive Strike (38015): during villain attacks, cancel
					// all boost icons on this card and deal the villain 1
					// damage per icon.
					if add > 0 && !scheme && g.attackActivationPending(v.ID) {
						if p, card, ok := g.handCardHolding("38015"); ok {
							g.Push(AskQuestion{Player: p.ID, Question: Ask(
								Tf("q.preemptivePrompt", p.Name, add, add),
								Choice{ID: "preemptive-play", Label: Tf("m.playPreemptive", add), Kind: ChoicePlay, CardCode: card.Code}.
									Msgs(ConsumeHandCard{Player: p.ID, CardID: card.ID},
										CancelBoostIcons{Enemy: v.ID, N: add},
										DamageEntity{Target: v.ID, Damage: add, Source: p.ID}),
								Choice{ID: "preemptive-pass", Label: Tf("m.pass"), Kind: ChoicePass},
							)})
						}
					}
					if b := behavior(def.Code); b.Boost != nil {
						g.Push(b.Boost(g, c)...)
					}
				} else {
					stillFacedown = append(stillFacedown, c)
				}
			}
			v.BoostCards = stillFacedown
		}

	case ClearBoosts:
		if v := g.Villains[m.Enemy]; v != nil {
			g.EncounterDiscard = append(g.EncounterDiscard, v.RevealedBoosts...)
			v.RevealedBoosts = nil
			v.BoostCount = 0
		}

	case RevealEncounterCard:
		g.revealEncounterCard(m.Player, m.Card)

	case ApplyVillainScheme:
		v := g.Villains[m.VillainID]
		p := g.Player(m.Player)
		if v == nil || p == nil || g.MainScheme == nil {
			return
		}
		threat := g.schemeValueOf(v.ID)
		// Emergency (01085): the resolving player may reduce the scheme's
		// threat by 1 by discarding it from hand.
		var emerg Card
		hasEmerg := false
		for _, c := range p.Hand {
			if c.Code == "01085" {
				emerg, hasEmerg = c, true
				break
			}
		}
		if hasEmerg {
			g.Push(AskQuestion{Player: p.ID, Question: Ask(
				Tf("q.emergencyPrompt"),
				Choice{ID: "play-emergency", Label: Tf("m.playEmergency"), Kind: ChoicePlay, CardCode: "01085"}.
					Msgs(DiscardCards{Player: p.ID, Cards: CardList{emerg}},
						SchemeThreat{Scheme: g.MainScheme.ID, N: threat - 1, Source: v.ID},
						ClearBoosts{Enemy: v.ID}),
				Choice{ID: "skip-emergency", Label: Tf("m.continue"), Kind: ChoicePass}.
					Msgs(SchemeThreat{Scheme: g.MainScheme.ID, N: threat, Source: v.ID},
						ClearBoosts{Enemy: v.ID}),
			)})
			return
		}
		g.Push(SchemeThreat{Scheme: g.MainScheme.ID, N: threat, Source: v.ID})
		g.Push(ClearBoosts{Enemy: v.ID})

	case VillainDefeated:
		v := g.Villains[m.VillainID]
		if v == nil {
			return
		}
		g.tlogMajorf("log.defeatedBang", v)
		g.Push(AdvanceVillainStage{VillainID: v.ID})

	case AdvanceVillainStage:
		g.advanceVillainStage(m.VillainID)

	case MinionDefeated:
		if mn := g.Minions[m.MinionID]; mn != nil {
			g.tlogMajorf("log.defeated", mn)
			// Its attachments drop off and are discarded alongside it.
			for _, a := range mn.Attachments {
				if at := g.Attachments[a]; at != nil {
					g.EncounterDiscard = append(g.EncounterDiscard, Card{ID: g.nextCardID(), Code: at.Code})
				}
				g.Delete(a)
			}
			// A facedown Drone (Ultron Drones 01140) is a player card: it
			// returns to its owner's discard pile, not the encounter pile.
			if mn.IsDrone && mn.Source != nil {
				src := *mn.Source
				owner := src.Owner
				if owner == "" {
					owner = mn.EngagedWith
					src.Owner = owner
				}
				if p := g.Player(owner); p != nil {
					p.Discard = append(p.Discard, src)
					g.Delete(m.MinionID)
					return
				}
			}
			if strings.Contains(mn.EDef().Text, "Victory") {
				g.VictoryDisplay = append(g.VictoryDisplay, Card{ID: g.nextCardID(), Code: mn.Code})
			} else {
				g.EncounterDiscard = append(g.EncounterDiscard, Card{ID: g.nextCardID(), Code: mn.Code})
			}
			g.Delete(m.MinionID)
		}

	case MainSchemeMaxed:
		if g.MainScheme != nil && m.Scheme == g.MainScheme.ID {
			s := g.MainScheme
			if scen := g.Scenario(); scen.OnMainSchemeMaxed != nil {
				g.Push(scen.OnMainSchemeMaxed(g, s)...)
			} else if s.Stage < len(s.StageCodes) {
				// Multi-stage schemes (Klaw): completing an intermediate
				// stage advances to the next one; only the final stage loses
				// the game (01117 prints "If this stage is completed, the
				// players lose the game." — intermediate stages don't).
				g.tlogMajorf("log.mainSchemeStageDone", s)
				g.Push(ReplaceMainScheme{Scheme: s.ID})
			} else {
				g.Push(GameOver{Won: false, Reason: Tf("reason.mainSchemeCompleted")})
			}
		}

	case SchemeDefeated:
		g.handleSchemeDefeated(m.Scheme)

	case ReplaceMainScheme:
		if g.MainScheme != nil && m.Scheme == g.MainScheme.ID {
			old := g.MainScheme
			stages := old.StageCodes
			next := old.Stage + 1
			if next-1 < len(stages) {
				s := g.spawnMainScheme(stages, next)
				g.Push(FlipMainScheme{Scheme: s.ID})
			}
		}

	case FlipMainScheme:
		if g.MainScheme != nil && m.Scheme == g.MainScheme.ID {
			s := g.MainScheme
			s.Code = s.StageCodes[s.Stage-1]
			// The scheme is the first arg (now the b-face code): hovering
			// the line shows the flipped-to face, and clients watch for
			// this entry to pop the a-face story that was just turned away.
			g.tlogMajorf("log.mainSchemeFlips", s, s.EDef().StageLabel, s.Threat, s.MaxThreat)
			// b-face "When Revealed" effects resolve at the flip. Exact-key
			// lookup, and only for true b-face codes: single-code stages
			// ("04061") already fired their base registration at the spawn
			// via behavior()'s base-code fallback, so dispatching them
			// again here would double-trigger. Distinct a-face (setup) and
			// b-face (reveal) registrations both firing is correct.
			if strings.HasSuffix(s.Code, "b") {
				if b, ok := behaviorRegistry[s.Code]; ok && b.MainSchemeRevealed != nil {
					g.Push(b.MainSchemeRevealed(g, s)...)
				}
			}
		}

	case GameOver:
		g.Over = true
		g.Won = m.Won
		g.Reason = m.Reason
		g.pending = nil
		g.queue = nil
		if m.Won {
			g.tlogMajorf("log.victory", m.Reason.Text)
		} else {
			g.tlogMajorf("log.defeat", m.Reason.Text)
		}

	case AskQuestion:
		g.pending = &PendingQuestion{Player: m.Player, Question: m.Question}

	case WindowAfterEnemyAttacked:
	case WindowAfterThwarted:

	case RunAbility:
		key := AbilityKey(m.Source, m.Index)
		g.UsedThisRound[key] = true
		g.UsedThisTurn[key] = true

	case FlipVillainPersona:
		if v := g.Villains[m.VillainID]; v != nil {
			side := "b"
			if m.FlipToNorman {
				side = "a"
			}
			v.Code = v.Code[:5] + side
			def := v.EDef()
			v.SchemeVal = deref(def.Scheme, 0)
			v.AttackVal = deref(def.Attack, 0)
			g.tlogMajorf("log.villainFlips", def)
		}

	case MillPlayerDeck:
		if p := g.Player(m.Player); p != nil {
			var discarded []Card
			for i := 0; i < m.N && len(p.Deck) > 0; i++ {
				c := p.Deck[0]
				p.Deck = p.Deck[1:]
				discarded = append(discarded, c)
			}
			if len(discarded) > 0 {
				g.Push(DiscardCards{Player: p.ID, Cards: discarded})
				// Luck riders react to cards leaving the deck top
				// (Jackpot!, White Fox, The Painted Lady...).
				for _, c := range discarded {
					g.Push(DeckTopDiscarded{Player: p.ID, Card: c})
				}
			}
		}

	case DealEncounterToPlayer:
		if p := g.Player(m.Player); p != nil {
			if card, ok := g.drawEncounter(); ok {
				p.EncounterDown = append(p.EncounterDown, card)
			}
		}

	case EngageMinion:
		if mn := g.Minions[m.MinionID]; mn != nil {
			mn.EngagedWith = m.Player
			if p := g.Player(m.Player); p != nil {
				g.tlogf("log.engages", mn, p.Name)
			}
		}

	case AddInfamyMsg:
		if env := g.EnvironmentByCode(m.Env); env != nil {
			env.Counters += m.N
			g.Logf("%s gains %d infamy counter(s) (%d total)", env.EDef().Name, m.N, env.Counters)
			return
		}
		if m.OrMadness > 0 {
			if env := g.EnvironmentByCode("02006b"); env != nil {
				env.Counters -= m.OrMadness
				if env.Counters < 0 {
					env.Counters = 0
				}
				g.Logf("State of Madness loses %d madness counter(s) (%d left)", m.OrMadness, env.Counters)
			}
		}

	case DiscardEncounterCard:
		for i, c := range g.EncounterDeck {
			if c.ID == m.Card.ID {
				g.EncounterDeck = append(g.EncounterDeck[:i], g.EncounterDeck[i+1:]...)
				g.EncounterDiscard = append(g.EncounterDiscard, c)
				g.Logf("%s discards %s from the encounter deck", "Heimdall", c.Def().Name)
				return
			}
		}

	case BoostEnemyAttack:
		switch e := g.Entity(m.Enemy).(type) {
		case *Villain:
			e.AttackVal += m.N
		case *Minion:
			e.AttackVal += m.N
		}

	case BoostActivation:
		switch e := g.Entity(m.Enemy).(type) {
		case *Villain:
			e.BoostCount += m.N
		case *Minion:
			// Minions share the boost-count semantics loosely; store on
			// the attack for the defense prompt.
			e.AttackVal += m.N
			// The bump is transient only for villains; minion callers
			// avoid this message today.
		}

	case RevealNemesisSet:
		g.handleRevealNemesisSet(m.Player)

	case SpawnDrone:
		g.handleSpawnDrone(m.Player)

	case ChooseDiscardFromHand:
		g.handleChooseDiscardFromHand(m)

	case ObligationResolve:
		if p := g.Player(m.Player); p != nil {
			if m.Remove {
				p.ObligationRemoved = append(p.ObligationRemoved, m.Card)
				g.tlogf("log.removedFromGame", m.Card)
			} else {
				p.ObligationDiscard = append(p.ObligationDiscard, m.Card)
				g.tlogf("log.discarded", m.Card)
			}
		}

	case DiscardControlled:
		g.discardControlled(m.Player, m.ID)

	case AddAccelerationToken:
		if g.MainScheme != nil && m.Scheme == g.MainScheme.ID {
			g.MainScheme.AccelerationTokens++
			g.tlogf("log.gainsAccel", g.MainScheme)
		}

	case RevealNextEncounter:
		if c, ok := g.drawEncounter(); ok {
			g.revealEncounterCard(m.Player, c)
		}

	case PlayDefenseEvent:
		g.handlePlayDefenseEvent(m)

	case AddEntityCounter:
		switch t := g.Entity(m.ID).(type) {
		case *Player:
			t.Counters += m.N
			g.tlogMinorf("log.counters", t.Name, t.Counters)
		case *Support:
			t.Counters += m.N
			g.tlogMinorf("log.counters", t, t.Counters)
		case *Upgrade:
			t.Counters += m.N
			g.tlogMinorf("log.counters", t, t.Counters)
		case *Ally:
			t.Counters += m.N
			g.tlogMinorf("log.counters", t, t.Counters)
		case *Villain:
			t.Counters += m.N
			g.tlogMinorf("log.counters", t, t.Counters)
		case *Minion:
			t.Counters += m.N
			g.tlogMinorf("log.counters", t, t.Counters)
		}

	case ReturnControlled:
		if p := g.Player(m.Player); p != nil {
			if e := g.Entity(m.ID); e != nil {
				code := e.ECode()
				g.Delete(m.ID)
				p.Hand = append(p.Hand, Card{ID: g.nextCardID(), Code: code, Owner: p.ID})
				g.tlogf("log.returnsToHand", p.Name, e)
			}
		}

	case AllyEntersPlayFree:
		g.handleAllyEntersPlayFree(m)

	case AttachUpgrade:
		if u := g.Upgrades[m.ID]; u != nil {
			u.AttachTo = m.Target
			switch t := g.Entity(m.Target).(type) {
			case *Player:
				if m.MaxHP > 0 {
					t.MaxHP += m.MaxHP
				}
				if m.GrantTrait != "" {
					t.ExtraTraits = append(t.ExtraTraits, m.GrantTrait)
				}
			case *Ally:
				if m.MaxHP > 0 {
					t.MaxHP += m.MaxHP
				}
				if m.ATK > 0 {
					t.PermATK += m.ATK
				}
				if m.THW > 0 {
					t.PermTHW += m.THW
				}
				if m.GrantTrait != "" {
					t.ExtraTraits = append(t.ExtraTraits, m.GrantTrait)
				}
			case *Minion:
				// Enemy attachments (Spider-Tracer...).
				t.Attachments = append(t.Attachments, m.ID)
			case *Villain:
				// Upgrades attached to the villain (Rogue's Touched, X-23's
				// Puncture Wound).
				t.Attachments = append(t.Attachments, m.ID)
			}
			if tgt := g.Entity(m.Target); tgt != nil {
				g.tlogf("log.attachesTo", u, tgt)
			}
		}

	case AddProgressCounters:
		if p := g.Player(m.Player); p != nil {
			p.GrowthCounters += m.N
			if m.N > 0 {
				g.Logf("%s gains %d progress counter(s) (%d total)", p.Name, m.N, p.GrowthCounters)
			}
		}

	case SwapHeroSide:
		if p := g.Player(m.Player); p != nil {
			p.HeroCode = m.HeroCode
			p.ExtraTraits = p.ExtraTraits[:0] // re-derive dynamic traits
			g.LogMajorf("%s swaps to %s", p.Name, p.HeroDef().Name)
		}

	case HoodFoulPlay:
		// Foul Play: discard N encounter cards, dealing non-Hood cards to
		// the player facedown.
		if p := g.Player(m.Player); p != nil {
			for i := 0; i < m.N; i++ {
				c, ok := g.DrawEncounter()
				if !ok {
					return
				}
				if c.Def().CardSet == "the_hood" {
					g.EncounterDiscard = append(g.EncounterDiscard, c)
				} else {
					p.EncounterDown = append(p.EncounterDown, c)
					g.Logf("Foul Play deals %s to %s facedown", c.Def().Name, p.Name)
				}
			}
		}

	case ReplaySideSchemeReveal:
		if s := g.SideSchemes[m.Scheme]; s != nil {
			if b := behavior(s.Code); b.OnPlay != nil {
				g.Push(b.OnPlay(g, s)...)
			}
		}

	case AllForOneDamage:
		n := 3
		if p := g.Player(m.Player); p != nil {
			if !p.Exhausted && g.EntityHasTrait(p.ID, "avenger") {
				n++
			}
			for _, id := range p.Allies {
				if a := g.Allies[id]; a != nil && a.Exhausted && a.EDef().HasTrait("avenger") {
					n++
				}
			}
		}
		g.Push(DamageEntity{Target: m.Target, Damage: n, Source: m.Player})

	case SpawnSymbiote:
		def, ok := DB.Lookup("20025")
		if !ok {
			return
		}
		fp := g.Players[0]
		for _, p := range g.Players {
			if p.FirstPlayer {
				fp = p
				break
			}
		}
		mn := &Minion{
			ID:          g.nextEntityID(KindMinion),
			Code:        def.Code,
			MaxHP:       deref(def.HP, 4),
			AttackVal:   deref(def.Attack, 2),
			SchemeVal:   deref(def.Scheme, 1),
			Tough:       def.HasKeyword("Toughness"),
			Guard:       def.HasKeyword("Guard"),
			EngagedWith: fp.ID,
		}
		g.Minions[mn.ID] = mn
		g.tlogMajorf("log.enragedSymbiote", fp.Name)

	case SetMassForm:
		if p := g.Player(m.Player); p != nil {
			var kept []string
			for _, t := range p.ExtraTraits {
				if t != "dense" && t != "intangible" {
					kept = append(kept, t)
				}
			}
			form := m.Form
			if form == "" {
				// Flip.
				for _, t := range p.ExtraTraits {
					if t == "dense" {
						form = "intangible"
					} else if t == "intangible" {
						form = "dense"
					}
				}
			}
			if form != "" {
				kept = append(kept, form)
			}
			p.ExtraTraits = kept
			if form != "" {
				g.Logf("%s takes %s mass form", p.Name, form)
			}
		}

	case SetAntForm:
		if p := g.Player(m.Player); p != nil {
			var kept []string
			for _, t := range p.ExtraTraits {
				if t != "giant" && t != "tiny" {
					kept = append(kept, t)
				}
			}
			if m.Form != "" {
				kept = append(kept, m.Form)
			}
			p.ExtraTraits = kept
		}

	case ChangeFormAgain:
		if p := g.Player(m.Player); p != nil {
			if p.IsHero() {
				p.Side = SideAlterEgo
			} else {
				p.Side = SideHero
			}
			p.FormChanged = true
			g.Logf("%s changes form", p.Name)
		}

	case ResolveTechnique:
		if p := g.Player(m.Player); p != nil {
			// The technique just entered play; find it by code.
			for _, id := range p.Upgrades {
				if u := g.Upgrades[id]; u != nil && u.Code[:5] == m.Code[:5] {
					if b := behavior(u.Code); b.React != nil {
						g.Push(b.React(g, u, PlayerTurnStart{Player: p.ID})...)
					}
					break
				}
			}
		}

	case MillEncounter:
		for i := 0; i < m.N; i++ {
			c, ok := g.DrawEncounter()
			if !ok {
				return
			}
			g.EncounterDiscard = append(g.EncounterDiscard, c)
			g.Logf("Chaos Magic mills %s", c.Def().Name)
		}

	case TopDeckPick:
		if p := g.Player(m.Player); p != nil {
			if c, ok := p.Deck.Remove(m.CardID); ok {
				p.Hand = append(p.Hand, c)
				// The remaining top 2 go to the bottom.
				var rest CardList
				for i := 0; i < 2 && len(p.Deck) > 0; i++ {
					rest = append(rest, p.Deck[0])
					p.Deck = p.Deck[1:]
				}
				p.Deck = append(p.Deck, rest...)
			}
		}

	case SlippingSanityMill:
		stars := 0
		for i := 0; i < 5; i++ {
			c, ok := g.DrawEncounter()
			if !ok {
				break
			}
			if b := c.Def().Boost; b != nil && *b > 0 {
				stars++
			}
			g.EncounterDiscard = append(g.EncounterDiscard, c)
		}
		if stars > 0 && g.MainScheme != nil {
			g.Push(SchemeThreat{Scheme: g.MainScheme.ID, N: stars, Source: EntityID("15023")})
		}

	case BunkerDiscard:
		if p := g.Player(m.Player); p != nil && len(p.Hand) > 0 {
			var picks []Choice
			for _, c := range p.Hand {
				picks = append(picks, Choice{Label: Tf("m.discardCard", c), Kind: ChoiceCard, CardCode: c.Code}.
					Msgs(DiscardCards{Player: p.ID, Cards: CardList{c}}))
			}
			q := AskN(Tf("q.discardTwoCards"), min(2, len(p.Hand)), picks...)
			g.Push(AskQuestion{Player: p.ID, Question: q})
		}

	case DiscardAttachmentMsg:
		if a := g.Attachments[m.ID]; a != nil {
			g.Delete(m.ID)
			g.EncounterDiscard = append(g.EncounterDiscard, Card{ID: g.nextCardID(), Code: a.Code})
			g.Logf("%s is discarded", a.EDef().Name)
		}

	case AddVengeance:
		if p := g.Player(m.Player); p != nil && p.GrowthCounters < 3 {
			p.GrowthCounters++
			p.BonusATK++
			g.Logf("%s gains a vengeance counter (+%d ATK)", p.Name, p.GrowthCounters)
		}

	case RapidReturn:
		if p := g.Player(m.Player); p != nil {
			for i, c := range p.Discard {
				if c.Code == m.Code && c.Def().Type == "ally" {
					p.Discard = append(p.Discard[:i], p.Discard[i+1:]...)
					a := &Ally{
						ID:        g.nextEntityID(KindAlly),
						Code:      c.Code,
						Owner:     p.ID,
						MaxHP:     deref(c.Def().HP, 1),
						AttackVal: deref(c.Def().Attack, 0),
						ThwartVal: deref(c.Def().Thwart, 0),
						Damage:    1,
					}
					g.Allies[a.ID] = a
					p.Allies = append(p.Allies, a.ID)
					g.tlogf("log.returnsRapidResponse", a)
					return
				}
			}
		}

	case TempHandSizeMsg:
		if p := g.Player(m.Player); p != nil {
			p.TempHandSize += m.N
			g.Logf("%s gets +%d hand size until the end of the phase", p.Name, m.N)
		}

	case IndirectDamage:
		if p := g.Player(m.Player); p != nil && m.N > 0 {
			chars := []EntityID{p.ID}
			for _, id := range p.Allies {
				if g.Allies[id] != nil {
					chars = append(chars, id)
				}
			}
			if len(chars) == 1 {
				g.Push(DamageEntity{Target: p.ID, Damage: m.N})
				return
			}
			g.Push(AskQuestion{Player: p.ID, Question: g.indirectQuestion(p, m.N, chars)})
		}

	case BarrageCharge:
		var ship *Environment
		for _, env := range g.Environments {
			if env != nil && data.BaseCode(env.Code) == "16063" {
				ship = env
				break
			}
		}
		if ship == nil {
			return
		}
		ship.Counters++
		g.Logf("Badoon Ship charges up (%d barrage counters)", ship.Counters)
		if ship.Counters >= 4 {
			ship.Counters = 0
			var msgs []Message
			for _, p := range g.Players {
				msgs = append(msgs, IndirectDamage{Player: p.ID, N: 2})
			}
			g.Push(msgs...)
		}

	case CollectCard:
		// If the card still sits in a player's deck (deck-top feeds),
		// remove it there.
		for _, p := range g.Players {
			p.Deck.Remove(m.Card.ID)
		}
		g.Collection = append(g.Collection, m.Card)
		g.Logf("%s is placed into The Collection", m.Card.Def().Name)

	case SummonSix:
		for _, base := range m.Cards {
			var kept CardList
			found := false
			for _, c := range g.SetAside {
				if BaseCodeOf(c.Code) == base && !found {
					found = true
					continue
				}
				kept = append(kept, c)
			}
			if !found {
				continue
			}
			g.SetAside = kept
			v := g.spawnVillain(VillainStageCodes(base), 1)
			if v != nil {
				g.ActiveVillain = v.ID
				g.tlogMajorf("log.ambush", v)
			}
		}

	case CostDiscountApply:
		if p := g.Player(m.Player); p != nil && m.Amount > 0 {
			p.CostDiscounts = append(p.CostDiscounts, CostDiscount{Amount: m.Amount})
			g.tlogf("log.nextCardCostsLess", p.Name, m.Amount)
		}

	case SenseEnterPlay:
		g.handleSenseEnterPlay(m)

	case ShuffleIntoDeck:
		if p := g.Player(m.Player); p != nil {
			if c, ok := p.Discard.Remove(m.CardID); ok {
				p.Deck = append(p.Deck, c)
				g.shuffle(&p.Deck)
				g.tlogf("log.shufflesIntoDeck", p.Name, c)
			}
		}

	case GrantTrait:
		switch t := g.Entity(m.Target).(type) {
		case *Player:
			t.ExtraTraits = append(t.ExtraTraits, m.Trait)
			g.tlogf("log.gainsTrait", t.Name, m.Trait)
		case *Ally:
			t.ExtraTraits = append(t.ExtraTraits, m.Trait)
			g.tlogf("log.gainsTrait", t, m.Trait)
		}

	case InvokeSpecial:
		if p := g.Player(m.Player); p != nil {
			card, ok := p.SenseDeck.Remove(m.Card.ID)
			if !ok {
				return
			}
			def := card.Def()
			if m.ReturnToTop {
				p.SenseDeck = append(CardList{card}, p.SenseDeck...)
				g.tlogf("log.resolvesTop", p.Name, def)
			} else {
				p.SideDiscard = append(p.SideDiscard, card)
				g.tlogf("log.resolves", p.Name, def)
			}
			if b := behavior(def.Code); b.OnPlay != nil {
				ec := &EventCard{Code: def.Code, Owner: p.ID}
				g.Push(b.OnPlay(g, ec)...)
			}
		}

	case SideDeckDiscardTop:
		if p := g.Player(m.Player); p != nil && len(p.SenseDeck) > 0 {
			card := p.SenseDeck[0]
			p.SenseDeck = p.SenseDeck[1:]
			p.SideDiscard = append(p.SideDiscard, card)
			g.tlogf("log.discardsSideDeck", p.Name, card)
		}

	case UpgradeEnterPlay:
		if p := g.Player(m.Player); p != nil {
			card, ok := p.Hand.Find(m.Card.ID)
			if !ok {
				card, ok = p.Deck.Find(m.Card.ID)
			}
			if !ok {
				card, ok = p.Discard.Find(m.Card.ID)
			}
			if !ok {
				g.tlogf("log.cannotFindPut", m.Card.Code)
				return
			}
			p.Hand.Remove(card.ID)
			p.Deck.Remove(card.ID)
			p.Discard.Remove(card.ID)
			def := card.Def()
			u := &Upgrade{ID: g.nextEntityID(KindUpgrade), Code: def.Code, Owner: p.ID}
			g.Upgrades[u.ID] = u
			p.Upgrades = append(p.Upgrades, u.ID)
			g.tlogf("log.putsIntoPlay", p.Name, def)
			if b := behavior(def.Code); b.OnPlay != nil {
				g.Push(b.OnPlay(g, u)...)
			}
		}

	case SideDeckToHand:
		if p := g.Player(m.Player); p != nil {
			for i, c := range p.SenseDeck {
				if c.ID == m.CardID {
					p.SenseDeck = append(p.SenseDeck[:i], p.SenseDeck[i+1:]...)
					p.Hand = append(p.Hand, c)
					g.tlogf("log.takesUnderEcho", p.Name, c)
					return
				}
			}
		}

	case RecycleFromDiscard:
		if p := g.Player(m.Player); p != nil {
			if src := g.Player(m.From); src != nil {
				if c, ok := src.Discard.Remove(m.CardID); ok {
					p.Hand = append(p.Hand, c)
					g.tlogf("log.takesFromDiscard", p.Name, c, src.Name)
				}
			}
		}

	case SwapHandWithDeckTop:
		if p := g.Player(m.Player); p != nil && len(p.Deck) > 0 {
			if c, ok := p.Hand.Remove(m.CardID); ok {
				top := p.Deck[0]
				p.Deck[0] = c
				p.Hand = append(p.Hand, top)
				g.tlogf("log.swapsTop", p.Name, c)
			}
		}

	case EventPlayed:
		// announcement only; reactions drive the effects

	case SetEventBonus:
		if g.EventDamageBonus == nil {
			g.EventDamageBonus = map[PlayerID]int{}
		}
		if g.EventThreatBonus == nil {
			g.EventThreatBonus = map[PlayerID]int{}
		}
		if m.Damage != 0 {
			g.EventDamageBonus[m.Player] += m.Damage
		}
		if m.Threat != 0 {
			g.EventThreatBonus[m.Player] += m.Threat
		}

	case ReturnDiscardCard:
		if p := g.Player(m.Player); p != nil {
			if c, ok := p.Discard.Remove(m.CardID); ok {
				p.Hand = append(p.Hand, c)
				g.tlogf("log.takesBack", p.Name, c)
			}
		}

	case DiscardToBottom:
		if p := g.Player(m.Player); p != nil {
			if c, ok := p.Discard.Remove(m.CardID); ok {
				p.Deck = append(p.Deck, c)
				g.tlogf("log.putsBottom", p.Name, c)
			}
		}

	case AllyDefeated:
		if a := g.Allies[m.AllyID]; a != nil {
			destroy := func() { g.Push(AllyDestroyed{AllyID: a.ID}) }
			if hook := behavior(a.Code).AllyDefeatInterrupt; hook != nil {
				if msgs := hook(g, a, destroy); msgs != nil {
					g.Push(msgs...)
					return
				}
			}
			// A save effect (e.g. Valkyrie) may have healed the ally back
			// above 0 between the defeat announcement and its resolution.
			if a.HP() > 0 {
				return
			}
			destroy()
		}

	case AllyDestroyed:
		g.destroyAlly(m.AllyID)

	case SupportStoreCard:
		if s := g.Supports[m.ID]; s != nil {
			if p := g.Player(s.Owner); p != nil {
				if c, ok := p.Hand.Remove(m.Card.ID); ok {
					s.AttachedCards = append(s.AttachedCards, c)
					s.Counters = len(s.AttachedCards)
					g.tlogMinorf("log.tucksUnder", p.Name, s, s.Counters)
				}
			}
		}

	case SupportRetrieveCards:
		if s := g.Supports[m.ID]; s != nil {
			if p := g.Player(s.Owner); p != nil {
				take := min(len(m.Cards), 3, len(s.AttachedCards))
				taken := append(CardList{}, s.AttachedCards[:take]...)
				s.AttachedCards = s.AttachedCards[take:]
				s.Counters = len(s.AttachedCards)
				p.Hand = append(p.Hand, taken...)
				g.tlogMinorf("log.takesStored", p.Name, take, s)
			}
		}

	case TreacheryWindow:
		g.handleTreacheryWindow(m)

	case TreacheryResolve:
		g.handleTreacheryResolve(m)

	case ConsumeHandCard:
		if p := g.Player(m.Player); p != nil {
			if c, ok := p.Hand.Remove(m.CardID); ok {
				p.Discard = append(p.Discard, c)
			}
		}

	case PlayDiscardAlly:
		g.handlePlayDiscardAlly(m)

	case ApplyStatBonus:
		if p := g.Player(m.Target); p != nil {
			p.BonusATK += m.ATK
			p.BonusTHW += m.THW
			p.BonusDEF += m.DEF
			g.tlogf("log.getsStats3", p.Name, m.THW, m.ATK, m.DEF)
		}

	case AllyStatBonus:
		if a := g.Allies[m.Ally]; a != nil {
			a.BonusTHW += m.THW
			a.BonusATK += m.ATK
			g.tlogf("log.getsStats2", a, m.THW, m.ATK)
		}

	case TakeDeckCard:
		p := g.Player(m.Player)
		if p == nil {
			return
		}
		if card, ok := p.Deck.Remove(m.CardID); ok {
			p.Hand = append(p.Hand, card)
			g.tlogf("log.takesFromDeck", p.Name, card)
			if m.FromTop > 0 {
				// Futurist: discard the remaining looked cards.
				n := min(m.FromTop-1, len(p.Deck))
				var discarded []Card
				for i := 0; i < n; i++ {
					c := p.Deck[0]
					p.Deck = p.Deck[1:]
					discarded = append(discarded, c)
				}
				if len(discarded) > 0 {
					g.Push(DiscardCards{Player: p.ID, Cards: discarded})
				}
			}
		}
	}
}

func (g *Game) handleRevealNemesisSet(pid PlayerID) {
	p := g.Player(pid)
	if p == nil {
		return
	}
	g.tlogMajorf("log.revealsNemesis", p.Name)
	var rest CardList
	for _, c := range p.NemesisDeck {
		def := c.Def()
		switch def.Type {
		case "minion":
			mn := &Minion{
				ID:        g.nextEntityID(KindMinion),
				Code:      def.Code,
				MaxHP:     deref(def.HP, 1),
				AttackVal: deref(def.Attack, 0),
				SchemeVal: deref(def.Scheme, 0),
				Tough:     def.HasKeyword("Toughness"),
				Guard:     def.HasKeyword("Guard"),
			}
			g.Minions[mn.ID] = mn
			g.tlogMajorf("log.entersEngaged", def, p.Name)
			g.Push(MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
			if def.HasKeyword("Quickstrike") && p.IsHero() {
				g.Push(DamageEntity{Target: p.ID, Damage: mn.AttackVal, Source: mn.ID})
			}
			if b := behavior(def.Code); b.OnPlay != nil {
				g.Push(b.OnPlay(g, mn)...)
			}
		case "ally":
			// Encounter-side allies (Longshot) join the revealing player
			// and surge.
			pid := p.ID
			a := &Ally{
				ID:        g.nextEntityID(KindAlly),
				Code:      def.Code,
				Owner:     pid,
				MaxHP:     deref(def.HP, 1),
				AttackVal: deref(def.Attack, 0),
				ThwartVal: deref(def.Thwart, 0),
			}
			g.Allies[a.ID] = a
			p.Allies = append(p.Allies, a.ID)
			g.tlogMajorf("log.joins", def, p.Name)
			g.Push(AllyEnteredPlay{Ally: a.ID, Player: pid})
			g.Push(RevealNextEncounter{Player: pid})
		case "side_scheme":
			s := &SideScheme{
				ID:        g.nextEntityID(KindSideScheme),
				Code:      def.Code,
				Threat:    deref(def.BaseThreat, 1) + len(g.Players) - 1,
				MaxThreat: deref(def.BaseThreat, 1) + 2*(len(g.Players)-1),
				Crisis:    sideSchemeIsCrisis(def),
				Hazard:    def.Hazards,
			}
			g.SideSchemes[s.ID] = s
			g.tlogMajorf("log.sideSchemeEnters", def, s.Threat)
			if b := behavior(def.Code); b.OnPlay != nil {
				g.Push(b.OnPlay(g, s)...)
			}
		default:
			p.NemesisDiscard = append(p.NemesisDiscard, c)
		}
	}
	p.NemesisDeck = rest
}

// spawnDroneMinion puts a facedown player card into play as a Drone (1/1/1).
func (g *Game) spawnDroneMinion(card Card) *Minion {
	mn := &Minion{
		ID:        g.nextEntityID(KindMinion),
		Code:      "01143", // Advanced Ultron Drone as the visual proxy
		MaxHP:     g.droneBonus() + 1,
		AttackVal: g.droneBonus() + 1,
		SchemeVal: 1,
		IsDrone:   true,
		Source:    &card,
	}
	g.Minions[mn.ID] = mn
	return mn
}

// droneBonus returns +1 while Ultron stage III is in play or the Upgraded
// Drones attachment is attached to the Ultron Drones environment.
func (g *Game) droneBonus() int {
	for _, v := range g.Villains {
		if v.Code == "01136" {
			return 1
		}
	}
	for _, a := range g.Attachments {
		if a.Code == "01142" {
			return 1
		}
	}
	return 0
}

func (g *Game) handleSpawnDrone(pid PlayerID) {
	p := g.Player(pid)
	if p == nil || len(p.Deck) == 0 {
		return
	}
	card := p.Deck[0]
	p.Deck = p.Deck[1:]
	if card.Owner == "" {
		card.Owner = pid
	}
	mn := g.spawnDroneMinion(card)
	mn.EngagedWith = pid
	g.tlogMajorf("log.droneEnters", p.Name)
}

// handleChooseDiscardFromHand asks for N discards from the hand as it is
// RIGHT NOW — the question is built at processing time, so cards drawn by
// messages queued ahead of this one are selectable (Spiritual Meditation).
func (g *Game) handleChooseDiscardFromHand(m ChooseDiscardFromHand) {
	p := g.Player(m.Player)
	if p == nil || len(p.Hand) == 0 {
		return
	}
	n := m.N
	if n <= 0 {
		n = 1
	}
	if n > len(p.Hand) {
		n = len(p.Hand)
	}
	prompt := m.Prompt
	if prompt.Key == "" && prompt.Text == "" {
		prompt = Tf("c.discardWhichCard")
	}
	var picks []Choice
	for _, c := range p.Hand {
		picks = append(picks, Choice{
			Label: Tf("m.discardCard", c), Kind: ChoiceCard, CardCode: c.Code, SourceID: EntityID(c.ID),
		}.Msgs(DiscardCards{Player: p.ID, Cards: CardList{c}}))
	}
	q := AskN(prompt, 0, picks...)
	q.Validate = fmt.Sprintf("discardCost:%d", n)
	q.Context = map[string]any{"player": p.ID.String()}
	g.Push(AskQuestion{Player: p.ID, Question: q})
}

func (g *Game) discardControlled(pid PlayerID, id EntityID) {
	p := g.Player(pid)
	e := g.Entity(id)
	if p == nil || e == nil {
		return
	}
	g.Delete(id)
	var code string
	switch t := e.(type) {
	case *Ally:
		code = t.Code
	case *Support:
		code = t.Code
	case *Upgrade:
		code = t.Code
	default:
		return
	}
	// Sense upgrades cycle back to the bottom of the Sense deck (Matt
	// Murdock's forced interrupt).
	if def, ok := DB.Lookup(code); ok && def.HasTrait("sense") {
		p.SenseDeck = append(p.SenseDeck, Card{ID: g.nextCardID(), Code: code, Owner: p.ID})
		g.tlogf("log.returnsSenseBottom", def)
		return
	}
	g.cardLeavesPlay(p, code, e.EDef().Name)
	g.tlogf("log.discarded", e)
}

// thwartBlocked reports whether a minion engaged with the player blocks
// thwarting (e.g. Baron Zemo). Approximation: gates basic hero thwarts
// only, not event-driven threat removal.
func (g *Game) thwartBlocked(p *Player) bool {
	return g.thwartBlockerName(p) != ""
}

func (g *Game) thwartBlockerName(p *Player) string {
	for _, id := range sortedIDs(g.Minions) {
		mn := g.Minions[id]
		if mn.EngagedWith != p.ID {
			continue
		}
		if behavior(mn.Code).EngagedBlocksThwart {
			return mn.EDef().Name
		}
		// Brix (30032) grants every Inheritor minion patrol.
		if mn.EDef().HasTrait("inheritor") && g.minionInPlay("30032") {
			return mn.EDef().Name + " (patrol)"
		}
	}
	return ""
}

func (g *Game) handleStartGame() {
	// First player is chosen by the group (official setup step 3);
	// modelled as a random pick with the seeded RNG.
	first := g.Random(len(g.Players))
	for i := range g.Players {
		g.Players[i].FirstPlayer = i == first
	}
	g.tlogf("log.takesFirstPlayer", g.Players[first].Name)

	for _, p := range g.Players {
		p.Deck = g.assignCardIDs(p.Deck, p.ID)
		g.shuffle(&p.Deck)
	}
	// Obligations join the encounter deck (official rules); they carry
	// their owner and resolve for them when revealed.
	for _, p := range g.Players {
		g.EncounterDeck = append(g.EncounterDeck, p.ObligationDeck...)
		p.ObligationDeck = nil
	}
	g.EncounterDeck = g.assignCardIDs(g.EncounterDeck, "")
	g.shuffle(&g.EncounterDeck)
	// Flip stage 1 to its b face (resolving 1B When Revealed effects)
	// before opening hands: 1A setups read "… Shuffle the encounter deck.
	// Advance to stage 1B", so the b face must see the shuffled,
	// ID-assigned deck. Only schemes still on their a face flip (queue
	// replayed on resumed saves must not re-trigger the reveal).
	if s := g.MainScheme; s != nil && s.Code != s.StageCodes[s.Stage-1] {
		g.Push(FlipMainScheme{Scheme: s.ID})
	}
	scen := g.Scenario()
	if scen.Setup != nil {
		g.Push(scen.Setup(g)...)
	}
	g.tlogMajorf("log.scenario", scen.Name)
	// Opening hands (official setup step 14), then mulligans (step 15),
	// then player setup abilities (step 16), then the first round.
	for _, p := range g.Players {
		g.Push(DrawCards{Player: p.ID, N: p.HandSize(g)})
	}
	for _, p := range g.Players {
		g.Push(ResolveMulligan{Player: p.ID})
	}
	for _, p := range g.Players {
		if b := behavior(p.HeroCode); b.HeroSetup != nil {
			g.Push(b.HeroSetup(g, p)...)
		}
	}
	g.Push(BeginRound{})
}

// askMulligan offers the setup mulligan: keep the hand or discard any number
// of cards and redraw that many.
func (g *Game) askMulligan(pid PlayerID) {
	p := g.Player(pid)
	if p == nil || len(p.Hand) == 0 {
		return
	}
	pick := AskN(Tf("q.selectMulligan"), 0)
	for _, c := range p.Hand {
		def := c.Def()
		pick.Choices = append(pick.Choices, Choice{
			Label: S(def.Name), Kind: ChoiceCard, CardCode: def.Code,
		}.Msgs(MulliganCard{Player: p.ID, CardID: c.ID}))
	}
	q := Ask(Tf("q.mulligan"),
		Choice{ID: "keep", Label: Tf("m.keepHand"), Kind: ChoicePass},
		Choice{ID: "mulligan", Label: Tf("m.mulliganRedraw"), Kind: ChoiceCard}.WithThen(pick),
	)
	g.Push(AskQuestion{Player: p.ID, Question: q})
}

// askDiscardToHandSize runs the end-of-player-phase discard step: over-hand
// players must discard down; others may optionally discard any number.
func (g *Game) askDiscardToHandSize(pid PlayerID) {
	p := g.Player(pid)
	if p == nil || p.KOed || len(p.Hand) == 0 {
		return
	}
	over := len(p.Hand) - p.HandSize(g)
	if over <= 0 {
		pick := AskN(Tf("q.discardAnyNumber"), 0)
		for _, c := range p.Hand {
			def := c.Def()
			pick.Choices = append(pick.Choices, Choice{
				Label: S(def.Name), Kind: ChoiceCard, CardCode: def.Code,
			}.Msgs(DiscardCards{Player: p.ID, Cards: CardList{c}}))
		}
		q := Ask(Tf("q.discardBeforeDraw"),
			Choice{ID: "keep", Label: Tf("m.keepHand"), Kind: ChoicePass},
			Choice{ID: "discard", Label: Tf("m.discardCards"), Kind: ChoiceCard}.WithThen(pick),
		)
		g.Push(AskQuestion{Player: p.ID, Question: q})
		return
	}
	q := AskN(Tf("q.discardToHandSize", p.HandSize(g), over), over)
	for _, c := range p.Hand {
		def := c.Def()
		q.Choices = append(q.Choices, Choice{
			Label: S(def.Name), Kind: ChoiceCard, CardCode: def.Code,
		}.Msgs(DiscardCards{Player: p.ID, Cards: CardList{c}}))
	}
	q.N = over
	q.Validate = fmt.Sprintf("discardDown:%d", over)
	q.Context = map[string]any{"player": p.ID.String()}
	q.assignIDs("")
	g.Push(AskQuestion{Player: p.ID, Question: q})
}

// expirePhaseEffects clears until-end-of-phase modifiers and discounts.
func (g *Game) expirePhaseEffects() {
	for _, p := range g.Players {
		p.CostDiscounts = nil
		p.TempHandSize = 0
		// Until-end-of-phase stat modifiers expire.
		p.BonusTHW, p.BonusATK, p.BonusDEF = 0, 0, 0
		for _, id := range p.Allies {
			if a := g.Allies[id]; a != nil {
				a.BonusTHW, a.BonusATK = 0, 0
			}
		}
	}
	g.EventDamageBonus = map[PlayerID]int{}
	g.EventThreatBonus = map[PlayerID]int{}
	// Blank-text effects expire.
	for _, mn := range g.Minions {
		mn.BlankText = false
	}
}

// playerOrder returns the players in player order, starting with the first
// player.
func (g *Game) playerOrder() []*Player {
	out := make([]*Player, 0, len(g.Players))
	start := 0
	for i, p := range g.Players {
		if p.FirstPlayer {
			start = i
			break
		}
	}
	for i := range g.Players {
		out = append(out, g.Players[(start+i)%len(g.Players)])
	}
	return out
}

// nextActivePlayer returns the next clockwise player after from who has not
// been eliminated, or nil.
func (g *Game) nextActivePlayer(from PlayerID) *Player {
	idx := -1
	for i, q := range g.Players {
		if q.ID == from {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}
	for k := 1; k <= len(g.Players); k++ {
		q := g.Players[(idx+k)%len(g.Players)]
		if q.ID != from && !q.KOed {
			return q
		}
	}
	return nil
}

func (g *Game) handleBeginPhase(phase Phase) {
	switch phase {
	case PhaseResource:
		// Legacy saves only: the round used to open with a resource phase
		// (ready + draw + acceleration). New games never enter it.
		for _, p := range g.Players {
			g.Push(ReadyAll{Player: p.ID})
		}
		for _, p := range g.Players {
			g.Push(DrawCards{Player: p.ID, N: max(0, p.HandSize(g)-len(p.Hand))})
		}
		if g.MainScheme != nil && g.MainScheme.AccelerationTokens > 0 {
			g.Push(SchemeThreat{
				Scheme: g.MainScheme.ID,
				N:      g.MainScheme.AccelerationTokens,
				Source: NewEntityID(KindMainScheme, 0),
			})
		}
		g.Push(EndPhase{Phase: PhaseResource})

	case PhasePlayer:
		g.UsedThisTurn = map[string]bool{}
		order := g.playerOrder()
		for i, p := range order {
			if p.FirstPlayer {
				g.TurnIndex = i
				break
			}
		}
		// An eliminated first player cannot hold the token; fall back to
		// the first active player in order.
		if g.Players[g.TurnIndex].KOed {
			for i, p := range order {
				if !p.KOed {
					g.TurnIndex = i
					break
				}
			}
		}
		g.Push(PlayerTurnStart{Player: g.Players[g.TurnIndex].ID})

	case PhaseVillain:
		g.UsedThisTurn = map[string]bool{}
		g.ActiveTurn = ""
		order := g.playerOrder()
		// Step 1: place threat from the main scheme's acceleration field,
		// acceleration tokens, and acceleration icons in play.
		if g.MainScheme != nil {
			if n := g.accelerationThreat(); n > 0 {
				g.tlogf("log.accelPlaces", n, g.MainScheme)
				g.Push(SchemeThreat{
					Scheme: g.MainScheme.ID,
					N:      n,
					Source: NewEntityID(KindMainScheme, 0),
				})
			}
		}
		// Step 2: the villain activates once per player in player order;
		// after each activation, the minions engaged with that player
		// activate against them (order chosen by that player).
		for _, p := range order {
			if p.KOed {
				continue
			}
			for _, id := range sortedIDs(g.Villains) {
				g.Push(VillainActivates{VillainID: id, Player: p.ID})
			}
			g.Push(MinionActivations{Player: p.ID})
		}
		// Step 3: deal one encounter card per player, plus one additional
		// card per hazard icon in play (extras dealt in player order).
		var active []*Player
		for _, p := range order {
			if !p.KOed {
				active = append(active, p)
			}
		}
		deal := func(p *Player) {
			if card, ok := g.drawEncounter(); ok {
				p.EncounterDown = append(p.EncounterDown, card)
			}
		}
		for _, p := range active {
			deal(p)
		}
		for h := 0; h < g.hazardIconCount() && len(active) > 0; h++ {
			deal(active[h%len(active)])
		}
		// Step 4: reveal and resolve dealt cards, one at a time, in player
		// order (first player first).
		for _, p := range active {
			for len(p.EncounterDown) > 0 {
				card := p.EncounterDown[0]
				p.EncounterDown = p.EncounterDown[1:]
				g.Push(RevealEncounterCard{Player: p.ID, Card: card})
			}
		}
		g.Push(EndPhase{Phase: PhaseVillain})
	}
}

// accelerationThreat totals the threat added at villain phase step one: the
// main scheme's printed acceleration field, acceleration tokens, and
// acceleration icons on side schemes in play.
func (g *Game) accelerationThreat() int {
	if g.MainScheme == nil {
		return 0
	}
	n := g.MainScheme.AccelerationTokens
	if d := g.MainScheme.EDef(); d != nil && d.Acceleration > 0 {
		n += d.Acceleration
	}
	for _, id := range sortedIDs(g.SideSchemes) {
		if d := g.SideSchemes[id].EDef(); d != nil && d.Acceleration > 0 {
			n += d.Acceleration
		}
	}
	return n
}

// hazardIconCount totals hazard icons on cards in play (main scheme and side
// schemes); each icon deals one extra encounter card in the villain phase.
func (g *Game) hazardIconCount() int {
	n := 0
	if g.MainScheme != nil {
		n += g.MainScheme.Hazard
	}
	for _, s := range g.SideSchemes {
		n += s.Hazard
	}
	return n
}

func (g *Game) handleVillainActivates(m VillainActivates) {
	v := g.Villains[m.VillainID]
	p := g.Player(m.Player)
	if v == nil || p == nil || p.KOed {
		return
	}
	def := v.EDef()
	if b := behavior(v.Code); b.VillainActivate != nil {
		g.Push(b.VillainActivate(g, v, p)...)
		return
	}
	if p.IsHero() {
		// Attack
		if v.Stunned {
			v.Stunned = false
			g.tlogf("log.stunnedCanceled", def)
			return
		}
		g.tlogf("log.attacks", def, p.Name)
		g.Push(DealBoost{Enemy: v.ID})
		g.Push(RevealBoost{Enemy: v.ID})
		g.Push(AskAttack{Enemy: v.ID, Player: p.ID, Trigger: TriggerVillainAttacksYou})
	} else {
		// Scheme
		if v.Confused {
			v.Confused = false
			g.tlogf("log.confusedCanceled", def)
			return
		}
		// Tempus (45042): discard her to cancel the scheme activation.
		for _, id := range p.Allies {
			a := g.Allies[id]
			if a == nil || a.Exhausted || data.BaseCode(a.Code) != "45042" {
				continue
			}
			g.Push(AskQuestion{Player: p.ID, Question: Ask(
				Tf("q.tempusPrompt", def.Name),
				Choice{ID: "tempus-cancel", Label: Tf("m.discardTempus"), Kind: ChoiceAbility, SourceID: a.ID, CardCode: a.Code}.
					Msgs(AllyDestroyed{AllyID: a.ID}, DealEncounterToPlayer{Player: p.ID}),
				Choice{ID: "tempus-pass", Label: Tf("m.letSchemeResolve"), Kind: ChoicePass}.
					Msgs(DealBoost{Enemy: v.ID}, RevealBoost{Enemy: v.ID}, ApplyVillainScheme{VillainID: v.ID, Player: p.ID}),
			)})
			return
		}
		g.tlogf("log.schemesAgainst", def, p.Name)
		g.Push(DealBoost{Enemy: v.ID})
		g.Push(RevealBoost{Enemy: v.ID})
		g.Push(ApplyVillainScheme{VillainID: v.ID, Player: p.ID})
	}
}

// ApplyVillainScheme resolves a villain scheme with boosts already revealed.
type ApplyVillainScheme struct {
	VillainID EntityID
	Player    PlayerID
}

func (ApplyVillainScheme) msg() {}

func (g *Game) handleMinionActivates(m MinionActivates) {
	mn := g.Minions[m.MinionID]
	p := g.Player(m.Player)
	if mn == nil || p == nil || p.KOed {
		return
	}
	// Distraction (44054): the attached minion cannot activate.
	for _, aid := range mn.Attachments {
		if a := g.Attachments[aid]; a != nil && a.Code == "44054" {
			g.tlogf("log.cannotActivateDistraction", mn)
			return
		}
	}
	def := mn.EDef()
	if b := behavior(mn.Code); b.MinionActivate != nil {
		g.Push(b.MinionActivate(g, mn, p)...)
		return
	}
	if p.IsHero() {
		if mn.Stunned {
			mn.Stunned = false
			g.tlogf("log.stunnedCanceled", def)
			return
		}
		g.tlogf("log.attacks", def, p.Name)
		// Solus (30037) grants every Inheritor minion villainous.
		if def.HasKeyword("Villainous") || (def.HasTrait("inheritor") && g.minionInPlay("30037")) {
			g.dealMinionBoost(mn)
		}
		g.Push(AskAttack{Enemy: mn.ID, Player: p.ID})
	} else {
		if mn.Confused {
			mn.Confused = false
			return
		}
		g.tlogf("log.schemesAgainst", def, p.Name)
		if g.MainScheme != nil {
			// Informant (50050): the preparation may be discarded to
			// invert the scheme — it removes threat instead.
			for _, uid := range p.Upgrades {
				if u := g.Upgrades[uid]; u != nil && u.Code == "50050" {
					g.Push(AskQuestion{Player: p.ID, Question: Ask(
						Tf("q.informantPrompt", p.Name, def.Name),
						Choice{ID: "informant-use", Label: Tf("m.discardInformant"), Kind: ChoicePlay, CardCode: u.Code}.
							Msgs(DiscardControlled{Player: p.ID, ID: u.ID},
								ThwartScheme{Scheme: g.MainScheme.ID, N: g.schemeValueOf(mn.ID), Source: u.ID}),
						Choice{ID: "informant-pass", Label: Tf("m.pass"), Kind: ChoicePass}.
							Msgs(SchemeThreat{Scheme: g.MainScheme.ID, N: g.schemeValueOf(mn.ID), Source: mn.ID}),
					)})
					return
				}
			}
			g.Push(SchemeThreat{Scheme: g.MainScheme.ID, N: g.schemeValueOf(mn.ID), Source: mn.ID})
		}
	}
}

// dealMinionBoost resolves the Villainous keyword: the minion's
// activation reveals one boost card, adding its icons to the attack.
func (g *Game) dealMinionBoost(mn *Minion) {
	c, ok := g.drawEncounter()
	if !ok {
		return
	}
	def := c.Def()
	if def.BoostEntersPlay {
		g.tlogMajorf("log.boostEnters", def)
		g.Push(RevealEncounterCard{Player: g.boostSpawnTarget(nil), Card: c})
		return
	}
	add := deref(def.Boost, 0)
	mn.BoostCount += add
	g.EncounterDiscard = append(g.EncounterDiscard, c)
	g.tlogf("log.boostCard", mn, def, add)
	if b := behavior(def.Code); b.Boost != nil {
		g.Push(b.Boost(g, c)...)
	}
}

// engagedMinions returns the ids of the minions currently engaged with the
// player, in stable order.
func (g *Game) engagedMinions(pid PlayerID) []EntityID {
	var out []EntityID
	for _, id := range sortedIDs(g.Minions) {
		if mn := g.Minions[id]; mn != nil && mn.EngagedWith == pid {
			out = append(out, id)
		}
	}
	return out
}

// beginMinionActivations starts the engaged-minion activation step for a
// player: with two or more minions engaged, the player picks the order in
// which they activate (official villain phase step 2b).
func (g *Game) beginMinionActivations(pid PlayerID) {
	p := g.Player(pid)
	if p == nil || p.KOed {
		return
	}
	ids := g.engagedMinions(pid)
	if len(ids) == 0 {
		return
	}
	if len(ids) == 1 {
		g.Push(MinionActivates{MinionID: ids[0], Player: pid})
		return
	}
	g.Push(AskMinionOrder{Player: pid, Remaining: ids})
}

// handleAskMinionOrder asks which of the remaining engaged minions
// activates next; the chosen minion resolves fully before the rest are
// considered (its messages are queued ahead of the follow-up ask).
func (g *Game) handleAskMinionOrder(m AskMinionOrder) {
	p := g.Player(m.Player)
	if p == nil || p.KOed {
		return
	}
	var live []EntityID
	for _, id := range m.Remaining {
		if mn := g.Minions[id]; mn != nil && mn.EngagedWith == p.ID {
			live = append(live, id)
		}
	}
	if len(live) == 0 {
		return
	}
	if len(live) == 1 {
		g.Push(MinionActivates{MinionID: live[0], Player: p.ID})
		return
	}
	var choices []Choice
	for i, id := range live {
		mn := g.Minions[id]
		rest := append([]EntityID{}, live[:i]...)
		rest = append(rest, live[i+1:]...)
		choices = append(choices, Choice{
			Label: Tf("c.atkSch", mn, mn.AttackVal, mn.SchemeVal),
			Kind:  ChoiceTarget, SourceID: id, CardCode: mn.Code,
		}.Msgs(
			MinionActivates{MinionID: id, Player: p.ID},
			AskMinionOrder{Player: p.ID, Remaining: rest},
		))
	}
	g.Push(AskQuestion{Player: p.ID, Question: Ask(Tf("q.nextMinionActivate"), choices...)})
}

// handleAskAttack builds the attack/defense prompt at resolve time, so the
// displayed attack value includes the boost icons revealed for this
// activation. Villain attacks wrap the defense in the interrupt window;
// minion attacks (Trigger "") ask the defense question directly.
func (g *Game) handleAskAttack(m AskAttack) {
	p := g.Player(m.Player)
	if p == nil || p.KOed {
		return
	}
	// Morlock allies (40079): a minion attacking their controller is
	// redirected to a Morlock instead (approximation: the first ready
	// Morlock takes the full attack; villain attacks still run their
	// interrupt window and are not redirected).
	if m.Trigger == "" {
		for _, id := range p.Allies {
			a := g.Allies[id]
			if a == nil || !a.EDef().HasTrait("morlock") {
				continue
			}
			g.tlogf("log.redirectsAttack", p.Name, a)
			g.Push(DamageEntity{Target: a.ID, Damage: g.attackValue(m.Enemy), Source: m.Enemy})
			if v := g.Villains[m.Enemy]; v != nil {
				g.Push(ClearBoosts{Enemy: v.ID})
			}
			return
		}
	}
	atk := g.attackValue(m.Enemy)
	var q *Question
	if m.Trigger == "" {
		q = g.defenderQuestion(m.Enemy, atk, p)
	} else {
		q = g.attackQuestion(m.Enemy, atk, p, m.Trigger)
	}
	g.Push(AskQuestion{Player: p.ID, Question: q})
}

// handleOtherDefenders offers the defense to the remaining players after
// the attacked player declined; once everyone passes, the attack resolves
// undefended against the attacked player.
func (g *Game) handleOtherDefenders(m OtherDefenders) {
	if g.Player(m.For) == nil {
		return
	}
	var live []PlayerID
	for _, pid := range m.Remaining {
		if q := g.Player(pid); q != nil && !q.KOed {
			live = append(live, pid)
		}
	}
	if len(live) == 0 {
		g.Push(Defends{Defender: m.For, Against: m.Against, Undefended: true})
		return
	}
	next := g.Player(live[0])
	rest := live[1:]
	g.Push(AskQuestion{Player: next.ID, Question: g.otherDefenderQuestion(m.Against, g.attackValue(m.Against), next, m.For, rest)})
}

// handleAskOtherAction hands the asked player a turn-like menu at the
// requester's behest: they perform one action of their choice (or nothing,
// via Done), after which the requester's turn resumes automatically —
// checkContinue re-offers the requester's menu because ActiveTurn never
// changed.
func (g *Game) handleAskOtherAction(m AskOtherAction) {
	q := g.Player(m.Asked)
	req := g.Player(m.Requester)
	if q == nil || q.KOed || req == nil {
		return
	}
	// Build the menu as if it were q's turn so choices stamp q as the
	// actor (attack/thwart targets key off ActiveTurn); ActiveTurn is
	// restored before any message processing resumes.
	saved := g.ActiveTurn
	g.ActiveTurn = q.ID
	menu := g.turnMenu(q, false)
	g.ActiveTurn = saved
	menu.Prompt = fmt.Sprintf("%s asks you to act", req.Name)
	g.Push(AskQuestion{Player: q.ID, Question: menu})
}

func (g *Game) handleDefends(m Defends) {
	against := g.Entity(m.Against)
	if against == nil {
		return
	}
	var attack int
	switch e := against.(type) {
	case *Villain:
		attack = e.AttackVal + e.BoostCount
	case *Minion:
		attack = e.AttackVal + e.BoostCount
		e.BoostCount = 0
	default:
		return
	}

	// Sense upgrades and similar attachments weaken the attacker's strike
	// (Heightened Hearing; approximation: auto-consumed when beneficial).
	if defender := g.Player(m.Defender); defender != nil {
		for _, id := range defender.Upgrades {
			u := g.Upgrades[id]
			if u == nil || u.AttachTo != m.Against {
				continue
			}
			if mod := behavior(u.Code).AttachedEnemyAttackMod; mod < 0 {
				attack += mod
				g.Logf("%s weakens the attack by %d", u.EDef().Name, -mod)
				g.Push(DiscardControlled{Player: u.Owner, ID: u.ID})
			}
		}
	}

	switch def := g.Entity(m.Defender).(type) {
	case *Player:
		damage := attack
		if !m.Undefended && def.IsHero() && !def.Exhausted {
			if !m.NoExhaust {
				def.Exhausted = true
			}
			prevented := def.DefenseStat(g) + m.DefBonus
			damage -= prevented
			g.tlogf("log.defendsPrevented", def, prevented)
		}
		if m.ExtraPrevent > 0 {
			damage -= m.ExtraPrevent
			g.tlogf("log.preventsAdditionalDamage", def, m.ExtraPrevent)
		}
		if m.PreventAll {
			g.tlogf("log.preventsAllDamage", def)
			damage = 0
		}
		g.Push(DamageEntity{Target: def.ID, Damage: max(0, damage), Source: m.Against})
		g.Push(WindowDefended{Defender: def.ID, Against: m.Against, DamageTaken: max(0, damage), Via: m.Via})
		// Retaliate on the defending identity.
		if rv := retaliateOf(g, def); rv > 0 {
			g.Push(DamageEntity{Target: m.Against, Damage: rv, Source: def.ID})
		}
	case *Ally:
		def.Exhausted = true
		damage := max(0, attack-def.Defense())
		g.Push(DamageEntity{Target: def.ID, Damage: damage, Source: m.Against})
		g.Push(WindowDefended{Defender: def.ID, Against: m.Against, DamageTaken: damage})
		if rv := retaliateOf(g, def); rv > 0 {
			g.Push(DamageEntity{Target: m.Against, Damage: rv, Source: def.ID})
		}
		// Overkill-like spill is not implemented for minions.
	}
	// Cleanup boost cards after the activation resolves.
	if v, ok := against.(*Villain); ok {
		g.Push(ClearBoosts{Enemy: v.ID})
		g.Push(WindowAfterEnemyAttacked{Enemy: v.ID, Player: m.Defender})
	}
}

func (g *Game) handlePlayCard(m PlayCard) {
	p := g.Player(m.Player)
	if p == nil {
		return
	}
	card, ok := p.Hand.Find(m.Card.ID)
	if !ok {
		g.tlogf("log.cannotPlayMissing", m.Card.Code)
		return
	}
	def := card.Def()
	g.consumeDiscount(p, def)
	if def.Type == "ally" {
		p.AllyPlayedThisRound = true
	}
	// Clobber (45046): the round's first played card returns to hand
	// after resolving.
	firstOfRound := !g.UsedThisRound["card-played"]
	g.UsedThisRound["card-played"] = true
	// A fresh play resets pending event bonuses (Embiggen!/Shrink are
	// per-event).
	delete(g.EventDamageBonus, p.ID)
	delete(g.EventThreatBonus, p.ID)
	leadershipBoost := map[PlayerID]bool{}
	// Pay resources: discard chosen cards. Every paid card is journaled so
	// the log shows what a play cost, not just that it happened.
	for _, id := range m.Paid.CardIDs {
		rc, ok := p.Hand.Remove(id)
		if !ok {
			continue
		}
		p.Discard = append(p.Discard, rc)
		g.tlogf("log.paysWith", p.Name, rc)
		// Mutant Genesis resource riders: Aggressive Energy (+1 damage for
		// Attack events) and Defensive Energy (draw 1 for Defense events).
		switch data.BaseCode(rc.Code) {
		case "32047", "35020":
			if def.Type == "event" && def.HasTrait("attack") {
				g.EventDamageBonus[p.ID] += 1
				g.tlogf("log.aggressiveEnergy", def)
			}
		case "32018", "38017":
			if def.Type == "event" && def.HasTrait("defense") {
				g.Push(DrawCards{Player: p.ID, N: 1})
				g.tlogf("log.defensiveEnergy")
			}
		case "33018", "36021":
			// Effective Leadership: the played ally gets +1 THW / +1 ATK
			// this phase (applied in the ally case below via a flag).
			if def.Type == "ally" {
				leadershipBoost[p.ID] = true
				g.tlogf("log.effectiveLeadership", def)
			}
		case "34020", "37016":
			// Passion for Justice: Thwart events remove 1 extra threat.
			if def.Type == "event" && def.HasTrait("thwart") {
				g.EventThreatBonus[p.ID] += 1
				g.tlogf("log.passionForJustice", def)
			}
		}
	}
	_, stillInHand := p.Hand.Find(card.ID)
	if stillInHand {
		p.Hand.Remove(card.ID)
	}

	switch def.Type {
	case "ally":
		a := &Ally{
			ID:        g.nextEntityID(KindAlly),
			Code:      def.Code,
			Owner:     p.ID,
			MaxHP:     deref(def.HP, 1),
			AttackVal: deref(def.Attack, 0),
			ThwartVal: deref(def.Thwart, 0),
			Tough:     def.HasKeyword("Toughness"),
		}
		if leadershipBoost[p.ID] {
			a.BonusTHW++
			a.BonusATK++
		}
		g.Allies[a.ID] = a
		p.Allies = append(p.Allies, a.ID)
		g.tlogf("log.plays", p.Name, def)
		g.Push(AllyEnteredPlay{Ally: a.ID, Player: p.ID})
		if b := behavior(def.Code); b.OnPlay != nil {
			g.Push(b.OnPlay(g, a)...)
		}
	case "support":
		s := &Support{ID: g.nextEntityID(KindSupport), Code: def.Code, Owner: p.ID}
		g.Supports[s.ID] = s
		p.Supports = append(p.Supports, s.ID)
		g.tlogf("log.plays", p.Name, def)
		if b := behavior(def.Code); b.OnPlay != nil {
			g.Push(b.OnPlay(g, s)...)
		}
	case "upgrade":
		u := &Upgrade{ID: g.nextEntityID(KindUpgrade), Code: def.Code, Owner: p.ID}
		g.Upgrades[u.ID] = u
		p.Upgrades = append(p.Upgrades, u.ID)
		g.tlogf("log.plays", p.Name, def)
		if b := behavior(def.Code); b.OnPlay != nil {
			g.Push(b.OnPlay(g, u)...)
		}
	case "player_side_scheme":
		// Player-owned side schemes (Focus the Senses, Establish
		// Perimeter): they enter play with their printed threat and can
		// be thwarted like any other scheme.
		s := &SideScheme{
			ID:         g.nextEntityID(KindSideScheme),
			Code:       def.Code,
			Owner:      p.ID,
			PlayerSide: true,
			Threat:     deref(def.BaseThreat, 2),
			MaxThreat:  deref(def.BaseThreat, 2),
			Crisis:     sideSchemeIsCrisis(def),
			Hazard:     def.Hazards,
		}
		g.SideSchemes[s.ID] = s
		g.tlogf("log.playsThreat", p.Name, def, s.Threat)
		if b := behavior(def.Code); b.OnPlay != nil {
			g.Push(b.OnPlay(g, s)...)
		}
	default:
		// Events and anything else: resolve then discard.
		g.tlogf("log.plays", p.Name, def)
		ec := &EventCard{Code: def.Code, Owner: p.ID, Paid: m.Paid}
		if def.Type == "event" {
			// Announce the play before the effect resolves so that
			// interrupts (Embiggen!) and responses (Morphogenetics)
			// can hook in.
			g.Push(EventPlayed{Player: p.ID, Card: card})
		}
		if b := behavior(def.Code); b.OnPlay != nil {
			g.Push(b.OnPlay(g, ec)...)
		}
		p.Discard = append(p.Discard, card)
		if firstOfRound && data.BaseCode(def.Code) == "45046" {
			g.tlogf("log.clobberReturns")
			g.PushFront(ReturnDiscardCard{Player: p.ID, CardID: card.ID})
		}
	}
}

// handlePlayDefenseEvent resolves a defense event chosen from the defense
// prompt: pay, discard, then let the behavior build the Defends result.
func (g *Game) handlePlayDefenseEvent(m PlayDefenseEvent) {
	p := g.Player(m.Player)
	if p == nil {
		return
	}
	card, ok := p.Hand.Find(m.Card.ID)
	if !ok {
		g.tlogf("log.cannotPlayMissingDefense", m.Card.Code)
		return
	}
	def := card.Def()
	for _, id := range m.Paid.CardIDs {
		if rc, ok := p.Hand.Remove(id); ok {
			p.Discard = append(p.Discard, rc)
		}
	}
	p.Hand.Remove(card.ID)
	p.Discard = append(p.Discard, card)
	g.tlogf("log.plays", p.Name, def)
	b := behavior(def.Code)
	if b.DefenseEvent == nil {
		return
	}
	ec := &EventCard{Code: def.Code, Owner: p.ID, Paid: m.Paid}
	d, extra, ok := b.DefenseEvent(g, p, ec, m.Against)
	if !ok {
		return
	}
	if d.Defender != "" {
		g.Push(d)
	}
	g.Push(extra...)
}

// handleAllyEntersPlayFree puts an ally into play without paying its cost
// (Quinjet, Make the Call, Lockjaw), running its enters-play response.
func (g *Game) handleAllyEntersPlayFree(m AllyEntersPlayFree) {
	p := g.Player(m.Player)
	if p == nil {
		return
	}
	var card Card
	var found bool
	if m.FromOwner == "" {
		card, found = p.Hand.Find(m.Card.ID)
		if found {
			p.Hand.Remove(card.ID)
		}
	} else if src := g.Player(m.FromOwner); src != nil {
		card, found = src.Discard.Find(m.Card.ID)
		if found {
			src.Discard.Remove(card.ID)
		}
	}
	if !found {
		g.tlogf("log.cannotPutMissingAlly", m.Card.Code)
		return
	}
	def := card.Def()
	a := &Ally{
		ID:        g.nextEntityID(KindAlly),
		Code:      def.Code,
		Owner:     p.ID,
		MaxHP:     deref(def.HP, 1),
		AttackVal: deref(def.Attack, 0),
		ThwartVal: deref(def.Thwart, 0),
		Tough:     def.HasKeyword("Toughness"),
	}
	g.Allies[a.ID] = a
	p.Allies = append(p.Allies, a.ID)
	p.AllyPlayedThisRound = true
	g.tlogf("log.putsIntoPlay", p.Name, def)
	g.Push(AllyEnteredPlay{Ally: a.ID, Player: p.ID})
	if b := behavior(def.Code); b.OnPlay != nil {
		g.Push(b.OnPlay(g, a)...)
	}
}

// handleTreacheryWindow offers hand-card treachery interrupts (Get Behind
// Me!) before the treachery resolves.
func (g *Game) handleTreacheryWindow(m TreacheryWindow) {
	p := g.Player(m.Player)
	if p == nil {
		g.Push(TreacheryResolve{Player: m.Player, Card: m.Card})
		return
	}
	var interrupts []Choice
	for _, hc := range p.Hand {
		hb := behavior(hc.Def().Code)
		if hb.TreacheryInterrupt == nil {
			continue
		}
		repl := hb.TreacheryInterrupt(g, p, hc)
		if repl == nil {
			continue
		}
		final := append([]Message{ConsumeHandCard{Player: p.ID, CardID: hc.ID}}, repl...)
		cost := deref(hc.Def().Cost, 0)
		choice := Choice{
			ID: "interrupt-" + hc.ID, Label: Tf("m.playCard", hc),
			Kind: ChoicePlay, CardCode: hc.Code,
		}
		if cost > 0 && len(p.Hand) > cost {
			var pays []Choice
			for _, rc := range p.Hand {
				if rc.ID == hc.ID {
					continue
				}
				pays = append(pays, Choice{
					Label: Tf("m.discardCard", rc), Kind: ChoiceCard, CardCode: rc.Code,
				}.Msgs(append([]Message{DiscardCards{Player: p.ID, Cards: CardList{rc}}}, final...)...))
			}
			interrupts = append(interrupts, choice.WithThen(
				Ask(Tf("c.discardCardSToPayFor", cost, hc), pays...)))
		} else if cost > 0 {
			continue // cannot pay
		} else {
			interrupts = append(interrupts, choice.Msgs(final...))
		}
	}
	// In-play ally cancels (Black Widow): exhaust the ally to cancel the
	// treachery's effects. Approximation: the [mental] payment and the
	// reveal-another-card rider are skipped.
	for _, id := range p.Allies {
		a := g.Allies[id]
		if a == nil || a.Exhausted || a.Code != "01075" {
			continue
		}
		interrupts = append(interrupts, Choice{
			ID: "bw-cancel", Label: Tf("m.bwCancel", m.Card),
			Kind: ChoiceAbility, SourceID: a.ID, CardCode: a.Code,
		}.Msgs(ExhaustEntity{ID: a.ID}))
	}
	// In-play upgrades with a treachery interrupt (Spider-Tingle): the
	// hook returns the full replacement including its own discard.
	for _, id := range sortedIDs(g.Upgrades) {
		u := g.Upgrades[id]
		if u == nil || u.Owner != m.Player {
			continue
		}
		hb := behavior(u.Code)
		if hb.TreacheryInterrupt == nil {
			continue
		}
		if repl := hb.TreacheryInterrupt(g, p, m.Card); repl != nil {
			interrupts = append(interrupts, Choice{
				ID: "upgrade-interrupt-" + u.ID.String(), Label: Tf("m.use", u),
				Kind: ChoiceAbility, SourceID: u.ID, CardCode: u.Code,
			}.Msgs(repl...))
		}
	}
	// Negasonic Teenage Warhead (44044): 2 damage to her cancels the
	// treachery's effects.
	for _, id := range p.Allies {
		a := g.Allies[id]
		if a == nil || data.BaseCode(a.Code) != "44044" {
			continue
		}
		interrupts = append(interrupts, Choice{
			ID: "ntw-cancel", Label: Tf("m.ntwCancel", m.Card),
			Kind: ChoiceAbility, SourceID: a.ID, CardCode: a.Code,
		}.Msgs(
			DamageEntity{Target: a.ID, Damage: 2, Source: p.ID},
			DiscardEncounterCard{Card: m.Card},
			RevealNextEncounter{Player: m.Player},
		))
	}
	// Stepford Cuckoos (45049): exhaust + a psi counter cancels the
	// treachery; the revealer draws a replacement encounter card.
	for _, id := range sortedIDs(g.Supports) {
		s2 := g.Supports[id]
		if s2 == nil || s2.Owner != m.Player || s2.Exhausted || s2.Counters <= 0 {
			continue
		}
		if data.BaseCode(s2.Code) != "45049" {
			continue
		}
		interrupts = append(interrupts, Choice{
			ID: "cuckoos-cancel", Label: Tf("m.cuckoosCancel", m.Card),
			Kind: ChoiceAbility, SourceID: s2.ID, CardCode: s2.Code,
		}.Msgs(
			ExhaustEntity{ID: s2.ID},
			AddEntityCounter{ID: s2.ID, N: -1},
			DiscardEncounterCard{Card: m.Card},
			RevealNextEncounter{Player: m.Player},
		))
	}
	if len(interrupts) == 0 {
		g.Push(TreacheryResolve{Player: m.Player, Card: m.Card})
		return
	}
	interrupts = append(interrupts, Choice{
		ID: "continue", Label: Tf("m.letResolve", m.Card), Kind: ChoicePass,
	}.Msgs(TreacheryResolve{Player: m.Player, Card: m.Card}))
	g.Push(AskQuestion{Player: p.ID, Question: Ask(Tf("q.interruptsQ", m.Card), interrupts...)})
}

// indirectQuestion builds the one-point distribution prompt for indirect
// damage: each pick deals 1 damage and queues the remaining points.
func (g *Game) indirectQuestion(p *Player, n int, chars []EntityID) *Question {
	var picks []Choice
	for _, id := range chars {
		label := p.Name
		if a := g.Allies[id]; a != nil {
			label = a.EDef().Name
		}
		msgs := []Message{DamageEntity{Target: id, Damage: 1}}
		if n > 1 {
			msgs = append(msgs, IndirectDamage{Player: p.ID, N: n - 1})
		}
		picks = append(picks, Choice{Label: Tf("c.1DamageTo", label), Kind: ChoiceTarget, SourceID: id}.
			Msgs(msgs...))
	}
	return Ask(Tf("q.assignIndirect", p.Name, n), picks...)
}

// handleTreacheryResolve performs the treachery resolution, or the
// cancellation replacement.
func (g *Game) handleTreacheryResolve(m TreacheryResolve) {
	p := g.Player(m.Player)
	if p == nil {
		return
	}
	def := m.Card.Def()
	if m.Cancelled {
		g.tlogf("log.cancelsEffects", p.Name, def)
		return
	}
	t := &Treachery{ID: g.nextEntityID(KindTreachery), Code: def.Code}
	g.Treacheries[t.ID] = t
	if b := behavior(def.Code); b.ResolveTreachery != nil {
		g.Push(b.ResolveTreachery(g, t, p)...)
	} else {
		g.tlogf("log.noEffect", def)
		g.Delete(t.ID)
	}
}

// handleSenseEnterPlay plays a Sense upgrade from the player's Sense deck
// into play, running its OnPlay (attach target choice).
func (g *Game) handleSenseEnterPlay(m SenseEnterPlay) {
	p := g.Player(m.Player)
	if p == nil {
		return
	}
	card, ok := p.SenseDeck.Remove(m.Card.ID)
	if !ok {
		g.tlogf("log.cannotPlayMissingSense", m.Card.Code)
		return
	}
	def := card.Def()
	u := &Upgrade{ID: g.nextEntityID(KindUpgrade), Code: def.Code, Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)
	g.tlogf("log.playsSense", p.Name, def)
	if b := behavior(def.Code); b.OnPlay != nil {
		g.Push(b.OnPlay(g, u)...)
	}
}

// handlePlayDiscardAlly plays an ally from the player's discard pile
// (Lockjaw).
func (g *Game) handlePlayDiscardAlly(m PlayDiscardAlly) {
	p := g.Player(m.Player)
	if p == nil {
		return
	}
	card, ok := p.Discard.Find(m.Card.ID)
	if !ok {
		g.tlogf("log.cannotPlayMissingDiscardAlly", m.Card.Code)
		return
	}
	for _, id := range m.Paid.CardIDs {
		if rc, ok := p.Hand.Remove(id); ok {
			p.Discard = append(p.Discard, rc)
		}
	}
	p.Discard.Remove(card.ID)
	def := card.Def()
	a := &Ally{
		ID:        g.nextEntityID(KindAlly),
		Code:      def.Code,
		Owner:     p.ID,
		MaxHP:     deref(def.HP, 1),
		AttackVal: deref(def.Attack, 0),
		ThwartVal: deref(def.Thwart, 0),
		Tough:     def.HasKeyword("Toughness"),
	}
	g.Allies[a.ID] = a
	p.Allies = append(p.Allies, a.ID)
	p.AllyPlayedThisRound = true
	g.consumeDiscount(p, def)
	g.tlogf("log.playsFromDiscard", p.Name, def)
	if b := behavior(def.Code); b.OnPlay != nil {
		g.Push(b.OnPlay(g, a)...)
	}
}

// EventCard adapts an event being resolved to the Entity interface for
// behavior hooks; Paid records how it was paid for so icon-dependent
// riders can fire.
type EventCard struct {
	Code  string
	Owner PlayerID
	Paid  CostPaid
}

func (e *EventCard) EID() EntityID                        { return EntityID("event") }
func (e *EventCard) ECode() string                        { return e.Code }
func (e *EventCard) EDef() *data.CardDef                  { return DB.MustLookup(e.Code) }
func (e *EventCard) EOwner() PlayerID                     { return e.Owner }
func (e *EventCard) EExhausted() bool                     { return false }
func (e *EventCard) React(g *Game, msg Message) []Message { return nil }

func (g *Game) revealEncounterCard(pid PlayerID, card Card) {
	p := g.Player(pid)
	def := card.Def()
	g.tlogf("log.reveals", p.Name, def)

	// Obligations resolve for their owner, not the revealer, and do not
	// enter the encounter discard.
	if def.Type == "obligation" {
		owner := g.Player(card.Owner)
		if owner == nil {
			owner = p
		}
		if b := behavior(def.Code); b.ResolveObligation != nil {
			g.Push(b.ResolveObligation(g, owner, card)...)
		} else {
			owner.ObligationDiscard = append(owner.ObligationDiscard, card)
			g.tlogf("log.noEffect", def)
		}
		return
	}

	// Cards that enter play (minion, side scheme, attachment, environment)
	// reach the encounter discard only when they later leave play; a
	// treachery (or anything unhandled) resolves and is discarded now.
	switch def.Type {
	case "minion", "side_scheme", "attachment", "environment":
	default:
		g.EncounterDiscard = append(g.EncounterDiscard, card)
	}

	switch def.Type {
	case "minion":
		mn := &Minion{
			ID:          g.nextEntityID(KindMinion),
			Code:        def.Code,
			MaxHP:       deref(def.HP, 1),
			AttackVal:   deref(def.Attack, 0),
			SchemeVal:   deref(def.Scheme, 0),
			Tough:       def.HasKeyword("Toughness"),
			Guard:       def.HasKeyword("Guard"),
			EngagedWith: pid,
		}
		g.Minions[mn.ID] = mn
		g.Push(MinionEntersPlay{MinionID: mn.ID, Player: pid})
		if b := behavior(def.Code); b.OnPlay != nil {
			g.Push(b.OnPlay(g, mn)...)
		}
		if def.HasKeyword("Quickstrike") && p.IsHero() {
			g.Push(DamageEntity{Target: p.ID, Damage: mn.AttackVal, Source: mn.ID})
		}
	case "side_scheme":
		s := &SideScheme{
			ID:        g.nextEntityID(KindSideScheme),
			Code:      def.Code,
			Threat:    deref(def.BaseThreat, 1) + len(g.Players) - 1,
			MaxThreat: deref(def.BaseThreat, 1) + 2*(len(g.Players)-1),
			Crisis:    sideSchemeIsCrisis(def),
			Hazard:    def.Hazards,
		}
		g.SideSchemes[s.ID] = s
		g.tlogMajorf("log.sideSchemeEnters", def, s.Threat)
		if b := behavior(def.Code); b.OnPlay != nil {
			g.Push(b.OnPlay(g, s)...)
		}
		// Hazard icons on side schemes deal their extra encounter cards
		// during the villain phase's deal step, not immediately.
	case "treachery":
		// The interrupt window (Get Behind Me!) runs before resolution.
		g.Push(TreacheryWindow{Player: pid, Card: card})
	case "attachment":
		t := &Attachment{ID: g.nextEntityID(KindAttachment), Code: def.Code}
		g.Attachments[t.ID] = t
		if b := behavior(def.Code); b.OnAttach != nil {
			g.Push(b.OnAttach(g, t, EntityID(""))...)
		} else {
			// default: attach to first villain
			for id := range g.Villains {
				t.Target = id
				break
			}
			g.tlogf("log.attachesVillain", def)
		}
	case "environment":
		env := &Environment{ID: g.nextEntityID(KindEnvironment), Code: def.Code}
		g.Environments[env.ID] = env
		g.tlogMajorf("log.environmentEnters", def)

	default:
		g.tlogf("log.unhandledType", def.Type, def)
	}

	if def.HasKeyword("Surge") {
		if c, ok := g.drawEncounter(); ok {
			g.Push(RevealEncounterCard{Player: pid, Card: c})
		}
	}
}

func (g *Game) advanceVillainStage(id EntityID) {
	v := g.Villains[id]
	if v == nil {
		return
	}
	// A scenario hook owns the whole defeat flow: these scenarios predate
	// data-driven stage chaining and handle every defeat themselves (the
	// Sinister Six sets members aside mid-progression, Kang time-jumps to a
	// scheme stage). The hook fires on each defeat, before any stage chain.
	if scen := g.Scenario(); scen.OnVillainDefeated != nil {
		g.Push(scen.OnVillainDefeated(g, v)...)
		return
	}
	if v.Stage >= len(v.stageCodes) {
		g.Push(GameOver{Won: true, Reason: Tf("reason.villainDefeated")})
		return
	}
	v.Stage++
	v.Code = v.stageCodes[v.Stage-1]
	def := v.EDef()
	v.Damage = 0
	v.MaxHP = deref(def.HP, v.MaxHP)
	v.SchemeVal = deref(def.Scheme, v.SchemeVal)
	v.AttackVal = deref(def.Attack, v.AttackVal)
	v.Tough = def.HasKeyword("Toughness")
	g.tlogf("log.advancesStage", def, def.StageLabel)
	if b := behavior(v.Code); b.VillainStage != nil {
		g.Push(b.VillainStage(g, v, v.Stage)...)
	}
}

// hinderTotal sums the printed Hinder values of side schemes in play
// (per hero). Hinder is parsed into Keywords at data load; logic never
// re-reads the printed text.
func (g *Game) hinderTotal() int {
	total := 0
	for _, s := range g.SideSchemes {
		if s == nil || s.PlayerSide {
			continue
		}
		total += s.EDef().KeywordValue("Hinder") * len(g.Players)
	}
	return total
}

// handCardHolding finds the first player holding a card with one of the
// given base codes in hand (interrupt-event windows).
func (g *Game) handCardHolding(codes ...string) (*Player, Card, bool) {
	for _, p := range g.Players {
		if p.KOed {
			continue
		}
		for _, hc := range p.Hand {
			for _, code := range codes {
				if data.BaseCode(hc.Code) == code {
					return p, hc, true
				}
			}
		}
	}
	return nil, Card{}, false
}

// attackActivationPending reports whether a villain's attack question is
// still queued (Preemptive Strike window).
func (g *Game) attackActivationPending(villain EntityID) bool {
	for _, msg := range g.queue {
		if m, ok := msg.(AskAttack); ok && m.Enemy == villain {
			return true
		}
	}
	return false
}

// schemeActivationPending reports whether a villain's scheme resolution is
// still queued (boost cards revealed during a scheme activation vs an
// attack).
func (g *Game) schemeActivationPending(villain EntityID) bool {
	for _, msg := range g.queue {
		if m, ok := msg.(ApplyVillainScheme); ok && m.VillainID == villain {
			return true
		}
	}
	return false
}

// sideSchemeInPlay reports whether a side scheme with the given code is
// in play (guard auras).
func (g *Game) sideSchemeInPlay(code string) bool {
	for _, s := range g.SideSchemes {
		if s != nil && s.Code == code {
			return true
		}
	}
	return false
}

// engineHopeLocked reports whether the ally is Hope Summers held by
// Captive Hope (40131).
func engineHopeLocked(g *Game, a *Ally) bool {
	if a == nil || data.BaseCode(a.Code) != "40130" {
		return false
	}
	return g.sideSchemeInPlay("40131")
}

// heroIs reports whether the entity is a player whose identity's base
// code matches (threat/damage locks keyed to a specific hero).
func (g *Game) heroIs(id EntityID, base string) bool {
	p := g.Player(PlayerID(id))
	return p != nil && data.BaseCode(p.HeroCode) == base
}

// minionInPlay reports whether a minion with the given base code is in
// play (Inheritor aura checks).
func (g *Game) minionInPlay(code string) bool {
	for _, mn := range g.Minions {
		if mn != nil && data.BaseCode(mn.Code) == code {
			return true
		}
	}
	return false
}

// boostSpawnTarget picks the first player for boost-spawned cards.
func (g *Game) boostSpawnTarget(v *Villain) PlayerID {
	for _, p := range g.Players {
		if p.FirstPlayer {
			return p.ID
		}
	}
	if len(g.Players) > 0 {
		return g.Players[0].ID
	}
	return ""
}

func (g *Game) handleSchemeDefeated(id EntityID) {
	if s := g.SideSchemes[id]; s != nil {
		g.tlogMajorf("log.sideSchemeDefeated", s)
		if b := behavior(s.Code); b.SideSchemeDefeated != nil {
			g.Push(b.SideSchemeDefeated(g, s)...)
		}
		if strings.Contains(s.EDef().Text, "Victory") {
			g.VictoryDisplay = append(g.VictoryDisplay, Card{ID: g.nextCardID(), Code: s.Code})
		} else {
			// Defeated side schemes are discarded to the encounter pile.
			g.EncounterDiscard = append(g.EncounterDiscard, Card{ID: g.nextCardID(), Code: s.Code})
		}
		g.Delete(id)
		return
	}
	if g.MainScheme != nil && id == g.MainScheme.ID {
		scen := g.Scenario()
		if scen.OnMainSchemeDefeated != nil {
			g.Push(scen.OnMainSchemeDefeated(g, g.MainScheme)...)
		}
		// Default: a cleared main scheme is not "defeated" — the threat
		// removal is already journaled (log.losesThreatMax). Only scenarios
		// with a printed clear-to-win condition hook the event.
	}
}
