package wolv_test

import (
	"strings"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// register content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/wolv"
)

// TestWolverineImplemented: both sides count as implemented.
func TestWolverineImplemented(t *testing.T) {
	if !engine.Implemented("35001a") {
		t.Fatal("Wolverine should count as implemented")
	}
	if !engine.Implemented("35001b") {
		t.Fatal("Logan (alter-ego) should count as implemented")
	}
}

// TestHealingFactorOnPlayerPhase: the React hook on the identity heals
// 2 damage when the player phase begins.
func TestHealingFactorOnPlayerPhase(t *testing.T) {
	g := mustNewWolvGame(t)
	p := g.Players[0]
	p.Damage = 5
	b := engine.LookupBehavior("35001")
	if b == nil || b.React == nil {
		t.Fatal("Wolverine should expose React for Healing Factor")
	}
	msgs := b.React(g, p, engine.BeginPhase{Phase: engine.PhasePlayer})
	if len(msgs) != 1 {
		t.Fatalf("Healing Factor should emit 1 message, got %d", len(msgs))
	}
	heal, ok := msgs[0].(engine.HealEntity)
	if !ok {
		t.Fatalf("Healing Factor should emit HealEntity, got %T", msgs[0])
	}
	if heal.N != 2 {
		t.Fatalf("Healing Factor should heal 2, got %d", heal.N)
	}
	if heal.Target != p.ID {
		t.Fatalf("Healing Factor should target the player, got %s", heal.Target)
	}
}

// TestHealingFactorIgnoresOtherPhases: the React only fires on the
// player phase.
func TestHealingFactorIgnoresOtherPhases(t *testing.T) {
	g := mustNewWolvGame(t)
	p := g.Players[0]
	p.Damage = 5
	b := engine.LookupBehavior("35001")
	for _, phase := range []engine.Phase{engine.PhaseSetup, engine.PhaseResource, engine.PhaseVillain} {
		msgs := b.React(g, p, engine.BeginPhase{Phase: phase})
		if len(msgs) != 0 {
			t.Fatalf("Healing Factor should not fire on %s, got %v", phase, msgs)
		}
	}
}

// TestAdamantiumSkeletonGrantsHPAndATK: the upgrade adds +4 to MaxHP
// and +1 to ATK via IdentityStats.
func TestAdamantiumSkeletonGrantsHPAndATK(t *testing.T) {
	g := mustNewWolvGame(t)
	p := g.Players[0]
	hp := p.MaxHP
	atk := p.AttackStat(g)
	upgrade := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "35004", Owner: p.ID}
	g.Upgrades[upgrade.ID] = upgrade
	p.Upgrades = append(p.Upgrades, upgrade.ID)

	b := engine.LookupBehavior("35004")
	if b == nil {
		t.Fatal("Adamantium Skeleton should be registered")
	}
	if b.OnPlay != nil {
		msgs := b.OnPlay(g, upgrade)
		if len(msgs) != 0 {
			t.Fatalf("Adamantium Skeleton OnPlay should emit no messages, got %v", msgs)
		}
	}
	if p.MaxHP != hp+4 {
		t.Fatalf("Adamantium Skeleton should add +4 HP, got %d -> %d", hp, p.MaxHP)
	}
	if b.IdentityStats == nil {
		t.Fatal("Adamantium Skeleton should expose IdentityStats")
	}
	bonus := b.IdentityStats(p)
	if bonus.ATK != 1 {
		t.Fatalf("Adamantium Skeleton should grant +1 ATK, got %d", bonus.ATK)
	}
	_ = atk
}

// TestBerserkerFrenzyDrawsOnEnemyDamage: the React draws 1 card when
// Wolverine takes damage from a villain.
func TestBerserkerFrenzyDrawsOnEnemyDamage(t *testing.T) {
	g := mustNewWolvGame(t)
	p := g.Players[0]
	villain := &engine.Villain{ID: g.NextEntityID("villain"), Code: "01094", MaxHP: 20}
	g.Villains[villain.ID] = villain
	upgrade := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "35005", Owner: p.ID}
	g.Upgrades[upgrade.ID] = upgrade
	p.Upgrades = append(p.Upgrades, upgrade.ID)

	b := engine.LookupBehavior("35005")
	if b == nil || b.React == nil {
		t.Fatal("Berserker Frenzy should expose React")
	}
	handBefore := len(p.Hand)
	msgs := b.React(g, upgrade, engine.DamageEntity{Target: p.ID, Damage: 3, Source: villain.ID})
	if len(msgs) != 1 {
		t.Fatalf("Berserker Frenzy should emit 1 message, got %d", len(msgs))
	}
	draw, ok := msgs[0].(engine.DrawCards)
	if !ok {
		t.Fatalf("Berserker Frenzy should emit DrawCards, got %T", msgs[0])
	}
	if draw.Player != p.ID || draw.N != 1 {
		t.Fatalf("Berserker Frenzy should draw 1 for the player, got %+v", draw)
	}
	_ = handBefore
}

// TestBerserkerFrenzyIgnoresNonEnemyDamage: damage from a non-villain
// source (e.g., another hero) does not draw a card.
func TestBerserkerFrenzyIgnoresNonEnemyDamage(t *testing.T) {
	g := mustNewWolvGame(t)
	p := g.Players[0]
	upgrade := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "35005", Owner: p.ID}
	g.Upgrades[upgrade.ID] = upgrade
	p.Upgrades = append(p.Upgrades, upgrade.ID)

	b := engine.LookupBehavior("35005")
	// Damage from a player source should not trigger.
	msgs := b.React(g, upgrade, engine.DamageEntity{Target: p.ID, Damage: 1, Source: p.ID})
	if len(msgs) != 0 {
		t.Fatalf("Berserker Frenzy should ignore non-enemy damage, got %v", msgs)
	}
}

// TestIGotBetterSavesFromDefeat: the upgrade React saves the player
// from lethal damage by setting HP to 5 and discarding the card.
func TestIGotBetterSavesFromDefeat(t *testing.T) {
	g := mustNewWolvGame(t)
	p := g.Players[0]
	p.MaxHP = 12
	p.Damage = 10 // 2 HP left; 3 damage would defeat
	upgrade := &engine.Upgrade{ID: g.NextEntityID("upgrade"), Code: "35006", Owner: p.ID}
	g.Upgrades[upgrade.ID] = upgrade
	p.Upgrades = append(p.Upgrades, upgrade.ID)

	b := engine.LookupBehavior("35006")
	if b == nil || b.React == nil {
		t.Fatal("\"I Got Better!\" should expose React")
	}
	msgs := b.React(g, upgrade, engine.DamageEntity{Target: p.ID, Damage: 3, Source: engine.EntityID("villain-1")})
	if len(msgs) != 0 {
		t.Fatalf("\"I Got Better!\" should emit no messages, got %d", len(msgs))
	}
	if p.HP() != 5 {
		t.Fatalf("\"I Got Better!\" should leave the player at 5 HP, got %d", p.HP())
	}
	if p.Exhausted {
		t.Fatal("\"I Got Better!\" should ready the player")
	}
	if g.Upgrades[upgrade.ID] != nil {
		t.Fatal("\"I Got Better!\" should be discarded")
	}
}

// TestLogansCabinShufflesWolverineCard: the support's Alter-Ego
// ability offers a choice of Wolverine cards from discard.
func TestLogansCabinShufflesWolverineCard(t *testing.T) {
	g := mustNewWolvGame(t)
	p := g.Players[0]
	p.Side = engine.SideAlterEgo
	p.Discard = engine.CardList{
		{ID: g.NextCardID(), Code: "35008", Owner: p.ID}, // Berserker Barrage (Wolverine card)
	}

	b := engine.LookupBehavior("35007")
	if b == nil || b.Abilities == nil {
		t.Fatal("Logan's Cabin should expose Abilities")
	}
	lc := &engine.Support{ID: g.NextEntityID("support"), Code: "35007", Owner: p.ID}
	abs := b.Abilities(g, lc)
	if len(abs) != 1 {
		t.Fatalf("Logan's Cabin should expose 1 ability, got %d", len(abs))
	}
	if !abs[0].AlterEgoOnly {
		t.Fatal("Logan's Cabin ability should be Alter-Ego only")
	}
	if !abs[0].Exhaust {
		t.Fatal("Logan's Cabin ability should require exhaustion")
	}
	msgs := abs[0].Execute(g, lc.ID)
	if len(msgs) != 1 {
		t.Fatalf("Logan's Cabin should ask 1 question, got %d", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok {
		t.Fatalf("Logan's Cabin should emit AskQuestion, got %T", msgs[0])
	}
	if !strings.Contains(ask.Question.Prompt, "Logan's Cabin") {
		t.Fatalf("Logan's Cabin prompt should mention the support, got %q", ask.Question.Prompt)
	}
	if len(ask.Question.Choices) != 1 {
		t.Fatalf("Logan's Cabin should offer 1 Wolverine card, got %d", len(ask.Question.Choices))
	}
	if ask.Question.Choices[0].CardCode != "35008" {
		t.Fatalf("Logan's Cabin should offer Berserker Barrage, got %s", ask.Question.Choices[0].CardCode)
	}
}

// TestPastDemonsStunAndConfuse: the obligation's penalty branch
// stuns and confuses the player. The msgs attached to a leaf choice
// are private, so we verify the question has both an "exhaust and
// remove" choice and a "penalty" choice (the standard
// ExhaustOrPenalty template).
func TestPastDemonsStunAndConfuse(t *testing.T) {
	g := mustNewWolvGame(t)
	p := g.Players[0]
	p.Side = engine.SideHero
	p.Stunned = false
	p.Confused = false
	obligation := engine.Card{ID: g.NextCardID(), Code: "35027", Owner: p.ID}

	b := engine.LookupBehavior("35027")
	if b == nil || b.ResolveObligation == nil {
		t.Fatal("Past Demons should expose ResolveObligation")
	}
	msgs := b.ResolveObligation(g, p, obligation)
	if len(msgs) != 1 {
		t.Fatalf("Past Demons should ask 1 question, got %d", len(msgs))
	}
	ask, ok := msgs[0].(engine.AskQuestion)
	if !ok {
		t.Fatalf("Past Demons should emit AskQuestion, got %T", msgs[0])
	}
	var hasExhaust, hasPenalty bool
	for _, c := range ask.Question.Choices {
		if strings.Contains(c.Label.Text, "Exhaust") {
			hasExhaust = true
		}
		if c.ID == "penalty" {
			hasPenalty = true
		}
	}
	if !hasExhaust {
		t.Fatal("Past Demons should offer an exhaust-and-remove branch")
	}
	if !hasPenalty {
		t.Fatal("Past Demons should offer a penalty branch")
	}
	// The penalty label is the second arg of ExhaustOrPenalty; we
	// passed "You are stunned and confused" so both effects should be
	// in the label.
	for _, c := range ask.Question.Choices {
		if c.ID == "penalty" {
			if !strings.Contains(c.Label.Text, "stunned") {
				t.Fatalf("Past Demons penalty should mention 'stunned', got %q", c.Label.Text)
			}
			if !strings.Contains(c.Label.Text, "confused") {
				t.Fatalf("Past Demons penalty should mention 'confused', got %q", c.Label.Text)
			}
		}
	}
}

// mustNewWolvGame returns a Wolverine game with the opening hand
// answered (mulligan kept). The deck is small and tests mutate state
// directly.
func mustNewWolvGame(t *testing.T) *engine.Game {
	t.Helper()
	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       3501,
		ScenarioID: "01097",
		Players: []engine.PlayerSpec{
			{Name: "Wolverine", HeroBase: "35001", Deck: map[string]int{
				"35002": 1, "35003": 1, "35004": 1, "35005": 1, "35006": 1,
				"35007": 1, "35008": 1, "35009": 1, "35010": 1, "35011": 1,
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if pq := g.Pending(); pq != nil && pq.Question.Prompt == "Mulligan?" {
		if err := g.Answer(pq.Player, []string{"keep"}); err != nil {
			t.Fatalf("keep mulligan: %v", err)
		}
	}
	return g
}

// TestRemainingwolvRegistered sweeps the pack's remaining cards.
func TestRemainingwolvSweep(t *testing.T) {
	for _, def := range engine.DB.All() {
		if def.PackCode != "wolv" {
			continue
		}
		if def.Text == "" {
			continue
		}
		if !engine.Implemented(def.Code) {
			t.Errorf("card %s (%s) has no registered behavior", def.Code, def.Name)
		}
	}
}
