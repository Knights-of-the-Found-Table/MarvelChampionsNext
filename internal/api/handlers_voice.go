package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/rooms"
)

// handleVoice 是语音信令 WebSocket：认证后加入该桌的语音在场表，客户端
// 帧只接受三类——voice-signal（转发 SDP/ICE 给指定同伴）、voice-state
// （静音标记广播）、voice-ping（保活，忽略）。服务器每 30s ping 一次：
// 既穿透代理空闲超时，也回收断网的僵尸连接（ping 失败即触发 peer-left）。
// 服务器不接触媒体；与对局 stream WS 完全独立，语音断线不影响观战流。
func (s *Server) handleVoice(w http.ResponseWriter, r *http.Request) {
	gameID, ok := s.pathGame(w, r)
	if !ok {
		return
	}
	// 语音必须可识别身份：from 字段的信任来自服务器盖章（auth 中间件
	// 已验过 ?token=），匿名连接在进入 handler 前就被拒了。
	uid := userID(r)
	var uidNum int64
	if _, err := fmt.Sscanf(uid, "%d", &uidNum); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return
	}
	user, err := s.Store.UserByID(uidNum)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return
	}

	self, ch, cancel, err := s.Rooms.VoiceJoin(gameID, uid, user.Username)
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
				// from 由服务器注入（self.ID），载荷原样转发不解释。
				_ = s.Rooms.VoiceRelay(gameID, self.ID, req.To, req.Data)
			case "voice-state":
				s.Rooms.VoiceSetMuted(gameID, self.ID, req.Muted)
			case "voice-ping":
				// keepalive; nothing to do
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
				cancel() // 幂等：close(ch) 通常就是它干的，防未来其它路径
				return
			}
			if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
				cancel()
				return
			}
		}
	}
}
