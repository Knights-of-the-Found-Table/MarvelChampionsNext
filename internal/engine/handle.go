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
		g.logMajorf("── Round %d ──", g.Round)
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
		g.logf("Player phase ends")
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
				g.logf("First player token passes to %s", next.Name)
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
			g.logf("%s mulligans %s away", p.Name, c.Def().Name)
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
		g.logf("%s's turn begins", p.Name)

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
					g.logf("%s has no cards left to draw", p.Name)
					break
				}
				// Player deck emptied: shuffle the discard pile into a new
				// deck and deal self one facedown encounter card.
				p.Deck = p.Discard
				p.Discard = nil
				g.shuffle(&p.Deck)
				g.logMajorf("%s shuffles their discard pile into a new deck", p.Name)
				if c, ok := g.drawEncounter(); ok {
					p.EncounterDown = append(p.EncounterDown, c)
					g.logf("%s deals themself a facedown encounter card", p.Name)
				}
			}
			card := p.Deck[0]
			p.Deck = p.Deck[1:]
			p.Hand = append(p.Hand, card)
		}

	case ShufflePlayerDeck:
		if p := g.Player(m.Player); p != nil {
			g.shuffle(&p.Deck)
			g.logf("%s shuffles their deck", p.Name)
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
			g.logf("%s changes to %s", p.Name, p.AlterEgoDef().Name)
		} else {
			p.Side = SideHero
			g.logf("%s changes to %s", p.Name, p.HeroDef().Name)
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
			g.logf("%s is confused and cannot thwart", p.Name)
			return
		}
		if g.thwartBlocked(p) {
			g.logf("%s cannot thwart while engaged with %s", p.Name, g.thwartBlockerName(p))
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
			g.logf("%s is stunned and cannot attack", p.Name)
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
		g.addThreat(m.Scheme, m.N, m.Source)

	case ThwartScheme:
		g.removeThreat(m.Scheme, m.N, m.Source)

	case DamageEntity:
		g.damage(m.Target, m.Damage, m.Source)

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
					if boostSpawnsMinion(def) {
						// "Boost: put this card into play" — spawn it
						// instead of contributing boost icons.
						g.logMajorf("Boost card %s enters play!", def.Name)
						g.Push(RevealEncounterCard{Player: g.boostSpawnTarget(v), Card: c})
						continue
					}
					add := deref(def.Boost, 0)
					v.BoostCount += add
					g.logf("Boost card revealed: %s (+%d)", def.Name, add)
					v.RevealedBoosts = append(v.RevealedBoosts, c)
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
		threat := v.SchemeVal + v.BoostCount
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
				"Play Emergency? (reduce the scheme's threat by 1)",
				Choice{ID: "play-emergency", Label: "Play Emergency", Kind: ChoicePlay, CardCode: "01085"}.
					Msgs(DiscardCards{Player: p.ID, Cards: CardList{emerg}},
						SchemeThreat{Scheme: g.MainScheme.ID, N: threat - 1, Source: v.ID},
						ClearBoosts{Enemy: v.ID}),
				Choice{ID: "skip-emergency", Label: "Continue", Kind: ChoicePass}.
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
		g.logMajorf("%s is defeated!", v.EDef().Name)
		g.Push(AdvanceVillainStage{VillainID: v.ID})

	case AdvanceVillainStage:
		g.advanceVillainStage(m.VillainID)

	case MinionDefeated:
		if mn := g.Minions[m.MinionID]; mn != nil {
			g.logMajorf("%s is defeated", mn.EDef().Name)
			for _, a := range mn.Attachments {
				g.Delete(a)
			}
			g.Delete(m.MinionID)
		}

	case MainSchemeMaxed:
		if g.MainScheme != nil && m.Scheme == g.MainScheme.ID {
			scen := g.Scenario()
			if scen.OnMainSchemeMaxed != nil {
				g.Push(scen.OnMainSchemeMaxed(g, g.MainScheme)...)
			} else {
				g.Push(GameOver{Won: false, Reason: "The main scheme completed"})
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
				g.spawnMainScheme(stages, next)
			}
		}

	case FlipMainScheme:
		if g.MainScheme != nil && m.Scheme == g.MainScheme.ID {
			s := g.MainScheme
			s.Code = s.StageCodes[s.Stage-1]
			g.logMajorf("Main scheme flips to stage %s (threat %d/%d)",
				s.EDef().StageLabel, s.Threat, s.MaxThreat)
		}

	case GameOver:
		g.Over = true
		g.Won = m.Won
		g.Reason = m.Reason
		g.pending = nil
		g.queue = nil
		if m.Won {
			g.logMajorf("🏆 Victory: %s", m.Reason)
		} else {
			g.logMajorf("💀 Defeat: %s", m.Reason)
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
			g.logMajorf("Villain flips to %s", def.Name)
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
				g.logf("%s engages %s", mn.EDef().Name, p.Name)
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

	case ObligationResolve:
		if p := g.Player(m.Player); p != nil {
			if m.Remove {
				p.ObligationRemoved = append(p.ObligationRemoved, m.Card)
				g.logf("%s is removed from the game", m.Card.Def().Name)
			} else {
				p.ObligationDiscard = append(p.ObligationDiscard, m.Card)
				g.logf("%s is discarded", m.Card.Def().Name)
			}
		}

	case DiscardControlled:
		g.discardControlled(m.Player, m.ID)

	case AddAccelerationToken:
		if g.MainScheme != nil && m.Scheme == g.MainScheme.ID {
			g.MainScheme.AccelerationTokens++
			g.logf("%s gains an acceleration token", g.MainScheme.EDef().Name)
		}

	case RevealNextEncounter:
		if c, ok := g.drawEncounter(); ok {
			g.revealEncounterCard(m.Player, c)
		}

	case PlayDefenseEvent:
		g.handlePlayDefenseEvent(m)

	case AddEntityCounter:
		switch t := g.Entity(m.ID).(type) {
		case *Support:
			t.Counters += m.N
			g.logMinorf("%s counters: %d", t.EDef().Name, t.Counters)
		case *Upgrade:
			t.Counters += m.N
			g.logMinorf("%s counters: %d", t.EDef().Name, t.Counters)
		case *Ally:
			t.Counters += m.N
			g.logMinorf("%s counters: %d", t.EDef().Name, t.Counters)
		}

	case ReturnControlled:
		if p := g.Player(m.Player); p != nil {
			if e := g.Entity(m.ID); e != nil {
				code := e.ECode()
				g.Delete(m.ID)
				p.Hand = append(p.Hand, Card{ID: g.nextCardID(), Code: code, Owner: p.ID})
				g.logf("%s returns %s to their hand", p.Name, e.EDef().Name)
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
			}
			if tgt := g.Entity(m.Target); tgt != nil {
				g.logf("%s attaches to %s", u.EDef().Name, tgt.EDef().Name)
			}
		}

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
				picks = append(picks, Choice{Label: "Discard " + c.Def().Name, Kind: ChoiceCard, CardCode: c.Code}.
					Msgs(DiscardCards{Player: p.ID, Cards: CardList{c}}))
			}
			q := AskN("Discard 2 cards", min(2, len(p.Hand)), picks...)
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
					g.logf("%s returns to play with 1 damage (Rapid Response)", a.EDef().Name)
					return
				}
			}
		}

	case TempHandSizeMsg:
		if p := g.Player(m.Player); p != nil {
			p.TempHandSize += m.N
			g.Logf("%s gets +%d hand size until the end of the phase", p.Name, m.N)
		}

	case CostDiscountApply:
		if p := g.Player(m.Player); p != nil && m.Amount > 0 {
			p.CostDiscounts = append(p.CostDiscounts, CostDiscount{Amount: m.Amount})
			g.logf("%s's next card this phase costs %d less", p.Name, m.Amount)
		}

	case SenseEnterPlay:
		g.handleSenseEnterPlay(m)

	case ShuffleIntoDeck:
		if p := g.Player(m.Player); p != nil {
			if c, ok := p.Discard.Remove(m.CardID); ok {
				p.Deck = append(p.Deck, c)
				g.shuffle(&p.Deck)
				g.logf("%s shuffles %s into their deck", p.Name, c.Def().Name)
			}
		}

	case GrantTrait:
		switch t := g.Entity(m.Target).(type) {
		case *Player:
			t.ExtraTraits = append(t.ExtraTraits, m.Trait)
			g.logf("%s gains the %s trait", t.Name, m.Trait)
		case *Ally:
			t.ExtraTraits = append(t.ExtraTraits, m.Trait)
			g.logf("%s gains the %s trait", t.EDef().Name, m.Trait)
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
				g.logf("%s resolves %s (back on top)", p.Name, def.Name)
			} else {
				p.SideDiscard = append(p.SideDiscard, card)
				g.logf("%s resolves %s", p.Name, def.Name)
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
			g.logf("%s discards %s from the side deck", p.Name, card.Def().Name)
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
				g.logf("cannot find %s to put into play", m.Card.Code)
				return
			}
			p.Hand.Remove(card.ID)
			p.Deck.Remove(card.ID)
			p.Discard.Remove(card.ID)
			def := card.Def()
			u := &Upgrade{ID: g.nextEntityID(KindUpgrade), Code: def.Code, Owner: p.ID}
			g.Upgrades[u.ID] = u
			p.Upgrades = append(p.Upgrades, u.ID)
			g.logf("%s puts %s into play", p.Name, def.Name)
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
					g.logf("%s takes %s from under Echo", p.Name, c.Def().Name)
					return
				}
			}
		}

	case RecycleFromDiscard:
		if p := g.Player(m.Player); p != nil {
			if src := g.Player(m.From); src != nil {
				if c, ok := src.Discard.Remove(m.CardID); ok {
					p.Hand = append(p.Hand, c)
					g.logf("%s takes %s from %s's discard pile", p.Name, c.Def().Name, src.Name)
				}
			}
		}

	case SwapHandWithDeckTop:
		if p := g.Player(m.Player); p != nil && len(p.Deck) > 0 {
			if c, ok := p.Hand.Remove(m.CardID); ok {
				top := p.Deck[0]
				p.Deck[0] = c
				p.Hand = append(p.Hand, top)
				g.logf("%s swaps %s with the top of their deck", p.Name, c.Def().Name)
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
				g.logf("%s takes %s back to hand", p.Name, c.Def().Name)
			}
		}

	case DiscardToBottom:
		if p := g.Player(m.Player); p != nil {
			if c, ok := p.Discard.Remove(m.CardID); ok {
				p.Deck = append(p.Deck, c)
				g.logf("%s puts %s on the bottom of their deck", p.Name, c.Def().Name)
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
					g.logMinorf("%s tucks a card under %s (%d stored)", p.Name, s.EDef().Name, s.Counters)
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
				g.logMinorf("%s takes %d stored cards from %s", p.Name, take, s.EDef().Name)
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
			g.logf("%s gets +%d THW / +%d ATK / +%d DEF until the end of the phase", p.Name, m.THW, m.ATK, m.DEF)
		}

	case AllyStatBonus:
		if a := g.Allies[m.Ally]; a != nil {
			a.BonusTHW += m.THW
			a.BonusATK += m.ATK
			g.logf("%s gets +%d THW / +%d ATK until the end of the phase", a.EDef().Name, m.THW, m.ATK)
		}

	case TakeDeckCard:
		p := g.Player(m.Player)
		if p == nil {
			return
		}
		if card, ok := p.Deck.Remove(m.CardID); ok {
			p.Hand = append(p.Hand, card)
			g.logf("%s takes %s from their deck", p.Name, card.Def().Name)
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
	g.logMajorf("%s reveals their nemesis set!", p.Name)
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
			g.logMajorf("%s enters play engaged with %s", def.Name, p.Name)
			g.Push(MinionEntersPlay{MinionID: mn.ID, Player: p.ID})
			if def.HasKeyword("Quickstrike") && p.IsHero() {
				g.Push(DamageEntity{Target: p.ID, Damage: mn.AttackVal, Source: mn.ID})
			}
			if b := behavior(def.Code); b.OnPlay != nil {
				g.Push(b.OnPlay(g, mn)...)
			}
		case "side_scheme":
			s := &SideScheme{
				ID:        g.nextEntityID(KindSideScheme),
				Code:      def.Code,
				Threat:    deref(def.BaseThreat, 1) + len(g.Players) - 1,
				MaxThreat: deref(def.BaseThreat, 1) + 2*(len(g.Players)-1),
				Crisis:    strings.Contains(def.Text, "Crisis") || strings.Contains(def.Text, "crisis"),
				Hazard:    def.Hazards,
			}
			g.SideSchemes[s.ID] = s
			g.logMajorf("Side scheme %s enters play (threat %d)", def.Name, s.Threat)
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
	g.spawnDroneMinion(card)
	g.logMajorf("A Drone enters play engaged with %s", p.Name)
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
		g.logf("%s returns to the bottom of the Sense deck", def.Name)
		return
	}
	p.Discard = append(p.Discard, Card{ID: g.nextCardID(), Code: code, Owner: p.ID})
	g.logf("%s is discarded", e.EDef().Name)
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
		if mn.EngagedWith == p.ID && behavior(mn.Code).EngagedBlocksThwart {
			return mn.EDef().Name
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
	g.logf("%s takes the first player token", g.Players[first].Name)

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
	scen := g.Scenario()
	if scen.Setup != nil {
		g.Push(scen.Setup(g)...)
	}
	g.logMajorf("Scenario: %s", scen.Name)
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
	pick := &Question{Type: "choose_n", Prompt: "Select cards to mulligan"}
	for _, c := range p.Hand {
		def := c.Def()
		pick.Choices = append(pick.Choices, Choice{
			Label: def.Name, Kind: ChoiceCard, CardCode: def.Code,
		}.Msgs(MulliganCard{Player: p.ID, CardID: c.ID}))
	}
	q := Ask("Mulligan?",
		Choice{ID: "keep", Label: "Keep hand", Kind: ChoicePass},
		Choice{ID: "mulligan", Label: "Discard selected cards and redraw", Kind: ChoiceCard}.WithThen(pick),
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
		pick := &Question{Type: "choose_n", Prompt: "Discard any number of cards"}
		for _, c := range p.Hand {
			def := c.Def()
			pick.Choices = append(pick.Choices, Choice{
				Label: def.Name, Kind: ChoiceCard, CardCode: def.Code,
			}.Msgs(DiscardCards{Player: p.ID, Cards: CardList{c}}))
		}
		q := Ask("Discard cards before drawing up?",
			Choice{ID: "keep", Label: "Keep hand", Kind: ChoicePass},
			Choice{ID: "discard", Label: "Discard cards", Kind: ChoiceCard}.WithThen(pick),
		)
		g.Push(AskQuestion{Player: p.ID, Question: q})
		return
	}
	q := &Question{
		Type:   "choose_n",
		Prompt: fmt.Sprintf("Discard down to hand size %d (discard at least %d)", p.HandSize(g), over),
	}
	for _, c := range p.Hand {
		def := c.Def()
		q.Choices = append(q.Choices, Choice{
			Label: def.Name, Kind: ChoiceCard, CardCode: def.Code,
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
				g.logf("Acceleration places %d threat on %s", n, g.MainScheme.EDef().Name)
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
			g.logf("%s is stunned; attack canceled", def.Name)
			return
		}
		g.logf("%s attacks %s", def.Name, p.Name)
		g.Push(DealBoost{Enemy: v.ID})
		g.Push(RevealBoost{Enemy: v.ID})
		g.Push(AskAttack{Enemy: v.ID, Player: p.ID, Trigger: TriggerVillainAttacksYou})
	} else {
		// Scheme
		if v.Confused {
			v.Confused = false
			g.logf("%s is confused; scheme canceled", def.Name)
			return
		}
		g.logf("%s schemes against %s", def.Name, p.Name)
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
	def := mn.EDef()
	if p.IsHero() {
		if mn.Stunned {
			mn.Stunned = false
			g.logf("%s is stunned; attack canceled", def.Name)
			return
		}
		g.logf("%s attacks %s", def.Name, p.Name)
		g.Push(AskAttack{Enemy: mn.ID, Player: p.ID})
	} else {
		if mn.Confused {
			mn.Confused = false
			return
		}
		g.logf("%s schemes against %s", def.Name, p.Name)
		if g.MainScheme != nil {
			g.Push(SchemeThreat{Scheme: g.MainScheme.ID, N: mn.SchemeVal, Source: mn.ID})
		}
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
			Label: fmt.Sprintf("%s — ATK %d / SCH %d", mn.EDef().Name, mn.AttackVal, mn.SchemeVal),
			Kind:  ChoiceTarget, SourceID: id, CardCode: mn.Code,
		}.Msgs(
			MinionActivates{MinionID: id, Player: p.ID},
			AskMinionOrder{Player: p.ID, Remaining: rest},
		))
	}
	g.Push(AskQuestion{Player: p.ID, Question: Ask("Choose the next minion to activate", choices...)})
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
		attack = e.AttackVal
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
			g.logf("%s defends (%d damage prevented)", def.Name, prevented)
		}
		if m.ExtraPrevent > 0 {
			damage -= m.ExtraPrevent
			g.logf("%s prevents %d additional damage", def.Name, m.ExtraPrevent)
		}
		if m.PreventAll {
			g.logf("%s prevents all damage", def.Name)
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
		g.logf("cannot play missing card %s", m.Card.Code)
		return
	}
	def := card.Def()
	g.consumeDiscount(p, def)
	if def.Type == "ally" {
		p.AllyPlayedThisRound = true
	}
	// A fresh play resets pending event bonuses (Embiggen!/Shrink are
	// per-event).
	delete(g.EventDamageBonus, p.ID)
	delete(g.EventThreatBonus, p.ID)
	// Pay resources: discard chosen cards.
	for _, id := range m.Paid.CardIDs {
		if rc, ok := p.Hand.Remove(id); ok {
			p.Discard = append(p.Discard, rc)
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
		g.Allies[a.ID] = a
		p.Allies = append(p.Allies, a.ID)
		g.logf("%s plays %s", p.Name, def.Name)
		if b := behavior(def.Code); b.OnPlay != nil {
			g.Push(b.OnPlay(g, a)...)
		}
	case "support":
		s := &Support{ID: g.nextEntityID(KindSupport), Code: def.Code, Owner: p.ID}
		g.Supports[s.ID] = s
		p.Supports = append(p.Supports, s.ID)
		g.logf("%s plays %s", p.Name, def.Name)
		if b := behavior(def.Code); b.OnPlay != nil {
			g.Push(b.OnPlay(g, s)...)
		}
	case "upgrade":
		u := &Upgrade{ID: g.nextEntityID(KindUpgrade), Code: def.Code, Owner: p.ID}
		g.Upgrades[u.ID] = u
		p.Upgrades = append(p.Upgrades, u.ID)
		g.logf("%s plays %s", p.Name, def.Name)
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
			Crisis:     strings.Contains(def.Text, "Crisis") || strings.Contains(def.Text, "crisis"),
			Hazard:     def.Hazards,
		}
		g.SideSchemes[s.ID] = s
		g.logf("%s plays %s (threat %d)", p.Name, def.Name, s.Threat)
		if b := behavior(def.Code); b.OnPlay != nil {
			g.Push(b.OnPlay(g, s)...)
		}
	default:
		// Events and anything else: resolve then discard.
		g.logf("%s plays %s", p.Name, def.Name)
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
		g.logf("cannot play missing defense event %s", m.Card.Code)
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
	g.logf("%s plays %s", p.Name, def.Name)
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
		g.logf("cannot put missing ally %s into play", m.Card.Code)
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
	g.logf("%s puts %s into play", p.Name, def.Name)
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
			ID: "interrupt-" + hc.ID, Label: "Play " + hc.Def().Name,
			Kind: ChoicePlay, CardCode: hc.Code,
		}
		if cost > 0 && len(p.Hand) > cost {
			var pays []Choice
			for _, rc := range p.Hand {
				if rc.ID == hc.ID {
					continue
				}
				pays = append(pays, Choice{
					Label: "Discard " + rc.Def().Name, Kind: ChoiceCard, CardCode: rc.Code,
				}.Msgs(append([]Message{DiscardCards{Player: p.ID, Cards: CardList{rc}}}, final...)...))
			}
			interrupts = append(interrupts, choice.WithThen(
				Ask(fmt.Sprintf("Discard %d card(s) to pay for %s", cost, hc.Def().Name), pays...)))
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
			ID: "bw-cancel", Label: "Exhaust Black Widow → cancel " + m.Card.Def().Name,
			Kind: ChoiceAbility, SourceID: a.ID, CardCode: a.Code,
		}.Msgs(ExhaustEntity{ID: a.ID}))
	}
	if len(interrupts) == 0 {
		g.Push(TreacheryResolve{Player: m.Player, Card: m.Card})
		return
	}
	interrupts = append(interrupts, Choice{
		ID: "continue", Label: "Let " + m.Card.Def().Name + " resolve", Kind: ChoicePass,
	}.Msgs(TreacheryResolve{Player: m.Player, Card: m.Card}))
	g.Push(AskQuestion{Player: p.ID, Question: Ask(m.Card.Def().Name+" — interrupts?", interrupts...)})
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
		g.logf("%s cancels the effects of %s", p.Name, def.Name)
		return
	}
	t := &Treachery{ID: g.nextEntityID(KindTreachery), Code: def.Code}
	g.Treacheries[t.ID] = t
	if b := behavior(def.Code); b.ResolveTreachery != nil {
		g.Push(b.ResolveTreachery(g, t, p)...)
	} else {
		g.logf("(no effect implemented for %s)", def.Name)
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
		g.logf("cannot play missing Sense card %s", m.Card.Code)
		return
	}
	def := card.Def()
	u := &Upgrade{ID: g.nextEntityID(KindUpgrade), Code: def.Code, Owner: p.ID}
	g.Upgrades[u.ID] = u
	p.Upgrades = append(p.Upgrades, u.ID)
	g.logf("%s plays %s from the Sense deck", p.Name, def.Name)
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
		g.logf("cannot play missing discard ally %s", m.Card.Code)
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
	g.logf("%s plays %s from their discard pile", p.Name, def.Name)
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
	g.logf("%s reveals %s", p.Name, def.Name)

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
			g.logf("(no effect implemented for %s)", def.Name)
		}
		return
	}

	g.EncounterDiscard = append(g.EncounterDiscard, card)

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
			Crisis:    strings.Contains(def.Text, "Crisis") || strings.Contains(def.Text, "crisis"),
			Hazard:    def.Hazards,
		}
		g.SideSchemes[s.ID] = s
		g.logMajorf("Side scheme %s enters play (threat %d)", def.Name, s.Threat)
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
			g.logf("%s attaches to the villain", def.Name)
		}
	case "environment":
		env := &Environment{ID: g.nextEntityID(KindEnvironment), Code: def.Code}
		g.Environments[env.ID] = env
		g.logMajorf("Environment %s enters play", def.Name)

	default:
		g.logf("(unhandled encounter type %s for %s)", def.Type, def.Name)
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
	if v.Stage >= len(v.stageCodes) {
		if scen := g.Scenario(); scen.OnVillainDefeated != nil {
			g.Push(scen.OnVillainDefeated(g, v)...)
			return
		}
		g.Push(GameOver{Won: true, Reason: "The villain was defeated"})
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
	g.logf("%s advances to stage %s", def.Name, def.StageLabel)
	if b := behavior(v.Code); b.VillainStage != nil {
		g.Push(b.VillainStage(g, v, v.Stage)...)
	}
}

// boostSpawnsMinion reports the "Boost: put this card into play" rider.
func boostSpawnsMinion(def *data.CardDef) bool {
	if def.Type != "minion" && def.Type != "side_scheme" {
		return false
	}
	return strings.Contains(def.Text, "Boost: Put") || strings.Contains(def.Text, "Boost: put")
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
		g.logMajorf("Side scheme %s is defeated", s.EDef().Name)
		g.Delete(id)
		return
	}
	if g.MainScheme != nil && id == g.MainScheme.ID {
		scen := g.Scenario()
		if scen.OnMainSchemeDefeated != nil {
			g.Push(scen.OnMainSchemeDefeated(g, g.MainScheme)...)
		} else {
			g.logMajorf("Main scheme %s is defeated", g.MainScheme.EDef().Name)
		}
	}
}
