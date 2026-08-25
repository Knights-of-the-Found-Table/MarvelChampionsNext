package engine

import (
	"strings"
	"testing"
)

// plainPayGame builds a minimal game with one player holding two basic
// resource cards (01088 Energy, 01089 Genius).
func plainPayGame() (*Game, *Player) {
	g := &Game{}
	p := &Player{ID: PlayerID("p1")}
	g.Players = []*Player{p}
	p.Hand = CardList{{ID: "c1", Code: "01088"}, {ID: "c2", Code: "01089"}}
	return g, p
}

func resourcePicks(q *Question) []*Choice {
	var picks []*Choice
	for i := range q.Choices {
		if q.Choices[i].Kind == ChoiceResource {
			picks = append(picks, &q.Choices[i])
		}
	}
	return picks
}

// TestUnroutedPaymentResolves: CustomPaymentQuestion flows that carry no
// routing context key (goblin Intimidation 02035, Sonic Boom 01123, Trouble
// in Otherworld 25028, Running Interference) must settle as a plain payment
// instead of failing with "payment context missing target".
func TestUnroutedPaymentResolves(t *testing.T) {
	g, p := plainPayGame()
	q := g.CustomPaymentQuestion(p, 2, S("Intimidation: spend 2 resources"),
		map[string]any{"player": "p1"})
	picks := resourcePicks(q)
	if len(picks) < 2 {
		t.Fatalf("need two resource choices, got %d", len(picks))
	}
	msgs, err := g.validateSelection(q, picks[:2])
	if err != nil {
		t.Fatalf("unrouted payment should resolve as a plain pay: %v", err)
	}
	var paid *ResourcePay
	for i := range msgs {
		if rp, ok := msgs[i].(ResourcePay); ok {
			paid = &rp
		}
	}
	if paid == nil || len(paid.Cards) != 2 {
		t.Fatalf("expected ResourcePay with 2 cards, got %+v", msgs)
	}
}

// TestCustomPaymentIconContext: icon constraints come from the explicit
// abilityIcons context key, never from parsing the prompt text.
func TestCustomPaymentIconContext(t *testing.T) {
	g, p := plainPayGame()
	q := g.CustomPaymentQuestion(p, 3, S("pay [energy] [mental] [physical]"),
		map[string]any{"player": "p1", "abilityIcons": "energy:1 mental:1 physical:1"})
	if got := q.Context["abilityIcons"]; got != "energy:1 mental:1 physical:1" {
		t.Fatalf("abilityIcons passthrough, got %v", got)
	}
	if len(q.PayIcons) != 3 || q.PayIcons[0] != (IconReq{Icon: "energy", N: 1}) {
		t.Fatalf("PayIcons mirror of the spec, got %+v", q.PayIcons)
	}
	// Without the key there is no constraint, even if the prose mentions icons.
	q2 := g.CustomPaymentQuestion(p, 2, S("pay [energy] [mental] or else"),
		map[string]any{"player": "p1"})
	if _, ok := q2.Context["abilityIcons"]; ok {
		t.Fatal("no icon spec may be invented from prompt text")
	}
	if q2.PayIcons != nil {
		t.Fatalf("PayIcons must stay empty without a spec, got %+v", q2.PayIcons)
	}
}

// TestResourceChoiceIcons: payment choices expose the icons they contribute
// as structured data (the client-side tracker must not parse label text).
func TestResourceChoiceIcons(t *testing.T) {
	g, p := plainPayGame()
	q := g.CustomPaymentQuestion(p, 2, S("pay 2"), map[string]any{"player": "p1"})
	picks := resourcePicks(q)
	if len(picks) < 2 {
		t.Fatalf("need two resource choices, got %d", len(picks))
	}
	// 01088 "Energy" prints two energy icons (resource_energy: 2).
	if got := picks[0].Icons; len(got) != 2 || got[0] != "energy" || got[1] != "energy" {
		t.Fatalf("hand-card choice should carry its printed icons, got %+v", got)
	}
}

// TestCheckIconRequirements: the server-side validator accepts exact and
// wild-covered payments and rejects real shortfalls.
func TestCheckIconRequirements(t *testing.T) {
	if err := checkIconRequirements([]string{"energy", "mental"}, "energy:1 mental:1"); err != nil {
		t.Fatalf("exact match should pass: %v", err)
	}
	if err := checkIconRequirements([]string{"wild", "physical"}, "energy:1 physical:1"); err != nil {
		t.Fatalf("wild should cover energy: %v", err)
	}
	err := checkIconRequirements([]string{"energy", "energy"}, "energy:1 mental:1")
	if err == nil || !strings.Contains(err.Error(), "mental") {
		t.Fatalf("missing mental should fail, got %v", err)
	}
}

// TestPrintedRiderParsing: Hinder (leading and behind a "Standard Mode
// Only." preamble) and the Boost self-spawn rider are parsed at data load;
// game logic reads structured fields, never the printed text.
func TestPrintedRiderParsing(t *testing.T) {
	// 32071 Medical Emergency — "Hinder 2[per_hero]. ..." (leading).
	if got := DB.MustLookup("32071").KeywordValue("Hinder"); got != 2 {
		t.Fatalf("32071 Hinder = %d, want 2", got)
	}
	// 16178a Legionnaires' Brainwashing — "**Standard Mode Only.**\nHinder 3...".
	if got := DB.MustLookup("16178a").KeywordValue("Hinder"); got != 3 {
		t.Fatalf("16178a Hinder = %d, want 3", got)
	}
	// A card without the keyword parses to 0.
	if got := DB.MustLookup("01088").KeywordValue("Hinder"); got != 0 {
		t.Fatalf("01088 Hinder = %d, want 0", got)
	}
	// Boost self-spawn rider: exact "this card" and "this minion" wordings
	// (39045 Fetch Quest side scheme, 31036 Spot minion) plus an
	// environment (04080 Dense Forest); a card without the rider stays off.
	for _, code := range []string{"39045", "31036", "04080"} {
		if !DB.MustLookup(code).BoostEntersPlay {
			t.Fatalf("%s should carry the Boost self-spawn rider", code)
		}
	}
	for _, code := range []string{"01088", "32071"} {
		if DB.MustLookup(code).BoostEntersPlay {
			t.Fatalf("%s should not carry the Boost self-spawn rider", code)
		}
	}
	// Retaliate keeps parsing through the shared accessor (01172 prints
	// "Retaliate 1").
	if got := DB.MustLookup("01172").KeywordValue("Retaliate"); got != 1 {
		t.Fatalf("01172 Retaliate = %d, want 1", got)
	}
}
