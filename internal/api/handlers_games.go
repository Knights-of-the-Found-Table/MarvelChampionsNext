package api

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/http"

	"github.com/coder/websocket"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/store"
)

type createGameRequest struct {
	DeckIDs    []string `json:"deckIds"` // opaque deck tokens
	ScenarioID string   `json:"scenarioId"`
	Name       string   `json:"name"`
	Difficulty string   `json:"difficulty"`
}

func (s *Server) handleCreateGame(w http.ResponseWriter, r *http.Request) {
	var uid int64
	if _, err := fmt.Sscanf(userID(r), "%d", &uid); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return
	}
	var req createGameRequest
	if err := jsonDecode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(req.DeckIDs) == 0 || len(req.DeckIDs) > 4 {
		writeErr(w, http.StatusBadRequest, "1-4 decks required")
		return
	}
	if _, ok := engine.LookupScenario(req.ScenarioID); !ok {
		writeErr(w, http.StatusBadRequest, "unknown scenario")
		return
	}
	if req.Difficulty == "" {
		req.Difficulty = "standard"
	}

	// 玩家名用用户名（而非牌组名）：日志/HUD/棋盘上指代的是玩家本人。
	playerName := ""
	if u, err := s.Store.UserByID(uid); err == nil {
		playerName = u.Username
	}
	if playerName == "" {
		playerName = fmt.Sprintf("Player %d", uid)
	}

	specs := make([]engine.PlayerSpec, 0, len(req.DeckIDs))
	players := make([]store.GamePlayer, 0, len(req.DeckIDs))
	for i, deckToken := range req.DeckIDs {
		deck, err := s.Store.DeckByToken(deckToken)
		if err != nil {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("deck %s not found", deckToken))
			return
		}
		heroBase := engine.BaseCodeOf(deck.InvestigatorCode)
		if !engine.Implemented(heroBase + "a") {
			writeErr(w, http.StatusBadRequest, "hero not implemented yet: "+heroBase)
			return
		}
		specs = append(specs, engine.PlayerSpec{
			Name:     playerName,
			UserID:   fmt.Sprint(uid),
			HeroBase: heroBase,
			Deck:     deck.Slots,
		})
		players = append(players, store.GamePlayer{
			Slot:     i,
			UserID:   &uid,
			DeckID:   deck.ID,
			HeroBase: heroBase,
		})
	}

	var seedBuf [8]byte
	_, _ = rand.Read(seedBuf[:])
	seed := int64(binary.LittleEndian.Uint64(seedBuf[:]))

	g, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: req.ScenarioID,
		Difficulty: req.Difficulty,
		Players:    specs,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	state, err := g.MarshalJSON()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "serialize failed")
		return
	}
	if req.Name == "" {
		req.Name = engine.LookupScenarioName(req.ScenarioID)
	}
	gameID, err := s.Store.CreateGame(req.Name, req.ScenarioID, req.Difficulty, seed, string(state), players)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	view, err := s.Rooms.View(gameID, fmt.Sprint(uid))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "room failed")
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (s *Server) handleListGames(w http.ResponseWriter, r *http.Request) {
	games, err := s.Store.ListGames()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	type gameList struct {
		ID         string `json:"id"` // opaque public token
		Name       string `json:"name"`
		ScenarioID string `json:"scenarioId"`
		Status     string `json:"status"`
		UpdatedAt  string `json:"updatedAt"`
	}
	out := make([]gameList, 0, len(games))
	for _, g := range games {
		out = append(out, gameList{ID: g.Token, Name: g.Name, ScenarioID: g.ScenarioID, Status: g.Status, UpdatedAt: g.UpdatedAt})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGameView(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathGame(w, r)
	if !ok {
		return
	}
	view, err := s.Rooms.View(gameID, userID(r))
	if err != nil {
		writeErr(w, http.StatusNotFound, "game not found")
		return
	}
	writeJSON(w, http.StatusOK, view)
}

type joinRequest struct {
	Slot int `json:"slot"`
}

func (s *Server) handleJoinGame(w http.ResponseWriter, r *http.Request) {
	var uid int64
	if _, err := fmt.Sscanf(userID(r), "%d", &uid); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return
	}
	gameID, ok := s.pathGame(w, r)
	if !ok {
		return
	}
	var req joinRequest
	_ = jsonDecode(r, &req)
	players, err := s.Store.GamePlayers(gameID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "game not found")
		return
	}
	slot := req.Slot
	if slot == 0 {
		for _, p := range players {
			if p.UserID == nil {
				slot = p.Slot
				break
			}
		}
	}
	if err := s.Store.ClaimPlayerSlot(gameID, slot, uid); err != nil {
		writeErr(w, http.StatusConflict, "slot not available")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"joined": slot})
}

type answerRequest struct {
	Paths []string `json:"paths"`
}

func (s *Server) handleAnswer(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathGame(w, r)
	if !ok {
		return
	}
	var req answerRequest
	if err := jsonDecode(r, &req); err != nil || len(req.Paths) == 0 {
		writeErr(w, http.StatusBadRequest, "invalid answer")
		return
	}
	if err := s.Rooms.Answer(gameID, userID(r), req.Paths); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	view, err := s.Rooms.View(gameID, userID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "view failed")
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleUndo(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathGame(w, r)
	if !ok {
		return
	}
	if err := s.Rooms.Undo(gameID, userID(r)); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	view, err := s.Rooms.View(gameID, userID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "view failed")
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleReplay(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathGame(w, r)
	if !ok {
		return
	}
	row, err := s.Store.GameByID(gameID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "game not found")
		return
	}
	actions, err := s.Store.GameActions(gameID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	players, err := s.Store.GamePlayers(gameID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	// 脱敏：数字 deckId 不出网，换成牌组的公开 token。
	type replayPlayer struct {
		Slot     int    `json:"slot"`
		UserID   *int64 `json:"userId,omitempty"`
		Deck     string `json:"deck"`
		HeroBase string `json:"heroBase"`
	}
	outPlayers := make([]replayPlayer, 0, len(players))
	for _, p := range players {
		rp := replayPlayer{Slot: p.Slot, UserID: p.UserID, HeroBase: p.HeroBase}
		if d, err := s.Store.DeckByID(p.DeckID); err == nil {
			rp.Deck = d.Token
		}
		outPlayers = append(outPlayers, rp)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"seed":       row.Seed,
		"scenarioId": row.ScenarioID,
		"difficulty": row.Difficulty,
		"players":    outPlayers,
		"actions":    actions,
	})
}

// handleStream upgrades to WebSocket and streams state updates.
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathGame(w, r)
	if !ok {
		return
	}
	// Viewer identity: valid token optional (spectators allowed).
	viewer := ""
	if tokenStr := bearerToken(r); tokenStr != "" {
		if uid, ok := s.parseSubject(tokenStr); ok {
			viewer = uid
		}
	}
	ch, cancel, err := s.Rooms.Subscribe(gameID, viewer)
	if err != nil {
		writeErr(w, http.StatusNotFound, "game not found")
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		cancel()
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "bye")
	ctx := r.Context()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		case <-done:
			cancel()
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
				cancel()
				return
			}
		}
	}
}

// handleChatHistory returns the room's recent public table-talk messages.
func (s *Server) handleChatHistory(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathGame(w, r)
	if !ok {
		return
	}
	messages, err := s.Rooms.ChatHistory(gameID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "game not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": messages})
}

// handleChatSend appends an authenticated message and broadcasts it through
// the existing per-game WebSocket stream. Chat never changes game state, so
// it bypasses answers, persistence, and undo snapshots by design.
func (s *Server) handleChatSend(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathGame(w, r)
	if !ok {
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := jsonDecode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	userID := userID(r)
	var uid int64
	if _, err := fmt.Sscanf(userID, "%d", &uid); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return
	}
	user, err := s.Store.UserByID(uid)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return
	}
	msg, err := s.Rooms.Chat(gameID, userID, user.Username, req.Text)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": msg})
}

// handlePileList serves the contents of a deck or discard pile for the pile
// viewer. Deck listings are shuffled server-side (see rooms.PileList), so
// the draw order is never revealed.
func (s *Server) handlePileList(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathGame(w, r)
	if !ok {
		return
	}
	player := r.URL.Query().Get("player")
	pile := r.URL.Query().Get("pile")
	cards, err := s.Rooms.PileList(gameID, player, pile)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"pile":   pile,
		"player": player,
		"cards":  cards,
	})
}
