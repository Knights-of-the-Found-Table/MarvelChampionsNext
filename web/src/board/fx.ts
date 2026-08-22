// 动效管线（阶段二）：两个来源合成动效——
// 1. view diff（纯前端）：数值变化 → 飘字/抖动/入场/退场，数字以界面真实
//    变化为准；
// 2. 引擎语义事件（WS events 帧）：damage/threat/thwart/status → 攻击突进、
//    目标箭头、状态弹入（提供"谁对谁"的关系信息）。
// 所有坐标均为 1920×1080 场景坐标，由 EffectsLayer 在场景内渲染。

import type { GameView } from '../api'
import type { PlacedCard } from './layout'
import { CARD_H, CARD_W } from './layout'

export interface GameEvt {
  type: string // damage | heal | threat | thwart | status
  src?: string
  dst: string
  n?: number
  status?: string
  on?: boolean
}

export interface Floater {
  key: number
  x: number
  y: number
  text: string
  kind: 'dmg' | 'heal' | 'threat' | 'thwart' | 'counter'
  index: number
}

export interface Arrow {
  key: number
  x: number
  y: number
  angle: number
  len: number
  kind: 'attack' | 'threat' | 'thwart'
}

export interface CardFx {
  shake: boolean
  lunge?: { dx: number; dy: number; rot: number }
  statusPop: boolean
}

let fxSeq = 1

// ---------------------------------------------------------------- view diff

interface StatSnapshot {
  hp?: number
  threat?: number
  counters?: number
}

function collectStats(v: GameView): Map<string, StatSnapshot> {
  const m = new Map<string, StatSnapshot>()
  const put = (id: string, s: StatSnapshot) => {
    const prev = m.get(id) ?? {}
    m.set(id, { ...prev, ...s })
  }
  for (const e of v.villains ?? []) put(e.id, { hp: e.hp })
  for (const e of v.minions ?? []) put(e.id, { hp: e.hp })
  for (const p of v.players ?? []) {
    put(p.id, { hp: p.hp })
    for (const a of p.allies ?? []) put(a.id, { hp: a.hp, counters: a.counters ?? 0 })
    for (const s of p.supports ?? []) put(s.id, { counters: s.counters ?? 0 })
    for (const u of p.upgrades ?? []) put(u.id, { counters: u.counters ?? 0 })
  }
  if (v.mainScheme) put(v.mainScheme.id, { threat: v.mainScheme.threat })
  for (const s of v.sideSchemes ?? []) put(s.id, { threat: s.threat })
  return m
}

// diff 出数值飘字；pos 在 next 布局里找不到时不产生（实体已离场）。
export function diffFloaters(prev: GameView, next: GameView, pos: Map<string, PlacedCard>): Floater[] {
  const a = collectStats(prev)
  const b = collectStats(next)
  const out: Floater[] = []
  let index = 0
  for (const [id, sb] of b) {
    const sa = a.get(id)
    if (!sa) continue
    const p = pos.get(id)
    if (!p) continue
    const push = (delta: number, kind: Floater['kind']) => {
      out.push({
        key: fxSeq++,
        x: p.x + CARD_W / 2,
        y: p.y + 6,
        text: delta > 0 ? `+${delta}` : `${delta}`,
        kind,
        index: index++,
      })
    }
    if (sa.hp !== undefined && sb.hp !== undefined && sa.hp !== sb.hp) {
      push(sb.hp - sa.hp, sb.hp < sa.hp ? 'dmg' : 'heal')
    }
    if (sa.threat !== undefined && sb.threat !== undefined && sa.threat !== sb.threat) {
      push(sb.threat - sa.threat, sb.threat > sa.threat ? 'threat' : 'thwart')
    }
    if (sa.counters !== undefined && sb.counters !== undefined && sa.counters !== sb.counters) {
      push(sb.counters - sa.counters, 'counter')
    }
  }
  return out
}

// ---------------------------------------------------------------- 事件映射

// 语义事件 → 卡牌动效（抖动/突进/状态弹入）与箭头。
export function eventsToFx(events: GameEvt[], pos: Map<string, PlacedCard>): {
  fx: Map<string, CardFx>
  arrows: Arrow[]
} {
  const fx = new Map<string, CardFx>()
  const arrows: Arrow[] = []
  const center = (id: string) => {
    const p = pos.get(id)
    if (!p) return null
    return { x: p.x + CARD_W / 2, y: p.y + CARD_H / 2 }
  }
  const cardFx = (id: string): CardFx => {
    let f = fx.get(id)
    if (!f) {
      f = { shake: false, statusPop: false }
      fx.set(id, f)
    }
    return f
  }
  for (const e of events) {
    switch (e.type) {
      case 'damage': {
        cardFx(e.dst).shake = true
        const s = e.src ? center(e.src) : null
        const d = center(e.dst)
        if (s && d && e.src !== e.dst) {
          cardFx(e.src!).lunge = { dx: d.x - s.x, dy: d.y - s.y, rot: lungeRot(s, d) }
          arrows.push({ key: fxSeq++, x: s.x, y: s.y, angle: (Math.atan2(d.y - s.y, d.x - s.x) * 180) / Math.PI, len: Math.hypot(d.x - s.x, d.y - s.y) - CARD_W * 0.55, kind: 'attack' })
        }
        break
      }
      case 'threat':
      case 'thwart': {
        const s = e.src ? center(e.src) : null
        const d = center(e.dst)
        if (s && d && e.src !== e.dst) {
          arrows.push({ key: fxSeq++, x: s.x, y: s.y, angle: (Math.atan2(d.y - s.y, d.x - s.x) * 180) / Math.PI, len: Math.hypot(d.x - s.x, d.y - s.y) - CARD_W * 0.55, kind: e.type === 'threat' ? 'threat' : 'thwart' })
        }
        break
      }
      case 'status':
        if (e.on) cardFx(e.dst).statusPop = true
        break
    }
  }
  return { fx, arrows }
}

// 突进旋转角：攻击者飞向目标时卡面朝向目标（legacy 的 atan2 定角）。
function lungeRot(s: { x: number; y: number }, d: { x: number; y: number }): number {
  return (Math.atan2(d.y - s.y, d.x - s.x) * 180) / Math.PI / 2
}
