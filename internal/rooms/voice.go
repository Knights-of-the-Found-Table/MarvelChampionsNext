package rooms

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// 语音信令：服务器只在同桌成员之间转发 WebRTC SDP offer/answer 与 ICE
// candidate，并维护"谁在语音里"的在场表。媒体永远点对点——服务器不转发
// 音频、不录音、不落任何媒体相关状态；在场表在内存里，重启即清空。

const (
	// 单条信令帧的载荷上限。SDP/ICE 都远小于此，纯防御性封顶。
	VoiceSignalLimit = 8 * 1024
	// 每连接的下行缓冲。信令量小（每对连接一次 offer/answer + 少量
	// candidate），慢消费者按既有惯例丢帧，绝不阻塞牌局锁。
	voiceChanBuffer = 64
)

// VoicePeer 描述语音房间里的一个连接。id 由服务器随机分配，是 mesh 的
// peer id（同一用户开两个标签页就是两个 peer，互不冲突）。
type VoicePeer struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Name      string `json:"name"`
	Spectator bool   `json:"spectator"`
	Muted     bool   `json:"muted"`
}

type voiceConn struct {
	peer VoicePeer
	ch   chan []byte
}

// voiceID 生成连接 id；带 "v" 前缀便于日志里区分于用户/游戏 id。
func voiceID() (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return "v" + hex.EncodeToString(buf[:]), nil
}

// VoiceJoin 把一个已认证的成员登记进房间的语音在场表，向其下发
// voice-welcome（自己的 id + 现有同伴），并向其他人广播 voice-peer-joined。
// 返回的 cancel 在连接结束时调用：移除在场表并广播 voice-peer-left。
// 参与者与旁观者一视同仁（spectator 标记仅供 UI 显示）。
func (m *Manager) VoiceJoin(gameID int64, userID, name string) (VoicePeer, <-chan []byte, func(), error) {
	r, err := m.Get(gameID)
	if err != nil {
		return VoicePeer{}, nil, nil, err
	}
	id, err := voiceID()
	if err != nil {
		return VoicePeer{}, nil, nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	peer := VoicePeer{ID: id, UserID: userID, Name: name, Spectator: !r.isParticipant(userID)}
	vc := &voiceConn{peer: peer, ch: make(chan []byte, voiceChanBuffer)}
	if r.voice == nil {
		r.voice = map[string]*voiceConn{}
	}
	r.voice[id] = vc

	peers := make([]VoicePeer, 0, len(r.voice)-1)
	for _, other := range r.voice {
		if other.peer.ID != id {
			peers = append(peers, other.peer)
		}
	}
	if b, err := json.Marshal(map[string]any{"type": "voice-welcome", "self": peer, "peers": peers}); err == nil {
		select {
		case vc.ch <- b:
		default:
		}
	}
	if b, err := json.Marshal(map[string]any{"type": "voice-peer-joined", "peer": peer}); err == nil {
		r.broadcastVoiceLocked(id, b)
	}

	cancel := func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if _, ok := r.voice[id]; !ok {
			return
		}
		delete(r.voice, id)
		close(vc.ch)
		if b, err := json.Marshal(map[string]any{"type": "voice-peer-left", "id": id}); err == nil {
			r.broadcastVoiceLocked(id, b)
		}
	}
	return peer, vc.ch, cancel, nil
}

// VoiceRelay 把一条信令载荷（SDP/ICE，服务器不解释）原样转发给同桌目标
// 连接。from 由持有连接身份的 handler 注入，客户端无法伪造来源。目标已
// 离开时静默返回错误——对端的 voice-peer-left 会随后到达并清理 mesh。
func (m *Manager) VoiceRelay(gameID int64, from, to string, data json.RawMessage) error {
	if len(data) > VoiceSignalLimit {
		return fmt.Errorf("signal too large")
	}
	r, err := m.Get(gameID)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	vc, ok := r.voice[to]
	if !ok {
		return fmt.Errorf("peer gone")
	}
	b, err := json.Marshal(map[string]any{"type": "voice-signal", "from": from, "to": to, "data": data})
	if err != nil {
		return err
	}
	select {
	case vc.ch <- b:
	default:
	}
	return nil
}

// VoiceSetMuted 更新自己连接的静音标记并广播给同桌（UI 显示 🔇）。
// 静音本身完全在客户端轨道上实现（track.enabled=false），服务器只转发标记。
func (m *Manager) VoiceSetMuted(gameID int64, selfID string, muted bool) {
	r, err := m.Get(gameID)
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	vc, ok := r.voice[selfID]
	if !ok {
		return
	}
	vc.peer.Muted = muted
	if b, err := json.Marshal(map[string]any{"type": "voice-peer-muted", "id": selfID, "muted": muted}); err == nil {
		r.broadcastVoiceLocked(selfID, b)
	}
}

// broadcastVoiceLocked 向除 exceptID 外的所有语音成员非阻塞发送。
// 调用方必须持有 r.mu。
func (r *Room) broadcastVoiceLocked(exceptID string, b []byte) {
	for id, vc := range r.voice {
		if id == exceptID {
			continue
		}
		select {
		case vc.ch <- b:
		default:
		}
	}
}
