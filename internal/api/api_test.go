package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/rooms"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/store"

	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

const testSecret = "test-secret"

// legalSpideyDeck builds a rulebook-legal Spider-Man deck: the exact hero
// set at printed counts plus core justice/basic filler up to the 40-card
// minimum, so game flows run against decks that pass ValidateDeck.
func legalSpideyDeck() map[string]int {
	slots := map[string]int{
		"01002": 1, "01003": 2, "01004": 2, "01005": 3, "01006": 1,
		"01007": 2, "01008": 2, "01009": 2, // Spider-Man hero set
	}
	total := 15
	for _, d := range engine.DB.All() {
		if total >= 40 {
			break
		}
		if d.PackCode != "core" || d.Unique || d.Quantity <= 0 {
			continue
		}
		if d.Aspect != "justice" && d.Aspect != "basic" {
			continue
		}
		switch d.Type {
		case "hero", "alter_ego", "obligation":
			continue
		}
		n := 3
		if total+n > 40 {
			n = 40 - total
		}
		slots[d.Code] = n
		total += n
	}
	return slots
}

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	srv := &Server{
		Store:  st,
		Rooms:  rooms.NewManager(st),
		Secret: []byte(testSecret),
	}
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)
	return ts, st
}

func doJSON(t *testing.T, method, url, token string, body any) (*http.Response, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	out := map[string]any{}
	if resp.StatusCode != http.StatusNoContent {
		_ = json.NewDecoder(resp.Body).Decode(&out)
	}
	resp.Body.Close()
	return resp, out
}

// TestMarvelDBDecklistParse pins the marvelcdb API payload shape: the hero is
// exposed as "hero_code", not ringsdb's "investigator_code".
func TestMarvelDBDecklistParse(t *testing.T) {
	payload := []byte(`{
		"id": 63988,
		"name": "The defense rests, your honor",
		"hero_code": "60001a",
		"hero_name": "Daredevil",
		"slots": {"60048": 3, "60050": 3, "01008": 2},
		"ignoreDeckLimitSlots": null,
		"version": "1.0",
		"meta": "{\"aspect\":\"protection\"}"
	}`)
	var dl marvelDBDecklist
	if err := json.Unmarshal(payload, &dl); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	code := dl.HeroCode
	if code == "" {
		code = dl.InvestigatorCode
	}
	if code != "60001a" {
		t.Fatalf("hero code: got %q, want 60001a", code)
	}
	if dl.Name != "The defense rests, your honor" || dl.Slots["60048"] != 3 {
		t.Fatalf("parsed decklist incomplete: %+v", dl)
	}
}

// TestAuthTokenFailures pins the distinct 401 reasons the frontend relies on
// (missing vs invalid vs expired) and the happy path.
func TestAuthTokenFailures(t *testing.T) {
	ts, _ := newTestServer(t)
	base := ts.URL

	_, b := doJSON(t, "POST", base+"/api/v1/register", "", credentials{Username: "tokuser", Password: "secret123"})
	good, _ := b["token"].(string)
	if good == "" {
		t.Fatalf("register returned no token: %v", b)
	}

	resp, body := doJSON(t, "GET", base+"/api/v1/whoami", good, nil)
	if resp.StatusCode != http.StatusOK || body["username"] != "tokuser" {
		t.Fatalf("whoami with good token: %d %v", resp.StatusCode, body)
	}

	resp, body = doJSON(t, "GET", base+"/api/v1/whoami", "", nil)
	if resp.StatusCode != http.StatusUnauthorized || body["error"] != "missing token" {
		t.Fatalf("missing token: %d %v", resp.StatusCode, body)
	}

	resp, body = doJSON(t, "GET", base+"/api/v1/whoami", good+"x", nil)
	if resp.StatusCode != http.StatusUnauthorized || body["error"] != "invalid token" {
		t.Fatalf("tampered token: %d %v", resp.StatusCode, body)
	}

	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "1",
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-25 * time.Hour)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}
	resp, body = doJSON(t, "GET", base+"/api/v1/whoami", expired, nil)
	if resp.StatusCode != http.StatusUnauthorized || body["error"] != "token expired" {
		t.Fatalf("expired token: %d %v", resp.StatusCode, body)
	}
}

// TestImportDeckFromTextAndFetchDetail covers the marvelcdb plain-text
// import path and the per-deck detail endpoint.
func TestImportDeckFromTextAndFetchDetail(t *testing.T) {
	ts, _ := newTestServer(t)
	base := ts.URL

	_, b := doJSON(t, "POST", base+"/api/v1/register", "", credentials{Username: "txtuser", Password: "secret123"})
	token := b["token"].(string)

	text := "Text Deck\r\n\r\nSpider-Man\r\nPacks: Core Set\r\n\r\nEvents\r\n2x Uppercut (Core Set)\r\n\r\nResources\r\n3x Energy (Core Set)\r\n"
	resp, body := doJSON(t, "POST", base+"/api/v1/marvel/decks", token, importDeckRequest{Text: text})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("import text deck: %d %v", resp.StatusCode, body)
	}
	deckID := body["id"].(string)
	if len(deckID) != 16 {
		t.Fatalf("deck id should be a 16-char token, got %q", deckID)
	}

	resp, body = doJSON(t, "GET", fmt.Sprintf("%s/api/v1/marvel/decks/%s", base, deckID), token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get deck: %d %v", resp.StatusCode, body)
	}
	if body["investigatorCode"] != "01001a" {
		t.Fatalf("investigator: %v", body["investigatorCode"])
	}
	slots, _ := body["slots"].(map[string]any)
	if len(slots) != 2 || slots["01054"].(float64) != 2 || slots["01088"].(float64) != 3 {
		t.Fatalf("slots: %v", slots)
	}

	// another user must not see the deck
	_, b2 := doJSON(t, "POST", base+"/api/v1/register", "", credentials{Username: "txtuser2", Password: "secret123"})
	resp, body = doJSON(t, "GET", fmt.Sprintf("%s/api/v1/marvel/decks/%s", base, deckID), b2["token"].(string), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("foreign deck should 404: %d %v", resp.StatusCode, body)
	}

	// bad text is a 400 with a useful message
	resp, body = doJSON(t, "POST", base+"/api/v1/marvel/decks", token, importDeckRequest{Text: "nonsense\n\nSpider-Man\n"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad text: %d %v", resp.StatusCode, body)
	}
}

func TestFullGameFlow(t *testing.T) {
	ts, _ := newTestServer(t)
	base := ts.URL

	// register
	resp, body := doJSON(t, "POST", base+"/api/v1/register", "", credentials{Username: "alice", Password: "secret123"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: %d %v", resp.StatusCode, body)
	}
	token := body["token"].(string)

	// whoami
	resp, body = doJSON(t, "GET", base+"/api/v1/whoami", token, nil)
	if resp.StatusCode != http.StatusOK || body["username"] != "alice" {
		t.Fatalf("whoami: %d %v", resp.StatusCode, body)
	}

	// scenarios
	resp2, err := http.Get(base + "/api/v1/marvel/scenarios")
	if err != nil {
		t.Fatalf("scenarios: %v", err)
	}
	var scens []map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&scens)
	resp2.Body.Close()
	if len(scens) < 2 {
		t.Fatalf("expected 2+ scenarios, got %v", scens)
	}

	// import deck directly
	deck := importDeckRequest{
		Name:             "Spidey Aggression",
		InvestigatorCode: "01001a",
		Slots:            legalSpideyDeck(),
	}
	resp, body = doJSON(t, "POST", base+"/api/v1/marvel/decks", token, deck)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("import deck: %d %v", resp.StatusCode, body)
	}
	deckID := body["id"].(string)

	// create game
	resp, body = doJSON(t, "POST", base+"/api/v1/marvel/games", token, createGameRequest{
		DeckIDs:    []string{deckID},
		ScenarioID: "01097",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create game: %d %v", resp.StatusCode, body)
	}
	view := body
	gameID := view["id"].(string)
	if len(gameID) != 16 {
		t.Fatalf("game id should be a 16-char token, got %q", gameID)
	}
	// The game opens in setup (round 0) paused on the mulligan question.
	if view["round"].(float64) != 0 {
		t.Fatalf("expected round 0 during setup, got %v", view["round"])
	}
	if q := view["question"]; q == nil {
		t.Fatal("expected a question for the creating player")
	}

	// answer loop: always pick the first choice
	answered := 0
	for i := 0; i < 200; i++ {
		resp, body = doJSON(t, "GET", fmt.Sprintf("%s/api/v1/marvel/games/%s", base, gameID), token, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("view: %d", resp.StatusCode)
		}
		view = body
		if over, _ := view["over"].(bool); over {
			break
		}
		q, ok := view["question"].(map[string]any)
		if !ok {
			t.Fatalf("no question at iteration %d", i)
		}
		qtype, _ := q["type"].(string)
		choices, _ := q["choices"].([]any)
		if len(choices) == 0 {
			t.Fatalf("question without choices: %v", q["prompt"])
		}
		var paths []string
		if qtype == "choose_n" {
			n := int(q["n"].(float64))
			if n == 0 {
				n = 1
			}
			for j := 0; j < n && j < len(choices); j++ {
				c := choices[j].(map[string]any)
				paths = append(paths, c["id"].(string))
			}
		} else {
			// Prefer a choice without a payment subtree (e.g. "Take the
			// attack" during a defense prompt): an empty hand leaves the
			// payment subtree unanswerable.
			var c map[string]any
			for _, raw := range choices {
				cc := raw.(map[string]any)
				then, ok := cc["then"].(map[string]any)
				if !ok || then["type"] != "choose_n" {
					c = cc
					break
				}
				if c == nil {
					c = cc
				}
			}
			var paths2 []string
			if then, ok := c["then"].(map[string]any); ok && then["type"] == "choose_n" {
				// A play/ability choice with a choose_n payment subtree:
				// select enough payment choices (each worth >= 1 icon) and
				// answer with the full nested paths.
				subChoices, _ := then["choices"].([]any)
				need := 1
				if n2, ok := then["n"].(float64); ok && int(n2) > 0 {
					need = int(n2)
				}
				for j := 0; j < need+2 && j < len(subChoices); j++ {
					sc := subChoices[j].(map[string]any)
					paths2 = append(paths2, sc["id"].(string))
				}
			}
			if len(paths2) > 0 {
				paths = paths2
			} else {
				paths = []string{c["id"].(string)}
			}
		}
		resp, body = doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%s/answer", base, gameID), token, answerRequest{Paths: paths})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("answer %v: %d %v", paths, resp.StatusCode, body)
		}
		answered++
	}

	if answered < 3 {
		t.Fatalf("suspiciously short game: %d answers", answered)
	}

	// undo once
	resp, body = doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%s/undo", base, gameID), token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("undo: %d %v", resp.StatusCode, body)
	}

	// replay log
	resp, body = doJSON(t, "GET", fmt.Sprintf("%s/api/v1/marvel/games/%s/replay", base, gameID), token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("replay: %d", resp.StatusCode)
	}
	if body["seed"] == nil || body["actions"] == nil {
		t.Fatalf("replay payload incomplete: %v", body)
	}
}

func TestHandRedaction(t *testing.T) {
	ts, _ := newTestServer(t)
	base := ts.URL

	// two users
	_, b1 := doJSON(t, "POST", base+"/api/v1/register", "", credentials{Username: "user1", Password: "secret123"})
	_, b2 := doJSON(t, "POST", base+"/api/v1/register", "", credentials{Username: "user2", Password: "secret123"})
	t1, t2 := b1["token"].(string), b2["token"].(string)

	deck := importDeckRequest{
		Name:             "Deck",
		InvestigatorCode: "01001a",
		Slots:            legalSpideyDeck(),
	}
	_, bd := doJSON(t, "POST", base+"/api/v1/marvel/decks", t1, deck)
	deckID := bd["id"].(string)

	resp, body := doJSON(t, "POST", base+"/api/v1/marvel/games", t1, createGameRequest{DeckIDs: []string{deckID}, ScenarioID: "01097"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create game: %d %v", resp.StatusCode, body)
	}
	view := body
	gameID := view["id"].(string)

	// u2 sees no hand codes and no question
	resp, body = doJSON(t, "GET", fmt.Sprintf("%s/api/v1/marvel/games/%s", base, gameID), t2, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("spectator view: %d", resp.StatusCode)
	}
	players := body["players"].([]any)
	if len(players) != 1 {
		t.Fatalf("players: %v", players)
	}
	p := players[0].(map[string]any)
	if hand, ok := p["hand"].([]any); ok && len(hand) > 0 {
		t.Fatal("spectator should not see hand contents")
	}
	if body["question"] != nil {
		t.Fatal("spectator should not receive the pending question")
	}
	if body["waitingFor"] == nil {
		t.Fatal("waitingFor should be public")
	}

	// u2 cannot answer u1's question
	resp, body = doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%s/answer", base, gameID), t2, answerRequest{Paths: []string{"0"}})
	if resp.StatusCode == http.StatusOK {
		t.Fatal("u2 must not answer u1's question")
	}
}

// TestOpaqueIdentifiers pins the anti-enumeration contract: game and deck
// identities in URLs and payloads are 16-char URL-safe tokens, sequential
// ids never resolve, and the replay payload leaks no numeric deck ids.
func TestOpaqueIdentifiers(t *testing.T) {
	ts, _ := newTestServer(t)
	base := ts.URL

	tokenRE := regexp.MustCompile(`^[A-Za-z0-9_-]{16}$`)

	_, b := doJSON(t, "POST", base+"/api/v1/register", "", credentials{Username: "opaque", Password: "secret123"})
	token := b["token"].(string)

	_, bd := doJSON(t, "POST", base+"/api/v1/marvel/decks", token, importDeckRequest{
		Name:             "Deck",
		InvestigatorCode: "01001a",
		Slots:            legalSpideyDeck(),
	})
	deckID := bd["id"].(string)
	if !tokenRE.MatchString(deckID) {
		t.Fatalf("deck id not a token: %q", deckID)
	}

	_, bg := doJSON(t, "POST", base+"/api/v1/marvel/games", token, createGameRequest{DeckIDs: []string{deckID}, ScenarioID: "01097"})
	gameID := bg["id"].(string)
	if !tokenRE.MatchString(gameID) {
		t.Fatalf("game id not a token: %q", gameID)
	}

	// sequential ids must not resolve anywhere (method-matched so routing
	// itself answers 404, not 405)
	numericCases := []struct{ method, path string }{
		{"GET", "/api/v1/marvel/games/1"},
		{"GET", "/api/v1/marvel/games/1/replay"},
		{"GET", "/api/v1/marvel/games/1/pile?player=p&pile=deck"},
		{"POST", "/api/v1/marvel/games/1/answer"},
		{"POST", "/api/v1/marvel/games/1/join"},
		{"GET", "/api/v1/marvel/decks/1"},
	}
	for _, tc := range numericCases {
		resp, _ := doJSON(t, tc.method, base+tc.path, token, map[string]any{})
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("numeric path should 404: %s %s -> %d", tc.method, tc.path, resp.StatusCode)
		}
	}

	// well-formed but unknown tokens 404 too
	resp, _ := doJSON(t, "GET", base+"/api/v1/marvel/games/aaaaaaaaaaaaaaaa", token, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown token should 404: %d", resp.StatusCode)
	}

	// the games list only carries tokens
	req, _ := http.NewRequest("GET", base+"/api/v1/marvel/games", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	lresp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list games: %v", err)
	}
	var games []map[string]any
	_ = json.NewDecoder(lresp.Body).Decode(&games)
	lresp.Body.Close()
	if len(games) == 0 {
		t.Fatal("expected the created game in the list")
	}
	for _, g := range games {
		if !tokenRE.MatchString(g["id"].(string)) {
			t.Fatalf("list exposed a non-token id: %v", g["id"])
		}
	}

	// replay exposes deck tokens, not numeric deck ids
	_, br := doJSON(t, "GET", fmt.Sprintf("%s/api/v1/marvel/games/%s/replay", base, gameID), token, nil)
	rawReplay, err := json.Marshal(br)
	if err != nil {
		t.Fatalf("replay re-marshal: %v", err)
	}
	if bytes.Contains(rawReplay, []byte(`"deckId"`)) {
		t.Fatal("replay leaked numeric deckId")
	}
	players, _ := br["players"].([]any)
	if len(players) == 0 {
		t.Fatal("replay missing players")
	}
	p0 := players[0].(map[string]any)
	if !tokenRE.MatchString(p0["deck"].(string)) {
		t.Fatalf("replay player deck not a token: %v", p0["deck"])
	}
}

// TestLobbyFlow covers the multiplayer invite flow: scenario-only creation,
// per-player deck choice through the invite token, the host's kick and
// start-with-fewer powers, and the lobby's 404-on-start poll contract.
func TestLobbyFlow(t *testing.T) {
	ts, _ := newTestServer(t)
	base := ts.URL

	var tokens [4]string // auth tokens, index 0 = host
	var decks [4]string  // each user's own deck token
	for i := 0; i < 4; i++ {
		_, b := doJSON(t, "POST", base+"/api/v1/register", "", credentials{Username: fmt.Sprintf("lobby%d", i), Password: "secret123"})
		tokens[i] = b["token"].(string)
		_, bd := doJSON(t, "POST", base+"/api/v1/marvel/decks", tokens[i], importDeckRequest{
			Name:             fmt.Sprintf("Deck %d", i),
			InvestigatorCode: "01001a",
			Slots:            legalSpideyDeck(),
		})
		decks[i] = bd["id"].(string)
	}

	// host creates a 3-player lobby: scenario only, no decks
	resp, body := doJSON(t, "POST", base+"/api/v1/marvel/games", tokens[0], createGameRequest{
		PlayerCount: 3,
		ScenarioID:  "01097",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create lobby: %d %v", resp.StatusCode, body)
	}
	if body["status"] != "lobby" || body["openSlots"].(float64) != 2 {
		t.Fatalf("lobby shape: %v", body)
	}
	gameToken := body["id"].(string)
	players := body["players"].([]any)
	if len(players) != 1 {
		t.Fatalf("expected synthetic host entry, got %v", players)
	}
	hostEntry := players[0].(map[string]any)
	if hostEntry["host"] != true || hostEntry["username"] != "lobby0" {
		t.Fatalf("host entry: %v", hostEntry)
	}

	join := func(user, deck string) (*http.Response, map[string]any) {
		return doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%s/join", base, gameToken), user, joinRequest{Deck: deck})
	}

	// using someone else's deck is rejected
	if resp, _ = join(tokens[0], decks[1]); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign deck should 403: %d", resp.StatusCode)
	}

	// u1 joins with their own deck (slot 1)
	resp, body = join(tokens[1], decks[1])
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("join: %d %v", resp.StatusCode, body)
	}
	if ps := body["players"].([]any); len(ps) != 2 {
		t.Fatalf("players after join: %v", ps)
	}

	// u2 joins (slot 2) → full; u3 cannot join
	if resp, _ = join(tokens[2], decks[2]); resp.StatusCode != http.StatusOK {
		t.Fatalf("join u2: %d", resp.StatusCode)
	}
	resp, _ = join(tokens[3], decks[3])
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("full lobby should 409: %d", resp.StatusCode)
	}

	// join without a deck (the old request shape) is rejected in the lobby
	resp, _ = doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%s/join", base, gameToken), tokens[3], map[string]any{})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("deck-less join should 400: %d", resp.StatusCode)
	}

	// only the host can kick; the host itself is not removable
	resp, _ = doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%s/kick", base, gameToken), tokens[1], kickRequest{Slot: 2})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-host kick should 403: %d", resp.StatusCode)
	}
	resp, _ = doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%s/kick", base, gameToken), tokens[0], kickRequest{Slot: 0})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("kicking the host should 400: %d", resp.StatusCode)
	}
	resp, body = doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%s/kick", base, gameToken), tokens[0], kickRequest{Slot: 2})
	if resp.StatusCode != http.StatusOK || body["openSlots"].(float64) != 1 {
		t.Fatalf("kick: %d %v", resp.StatusCode, body)
	}

	// start requires the host's own deck
	resp, _ = doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%s/start", base, gameToken), tokens[0], map[string]any{})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("start without host deck should 400: %d", resp.StatusCode)
	}

	// host picks their deck, then starts with 2 of 3 configured players
	if resp, _ = join(tokens[0], decks[0]); resp.StatusCode != http.StatusOK {
		t.Fatalf("host deck pick: %d", resp.StatusCode)
	}
	resp, body = doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%s/start", base, gameToken), tokens[0], map[string]any{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("start: %d %v", resp.StatusCode, body)
	}
	if view := body; view["id"] != gameToken {
		t.Fatalf("started view id: %v", view["id"])
	}
	if ps := body["players"].([]any); len(ps) != 2 {
		t.Fatalf("expected 2 of 3 players, got %d", len(ps))
	}
	if _, ok := body["question"]; !ok {
		t.Fatal("host should receive the pending question after start")
	}

	// the lobby is gone: polls 404, joins and starts are rejected
	resp, _ = doJSON(t, "GET", fmt.Sprintf("%s/api/v1/marvel/games/%s/lobby", base, gameToken), tokens[0], nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("lobby after start should 404: %d", resp.StatusCode)
	}
	if resp, _ = join(tokens[3], decks[3]); resp.StatusCode != http.StatusConflict {
		t.Fatalf("join after start should 409: %d", resp.StatusCode)
	}
	resp, _ = doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%s/start", base, gameToken), tokens[0], map[string]any{})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("double start should 409: %d", resp.StatusCode)
	}

	// a user who never joined spectates the started game
	resp, body = doJSON(t, "GET", fmt.Sprintf("%s/api/v1/marvel/games/%s", base, gameToken), tokens[3], nil)
	if resp.StatusCode != http.StatusOK || body["question"] != nil {
		t.Fatalf("spectator view: %d %v", resp.StatusCode, body["question"])
	}
}

// TestValidateDeckEndpoint pins the deck builder's live-validation endpoint:
// same engine.ValidateDeck rules as the stored-deck projection, structured
// issues in, no persistence.
func TestValidateDeckEndpoint(t *testing.T) {
	ts, _ := newTestServer(t)
	base := ts.URL

	_, b := doJSON(t, "POST", base+"/api/v1/register", "", credentials{Username: "validator", Password: "secret123"})
	token := b["token"].(string)

	// a rulebook-legal deck validates clean
	resp, body := doJSON(t, "POST", base+"/api/v1/marvel/decks/validate", token, map[string]any{
		"investigatorCode": "01001a",
		"slots":            legalSpideyDeck(),
	})
	if resp.StatusCode != http.StatusOK || body["valid"] != true {
		t.Fatalf("legal deck should validate: %d %v", resp.StatusCode, body)
	}
	if issues, ok := body["issues"].([]any); !ok || len(issues) != 0 {
		t.Fatalf("legal deck should have no issues: %v", body["issues"])
	}

	// a stub deck reports the structured issues the UI renders
	resp, body = doJSON(t, "POST", base+"/api/v1/marvel/decks/validate", token, map[string]any{
		"investigatorCode": "01001a",
		"slots":            map[string]int{"01088": 3},
	})
	if resp.StatusCode != http.StatusOK || body["valid"] != false {
		t.Fatalf("stub deck should not validate: %d %v", resp.StatusCode, body)
	}
	issues, _ := body["issues"].([]any)
	if len(issues) == 0 {
		t.Fatal("stub deck should carry issues")
	}
	found := map[string]bool{}
	for _, raw := range issues {
		is := raw.(map[string]any)
		found[is["key"].(string)] = true
	}
	if !found["setMissing"] || !found["tooSmall"] {
		t.Fatalf("expected setMissing+tooSmall issues: %v", issues)
	}

	// an unknown identity is reported, not a 500
	resp, body = doJSON(t, "POST", base+"/api/v1/marvel/decks/validate", token, map[string]any{
		"investigatorCode": "99999a",
		"slots":            legalSpideyDeck(),
	})
	if resp.StatusCode != http.StatusOK || body["valid"] != false {
		t.Fatalf("unknown identity: %d %v", resp.StatusCode, body)
	}

	// the endpoint requires auth like the other deck endpoints
	resp, _ = doJSON(t, "POST", base+"/api/v1/marvel/decks/validate", "", map[string]any{})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated validate should 401: %d", resp.StatusCode)
	}
}

// TestCardsCatalogRiderFields pins the structured deckbuilding-rider fields
// the builder UI consumes from /marvel/cards (Spider-Woman's alter-ego side,
// Gamora's exception with its cap, plain Spider-Man carries none).
func TestCardsCatalogRiderFields(t *testing.T) {
	ts, _ := newTestServer(t)

	resp, err := http.Get(ts.URL + "/api/v1/marvel/cards")
	if err != nil {
		t.Fatalf("cards: %v", err)
	}
	var cards []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&cards)
	resp.Body.Close()

	byCode := map[string]map[string]any{}
	for _, c := range cards {
		byCode[c["code"].(string)] = c
	}

	sw := byCode["04031b"]
	if sw == nil || sw["aspectMode"] != "two_equal" {
		t.Fatalf("spider-woman alter ego aspectMode: %v", sw["aspectMode"])
	}
	gamora := byCode["18001b"]
	if gamora == nil {
		t.Fatal("gamora alter ego missing from catalog")
	}
	ex, ok := gamora["aspectException"].(map[string]any)
	if !ok || ex["cardType"] != "event" || ex["total"].(float64) != 6 {
		t.Fatalf("gamora aspectException: %v", gamora["aspectException"])
	}
	spidey := byCode["01001b"]
	if spidey == nil {
		t.Fatal("spider-man alter ego missing from catalog")
	}
	if _, has := spidey["aspectMode"]; has {
		t.Fatal("plain alter ego should omit aspectMode")
	}
	if _, has := spidey["aspectException"]; has {
		t.Fatal("plain alter ego should omit aspectException")
	}
}
