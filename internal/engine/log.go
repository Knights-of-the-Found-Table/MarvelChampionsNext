package engine

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Log entry levels. Info is the default for routine events (plays, damage,
// attacks); major marks pivotal moments (KO, villain defeated, scheme stage
// flips, round changes, victory/defeat); minor marks bookkeeping noise
// (counter ticks, discounts, shuffles).
const (
	LogMinor = "minor"
	LogInfo  = "info"
	LogMajor = "major"
)

// maxLogEntries caps the in-game journal; oldest entries are dropped.
const maxLogEntries = 500

// LogEntry is one line of the in-game event journal.
type LogEntry struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

// LogEntries is the game journal. It unmarshals both the current form
// ([{"level":"info","text":"..."}]) and the legacy plain-string form
// (["..."], treated as info) so pre-upgrade saves and undo snapshots load.
type LogEntries []LogEntry

func (l *LogEntries) UnmarshalJSON(b []byte) error {
	var legacy []string
	if err := json.Unmarshal(b, &legacy); err == nil {
		out := make(LogEntries, len(legacy))
		for i, s := range legacy {
			out[i] = LogEntry{Level: LogInfo, Text: s}
		}
		*l = out
		return nil
	}
	var entries []LogEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return err
	}
	for i := range entries {
		if entries[i].Level == "" {
			entries[i].Level = LogInfo
		}
	}
	*l = entries
	return nil
}

func (g *Game) addLog(level, text string) {
	g.Log = append(g.Log, LogEntry{Level: level, Text: text})
	if len(g.Log) > maxLogEntries {
		g.Log = g.Log[len(g.Log)-maxLogEntries:]
	}
}

func (g *Game) logf(format string, args ...any) {
	g.addLog(LogInfo, fmt.Sprintf(format, args...))
}

func (g *Game) logMinorf(format string, args ...any) {
	g.addLog(LogMinor, fmt.Sprintf(format, args...))
}

func (g *Game) logMajorf(format string, args ...any) {
	g.addLog(LogMajor, fmt.Sprintf(format, args...))
}

// tlogf/tlogMinorf/tlogMajorf log a message-catalog entry (i18n): the key
// is resolved and formatted by Tf in the current UI language. Plain logf
// remains for ad-hoc strings (card-package text not yet migrated).
func (g *Game) tlogf(key string, args ...any) {
	g.addLog(LogInfo, Tf(key, args...))
}

func (g *Game) tlogMinorf(key string, args ...any) {
	g.addLog(LogMinor, Tf(key, args...))
}

func (g *Game) tlogMajorf(key string, args ...any) {
	g.addLog(LogMajor, Tf(key, args...))
}

// LogText joins the journal's text lines (tests, debugging).
func (g *Game) LogText() string {
	lines := make([]string, len(g.Log))
	for i, e := range g.Log {
		lines[i] = e.Text
	}
	return strings.Join(lines, "\n")
}
