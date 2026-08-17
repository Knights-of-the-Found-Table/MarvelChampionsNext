package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/rooms"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/store"

	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
)

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
		Secret: []byte("test-secret"),
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
			c := choices[0].(map[string]any)
			paths = []string{c["id"].(string)}
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
