package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/rooms"
)

// handleVoiceTest 是 /voice-test/ 静态测试页的信令端点：匿名、按路径里
// 的房间名分组转发。不走 s.auth（测试页要让未登录的真机直连验证），
// 因此 from 同样由服务器盖章、载荷长度封顶、成员数与房间数在 Hub 侧
// 有上限。只转发信令，媒体永远点对点。
func (s *Server) handleVoiceTest(w http.ResponseWriter, r *http.Request) {
	room := r.PathValue("room")
	name := r.URL.Query().Get("name")

	self, ch, cancel, err := s.VoiceTest.VoiceTestJoin(room, name)
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		cancel()
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "bye")
	conn.SetReadLimit(rooms.VoiceSignalLimit + 1024)

	ctx := r.Context()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			mt, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			if mt != websocket.MessageText {
				continue
			}
			var req struct {
				Type  string          `json:"type"`
				To    string          `json:"to"`
				Data  json.RawMessage `json:"data"`
				Muted bool            `json:"muted"`
			}
			if json.Unmarshal(data, &req) != nil {
				continue
			}
			switch req.Type {
			case "voice-signal":
				if req.To == "" || len(req.Data) == 0 {
					continue
				}
				_ = s.VoiceTest.VoiceTestRelay(room, self.ID, req.To, req.Data)
			case "voice-state":
				s.VoiceTest.VoiceTestSetMuted(room, self.ID, req.Muted)
			case "voice-ping":
			}
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		case <-done:
			cancel()
			return
		case <-ticker.C:
			if err := conn.Ping(ctx); err != nil {
				cancel()
				return
			}
		case msg, ok := <-ch:
			if !ok {
				cancel()
				return
			}
			if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
				cancel()
				return
			}
		}
	}
}
