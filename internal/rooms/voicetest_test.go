package rooms

import (
	"encoding/json"
	"testing"
	"time"
)

func recvMsg(t *testing.T, ch <-chan []byte) map[string]any {
	t.Helper()
	select {
	case b := <-ch:
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("bad frame: %v", err)
		}
		return m
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for frame")
		return nil
	}
}

func TestVoiceTestHub(t *testing.T) {
	h := NewVoiceTestHub()

	a, chA, cancelA, err := h.VoiceTestJoin("r1", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	defer cancelA()
	welcome := recvMsg(t, chA)
	if welcome["type"] != "voice-welcome" {
		t.Fatalf("welcome: %v", welcome)
	}
	if peers := welcome["peers"].([]any); len(peers) != 0 {
		t.Fatalf("first join should see no peers: %v", peers)
	}

	b, chB, cancelB, err := h.VoiceTestJoin("r1", "Bob")
	if err != nil {
		t.Fatal(err)
	}
	defer cancelB()
	// Bob 的 welcome 里应看到 Alice；Alice 收到 peer-joined。
	wb := recvMsg(t, chB)
	if peers := wb["peers"].([]any); len(peers) != 1 {
		t.Fatalf("second join should see one peer: %v", peers)
	}
	if j := recvMsg(t, chA); j["type"] != "voice-peer-joined" {
		t.Fatalf("peer-joined: %v", j)
	}

	// 信令转发：from 由 hub 注入，载荷原样。
	if err := h.VoiceTestRelay("r1", a.ID, b.ID, json.RawMessage(`{"sdp":"x"}`)); err != nil {
		t.Fatal(err)
	}
	sig := recvMsg(t, chB)
	if sig["type"] != "voice-signal" || sig["from"] != a.ID {
		t.Fatalf("signal: %v", sig)
	}
	if err := h.VoiceTestRelay("r1", a.ID, b.ID, make(json.RawMessage, VoiceSignalLimit+1)); err == nil {
		t.Fatal("oversized signal should be rejected")
	}
	if err := h.VoiceTestRelay("r1", a.ID, "vgone", json.RawMessage(`{}`)); err == nil {
		t.Fatal("relay to missing peer should error")
	}

	// 静音广播。
	h.VoiceTestSetMuted("r1", a.ID, true)
	if m := recvMsg(t, chB); m["type"] != "voice-peer-muted" || m["muted"] != true {
		t.Fatalf("muted: %v", m)
	}

	// 全员离开后房间销毁。
	cancelB()
	if l := recvMsg(t, chA); l["type"] != "voice-peer-left" {
		t.Fatalf("peer-left: %v", l)
	}
	cancelA()
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.rooms) != 0 {
		t.Fatalf("empty room should be destroyed, have %d", len(h.rooms))
	}
}

func TestVoiceTestHubLimits(t *testing.T) {
	h := NewVoiceTestHub()
	var cancels []func()
	defer func() {
		for _, c := range cancels {
			c()
		}
	}()
	for i := 0; i < voiceTestMaxPeers; i++ {
		_, _, c, err := h.VoiceTestJoin("full", "")
		if err != nil {
			t.Fatalf("join %d: %v", i, err)
		}
		cancels = append(cancels, c)
	}
	if _, _, _, err := h.VoiceTestJoin("full", ""); err == nil {
		t.Fatal("room should be full")
	}
	if _, _, _, err := h.VoiceTestJoin("", ""); err == nil {
		t.Fatal("empty room name should be rejected")
	}
}
