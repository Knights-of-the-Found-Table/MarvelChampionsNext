package rooms

import (
	"encoding/json"
	"fmt"
	"sync"
)

// 语音测试厅：与对局无关的匿名信令房间（/voice-test/ 静态页用）。
// 协议与对局语音一致（voice-welcome / peer-joined / signal / peer-left /
// peer-muted），但不绑定游戏、不要求登录——房间名本身就是门槛，仅用于
// 部署后真机验证 WebRTC 连通性。媒体仍永远点对点，服务器只转发信令。
// 内存态，重启即清空；房间在最后一个成员离开时销毁。

const (
	// 测试厅上限：防御恶意扫房间名灌内存。正常测试一只手数得过来。
	voiceTestMaxRooms = 64
	// 单房间成员上限：mesh 规模本来就只适合小房间。
	voiceTestMaxPeers = 8
)

type voiceTestRoom struct {
	conns map[string]*voiceConn
}

type VoiceTestHub struct {
	mu    sync.Mutex
	rooms map[string]*voiceTestRoom
}

func NewVoiceTestHub() *VoiceTestHub {
	return &VoiceTestHub{rooms: map[string]*voiceTestRoom{}}
}

// VoiceTestJoin 把匿名连接登记进命名房间。name 是显示名（客户端自报，
// 仅用于 UI，无任何信任含义）。返回结构同 VoiceJoin。
func (h *VoiceTestHub) VoiceTestJoin(room, name string) (VoicePeer, <-chan []byte, func(), error) {
	if room == "" || len(room) > 64 {
		return VoicePeer{}, nil, nil, fmt.Errorf("bad room name")
	}
	if name == "" {
		name = "anonymous"
	}
	if len([]rune(name)) > 32 {
		name = string([]rune(name)[:32])
	}
	id, err := voiceID()
	if err != nil {
		return VoicePeer{}, nil, nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	tr, ok := h.rooms[room]
	if !ok {
		if len(h.rooms) >= voiceTestMaxRooms {
			return VoicePeer{}, nil, nil, fmt.Errorf("too many rooms")
		}
		tr = &voiceTestRoom{conns: map[string]*voiceConn{}}
		h.rooms[room] = tr
	}
	if len(tr.conns) >= voiceTestMaxPeers {
		return VoicePeer{}, nil, nil, fmt.Errorf("room full")
	}

	peer := VoicePeer{ID: id, Name: name}
	vc := &voiceConn{peer: peer, ch: make(chan []byte, voiceChanBuffer)}
	tr.conns[id] = vc

	peers := make([]VoicePeer, 0, len(tr.conns)-1)
	for _, other := range tr.conns {
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
		broadcastVoiceTestLocked(tr, id, b)
	}

	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		tr, ok := h.rooms[room]
		if !ok {
			return
		}
		if _, ok := tr.conns[id]; !ok {
			return
		}
		delete(tr.conns, id)
		close(vc.ch)
		if b, err := json.Marshal(map[string]any{"type": "voice-peer-left", "id": id}); err == nil {
			broadcastVoiceTestLocked(tr, id, b)
		}
		if len(tr.conns) == 0 {
			delete(h.rooms, room)
		}
	}
	return peer, vc.ch, cancel, nil
}

// VoiceTestRelay 同 VoiceRelay：载荷原样转发，from 由 handler 注入。
func (h *VoiceTestHub) VoiceTestRelay(room, from, to string, data json.RawMessage) error {
	if len(data) > VoiceSignalLimit {
		return fmt.Errorf("signal too large")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	tr, ok := h.rooms[room]
	if !ok {
		return fmt.Errorf("room gone")
	}
	vc, ok := tr.conns[to]
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

// VoiceTestSetMuted 同 VoiceSetMuted。
func (h *VoiceTestHub) VoiceTestSetMuted(room, selfID string, muted bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	tr, ok := h.rooms[room]
	if !ok {
		return
	}
	vc, ok := tr.conns[selfID]
	if !ok {
		return
	}
	vc.peer.Muted = muted
	if b, err := json.Marshal(map[string]any{"type": "voice-peer-muted", "id": selfID, "muted": muted}); err == nil {
		broadcastVoiceTestLocked(tr, selfID, b)
	}
}

func broadcastVoiceTestLocked(tr *voiceTestRoom, exceptID string, b []byte) {
	for id, vc := range tr.conns {
		if id == exceptID {
			continue
		}
		select {
		case vc.ch <- b:
		default:
		}
	}
}
