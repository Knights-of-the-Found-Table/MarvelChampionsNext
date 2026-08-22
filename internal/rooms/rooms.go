package rooms

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/store"
)

// Room holds an active game and its subscribers.
type Room struct {
	ID   int64
	Name string

	mu   sync.Mutex
	game *engine.Game
	// owners maps player entity id -> user id ("").
	owners map[string]string
	subs   map[chan []byte]string // channel -> viewer user id ("" = spectator)
}

type Manager struct {
	store *store.Store
	mu    sync.Mutex
	rooms map[int64]*Room
}

func NewManager(s *store.Store) *Manager {
	return &Manager{store: s, rooms: map[int64]*Room{}}
}

// Get loads (or activates) the room for a game id.
func (m *Manager) Get(gameID int64) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.rooms[gameID]; ok {
		return r, nil
	}
	row, err := m.store.GameByID(gameID)
	if err != nil {
		return nil, err
	}
	g := &engine.Game{}
	if err := g.UnmarshalJSON([]byte(row.State)); err != nil {
		return nil, fmt.Errorf("decode game state: %w", err)
	}
	players, err := m.store.GamePlayers(gameID)
	if err != nil {
		return nil, err
	}
	owners := map[string]string{}
	for i, p := range players {
		if p.UserID != nil && i < len(g.Players) {
			owners[string(g.Players[i].ID)] = fmt.Sprint(*p.UserID)
		}
	}
	r := &Room{ID: gameID, Name: row.Name, game: g, owners: owners, subs: map[chan []byte]string{}}
	m.rooms[gameID] = r
	return r, nil
}

// Snapshot returns a JSON snapshot of the current state (for persistence).
func (r *Room) Snapshot() (string, error) {
	b, err := r.game.MarshalJSON()
	return string(b), err
}

// Answer applies a player answer after authorization.
func (m *Manager) Answer(gameID int64, userID string, paths []string) error {
	r, err := m.Get(gameID)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	pq := r.game.Pending()
	if pq == nil {
		return fmt.Errorf("no pending question")
	}
	if r.owners[string(pq.Player)] != userID {
		return fmt.Errorf("not your question")
	}
	// snapshot for undo before mutating
	snap, err := r.Snapshot()
	if err == nil {
		_ = m.store.PushSnapshot(gameID, snap)
	}
	playerStr := string(pq.Player)
	if err := r.game.Answer(pq.Player, paths); err != nil {
		return err
	}
	_ = m.store.RecordAction(gameID, playerStr, paths)
	m.persist(r)
	r.broadcastEvents(r.game.DrainEvents())
	return nil
}

// Undo restores the latest snapshot.
func (m *Manager) Undo(gameID int64, userID string) error {
	r, err := m.Get(gameID)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	// only let participants undo
	if !r.isParticipant(userID) {
		return fmt.Errorf("not a participant")
	}
	state, err := m.store.PopSnapshot(gameID)
	if err != nil {
		return fmt.Errorf("nothing to undo")
	}
	g := &engine.Game{}
	if err := g.UnmarshalJSON([]byte(state)); err != nil {
		return fmt.Errorf("decode snapshot: %w", err)
	}
	r.game = g
	m.persist(r)
	r.broadcast()
	return nil
}

func (r *Room) isParticipant(userID string) bool {
	if userID == "" {
		return false
	}
	for _, u := range r.owners {
		if u == userID {
			return true
		}
	}
	return false
}

func (m *Manager) persist(r *Room) {
	state, err := r.Snapshot()
	if err != nil {
		slog.Error("rooms: snapshot game", "game", r.ID, "error", err)
		return
	}
	status := "active"
	if r.game.Over {
		status = "finished"
	}
	if err := m.store.SaveGameState(r.ID, state, status); err != nil {
		slog.Error("rooms: persist game", "game", r.ID, "error", err)
	}
}

// Subscribe registers a viewer channel; it immediately receives the current
// view. The returned cancel function unsubscribes.
func (m *Manager) Subscribe(gameID int64, viewerUserID string) (chan []byte, func(), error) {
	r, err := m.Get(gameID)
	if err != nil {
		return nil, nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan []byte, 16)
	r.subs[ch] = viewerUserID
	cancel := func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if _, ok := r.subs[ch]; ok {
			delete(r.subs, ch)
			close(ch)
		}
	}
	// initial view
	if v := BuildView(r.ID, r.Name, r.game, viewerUserID, r.owners); v != nil {
		if b, err := json.Marshal(map[string]any{"type": "state", "view": v}); err == nil {
			select {
			case ch <- b:
			default:
			}
		}
	}
	return ch, cancel, nil
}

// View builds the current view for a user (spectator when userID empty).
func (m *Manager) View(gameID int64, userID string) (*GameView, error) {
	r, err := m.Get(gameID)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return BuildView(r.ID, r.Name, r.game, userID, r.owners), nil
}

func (r *Room) broadcast() {
	for ch, viewer := range r.subs {
		v := BuildView(r.ID, r.Name, r.game, viewer, r.owners)
		b, err := json.Marshal(map[string]any{"type": "state", "view": v})
		if err != nil {
			continue
		}
		select {
		case ch <- b:
		default: // slow viewer; drop rather than block the game
		}
	}
}

// broadcastEvents sends the semantic event batch produced by the answer that
// just resolved, then a plain state frame. Events carry only board-public
// entity ids, so every viewer gets the same list.
func (r *Room) broadcastEvents(events []engine.Evt) {
	if len(events) == 0 {
		r.broadcast()
		return
	}
	b, err := json.Marshal(map[string]any{"type": "events", "events": events})
	if err == nil {
		for ch := range r.subs {
			select {
			case ch <- b:
			default:
			}
		}
	}
	r.broadcast()
}
