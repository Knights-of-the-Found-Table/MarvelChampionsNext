package engine

import "testing"

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
