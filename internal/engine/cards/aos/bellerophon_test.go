package aos_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/aos"
)

// Contract test: invoke the hooks directly and inspect the target choice's
// exact message chain. This avoids entering the recurring turn menu.
func TestBellerophonStartsLoadedAndFiresThroughTough(t *testing.T) {
	g := newAOSGame(t, "50001", mariaDeck())
	p := g.Players[0]
	s := &engine.Support{ID: g.NextEntityID("support"), Code: "50018", Owner: p.ID}
	g.Supports[s.ID] = s
	p.Supports = append(p.Supports, s.ID)

	b := engine.LookupBehavior("50018")
	if b == nil || b.OnPlay == nil || b.Abilities == nil {
		t.Fatal("The Bellerophon behavior is incomplete")
	}
	b.OnPlay(g, s)
	if s.Counters != 3 {
		t.Fatalf("Bellerophon counters = %d, want 3", s.Counters)
	}

	var villainID engine.EntityID
	for id, v := range g.Villains {
		villainID = id
		v.Tough = true
		break
	}
	if villainID == "" {
		t.Fatal("test game has no villain")
	}
	mn := &engine.Minion{
		ID: g.NextEntityID("minion"), Code: "01101", MaxHP: 5,
		EngagedWith: p.ID, Tough: true,
	}
	g.Minions[mn.ID] = mn

	abilities := b.Abilities(g, s)
	if len(abilities) != 1 || !abilities[0].Exhaust || abilities[0].Execute == nil {
		t.Fatalf("Bellerophon abilities = %#v, want one exhaust action", abilities)
	}
	msgs := abilities[0].Execute(g, s.ID)
	if len(msgs) != 1 {
		t.Fatalf("fire returned %d messages, want one question", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok || len(ask.Question.Choices) != 1 {
		t.Fatalf("fire message = %#v, want one player choice", msgs[0])
	}
	if _, err := ask.Question.Chain(ask.Question.Choices[0].ID); err != nil {
		t.Fatalf("choice chain: %v", err)
	}
	raw, err := json.Marshal(ask.Question)
	if err != nil {
		t.Fatalf("marshal fire question: %v", err)
	}
	encoded := string(raw)
	if strings.Count(encoded, `"t":"engine.AddEntityCounter"`) != 1 ||
		strings.Count(encoded, `"t":"engine.ClearTough"`) != 2 ||
		strings.Count(encoded, `"t":"engine.DamageEntity"`) != 2 {
		t.Fatalf("unexpected fire payload: %s", encoded)
	}
}
