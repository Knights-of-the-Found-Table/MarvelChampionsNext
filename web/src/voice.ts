// 牌桌语音：WebRTC 全互联 mesh（4 人以内完全够用），服务器只做信令。
// 本模块拥有完整生命周期——信令 WS、每对同伴一条 RTCPeerConnection、
// 本地麦克风、远端音频回放（隐藏 <audio>）与说话检测（Web Audio RMS）。
// 断线策略：信令 WS 意外断开即整体拆网并上报，由用户手动重进——mesh
// 半死状态（部分 pc 活着）比干净重来更难排查。
import { useCallback, useEffect, useRef, useState } from 'react'
import { getToken } from './api'

export interface VoicePeerInfo {
  id: string
  userId: string
  name: string
  spectator: boolean
  muted: boolean
}

export type VoiceStatus = 'idle' | 'connecting' | 'live' | 'error'

// 说话检测参数：RMS 超阈值记一次活跃，450ms 内有活跃算"说话中"。
const SPEAK_RMS = 0.04
const SPEAK_HOLD_MS = 450
const METER_TICK_MS = 150

const ICE_SERVERS: RTCConfiguration = {
  iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
}

interface PeerEntry {
  info: VoicePeerInfo
  pc: RTCPeerConnection
  pendingIce: RTCIceCandidateInit[]
  remoteReady: boolean
}

interface Meter {
  analyser: AnalyserNode
  // TS 5.7 的 DOM 类型要求 getFloatTimeDomainData 接受 Float32Array<ArrayBuffer>。
  buf: Float32Array<ArrayBuffer>
  lastActive: number
}

export interface VoiceEvents {
  onStatus: (status: VoiceStatus, errorKey?: string) => void
  onPeers: (peers: VoicePeerInfo[], self: VoicePeerInfo | null) => void
  onSpeaking: (ids: ReadonlySet<string>) => void
}

export class VoiceSession {
  private ws: WebSocket | null = null
  private self: VoicePeerInfo | null = null
  private peers = new Map<string, PeerEntry>()
  private local: MediaStream | null = null
  private audioEls = new Map<string, HTMLAudioElement>()
  private meters = new Map<string, Meter>()
  private audioCtx: AudioContext | null = null
  private meterTimer: number | null = null
  private muted = false
  private speaking = new Set<string>()
  private closed = false

  constructor(
    private gameId: string,
    private events: VoiceEvents,
  ) {}

  /** 加入语音：请求麦克风 → 连信令 WS。失败返回 false（错误经 onStatus 上报）。 */
  async join(): Promise<boolean> {
    this.events.onStatus('connecting')
    if (!navigator.mediaDevices?.getUserMedia) {
      this.events.onStatus('idle', 'voice.micUnavailable')
      return false
    }
    try {
      this.local = await navigator.mediaDevices.getUserMedia({
        audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true },
        video: false,
      })
    } catch (e) {
      const name = e instanceof DOMException ? e.name : ''
      const key = name === 'NotAllowedError' || name === 'SecurityError' ? 'voice.micDenied' : 'voice.micUnavailable'
      this.events.onStatus('idle', key)
      return false
    }
    try {
      const AC: typeof AudioContext | undefined =
        window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
      if (AC) {
        this.audioCtx = new AC()
        void this.audioCtx.resume().catch(() => undefined)
      }
    } catch {
      this.audioCtx = null // 说话指示降级为永远不亮，语音本体不受影响
    }
    if (this.local) this.meterStream('self', this.local)

    const token = getToken() ?? ''
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(
      `${proto}://${location.host}/api/v1/marvel/games/${this.gameId}/voice?token=${encodeURIComponent(token)}`,
    )
    this.ws = ws
    ws.onmessage = (ev) => this.onFrame(ev.data)
    ws.onclose = () => {
      if (this.closed) return
      // 意外断线（服务器重启 / 网络抖动）：拆干净，让用户重进。
      this.teardown()
      this.events.onStatus('error', 'voice.disconnected')
    }
    return true
  }

  /** 切换静音。静音 = 本地轨道禁用（连接保持），并广播标记给同桌。 */
  toggleMute(): boolean {
    this.muted = !this.muted
    this.local?.getAudioTracks().forEach((t) => (t.enabled = !this.muted))
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: 'voice-state', muted: this.muted }))
    }
    this.emitPeers()
    return this.muted
  }

  /** 干净离开：拆 mesh、停轨道、关 WS。幂等。 */
  leave() {
    this.teardown()
  }

  // ---------------------------------------------------------------- signaling

  private onFrame(raw: MessageEvent['data'] | string) {
    let msg: {
      type: string
      self?: VoicePeerInfo
      peers?: VoicePeerInfo[]
      peer?: VoicePeerInfo
      id?: string
      muted?: boolean
      from?: string
      to?: string
      data?: { sdp?: RTCSessionDescriptionInit; candidate?: RTCIceCandidateInit }
    }
    try {
      msg = JSON.parse(String(raw))
    } catch {
      return
    }
    switch (msg.type) {
      case 'voice-welcome': {
        this.self = msg.self ?? null
        for (const p of msg.peers ?? []) this.addPeer(p)
        this.events.onStatus('live')
        this.startMeter()
        this.emitPeers()
        break
      }
      case 'voice-peer-joined':
        if (msg.peer) this.addPeer(msg.peer)
        break
      case 'voice-peer-left':
        if (msg.id) this.removePeer(msg.id)
        break
      case 'voice-peer-muted': {
        const entry = msg.id ? this.peers.get(msg.id) : undefined
        if (entry && typeof msg.muted === 'boolean') {
          entry.info.muted = msg.muted
          this.emitPeers()
        }
        break
      }
      case 'voice-signal':
        if (msg.from && msg.data) void this.onSignal(msg.from, msg.data)
        break
    }
  }

  private send(to: string, data: unknown) {
    if (this.ws?.readyState !== WebSocket.OPEN) return
    this.ws.send(JSON.stringify({ type: 'voice-signal', to, data }))
  }

  // ---------------------------------------------------------------- mesh

  // 发起方判定：id 字典序小的一方 createOffer。双方都能对同一对 id 算出
  // 一致结果，杜绝 offer/answer 迎头相撞（glare）。
  private addPeer(info: VoicePeerInfo) {
    if (this.peers.has(info.id) || info.id === this.self?.id) return
    const pc = new RTCPeerConnection(ICE_SERVERS)
    const entry: PeerEntry = { info, pc, pendingIce: [], remoteReady: false }
    this.peers.set(info.id, entry)

    this.local?.getTracks().forEach((t) => pc.addTrack(t, this.local!))
    pc.onicecandidate = (e) => {
      if (e.candidate) this.send(info.id, { candidate: e.candidate.toJSON() })
    }
    pc.ontrack = (e) => {
      const [stream] = e.streams
      if (!stream) return
      this.attach(info.id, stream)
    }
    if (this.self && this.self.id < info.id) void this.makeOffer(entry)
    this.emitPeers()
  }

  private removePeer(id: string) {
    const entry = this.peers.get(id)
    if (!entry) return
    this.peers.delete(id)
    try {
      entry.pc.close()
    } catch {
      /* already closed */
    }
    this.detach(id)
    this.emitPeers()
  }

  private async makeOffer(entry: PeerEntry) {
    try {
      const offer = await entry.pc.createOffer()
      await entry.pc.setLocalDescription(offer)
      this.send(entry.info.id, { sdp: entry.pc.localDescription?.toJSON() ?? offer })
    } catch {
      this.removePeer(entry.info.id)
    }
  }

  private async onSignal(from: string, data: { sdp?: RTCSessionDescriptionInit; candidate?: RTCIceCandidateInit }) {
    const entry = this.peers.get(from)
    if (!entry) return // 对端刚离开、peer-left 在途：丢弃
    try {
      if (data.sdp) {
        await entry.pc.setRemoteDescription(new RTCSessionDescription(data.sdp))
        entry.remoteReady = true
        for (const c of entry.pendingIce.splice(0)) {
          await entry.pc.addIceCandidate(c).catch(() => undefined)
        }
        if (data.sdp.type === 'offer') {
          const answer = await entry.pc.createAnswer()
          await entry.pc.setLocalDescription(answer)
          this.send(from, { sdp: entry.pc.localDescription?.toJSON() ?? answer })
        }
      } else if (data.candidate) {
        if (entry.remoteReady) await entry.pc.addIceCandidate(data.candidate).catch(() => undefined)
        else entry.pendingIce.push(data.candidate)
      }
    } catch {
      // 信令错乱（竞态/格式）时拆掉这一条重比挂着强。
      this.removePeer(from)
    }
  }

  // ---------------------------------------------------------------- audio

  // 远端回放走独立 <audio>（浏览器自动播放策略在用户点击加入后放行），
  // AudioContext 只做电平采样，不参与输出路由。
  private attach(id: string, stream: MediaStream) {
    this.detach(id)
    const el = document.createElement('audio')
    el.autoplay = true
    el.style.display = 'none'
    el.srcObject = stream
    document.body.appendChild(el)
    void el.play().catch(() => undefined)
    this.audioEls.set(id, el)
    this.meterStream(id, stream)
  }

  private detach(id: string) {
    this.audioEls.get(id)?.remove()
    this.audioEls.delete(id)
    this.meters.get(id)?.analyser.disconnect()
    this.meters.delete(id)
  }

  private meterStream(id: string, stream: MediaStream) {
    if (!this.audioCtx) return
    try {
      const src = this.audioCtx.createMediaStreamSource(stream)
      const analyser = this.audioCtx.createAnalyser()
      analyser.fftSize = 512
      src.connect(analyser) // 只接 analyser，不接 destination：静默采样
      this.meters.set(id, { analyser, buf: new Float32Array(analyser.fftSize), lastActive: 0 })
    } catch {
      /* 该路没有说话指示 */
    }
  }

  private startMeter() {
    if (this.meterTimer != null || !this.audioCtx) return
    this.meterTimer = window.setInterval(() => {
      const now = performance.now()
      const next = new Set<string>()
      for (const [id, m] of this.meters) {
        m.analyser.getFloatTimeDomainData(m.buf)
        let sum = 0
        for (let i = 0; i < m.buf.length; i++) sum += m.buf[i] * m.buf[i]
        if (Math.sqrt(sum / m.buf.length) > SPEAK_RMS) m.lastActive = now
        // 自己静音时不算说话；对方静音时对端已停发音频，RMS 自然归零。
        if (now - m.lastActive < SPEAK_HOLD_MS && !(id === 'self' && this.muted)) next.add(id)
      }
      if (!sameSet(next, this.speaking)) {
        this.speaking = next
        this.events.onSpeaking(next)
      }
    }, METER_TICK_MS)
  }

  private emitPeers() {
    this.events.onPeers([...this.peers.values()].map((e) => ({ ...e.info })), this.self ? { ...this.self } : null)
  }

  private teardown() {
    this.closed = true
    if (this.meterTimer != null) {
      window.clearInterval(this.meterTimer)
      this.meterTimer = null
    }
    for (const entry of this.peers.values()) {
      try {
        entry.pc.close()
      } catch {
        /* already closed */
      }
    }
    this.peers.clear()
    for (const id of [...this.audioEls.keys()]) this.detach(id)
    this.local?.getTracks().forEach((t) => t.stop())
    this.local = null
    if (this.audioCtx) {
      void this.audioCtx.close().catch(() => undefined)
      this.audioCtx = null
    }
    this.meters.clear()
    const ws = this.ws
    this.ws = null
    if (ws && ws.readyState <= WebSocket.OPEN) ws.close()
    this.self = null
    this.speaking = new Set()
    this.events.onSpeaking(this.speaking)
    this.events.onPeers([], null)
  }
}

function sameSet(a: ReadonlySet<string>, b: ReadonlySet<string>): boolean {
  if (a.size !== b.size) return false
  for (const v of a) if (!b.has(v)) return false
  return true
}

// ---------------------------------------------------------------- hook

// useVoice：VoiceSession 的 React 绑定。状态只在事件到达时更新；
// gameId 变化或组件卸载时整体拆除。
export function useVoice(gameId: string) {
  const [status, setStatus] = useState<VoiceStatus>('idle')
  const [errorKey, setErrorKey] = useState('')
  const [peers, setPeers] = useState<VoicePeerInfo[]>([])
  const [self, setSelf] = useState<VoicePeerInfo | null>(null)
  const [speaking, setSpeaking] = useState<ReadonlySet<string>>(() => new Set())
  const [muted, setMuted] = useState(false)
  const sessionRef = useRef<VoiceSession | null>(null)

  useEffect(
    () => () => {
      sessionRef.current?.leave()
      sessionRef.current = null
    },
    [gameId],
  )

  const join = useCallback(async () => {
    if (sessionRef.current) return
    setErrorKey('')
    const session = new VoiceSession(gameId, {
      onStatus: (s, key) => {
        setStatus(s)
        if (key) setErrorKey(key)
        if (s === 'idle' || s === 'error') sessionRef.current = null
      },
      onPeers: (list, me) => {
        setPeers(list)
        setSelf(me)
      },
      onSpeaking: setSpeaking,
    })
    sessionRef.current = session
    const ok = await session.join()
    if (!ok) sessionRef.current = null
  }, [gameId])

  const leave = useCallback(() => {
    sessionRef.current?.leave()
    sessionRef.current = null
    setStatus('idle')
    setErrorKey('')
    setPeers([])
    setSelf(null)
    setSpeaking(new Set())
    setMuted(false)
  }, [])

  const toggleMute = useCallback(() => {
    const s = sessionRef.current
    if (!s) return false
    const next = s.toggleMute()
    setMuted(next)
    return next
  }, [])

  return { status, errorKey, peers, self, speaking, muted, join, leave, toggleMute }
}
