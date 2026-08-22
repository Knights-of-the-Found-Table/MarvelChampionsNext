package engine

import (
	"testing"
)

func stubMsg(text string) Message { return DrawCards{Player: "p", N: 1} }

// TestWithThenIsolatesSubtrees: chaining several choices into the same
// question object gives each branch its own id namespace and an
// independently mutable copy.
func TestWithThenIsolatesSubtrees(t *testing.T) {
	shared := Ask("defend?",
		Choice{ID: "", Label: "Take the attack"}.Msgs(stubMsg("take")),
		Choice{ID: "", Label: "Hero defend"}.Msgs(stubMsg("defend")),
	)
	root := Ask("Interrupts",
		Choice{ID: "interrupt-player-0", Label: "interrupt"}.Msgs(stubMsg("interrupt")).WithThen(shared),
		Choice{ID: "pass-interrupt", Label: "pass"}.WithThen(shared),
	)
	if !root.idsUnique() {
		t.Fatal("shared subtree must yield unique ids after WithThen copies it")
	}
	var underInterrupt, underPass []Choice
	for _, c := range root.Choices {
		switch c.ID {
		case "interrupt-player-0":
			underInterrupt = c.Then.Choices
		case "pass-interrupt":
			underPass = c.Then.Choices
		}
	}
	if len(underInterrupt) != 2 || len(underPass) != 2 {
		t.Fatalf("both branches should keep the subtree, got %d and %d choices",
			len(underInterrupt), len(underPass))
	}
	if underInterrupt[0].ID != "interrupt-player-0.0" || underPass[0].ID != "pass-interrupt.0" {
		t.Fatalf("branches must own their prefixes, got %q and %q",
			underInterrupt[0].ID, underPass[0].ID)
	}
	// Mutating one branch's subtree must not leak into the other.
	underInterrupt[0].Label = "mutated"
	if underPass[0].Label == "mutated" {
		t.Fatal("WithThen must deep-copy: mutating one branch leaked into the other")
	}
	// The caller's original question is untouched by WithThen.
	if shared.Choices[0].ID != "0" {
		t.Fatalf("original standalone question should keep its own ids, got %q", shared.Choices[0].ID)
	}
}

// TestChainResolvesAllLevels: Chain returns the root-to-leaf choices, and
// descending past a leaf errors.
func TestChainResolvesAllLevels(t *testing.T) {
	leaf := Ask("defend?",
		Choice{Label: "Take the attack"}.Msgs(stubMsg("take")),
	)
	root := Ask("Interrupts",
		Choice{ID: "interrupt-player-0", Label: "interrupt"}.Msgs(stubMsg("interrupt")).WithThen(leaf),
		Choice{ID: "pass", Label: "pass"}.Msgs(stubMsg("pass")),
	)
	chain, err := root.Chain("interrupt-player-0.0")
	if err != nil {
		t.Fatalf("chain: %v", err)
	}
	if len(chain) != 2 {
		t.Fatalf("expected 2 choices in the chain, got %d", len(chain))
	}
	if chain[0].ID != "interrupt-player-0" || chain[1].ID != "interrupt-player-0.0" {
		t.Fatalf("unexpected chain ids: %q -> %q", chain[0].ID, chain[1].ID)
	}
	if _, err := root.Chain("pass.0"); err == nil {
		t.Fatal("descending past a leaf choice must error")
	}
	if _, err := root.Chain("nope"); err == nil {
		t.Fatal("unknown path must error")
	}
}

// TestChainMsgsFallsBackOnDuplicateIDs: questions persisted before the
// WithThen copy carried the same ids under several branches; answering
// those must keep the legacy leaf-only semantics.
func TestChainMsgsFallsBackOnDuplicateIDs(t *testing.T) {
	// Hand-craft the legacy shape: both branches' subtrees carry the first
	// branch's prefix.
	root := &Question{Type: "choose_one"}
	subA := &Question{Type: "choose_one", Choices: []Choice{
		{ID: "interrupt-player-0.0", Label: "Take the attack", msgs: []Message{stubMsg("take")}},
	}}
	subB := &Question{Type: "choose_one", Choices: []Choice{
		{ID: "interrupt-player-0.0", Label: "Take the attack", msgs: []Message{stubMsg("take")}},
	}}
	root.Choices = []Choice{
		{ID: "interrupt-player-0", Label: "interrupt", msgs: []Message{stubMsg("interrupt")}, Then: subA},
		{ID: "pass-interrupt", Label: "pass", Then: subB},
	}
	if root.idsUnique() {
		t.Fatal("duplicate ids should be detected")
	}
	g := &Game{}
	msgs, err := g.chainMsgs(root, "interrupt-player-0.0")
	if err != nil {
		t.Fatalf("chainMsgs fallback: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("legacy trees must resolve leaf-only, got %d messages", len(msgs))
	}
}

// TestChainMsgsAggregatesLevels: unique-id trees fire every level's
// messages root first.
func TestChainMsgsAggregatesLevels(t *testing.T) {
	root := Ask("Interrupts",
		Choice{ID: "interrupt-player-0", Label: "interrupt"}.
			Msgs(DrawCards{Player: "p", N: 1}).
			WithThen(Ask("defend?",
				Choice{Label: "Take the attack"}.Msgs(DiscardCards{Player: "p"}),
			)),
	)
	g := &Game{}
	msgs, err := g.chainMsgs(root, "interrupt-player-0.0")
	if err != nil {
		t.Fatalf("chainMsgs: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected both levels' messages, got %d", len(msgs))
	}
	if _, ok := msgs[0].(DrawCards); !ok {
		t.Fatal("root choice's message must fire first")
	}
	if _, ok := msgs[1].(DiscardCards); !ok {
		t.Fatal("leaf choice's message must fire second")
	}
}
