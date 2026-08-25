package rooms

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	// Chat messages are public table talk, not game history: in-memory and
	// intentionally dropped on restart. Keep enough context for reconnects.
	chatHistoryLimit = 200
	chatTextLimit    = 300
)

// ChatMessage is one public table-talk line. It deliberately has no effect
// payload and never enters the deterministic engine log.
type ChatMessage struct {
	ID        int64  `json:"id"`
	At        int64  `json:"at"`
	UserID    string `json:"userId"`
	Name      string `json:"name"`
	Text      string `json:"text"`
	Spectator bool   `json:"spectator"`
}

// Chat appends a message for this room and broadcasts it to every viewer.
func (m *Manager) Chat(gameID int64, userID, name, text string) (ChatMessage, error) {
	r, err := m.Get(gameID)
	if err != nil {
		return ChatMessage{}, err
	}
	if text == "" {
		return ChatMessage{}, fmt.Errorf("empty message")
	}
	runes := []rune(text)
	if len(runes) > chatTextLimit {
		text = string(runes[:chatTextLimit])
	}

	r.mu.Lock()
	msg := ChatMessage{
		ID:        time.Now().UnixNano(),
		At:        time.Now().UnixMilli(),
		UserID:    userID,
		Name:      name,
		Text:      text,
		Spectator: !r.isParticipant(userID),
	}
	r.chat = append(r.chat, msg)
	if len(r.chat) > chatHistoryLimit {
		r.chat = r.chat[len(r.chat)-chatHistoryLimit:]
	}
	frame, err := json.Marshal(map[string]any{"type": "chat", "message": msg})
	if err == nil {
		for ch := range r.subs {
			select {
			case ch <- frame:
			default:
			}
		}
	}
	r.mu.Unlock()
	return msg, nil
}

// ChatHistory returns the retained table-talk log, oldest first.
func (m *Manager) ChatHistory(gameID int64) ([]ChatMessage, error) {
	r, err := m.Get(gameID)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ChatMessage(nil), r.chat...), nil
}
