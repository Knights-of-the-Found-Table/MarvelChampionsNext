package engine

import (
	"strings"
	"testing"
)

// 卡牌数据（cards/core 的注册）由本包测试二进制中的 engine_test 包
// 文件 import 触发 init，进程内共享 DB，这里无需（也不能）直接 import。

func newLegacyTestGame(t *testing.T) *Game {
	t.Helper()
	g, err := NewGame(NewGameOptions{
		Seed:       42,
		ScenarioID: "01097",
		Players: []PlayerSpec{
			{Name: "Tester", HeroBase: "01001", Deck: map[string]int{
				"01002": 1, "01003": 2, "01004": 2, "01005": 2, "01006": 1,
				"01007": 2, "01008": 2, "01088": 3, "01089": 3, "01090": 3,
			}},
		},
	})
	if err != nil {
		t.Fatalf("new game: %v", err)
	}
	return g
}

// legacyTurnMenu reproduces a pending question persisted before nested
// choice ids carried path prefixes. Answering against such a question must
// rebuild it first — otherwise paths resolve to the wrong root choice (e.g.
// a target answer plays a hand card instead).
func TestAnswerRebuildsLegacyPendingQuestion(t *testing.T) {
	g := newLegacyTestGame(t)
	p := g.Players[0]
	// The game opens paused on the setup mulligan; keep the hand so the
	// first player phase begins and the turn menu becomes pending.
	if pq := g.Pending(); pq != nil {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("answer mulligan: %v", err)
		}
	}
	p.Side = SideHero // basic-attack is hero-side only
	p.Exhausted = false
	p.Stunned = false

	q := g.TurnMenu(p)
	// 模拟旧格式：剥掉子树选项 id 的父路径前缀
	for i := range q.Choices {
		c := &q.Choices[i]
		if c.Then == nil {
			continue
		}
		for j := range c.Then.Choices {
			if id := c.Then.Choices[j].ID; strings.HasPrefix(id, c.ID+".") {
				c.Then.Choices[j].ID = id[len(c.ID)+1:]
			}
		}
	}
	g.pending = &PendingQuestion{Player: p.ID, Question: q}

	var attackID, targetID string
	for i := range q.Choices {
		c := &q.Choices[i]
		if c.ID != "basic-attack" {
			continue
		}
		attackID = c.ID
		for j := range c.Then.Choices {
			targetID = c.Then.Choices[j].ID // legacy bare id ("0")
		}
	}
	if attackID == "" {
		t.Fatal("basic-attack not available")
	}
	legacyPath := attackID + "." + targetID

	vid := sortedIDs(g.Villains)[0]
	hpBefore := g.Villains[vid].HP()
	if err := g.Answer(p.ID, []string{legacyPath}); err != nil {
		t.Fatalf("answer legacy path: %v", err)
	}
	if hp := g.Villains[vid].HP(); hp >= hpBefore {
		t.Fatalf("attack did not resolve: hp %d -> %d", hpBefore, hp)
	}
	if !p.Exhausted {
		t.Fatal("hero should be exhausted after basic attack")
	}
}

// RebuildTurnMenu must be a no-op for current-format questions.
func TestRebuildTurnMenuNoOpForCurrentFormat(t *testing.T) {
	g := newLegacyTestGame(t)
	p := g.Players[0]
	q := g.TurnMenu(p)
	g.pending = &PendingQuestion{Player: p.ID, Question: q}
	if g.RebuildTurnMenu() {
		t.Fatal("current-format question must not be rebuilt")
	}
}
