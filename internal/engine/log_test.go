package engine

import (
	"encoding/json"
	"strings"
	"testing"
)

// Legacy saves stored the journal as plain strings; loading one must map
// every line to an info-level entry.
func TestLogEntriesUnmarshalLegacyStrings(t *testing.T) {
	var log LogEntries
	if err := json.Unmarshal([]byte(`["Captain America plays Shield Block","Rhino attacks"]`), &log); err != nil {
		t.Fatalf("unmarshal legacy log: %v", err)
	}
	if len(log) != 2 {
		t.Fatalf("len = %d, want 2", len(log))
	}
	for _, e := range log {
		if e.Level != LogInfo {
			t.Errorf("entry %q level = %q, want %q", e.Text, e.Level, LogInfo)
		}
	}
	if log[1].Text != "Rhino attacks" {
		t.Errorf("text = %q, want %q", log[1].Text, "Rhino attacks")
	}
}

func TestLogEntriesUnmarshalEntries(t *testing.T) {
	var log LogEntries
	if err := json.Unmarshal([]byte(`[{"level":"major","text":"KO'd!"},{"text":"no level given"}]`), &log); err != nil {
		t.Fatalf("unmarshal entries: %v", err)
	}
	if log[0].Level != LogMajor || log[0].Text != "KO'd!" {
		t.Errorf("entry 0 = %+v", log[0])
	}
	if log[1].Level != LogInfo {
		t.Errorf("missing level should default to info, got %q", log[1].Level)
	}
}

// A full legacy game JSON (with a string-array log) must round-trip through
// UnmarshalJSON like a saved game load.
func TestGameUnmarshalLegacyLog(t *testing.T) {
	legacy := `{"seed":1,"scenarioId":"rhino","players":[{"id":"player-1"}],"log":["── Round 1 ──"],"villains":{},"minions":{}}`
	g := &Game{}
	if err := g.UnmarshalJSON([]byte(legacy)); err != nil {
		t.Fatalf("unmarshal legacy game: %v", err)
	}
	if len(g.Log) != 1 || g.Log[0].Level != LogInfo || g.Log[0].Text != "── Round 1 ──" {
		t.Fatalf("log = %+v, want one info entry", g.Log)
	}
	out, err := g.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"log":[{"level":"info","text":"── Round 1 ──"}]`) {
		t.Errorf("re-marshaled log not in entry form: %s", string(out))
	}
}
