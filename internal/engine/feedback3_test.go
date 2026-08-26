package engine_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// behaviors referenced by these tests
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/sinistermotives"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/spiderwoman"
)

// TestVillainStageCodesData: stage grouping must follow the printed data —
// same-name roman stages chain across separate codes (Rhino), b-side
// personas chain per stage (Green Goblin), lettered variants never chain
// (Baron Zemo A/B), alternate a/c forms of one stage collapse (En Sabah
// Nur), and a linked back face that IS the next stage chains (Zemo A1→A2,
// Apocalypse I→II on the flip side).
func TestVillainStageCodesData(t *testing.T) {
	cases := []struct {
		base string
		want []string
	}{
		{"01094", []string{"01094", "01095", "01096"}},          // Rhino I/II/III
		{"01113", []string{"01113", "01114", "01115"}},          // Klaw
		{"02001", []string{"02001a", "02001b"}},                 // double-sided: persona sides only
		{"50165a", []string{"50165a", "50165b"}},                // Zemo A1→A2 (flip is next stage)
		{"50166a", []string{"50166a", "50166b"}},                // Zemo B1→B2, A/B variants never mix
		{"45184a", []string{"45184a", "45185a", "45186a"}},      // En Sabah Nur: a/c forms collapse
		{"45101a", []string{"45101a", "45101b", "45102a"}},      // Apocalypse I→II (flip) → III
		{"11001", []string{"11001", "11006"}},                   // Kang I→III (scenario hook drives it)
		{"56059", []string{"56059", "56060", "56061", "56062"}}, // Civil War leader Iron Man
	}
	for _, c := range cases {
		got := engine.VillainStageCodes(c.base)
		if len(got) != len(c.want) {
			t.Errorf("%s: stages %v, want %v", c.base, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: stages %v, want %v", c.base, got, c.want)
				break
			}
		}
	}
}

// TestAllyDamageTriggersMagicShield: 15008 prevents damage to a friendly
// ALLY too (consequential damage), not only the identity.
func TestAllyDamageTriggersMagicShield(t *testing.T) {
	g := newRulesGame(t, 7)
	p := g.Players[0]
	p.Exhausted = false
	ally := &engine.Ally{ID: g.NextEntityID("ally"), Code: "01083", Owner: p.ID, MaxHP: 3}
	g.Allies[ally.ID] = ally
	p.Allies = append(p.Allies, ally.ID)
	shield := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "15008", Owner: p.ID}
	g.Upgrades[shield.ID] = shield
	p.Upgrades = append(p.Upgrades, shield.ID)

	g.Push(engine.DamageEntity{Target: ally.ID, Damage: 2, Source: p.ID})
	// The fresh game sits at its mulligan question; answering resumes the
	// queue (the damage rides ahead of it).
	pq := g.Pending()
	if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
		t.Fatalf("answer mulligan: %v", err)
	}

	if ally.Damage != 0 {
		t.Fatalf("Magic Shield should prevent all 2 damage, ally damage = %d", ally.Damage)
	}
	if _, still := g.Upgrades[shield.ID]; still {
		t.Fatal("Magic Shield should be discarded after preventing")
	}
	found := false
	for _, c := range p.Discard {
		if c.Code == "15008" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Magic Shield should be in the discard, got %v", p.Discard)
	}
}

// TestSkyDestroyerTriggersOnShieldPlay: 27055's reaction keys off the
// structured S.H.I.E.L.D. trait (parseTraits no longer shatters the
// abbreviation into single letters).
func TestSkyDestroyerTriggersOnShieldPlay(t *testing.T) {
	g := newRulesGame(t, 7)
	p := g.Players[0]
	sd := &engine.Support{ID: g.NextEntityID("support"), Code: "27055", Owner: p.ID}
	g.Supports[sd.ID] = sd
	p.Supports = append(p.Supports, sd.ID)
	villain := &engine.Villain{ID: g.NextEntityID("villain"), Code: "01094", MaxHP: 14}
	g.Villains[villain.ID] = villain

	b := engine.LookupBehavior("27055")
	if b == nil || b.React == nil {
		t.Fatal("27055 behavior missing")
	}
	msgs := b.React(g, sd, engine.PlayCard{Player: p.ID, Card: engine.Card{ID: "x1", Code: "01092"}})
	if len(msgs) == 0 {
		t.Fatal("Sky-Destroyer should trigger after playing a S.H.I.E.L.D. card (01092 Helicarrier)")
	}
	nonShield := b.React(g, sd, engine.PlayCard{Player: p.ID, Card: engine.Card{ID: "x2", Code: "01088"}})
	if len(nonShield) != 0 {
		t.Fatalf("Sky-Destroyer must not trigger for a non-SHIELD card, got %v", nonShield)
	}
}

// TestUpgradedDronesAttachToEnvironment: 01142 attaches to the Ultron
// Drones environment in play, not to the villain.
func TestUpgradedDronesAttachToEnvironment(t *testing.T) {
	g := newRulesGame(t, 7)
	env := g.SpawnEnvironment("01140")
	villain := &engine.Villain{ID: g.NextEntityID("villain"), Code: "01134", MaxHP: 20}
	g.Villains[villain.ID] = villain

	b := engine.LookupBehavior("01142")
	if b == nil || b.OnAttach == nil {
		t.Fatal("01142 OnAttach missing")
	}
	att := &engine.Attachment{ID: g.NextEntityID("attachment"), Code: "01142"}
	g.Attachments[att.ID] = att
	b.OnAttach(g, att, "")
	if att.Target != env.ID {
		t.Fatalf("01142 should attach to the environment %s, got %s", env.ID, att.Target)
	}
}

// TestHexBoltResolvesEachMilledCard: 15004 evaluates EACH of the three milled
// encounter cards independently by its own printed boost icons — 0 → 2 damage
// to an enemy, 1 → 2 threat from a scheme, 2 → draw 1 card.
func TestHexBoltResolvesEachMilledCard(t *testing.T) {
	b := engine.LookupBehavior("15004")
	if b == nil || b.OnPlay == nil {
		t.Fatal("15004 behavior missing")
	}

	mk := func(codes ...string) []engine.Card {
		out := make([]engine.Card, 0, len(codes))
		for _, c := range codes {
			out = append(out, engine.Card{ID: "e-" + c, Code: c})
		}
		return out
	}

	// 0+0+0 → three independent choose-enemy damage questions.
	g := newRulesGame(t, 7)
	p := g.Players[0]
	villain := &engine.Villain{ID: g.NextEntityID("villain"), Code: "01094", MaxHP: 14}
	g.Villains[villain.ID] = villain
	g.EncounterDeck = mk("01104", "01105", "01104")
	msgs := b.OnPlay(g, &engine.EventCard{Code: "15004", Owner: p.ID})
	if n := countAskQuestion(msgs); n != 3 {
		t.Fatalf("three 0-boost cards should ask 3 enemy questions, got %d: %v", n, msgs)
	}

	// 2+1+0 → draw 1 (for the boost-2 card), a scheme question (boost 1)
	// and an enemy question (boost 0), in mill order.
	g2 := newRulesGame(t, 7)
	p2 := g2.Players[0]
	villain2 := &engine.Villain{ID: g2.NextEntityID("villain"), Code: "01094", MaxHP: 14}
	g2.Villains[villain2.ID] = villain2
	g2.EncounterDeck = mk("01099", "01101", "01104")
	msgs2 := b.OnPlay(g2, &engine.EventCard{Code: "15004", Owner: p2.ID})
	drawIdx, askIdx := -1, -1
	for i, m := range msgs2 {
		if d, ok := m.(engine.DrawCards); ok && d.N == 1 && d.Player == p2.ID && drawIdx == -1 {
			drawIdx = i
		}
		if _, ok := m.(engine.AskQuestion); ok && askIdx == -1 {
			askIdx = i
		}
	}
	if drawIdx != 0 {
		t.Fatalf("boost-2 card (milled first) should draw 1 first, got %v", msgs2)
	}
	if countAskQuestion(msgs2) != 2 {
		t.Fatalf("boost-1 and boost-0 cards should ask 2 questions, got %v", msgs2)
	}
	if askIdx != 1 {
		t.Fatalf("boost-1 scheme question should come after the draw, got %v", msgs2)
	}
}

func countAskQuestion(msgs []engine.Message) int {
	n := 0
	for _, m := range msgs {
		if _, ok := m.(engine.AskQuestion); ok {
			n++
		}
	}
	return n
}
