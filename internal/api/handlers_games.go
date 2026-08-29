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
	DeckIDs     []string `json:"deckIds"` // solo games only: exactly one, the creator's
	ScenarioID  string   `json:"scenarioId"`
	Name        string   `json:"name"`
	Difficulty  string   `json:"difficulty"`
	PlayerCount int      `json:"playerCount"`
}

// deckHero validates the deck's hero is implemented and returns its base code.
func deckHero(deck *store.Deck) (string, error) {
	heroBase := engine.BaseCodeOf(deck.InvestigatorCode)
	if !engine.Implemented(heroBase + "a") {
		return "", fmt.Errorf("hero not implemented yet: %s", heroBase)
	}
	return heroBase, nil
}

// checkDeckPlayable refuses to seat a rulebook-illegal deck: illegal decks
// import fine (for viewing / future editing) but never start games. The
// structured issues ride along in the body so the UI can explain why.
func checkDeckPlayable(w http.ResponseWriter, deck *store.Deck) bool {
	if issues := engine.ValidateDeck(deck.InvestigatorCode, deck.Slots); len(issues) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":      "deck is not legal for play",
			"deckIssues": issues,
		})
		return false
	}
	return true
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
	if req.PlayerCount == 0 {
		req.PlayerCount = 1
	}
	if req.PlayerCount < 1 || req.PlayerCount > 4 {
		writeErr(w, http.StatusBadRequest, "playerCount must be 1-4")
		return
	}
	if _, ok := engine.LookupScenario(req.ScenarioID); !ok {
		writeErr(w, http.StatusBadRequest, "unknown scenario")
		return
	}
	if req.Difficulty == "" {
		req.Difficulty = "standard"
	}

	if req.PlayerCount > 1 {
		// 多人对局：创建者只定剧本，进大厅后各玩家（含房主）自选牌组。
		if len(req.DeckIDs) > 0 {
			writeErr(w, http.StatusBadRequest, "multiplayer lobbies pick decks per player")
			return
		}
		if req.Name == "" {
			req.Name = engine.LookupScenarioName(req.ScenarioID)
		}
		gameID, err := s.Store.CreateGame(req.Name, req.ScenarioID, req.Difficulty, 0, "", "lobby", uid, req.PlayerCount, nil)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "store failed")
			return
		}
		g, err := s.Store.GameByID(gameID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "store failed")
			return
		}
		payload, err := s.lobbyPayload(g)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "store failed")
			return
		}
		writeJSON(w, http.StatusCreated, payload)
		return
	}

	if len(req.DeckIDs) != 1 {
		writeErr(w, http.StatusBadRequest, "solo games need exactly one deck")
		return
	}
	// 玩家名用用户名（而非牌组名）：日志/HUD/棋盘上指代的是玩家本人。
	playerName := ""
	if u, err := s.Store.UserByID(uid); err == nil {
		playerName = u.Username
	}
	if playerName == "" {
		playerName = fmt.Sprintf("Player %d", uid)
	}
	deck, err := s.Store.DeckByToken(req.DeckIDs[0])
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("deck %s not found", req.DeckIDs[0]))
		return
	}
	heroBase, err := deckHero(deck)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !checkDeckPlayable(w, deck) {
		return
	}
	specs := []engine.PlayerSpec{{
		Name: playerName, UserID: fmt.Sprint(uid), HeroBase: heroBase, Deck: deck.Slots,
	}}
	players := []store.GamePlayer{{
		Slot: 0, UserID: &uid, DeckID: deck.ID, HeroBase: heroBase,
	}}

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
	gameID, err := s.Store.CreateGame(req.Name, req.ScenarioID, req.Difficulty, seed, string(state), "active", uid, 1, players)
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

// lobbyLobbyPlayer / lobbyPayload compose the invite-screen projection of a
// game waiting in the lobby. The host always heads the player list (with a
// synthesized entry before picking a deck).
type lobbyPlayer struct {
	UserID   string     `json:"userId"`
	Username string     `json:"username"`
	Slot     int        `json:"slot"`
	Host     bool       `json:"host"`
	Deck     *lobbyDeck `json:"deck"`
}

type lobbyDeck struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	HeroCode string `json:"heroCode"`
	// 组牌校验结果：非法牌组已在入座时被拒，这里供大厅界面给仍持有
	// 引用的旧行打警告标。
	Valid bool `json:"valid"`
}

func (s *Server) lobbyDeckView(deckID int64) *lobbyDeck {
	d, err := s.Store.DeckByID(deckID)
	if err != nil {
		return nil
	}
	return &lobbyDeck{
		ID: d.Token, Name: d.Name, HeroCode: d.InvestigatorCode,
		Valid: len(engine.ValidateDeck(d.InvestigatorCode, d.Slots)) == 0,
	}
}

func (s *Server) lobbyPayload(g *store.GameRow) (map[string]any, error) {
	players, err := s.Store.GamePlayers(g.ID)
	if err != nil {
		return nil, err
	}
	out := make([]lobbyPlayer, 0, g.PlayerCount)
	hostRowSeen := false
	for _, p := range players {
		lp := lobbyPlayer{Slot: p.Slot, Host: p.Slot == 0}
		if p.Slot == 0 {
			hostRowSeen = true
		}
		if p.UserID != nil {
			lp.UserID = fmt.Sprint(*p.UserID)
			if u, err := s.Store.UserByID(*p.UserID); err == nil {
				lp.Username = u.Username
			}
		}
		lp.Deck = s.lobbyDeckView(p.DeckID)
		out = append(out, lp)
	}
	if !hostRowSeen {
		host := lobbyPlayer{Slot: 0, Host: true, UserID: fmt.Sprint(g.HostUserID)}
		if u, err := s.Store.UserByID(g.HostUserID); err == nil {
			host.Username = u.Username
		}
		out = append([]lobbyPlayer{host}, out...)
	}
	// 空位 = 总人数 − 已占行 −（房主尚未落座时房主那一席）。
	open := g.PlayerCount - len(players)
	if !hostRowSeen {
		open--
	}
	if open < 0 {
		open = 0
	}
	return map[string]any{
		"id":          g.Token,
		"name":        g.Name,
		"scenarioId":  g.ScenarioID,
		"difficulty":  g.Difficulty,
		"status":      g.Status,
		"playerCount": g.PlayerCount,
		"players":     out,
		"openSlots":   open,
	}, nil
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
	Deck string `json:"deck"` // opaque deck token of the joining player
}

// handleJoinGame claims a lobby slot for the caller with their own deck.
// The host always owns slot 0 (and may re-pick their deck through the same
// route); everyone else takes the first open slot. Already-joined players
// can change their deck until the game starts.
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
	g, err := s.Store.GameByID(gameID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "game not found")
		return
	}
	if g.Status != "lobby" {
		writeErr(w, http.StatusConflict, "game already started")
		return
	}
	var req joinRequest
	if err := jsonDecode(r, &req); err != nil || req.Deck == "" {
		writeErr(w, http.StatusBadRequest, "deck required")
		return
	}
	deck, err := s.Store.DeckByToken(req.Deck)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "deck not found")
		return
	}
	if deck.UserID != uid {
		writeErr(w, http.StatusForbidden, "not your deck")
		return
	}
	heroBase, err := deckHero(deck)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if !checkDeckPlayable(w, deck) {
		return
	}
	slot := 0
	if uid != g.HostUserID {
		players, err := s.Store.GamePlayers(gameID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "store failed")
			return
		}
		used := map[int]bool{}
		for _, p := range players {
			used[p.Slot] = true
		}
		slot = -1
		for i := 1; i < g.PlayerCount; i++ {
			if !used[i] {
				slot = i
				break
			}
		}
		if slot < 0 {
			writeErr(w, http.StatusConflict, "lobby is full")
			return
		}
	}
	if err := s.Store.JoinLobbySlot(gameID, uid, deck.ID, heroBase, slot); err != nil {
		writeErr(w, http.StatusConflict, "slot not available")
		return
	}
	payload, err := s.lobbyPayload(g)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

// handleLobby serves the invite screen data; 404 once the game started so
// pollers switch to the board.
func (s *Server) handleLobby(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathGame(w, r)
	if !ok {
		return
	}
	g, err := s.Store.GameByID(gameID)
	if err != nil || g.Status != "lobby" {
		writeErr(w, http.StatusNotFound, "not in lobby")
		return
	}
	payload, err := s.lobbyPayload(g)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

type kickRequest struct {
	Slot int `json:"slot"`
}

// handleKick lets the host remove a joined player while still in the lobby.
func (s *Server) handleKick(w http.ResponseWriter, r *http.Request) {
	var uid int64
	if _, err := fmt.Sscanf(userID(r), "%d", &uid); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return
	}
	gameID, ok := s.pathGame(w, r)
	if !ok {
		return
	}
	g, err := s.Store.GameByID(gameID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "game not found")
		return
	}
	if g.Status != "lobby" {
		writeErr(w, http.StatusConflict, "game already started")
		return
	}
	if uid != g.HostUserID {
		writeErr(w, http.StatusForbidden, "only the host can remove players")
		return
	}
	var req kickRequest
	if err := jsonDecode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Slot <= 0 {
		writeErr(w, http.StatusBadRequest, "cannot remove the host")
		return
	}
	if err := s.Store.RemoveLobbyPlayer(gameID, req.Slot); err != nil {
		writeErr(w, http.StatusNotFound, "player not found")
		return
	}
	payload, err := s.lobbyPayload(g)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

// handleStart lets the host launch the game with whoever has joined so far;
// unclaimed slots are simply dropped (fewer players than configured).
func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	var uid int64
	if _, err := fmt.Sscanf(userID(r), "%d", &uid); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return
	}
	gameID, ok := s.pathGame(w, r)
	if !ok {
		return
	}
	g, err := s.Store.GameByID(gameID)
	if err != nil {
		writeErr(w, http.StatusNotFound, "game not found")
		return
	}
	if g.Status != "lobby" {
		writeErr(w, http.StatusConflict, "game already started")
		return
	}
	if uid != g.HostUserID {
		writeErr(w, http.StatusForbidden, "only the host can start")
		return
	}
	players, err := s.Store.GamePlayers(gameID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	if len(players) == 0 || players[0].Slot != 0 {
		writeErr(w, http.StatusBadRequest, "host must pick a deck first")
		return
	}
	specs := make([]engine.PlayerSpec, 0, len(players))
	for _, p := range players {
		deck, err := s.Store.DeckByID(p.DeckID)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "player deck is gone")
			return
		}
		if _, err := deckHero(deck); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if !checkDeckPlayable(w, deck) {
			return
		}
		name := ""
		if p.UserID != nil {
			if u, err := s.Store.UserByID(*p.UserID); err == nil {
				name = u.Username
			}
		}
		if name == "" {
			name = fmt.Sprintf("Player %d", p.Slot)
		}
		specs = append(specs, engine.PlayerSpec{
			Name: name, UserID: fmt.Sprint(*p.UserID), HeroBase: p.HeroBase, Deck: deck.Slots,
		})
	}
	var seedBuf [8]byte
	_, _ = rand.Read(seedBuf[:])
	seed := int64(binary.LittleEndian.Uint64(seedBuf[:]))
	state, err := engine.NewGame(engine.NewGameOptions{
		Seed:       seed,
		ScenarioID: g.ScenarioID,
		Difficulty: g.Difficulty,
		Players:    specs,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	stateJSON, err := state.MarshalJSON()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "serialize failed")
		return
	}
	if err := s.Store.StartGame(gameID, seed, string(stateJSON)); err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	view, err := s.Rooms.View(gameID, fmt.Sprint(uid))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "room failed")
		return
	}
	writeJSON(w, http.StatusOK, view)
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
	// Campaign chapters fold their result into the campaign log here.
	s.reportCampaignResult(gameID, view.Over)
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
	// Undoing the final answer rewinds the campaign chapter as well.
	s.reportCampaignResult(gameID, view.Over)
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
// viewer. Deck listings are sorted by card code server-side (see
// rooms.PileList), so the draw order is never revealed.
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
