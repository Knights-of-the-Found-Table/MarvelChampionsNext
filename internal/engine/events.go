package engine

// Evt is a semantic game event streamed to clients for presentation
// (animations, sound). Events are transient: they are drained into the
// broadcast that follows the answer which produced them, never persisted
// with the game state.
//
// Only board-public information appears here (entity ids on the table and
// amounts); hidden information (hand cards, deck order) is never referenced,
// so the same event list is safe for every viewer.
type Evt struct {
	Type   string   `json:"type"` // damage | heal | threat | thwart | status
	Src    EntityID `json:"src,omitempty"`
	Dst    EntityID `json:"dst"`
	N      int      `json:"n,omitempty"`
	Status string   `json:"status,omitempty"`
	On     bool     `json:"on,omitempty"`
}

func (g *Game) emit(e Evt) {
	g.events = append(g.events, e)
}

// DrainEvents returns and clears the pending event batch.
func (g *Game) DrainEvents() []Evt {
	e := g.events
	g.events = nil
	return e
}
