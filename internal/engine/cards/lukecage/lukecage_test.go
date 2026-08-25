package lukecage_test

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/lukecage"
)

func TestLukeCageContracts(t *testing.T) {
	identity := engine.LookupBehavior("62001")
	if identity == nil || !identity.UnlimitedTough || identity.React == nil {
		t.Fatal("Luke Cage must register unlimited tough and a reaction")
	}
	p := &engine.Player{ID: "p1", HeroCode: "62001", Tough: 3}
	msgs := identity.React(&engine.Game{}, p, engine.ToughDiscarded{Target: p.ID})
	if len(msgs) != 1 {
		t.Fatalf("tough discard reaction = %#v, want one message", msgs)
	}
	dmg, ok := msgs[0].(engine.DamageEntity)
	if !ok || dmg.Target != p.ID || dmg.Damage != 1 || !dmg.Unpreventable {
		t.Fatalf("forced response = %#v, want 1 unpreventable damage", msgs[0])
	}

	cost := engine.LookupBehavior("62006").CardCost
	if cost == nil {
		t.Fatal("Sweet Christmas must expose a cost hook")
	}
	def := engine.DB.MustLookup("62006")
	if got := cost(&engine.Game{}, p, def); got != 3 {
		t.Fatalf("Sweet Christmas discount = %d, want 3", got)
	}
}

func TestLukeCagePackCodesAreRegistered(t *testing.T) {
	for _, code := range []string{
		"62001", "62002", "62003", "62004", "62005", "62006", "62007", "62008", "62009", "62010", "62011", "62012", "62013", "62014", "62015", "62016", "62017", "62018", "62019", "62020", "62021", "62022", "62023", "62024", "62025", "62026", "62027", "62028", "62029", "62030", "62031", "62032", "62033", "62034", "62035", "62036", "62037",
	} {
		if engine.LookupBehavior(code) == nil {
			t.Fatalf("card %s has no registered behavior", code)
		}
	}
}

func TestCottonmouthSchedulesTwoAdditionalAttacks(t *testing.T) {
	b := engine.LookupBehavior("62029")
	if b == nil || b.React == nil {
		t.Fatal("Cottonmouth must register a forced response")
	}
	mn := &engine.Minion{ID: "m1", Code: "62029"}
	msgs := b.React(&engine.Game{}, mn, engine.WindowAfterEnemyAttacked{Enemy: mn.ID, Player: "p1"})
	if len(msgs) != 2 {
		t.Fatalf("Cottonmouth follow-up attacks = %#v, want 2", msgs)
	}
	for _, msg := range msgs {
		attack, ok := msg.(engine.AskAttack)
		if !ok || attack.Enemy != mn.ID || attack.Player != "p1" {
			t.Fatalf("follow-up attack = %#v, want same Cottonmouth/player", msg)
		}
	}
}
