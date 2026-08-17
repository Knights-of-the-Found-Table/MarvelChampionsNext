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

// WithThen attaches a follow-up question under this choice.
func (c Choice) WithThen(q *Question) Choice {
	c.Then = q
	return c
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

// Leaf resolves an answer path to the innermost chosen choice.
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
			return c.Then.Leaf(pathSuffix(answer, c.ID))
		}
	}
	return nil, fmt.Errorf("invalid answer %q for question", answer)
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

func pathSuffix(path, prefix string) string {
	if len(path) > len(prefix) && path[:len(prefix)] == prefix {
		return path[len(prefix)+1:]
	}
	return ""
}
