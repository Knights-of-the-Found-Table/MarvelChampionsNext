// 棋盘布局引擎：把 GameView 投影为 1920×1080 虚拟场景上的绝对坐标卡牌列表。
// 布局是 view 的纯函数：坐标写入 CSS 变量（--x/--y/--rot），卡牌 DOM 节点按
// 实体 id 保持稳定，因此位置变化由 CSS transition 自动补间出移动动画。
// 布局参照 legacy 漫威网页版的共享行设计：任意玩家数（1-4）都使用同一组
// 横向行（爪牙/盟友/支援/英雄），按玩家分组着色，右侧竖排放阴谋。

import type {
  AllyView,
  AttachmentView,
  EntityLite,
  GameView,
  MinionView,
  PlayerView,
  SchemeView,
  VillainView,
} from '../api'

export const SCENE_W = 1920
export const SCENE_H = 1080
export const CARD_W = 127
export const CARD_H = 176

export type CardKind =
  | 'villain'
  | 'environment'
  | 'minion'
  | 'ally'
  | 'support'
  | 'hero'
  | 'upgrade'
  | 'attachment'
  | 'treachery'
  | 'scheme'
  | 'hand'
  | 'pile'

// 摆放结果：一张卡（或一堆牌）在场景上的位置与展示数据。
export interface PlacedCard {
  id: string
  code: string
  kind: CardKind
  x: number
  y: number
  rotate?: number
  z?: number
  scale?: number
  faceDown?: boolean
  // 控制者（决定描边配色），-1 = 遭遇方
  playerIndex: number
  title: string
  hp?: number
  maxHp?: number
  threat?: number
  maxThreat?: number
  attack?: number
  scheme?: number
  thwart?: number
  stunned?: boolean
  confused?: boolean
  tough?: boolean
  guard?: boolean
  counters?: number
  boosts?: number
  stageLabel?: string
  firstPlayer?: boolean
  exhausted?: boolean
  koed?: boolean
  acceleration?: number
  crisis?: boolean
  hazard?: number
  count?: number // 牌堆数量
  label?: string // 牌堆标签
  pileScale?: number
}

const ROW_MIN_X = 310
const ROW_MAX_X = 1640
const ROW_CENTER = (ROW_MIN_X + ROW_MAX_X) / 2
const ROW_AVAIL = ROW_MAX_X - ROW_MIN_X

// 一行内多个玩家组的水平排布。返回每项的 x。放不下时先压组间距、
// 再压组内间距、最后允许重叠（保持首尾完整可见）。
function layoutRow(unitWidths: number[], groupOf: (i: number) => number, innerGap: number, groupGap: number): number[] {
  const n = unitWidths.length
  if (n === 0) return []
  const distinctNeighbors = Array.from({ length: n - 1 }, (_, i) => (groupOf(i) !== groupOf(i + 1) ? groupGap : innerGap))
  const width = () => unitWidths.reduce((a, b) => a + b, 0) + distinctNeighbors.reduce((a, b) => a + b, 0)
  let gaps = [...distinctNeighbors]
  if (width() > ROW_AVAIL) {
    // 压组间距
    gaps = gaps.map((g) => (g === groupGap ? Math.max(24, groupGap / 2) : g))
  }
  if (width() > ROW_AVAIL) {
    // 压所有间距
    const over = width() - ROW_AVAIL
    gaps = gaps.map((g) => Math.max(-CARD_W * 0.75, g - over / gaps.length))
  }
  let x = ROW_CENTER - width() / 2
  return unitWidths.map((w, i) => {
    const start = x
    x += w + (gaps[i] ?? 0)
    return start
  })
}

function villainCard(v: VillainView, x: number, y: number): PlacedCard {
  return {
    id: v.id, code: v.code, kind: 'villain', x, y, playerIndex: -1,
    title: v.name, hp: v.hp, maxHp: v.maxHp, attack: v.attack, scheme: v.scheme,
    stunned: v.stunned, confused: v.confused, tough: v.tough,
    boosts: v.boosts, stageLabel: v.stageLabel, z: 2,
  }
}

function schemeCard(s: SchemeView, x: number, y: number, main: boolean): PlacedCard {
  const roman = ['I', 'II', 'III', 'IV', 'V', 'VI', 'VII', 'VIII']
  return {
    id: s.id, code: s.code, kind: 'scheme', x, y, rotate: 90, playerIndex: -1,
    title: s.name, threat: s.threat, maxThreat: s.maxThreat,
    acceleration: s.acceleration, crisis: s.crisis, hazard: s.hazard,
    stageLabel: main && s.stage ? roman[s.stage - 1] : undefined,
    z: 2,
  }
}

function minionCard(m: MinionView, x: number, y: number, playerIndex: number): PlacedCard {
  return {
    id: m.id, code: m.code, kind: 'minion', x, y, playerIndex,
    title: m.name, hp: m.hp, maxHp: m.maxHp, attack: m.attack, scheme: m.scheme,
    stunned: m.stunned, confused: m.confused, tough: m.tough, guard: m.guard, z: 2,
  }
}

function allyCard(a: AllyView, x: number, y: number, playerIndex: number): PlacedCard {
  return {
    id: a.id, code: a.code, kind: 'ally', x, y, playerIndex,
    title: a.name, hp: a.hp, maxHp: a.maxHp, attack: a.attack, thwart: a.thwart,
    exhausted: a.exhausted, stunned: a.stunned, confused: a.confused, tough: a.tough,
    counters: a.counters, z: 2,
  }
}

function entityCard(kind: 'support' | 'upgrade' | 'environment', e: EntityLite, x: number, y: number, playerIndex: number, scale?: number): PlacedCard {
  return {
    id: e.id, code: e.code, kind, x, y, playerIndex, title: e.name,
    exhausted: e.exhausted, counters: e.counters, z: 2, scale,
  }
}

// ---------------------------------------------------------------- 手牌扇形

export function layoutHand(view: GameView): PlacedCard[] {
  const owner = view.players.find((p) => p.hand && p.hand.length > 0)
  if (!owner || !owner.hand) return []
  const n = owner.hand.length
  const playerIndex = view.players.indexOf(owner)
  const maxW = 920
  const step = n > 1 ? Math.min(CARD_W + 14, (maxW - CARD_W) / (n - 1)) : 0
  const startX = SCENE_W / 2 - (CARD_W + (n - 1) * step) / 2
  const totalDeg = Math.min(36, n * 5)
  const perDeg = n > 1 ? totalDeg / (n - 1) : 0
  return owner.hand.map((c, i) => {
    const t = n > 1 ? i / (n - 1) - 0.5 : 0 // -0.5..0.5
    return {
      id: c.id, code: c.code, kind: 'hand' as CardKind,
      x: startX + i * step,
      y: SCENE_H - CARD_H + 44 + t * t * 60,
      rotate: (i - (n - 1) / 2) * perDeg,
      z: 20 + i, playerIndex, title: c.name,
    }
  })
}

// ---------------------------------------------------------------- 主布局

export function layoutBoard(view: GameView): PlacedCard[] {
  const cards: PlacedCard[] = []
  const players = view.players ?? []

  // 顶部：遭遇牌库（背面堆）+ 环境区
  cards.push({
    id: 'pile-encounter', code: '', kind: 'pile', x: 36, y: 14, playerIndex: -1,
    faceDown: true, title: 'Encounter', z: 1, count: view.encounterCount, label: 'encounter',
  })
  const envs = view.environments ?? []
  envs.forEach((e, i) => {
    cards.push(entityCard('environment', e, 200 + i * (CARD_W + 16), 14, -1))
  })

  // 顶部中央：反派行
  const villains = view.villains ?? []
  const villainGap = 32
  const vw = villains.length * CARD_W + Math.max(0, villains.length - 1) * villainGap
  const vx0 = 1020 - vw / 2
  villains.forEach((v, i) => cards.push(villainCard(v, vx0 + i * (CARD_W + villainGap), 14)))

  // 右侧：主阴谋（旋转 90°）+ 支线阴谋竖排
  if (view.mainScheme) cards.push(schemeCard(view.mainScheme, 1706, 40, true))
  const sides = view.sideSchemes ?? []
  sides.forEach((s, i) => cards.push(schemeCard(s, 1724, 208 + i * 138, false)))

  // 共享行 y 坐标：爪牙 / 盟友 / 支援 / 英雄
  const yMinions = 224
  const yAllies = 428
  const ySupports = 620
  const yHeroes = 812

  // 爪牙行：按被交战玩家分组；未交战爪牙作为收尾组
  const minionItems: Array<{ m: MinionView; group: number }> = []
  players.forEach((p, pi) => {
    for (const m of view.minions ?? []) if (m.engagedWith === p.id) minionItems.push({ m, group: pi })
  })
  for (const m of view.minions ?? []) if (!m.engagedWith) minionItems.push({ m, group: -1 })
  if (minionItems.length > 0) {
    const xs = layoutRow(minionItems.map(() => CARD_W), (i) => minionItems[i].group, 24, 56)
    minionItems.forEach(({ m, group }, i) => cards.push(minionCard(m, xs[i], yMinions, group)))
  }

  // 盟友行
  const allyItems: Array<{ a: AllyView; group: number }> = []
  players.forEach((p, pi) => {
    for (const a of p.allies ?? []) allyItems.push({ a, group: pi })
  })
  if (allyItems.length > 0) {
    const xs = layoutRow(allyItems.map(() => CARD_W), (i) => allyItems[i].group, 20, 56)
    allyItems.forEach(({ a, group }, i) => cards.push(allyCard(a, xs[i], yAllies, group)))
  }

  // 支援行
  const supportItems: Array<{ s: EntityLite; group: number }> = []
  players.forEach((p, pi) => {
    for (const s of p.supports ?? []) supportItems.push({ s, group: pi })
  })
  if (supportItems.length > 0) {
    const xs = layoutRow(supportItems.map(() => CARD_W), (i) => supportItems[i].group, 20, 56)
    supportItems.forEach(({ s, group }, i) => cards.push(entityCard('support', s, xs[i], ySupports, group)))
  }

  // 英雄行：每名玩家一个单元（身份卡 + 半宽叠放的升级 + 遭遇背面堆）
  const heroUnits = players.map((p, pi) => {
    const upgrades = p.upgrades ?? []
    const upW = CARD_W * 0.82 + Math.max(0, upgrades.length - 1) * CARD_W * 0.42
    const encW = p.encounterDown > 0 ? 70 : 0
    return { p, pi, upgrades, width: CARD_W + (upgrades.length > 0 ? upW + 8 : 0) + encW }
  })
  if (heroUnits.length > 0) {
    const xs = layoutRow(heroUnits.map((u) => u.width), (i) => i, 0, 56)
    heroUnits.forEach((u, i) => {
      let x = xs[i]
      cards.push(heroCard(u.p, x, yHeroes, u.pi))
      x += CARD_W + 6
      u.upgrades.forEach((up, j) => {
        cards.push({ ...entityCard('upgrade', up, x, yHeroes + 26, u.pi, 0.82), z: 2 + j })
        x += j === u.upgrades.length - 1 ? CARD_W * 0.82 : CARD_W * 0.42
      })
      if (u.upgrades.length > 0) x += 8
      if (u.p.encounterDown > 0) {
        cards.push({
          id: `pile-enc-${u.p.id}`, code: '', kind: 'pile', x, y: yHeroes + 26,
          playerIndex: u.pi, faceDown: true, title: '', z: 2,
          count: u.p.encounterDown, pileScale: 0.82, label: 'enc',
        })
      }
    })
  }

  // 附件/持久诡计：叠放在宿主右侧下方（小尺寸）
  const byId = new Map(cards.map((c) => [c.id, c]))
  const placeAttached = (list: AttachmentView[] | null | undefined, kind: 'attachment' | 'treachery') => {
    let homeless = 0
    for (const a of list ?? []) {
      const host = a.host ? byId.get(a.host) : undefined
      if (host) {
        const attachCount = cards.filter((c) => c.kind === kind && Math.abs(c.x - (host.x + CARD_W * 0.6)) < 6 && c.y > host.y - 40 && c.y < host.y + CARD_H).length
        cards.push({
          id: a.id, code: a.code, kind, x: host.x + CARD_W * 0.6, y: host.y + 10 + attachCount * 20,
          playerIndex: -1, title: a.name, scale: 0.6, z: (host.z ?? 2) + 1 + attachCount,
        })
      } else {
        cards.push({
          id: a.id, code: a.code, kind, x: 1580, y: 60 + homeless * 120,
          playerIndex: -1, title: a.name, scale: 0.8, z: 2,
        })
        homeless++
      }
    }
  }
  placeAttached(view.attachments, 'attachment')
  placeAttached(view.treacheries, 'treachery')

  // 左列：玩家牌库/弃牌堆（缩小的背面堆 + 数量徽章 + 弃牌堆顶）
  players.forEach((p, i) => {
    const py = 248 + i * 148
    cards.push({
      id: `pile-deck-${p.id}`, code: '', kind: 'pile', x: 36, y: py, playerIndex: i,
      faceDown: true, title: p.name, z: 1, count: p.deckCount, pileScale: 0.74, label: 'deck',
    })
    cards.push({
      id: `pile-discard-${p.id}`, code: p.discardTop?.code ?? '', kind: 'pile', x: 140, y: py,
      playerIndex: i, faceDown: !p.discardTop, title: p.name, z: 1,
      count: 0, pileScale: 0.74, label: 'discard',
    })
  })

  // 手牌扇形（仅查看者本人）
  cards.push(...layoutHand(view))

  return cards
}

function heroCard(p: PlayerView, x: number, y: number, playerIndex: number): PlacedCard {
  return {
    id: p.id,
    code: p.side === 'hero' ? p.heroCode : p.alterEgo,
    kind: 'hero', x, y, playerIndex,
    title: p.name, hp: p.hp, maxHp: p.maxHp,
    exhausted: p.exhausted, stunned: p.stunned, confused: p.confused, tough: p.tough,
    firstPlayer: p.firstPlayer, koed: p.koed, z: 3,
  }
}
