package engine

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// DB is the process-wide card database.
var DB = data.MustLoad()

// Game is the authoritative game state. Everything needed to persist and
// resume a game is JSON-serializable: the message queue and pending
// questions carry only data, and randomness is a pure function of (Seed,
// Counter). Card behaviors live in the registry and are re-derived from
// card codes.
type Game struct {
	Seed       int64  `json:"seed"`
	Counter    uint64 `json:"counter"` // RNG draws consumed
	ScenarioID string `json:"scenarioId"`
	Difficulty string `json:"difficulty"` // standard | expert
	Round      int    `json:"round"`
	// Phase is the current round phase (phase-dependent reactions such as
	// Change of Fortune).
	Phase Phase `json:"phase,omitempty"`

	Players []*Player `json:"players"`

	Villains     map[EntityID]*Villain     `json:"villains"`
	Minions      map[EntityID]*Minion      `json:"minions"`
	Allies       map[EntityID]*Ally        `json:"allies"`
	Supports     map[EntityID]*Support     `json:"supports"`
	Upgrades     map[EntityID]*Upgrade     `json:"upgrades"`
	Attachments  map[EntityID]*Attachment  `json:"attachments"`
	Treacheries  map[EntityID]*Treachery   `json:"treacheries"`
	SideSchemes  map[EntityID]*SideScheme  `json:"sideSchemes"`
	Environments map[EntityID]*Environment `json:"environments"`

	MainScheme *MainScheme `json:"mainScheme"`

	EncounterDeck    CardList `json:"encounterDeck"`
	EncounterDiscard CardList `json:"encounterDiscard"`

	// Collection is the Collector scenario's game area: cards removed
	// from the game faceup instead of reaching their discard piles.
	Collection CardList `json:"collection,omitempty"`
	// SetAside holds cards removed from the encounter deck by scenario
	// setup (The Missing Milano stashes Ship Command, the Sinister Six's
	// bench).
	SetAside CardList `json:"setAside,omitempty"`
	// GliderCounter marks which scheme currently holds the Venom
	// Goblin's glider (approximation of the three-Manhattan board).
	GliderCounter EntityID `json:"gliderCounter,omitempty"`
	// VictoryDisplay holds defeated cards with printed victory points;
	// several mts villains scale off it.
	VictoryDisplay CardList `json:"victoryDisplay,omitempty"`

	// queue holds pending messages.
	queue []Message

	// pending is the question blocking the game, if any.
	pending *PendingQuestion

	// ActiveTurn is the player currently taking their player-phase turn.
	ActiveTurn PlayerID `json:"activeTurn,omitempty"`
	// MutantBombCounters tracks Boom Bang bomb counters in play (shared
	// pool approximation).
	MutantBombCounters int `json:"mutantBombs,omitempty"`
	// ActiveVillain holds the active-counter villain in multi-villain
	// scenarios (Wrecking Crew); empty otherwise.
	ActiveVillain EntityID `json:"activeVillain,omitempty"`
	TurnIndex     int      `json:"turnIndex"` // index into Players for player-phase order

	UsedThisRound map[string]bool `json:"usedThisRound"`
	UsedThisTurn  map[string]bool `json:"usedThisTurn"`

	// EventDamageBonus / EventThreatBonus hold pending per-player bonuses
	// for the event currently resolving (Embiggen!, Shrink); consumed by
	// the first matching application.
	EventDamageBonus map[PlayerID]int `json:"eventDamageBonus,omitempty"`
	EventThreatBonus map[PlayerID]int `json:"eventThreatBonus,omitempty"`

	nextID int
	Over   bool   `json:"over"`
	Won    bool   `json:"won"`
	Reason string `json:"reason,omitempty"`

	Log LogEntries `json:"log"`

	// Transient: presentation events drained after each answer (events.go).
	events []Evt
	// Transient: scenario def hooks (rebuilt from ScenarioID).
	scenario *ScenarioDef
}

// PendingQuestion blocks the game until answered.
type PendingQuestion struct {
	Player   PlayerID  `json:"player"`
	Question *Question `json:"question"`
}

// Random draws a deterministic value in [0,n).
func (g *Game) Random(n int) int {
	r := rand.New(rand.NewPCG(uint64(g.Seed), g.Counter))
	g.Counter++
	return r.IntN(n)
}

func (g *Game) nextEntityID(kind string) EntityID {
	g.nextID++
	return NewEntityID(kind, g.nextID)
}

func (g *Game) nextCardID() string {
	g.nextID++
	return fmt.Sprintf("card-%d", g.nextID)
}

// SpawnSupport puts a support into play under owner (scenario setup).
func (g *Game) SpawnSupport(code string, owner PlayerID) *Support {
	s := &Support{ID: g.nextEntityID(KindSupport), Code: code, Owner: owner}
	g.Supports[s.ID] = s
	g.logMajorf("%s enters play under %s's control", s.EDef().Name, g.Player(owner).Name)
	return s
}

// SpawnVillainFromCard brings a villain into play from its base card
// code (Sinister Six ambushes from the set-aside area).
func (g *Game) SpawnVillainFromCard(base string) *Villain {
	stages := VillainStageCodes(base)
	if len(stages) == 0 {
		return nil
	}
	return g.spawnVillain(stages, 1)
}

// SpawnAttachment brings an encounter attachment into play targeting the
// given entity (running its OnAttach preference when target is empty).
func (g *Game) SpawnAttachment(code string, target EntityID) *Attachment {
	t := &Attachment{ID: g.nextEntityID(KindAttachment), Code: code}
	g.Attachments[t.ID] = t
	if target != "" {
		t.Target = target
		g.Logf("%s attaches to %s", t.EDef().Name, g.Entity(target).EDef().Name)
		if b := behavior(code); b.OnAttach != nil {
			g.Push(b.OnAttach(g, t, target)...)
		}
		return t
	}
	if b := behavior(code); b.OnAttach != nil {
		g.Push(b.OnAttach(g, t, EntityID(""))...)
	} else {
		for id := range g.Villains {
			t.Target = id
			break
		}
	}
	return t
}

// SpawnEnvironment brings a scenario environment into play.
func (g *Game) SpawnEnvironment(code string) *Environment {
	e := &Environment{ID: g.nextEntityID(KindEnvironment), Code: code}
	g.Environments[e.ID] = e
	g.logMajorf("Environment %s enters play", e.EDef().Name)
	return e
}

// cardLeavesPlay routes a card leaving play to its owner's discard pile —
// or into The Collection while the Collector's forced interrupt is active
// (stage III additionally places 1 threat on the main scheme).
func (g *Game) cardLeavesPlay(p *Player, code, name string) {
	for _, v := range g.Villains {
		if v == nil {
			continue
		}
		switch data.BaseCode(v.Code) {
		case "16070", "16071", "16072":
			g.Collection = append(g.Collection, Card{ID: g.nextCardID(), Code: code, Owner: p.ID})
			g.Logf("%s is placed into The Collection instead of the discard", name)
			if data.BaseCode(v.Code) == "16072" && g.MainScheme != nil {
				g.Push(SchemeThreat{Scheme: g.MainScheme.ID, N: 1, Source: v.ID})
			}
			return
		}
	}
	if p != nil {
		p.Discard = append(p.Discard, Card{ID: g.nextCardID(), Code: code, Owner: p.ID})
	}
}

// Scenario returns the scenario def (with hooks) for this game.
func (g *Game) Scenario() *ScenarioDef {
	if g.scenario == nil {
		def, ok := LookupScenario(g.ScenarioID)
		if !ok {
			return &ScenarioDef{Name: g.ScenarioID}
		}
		g.scenario = def
	}
	return g.scenario
}

// Player returns the player by id.
func (g *Game) Player(id PlayerID) *Player {
	for _, p := range g.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// Entity finds any entity by id.
func (g *Game) Entity(id EntityID) Entity {
	switch id.Kind() {
	case KindPlayer:
		if p := g.Player(id); p != nil {
			return p
		}
	case KindVillain:
		return g.Villains[id]
	case KindMinion:
		return g.Minions[id]
	case KindAlly:
		return g.Allies[id]
	case KindSupport:
		return g.Supports[id]
	case KindUpgrade:
		return g.Upgrades[id]
	case KindAttachment:
		return g.Attachments[id]
	case KindTreachery:
		return g.Treacheries[id]
	case KindSideScheme:
		return g.SideSchemes[id]
	case KindMainScheme:
		return g.MainScheme
	case KindEnvironment:
		return g.Environments[id]
	}
	return nil
}

// Delete removes an entity from play.
func (g *Game) Delete(id EntityID) {
	switch id.Kind() {
	case KindMinion:
		delete(g.Minions, id)
	case KindAlly:
		if a, ok := g.Allies[id]; ok {
			if p := g.Player(a.Owner); p != nil {
				p.Allies = removeID(p.Allies, id)
			}
		}
		delete(g.Allies, id)
	case KindSupport:
		if s, ok := g.Supports[id]; ok {
			if p := g.Player(s.Owner); p != nil {
				p.Supports = removeID(p.Supports, id)
			}
		}
		delete(g.Supports, id)
	case KindUpgrade:
		if u, ok := g.Upgrades[id]; ok {
			if p := g.Player(u.Owner); p != nil {
				p.Upgrades = removeID(p.Upgrades, id)
			}
		}
		delete(g.Upgrades, id)
	case KindAttachment:
		delete(g.Attachments, id)
	case KindTreachery:
		delete(g.Treacheries, id)
	case KindSideScheme:
		delete(g.SideSchemes, id)
	case KindEnvironment:
		delete(g.Environments, id)
	}
}

func removeID(ids []EntityID, id EntityID) []EntityID {
	for i, x := range ids {
		if x == id {
			return append(ids[:i], ids[i+1:]...)
		}
	}
	return ids
}

// Enemies returns villain and minion ids.
func (g *Game) Enemies() []EntityID {
	var out []EntityID
	for id := range g.Villains {
		out = append(out, id)
	}
	for id := range g.Minions {
		out = append(out, id)
	}
	return out
}

// Schemes returns main + side scheme ids.
func (g *Game) Schemes() []EntityID {
	var out []EntityID
	if g.MainScheme != nil {
		out = append(out, g.MainScheme.ID)
	}
	for id := range g.SideSchemes {
		out = append(out, id)
	}
	return out
}

// AllEntities returns every reacting entity in a deterministic order
// (players, villains, then other maps sorted by id).
func (g *Game) AllEntities() []Entity {
	out := []Entity{}
	for _, p := range g.Players {
		out = append(out, p)
	}
	for _, id := range sortedIDs(g.Villains) {
		out = append(out, g.Villains[id])
	}
	for _, id := range sortedIDs(g.Minions) {
		out = append(out, g.Minions[id])
	}
	for _, id := range sortedIDs(g.Allies) {
		out = append(out, g.Allies[id])
	}
	for _, id := range sortedIDs(g.Supports) {
		out = append(out, g.Supports[id])
	}
	for _, id := range sortedIDs(g.Upgrades) {
		out = append(out, g.Upgrades[id])
	}
	for _, id := range sortedIDs(g.Attachments) {
		out = append(out, g.Attachments[id])
	}
	for _, id := range sortedIDs(g.Treacheries) {
		out = append(out, g.Treacheries[id])
	}
	for _, id := range sortedIDs(g.SideSchemes) {
		out = append(out, g.SideSchemes[id])
	}
	for _, id := range sortedIDs(g.Environments) {
		out = append(out, g.Environments[id])
	}
	if g.MainScheme != nil {
		out = append(out, g.MainScheme)
	}
	return out
}

func sortedIDs[T any](m map[EntityID]T) []EntityID {
	ids := make([]EntityID, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	// numeric sort for stability
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j].Num() < ids[j-1].Num(); j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
	return ids
}

// ---------------------------------------------------------------- loop

// Push appends messages to the queue.
func (g *Game) Push(msgs ...Message) { g.queue = append(g.queue, msgs...) }

// PushFront enqueues messages at the front of the queue, so they resolve
// right after the current pending question instead of after everything
// already queued.
func (g *Game) PushFront(msgs ...Message) { g.queue = append(msgs, g.queue...) }

// Run processes messages until the queue drains or a question blocks.
func (g *Game) Run() {
	for !g.Over {
		if g.pending != nil {
			return
		}
		if len(g.queue) == 0 {
			g.checkContinue()
			if g.pending != nil || len(g.queue) == 0 {
				return
			}
			continue
		}
		msg := g.queue[0]
		g.queue = g.queue[1:]
		g.process(msg)
	}
}

// checkContinue restarts flows that are driven by engine state rather than
// queued messages (the active player's turn menu).
func (g *Game) checkContinue() {
	if g.ActiveTurn != "" && !g.Over {
		if p := g.Player(g.ActiveTurn); p != nil && !p.EndedTurn {
			g.Push(AskQuestion{Player: p.ID, Question: g.TurnMenu(p)})
		}
	}
}

// process fans one message out to every entity and the core handler.
// Handler output and reactions are inserted at the FRONT of the queue
// (reactions first), giving depth-first flow semantics: a villain
// activation's boost/damage steps always resolve before the phase
// driver's later steps.
func (g *Game) process(msg Message) {
	var reactions []Message
	for _, e := range g.AllEntities() {
		reactions = append(reactions, e.React(g, msg)...)
	}
	nBefore := len(g.queue)
	g.handle(msg)
	if g.Over || len(g.queue) < nBefore {
		// handler cleared the queue (GameOver)
		return
	}
	handlerOut := g.queue[nBefore:]
	rest := g.queue[:nBefore:nBefore]
	out := make([]Message, 0, len(reactions)+len(handlerOut)+len(rest))
	out = append(out, reactions...)
	out = append(out, handlerOut...)
	out = append(out, rest...)
	g.queue = out
}

// PendingQuestion exposes the blocking question, if any.
func (g *Game) Pending() *PendingQuestion { return g.pending }

// Answer resolves the pending question with the given choice paths and
// resumes the game.
func (g *Game) Answer(playerID PlayerID, paths []string) error {
	if g.pending == nil {
		return fmt.Errorf("no pending question")
	}
	// Pending questions persisted before nested choice ids carried path
	// prefixes answer incorrectly (paths resolve to the wrong root choice).
	// Rebuild the turn menu first — no-op for current-format questions.
	g.RebuildTurnMenu()
	if g.pending.Player != playerID {
		return fmt.Errorf("question is for player %s", g.pending.Player)
	}
	q := g.pending.Question
	pending := g.pending
	g.pending = nil

	var msgs []Message
	var err error
	switch q.Type {
	case "choose_n":
		msgs, err = g.resolveChooseN(q, paths)
		if err != nil {
			g.pending = pending
			return err
		}
	default:
		// A choose_n subtree (resource payment) nested under this choose_one
		// question is answered by submitting all its selections at once
		// ({"6.0","6.1"} = the payment question under choice "6"). Choices
		// along the common prefix (e.g. an interrupt ahead of a paid
		// defense) contribute their messages as well.
		if sub, subPaths, prefix, ok := nestedChooseN(q, paths); ok {
			msgs, err = g.resolveChooseN(sub, subPaths)
			if err != nil {
				g.pending = pending
				return err
			}
			prefixMsgs, err := g.chainMsgs(q, prefix)
			if err != nil {
				g.pending = pending
				return err
			}
			msgs = append(prefixMsgs, msgs...)
			break
		}
		if len(paths) != 1 {
			g.pending = pending
			return fmt.Errorf("expected exactly one answer path")
		}
		msgs, err = g.chainMsgs(q, paths[0])
		if err != nil {
			g.pending = pending
			return err
		}
	}
	// Answers continue the interrupted flow: front-insert.
	g.queue = append(msgs, g.queue...)
	g.Run()
	return nil
}

// resolveChooseN turns a choose_n answer into effect messages, applying the
// question's validation rule (payment:N) when present.
func (g *Game) resolveChooseN(q *Question, paths []string) ([]Message, error) {
	choices, err := q.Selective(paths)
	if err != nil {
		return nil, err
	}
	if q.Validate != "" {
		return g.validateSelection(q, choices)
	}
	var msgs []Message
	for _, c := range choices {
		msgs = append(msgs, c.msgs...)
	}
	return msgs, nil
}

// chainMsgs collects the messages of every choice along an answer path,
// root first and leaf last ("interrupt → defend" fires the interrupt's
// effect before the defense). Questions persisted before WithThen copied
// subtrees carry duplicate ids across branches; answering those keeps the
// legacy leaf-only semantics instead of risking another branch's messages.
func (g *Game) chainMsgs(q *Question, path string) ([]Message, error) {
	if !q.idsUnique() {
		leaf, err := q.Leaf(path)
		if err != nil {
			return nil, err
		}
		return leaf.msgs, nil
	}
	chain, err := q.Chain(path)
	if err != nil {
		return nil, err
	}
	var msgs []Message
	for _, c := range chain {
		msgs = append(msgs, c.msgs...)
	}
	return msgs, nil
}

// nestedChooseN detects an answer selecting from a choose_n subtree nested
// under a choose_one root: the paths' common prefix must resolve to a choice
// whose Then question is a choose_n ("Pay N resources…"). Returns that
// subtree, the original paths (sub-choice ids carry the full prefix) and
// the common prefix (whose chain messages fire alongside the payment).
func nestedChooseN(q *Question, paths []string) (*Question, []string, string, bool) {
	prefix := paths[0]
	for _, p := range paths[1:] {
		for prefix != "" && p != prefix && !strings.HasPrefix(p, prefix+".") {
			prefix = trimLastSegment(prefix)
		}
	}
	for prefix != "" {
		if c, err := q.Leaf(prefix); err == nil && c.Then != nil &&
			c.Then.Type == "choose_n" && len(c.Then.Choices) > 0 {
			return c.Then, paths, prefix, true
		}
		prefix = trimLastSegment(prefix)
	}
	return nil, nil, "", false
}

func trimLastSegment(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[:i]
	}
	return ""
}

// validateSelection applies server-side validation rules to choose_n
// answers, converting payment stubs into their real effect messages.
func (g *Game) validateSelection(q *Question, choices []*Choice) ([]Message, error) {
	switch {
	case strings.HasPrefix(q.Validate, "discardDown:"):
		// End-of-player-phase discard: at least N selected cards must be
		// discarded (hand over hand size).
		var need int
		fmt.Sscanf(q.Validate, "discardDown:%d", &need)
		var msgs []Message
		n := 0
		for _, c := range choices {
			msgs = append(msgs, c.msgs...)
			for _, msg := range c.msgs {
				if _, ok := msg.(DiscardCards); ok {
					n++
				}
			}
		}
		if n < need {
			return nil, fmt.Errorf("must discard at least %d card(s), selected %d", need, n)
		}
		return msgs, nil
	case strings.HasPrefix(q.Validate, "payment:"):
		var cost int
		fmt.Sscanf(q.Validate, "payment:%d", &cost)
		playerID := PlayerID(fmt.Sprint(q.Context["player"]))
		p := g.Player(playerID)
		if p == nil {
			return nil, fmt.Errorf("unknown player in payment context")
		}
		// The card being paid for (absent for ability payments).
		var targetDef *data.CardDef
		var targetCard Card
		if cardID, ok := q.Context["cardId"].(string); ok {
			card, found := p.Hand.Find(cardID)
			if !found {
				return nil, fmt.Errorf("card no longer in hand")
			}
			targetDef = card.Def()
			targetCard = card
		}
		total := 0
		var paidIDs []string
		var icons []string
		seenAbility := map[EntityID]bool{}
		var exhausts []Message
		for _, c := range choices {
			for _, msg := range c.msgs {
				switch st := msg.(type) {
				case ResourcePayStub:
					total += iconCount(st.Card.Def()) + powerOfBonus(st.Card.Def(), targetDef)
					paidIDs = append(paidIDs, st.Card.ID)
					icons = append(icons, st.Card.Def().Resources...)
				case AbilityPayStub:
					if seenAbility[st.Source] {
						continue
					}
					seenAbility[st.Source] = true
					total++
					icons = append(icons, st.Icon)
					src := g.Entity(st.Source)
					if src != nil {
						if ra := behavior(src.ECode()).Resource; ra != nil {
							if !ra.NoExhaust {
								exhausts = append(exhausts, ExhaustEntity{ID: st.Source})
							}
							if ra.UsesCounters {
								exhausts = append(exhausts, AddEntityCounter{ID: st.Source, N: -1})
							}
							// Clarity of Purpose: the resource costs 1
							// damage on the attached character.
							if ra.DamageAttached > 0 {
								if u, ok := src.(*Upgrade); ok && u.AttachTo != "" {
									exhausts = append(exhausts, DamageEntity{Target: u.AttachTo, Damage: ra.DamageAttached, Source: u.Owner})
								}
							}
						}
					}
				}
			}
		}
		if total < cost {
			return nil, fmt.Errorf("need %d resource icons, selected %d", cost, total)
		}
		if iconSpec, ok := q.Context["abilityIcons"].(string); ok && iconSpec != "" {
			if err := checkIconRequirements(icons, iconSpec); err != nil {
				return nil, err
			}
		}
		paid := CostPaid{CardIDs: paidIDs, Icons: icons}
		if invID, ok := q.Context["invocationCard"].(string); ok {
			card, found := p.SenseDeck.Find(invID)
			if !found {
				return nil, fmt.Errorf("invocation card no longer on top")
			}
			var paidCards []Card
			for _, id := range paidIDs {
				if c, ok := p.Hand.Find(id); ok {
					paidCards = append(paidCards, c)
				}
			}
			out := exhausts
			if len(paidCards) > 0 {
				out = append(out, ResourcePay{Player: playerID, Cards: paidCards})
			}
			out = append(out, InvokeSpecial{
				Player:      playerID,
				Card:        card,
				ReturnToTop: q.Context["returnToTop"] == true,
			})
			return out, nil
		}
		if senseID, ok := q.Context["senseCard"].(string); ok {
			card, found := p.SenseDeck.Find(senseID)
			if !found {
				return nil, fmt.Errorf("sense card no longer in the sense deck")
			}
			var paidCards []Card
			for _, id := range paidIDs {
				if c, ok := p.Hand.Find(id); ok {
					paidCards = append(paidCards, c)
				}
			}
			out := exhausts
			if len(paidCards) > 0 {
				out = append(out, ResourcePay{Player: playerID, Cards: paidCards})
			}
			out = append(out, SenseEnterPlay{Player: playerID, Card: card})
			return out, nil
		}
		if discardID, ok := q.Context["playDiscard"].(string); ok {
			card, found := p.Discard.Find(discardID)
			if !found {
				return nil, fmt.Errorf("card no longer in discard pile")
			}
			out := append(exhausts, PlayDiscardAlly{Player: playerID, Card: card, Paid: paid})
			return out, nil
		}
		if saveID, ok := q.Context["saveAlly"].(string); ok {
			// Generic "pay to save an ally" flow (Red Dagger): pay,
			// return the ally to hand, optionally hit an enemy.
			var paidCards []Card
			for _, id := range paidIDs {
				if c, ok := p.Hand.Find(id); ok {
					paidCards = append(paidCards, c)
				}
			}
			out := exhausts
			if len(paidCards) > 0 {
				out = append(out, ResourcePay{Player: playerID, Cards: paidCards})
			}
			out = append(out, ReturnControlled{Player: playerID, ID: EntityID(saveID)})
			dmg := ctxInt(q.Context["saveDamage"])
			if dmg > 0 && len(g.Enemies()) > 0 {
				var picks []Choice
				for _, eid := range sortedIDs(g.Villains) {
					v := g.Villains[eid]
					picks = append(picks, Choice{
						Label: fmt.Sprintf("%s — %d/%d HP", v.EDef().Name, v.HP(), v.MaxHP),
						Kind:  ChoiceTarget, SourceID: eid, CardCode: v.Code,
					}.Msgs(DamageEntity{Target: eid, Damage: dmg, Source: playerID}))
				}
				for _, eid := range sortedIDs(g.Minions) {
					mn := g.Minions[eid]
					picks = append(picks, Choice{
						Label: fmt.Sprintf("%s — %d/%d HP", mn.EDef().Name, mn.HP(), mn.MaxHP),
						Kind:  ChoiceTarget, SourceID: eid, CardCode: mn.Code,
					}.Msgs(DamageEntity{Target: eid, Damage: dmg, Source: playerID}))
				}
				out = append(out, AskQuestion{Player: playerID, Question: Ask("Choose an enemy", picks...)})
			}
			return out, nil
		}
		if fromID, ok := q.Context["makeCallFrom"].(string); ok {
			cardID, _ := q.Context["makeCallCard"].(string)
			var paidCards []Card
			for _, id := range paidIDs {
				if c, ok := p.Hand.Find(id); ok {
					paidCards = append(paidCards, c)
				}
			}
			out := exhausts
			if len(paidCards) > 0 {
				out = append(out, ResourcePay{Player: playerID, Cards: paidCards})
			}
			out = append(out, AllyEntersPlayFree{
				Player:    playerID,
				Card:      Card{ID: cardID},
				FromOwner: PlayerID(fromID),
			})
			return out, nil
		}
		if against, ok := q.Context["defenseAgainst"].(string); ok && targetDef != nil {
			out := append(exhausts, PlayDefenseEvent{
				Player: playerID, Card: targetCard, Paid: paid, Against: EntityID(against),
			})
			return out, nil
		}
		if targetDef != nil {
			out := append(exhausts, PlayCard{Player: playerID, Card: targetCard, Paid: paid})
			return out, nil
		}
		if srcID, ok := q.Context["abilitySource"].(string); ok {
			src := g.Entity(EntityID(srcID))
			idx := ctxInt(q.Context["abilityIndex"])
			if src == nil {
				return nil, fmt.Errorf("ability source gone")
			}
			var abilities []Ability
			if srcID == string(playerID) {
				if b := behavior(p.HeroCode); b.HeroAbilities != nil {
					abilities = b.HeroAbilities(g, p)
				}
			} else if hb := behavior(src.ECode()); hb.Abilities != nil {
				abilities = hb.Abilities(g, src)
			}
			if idx < 0 || idx >= len(abilities) {
				return nil, fmt.Errorf("ability index out of range")
			}
			ab := abilities[idx]
			var msgs []Message
			msgs = append(msgs, exhausts...)
			if ab.Exhaust {
				msgs = append(msgs, ExhaustEntity{ID: src.EID()})
			}
			if ab.Execute != nil {
				msgs = append(msgs, ab.Execute(g, src.EID())...)
			}
			msgs = append(msgs, RunAbility{Player: playerID, Source: src.EID(), Index: idx})
			return msgs, nil
		}
		return nil, fmt.Errorf("payment context missing target")
	}
	for _, c := range choices {
		_ = c
	}
	return nil, fmt.Errorf("unknown validation %q", q.Validate)
}

// iconCount returns the number of resource icons a card contributes when
// discarded for payment.
func iconCount(def *data.CardDef) int {
	return len(def.Resources)
}

// powerOfBonus returns the extra icon a "The Power of <Aspect>" resource
// card contributes when paying for a card of that aspect (data-driven:
// parsed from the card name).
func powerOfBonus(paying, target *data.CardDef) int {
	if target == nil || !strings.HasPrefix(paying.Name, "The Power of ") {
		return 0
	}
	aspect := strings.ToLower(strings.TrimPrefix(paying.Name, "The Power of "))
	if target.Aspect == aspect {
		return 1
	}
	return 0
}

// costFor returns the payable cost for a card after the identity's passive
// discounts and one pending CostDiscount.
func (g *Game) costFor(p *Player, def *data.CardDef) int {
	cost := deref(def.Cost, 0)
	if cost <= 0 {
		return 0
	}
	if b := behavior(p.HeroCode); b.CardCost != nil {
		cost -= b.CardCost(g, p, def)
	}
	// Self-referential discounts on the played card itself (Web of Life
	// and Destiny: free for Web-Warrior identities).
	if b := behavior(def.Code); b.CardCost != nil {
		cost -= b.CardCost(g, p, def)
	}
	// Discounts from controlled allies (Iron Man 09039: upgrades cost 1
	// less while he is in play).
	for _, id := range p.Allies {
		a := g.Allies[id]
		if a == nil {
			continue
		}
		if b := behavior(a.Code); b.CardCost != nil {
			cost -= b.CardCost(g, p, def)
		}
	}
	for _, d := range p.CostDiscounts {
		if discountMatches(d, def) && d.Amount > 0 {
			cost -= d.Amount
			break
		}
	}
	return max(0, cost)
}

// consumeDiscount removes the pending discount that applied to a card
// being played.
func (g *Game) consumeDiscount(p *Player, def *data.CardDef) {
	if deref(def.Cost, 0) <= 0 {
		return
	}
	for i, d := range p.CostDiscounts {
		if discountMatches(d, def) && d.Amount > 0 {
			g.logMinorf("%s costs %d less (%s)", def.Name, d.Amount, p.Name)
			p.CostDiscounts = append(p.CostDiscounts[:i], p.CostDiscounts[i+1:]...)
			return
		}
	}
}

func discountMatches(d CostDiscount, def *data.CardDef) bool {
	if d.Type != "" && d.Type != def.Type {
		return false
	}
	if d.Trait != "" && !def.HasTrait(d.Trait) {
		return false
	}
	return true
}

// ctxInt coerces a JSON context value (float64) or in-memory int.
// checkIconRequirements verifies paid resources against icon-specific cost
// constraints ("physical:3" / "energy:1 mental:1"); each wild resource
// counts as one resource of any single type.
func checkIconRequirements(icons []string, spec string) error {
	wilds := 0
	pool := map[string]int{}
	for _, ic := range icons {
		if ic == "wild" {
			wilds++
			continue
		}
		pool[ic]++
	}
	for _, part := range strings.Fields(spec) {
		var icon string
		var n int
		if _, err := fmt.Sscanf(part, "%[^:]:%d", &icon, &n); err != nil || n <= 0 {
			return fmt.Errorf("bad icon requirement %q", part)
		}
		use := min(n, pool[icon])
		pool[icon] -= use
		if n-use > wilds {
			return fmt.Errorf("need %d [%s] resources among the payment", n-use+use, icon)
		}
		wilds -= n - use
	}
	return nil
}

func ctxInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case float64:
		return int(x)
	case int64:
		return int(x)
	}
	return -1
}

// Clone deep-copies the game via JSON round-trip (for undo snapshots).
func (g *Game) Clone() *Game {
	raw, err := g.MarshalJSON()
	if err != nil {
		panic(err)
	}
	clone := &Game{}
	if err := clone.UnmarshalJSON(raw); err != nil {
		panic(err)
	}
	return clone
}

// MarshalJSON serializes the game including queue and pending question;
// message payloads ride the envelope codec.
func (g *Game) MarshalJSON() ([]byte, error) {
	type alias Game
	out := struct {
		*alias
		Queue   []msgEnvelope    `json:"queue,omitempty"`
		Pending *PendingQuestion `json:"pending,omitempty"`
	}{alias: (*alias)(g), Pending: g.pending}
	var err error
	out.Queue, err = marshalMessages(g.queue)
	if err != nil {
		return nil, err
	}
	return jsonMarshal(out)
}

// UnmarshalJSON restores a game; scenario hooks re-attach via Scenario().
func (g *Game) UnmarshalJSON(b []byte) error {
	type alias Game
	in := struct {
		*alias
		Queue   []msgEnvelope    `json:"queue"`
		Pending *PendingQuestion `json:"pending"`
	}{alias: (*alias)(g)}
	if err := jsonUnmarshal(b, &in); err != nil {
		return err
	}
	queue, err := unmarshalMessages(in.Queue)
	if err != nil {
		return fmt.Errorf("restore queue: %w", err)
	}
	g.queue = queue
	g.pending = in.Pending
	g.scenario = nil
	g.migrateMainSchemeCodes()
	return nil
}

// migrateMainSchemeCodes re-derives the main scheme's stage codes from the
// current scenario registration. Games persisted before the b-face
// convention switch stored base codes ("01097") or a-face codes ("56063a")
// — the Drang scenario even pointed at a treachery — while the image layer
// and zh names now key by the registered b-face codes. A scheme caught
// mid-reveal (its FlipMainScheme still queued) keeps its a face.
func (g *Game) migrateMainSchemeCodes() {
	if g.MainScheme == nil {
		return
	}
	stages := g.Scenario().MainSchemeStages
	if len(stages) == 0 {
		return
	}
	revealing := false
	for _, m := range g.queue {
		if f, ok := m.(FlipMainScheme); ok && f.Scheme == g.MainScheme.ID {
			revealing = true
			break
		}
	}
	g.MainScheme.StageCodes = append([]string(nil), stages...)
	if !revealing && g.MainScheme.Stage >= 1 && g.MainScheme.Stage <= len(stages) {
		g.MainScheme.Code = stages[g.MainScheme.Stage-1]
	}
}

// NewGameOptions configures game creation.
type NewGameOptions struct {
	Seed       int64
	ScenarioID string
	Difficulty string
	// Players: identity hero base code (e.g. "01001") + deck card codes
	// with counts.
	Players []PlayerSpec
}

type PlayerSpec struct {
	Name     string
	UserID   string
	HeroBase string // e.g. "01001"
	Deck     map[string]int
}

// NewGame constructs a game at the moment before setup resolves.
func NewGame(opts NewGameOptions) (*Game, error) {
	scen, ok := LookupScenario(opts.ScenarioID)
	if !ok {
		return nil, fmt.Errorf("unknown scenario %q", opts.ScenarioID)
	}
	g := &Game{
		Seed:             opts.Seed,
		ScenarioID:       opts.ScenarioID,
		Difficulty:       opts.Difficulty,
		Villains:         map[EntityID]*Villain{},
		Minions:          map[EntityID]*Minion{},
		Allies:           map[EntityID]*Ally{},
		Supports:         map[EntityID]*Support{},
		Upgrades:         map[EntityID]*Upgrade{},
		Attachments:      map[EntityID]*Attachment{},
		Treacheries:      map[EntityID]*Treachery{},
		SideSchemes:      map[EntityID]*SideScheme{},
		Environments:     map[EntityID]*Environment{},
		UsedThisRound:    map[string]bool{},
		UsedThisTurn:     map[string]bool{},
		EventDamageBonus: map[PlayerID]int{},
		EventThreatBonus: map[PlayerID]int{},
		scenario:         scen,
	}
	if g.Difficulty == "" {
		g.Difficulty = "standard"
	}

	for i, spec := range opts.Players {
		p, err := g.newPlayer(spec, i)
		if err != nil {
			return nil, err
		}
		g.Players = append(g.Players, p)
	}
	// First player is picked during StartGame (group decision modelled
	// with the seeded RNG).
	g.setupScenario(scen)
	g.Push(StartGame{})
	g.Run()
	return g, nil
}

func (g *Game) newPlayer(spec PlayerSpec, i int) (*Player, error) {
	heroDef, ok := DB.Lookup(data.HeroSideCode(spec.HeroBase))
	if !ok {
		return nil, fmt.Errorf("unknown hero %q", spec.HeroBase)
	}
	p := &Player{
		ID:           NewEntityID(KindPlayer, i+1),
		Name:         spec.Name,
		UserID:       spec.UserID,
		HeroCode:     heroDef.Code,
		AlterEgoCode: data.AlterEgoSideCode(spec.HeroBase),
		Side:         SideAlterEgo,
		MaxHP:        deref(heroDef.HP, 10),
	}
	if p.Name == "" {
		p.Name = heroDef.Name
	}
	// Deterministic deck assembly: sort card codes so identical specs build
	// identical decks (map iteration order is randomized).
	codes := make([]string, 0, len(spec.Deck))
	for code := range spec.Deck {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		qty := spec.Deck[code]
		def, ok := DB.Lookup(code)
		if !ok {
			return nil, fmt.Errorf("unknown card %q in deck", code)
		}
		switch def.Type {
		case "hero", "alter_ego", "obligation":
			if def.Type == "obligation" {
				for j := 0; j < qty; j++ {
					p.ObligationDeck = append(p.ObligationDeck, Card{Code: code, Owner: p.ID})
				}
			}
			continue
		}
		for j := 0; j < qty; j++ {
			p.Deck = append(p.Deck, Card{Code: code, Owner: p.ID})
		}
	}
	// Nemesis set for the hero.
	if set := heroNemesisSet(spec.HeroBase); set != "" {
		p.NemesisDeck = append(p.NemesisDeck, EncounterSetCards(set)...)
	}
	return p, nil
}

// heroNemesisSet finds the "<hero>_nemesis" set code for a hero base code.
func heroNemesisSet(heroBase string) string {
	def, ok := DB.Lookup(data.HeroSideCode(heroBase))
	if !ok {
		return ""
	}
	set := def.CardSet + "_nemesis"
	if cards := DB.InSet(set); len(cards) > 0 {
		return set
	}
	return ""
}

// setupScenario builds villains, main scheme and the encounter deck. The
// deck is gathered before the main scheme spawns: a-side setup effects
// (e.g. Klaw searching for a side scheme) need to see it.
func (g *Game) setupScenario(scen *ScenarioDef) {
	for _, base := range scen.VillainBases {
		stages := VillainStageCodes(base)
		if len(stages) == 0 {
			continue
		}
		g.spawnVillain(stages, 1)
	}
	g.EncounterDeck = scen.gatherEncounterDeck()
	if g.Difficulty == "expert" {
		g.EncounterDeck = append(g.EncounterDeck, EncounterSetCards("expert")...)
	}
	if len(scen.MainSchemeStages) > 0 {
		g.spawnMainScheme(scen.MainSchemeStages, 1)
	}
}

func (g *Game) spawnVillain(stages []string, stage int) *Villain {
	code := stages[stage-1]
	def := DB.MustLookup(code)
	v := &Villain{
		ID:        g.nextEntityID(KindVillain),
		Code:      code,
		Stage:     stage,
		MaxHP:     deref(def.HP, 10),
		SchemeVal: deref(def.Scheme, 1),
		AttackVal: deref(def.Attack, 1),
		Tough:     def.HasKeyword("Toughness"),
	}
	v.stageCodes = stages
	g.Villains[v.ID] = v
	g.logMajorf("%s enters play (stage %s)", def.Name, def.StageLabel)
	return v
}

// spawnMainScheme brings a scheme stage in on its a face ("01097a"),
// queues the a face's reveal effects and then the flip to the b face —
// the stage codes registered in scenarios are b codes carrying the
// gameplay stats, which are read up front.
func (g *Game) spawnMainScheme(stages []string, stage int) *MainScheme {
	code := stages[stage-1]
	def := DB.MustLookup(code)
	s := &MainScheme{
		ID:         g.nextEntityID(KindMainScheme),
		Code:       data.BaseCode(code) + "a",
		StageCodes: stages,
		Stage:      stage,
		MaxThreat:  deref(def.Threat, 10),
		Threat:     deref(def.BaseThreat, 0) + deref(def.EscalationThreat, 0)*len(g.Players),
		Hazard:     def.Hazards,
	}
	g.MainScheme = s
	g.logMajorf("Main scheme: %s reveals stage %s", def.Name, s.EDef().StageLabel)
	if b := behavior(s.Code); b.MainSchemeRevealed != nil {
		g.Push(b.MainSchemeRevealed(g, s)...)
	}
	g.Push(FlipMainScheme{Scheme: s.ID})
	return s
}
