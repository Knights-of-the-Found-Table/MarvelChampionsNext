package engine

import (
	"strings"
	"testing"
)

// In-package tests for handler-level rules that external tests can only
// reach through the answer queue (which interleaves turn flow).

// While any crisis side scheme is in play, removeThreat must refuse to
// remove threat from the main scheme — whatever the source (basic thwart,
// event effects), matching the printed crisis icon.
func TestRemoveThreatCrisisLock(t *testing.T) {
	g := &Game{SideSchemes: map[EntityID]*SideScheme{}}
	ms := &MainScheme{ID: "mainscheme-1", Code: "01097b", StageCodes: []string{"01097b"}, Threat: 5, MaxThreat: 10}
	g.MainScheme = ms
	sid := EntityID("sidescheme-1")
	g.SideSchemes[sid] = &SideScheme{ID: sid, Code: "01108", Threat: 2, MaxThreat: 4, Crisis: true}

	g.removeThreat(ms.ID, 3, "player-1")
	if ms.Threat != 5 {
		t.Fatalf("crisis in play: main scheme threat = %d, want unchanged 5", ms.Threat)
	}

	g.SideSchemes[sid].Crisis = false
	g.removeThreat(ms.ID, 3, "player-1")
	if ms.Threat != 2 {
		t.Fatalf("no crisis: main scheme threat = %d, want 5-3=2", ms.Threat)
	}
}

// A defeated minion's card reaches the encounter discard pile (unless it
// carries victory points), and its attachments drop off into the discard
// alongside it.
func TestMinionDefeatedGoesToEncounterDiscard(t *testing.T) {
	g := &Game{
		Minions:     map[EntityID]*Minion{},
		Attachments: map[EntityID]*Attachment{},
	}
	mid := EntityID("minion-1")
	aid := EntityID("attachment-1")
	g.Attachments[aid] = &Attachment{ID: aid, Code: "01118"}
	g.Minions[mid] = &Minion{ID: mid, Code: "01103", Attachments: []EntityID{aid}}

	g.process(MinionDefeated{MinionID: mid})

	counts := map[string]int{}
	for _, c := range g.EncounterDiscard {
		counts[c.Code]++
	}
	if counts["01103"] != 1 {
		t.Fatalf("defeated minion should be in the encounter discard once, got %v", g.EncounterDiscard)
	}
	if counts["01118"] != 1 {
		t.Fatalf("the minion's attachment should drop into the encounter discard, got %v", g.EncounterDiscard)
	}
	if _, still := g.Minions[mid]; still {
		t.Fatal("defeated minion should have left play")
	}
	if _, still := g.Attachments[aid]; still {
		t.Fatal("the attachment should have left play with its host")
	}
	if len(g.VictoryDisplay) != 0 {
		t.Fatalf("01103 carries no victory points, victory display = %v", g.VictoryDisplay)
	}
}

// TestVillainDefeatedAdvancesStage: with no scenario hook, defeating a
// multi-stage villain advances to the next stage instead of ending the
// game (Rhino I → II); only the final stage wins.
func TestVillainDefeatedAdvancesStage(t *testing.T) {
	g := &Game{Villains: map[EntityID]*Villain{}}
	v := g.spawnVillain([]string{"01094", "01095", "01096"}, 1)
	g.Push(VillainDefeated{VillainID: v.ID})
	g.Run()
	if g.Over {
		t.Fatal("defeating Rhino stage I must not end the game")
	}
	if v.Code != "01095" || v.Stage != 2 {
		t.Fatalf("Rhino should advance to stage II (01095), got %s stage %d", v.Code, v.Stage)
	}
	// Final stage: defeat wins.
	v.Damage = 0
	g.Push(VillainDefeated{VillainID: v.ID})
	g.Run()
	if v.Stage != 3 {
		t.Fatalf("should be stage III now, got %d", v.Stage)
	}
	v.Damage = 0
	g.Push(VillainDefeated{VillainID: v.ID})
	g.Run()
	if !g.Over || !g.Won {
		t.Fatal("defeating the final villain stage should win the game")
	}
}

// TestVillainDefeatedHookOwnsFlow: a scenario with OnVillainDefeated takes
// the defeat on every stage (the Sinister Six sets members aside instead of
// staging up) — data-driven chaining must not bypass it.
func TestVillainDefeatedHookOwnsFlow(t *testing.T) {
	RegisterScenario(&ScenarioDef{
		ID:   "test-hook-scenario",
		Name: "hook scenario",
		OnVillainDefeated: func(g *Game, v *Villain) []Message {
			delete(g.Villains, v.ID)
			return nil
		},
	})
	g := &Game{Villains: map[EntityID]*Villain{}, ScenarioID: "test-hook-scenario"}
	v := g.spawnVillain([]string{"01094", "01095", "01096"}, 1)
	g.Push(VillainDefeated{VillainID: v.ID})
	g.Run()
	if _, still := g.Villains[v.ID]; still {
		t.Fatal("the scenario hook should have removed the villain on first defeat")
	}
	if g.Over {
		t.Fatal("the hook decided the flow; default win must not also fire")
	}
}

// TestMainSchemeClearedNotDefeated: clearing main-scheme threat to zero is
// journaled as threat loss, never as "Main scheme is defeated".
func TestMainSchemeClearedNotDefeated(t *testing.T) {
	g := &Game{SideSchemes: map[EntityID]*SideScheme{}}
	g.MainScheme = &MainScheme{ID: "mainscheme-1", Code: "01097b", StageCodes: []string{"01097b"}, Threat: 3, MaxThreat: 7}
	g.removeThreat(g.MainScheme.ID, 3, "player-1")
	if g.MainScheme.Threat != 0 {
		t.Fatalf("threat should be 0, got %d", g.MainScheme.Threat)
	}
	for _, e := range g.Log {
		if strings.Contains(e.Key, "mainSchemeDefeated") {
			t.Fatalf("main scheme must not be logged as defeated: %+v", e)
		}
	}
}

// TestDroneDefeatToOwnerDiscard: a facedown Drone minion is the owner's
// card; defeating it puts that facedown card into the owner's discard
// pile, not the encounter discard.
func TestDroneDefeatToOwnerDiscard(t *testing.T) {
	g := &Game{Minions: map[EntityID]*Minion{}, Players: []*Player{{ID: "player-1", Name: "P1"}}}
	g.Players[0].Deck = CardList{{ID: "card-9", Code: "01088", Owner: "player-1"}}
	g.process(SpawnDrone{Player: "player-1"})
	var drone *Minion
	for _, m := range g.Minions {
		drone = m
	}
	if drone == nil || !drone.IsDrone {
		t.Fatal("SpawnDrone should create a facedown drone minion")
	}
	g.process(MinionDefeated{MinionID: drone.ID})
	if _, still := g.Minions[drone.ID]; still {
		t.Fatal("drone should leave play")
	}
	if len(g.Players[0].Discard) != 1 || g.Players[0].Discard[0].Code != "01088" {
		t.Fatalf("the facedown card should reach the owner's discard, got %v", g.Players[0].Discard)
	}
	if len(g.EncounterDiscard) != 0 {
		t.Fatalf("no encounter card should be discarded for a drone, got %v", g.EncounterDiscard)
	}
}

// TestChooseDiscardFromHandSeesDrawnCards: the discard question is built at
// processing time — cards drawn by messages queued ahead of it are
// selectable, and labels ride the translated m.discardCard key.
func TestChooseDiscardFromHandSeesDrawnCards(t *testing.T) {
	g := &Game{Players: []*Player{{ID: "player-1", Name: "P1"}}}
	p := g.Players[0]
	p.Deck = CardList{
		{ID: "card-1", Code: "01088", Owner: "player-1"},
		{ID: "card-2", Code: "01089", Owner: "player-1"},
	}
	p.Hand = CardList{{ID: "card-3", Code: "01090", Owner: "player-1"}}

	g.Push(DrawCards{Player: p.ID, N: 2})
	g.Push(ChooseDiscardFromHand{Player: p.ID, N: 1, Prompt: Tf("c.discardWhichCard")})
	g.Run()

	pq := g.Pending()
	if pq == nil {
		t.Fatal("the discard question should be pending")
	}
	if got := len(pq.Question.Choices); got != 3 {
		t.Fatalf("all 3 cards (1 kept + 2 drawn) must be selectable, got %d", got)
	}
	for _, c := range pq.Question.Choices {
		if c.Label.Key != "m.discardCard" {
			t.Fatalf("choice label should use the translated m.discardCard key, got %+v", c.Label)
		}
	}
	if pq.Question.Validate != "discardCost:1" {
		t.Fatalf("expected discardCost:1 validation, got %q", pq.Question.Validate)
	}
}

// TestNextIDRoundTripAndHeal: nextId persists across the JSON round trip,
// and a legacy save without it heals the counter above every live id (the
// entity-morphing save corruption).
func TestNextIDRoundTripAndHeal(t *testing.T) {
	g := &Game{Players: []*Player{{ID: "player-0", Name: "P1"}}}
	g.nextID = 12
	raw, err := g.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"nextId":12`) {
		t.Fatalf("nextId must persist: %s", raw)
	}
	loaded := &Game{}
	if err := loaded.UnmarshalJSON(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if loaded.nextID != 12 {
		t.Fatalf("nextId should round-trip, got %d", loaded.nextID)
	}

	// Legacy save: no nextId field, but an upgrade-7 exists → the counter
	// must heal above 7 before the next spawn can collide.
	legacy := `{"players":[{"id":"player-0","name":"P1"}],"upgrades":{"upgrade-7":{"id":"upgrade-7","code":"01008"}}}`
	old := &Game{}
	if err := old.UnmarshalJSON([]byte(legacy)); err != nil {
		t.Fatalf("legacy unmarshal: %v", err)
	}
	if old.nextID <= 7 {
		t.Fatalf("healed counter must exceed 7, got %d", old.nextID)
	}
	if id := old.nextEntityID(KindUpgrade); id.Num() <= 7 {
		t.Fatalf("new id must not collide with upgrade-7, got %s", id)
	}
}
