package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/store"
)

func (s *Server) handleCards(w http.ResponseWriter, r *http.Request) {
	all := engine.DB.All()
	type cardOut struct {
		Code        string   `json:"code"`
		Name        string   `json:"name"`
		Subname     string   `json:"subname,omitempty"`
		PackCode    string   `json:"packCode"`
		PackName    string   `json:"packName,omitempty"`
		Type        string   `json:"type"`
		Category    string   `json:"category"`
		Aspect      string   `json:"aspect,omitempty"`
		CardSet     string   `json:"cardSet,omitempty"`
		Cost        *int     `json:"cost,omitempty"`
		Unique      bool     `json:"unique"`
		Traits      []string `json:"traits,omitempty"`
		Text        string   `json:"text,omitempty"`
		Resources   []string `json:"resources,omitempty"`
		Quantity    int      `json:"quantity,omitempty"`
		Implemented bool     `json:"implemented"`

		// Printed hero/ally stats for the deck-detail hero panel; nil when
		// the card has no printed value of that kind.
		HP       *int `json:"hp,omitempty"`
		Attack   *int `json:"attack,omitempty"`
		Thwart   *int `json:"thwart,omitempty"`
		Defense  *int `json:"defense,omitempty"`
		Recover  *int `json:"recover,omitempty"`
		HandSize *int `json:"handSize,omitempty"`
	}
	out := make([]cardOut, 0, len(all))
	for _, def := range all {
		out = append(out, cardOut{
			Code: def.Code, Name: def.Name, Subname: def.Subname,
			PackCode: def.PackCode, PackName: def.PackName, Type: def.Type, Category: def.Category,
			Aspect: def.Aspect, CardSet: def.CardSet, Cost: def.Cost,
			Unique: def.Unique, Traits: def.Traits, Text: def.Text, Resources: def.Resources,
			Quantity:    def.Quantity,
			Implemented: engine.Implemented(def.Code),
			HP:          def.HP, Attack: def.Attack, Thwart: def.Thwart,
			Defense: def.Defense, Recover: def.Recover, HandSize: def.HandSize,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleHeroes(w http.ResponseWriter, r *http.Request) {
	type heroOut struct {
		Base         string `json:"base"`
		HeroCode     string `json:"heroCode"`
		AlterEgoCode string `json:"alterEgoCode"`
		Name         string `json:"name"`
		AlterEgoName string `json:"alterEgoName"`
		PackCode     string `json:"packCode"`
		Implemented  bool   `json:"implemented"`
	}
	seen := map[string]bool{}
	var out []heroOut
	for _, def := range engine.DB.All() {
		if def.Type != "hero" || def.Side != "a" {
			continue
		}
		base := engine.BaseCodeOf(def.Code)
		if seen[base] {
			continue
		}
		seen[base] = true
		h := heroOut{
			Base: base, HeroCode: def.Code,
			AlterEgoCode: base + "b",
			Name:         def.Name, PackCode: def.PackCode,
			Implemented: engine.Implemented(def.Code),
		}
		if back, ok := engine.DB.Lookup(base + "b"); ok {
			h.AlterEgoName = back.Name
		}
		out = append(out, h)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleScenarios(w http.ResponseWriter, r *http.Request) {
	type scenOut struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var out []scenOut
	for _, def := range engine.Scenarios() {
		out = append(out, scenOut{ID: def.ID, Name: def.Name})
	}
	writeJSON(w, http.StatusOK, out)
}

// ---------------------------------------------------------------- decks

type importDeckRequest struct {
	// URL of a marvelcdb decklist to import.
	URL string `json:"url"`
	// Or the contents of a marvelcdb plain-text decklist export.
	Text string `json:"text"`
	// Or direct contents.
	Name             string         `json:"name"`
	InvestigatorCode string         `json:"investigatorCode"`
	Slots            map[string]int `json:"slots"`
}

var deckURLRE = regexp.MustCompile(`/(deck(list)?)(/view)?/([^/]+)`)

// marvelDBDecklist mirrors the marvelcdb decklist API JSON. marvelcdb exposes
// the hero as "hero_code" (unlike the ringsdb "investigator_code" some forks use).
type marvelDBDecklist struct {
	Name             string         `json:"name"`
	InvestigatorCode string         `json:"investigator_code"`
	HeroCode         string         `json:"hero_code"`
	Slots            map[string]int `json:"slots"`
}

func (s *Server) handleImportDeck(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var req importDeckRequest
	if err := jsonDecode(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.URL != "" {
		m := deckURLRE.FindStringSubmatch(req.URL)
		if m == nil {
			writeErr(w, http.StatusBadRequest, "not a marvelcdb deck URL")
			return
		}
		apiURL := fmt.Sprintf("https://marvelcdb.com/api/public/%s/%s.json", m[1], m[4])
		resp, err := http.Get(apiURL)
		if err != nil {
			writeErr(w, http.StatusBadGateway, "marvelcdb fetch failed")
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		if err != nil || resp.StatusCode != http.StatusOK {
			writeErr(w, http.StatusBadGateway, "marvelcdb fetch failed")
			return
		}
		var dl marvelDBDecklist
		if err := json.Unmarshal(body, &dl); err != nil || (dl.InvestigatorCode == "" && dl.HeroCode == "") {
			writeErr(w, http.StatusBadGateway, "invalid marvelcdb decklist")
			return
		}
		req.Name = dl.Name
		req.InvestigatorCode = dl.HeroCode
		if req.InvestigatorCode == "" {
			req.InvestigatorCode = dl.InvestigatorCode
		}
		req.Slots = dl.Slots
	} else if req.Text != "" {
		parsed, err := parseDecklistText(req.Text)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		req.Name = parsed.Name
		req.InvestigatorCode = parsed.InvestigatorCode
		req.Slots = parsed.Slots
	}
	if req.InvestigatorCode == "" || len(req.Slots) == 0 {
		writeErr(w, http.StatusBadRequest, "decklist requires investigatorCode and slots")
		return
	}
	// validate codes against the database
	var id int64
	if _, err := fmt.Sscanf(uid, "%d", &id); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return
	}
	for code := range req.Slots {
		if _, ok := engine.DB.Lookup(code); !ok {
			writeErr(w, http.StatusBadRequest, "unknown card code "+code)
			return
		}
	}
	if _, ok := engine.DB.Lookup(engine.BaseCodeOf(req.InvestigatorCode) + "a"); !ok {
		writeErr(w, http.StatusBadRequest, "unknown hero "+req.InvestigatorCode)
		return
	}
	if req.Name == "" {
		req.Name = "Unnamed deck"
	}
	deck, err := s.Store.CreateDeck(id, req.Name, req.InvestigatorCode, req.Slots)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": deck.Token, "name": deck.Name})
}

func (s *Server) handleGetDeck(w http.ResponseWriter, r *http.Request) {
	var uid int64
	if _, err := fmt.Sscanf(userID(r), "%d", &uid); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return
	}
	deck, err := s.Store.DeckByToken(r.PathValue("id"))
	if err != nil || deck.UserID != uid {
		writeErr(w, http.StatusNotFound, "deck not found")
		return
	}
	writeJSON(w, http.StatusOK, deck)
}

func (s *Server) handleListDecks(w http.ResponseWriter, r *http.Request) {
	var id int64
	if _, err := fmt.Sscanf(userID(r), "%d", &id); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return
	}
	decks, err := s.Store.DecksForUser(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	if decks == nil {
		decks = []store.Deck{}
	}
	writeJSON(w, http.StatusOK, decks)
}

func (s *Server) handleDeleteDeck(w http.ResponseWriter, r *http.Request) {
	var id int64
	if _, err := fmt.Sscanf(userID(r), "%d", &id); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return
	}
	deck, err := s.Store.DeckByToken(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "deck not found")
		return
	}
	if err := s.Store.DeleteDeck(id, deck.ID); err != nil {
		writeErr(w, http.StatusNotFound, "deck not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
