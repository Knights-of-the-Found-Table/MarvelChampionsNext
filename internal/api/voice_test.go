package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// dialVoice opens a voice signaling WebSocket against the test server.
func dialVoice(t *testing.T, ts *httptest.Server, gameID, token string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "ws://" + ts.Listener.Addr().String() + "/api/v1/marvel/games/" + gameID + "/voice"
	if token != "" {
		url += "?token=" + token
	}
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("voice dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })
	return conn
}

// readFrame waits for the next JSON frame whose type is in want.
func readFrame(t *testing.T, conn *websocket.Conn, want ...string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, data, err := conn.Read(ctx)
		cancel()
		if err != nil {
			t.Fatalf("voice read: %v", err)
		}
		var m map[string]any
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		for _, w := range want {
			if m["type"] == w {
				return m
			}
		}
	}
	t.Fatalf("no frame of types %v arrived", want)
	return nil
}

func writeFrame(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	b, _ := json.Marshal(v)
	if err := conn.Write(ctx, websocket.MessageText, b); err != nil {
		t.Fatalf("voice write: %v", err)
	}
}

// TestVoiceSignaling pins the signaling relay contract: welcome carries self
// id and current peers, peer-joined/left broadcast, signals relay untouched
// with a server-stamped from, mute markers propagate, and anonymous dials
// are rejected before the upgrade.
func TestVoiceSignaling(t *testing.T) {
	ts, _ := newTestServer(t)
	base := ts.URL

	// two users, a legal deck, and a live (solo) game
	_, b := doJSON(t, "POST", base+"/api/v1/register", "", credentials{Username: "voalice", Password: "secret123"})
	if tokenA := b["token"].(string); tokenA == "" {
		t.Fatal("register A failed")
	}
	tokenA := b["token"].(string)
	_, b = doJSON(t, "POST", base+"/api/v1/register", "", credentials{Username: "vobob", Password: "secret123"})
	tokenB := b["token"].(string)

	_, b = doJSON(t, "POST", base+"/api/v1/marvel/decks", tokenA, importDeckRequest{
		Name:             "Voice Spidey",
		InvestigatorCode: "01001a",
		Slots:            legalSpideyDeck(),
	})
	deckID := b["id"].(string)
	_, b = doJSON(t, "POST", base+"/api/v1/marvel/games", tokenA, createGameRequest{
		DeckIDs:    []string{deckID},
		ScenarioID: "01097",
	})
	gameID := b["id"].(string)

	// anonymous dial is refused before upgrade
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, resp, err := websocket.Dial(ctx, "ws://"+strings.TrimPrefix(base, "http://")+"/api/v1/marvel/games/"+gameID+"/voice", nil)
	if err == nil {
		t.Fatal("anonymous voice dial should fail")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous dial: want 401, got %d", resp.StatusCode)
	}

	// Alice joins first: welcome with self, no peers yet.
	connA := dialVoice(t, ts, gameID, tokenA)
	welcome := readFrame(t, connA, "voice-welcome")
	selfA, _ := welcome["self"].(map[string]any)
	if selfA["id"] == "" || selfA["name"] != "voalice" {
		t.Fatalf("welcome self malformed: %v", welcome)
	}
	if selfA["spectator"] != false {
		t.Fatalf("owner should not be a spectator: %v", selfA)
	}

	// Bob (not a participant) joins: he is a spectator peer.
	connB := dialVoice(t, ts, gameID, tokenB)
	welcomeB := readFrame(t, connB, "voice-welcome")
	selfB, _ := welcomeB["self"].(map[string]any)
	if selfB["spectator"] != true {
		t.Fatalf("non-participant should be flagged spectator: %v", selfB)
	}
	peers, _ := welcomeB["peers"].([]any)
	if len(peers) != 1 {
		t.Fatalf("welcome should list Alice, got %v", peers)
	}
	peerA, _ := peers[0].(map[string]any)
	if peerA["id"] != selfA["id"] {
		t.Fatalf("peer id mismatch: %v vs %v", peerA["id"], selfA["id"])
	}

	// Alice hears about Bob.
	joined := readFrame(t, connA, "voice-peer-joined")
	joinedPeer, _ := joined["peer"].(map[string]any)
	if joinedPeer["id"] != selfB["id"] {
		t.Fatalf("peer-joined id mismatch: %v", joined)
	}

	// Bob relays a signal to Alice: payload passes through untouched and
	// `from` is the server-stamped connection id.
	writeFrame(t, connB, map[string]any{
		"type": "voice-signal",
		"to":   selfA["id"],
		"data": map[string]any{"sdp": map[string]any{"type": "offer", "sdp": "v=0 fake"}},
	})
	sig := readFrame(t, connA, "voice-signal")
	if sig["from"] != selfB["id"] || sig["to"] != selfA["id"] {
		t.Fatalf("relay headers wrong: %v", sig)
	}
	data, _ := sig["data"].(map[string]any)
	sdp, _ := data["sdp"].(map[string]any)
	if sdp["sdp"] != "v=0 fake" {
		t.Fatalf("relay payload altered: %v", sig)
	}

	// Signals to unknown peers are dropped silently (no echo, no error frame).
	writeFrame(t, connB, map[string]any{"type": "voice-signal", "to": "vdeadbeef", "data": map[string]any{"x": 1}})

	// Mute marker propagates to the table.
	writeFrame(t, connB, map[string]any{"type": "voice-state", "muted": true})
	muted := readFrame(t, connA, "voice-peer-muted")
	if muted["id"] != selfB["id"] || muted["muted"] != true {
		t.Fatalf("mute broadcast wrong: %v", muted)
	}

	// Bob leaves: Alice gets peer-left.
	_ = connB.Close(websocket.StatusNormalClosure, "bye")
	left := readFrame(t, connA, "voice-peer-left")
	if left["id"] != selfB["id"] {
		t.Fatalf("peer-left id mismatch: %v", left)
	}
}
