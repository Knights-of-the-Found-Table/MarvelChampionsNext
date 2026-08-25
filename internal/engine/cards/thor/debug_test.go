package thor_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/thor"
)

func TestDebugWorthy(t *testing.T) {
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       5,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Thor", HeroBase: "06001", Deck: map[string]int{"06009": 1}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	p := g.Players[0]
	p.Side = engine.SideAlterEgo
	p.Hand = engine.CardList{}
	for i := 0; i < 5; i++ {
		pq := g.Pending()
		if pq == nil {
			break
		}
		fmt.Printf("Drain Q%d: %q\n", i, pq.Question.Prompt)
		if err := g.Answer(pq.Player, []string{"end-turn"}); err != nil {
			fmt.Printf("Drain err: %v\n", err)
			break
		}
	}
	pq := g.Pending()
	fmt.Printf("After drain pending: %v\n", pq)
	if pq != nil {
		fmt.Printf("After drain PROMPT: %q\n", pq.Question.Prompt)
	}
	b := engine.LookupBehavior("06001")
	abs := b.HeroAbilities(g, p)
	fmt.Printf("Has %d abilities\n", len(abs))
	if abs[0].Execute == nil {
		t.Fatal("no Execute")
	}
	msgs := abs[0].Execute(g, p.ID)
	fmt.Printf("Got %d messages\n", len(msgs))
	for i, m := range msgs {
		fmt.Printf("  msg[%d] = %T\n", i, m)
	}
	g.Push(msgs...)
	g.Run()
	for _, line := range g.Log {
		fmt.Printf("LOG: %s\n", line.Text)
	}
	pq = g.Pending()
	fmt.Printf("After worthy pending: %v\n", pq)
	if pq != nil {
		fmt.Printf("After worthy PROMPT: %q\n", pq.Question.Prompt)
		for _, c := range pq.Question.Choices {
			fmt.Printf("  CHOICE id=%q label=%q code=%q\n", c.ID, c.Label.Text, c.CardCode)
		}
	}
	_ = strings.Contains
}
