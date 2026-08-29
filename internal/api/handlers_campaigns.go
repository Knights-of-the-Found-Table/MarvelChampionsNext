// Campaign mode HTTP surface: create/join/start campaigns, play the next
// chapter, resolve interlude choices, buy Market cards, and fold finished
// chapter games back into the campaign log.
package api

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/campaign"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/store"
)

type createCampaignRequest struct {
	Box         string `json:"box"`
	Difficulty  string `json:"difficulty"`
	PlayerCount int    `json:"playerCount"`
	DeckID      string `json:"deckId"` // solo campaigns: the creator's deck
}

func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.userIDInt(w, r)
	if !ok {
		return
	}
	var req createCampaignRequest
	if err := jsonDecode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if _, ok := campaign.Boxes[req.Box]; !ok {
		writeErr(w, http.StatusBadRequest, "unknown campaign box")
		return
	}
	if req.Difficulty == "" {
		req.Difficulty = "standard"
	}
	if req.Difficulty != "standard" && req.Difficulty != "expert" {
		writeErr(w, http.StatusBadRequest, "difficulty must be standard or expert")
		return
	}
	if req.PlayerCount == 0 {
		req.PlayerCount = 1
	}
	if req.PlayerCount < 1 || req.PlayerCount > 4 {
		writeErr(w, http.StatusBadRequest, "playerCount must be 1-4")
		return
	}
	players := []store.CampaignPlayer{}
	if req.PlayerCount == 1 {
		if req.DeckID == "" {
			writeErr(w, http.StatusBadRequest, "solo campaigns need a deck")
			return
		}
		deck, ok := s.deckForCampaign(w, req.DeckID)
		if !ok {
			return
		}
		uidCopy := uid
		players = append(players, store.CampaignPlayer{Slot: 0, UserID: &uidCopy, DeckID: deck.ID, HeroBase: engine.BaseCodeOf(deck.InvestigatorCode)})
	}
	st := campaign.New(req.Box, req.Difficulty, []campaign.PlayerLog{})
	state, err := json.Marshal(st)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "serialize failed")
		return
	}
	row, err := s.Store.CreateCampaign(req.Box, req.Difficulty, string(state), uid, req.PlayerCount)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	// Solo: claim the only seat right away.
	for _, p := range players {
		if _, err := s.Store.JoinCampaignSlot(row.ID, *p.UserID, p.DeckID, p.HeroBase); err != nil {
			writeErr(w, http.StatusInternalServerError, "store failed")
			return
		}
	}
	s.writeCampaign(w, r, row.ID, http.StatusCreated)
}

// deckForCampaign loads a deck and validates its hero (deck legality is
// NOT re-checked: campaign decks legally include campaign-only cards).
func (s *Server) deckForCampaign(w http.ResponseWriter, token string) (*store.Deck, bool) {
	deck, err := s.Store.DeckByToken(token)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("deck %s not found", token))
		return nil, false
	}
	heroBase := engine.BaseCodeOf(deck.InvestigatorCode)
	if !engine.Implemented(heroBase + "a") {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("hero not implemented yet: %s", heroBase))
		return nil, false
	}
	return deck, true
}

func (s *Server) handleCampaignBoxes(w http.ResponseWriter, r *http.Request) {
	boxes := []map[string]any{}
	for _, key := range []string{"rrs", "gmw", "mts", "sm", "mg", "nx", "aoa", "aos",
		"cowl", "whatif", "awesome", "alias", "watchers", "mojo", "bord", "night", "viral", "entropy"} {
		b := campaign.Boxes[key]
		if b == nil {
			continue
		}
		boxes = append(boxes, map[string]any{"key": b.Key, "name": b.Name, "desc": b.Desc, "scenarios": len(b.Scenarios)})
	}
	writeJSON(w, http.StatusOK, boxes)
}

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	uid, ok := s.userIDInt(w, r)
	if !ok {
		return
	}
	rows, err := s.Store.CampaignsByUser(uid)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	type summary struct {
		ID      string `json:"id"`
		Box     string `json:"box"`
		Name    string `json:"name"`
		Status  string `json:"status"`
		Index   int    `json:"index"`
		Played  int    `json:"playerCount"`
		Updated string `json:"updatedAt"`
	}
	out := []summary{}
	for _, c := range rows {
		name := c.Box
		if b := campaign.Boxes[c.Box]; b != nil {
			name = b.Name
		}
		out = append(out, summary{ID: c.Token, Box: c.Box, Name: name, Status: c.Status, Index: c.Index, Played: c.PlayerCount, Updated: c.UpdatedAt})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathCampaign(w, r)
	if !ok {
		return
	}
	s.writeCampaign(w, r, id, http.StatusOK)
}

type joinCampaignRequest struct {
	DeckID string `json:"deckId"`
}

func (s *Server) handleJoinCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathCampaign(w, r)
	if !ok {
		return
	}
	uid, ok := s.userIDInt(w, r)
	if !ok {
		return
	}
	var req joinCampaignRequest
	if err := jsonDecode(r, &req); err != nil || req.DeckID == "" {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	row, err := s.Store.CampaignByID(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "campaign not found")
		return
	}
	if row.Status != "forming" {
		writeErr(w, http.StatusBadRequest, "campaign already started")
		return
	}
	if slot, err := s.Store.CampaignSlotByUser(id, uid); err == nil && slot >= 0 {
		writeErr(w, http.StatusBadRequest, "already joined")
		return
	}
	deck, ok := s.deckForCampaign(w, req.DeckID)
	if !ok {
		return
	}
	if _, err := s.Store.JoinCampaignSlot(id, uid, deck.ID, engine.BaseCodeOf(deck.InvestigatorCode)); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.writeCampaign(w, r, id, http.StatusOK)
}

type kickCampaignRequest struct {
	Slot int `json:"slot"`
}

func (s *Server) handleKickCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathCampaign(w, r)
	if !ok {
		return
	}
	uid, ok := s.userIDInt(w, r)
	if !ok {
		return
	}
	row, err := s.Store.CampaignByID(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "campaign not found")
		return
	}
	if row.HostUserID != uid {
		writeErr(w, http.StatusForbidden, "only the host can kick")
		return
	}
	if row.Status != "forming" {
		writeErr(w, http.StatusBadRequest, "campaign already started")
		return
	}
	var req kickCampaignRequest
	if err := jsonDecode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := s.Store.KickCampaignSlot(id, req.Slot); err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	s.writeCampaign(w, r, id, http.StatusOK)
}

// handleStartCampaign locks the seats and builds the campaign log from
// each player's deck (the campaign snapshot lives in the state from here
// on).
func (s *Server) handleStartCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathCampaign(w, r)
	if !ok {
		return
	}
	uid, ok := s.userIDInt(w, r)
	if !ok {
		return
	}
	row, err := s.Store.CampaignByID(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "campaign not found")
		return
	}
	if row.HostUserID != uid {
		writeErr(w, http.StatusForbidden, "only the host can start")
		return
	}
	if row.Status != "forming" {
		writeErr(w, http.StatusBadRequest, "campaign already started")
		return
	}
	seats, err := s.Store.CampaignPlayers(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	if len(seats) != row.PlayerCount {
		writeErr(w, http.StatusBadRequest, "not all seats are claimed")
		return
	}
	st := campaign.New(row.Box, row.Difficulty, []campaign.PlayerLog{})
	for _, seat := range seats {
		deck, err := s.Store.DeckByID(seat.DeckID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "deck missing")
			return
		}
		name := fmt.Sprintf("Player %d", seat.Slot+1)
		if seat.UserID != nil {
			if u, err := s.Store.UserByID(*seat.UserID); err == nil {
				name = u.Username
			}
		}
		st.Players = append(st.Players, campaign.PlayerLog{
			Slot:     seat.Slot,
			UserID:   derefInt64(seat.UserID),
			Name:     name,
			HeroBase: seat.HeroBase,
			Deck:     copySlots(deck.Slots),
		})
	}
	st.Status = "interlude"
	// Mutant Genesis: the Future Past deck starts complete, and every
	// player owes a role pick before chapter 1.
	if st.Box == "nx" {
		campaign.QueueNXScheme(st)
	}
	if st.Box == "aos" {
		campaign.PrepareAOSEvidence(st)
	}
	if st.Box == "mg" {
		if len(st.MGFuturePast) == 0 {
			st.MGFuturePast = append(st.MGFuturePast, campaign.FuturePastSeed()...)
		}
		for i := range st.Players {
			if st.Players[i].MGRole == "" {
				st.AddPending(i, campaign.ChoiceMGRole)
			}
		}
	}
	// Contest campaign openings.
	switch st.Box {
	case "mojo":
		for i := range st.Players {
			if st.Players[i].MojoRole == "" {
				st.AddPending(i, campaign.ChoiceMojoRole)
			}
		}
		st.AddPending(0, campaign.ChoiceMojoTraining)
	case "bord":
		if st.Selections == nil {
			st.Selections = map[string]string{}
		}
		if st.Selections["path"] == "" {
			st.AddPending(0, campaign.ChoiceBordPath)
		}
	case "awesome":
		for i := range st.Players {
			if st.Players[i].AWAlly == "" {
				st.AddPending(i, campaign.ChoiceAWAlly)
			}
			if st.Players[i].AWIdentity == "" {
				st.AddPending(i, campaign.ChoiceAWIdentity)
			}
		}
	case "entropy":
		st.AddPending(0, campaign.ChoiceEnPath1)
	case "viral":
		// nothing owed before chapter 1; the branch pick lands after it
	case "watchers":
		if err := campaign.WatchersCheck(st); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	s.saveCampaignState(w, id, st)
	s.writeCampaign(w, r, id, http.StatusOK)
}

// handlePlayCampaign creates the current chapter's game from the campaign
// log and links it back. Solo and multiplayer games both start immediately
// (async multiplayer: players answer whenever they are around).
func (s *Server) handlePlayCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathCampaign(w, r)
	if !ok {
		return
	}
	uid, ok := s.userIDInt(w, r)
	if !ok {
		return
	}
	row, err := s.Store.CampaignByID(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "campaign not found")
		return
	}
	if row.HostUserID != uid {
		writeErr(w, http.StatusForbidden, "only the host can start the next chapter")
		return
	}
	if row.Status != "interlude" {
		writeErr(w, http.StatusBadRequest, "campaign is not ready to play")
		return
	}
	st, ok := s.campaignState(w, row)
	if !ok {
		return
	}
	var seedBuf [8]byte
	_, _ = rand.Read(seedBuf[:])
	seed := int64(binary.LittleEndian.Uint64(seedBuf[:]))
	opts, err := campaign.BuildGame(st, seed)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	g, err := engine.NewGame(opts)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	state, err := g.MarshalJSON()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "serialize failed")
		return
	}
	seats, err := s.Store.CampaignPlayers(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	players := make([]store.GamePlayer, 0, len(seats))
	for _, seat := range seats {
		p := seat
		players = append(players, store.GamePlayer{Slot: p.Slot, UserID: p.UserID, DeckID: p.DeckID, HeroBase: p.HeroBase})
	}
	box := st.BoxDef()
	name := row.Box
	if box != nil && st.Index < len(box.Scenarios) {
		name = box.Name + " · " + box.Scenarios[st.Index].Name
	}
	gameID, err := s.Store.CreateGame(name, opts.ScenarioID, opts.Difficulty, seed, string(state), "active", row.HostUserID, len(opts.Players), players)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	if err := s.Store.SetGameCampaign(gameID, id); err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	st.Status = "active"
	st.LastResult = ""
	s.saveCampaignState(w, id, st)
	game, err := s.Store.GameByID(gameID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"gameId": game.Token})
}

type campaignChoiceRequest struct {
	Kind     string `json:"kind"`
	CardCode string `json:"cardCode"`
}

func (s *Server) handleCampaignChoice(w http.ResponseWriter, r *http.Request) {
	id, slot, ok := s.myCampaignSlot(w, r)
	if !ok {
		return
	}
	var req campaignChoiceRequest
	if err := jsonDecode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	st, row, ok := s.campaignStateByID(w, id)
	if !ok {
		return
	}
	if row.Status != "interlude" {
		writeErr(w, http.StatusBadRequest, "no choices are pending")
		return
	}
	kind := req.Kind
	if kind == "" {
		// Legacy clients omit the kind; accept when exactly one is owed.
		if kinds := st.PendingKinds(slot); len(kinds) == 1 {
			kind = kinds[0]
		} else {
			writeErr(w, http.StatusBadRequest, "choice kind required")
			return
		}
	}
	if err := campaign.ApplyChoice(st, slot, kind, req.CardCode); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.saveCampaignState(w, id, st)
	s.writeCampaign(w, r, id, http.StatusOK)
}

func (s *Server) handleCampaignMarket(w http.ResponseWriter, r *http.Request) {
	id, slot, ok := s.myCampaignSlot(w, r)
	if !ok {
		return
	}
	var req campaignChoiceRequest
	if err := jsonDecode(r, &req); err != nil || req.CardCode == "" {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	st, _, ok := s.campaignStateByID(w, id)
	if !ok {
		return
	}
	if err := campaign.SpendOrBuyMarket(st, slot, req.CardCode); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.saveCampaignState(w, id, st)
	s.writeCampaign(w, r, id, http.StatusOK)
}

type campaignHealRequest struct {
	On bool `json:"on"`
}

// handleCampaignDeck re-seats the caller with a new deck between chapters
// (The Watcher's Team changes identities every chapter; several contest
// campaigns allow deck customization in the interlude).
func (s *Server) handleCampaignDeck(w http.ResponseWriter, r *http.Request) {
	id, slot, ok := s.myCampaignSlot(w, r)
	if !ok {
		return
	}
	var req joinCampaignRequest
	if err := jsonDecode(r, &req); err != nil || req.DeckID == "" {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	row, err := s.Store.CampaignByID(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "campaign not found")
		return
	}
	if row.Status != "interlude" {
		writeErr(w, http.StatusBadRequest, "decks can only change in the interlude")
		return
	}
	deck, ok := s.deckForCampaign(w, req.DeckID)
	if !ok {
		return
	}
	heroBase := engine.BaseCodeOf(deck.InvestigatorCode)
	if err := s.Store.UpdateCampaignDeck(id, slot, deck.ID, heroBase); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	st, _, ok := s.campaignStateByID(w, id)
	if !ok {
		return
	}
	if pl := st.Slot(slot); pl != nil {
		pl.HeroBase = heroBase
		pl.Deck = copySlots(deck.Slots)
		pl.SetupHand = ""
	}
	s.saveCampaignState(w, id, st)
	s.writeCampaign(w, r, id, http.StatusOK)
}

func (s *Server) handleCampaignHeal(w http.ResponseWriter, r *http.Request) {
	id, slot, ok := s.myCampaignSlot(w, r)
	if !ok {
		return
	}
	var req campaignHealRequest
	if err := jsonDecode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	st, _, ok := s.campaignStateByID(w, id)
	if !ok {
		return
	}
	if err := campaign.SetHeal(st, slot, req.On); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.saveCampaignState(w, id, st)
	s.writeCampaign(w, r, id, http.StatusOK)
}

// ---------------------------------------------------------------- report

// reportCampaignResult folds a finished chapter game into its campaign.
// Deterministic and guarded by the campaign status, so answer replays and
// double posts cannot double-apply.
func (s *Server) reportCampaignResult(gameID int64, over bool) {
	campID, err := s.Store.CampaignIDByGame(gameID)
	if err != nil || campID == 0 {
		return
	}
	row, err := s.Store.CampaignByID(campID)
	if err != nil {
		return
	}
	st, ok := decodeState(row)
	if !ok {
		return
	}
	gameRow, err := s.Store.GameByID(gameID)
	if err != nil {
		return
	}
	chapter := chapterIndex(st, gameRow.ScenarioID)
	if over {
		if st.Status != "active" {
			return
		}
		var g engine.Game
		if err := json.Unmarshal([]byte(gameRow.State), &g); err != nil {
			return
		}
		snap := campaign.Observe(&g)
		if g.Won {
			campaign.ApplyVictory(st, snap)
		} else {
			campaign.ApplyDefeat(st)
		}
		s.saveState(campID, st)
		return
	}
	// Undo rewound the game below its end: roll the campaign back to the
	// chapter so it can be replayed.
	if st.Status == "active" || chapter < 0 || st.Index != chapter+1 {
		return
	}
	st.Index = chapter
	st.Status = "active"
	st.PendingChoices = nil
	st.LastResult = ""
	s.saveState(campID, st)
}

func chapterIndex(st *campaign.State, scenarioID string) int {
	box := st.BoxDef()
	if box == nil {
		return -1
	}
	for i, sc := range box.Scenarios {
		if sc.ID == scenarioID {
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------- helpers

func (s *Server) pathCampaign(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := s.Store.CampaignIDByToken(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "campaign not found")
		return 0, false
	}
	return id, true
}

func (s *Server) userIDInt(w http.ResponseWriter, r *http.Request) (int64, bool) {
	var uid int64
	if _, err := fmt.Sscanf(userID(r), "%d", &uid); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return 0, false
	}
	return uid, true
}

// myCampaignSlot resolves the caller's seat and the campaign id.
func (s *Server) myCampaignSlot(w http.ResponseWriter, r *http.Request) (int64, int, bool) {
	id, ok := s.pathCampaign(w, r)
	if !ok {
		return 0, 0, false
	}
	uid, ok := s.userIDInt(w, r)
	if !ok {
		return 0, 0, false
	}
	slot, err := s.Store.CampaignSlotByUser(id, uid)
	if err != nil || slot < 0 {
		writeErr(w, http.StatusForbidden, "not a campaign player")
		return 0, 0, false
	}
	return id, slot, true
}

func (s *Server) campaignState(w http.ResponseWriter, row *store.CampaignRow) (*campaign.State, bool) {
	st, ok := decodeState(row)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "campaign state corrupted")
	}
	return st, ok
}

func (s *Server) campaignStateByID(w http.ResponseWriter, id int64) (*campaign.State, *store.CampaignRow, bool) {
	row, err := s.Store.CampaignByID(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "campaign not found")
		return nil, nil, false
	}
	st, ok := s.campaignState(w, row)
	return st, row, ok
}

func decodeState(row *store.CampaignRow) (*campaign.State, bool) {
	var st campaign.State
	if err := json.Unmarshal([]byte(row.State), &st); err != nil {
		return nil, false
	}
	if st.Players == nil {
		st.Players = []campaign.PlayerLog{}
	}
	// Maps omitted at save time (empty + omitempty) must come back as
	// writable maps, or the first flag/counter write panics.
	st.EnsureMaps()
	return &st, true
}

func (s *Server) saveCampaignState(w http.ResponseWriter, id int64, st *campaign.State) {
	if err := s.saveState(id, st); err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
	}
}

func (s *Server) saveState(id int64, st *campaign.State) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	index := st.Index
	status := st.Status
	return s.Store.SaveCampaignState(id, status, index, string(b))
}

// campaignSeatPayload is one seat in the detail payload.
type campaignSeatPayload struct {
	Slot     int    `json:"slot"`
	UserID   string `json:"userId,omitempty"`
	Username string `json:"username"`
	Hero     string `json:"hero"`
	DeckName string `json:"deckName"`
}

// writeCampaign renders the full campaign projection (log, seats,
// chapters, market).
func (s *Server) writeCampaign(w http.ResponseWriter, r *http.Request, id int64, status int) {
	row, err := s.Store.CampaignByID(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "campaign not found")
		return
	}
	st, ok := s.campaignState(w, row)
	if !ok {
		return
	}
	seats, err := s.Store.CampaignPlayers(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	seatPayload := []campaignSeatPayload{}
	deckBySeat := map[int]string{}
	for _, seat := range seats {
		p := campaignSeatPayload{Slot: seat.Slot}
		if seat.UserID != nil {
			p.UserID = fmt.Sprint(*seat.UserID)
			if u, err := s.Store.UserByID(*seat.UserID); err == nil {
				p.Username = u.Username
			}
		}
		if deck, err := s.Store.DeckByID(seat.DeckID); err == nil {
			p.DeckName = deck.Name
			if def, ok := engine.DB.Lookup(engine.BaseCodeOf(seat.HeroBase) + "a"); ok {
				p.Hero = def.EName
			}
		}
		deckBySeat[seat.Slot] = p.DeckName
		seatPayload = append(seatPayload, p)
	}
	box := st.BoxDef()
	type chapter struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Requires string `json:"requires,omitempty"`
	}
	chapters := []chapter{}
	if box != nil {
		for _, sc := range box.Scenarios {
			chapters = append(chapters, chapter{ID: sc.ID, Name: sc.Name, Requires: sc.Requires})
		}
	}
	games, _ := s.Store.CampaignGames(id)
	type gameRef struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		ScenarioID string `json:"scenarioId"`
		Status     string `json:"status"`
	}
	gameList := []gameRef{}
	for _, g := range games {
		gameList = append(gameList, gameRef{ID: g.Token, Name: g.Name, ScenarioID: g.ScenarioID, Status: g.Status})
	}
	uid, _ := s.userIDInt(w, r)
	slot, _ := s.Store.CampaignSlotByUser(id, uid)
	// EN display names for every campaign card code appearing in the
	// log, so clients can localize by code without a second lookup.
	names := map[string]string{}
	note := func(codes ...string) {
		for _, code := range codes {
			if code == "" {
				continue
			}
			if _, done := names[code]; done {
				continue
			}
			if def, ok := engine.DB.Lookup(code); ok {
				names[code] = def.EName
			}
		}
	}
	note(campaign.TechUpgrades()...)
	note(campaign.ConditionUpgrades()...)
	for _, pl := range st.Players {
		note(pl.Tech, pl.Condition)
		note(pl.Allies...)
		note(pl.Market...)
		note(pl.WIAllies...)
		note(pl.WIRewards...)
		note(pl.MojoEvent, pl.MojoMarket, pl.MojoScheme)
		note(pl.BordObligations...)
		note(pl.BordGear...)
		note(pl.AWAlly, pl.AWIdentity)
	}
	note(st.Experimental...)
	note(st.RemovedAllies...)
	note(st.Collection...)
	note(st.Artifacts...)
	note(st.Victims...)
	note(st.CowlCaught...)
	note(campaign.AliasEvidence(st)...)
	for _, code := range st.Pool {
		note(strings.TrimPrefix(code, "evidence:"))
	}
	for _, sel := range st.Selections {
		note(sel)
	}
	for _, mc := range campaign.MarketCards() {
		note(mc.Code)
	}
	note(campaign.NXAllSchemes()...)
	note(campaign.AOBoardMembers()...)
	note(campaign.SMAllTech()...)
	note(campaign.SMCommunitySchemes()...)
	// The A.I.M. envelope stays server-side: players must deduce the
	// mole, so the payload redacts it (state storage keeps it).
	st.AOImEnvelope = campaign.AOSCombo{}
	payload := map[string]any{
		"id":          row.Token,
		"box":         row.Box,
		"name":        row.Box,
		"difficulty":  row.Difficulty,
		"status":      row.Status,
		"index":       row.Index,
		"playerCount": row.PlayerCount,
		"host":        row.HostUserID == uid,
		"yourSlot":    slot,
		"state":       st,
		"seats":       seatPayload,
		"chapters":    chapters,
		"games":       gameList,
		"market":      campaign.MarketCards(),
		"names":       names,
		"pools": map[string][]string{
			"tech":       campaign.TechUpgrades(),
			"condition":  campaign.ConditionUpgrades(),
			"roles":      campaign.MGRoles(),
			"nx":         st.NXAvailable(),
			"aosMembers": campaign.AOBoardMembers(),
			"smTech":     campaign.SMAllTech(),
			"community":  campaign.SMCommunitySchemes(),
			"traits":     campaign.WhatIfTraits(),
			"soe":        campaign.EntSoeOptions(),
			"viralNext":  campaign.ViralNextOptions(st),
			"allNx":      campaign.NXAllSchemes(),
		},
		"tables": campaign.BoxTables(st),
	}
	if box != nil {
		payload["name"] = box.Name
		payload["desc"] = box.Desc
	}
	writeJSON(w, status, payload)
}

func copySlots(src map[string]int) map[string]int {
	out := make(map[string]int, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
