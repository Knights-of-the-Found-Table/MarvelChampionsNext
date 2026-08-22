package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/rooms"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/store"

	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

const testSecret = "test-secret"

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
	deckID := int64(body["id"].(float64))

	resp, body = doJSON(t, "GET", fmt.Sprintf("%s/api/v1/marvel/decks/%d", base, deckID), token, nil)
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
	resp, body = doJSON(t, "GET", fmt.Sprintf("%s/api/v1/marvel/decks/%d", base, deckID), b2["token"].(string), nil)
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
		Slots: map[string]int{
			"01002": 1, "01003": 2, "01004": 2, "01005": 2, "01006": 1,
			"01007": 2, "01008": 2,
			"01088": 3, "01089": 3, "01090": 3, "01054": 2, "01055": 1,
		},
	}
	resp, body = doJSON(t, "POST", base+"/api/v1/marvel/decks", token, deck)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("import deck: %d %v", resp.StatusCode, body)
	}
	deckID := int64(body["id"].(float64))

	// create game
	resp, body = doJSON(t, "POST", base+"/api/v1/marvel/games", token, createGameRequest{
		DeckIDs:    []int64{deckID},
		ScenarioID: "01097",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create game: %d %v", resp.StatusCode, body)
	}
	view := body
	gameID := int64(view["id"].(float64))
	if view["round"].(float64) != 1 {
		t.Fatalf("expected round 1, got %v", view["round"])
	}
	if q := view["question"]; q == nil {
		t.Fatal("expected a question for the creating player")
	}

	// answer loop: always pick the first choice
	answered := 0
	for i := 0; i < 200; i++ {
		resp, body = doJSON(t, "GET", fmt.Sprintf("%s/api/v1/marvel/games/%d", base, gameID), token, nil)
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
		resp, body = doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%d/answer", base, gameID), token, answerRequest{Paths: paths})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("answer %v: %d %v", paths, resp.StatusCode, body)
		}
		answered++
	}

	if answered < 3 {
		t.Fatalf("suspiciously short game: %d answers", answered)
	}

	// undo once
	resp, body = doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%d/undo", base, gameID), token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("undo: %d %v", resp.StatusCode, body)
	}

	// replay log
	resp, body = doJSON(t, "GET", fmt.Sprintf("%s/api/v1/marvel/games/%d/replay", base, gameID), token, nil)
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
		Slots:            map[string]int{"01088": 3, "01089": 3, "01090": 3, "01005": 2, "01006": 1, "01002": 1},
	}
	_, bd := doJSON(t, "POST", base+"/api/v1/marvel/decks", t1, deck)
	deckID := int64(bd["id"].(float64))

	resp, body := doJSON(t, "POST", base+"/api/v1/marvel/games", t1, createGameRequest{DeckIDs: []int64{deckID}, ScenarioID: "01097"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create game: %d %v", resp.StatusCode, body)
	}
	view := body
	gameID := int64(view["id"].(float64))

	// u2 sees no hand codes and no question
	resp, body = doJSON(t, "GET", fmt.Sprintf("%s/api/v1/marvel/games/%d", base, gameID), t2, nil)
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
	resp, body = doJSON(t, "POST", fmt.Sprintf("%s/api/v1/marvel/games/%d/answer", base, gameID), t2, answerRequest{Paths: []string{"0"}})
	if resp.StatusCode == http.StatusOK {
		t.Fatal("u2 must not answer u1's question")
	}
}
