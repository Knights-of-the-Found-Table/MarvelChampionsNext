package engine

import (
	"fmt"
	"strconv"
	"strings"
)

// EntityID identifies any entity in play. String form is "<kind>-<n>", e.g.
// "villain-1", "ally-3", "player-0", "scheme-2".
type EntityID string

const (
	KindPlayer     = "player"
	KindVillain    = "villain"
	KindMinion     = "minion"
	KindAlly       = "ally"
	KindSupport    = "support"
	KindUpgrade    = "upgrade"
	KindAttachment = "attachment"
	KindTreachery  = "treachery"
	KindSideScheme = "sidescheme"
	KindMainScheme = "mainscheme"
	KindEnvironment = "environment"
)

func NewEntityID(kind string, n int) EntityID {
	return EntityID(fmt.Sprintf("%s-%d", kind, n))
}

// Kind returns the entity kind portion of the id.
func (id EntityID) Kind() string {
	if i := strings.LastIndexByte(string(id), '-'); i > 0 {
		return string(id)[:i]
	}
	return string(id)
}

// Num returns the numeric portion of the id (0 when malformed).
func (id EntityID) Num() int {
	i := strings.LastIndexByte(string(id), '-')
	if i < 0 || i == len(id)-1 {
		return 0
	}
	n, _ := strconv.Atoi(string(id)[i+1:])
	return n
}

func (id EntityID) String() string { return string(id) }

// Is reports whether the id has the given kind.
func (id EntityID) Is(kind string) bool { return id.Kind() == kind }

// PlayerID is an EntityID of kind player; alias for readability.
type PlayerID = EntityID
