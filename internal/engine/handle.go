package engine

import (
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
		g.logf("── Round %d ──", g.Round)
		g.Push(BeginPhase{Phase: PhaseResource})

	case BeginPhase:
		g.handleBeginPhase(m.Phase)

	case EndPhase:
		switch m.Phase {
		case PhaseResource:
			g.Push(BeginPhase{Phase: PhasePlayer})
		case PhasePlayer:
			g.Push(BeginPhase{Phase: PhaseVillain})
		case PhaseVillain:
			g.Push(EndRound{})
		}

	case EndRound:
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
			if !q.EndedTurn {
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
				g.logf("%s's deck is empty; no card drawn", p.Name)
				break
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
			p.Discard = append(p.Discard, c)
		}

	case ChangeForm:
		p := g.Player(m.Player)
		if p == nil || p.FormChanged || p.Exhausted {
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
			if v := retaliateOf(e); v > 0 {
				g.Push(DamageEntity{Target: p.ID, Damage: v, Source: m.Target})
			}
		}

	case BasicRecover:
		p := g.Player(m.Player)
		if p == nil {
			return
		}
		p.Exhausted = true
		g.Push(HealEntity{Target: p.ID, N: p.RecoverStat()})

	case VillainActivates:
		g.handleVillainActivates(m)

	case MinionActivates:
		g.handleMinionActivates(m)

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
						g.logf("Boost card %s enters play!", def.Name)
						g.Push(RevealEncounterCard{Player: g.boostSpawnTarget(v), Card: c})
						continue
					}
					add := deref(def.Boost, 0)
					v.BoostCount += add
					g.logf("Boost card revealed: %s (+%d)", def.Name, add)
					v.RevealedBoosts = append(v.RevealedBoosts, c)
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
		g.Push(SchemeThreat{Scheme: g.MainScheme.ID, N: threat, Source: v.ID})
		g.Push(ClearBoosts{Enemy: v.ID})

	case VillainDefeated:
		v := g.Villains[m.VillainID]
		if v == nil {
			return
		}
		g.logf("%s is defeated!", v.EDef().Name)
		g.Push(AdvanceVillainStage{VillainID: v.ID})

	case AdvanceVillainStage:
		g.advanceVillainStage(m.VillainID)

	case MinionDefeated:
		if mn := g.Minions[m.MinionID]; mn != nil {
			g.logf("%s is defeated", mn.EDef().Name)
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
				g.logf("Main scheme advances to stage %d", next)
			}
		}

	case GameOver:
		g.Over = true
		g.Won = m.Won
		g.Reason = m.Reason
		g.pending = nil
		g.queue = nil
		if m.Won {
			g.logf("🏆 Victory: %s", m.Reason)
		} else {
			g.logf("💀 Defeat: %s", m.Reason)
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
			g.logf("Villain flips to %s", def.Name)
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

	case BoostEnemyAttack:
		switch e := g.Entity(m.Enemy).(type) {
		case *Villain:
			e.AttackVal += m.N
		case *Minion:
			e.AttackVal += m.N
		}

	case RevealNemesisSet:
		g.handleRevealNemesisSet(m.Player)

	case SpawnDrone:
		g.handleSpawnDrone(m.Player)

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
	g.logf("%s reveals their nemesis set!", p.Name)
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
			g.logf("%s enters play engaged with %s", def.Name, p.Name)
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
			g.logf("Side scheme %s enters play (threat %d)", def.Name, s.Threat)
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

// droneBonus returns +1 while Ultron stage III is in play.
func (g *Game) droneBonus() int {
	for _, v := range g.Villains {
		if v.Code == "01136" {
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
	g.logf("A Drone enters play engaged with %s", p.Name)
}

func (g *Game) handleStartGame() {
	for _, p := range g.Players {
		p.Deck = g.assignCardIDs(p.Deck, p.ID)
		p.ObligationDeck = g.assignCardIDs(p.ObligationDeck, p.ID)
		g.shuffle(&p.Deck)
		g.shuffle(&p.ObligationDeck)
		p.Hand = append(p.Hand, p.Deck[:min(len(p.Deck), p.HandSize(g))]...)
		p.Deck = p.Deck[len(p.Hand):]
	}
	g.EncounterDeck = g.assignCardIDs(g.EncounterDeck, "")
	g.shuffle(&g.EncounterDeck)
	scen := g.Scenario()
	if scen.Setup != nil {
		g.Push(scen.Setup(g)...)
	}
	g.logf("Scenario: %s", scen.Name)
	// Hero setup hooks run after opening hands are drawn.
	for _, p := range g.Players {
		if b := behavior(p.HeroCode); b.HeroSetup != nil {
			g.Push(b.HeroSetup(g, p)...)
		}
	}
	g.Push(BeginRound{})
}

func (g *Game) handleBeginPhase(phase Phase) {
	switch phase {
	case PhaseResource:
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
		for i, p := range g.Players {
			if p.FirstPlayer {
				g.TurnIndex = i
			}
		}
		g.Push(PlayerTurnStart{Player: g.Players[g.TurnIndex].ID})

	case PhaseVillain:
		g.UsedThisTurn = map[string]bool{}
		g.ActiveTurn = ""
		for i := range g.Players {
			p := g.Players[(g.TurnIndex+i)%len(g.Players)]
			for _, id := range sortedIDs(g.Villains) {
				g.Push(VillainActivates{VillainID: id, Player: p.ID})
			}
			// Minions activate against the first player once per phase.
			if i == 0 {
				first := g.Players[g.TurnIndex]
				for _, id := range sortedIDs(g.Minions) {
					g.Push(MinionActivates{MinionID: id, Player: first.ID})
				}
			}
		}
		for i := range g.Players {
			p := g.Players[(g.TurnIndex+i)%len(g.Players)]
			if card, ok := g.drawEncounter(); ok {
				p.EncounterDown = append(p.EncounterDown, card)
			}
		}
		for i := range g.Players {
			p := g.Players[(g.TurnIndex+i)%len(g.Players)]
			for len(p.EncounterDown) > 0 {
				card := p.EncounterDown[0]
				p.EncounterDown = p.EncounterDown[1:]
				g.Push(RevealEncounterCard{Player: p.ID, Card: card})
			}
		}
		g.Push(EndPhase{Phase: PhaseVillain})
	}
}

func (g *Game) handleVillainActivates(m VillainActivates) {
	v := g.Villains[m.VillainID]
	p := g.Player(m.Player)
	if v == nil || p == nil {
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
		g.Push(AskQuestion{Player: p.ID, Question: g.attackQuestion(v.ID, v.AttackVal, p, TriggerVillainAttacksYou)})
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
	if mn == nil || p == nil {
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
		g.Push(AskQuestion{Player: p.ID, Question: g.defenderQuestion(mn.ID, mn.AttackVal, p)})
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

	switch def := g.Entity(m.Defender).(type) {
	case *Player:
		damage := attack
		if !m.Undefended && def.IsHero() && !def.Exhausted {
			def.Exhausted = true
			damage -= def.DefenseStat()
			g.logf("%s defends (%d damage prevented)", def.Name, def.DefenseStat())
		}
		g.Push(DamageEntity{Target: def.ID, Damage: max(0, damage), Source: m.Against})
		// Retaliate on the defending identity.
		if rv := retaliateOf(def); rv > 0 {
			g.Push(DamageEntity{Target: m.Against, Damage: rv, Source: def.ID})
		}
	case *Ally:
		def.Exhausted = true
		damage := max(0, attack-def.Defense())
		g.Push(DamageEntity{Target: def.ID, Damage: damage, Source: m.Against})
		if rv := retaliateOf(def); rv > 0 {
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
	default:
		// Events and anything else: resolve then discard.
		g.logf("%s plays %s", p.Name, def.Name)
		ec := &EventCard{Code: def.Code, Owner: p.ID, Paid: m.Paid}
		if b := behavior(def.Code); b.OnPlay != nil {
			g.Push(b.OnPlay(g, ec)...)
		}
		p.Discard = append(p.Discard, card)
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

func (e *EventCard) EID() EntityID          { return EntityID("event") }
func (e *EventCard) ECode() string          { return e.Code }
func (e *EventCard) EDef() *data.CardDef    { return DB.MustLookup(e.Code) }
func (e *EventCard) EOwner() PlayerID       { return e.Owner }
func (e *EventCard) EExhausted() bool       { return false }
func (e *EventCard) React(g *Game, msg Message) []Message { return nil }

func (g *Game) revealEncounterCard(pid PlayerID, card Card) {
	p := g.Player(pid)
	def := card.Def()
	g.logf("%s reveals %s", p.Name, def.Name)
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
		g.logf("Side scheme %s enters play (threat %d)", def.Name, s.Threat)
		for i := 0; i < s.Hazard; i++ {
			if c, ok := g.drawEncounter(); ok {
				g.Push(RevealEncounterCard{Player: pid, Card: c})
			}
		}
	case "treachery":
		t := &Treachery{ID: g.nextEntityID(KindTreachery), Code: def.Code}
		g.Treacheries[t.ID] = t
		if b := behavior(def.Code); b.ResolveTreachery != nil {
			g.Push(b.ResolveTreachery(g, t, p)...)
		} else {
			g.logf("(no effect implemented for %s)", def.Name)
			g.Delete(t.ID)
		}
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
		g.logf("Environment %s enters play", def.Name)

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
		g.logf("Side scheme %s is defeated", s.EDef().Name)
		g.Delete(id)
		return
	}
	if g.MainScheme != nil && id == g.MainScheme.ID {
		scen := g.Scenario()
		if scen.OnMainSchemeDefeated != nil {
			g.Push(scen.OnMainSchemeDefeated(g, g.MainScheme)...)
		} else {
			g.logf("Main scheme %s is defeated", g.MainScheme.EDef().Name)
		}
	}
}
