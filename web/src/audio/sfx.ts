// WebAudio 程序化音效（暂时资产，零素材、无版权问题）。所有声音由振荡器
// 与噪声实时合成；接口按"可替换为真实素材"设计——将来只需把 play 换成
// <audio> 采样播放。AudioContext 在首次用户手势时惰性创建（自动播放策略）。
//
// 设置持久化在 localStorage：muted（静音）。

export type SfxName =
  | 'damage'
  | 'heal'
  | 'threat'
  | 'thwart'
  | 'attack'
  | 'status'
  | 'play'
  | 'select'
  | 'victory'
  | 'defeat'

const STORAGE_KEY = 'mc-sfx'

interface SfxSettings {
  muted: boolean
  volume: number
}

let ctx: AudioContext | null = null
let master: GainNode | null = null
let settings: SfxSettings = loadSettings()

function loadSettings(): SfxSettings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return { muted: false, volume: 0.5, ...JSON.parse(raw) }
  } catch {
    /* ignore */
  }
  return { muted: false, volume: 0.5 }
}

function saveSettings() {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
  } catch {
    /* ignore */
  }
}

export function sfxSettings(): SfxSettings {
  return { ...settings }
}

export function setSfxMuted(muted: boolean) {
  settings.muted = muted
  saveSettings()
}

export function setSfxVolume(v: number) {
  settings.volume = Math.max(0, Math.min(1, v))
  saveSettings()
}

// 在首次手势时调用：创建/resume AudioContext。安全重复调用。
export function initSfx() {
  const ensure = () => {
    if (!ctx) {
      const AC = window.AudioContext ?? (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext
      if (!AC) return
      ctx = new AC()
      master = ctx.createGain()
      master.gain.value = settings.volume
      master.connect(ctx.destination)
    }
    if (ctx.state === 'suspended') void ctx.resume()
  }
  window.addEventListener('pointerdown', ensure, { once: false })
  window.addEventListener('keydown', ensure, { once: false })
  ensure()
}

// ---------------------------------------------------------------- 合成器

function now(): number {
  return ctx ? ctx.currentTime : 0
}

// 白噪声缓冲（缓存一份复用）。
let noiseBuf: AudioBuffer | null = null
function noise(): AudioBuffer | null {
  if (!ctx) return null
  if (!noiseBuf) {
    noiseBuf = ctx.createBuffer(1, ctx.sampleRate * 1, ctx.sampleRate)
    const data = noiseBuf.getChannelData(0)
    for (let i = 0; i < data.length; i++) data[i] = Math.random() * 2 - 1
  }
  return noiseBuf
}

function tone(
  freq: number,
  opts: {
    type?: OscillatorType
    dur?: number
    at?: number
    gain?: number
    slideTo?: number
    attack?: number
  } = {}
) {
  if (!ctx || !master) return
  const { type = 'sine', dur = 0.2, at = 0, gain = 0.4, slideTo, attack = 0.005 } = opts
  const t0 = now() + at
  const osc = ctx.createOscillator()
  osc.type = type
  osc.frequency.setValueAtTime(freq, t0)
  if (slideTo !== undefined) osc.frequency.exponentialRampToValueAtTime(Math.max(1, slideTo), t0 + dur)
  const g = ctx.createGain()
  g.gain.setValueAtTime(0, t0)
  g.gain.linearRampToValueAtTime(gain, t0 + attack)
  g.gain.exponentialRampToValueAtTime(0.0001, t0 + dur)
  osc.connect(g).connect(master)
  osc.start(t0)
  osc.stop(t0 + dur + 0.05)
}

function noiseBurst(
  opts: {
    dur?: number
    at?: number
    gain?: number
    filter?: BiquadFilterType
    from?: number
    to?: number
    q?: number
  } = {}
) {
  if (!ctx || !master) return
  const { dur = 0.2, at = 0, gain = 0.35, filter = 'bandpass', from = 800, to, q = 1 } = opts
  const buf = noise()
  if (!buf) return
  const t0 = now() + at
  const src = ctx.createBufferSource()
  src.buffer = buf
  src.loop = true
  const f = ctx.createBiquadFilter()
  f.type = filter
  f.Q.value = q
  f.frequency.setValueAtTime(from, t0)
  if (to !== undefined) f.frequency.exponentialRampToValueAtTime(Math.max(1, to), t0 + dur)
  const g = ctx.createGain()
  g.gain.setValueAtTime(0, t0)
  g.gain.linearRampToValueAtTime(gain, t0 + 0.008)
  g.gain.exponentialRampToValueAtTime(0.0001, t0 + dur)
  src.connect(f).connect(g).connect(master)
  src.start(t0)
  src.stop(t0 + dur + 0.05)
}

// ---------------------------------------------------------------- 音色

const sounds: Record<SfxName, () => void> = {
  // 受击：低频闷响 + 滤波噪声
  damage() {
    tone(110, { type: 'triangle', dur: 0.22, gain: 0.55, slideTo: 55 })
    noiseBurst({ dur: 0.16, gain: 0.28, from: 900, to: 200 })
  },
  // 回复：柔和上行双音
  heal() {
    tone(523, { type: 'sine', dur: 0.16, gain: 0.3 })
    tone(784, { type: 'sine', dur: 0.26, gain: 0.3, at: 0.09 })
  },
  // 威胁增长：低音小二度不和谐音
  threat() {
    tone(147, { type: 'sawtooth', dur: 0.34, gain: 0.2 })
    tone(156, { type: 'sawtooth', dur: 0.34, gain: 0.18 })
  },
  // 化解：明亮上扫
  thwart() {
    tone(440, { type: 'square', dur: 0.2, gain: 0.14, slideTo: 880 })
    noiseBurst({ dur: 0.14, gain: 0.12, from: 1600, to: 3200 })
  },
  // 攻击挥击：噪声 whoosh
  attack() {
    noiseBurst({ dur: 0.24, gain: 0.3, from: 2400, to: 300, q: 0.8 })
  },
  // 状态获得：短促拨弦
  status() {
    tone(660, { type: 'triangle', dur: 0.14, gain: 0.3, slideTo: 880 })
  },
  // 出牌：轻快弹响
  play() {
    noiseBurst({ dur: 0.07, gain: 0.2, from: 2600, to: 1400 })
    tone(880, { type: 'triangle', dur: 0.1, gain: 0.16 })
  },
  // 选中：轻 tick
  select() {
    tone(1320, { type: 'sine', dur: 0.05, gain: 0.18 })
  },
  // 胜利：大调琶音
  victory() {
    const notes = [523, 659, 784, 1047]
    notes.forEach((f, i) => tone(f, { type: 'triangle', dur: 0.5, gain: 0.32, at: i * 0.12 }))
  },
  // 败北：下行小调
  defeat() {
    const notes = [440, 349, 294, 220]
    notes.forEach((f, i) => tone(f, { type: 'sawtooth', dur: 0.55, gain: 0.2, at: i * 0.16 }))
  },
}

export function playSfx(name: SfxName) {
  if (settings.muted) return
  if (!ctx) return // 尚无手势创建的上下文：跳过（首帧前的事件）
  try {
    sounds[name]()
  } catch {
    /* 音频失败不影响游戏 */
  }
}
