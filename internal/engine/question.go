package engine

import (
	"encoding/json"
	"fmt"
)

// Question models a prompt shown to a player. The client renders the choice
// tree as buttons; the player's answer is a choice ID path (e.g. "0", "1.2").
//
// Choice effects are plain message payloads, never closures, so pending
// questions survive serialization exactly like the reference design.
type Question struct {
	Type    string   `json:"type"` // choose_one | choose_n | choose_player_order
	Prompt  string   `json:"prompt,omitempty"`
	Choices []Choice `json:"choices"`
	N       int      `json:"n,omitempty"` // for choose_n: number to pick

	// Validate optionally names a server-side validation rule for the
	// selection, e.g. "payment:3" (total resource icons >= 3).
	Validate string `json:"validate,omitempty"`
	// Context carries arbitrary JSON data needed to resolve validated
	// selections (payment target card/ability...).
	Context map[string]any `json:"context,omitempty"`
}

// Choice kinds drive frontend styling.
const (
	ChoiceLabel      = "label"       // plain button
	ChoiceCard       = "card"        // card-backed button (hand card etc.)
	ChoicePlay       = "play"        // play a card from hand
	ChoiceAbility    = "ability"     // activated ability of an entity
	ChoiceBasicPower = "basic_power" // attack / thwart / recover / defend
	ChoiceEndTurn    = "end_turn"
	ChoicePass       = "pass"
	ChoiceForm       = "form" // change form
	ChoiceTarget     = "target"
	ChoiceResource   = "resource" // select cards to pay
)

type Choice struct {
	ID       string    `json:"id"`
	Label    string    `json:"label"`
	Kind     string    `json:"kind"`
	CardCode string    `json:"cardCode,omitempty"`
	SourceID EntityID  `json:"sourceId,omitempty"`
	Disabled bool      `json:"disabled,omitempty"`
	Then     *Question `json:"then,omitempty"`

	// msgs are enqueued when this leaf choice is picked.
	msgs []Message `json:"-"`
}

// MarshalJSON encodes the effect payload via the message envelope codec so
// pending questions survive persistence.
func (c Choice) MarshalJSON() ([]byte, error) {
	type alias Choice
	out := struct {
		*alias
		Msgs []msgEnvelope `json:"msgs,omitempty"`
	}{alias: (*alias)(&c)}
	var err error
	out.Msgs, err = marshalMessages(c.msgs)
	if err != nil {
		return nil, err
	}
	return json.Marshal(out)
}

// UnmarshalJSON restores the effect payload.
func (c *Choice) UnmarshalJSON(b []byte) error {
	type alias Choice
	in := struct {
		*alias
		Msgs []msgEnvelope `json:"msgs"`
	}{alias: (*alias)(c)}
	if err := json.Unmarshal(b, &in); err != nil {
		return err
	}
	msgs, err := unmarshalMessages(in.Msgs)
	if err != nil {
		return err
	}
	c.msgs = msgs
	return nil
}

// Msgs attaches the effect payload for a leaf choice.
func (c Choice) Msgs(msgs ...Message) Choice {
	c.msgs = msgs
	return c
}

// WithThen attaches a follow-up question under this choice. The subtree is
// deep-copied so each branch owns an independent id namespace: several
// choices may chain into the same question object (attackQuestion's shared
// defense prompt) without inheriting each other's prefixes. The subtree's
// choice ids are cleared so the root question reassigns them with the full
// answer-path prefix ("2.0"): ids assigned while the subtree was built
// standalone (its own assignIDs("") call) would collide with root-level ids
// and answers referencing them would resolve to the wrong root choice.
func (c Choice) WithThen(q *Question) Choice {
	c.Then = copyQuestion(q)
	clearChoiceIDs(c.Then)
	return c
}

// copyQuestion deep-copies a question tree. msg payloads are shared
// read-only (they are immutable data, and Choice.msgs already round-trips
// through the envelope codec when persisted). A manual copy is used instead
// of a JSON round-trip because Question.Context may hold non-string values
// whose types would drift (int → float64) through JSON.
func copyQuestion(q *Question) *Question {
	if q == nil {
		return nil
	}
	out := &Question{
		Type:     q.Type,
		Prompt:   q.Prompt,
		Choices:  make([]Choice, len(q.Choices)),
		N:        q.N,
		Validate: q.Validate,
	}
	if q.Context != nil {
		out.Context = make(map[string]any, len(q.Context))
		for k, v := range q.Context {
			out.Context[k] = v
		}
	}
	for i := range q.Choices {
		ch := q.Choices[i]
		ch.Then = copyQuestion(q.Choices[i].Then)
		out.Choices[i] = ch
	}
	return out
}

func clearChoiceIDs(q *Question) {
	for i := range q.Choices {
		q.Choices[i].ID = ""
		if q.Choices[i].Then != nil {
			clearChoiceIDs(q.Choices[i].Then)
		}
	}
}

// Ask builds a choose_one question.
func Ask(prompt string, choices ...Choice) *Question {
	q := &Question{Type: "choose_one", Prompt: prompt, Choices: choices}
	q.assignIDs("")
	return q
}

// AskN builds a choose_n question.
func AskN(prompt string, n int, choices ...Choice) *Question {
	q := &Question{Type: "choose_n", Prompt: prompt, Choices: choices, N: n}
	q.assignIDs("")
	return q
}

func (q *Question) assignIDs(prefix string) {
	for i := range q.Choices {
		c := &q.Choices[i]
		if c.ID == "" {
			c.ID = pathJoin(prefix, fmt.Sprint(i))
		}
		if c.Then != nil {
			c.Then.assignIDs(c.ID)
		}
	}
}

func pathJoin(prefix, leaf string) string {
	if prefix == "" {
		return leaf
	}
	return prefix + "." + leaf
}

// Leaf resolves an answer path to the innermost chosen choice. Choice ids
// are full answer paths ("basic-attack.0"), assigned by the root question's
// assignIDs, so recursion passes the path down unchanged — subtree ids carry
// their own prefix.
func (q *Question) Leaf(answer string) (*Choice, error) {
	for i := range q.Choices {
		c := &q.Choices[i]
		if answer == c.ID {
			return c, nil
		}
		if pathHasPrefix(answer, c.ID) {
			if c.Then == nil {
				return nil, fmt.Errorf("answer %q descends past a leaf choice", answer)
			}
			return c.Then.Leaf(answer)
		}
	}
	return nil, fmt.Errorf("invalid answer %q for question", answer)
}

// Chain resolves an answer path to the sequence of choices from the root to
// the answered choice (the leaf last). Answering "interrupt.defend" must
// fire both the interrupt choice's messages and the defense leaf's, so
// callers concatenate the chain's msgs in order.
func (q *Question) Chain(answer string) ([]*Choice, error) {
	var chain []*Choice
	cur := q
	rest := answer
	for cur != nil {
		found := false
		for i := range cur.Choices {
			c := &cur.Choices[i]
			if rest == c.ID {
				return append(chain, c), nil
			}
			if pathHasPrefix(rest, c.ID) {
				if c.Then == nil {
					return nil, fmt.Errorf("answer %q descends past a leaf choice", answer)
				}
				chain = append(chain, c)
				cur = c.Then
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("invalid answer %q for question", answer)
		}
	}
	return chain, nil
}

// idsUnique reports whether every choice id in the tree occurs exactly
// once. Questions persisted before WithThen copied subtrees carry the same
// ids under several branches (the shared-subtree defect); answering those
// falls back to leaf-only semantics instead of risking another branch's
// messages firing.
func (q *Question) idsUnique() bool {
	seen := map[string]bool{}
	var walk func(q *Question) bool
	walk = func(q *Question) bool {
		for i := range q.Choices {
			c := &q.Choices[i]
			if c.ID != "" {
				if seen[c.ID] {
					return false
				}
				seen[c.ID] = true
			}
			if c.Then != nil && !walk(c.Then) {
				return false
			}
		}
		return true
	}
	return walk(q)
}

// Selective returns the choices matching the given answer paths (for
// choose_n questions).
func (q *Question) Selective(paths []string) ([]*Choice, error) {
	var out []*Choice
	for _, p := range paths {
		c, err := q.Leaf(p)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty selection")
	}
	return out, nil
}

func pathHasPrefix(path, prefix string) bool {
	return path == prefix || len(path) > len(prefix) && path[len(prefix)] == '.' && path[:len(prefix)] == prefix
}

// legacySubtreeIDs reports whether any nested Then-question carries choice
// ids without the parent-path prefix ("0" under choice "basic-attack").
// Questions persisted before ids were assigned root-relative have this shape
// and cannot be answered by the path protocol anymore.
func legacySubtreeIDs(q *Question) bool {
	for i := range q.Choices {
		c := &q.Choices[i]
		if c.Then == nil {
			continue
		}
		for j := range c.Then.Choices {
			sc := &c.Then.Choices[j]
			if sc.ID != "" && sc.ID != c.ID && !pathHasPrefix(sc.ID, c.ID) {
				return true
			}
		}
		if legacySubtreeIDs(c.Then) {
			return true
		}
	}
	return false
}

// RebuildTurnMenu replaces a pending "Your turn" question whose subtree ids
// predate the root-relative id scheme. TurnMenu is a pure function of state,
// so rebuilding is safe and only changes presentation (ids), not semantics.
func (g *Game) RebuildTurnMenu() bool {
	pq := g.Pending()
	if pq == nil || pq.Question.Prompt != "Your turn" || !legacySubtreeIDs(pq.Question) {
		return false
	}
	p := g.Player(pq.Player)
	if p == nil {
		return false
	}
	g.pending = &PendingQuestion{Player: pq.Player, Question: g.TurnMenu(p)}
	return true
}
