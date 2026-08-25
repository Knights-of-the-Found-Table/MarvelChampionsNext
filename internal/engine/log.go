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

// LogEntry is one line of the in-game event journal. Key/Args carry the
// structured, language-neutral form the client renders in the viewer's
// locale; Text is the canonical English rendering (also the fallback for
// entries written before the catalog existed, and for plain logf calls).
type LogEntry struct {
	Level string `json:"level"`
	Key   string `json:"key,omitempty"`
	Args  []Arg  `json:"args,omitempty"`
	Text  string `json:"text"`
}

// LogEntries is the game journal. It unmarshals both the current form
// ([{"level":"info","text":"..."}]) and the legacy plain-string form
// (["..."], treated as info) so pre-upgrade saves and undo snapshots load.
// Legacy English-only entries keep their text verbatim; they are never
// re-translated at runtime (see the i18n notes in i18n.go).
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
	type entry LogEntry
	var entries []entry
	if err := json.Unmarshal(b, &entries); err != nil {
		return err
	}
	out := make(LogEntries, len(entries))
	for i, e := range entries {
		if e.Level == "" {
			e.Level = LogInfo
		}
		out[i] = LogEntry(e)
	}
	*l = out
	return nil
}

func (g *Game) addLog(level, text string) {
	g.appendLog(LogEntry{Level: level, Text: text})
}

func (g *Game) appendLog(e LogEntry) {
	g.Log = append(g.Log, e)
	if len(g.Log) > maxLogEntries {
		g.Log = g.Log[len(g.Log)-maxLogEntries:]
	}
}

// logf appends an unkeyed log line from a raw format string. Only for
// not-yet-translated legacy call sites; new code uses the tlog family with a
// catalog key (see i18n.go).
func (g *Game) logf(format string, args ...any) {
	g.addLog(LogInfo, fmt.Sprintf(format, args...))
}

func (g *Game) logMinorf(format string, args ...any) {
	g.addLog(LogMinor, fmt.Sprintf(format, args...))
}

func (g *Game) logMajorf(format string, args ...any) {
	g.addLog(LogMajor, fmt.Sprintf(format, args...))
}

// tlogf/tlogMinorf/tlogMajorf log a catalog-keyed line, storing the
// structured args so the client can render them per locale.
func (g *Game) tlogf(key string, args ...any) {
	g.appendTLog(LogInfo, key, args...)
}

func (g *Game) tlogMinorf(key string, args ...any) {
	g.appendTLog(LogMinor, key, args...)
}

func (g *Game) tlogMajorf(key string, args ...any) {
	g.appendTLog(LogMajor, key, args...)
}

func (g *Game) appendTLog(level, key string, args ...any) {
	e := LogEntry{Level: level, Key: key}
	if len(args) == 0 {
		e.Text = msgFormat(key)
		g.appendLog(e)
		return
	}
	e.Args = make([]Arg, len(args))
	disp := make([]any, len(args))
	for i, a := range args {
		e.Args[i] = Argify(a)
		disp[i] = e.Args[i].value()
	}
	e.Text = fmt.Sprintf(msgFormat(key), disp...)
	g.appendLog(e)
}

// LogText joins the journal's text lines (tests, debugging).
func (g *Game) LogText() string {
	lines := make([]string, len(g.Log))
	for i, e := range g.Log {
		lines[i] = e.Text
	}
	return strings.Join(lines, "\n")
}
