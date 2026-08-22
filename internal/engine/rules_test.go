package engine_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register core set content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

// fillerDeck is a minimal deck of basic resource cards.
func fillerDeck() map[string]int {
	return map[string]int{
		"01088": 9, // Energy
		"01089": 9, // Genius
	}
}

func newRulesGame(t *testing.T, seed int64, heroes ...string) *engine.Game {
	t.Helper()
	if len(heroes) == 0 {
		heroes = []string{"01001"}
	}
	specs := make([]engine.PlayerSpec, len(heroes))
	for i, h := range heroes {
		specs[i] = engine.PlayerSpec{
			Name:     "P" + string(rune('1'+i)),
			HeroBase: h,
			Deck:     fillerDeck(),
		}
	}
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: "01097",
		Players:    specs,
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

func promptOf(pq *engine.PendingQuestion) string {
	if pq == nil {
		return "<none>"
	}
	return pq.Question.Prompt
}

// keepHands answers the setup mulligan(s) with "keep hand".
func keepHands(t *testing.T, g *engine.Game) {
	t.Helper()
	for i := 0; i < len(g.Players); i++ {
		pq := g.Pending()
		if pq == nil || pq.Question.Prompt != "Mulligan?" {
			t.Fatalf("expected mulligan prompt, got %q", promptOf(pq))
		}
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("keep hand: %v", err)
		}
	}
}

// drillForm answers the pending turn menu with the form change: an
// in-turn choice whose message processing lets previously pushed messages
// resolve without advancing the phase.
func drillForm(t *testing.T, g *engine.Game) {
	t.Helper()
	pq := g.Pending()
	if pq == nil {
		t.Fatal("no pending question to drill")
	}
	if _, err := pq.Question.Leaf("form"); err != nil {
		t.Fatalf("no form choice in %q: %v", pq.Question.Prompt, err)
	}
	if err := g.Answer(pq.Player, []string{"form"}); err != nil {
		t.Fatalf("answer form: %v", err)
	}
}

func firstPlayer(g *engine.Game) *engine.Player {
	for _, p := range g.Players {
		if p.FirstPlayer {
			return p
		}
	}
	return g.Players[0]
}

// TestMulliganDiscardRedraws: mulliganning N cards discards them and draws
// N replacements (hand size unchanged, discard grows).
func TestMulliganDiscardRedraws(t *testing.T) {
	g := newRulesGame(t, 11)
	p := g.Players[0]
	hand := len(p.Hand)
	pq := g.Pending()
	if pq.Question.Prompt != "Mulligan?" {
		t.Fatalf("expected mulligan prompt, got %q", pq.Question.Prompt)
	}
	// Mulliganning the first and third card: nested choose_n under the
	// "mulligan" root choice (sub-ids carry the "mulligan." prefix).
	if err := g.Answer(pq.Player, []string{"mulligan.0", "mulligan.2"}); err != nil {
		t.Fatalf("answer mulligan: %v", err)
	}
	if len(p.Hand) != hand {
		t.Fatalf("hand size should stay %d after mulligan, got %d", hand, len(p.Hand))
	}
	if len(p.Discard) != 2 {
		t.Fatalf("expected 2 mulliganned cards in discard, got %d", len(p.Discard))
	}
}

// TestEndOfPlayerPhaseDrawsAndReadies: after the last turn, hands refill
// to hand size and the identity readies before the villain activates.
func TestEndOfPlayerPhaseDrawsAndReadies(t *testing.T) {
	g := newRulesGame(t, 3)
	keepHands(t, g)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Exhausted = true
	p.Hand = p.Hand[:3] // shrink the hand so the draw-up matters

	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"end-turn"}); err != nil {
		t.Fatalf("end turn: %v", err)
	}
	if pq = g.Pending(); pq != nil && pq.Question.Prompt == "Discard cards before drawing up?" {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("keep: %v", err)
		}
	}
	// Hero form means the villain attacks. Spider-Sense style interrupts
	// wrap the defense prompt; the defense choices live in each choice's
	// Then subtree.
	pq = g.Pending()
	if pq == nil || pq.Question.Prompt != "Interrupts" {
		t.Fatalf("expected villain attack interrupts prompt, got %q", promptOf(pq))
	}
	// The hero defense is only offered when the identity was readied by
	// the end of the player phase.
	found := false
	for _, c := range pq.Question.Choices {
		if c.Then == nil {
			continue
		}
		for _, d := range c.Then.Choices {
			if strings.Contains(d.Label, "to defend") {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("hero should be readied before the villain activates (no defend choice)")
	}
	if len(p.Hand) != p.HandSize(g) {
		t.Fatalf("hand should be drawn up to %d, got %d", p.HandSize(g), len(p.Hand))
	}
}

// TestHandSizeDiscardDownForced: a hand over hand size must discard down
// at the end of the player phase; fewer than the required discards is
// rejected by validation.
func TestHandSizeDiscardDownForced(t *testing.T) {
	g := newRulesGame(t, 4)
	keepHands(t, g)
	p := g.Players[0]
	for len(p.Hand) < p.HandSize(g)+2 {
		p.Hand = append(p.Hand, engine.Card{ID: g.NextCardID(), Code: "01088", Owner: p.ID})
	}
	over := len(p.Hand) - p.HandSize(g)

	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"end-turn"}); err != nil {
		t.Fatalf("end turn: %v", err)
	}
	pq = g.Pending()
	if pq == nil || !strings.Contains(pq.Question.Prompt, "Discard down to hand size") {
		t.Fatalf("expected forced discard prompt, got %q", promptOf(pq))
	}
	// Fewer than the required discards must be rejected.
	var few []string
	for i, c := range pq.Question.Choices {
		if i >= over-1 {
			break
		}
		few = append(few, c.ID)
	}
	if err := g.Answer(pq.Player, few); err == nil {
		t.Fatal("expected validation error for discarding too few cards")
	}
	// Discarding exactly the overflow brings the hand down to hand size.
	var exact []string
	for i, c := range pq.Question.Choices {
		if i >= over {
			break
		}
		exact = append(exact, c.ID)
	}
	if err := g.Answer(pq.Player, exact); err != nil {
		t.Fatalf("discard down: %v", err)
	}
	if len(p.Hand) != p.HandSize(g) {
		t.Fatalf("hand should be %d after discarding down, got %d", p.HandSize(g), len(p.Hand))
	}
}

// TestFirstPlayerTokenPasses: the token moves clockwise when the round
// ends.
func TestFirstPlayerTokenPasses(t *testing.T) {
	g := newRulesGame(t, 5, "01001", "03001")
	keepHands(t, g)
	before := firstPlayer(g)
	other := g.Players[0]
	if before == other {
		other = g.Players[1]
	}

	// Play out round 1 passively: both turns, discards, villain phase.
	for i := 0; i < 60 && g.Round < 2; i++ {
		pq := g.Pending()
		if pq == nil {
			g.Run()
			continue
		}
		prefer := []string{"pass-interrupt", "continue", "keep", "take", "end-turn"}
		ans := []string(nil)
		for _, id := range prefer {
			for _, c := range pq.Question.Choices {
				if c.ID == id && !c.Disabled {
					ans = []string{id}
					break
				}
			}
			if ans != nil {
				break
			}
		}
		if ans == nil {
			ans = pickDefault(pq.Question)
		}
		if ans == nil {
			t.Fatalf("no answer for %q", pq.Question.Prompt)
		}
		if err := g.Answer(pq.Player, ans); err != nil {
			t.Fatalf("answer %v on %q: %v", ans, pq.Question.Prompt, err)
		}
	}
	if g.Round < 2 {
		t.Fatalf("round should have advanced past 1, round=%d", g.Round)
	}
	if !other.FirstPlayer || before.FirstPlayer {
		t.Fatalf("first player token should pass from %s to %s", before.Name, other.Name)
	}
}

// TestVillainActivatesPerPlayerThenTheirMinions: in the villain phase the
// villain activates against each player in player order, and after each
// activation that player's engaged minions activate against them.
func TestVillainActivatesPerPlayerThenTheirMinions(t *testing.T) {
	g := newRulesGame(t, 6, "01001", "03001")
	keepHands(t, g)
	p1 := firstPlayer(g)
	p2 := g.Players[0]
	if p1 == p2 {
		p2 = g.Players[1]
	}
	// Both in hero form: every activation becomes an attack question asked
	// of the targeted player.
	p1.Side = engine.SideHero
	p2.Side = engine.SideHero
	m1 := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01099", MaxHP: 3, AttackVal: 2, EngagedWith: p1.ID}
	m2 := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01099", MaxHP: 3, AttackVal: 2, EngagedWith: p2.ID}
	g.Minions[m1.ID] = m1
	g.Minions[m2.ID] = m2

	// End both turns (discard prompts default to keep / discard down).
prompts:
	for i := 0; i < 12; i++ {
		pq := g.Pending()
		if pq == nil {
			t.Fatalf("no pending question at step %d", i)
		}
		switch {
		case pq.Question.Prompt == "Your turn":
			if err := g.Answer(pq.Player, []string{"end-turn"}); err != nil {
				t.Fatalf("end turn: %v", err)
			}
		case pq.Question.Prompt == "Discard cards before drawing up?":
			if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
				t.Fatalf("keep: %v", err)
			}
		case strings.Contains(pq.Question.Prompt, "Discard down to hand size"):
			if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
				t.Fatalf("discard down: %v", err)
			}
		default:
			break prompts // villain phase reached
		}
	}

	// Activation order: villain vs P1, P1's minion, villain vs P2, P2's
	// minion — each asking the targeted player to defend. Villain attacks
	// may wrap the defense prompt in an interrupts question (the defense
	// choices live in the first choice's Then subtree); minion attacks ask
	// directly.
	answerAttack := func(i int, expect *engine.Player) {
		t.Helper()
		pq := g.Pending()
		if pq == nil {
			t.Fatalf("step %d: no pending question", i)
		}
		if pq.Player != expect.ID {
			t.Fatalf("step %d: prompt %q should target %s, targets %s",
				i, pq.Question.Prompt, expect.Name, pq.Player)
		}
		var path string
		switch {
		case pq.Question.Prompt == "Interrupts":
			// Take the attack via the first choice's defense subtree.
			outer := pq.Question.Choices[0]
			if outer.Then == nil {
				t.Fatalf("step %d: interrupts choice lacks defense subtree", i)
			}
			for _, d := range outer.Then.Choices {
				if d.Label == "Take the attack" {
					path = d.ID
				}
			}
		case strings.Contains(pq.Question.Prompt, "defend?"):
			path = "take"
		default:
			t.Fatalf("step %d: expected attack prompt, got %q", i, pq.Question.Prompt)
		}
		if path == "" {
			t.Fatalf("step %d: no take-the-attack path found", i)
		}
		if err := g.Answer(pq.Player, []string{path}); err != nil {
			t.Fatalf("step %d answer %q: %v", i, path, err)
		}
	}
	for i, expect := range []*engine.Player{p1, p1, p2, p2} {
		answerAttack(i, expect)
	}
}

// TestPlayerEliminationNotInstantLoss: eliminating one player continues
// the game; engaged minions re-engage; only total elimination loses.
func TestPlayerEliminationNotInstantLoss(t *testing.T) {
	g := newRulesGame(t, 7, "01001", "03001")
	keepHands(t, g)
	p1 := g.Players[0]
	p2 := g.Players[1]
	mn := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01099", MaxHP: 3, EngagedWith: p2.ID}
	g.Minions[mn.ID] = mn

	heldToken := p2.FirstPlayer
	// Kill P2 outright.
	g.Push(engine.DamageEntity{Target: p2.ID, Damage: 999, Source: p1.ID})
	drillForm(t, g)

	if g.Over {
		t.Fatalf("game should continue after one elimination: %s", g.Reason)
	}
	if !p2.KOed {
		t.Fatal("P2 should be eliminated")
	}
	if mn.EngagedWith != p1.ID {
		t.Fatalf("P2's minion should re-engage P1, engaged with %v", mn.EngagedWith)
	}
	if heldToken && !p1.FirstPlayer {
		t.Fatal("first player token should pass off the eliminated player")
	}
	if len(p2.Allies) != 0 || len(p2.Supports) != 0 || len(p2.Upgrades) != 0 {
		t.Fatal("eliminated player's permanents should be discarded")
	}

	// The game keeps running: P2 never takes another turn.
	for i := 0; i < 40; i++ {
		pq := g.Pending()
		if pq == nil {
			g.Run()
			if pq = g.Pending(); pq == nil {
				break
			}
		}
		if g.Over {
			break
		}
		if pq.Player == p2.ID && pq.Question.Prompt == "Your turn" {
			t.Fatal("eliminated player must not take turns")
		}
		ans := pickDefault(pq.Question)
		if ans == nil {
			break
		}
		if err := g.Answer(pq.Player, ans); err != nil {
			break
		}
	}
}

// TestAllPlayersEliminatedLoses: the game is only lost when every player
// has been eliminated.
func TestAllPlayersEliminatedLoses(t *testing.T) {
	g := newRulesGame(t, 8)
	keepHands(t, g)
	p := g.Players[0]
	g.Push(engine.DamageEntity{Target: p.ID, Damage: 999, Source: p.ID})
	drillForm(t, g)
	if !g.Over || g.Won {
		t.Fatalf("solo elimination should lose the game, over=%v won=%v", g.Over, g.Won)
	}
}

// TestPlayerDeckExhaustionReshufflesAndDealsEncounter: drawing from an
// empty player deck reshuffles the discard pile into a new deck and deals
// the player a facedown encounter card.
func TestPlayerDeckExhaustionReshufflesAndDealsEncounter(t *testing.T) {
	g := newRulesGame(t, 9)
	keepHands(t, g)
	p := g.Players[0]
	p.Deck = engine.CardList{}
	for i := 0; i < 3; i++ {
		p.Discard = append(p.Discard, engine.Card{ID: g.NextCardID(), Code: "01088", Owner: p.ID})
	}
	hand := len(p.Hand)

	g.Push(engine.DrawCards{Player: p.ID, N: 2})
	drillForm(t, g)

	if len(p.Hand) != hand+2 {
		t.Fatalf("should have drawn 2 after reshuffle, hand %d -> %d", hand, len(p.Hand))
	}
	if len(p.Deck) != 1 {
		t.Fatalf("reshuffled deck should hold the leftover card, got %d", len(p.Deck))
	}
	if len(p.EncounterDown) != 1 {
		t.Fatalf("deck exhaustion should deal 1 facedown encounter card, got %d", len(p.EncounterDown))
	}
}

// TestEncounterReshuffleAddsAccelerationToken: rebuilding the encounter
// deck places an acceleration token next to the main scheme.
func TestEncounterReshuffleAddsAccelerationToken(t *testing.T) {
	g := newRulesGame(t, 10)
	keepHands(t, g)
	g.EncounterDeck = engine.CardList{}
	g.EncounterDiscard = engine.CardList{
		{ID: g.NextCardID(), Code: "01099"}, // Shocker (minion)
	}
	if g.MainScheme == nil {
		t.Fatal("expected a main scheme")
	}
	tokens := g.MainScheme.AccelerationTokens
	if _, ok := g.DrawEncounter(); !ok {
		t.Fatal("expected a card from the reshuffled encounter deck")
	}
	if g.MainScheme.AccelerationTokens != tokens+1 {
		t.Fatalf("encounter reshuffle should add an acceleration token, %d -> %d",
			tokens, g.MainScheme.AccelerationTokens)
	}
}

// TestAccelerationThreatAppliedInVillainPhase: acceleration tokens add
// threat at the start of the villain phase.
func TestAccelerationThreatAppliedInVillainPhase(t *testing.T) {
	g := newRulesGame(t, 12)
	keepHands(t, g)
	g.MainScheme.AccelerationTokens = 1

	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"end-turn"}); err != nil {
		t.Fatalf("end turn: %v", err)
	}
	if pq = g.Pending(); pq != nil && pq.Question.Prompt == "Discard cards before drawing up?" {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("keep: %v", err)
		}
	}
	if !strings.Contains(g.LogText(), "Acceleration places 1 threat") {
		t.Fatalf("expected acceleration threat in the villain phase, log:\n%s", g.LogText())
	}
}

// TestGuardBlocksVillainAttacks: while a guard minion is engaged, the
// player cannot attack villains (only the guard minion).
func TestGuardBlocksVillainAttacks(t *testing.T) {
	g := newRulesGame(t, 13)
	p := g.Players[0]
	mn := &engine.Minion{
		ID: g.NextEntityID("minion"), Code: "01099", MaxHP: 3, AttackVal: 2,
		Guard: true, EngagedWith: p.ID,
	}
	g.Minions[mn.ID] = mn

	// The first menu was built for alter-ego form; flip to hero so the
	// next menu carries the basic attack.
	pq := g.Pending()
	if pq.Question.Prompt != "Mulligan?" {
		t.Fatalf("expected mulligan prompt, got %q", pq.Question.Prompt)
	}
	if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
		t.Fatalf("keep: %v", err)
	}
	drillForm(t, g) // flips to hero form; menu rebuilds with basic-attack

	pq = g.Pending()
	var attack *engine.Choice
	for i := range pq.Question.Choices {
		if pq.Question.Choices[i].ID == "basic-attack" {
			attack = &pq.Question.Choices[i]
			break
		}
	}
	if attack == nil {
		t.Fatal("hero should have a basic attack choice")
	}
	var villainTargets, minionTargets int
	for _, c := range attack.Then.Choices {
		if c.SourceID.Kind() == "villain" {
			villainTargets++
		}
		if c.SourceID == mn.ID {
			minionTargets++
		}
	}
	if villainTargets != 0 {
		t.Fatalf("guard minion should block villain attacks, %d villain targets offered", villainTargets)
	}
	if minionTargets != 1 {
		t.Fatalf("the guard minion itself should remain targetable, %d targets", minionTargets)
	}
}

// TestCrisisSideSchemeBlocksMainSchemeThwart: while a crisis side scheme
// is in play, the main scheme cannot be thwarted, but the crisis scheme
// itself can.
func TestCrisisSideSchemeBlocksMainSchemeThwart(t *testing.T) {
	g := newRulesGame(t, 14)
	ss := &engine.SideScheme{
		ID: g.NextEntityID("side_scheme"), Code: "01107",
		Threat: 2, MaxThreat: 3, Crisis: true,
	}
	g.SideSchemes[ss.ID] = ss

	pq := g.Pending()
	if pq.Question.Prompt != "Mulligan?" {
		t.Fatalf("expected mulligan prompt, got %q", pq.Question.Prompt)
	}
	if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
		t.Fatalf("keep: %v", err)
	}
	drillForm(t, g) // hero form

	pq = g.Pending()
	var thwart *engine.Choice
	for i := range pq.Question.Choices {
		if pq.Question.Choices[i].ID == "basic-thwart" {
			thwart = &pq.Question.Choices[i]
			break
		}
	}
	if thwart == nil {
		t.Fatal("hero should have a basic thwart choice (the crisis scheme is thwartable)")
	}
	var mainTargets, crisisTargets int
	for _, c := range thwart.Then.Choices {
		if g.MainScheme != nil && c.SourceID == g.MainScheme.ID {
			mainTargets++
		}
		if c.SourceID == ss.ID {
			crisisTargets++
		}
	}
	if mainTargets != 0 {
		t.Fatalf("crisis side scheme should block main-scheme thwarting, %d main targets", mainTargets)
	}
	if crisisTargets != 1 {
		t.Fatalf("the crisis side scheme itself should be thwartable, %d targets", crisisTargets)
	}
}

// TestMinionActivationOrderChosenByPlayer: with 2+ minions engaged, the
// player chooses the order in which they activate (official villain phase
// step 2b); with a single minion no order question is asked.
func TestMinionActivationOrderChosenByPlayer(t *testing.T) {
	g := newRulesGame(t, 16)
	keepHands(t, g)
	p := g.Players[0]
	p.Side = engine.SideHero
	// Two distinct minions: Sandman (ATK 3) and Hydra Mercenary (ATK 1).
	sandman := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01102", MaxHP: 4, AttackVal: 3, SchemeVal: 2, EngagedWith: p.ID}
	merc := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01101", MaxHP: 3, AttackVal: 1, SchemeVal: 0, EngagedWith: p.ID}
	g.Minions[sandman.ID] = sandman
	g.Minions[merc.ID] = merc

	// End the turn; the player is in hero form so the villain attacks
	// first (Spider-Sense interrupts wrap the defense prompt).
	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"end-turn"}); err != nil {
		t.Fatalf("end turn: %v", err)
	}
	if pq = g.Pending(); pq != nil && pq.Question.Prompt == "Discard cards before drawing up?" {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("keep: %v", err)
		}
	}
	if pq = g.Pending(); pq != nil && strings.Contains(pq.Question.Prompt, "Discard down to hand size") {
		if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
			t.Fatalf("discard down: %v", err)
		}
	}
	// Villain attack: take it via the interrupts' defense subtree.
	pq = g.Pending()
	if pq == nil || pq.Question.Prompt != "Interrupts" {
		t.Fatalf("expected villain attack interrupts, got %q", promptOf(pq))
	}
	takePath := ""
	for _, c := range pq.Question.Choices {
		if c.Then == nil {
			continue
		}
		for _, d := range c.Then.Choices {
			if d.Label == "Take the attack" {
				takePath = d.ID
			}
		}
	}
	if takePath == "" {
		t.Fatal("no take-the-attack path")
	}
	if err := g.Answer(pq.Player, []string{takePath}); err != nil {
		t.Fatalf("take villain attack: %v", err)
	}

	// The player now chooses which minion activates first.
	pq = g.Pending()
	if pq == nil || pq.Question.Prompt != "Choose the next minion to activate" {
		t.Fatalf("expected minion order question, got %q", promptOf(pq))
	}
	if pq.Player != p.ID {
		t.Fatalf("order question should ask the engaged player, asks %s", pq.Player)
	}
	var mercPath string
	for _, c := range pq.Question.Choices {
		if c.SourceID == merc.ID {
			mercPath = c.ID
		}
	}
	if mercPath == "" {
		t.Fatal("Hydra Mercenary should be orderable first")
	}
	if err := g.Answer(pq.Player, []string{mercPath}); err != nil {
		t.Fatalf("pick merc first: %v", err)
	}
	// The chosen minion's attack resolves first.
	pq = g.Pending()
	if pq == nil || !strings.Contains(pq.Question.Prompt, "Hydra Mercenary attacks for 1") {
		t.Fatalf("expected Hydra Mercenary's attack first, got %q", promptOf(pq))
	}
	if err := g.Answer(pq.Player, []string{"take"}); err != nil {
		t.Fatalf("take merc attack: %v", err)
	}
	// The remaining single minion activates without another order question.
	pq = g.Pending()
	if pq == nil || !strings.Contains(pq.Question.Prompt, "Sandman attacks for 3") {
		t.Fatalf("expected Sandman's attack second (no extra order ask), got %q", promptOf(pq))
	}
}

// TestOtherPlayerMayDefend: the attacked player holds the initial decision.
// "Take the attack" resolves immediately without asking anyone; choosing
// "Ask another player to defend" offers the defense to the other players in
// clockwise order — if all decline the attack resolves undefended against
// the attacked player, and the first player to accept defends in their
// place (exhausting and taking the reduced damage).
func TestOtherPlayerMayDefend(t *testing.T) {
	g := newRulesGame(t, 17, "03001", "03001")
	keepHands(t, g)
	p1 := firstPlayer(g)
	p2 := g.Players[0]
	if p1 == p2 {
		p2 = g.Players[1]
	}
	p1.Side = engine.SideHero
	p2.Side = engine.SideHero // ready to defend in P1's place
	mn := &engine.Minion{ID: g.NextEntityID("minion"), Code: "01101", MaxHP: 3, AttackVal: 1, EngagedWith: p1.ID}
	g.Minions[mn.ID] = mn

	// Run out both turns: end-turn, optional/forced discards, then the
	// villain attacks P1 (hero form).
	answerTurnPrompts := func() {
		t.Helper()
		for i := 0; i < 12; i++ {
			pq := g.Pending()
			if pq == nil {
				t.Fatal("no pending question while advancing turns")
			}
			switch {
			case pq.Question.Prompt == "Your turn":
				if err := g.Answer(pq.Player, []string{"end-turn"}); err != nil {
					t.Fatalf("end turn: %v", err)
				}
			case pq.Question.Prompt == "Discard cards before drawing up?":
				if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
					t.Fatalf("keep: %v", err)
				}
			case strings.Contains(pq.Question.Prompt, "Discard down to hand size"):
				if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
					t.Fatalf("discard down: %v", err)
				}
			default:
				return // attack prompt reached
			}
		}
	}
	answerTurnPrompts()

	// findLeaf locates a choice by label in the root question or any Then
	// subtree (the interrupts question nests the defense choices).
	findLeaf := func(q *engine.Question, label string) string {
		var walk func(q *engine.Question) string
		walk = func(q *engine.Question) string {
			for _, c := range q.Choices {
				if c.Label == label {
					return c.ID
				}
				if c.Then != nil {
					if id := walk(c.Then); id != "" {
						return id
					}
				}
			}
			return ""
		}
		return walk(q)
	}

	// Attack 1 (villain vs P1): "take" resolves immediately — no other
	// player is asked.
	pq := g.Pending()
	if pq == nil || pq.Player != p1.ID {
		t.Fatalf("expected villain attack on %s, got %q for %v", p1.Name, promptOf(pq), pq.Player)
	}
	takePath := findLeaf(pq.Question, "Take the attack")
	if takePath == "" {
		t.Fatal("no take-the-attack choice")
	}
	dmgBefore := p1.Damage
	if err := g.Answer(pq.Player, []string{takePath}); err != nil {
		t.Fatalf("take: %v", err)
	}
	if p1.Damage <= dmgBefore {
		t.Fatalf("P1 should take the attack directly, damage %d -> %d", dmgBefore, p1.Damage)
	}
	if p2.Damage != 0 {
		t.Fatal("P2 must not be asked or harmed when P1 simply takes the attack")
	}

	// Attack 2 (P1's minion, ATK 1): P1 asks for a substitute defender, P2
	// declines, and the attack falls back to P1 undefended.
	pq = g.Pending()
	if pq == nil || pq.Player != p1.ID || !strings.Contains(pq.Question.Prompt, "defend?") {
		t.Fatalf("expected minion attack on %s, got %q for %v", p1.Name, promptOf(pq), pq.Player)
	}
	askPath := findLeaf(pq.Question, "Ask another player to defend")
	if askPath == "" {
		t.Fatal("attacked player should be offered the ask-another-player option")
	}
	if err := g.Answer(pq.Player, []string{askPath}); err != nil {
		t.Fatalf("ask defend: %v", err)
	}
	pq = g.Pending()
	if pq == nil || pq.Player != p2.ID || !strings.Contains(pq.Question.Prompt, "defend?") {
		t.Fatalf("expected cross-defense offer to %s, got %q for %v", p2.Name, promptOf(pq), pq.Player)
	}
	if !strings.Contains(pq.Question.Prompt, p1.Name) {
		t.Fatalf("cross-defense prompt should name the attacked player: %q", pq.Question.Prompt)
	}
	afterAttack1 := p1.Damage
	if err := g.Answer(pq.Player, []string{"pass"}); err != nil {
		t.Fatalf("pass: %v", err)
	}
	if p1.Damage != afterAttack1+1 {
		t.Fatalf("after everyone passes the minion attack (ATK 1) should hit P1 undefended, damage %d -> %d",
			afterAttack1, p1.Damage)
	}
	if p2.Damage != 0 || p2.Exhausted {
		t.Fatal("P2 passing must leave them untouched")
	}

	// Attack 3 (villain vs P2): P2 asks for a substitute defender and P1
	// accepts with their hero — P2 takes nothing, P1 exhausts.
	pq = g.Pending()
	if pq == nil || pq.Player != p2.ID {
		t.Fatalf("expected villain attack on %s, got %q for %v", p2.Name, promptOf(pq), pq.Player)
	}
	askPath = findLeaf(pq.Question, "Ask another player to defend")
	if askPath == "" {
		t.Fatal("ask-another-player option missing on P2's defense question")
	}
	if err := g.Answer(pq.Player, []string{askPath}); err != nil {
		t.Fatalf("ask defend: %v", err)
	}
	pq = g.Pending()
	if pq == nil || pq.Player != p1.ID || !strings.Contains(pq.Question.Prompt, "defend?") {
		t.Fatalf("expected cross-defense offer to %s, got %q for %v", p1.Name, promptOf(pq), pq.Player)
	}
	found := false
	for _, c := range pq.Question.Choices {
		if c.ID == "hero-defend" {
			found = true
		}
	}
	if !found {
		t.Fatal("cross-defense offer should include the hero defense")
	}
	p2Before := p2.Damage
	if err := g.Answer(pq.Player, []string{"hero-defend"}); err != nil {
		t.Fatalf("hero-defend: %v", err)
	}
	if !p1.Exhausted {
		t.Fatal("defending player should be exhausted")
	}
	if p2.Damage != p2Before {
		t.Fatalf("attacked player should take no damage when defended for, %d -> %d", p2Before, p2.Damage)
	}
}

// TestInterruptEffectFiresWithDefense: answering an interrupt branch fires
// the interrupt's own effect (Spider-Sense draws 1) before the defense
// resolves, and each branch answers via its own id prefix (the original
// repro: "pass-interrupt.<leaf>" used to be an invalid answer).
func TestInterruptEffectFiresWithDefense(t *testing.T) {
	g := newRulesGame(t, 18)
	keepHands(t, g)
	p := g.Players[0]
	p.Side = engine.SideHero

	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"end-turn"}); err != nil {
		t.Fatalf("end turn: %v", err)
	}
	for pq = g.Pending(); pq != nil; {
		switch {
		case pq.Question.Prompt == "Discard cards before drawing up?":
			if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
				t.Fatalf("keep: %v", err)
			}
		case strings.Contains(pq.Question.Prompt, "Discard down to hand size"):
			if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
				t.Fatalf("discard down: %v", err)
			}
		default:
			return // attack prompt reached
		}
		pq = g.Pending()
	}

	// Villain attack wrapped in Spider-Sense interrupts.
	pq = g.Pending()
	if pq == nil || pq.Question.Prompt != "Interrupts" {
		t.Fatalf("expected interrupts prompt, got %q", promptOf(pq))
	}
	var interruptID, passTakePath string
	for _, c := range pq.Question.Choices {
		if c.ID == "interrupt-player-0" && c.Then != nil {
			interruptID = c.ID
			for _, d := range c.Then.Choices {
				if d.Label == "Take the attack" {
					passTakePath = d.ID // interrupt branch's take leaf
				}
			}
		}
	}
	if interruptID == "" || passTakePath == "" {
		t.Fatal("interrupt branch with a take leaf should exist")
	}
	// The pass branch must own its namespace now.
	passBranchTake := ""
	for _, c := range pq.Question.Choices {
		if c.ID == "pass-interrupt" && c.Then != nil {
			for _, d := range c.Then.Choices {
				if d.Label == "Take the attack" {
					passBranchTake = d.ID
				}
			}
		}
	}
	if passBranchTake == "" || !strings.HasPrefix(passBranchTake, "pass-interrupt.") {
		t.Fatalf("pass branch must own its id namespace, got %q", passBranchTake)
	}

	// Take the attack THROUGH the interrupt: Spider-Sense draws 1 and the
	// attack still resolves against the player.
	handBefore := len(p.Hand)
	damageBefore := p.Damage
	if err := g.Answer(pq.Player, []string{passTakePath}); err != nil {
		t.Fatalf("answer interrupt take: %v", err)
	}
	if len(p.Hand) != handBefore+1 {
		t.Fatalf("Spider-Sense should draw 1 when its interrupt is chosen, hand %d -> %d",
			handBefore, len(p.Hand))
	}
	if p.Damage <= damageBefore {
		t.Fatalf("the attack should still resolve, damage %d -> %d", damageBefore, p.Damage)
	}
}

// TestPassInterruptBranchSkipsEffect: answering through the pass branch
// fires no interrupt effect (hand unchanged) while the attack resolves.
func TestPassInterruptBranchSkipsEffect(t *testing.T) {
	g := newRulesGame(t, 18)
	keepHands(t, g)
	p := g.Players[0]
	p.Side = engine.SideHero

	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"end-turn"}); err != nil {
		t.Fatalf("end turn: %v", err)
	}
	for pq = g.Pending(); pq != nil; {
		switch {
		case pq.Question.Prompt == "Discard cards before drawing up?":
			if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
				t.Fatalf("keep: %v", err)
			}
		case strings.Contains(pq.Question.Prompt, "Discard down to hand size"):
			if err := g.Answer(pq.Player, pickDefault(pq.Question)); err != nil {
				t.Fatalf("discard down: %v", err)
			}
		default:
			return // attack prompt reached
		}
		pq = g.Pending()
	}

	pq = g.Pending()
	if pq == nil || pq.Question.Prompt != "Interrupts" {
		t.Fatalf("expected interrupts prompt, got %q", promptOf(pq))
	}
	var takePath string
	for _, c := range pq.Question.Choices {
		if c.ID != "pass-interrupt" || c.Then == nil {
			continue
		}
		for _, d := range c.Then.Choices {
			if d.Label == "Take the attack" {
				takePath = d.ID
			}
		}
	}
	if takePath == "" {
		t.Fatal("pass branch take leaf missing")
	}
	handBefore := len(p.Hand)
	damageBefore := p.Damage
	// The original repro path: this used to be an invalid answer.
	if err := g.Answer(pq.Player, []string{takePath}); err != nil {
		t.Fatalf("answer pass-interrupt take: %v", err)
	}
	if len(p.Hand) != handBefore {
		t.Fatalf("passing interrupts must not draw, hand %d -> %d", handBefore, len(p.Hand))
	}
	if p.Damage <= damageBefore {
		t.Fatalf("the attack should resolve, damage %d -> %d", damageBefore, p.Damage)
	}
}

// TestAskAnotherPlayerToAct: the active player may ask another player to
// act — picking the target only; the asked player gets a turn-like menu
// (no form change, no end-turn, no re-asking) and performs one action or
// Done, after which the requester's turn resumes. The requester can back
// out with "Never mind" before sending.
func TestAskAnotherPlayerToAct(t *testing.T) {
	g := newRulesGame(t, 19, "01001", "01001")
	keepHands(t, g)
	req := firstPlayer(g)
	asked := g.Players[0]
	if asked == req {
		asked = g.Players[1]
	}
	// Aunt May (support, Alter-Ego Action: exhaust → heal 4) in play for
	// the asked player, who starts wounded and in alter-ego form.
	may := &engine.Support{ID: g.NextEntityID("support"), Code: "01006", Owner: asked.ID}
	g.Supports[may.ID] = may
	asked.Supports = append(asked.Supports, may.ID)
	asked.Damage = 5

	pq := g.Pending()
	if pq == nil || pq.Question.Prompt != "Your turn" || pq.Player != req.ID {
		t.Fatalf("expected requester's turn menu, got %q for %v", promptOf(pq), pq.Player)
	}

	// 1) Back out before asking: "Never mind" returns to the turn menu.
	askPath, pickNever := "", ""
	for _, c := range pq.Question.Choices {
		if c.ID == "ask-action" {
			askPath = c.ID
			for _, d := range c.Then.Choices {
				if d.Label == "Never mind" {
					pickNever = d.ID
				}
			}
		}
	}
	if askPath == "" || pickNever == "" {
		t.Fatal("ask branch and its Never mind choice should exist in multiplayer")
	}
	if err := g.Answer(pq.Player, []string{pickNever}); err != nil {
		t.Fatalf("never mind: %v", err)
	}
	if pq = g.Pending(); pq == nil || pq.Question.Prompt != "Your turn" || pq.Player != req.ID {
		t.Fatalf("Never mind should return to the requester's turn, got %q for %v", promptOf(pq), pq.Player)
	}

	// 2) Send the request: the asked player receives a turn-like menu.
	var askWho string
	for _, c := range pq.Question.Choices {
		if c.ID == "ask-action" {
			for _, d := range c.Then.Choices {
				if d.Label == asked.Name {
					askWho = d.ID
				}
			}
		}
	}
	if askWho == "" {
		t.Fatal("asked player should be selectable")
	}
	if err := g.Answer(pq.Player, []string{askWho}); err != nil {
		t.Fatalf("ask: %v", err)
	}
	pq = g.Pending()
	if pq == nil || pq.Player != asked.ID {
		t.Fatalf("asked player should hold the prompt, got %q for %v", promptOf(pq), pq.Player)
	}
	if !strings.Contains(pq.Question.Prompt, "asks you to act") {
		t.Fatalf("unexpected asked-menu prompt %q", pq.Question.Prompt)
	}
	var mayPath, donePath string
	for _, c := range pq.Question.Choices {
		switch {
		case c.ID == "form" || c.ID == "end-turn" || c.ID == "ask-action":
			t.Fatalf("asked menu must not offer %q", c.ID)
		case c.CardCode == "01006":
			mayPath = c.ID
		case c.ID == "done":
			donePath = c.ID
		}
	}
	if mayPath == "" || donePath == "" {
		t.Fatal("asked menu should list Aunt May's action and Done")
	}

	// 3) Done: nothing happens, requester's turn resumes.
	if err := g.Answer(pq.Player, []string{donePath}); err != nil {
		t.Fatalf("done: %v", err)
	}
	if pq = g.Pending(); pq == nil || pq.Player != req.ID || pq.Question.Prompt != "Your turn" {
		t.Fatalf("Done should return to the requester's turn, got %q for %v", promptOf(pq), pq.Player)
	}
	if asked.Damage != 5 || may.Exhausted {
		t.Fatal("declining to act must change nothing")
	}

	// 4) Ask again and trigger Aunt May: one action resolves, then the
	// requester's turn resumes.
	if err := g.Answer(pq.Player, []string{askWho}); err != nil {
		t.Fatalf("ask again: %v", err)
	}
	if pq = g.Pending(); pq == nil || pq.Player != asked.ID {
		t.Fatalf("asked player should hold the prompt again, got %q", promptOf(pq))
	}
	mayPath = ""
	for _, c := range pq.Question.Choices {
		if c.CardCode == "01006" {
			mayPath = c.ID
		}
	}
	if mayPath == "" {
		t.Fatal("Aunt May's action should be offered")
	}
	if err := g.Answer(pq.Player, []string{mayPath}); err != nil {
		t.Fatalf("trigger Aunt May: %v", err)
	}
	if !may.Exhausted {
		t.Fatal("Aunt May should be exhausted after her action")
	}
	if asked.Damage != 1 {
		t.Fatalf("Aunt May should heal the asked player 4 (5 -> 1), got %d", asked.Damage)
	}
	if pq = g.Pending(); pq == nil || pq.Player != req.ID || pq.Question.Prompt != "Your turn" {
		t.Fatalf("requester's turn should resume after the action, got %q for %v", promptOf(pq), pq.Player)
	}
}

// TestAskBranchAbsentInSolo: solo games offer no ask-another-player branch.
func TestAskBranchAbsentInSolo(t *testing.T) {
	g := newRulesGame(t, 20)
	keepHands(t, g)
	pq := g.Pending()
	for _, c := range pq.Question.Choices {
		if c.ID == "ask-action" {
			t.Fatal("solo games must not offer the ask branch")
		}
	}
}

// TestChangeFormAllowedWhileExhausted: changing form keeps the character's
// ready/exhausted state and is allowed while exhausted.
func TestChangeFormAllowedWhileExhausted(t *testing.T) {
	g := newRulesGame(t, 15)
	keepHands(t, g)
	p := g.Players[0]
	if p.IsHero() {
		t.Fatal("test setup: player should start in alter-ego form")
	}
	p.Exhausted = true

	pq := g.Pending()
	found := false
	for _, c := range pq.Question.Choices {
		if c.ID == "form" {
			found = true
		}
	}
	if !found {
		t.Fatal("form choice should be offered while exhausted")
	}
	if err := g.Answer(pq.Player, []string{"form"}); err != nil {
		t.Fatalf("change form: %v", err)
	}
	if !p.IsHero() {
		t.Fatal("should be in hero form after the change")
	}
	if !p.Exhausted {
		t.Fatal("changing form must preserve the exhausted state")
	}
}
