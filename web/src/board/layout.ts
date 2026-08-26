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
  // 槽位尺寸（默认 127×176 竖版）。阴谋卡图源本身是横版，不旋转、
  // 直接用横版槽位摆放。
  w?: number
  h?: number
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
  mainScheme?: boolean
  active?: boolean
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
    id: v.id, code: v.code, kind: 'villain', x, y, w: 164, h: 164, playerIndex: -1,
    title: v.name, hp: v.hp, maxHp: v.maxHp, attack: v.attack, scheme: v.scheme,
    stunned: v.stunned, confused: v.confused, tough: v.tough,
    boosts: v.boosts, stageLabel: v.stageLabel, z: 2,
  }
}

// 阴谋卡槽位：横版（图源即横置，无需旋转）。
export const SCHEME_W = 228
export const SCHEME_H = 108

function schemeCard(s: SchemeView, x: number, y: number, main: boolean): PlacedCard {
  const roman = ['I', 'II', 'III', 'IV', 'V', 'VI', 'VII', 'VIII']
  return {
    id: s.id, code: s.code, kind: 'scheme', x, y, w: main ? 320 : SCHEME_W, h: SCHEME_H,
    playerIndex: -1,
    title: s.name, threat: s.threat, maxThreat: s.maxThreat,
    acceleration: s.acceleration, crisis: s.crisis, hazard: s.hazard,
    stageLabel: main && s.stage ? roman[s.stage - 1] : undefined,
    mainScheme: main,
    z: main ? 4 : 2,
  }
}

function minionCard(m: MinionView, x: number, y: number, playerIndex: number): PlacedCard {
  return {
    // 面向下无人机（奥创）：显示牌主玩家牌背，不带卡图。
    id: m.id, code: m.faceDown ? '' : m.code, kind: 'minion', x, y, playerIndex,
    faceDown: m.faceDown,
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
  // 手牌扇形：所有牌绕下方同一圆心排布（真实持牌的轮辐模型）——牌中心
  // 落在半径 R 的圆周上、旋转角等于辐条角，顶边自然共圆，边缘牌下垂。
  // R 是形态主控参数：半径越大弧越平缓。牌距沿圆弧量取（轻微重叠），
  // 牌多时收窄弧角上限以防过宽。
  const R = 1100
  const arcStep = CARD_W - 8
  const maxHalf = (28 * Math.PI) / 180
  const dTheta = n > 1 ? Math.min(arcStep / R, maxHalf / (n - 1)) : 0
  const midTop = SCENE_H - CARD_H - 8 // 中间牌（θ=0）顶边
  const cy = midTop + CARD_H / 2 + R // 圆心 y（手下方远处）
  const cx = SCENE_W / 2
  return owner.hand.map((c, i) => {
    const theta = (i - (n - 1) / 2) * dTheta
    return {
      id: c.id, code: c.code, kind: 'hand' as CardKind,
      x: cx + R * Math.sin(theta) - CARD_W / 2,
      y: cy - R * Math.cos(theta) - CARD_H / 2,
      rotate: (theta * 180) / Math.PI,
      z: 20 + i, playerIndex, title: c.name,
    }
  })
}

// ---------------------------------------------------------------- 主布局

export function layoutBoard(view: GameView): PlacedCard[] {
  const cards: PlacedCard[] = []
  const players = view.players ?? []

  // 敌方指挥区：反派肖像与主线密谋构成视觉中轴，牌库留在左翼。
  const villains = view.villains ?? []
  const envs = view.environments ?? []
  const villainGap = 28
  const villainVisualW = 164
  const villainGroupW = villains.length * villainVisualW + Math.max(0, villains.length - 1) * villainGap
  const commandGap = 42
  const mainSchemeW = 320
  const commandW = villainGroupW + (villains.length > 0 && view.mainScheme ? commandGap : 0) + (view.mainScheme ? mainSchemeW : 0)
  let commandX = SCENE_W / 2 - commandW / 2

  cards.push({
    id: 'pile-encounter', code: '', kind: 'pile', x: 236, y: 82, playerIndex: -1,
    faceDown: true, title: 'Encounter', z: 1, count: view.encounterCount, label: 'encounter',
  })
  // 遭遇弃牌堆：与玩家弃牌堆同款堆叠 + 顶牌朝上，点击查看详情。
  cards.push({
    id: 'pile-encounter-discard', code: view.encounterDiscardTop?.code ?? '', kind: 'pile',
    x: 236 + CARD_W + 14, y: 82, playerIndex: -1, faceDown: !view.encounterDiscardTop,
    title: 'Encounter discard', z: 1, count: view.encounterDiscardCount ?? 0, label: 'encounter-discard',
  })

  villains.forEach((v, i) => cards.push(villainCard(v, commandX + i * (villainVisualW + villainGap), 62)))
  commandX += villainGroupW + (villains.length > 0 && view.mainScheme ? commandGap : 0)
  if (view.mainScheme) cards.push(schemeCard(view.mainScheme, commandX, 88, true))

  // 爪牙在敌方阵线横排，按交战玩家分组。
  const yMinions = 276
  const minionItems: Array<{ m: MinionView; group: number }> = []
  players.forEach((p, pi) => {
    for (const m of view.minions ?? []) if (m.engagedWith === p.id) minionItems.push({ m, group: pi })
  })
  for (const m of view.minions ?? []) if (!m.engagedWith) minionItems.push({ m, group: -1 })
  if (minionItems.length > 0) {
    const xs = layoutRow(minionItems.map(() => CARD_W), (i) => minionItems[i].group, 26, 62)
    minionItems.forEach(({ m, group }, i) => cards.push(minionCard(m, xs[i], yMinions, group)))
  }

  // 共享区：环境与支线密谋位于牌桌中线，不再挤在反派指挥区。
  const sides = view.sideSchemes ?? []
  const sharedWidths = [...envs.map(() => CARD_W), ...sides.map(() => SCHEME_W)]
  const sharedGap = 28
  const sharedTotal = sharedWidths.reduce((sum, width) => sum + width, 0) + Math.max(0, sharedWidths.length - 1) * sharedGap
  let sharedX = SCENE_W / 2 - sharedTotal / 2
  envs.forEach((e) => {
    cards.push(entityCard('environment', e, sharedX, 428, -1))
    sharedX += CARD_W + sharedGap
  })
  sides.forEach((s) => {
    cards.push(schemeCard(s, sharedX, 452, false))
    sharedX += SCHEME_W + sharedGap
  })

  // ---------------------------------------------------------------- 玩家区
  // 每名玩家一个「带」：场地行（盟友-升级-支援）→ 身份行（身份牌-牌库-
  // 弃牌堆）；查看者的带在最下方并附带手牌扇形。其他玩家压缩为单行：
  // 2 人局居中一行，3-4 人局左右两侧竖排，保证任意人数都能容纳。
  const handCards = layoutHand(view)
  const viewerIdx = players.findIndex((p) => p.hand && p.hand.length > 0)
  const viewer = viewerIdx >= 0 ? viewerIdx : 0
  const waitingIdx = view.waitingFor ? players.findIndex((p) => p.name === view.waitingFor) : -1
  const activePlayer = waitingIdx >= 0 ? waitingIdx : view.question ? viewer : -1
  const N = players.length

  // 横置卡占位：90° 旋转后视觉宽度是 CARD_H，槽位随之加宽并在槽内居中，
  // 避免与左右卡重叠。
  const slotW = (exhausted: boolean | undefined, scale: number): number =>
    (exhausted ? CARD_H : CARD_W) * scale
  const slotDx = (exhausted: boolean | undefined, scale: number): number =>
    exhausted ? ((CARD_H - CARD_W) / 2) * scale : 0

  // 场地行：一名玩家的盟友 + 升级 + 支援，以 cx 居中（横置卡占宽位）。
  // 已附属到非玩家宿主（06031 监视 → 主线密谋）的升级不在此行，
  // 由文件末尾的附属摆放逻辑叠放到宿主旁。
  function fieldRow(p: PlayerView, pi: number, y: number, scale: number, cx: number): void {
    const gap = 18 * scale
    const items: PlacedCard[] = []
    // 场地行内盟友/升级/支援同尺寸、同基线对齐
    for (const a of p.allies ?? []) items.push({ ...allyCard(a, 0, y, pi), scale })
    for (const u of p.upgrades ?? []) {
      if (u.attachTo) continue
      items.push({ ...entityCard('upgrade', u, 0, y, pi), scale })
    }
    for (const s of p.supports ?? []) items.push({ ...entityCard('support', s, 0, y, pi), scale })
    if (items.length === 0) return
    const widths = items.map((it) => slotW(it.exhausted, it.scale ?? 1))
    const total = widths.reduce((a, b) => a + b, 0) + (items.length - 1) * gap
    let x = cx - total / 2
    items.forEach((it, i) => {
      cards.push({ ...it, x: x + slotDx(it.exhausted, it.scale ?? 1) })
      x += widths[i] + gap
    })
  }

  // 身份行：身份牌 + 牌库 + 弃牌堆（+ 遭遇背面堆），以 cx 居中
  function identityRow(p: PlayerView, pi: number, y: number, scale: number, cx: number): void {
    const w = CARD_W * scale
    const gap = 22 * scale
    const heroW = 164 * scale
    const enc = p.encounterDown > 0
      ? [{ id: `pile-enc-${p.id}`, w: w * 0.8 } as { id: string; w: number }]
      : []
    const total = heroW + gap + w + gap + w + enc.length * (w * 0.8 + gap)
    let x = cx - total / 2
    cards.push({ ...heroCard(p, x, y, pi, pi === activePlayer), scale })
    x += heroW + gap
    cards.push({
      id: `pile-deck-${p.id}`, code: '', kind: 'pile', x, y, playerIndex: pi,
      faceDown: true, title: p.name, z: 1, count: p.deckCount, pileScale: scale, label: 'deck',
    })
    x += w + gap
    cards.push({
      id: `pile-discard-${p.id}`, code: p.discardTop?.code ?? '', kind: 'pile', x, y,
      playerIndex: pi, faceDown: !p.discardTop, title: p.name, z: 1,
      count: p.discardCount ?? 0, pileScale: scale, label: 'discard',
    })
    x += w + gap
    if (enc.length > 0) {
      cards.push({
        id: `pile-enc-${p.id}`, code: '', kind: 'pile', x, y, playerIndex: pi,
        faceDown: true, title: '', z: 2, count: p.encounterDown, pileScale: scale * 0.8, label: 'enc',
      })
    }
  }

  // 紧凑单行带：身份 + 牌库/弃牌堆 + 场地卡混排（其他玩家）
  function compactBand(p: PlayerView, pi: number, y: number, scale: number, cx: number): void {
    const w = CARD_W * scale
    const gap = 18
    const heroW = 164 * scale
    const fieldItems: Array<{ c: PlacedCard; w: number }> = []
    for (const a of p.allies ?? []) {
      const c = { ...allyCard(a, 0, y, pi), scale }
      fieldItems.push({ c, w: slotW(c.exhausted, scale) })
    }
    for (const u of p.upgrades ?? []) {
      if (u.attachTo) continue
      const c = { ...entityCard('upgrade', u, 0, y, pi), scale }
      fieldItems.push({ c, w: slotW(c.exhausted, scale) })
    }
    for (const s of p.supports ?? []) {
      const c = { ...entityCard('support', s, 0, y, pi), scale }
      fieldItems.push({ c, w: slotW(c.exhausted, scale) })
    }
    const total =
      heroW + gap + w + gap + w + gap +
      (p.encounterDown > 0 ? w * 0.8 + gap : 0) +
      fieldItems.reduce((a, f) => a + f.w + 14, 0)
    let x = cx - total / 2
    cards.push({ ...heroCard(p, x, y, pi, pi === activePlayer), scale })
    x += heroW + gap
    const pileY = y + (CARD_H - CARD_H * scale) / 2
    cards.push({
      id: `pile-deck-${p.id}`, code: '', kind: 'pile', x, y: pileY, playerIndex: pi,
      faceDown: true, title: p.name, z: 1, count: p.deckCount, pileScale: scale, label: 'deck',
    })
    x += w + gap
    cards.push({
      id: `pile-discard-${p.id}`, code: p.discardTop?.code ?? '', kind: 'pile', x, y: pileY,
      playerIndex: pi, faceDown: !p.discardTop, title: p.name, z: 1, count: p.discardCount ?? 0, pileScale: scale, label: 'discard',
    })
    x += w + gap
    if (p.encounterDown > 0) {
      cards.push({
        id: `pile-enc-${p.id}`, code: '', kind: 'pile', x, y: pileY, playerIndex: pi,
        faceDown: true, title: '', z: 2, count: p.encounterDown, pileScale: scale * 0.8, label: 'enc',
      })
      x += w * 0.8 + gap
    }
    for (const f of fieldItems) {
      cards.push({ ...f.c, x: x + slotDx(f.c.exhausted, f.c.scale ?? 1) })
      x += f.w + 14
    }
  }

  // 查看者带：英雄与牌堆固定在左下指挥位，场上资产横向铺在其右侧。
  const vScale = N >= 3 ? 0.82 : 0.94
  fieldRow(players[viewer], viewer, 610, vScale, 1120)
  identityRow(players[viewer], viewer, 720, vScale, 420)

  // 其他玩家压缩到共享区两翼，保留多人局识别但不争夺主视线。
  const others = players.map((p, i) => ({ p, i })).filter(({ i }) => i !== viewer)
  if (others.length === 1) {
    compactBand(others[0].p, others[0].i, 432, 0.68, 360)
  } else if (others.length > 1) {
    const scale = 0.5
    const h = CARD_H * scale
    others.forEach((o, k) => {
      const left = k % 2 === 0
      const cx = left ? 300 : SCENE_W - 300
      const y = 416 + Math.floor(k / 2) * (h + 18)
      compactBand(o.p, o.i, y, scale, cx)
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

  // 附属到非玩家宿主的玩家升级（06031 监视 → 主线密谋）：小尺寸叠放在
  // 宿主卡旁；找不到宿主则放到右上角无宿主区。
  let homelessUp = 0
  for (const p of players) {
    const ownerIdx = players.indexOf(p)
    for (const u of p.upgrades ?? []) {
      if (!u.attachTo) continue
      const card = { ...entityCard('upgrade', u, 0, 0, ownerIdx) }
      const host = byId.get(u.attachTo)
      if (host) {
        const anchorX = host.x + (host.w ?? CARD_W) * 0.55
        const stack = cards.filter((c) => c.kind === 'upgrade' && Math.abs(c.x - anchorX) < 8 && c.y > host.y - 30 && c.y < host.y + (host.h ?? CARD_H)).length
        card.x = anchorX
        card.y = host.y + 8 + stack * 22
        card.z = (host.z ?? 2) + 1 + stack
        card.scale = 0.6
      } else {
        card.x = 1580
        card.y = 60 + homelessUp * 120
        card.z = 2
        card.scale = 0.8
        homelessUp++
      }
      cards.push(card)
    }
  }

  // 手牌扇形（仅查看者本人）
  cards.push(...handCards)

  return cards
}

function heroCard(p: PlayerView, x: number, y: number, playerIndex: number, active = false): PlacedCard {
  const isHero = p.side === 'hero'
  return {
    id: p.id,
    code: isHero ? p.heroCode : p.alterEgoCode,
    kind: 'hero', x, y, w: 164, h: 164, playerIndex,
    // 悬浮提示用当前面的卡牌名（与其他卡一致），不用玩家名
    title: (isHero ? p.heroName : p.alterEgoName) || p.name,
    hp: p.hp, maxHp: p.maxHp,
    exhausted: p.exhausted, stunned: p.stunned, confused: p.confused, tough: p.tough,
    firstPlayer: p.firstPlayer, koed: p.koed, active, z: 3,
  }
}
