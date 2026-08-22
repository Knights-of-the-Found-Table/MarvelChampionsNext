// 棋盘：固定 1920×1080 虚拟场景等比缩放适配视口（legacy scene.js 技巧），
// 场景内全部卡牌由布局引擎摆放在绝对坐标上。HUD（提示面板/日志等）由
// Game.tsx 以常规 HTML 覆盖层渲染，不参与缩放，保证文字清晰。
//
// 动效管线：view 变化经 diff 产生数值飘字与离场残影（fx.ts），引擎语义
// 事件（events prop，WS 单独帧）产生攻击突进/目标箭头/状态弹入；二者汇入
// 本组件的定时清理状态，渲染进 EffectsLayer 与各卡牌的 fx class。
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import type { GameView, Question } from '../api'
import { CARD_H, layoutBoard, SCENE_H, SCENE_W, type PlacedCard } from '../board/layout'
import { diffFloaters, eventsToFx, type Arrow, type CardFx, type Floater, type GameEvt } from '../board/fx'
import { playSfx } from '../audio/sfx'
import { useT } from '../i18n'
import EffectsLayer from './EffectsLayer'
import GameCard from './GameCard'

const FLOAT_MS = 1900
const ARROW_MS = 1200
const CARD_FX_MS = 1500
const GHOST_MS = 600

// 选项 kind → 高亮样式（legacy card-select.css 的四色体系）。
function hlClass(kind: string): string | null {
  switch (kind) {
    case 'play':
      return 'hl-play'
    case 'target':
      return 'hl-target'
    case 'ability':
    case 'basic_power':
      return 'hl-ability'
    case 'resource':
      return 'hl-pay'
    default:
      return null
  }
}

export default function Board({
  view,
  events,
  question,
  selected,
  onCardClick,
  onEndTurn,
  onRecover,
}: {
  view: GameView
  events?: GameEvt[]
  question?: Question | null
  selected?: Set<string>
  onCardClick?: (card: PlacedCard) => void
  onEndTurn?: () => void
  onRecover?: () => void
}) {
  // 高亮 + 暗幕只在"必须立即操作"的问题上启用：回合主菜单（含 end_turn
  // 选项）里的出牌/技能随时可做，不高亮不压暗；选目标、支付、防御这类
  // 强制交互才进入高亮模式。点卡作答本身不受影响（始终接线）。
  const viewportRef = useRef<HTMLDivElement | null>(null)
  const [fit, setFit] = useState({ scale: 1, left: 0, top: 0 })

  const cards = layoutBoard(view)
  const t = useT()

  // 场景快捷按钮：恢复（英雄牌左侧）、结束回合（弃牌堆右侧）。
  // 位置从布局结果锚定，跟随身份行缩放。
  const viewerPlayer = view.players[view.players.findIndex((p) => p.hand && p.hand.length > 0) >= 0 ? view.players.findIndex((p) => p.hand && p.hand.length > 0) : 0]
  const heroCard = cards.find((c) => c.id === viewerPlayer?.id)
  const discardPile = cards.find((c) => c.id === `pile-discard-${viewerPlayer?.id ?? ''}`)
  const btnScale = heroCard?.scale ?? 1
  const endTurnBtn =
    onEndTurn && discardPile
      ? { x: discardPile.x + 140 * btnScale + 18, y: discardPile.y + CARD_H * btnScale - 50 }
      : null
  const recoverBtn =
    onRecover && heroCard
      ? { x: heroCard.x - 160 * btnScale - 14, y: heroCard.y + CARD_H * btnScale - 50 }
      : null
  const posRef = useRef(new Map(cards.map((c) => [c.id, c])))
  const prevViewRef = useRef<GameView | null>(null)
  const prevLayoutRef = useRef<PlacedCard[]>(cards)

  const [floaters, setFloaters] = useState<Floater[]>([])
  const [arrows, setArrows] = useState<Arrow[]>([])
  const [cardFx, setCardFx] = useState<Map<string, CardFx>>(new Map())
  const [ghosts, setGhosts] = useState<PlacedCard[]>([])
  const timersRef = useRef<number[]>([])

  useEffect(() => {
    return () => {
      for (const t of timersRef.current) clearTimeout(t)
    }
  }, [])

  const later = (fn: () => void, ms: number) => {
    const t = window.setTimeout(() => {
      timersRef.current = timersRef.current.filter((x) => x !== t)
      fn()
    }, ms)
    timersRef.current.push(t)
  }

  useLayoutEffect(() => {
    const el = viewportRef.current
    if (!el) return
    const update = () => {
      const w = el.clientWidth
      const h = el.clientHeight
      // 顶部为 hud-top 预留安全区，避免小视口下场景与顶栏重叠
      const safeTop = 54
      const availH = Math.max(200, h - safeTop)
      const scale = Math.min(w / SCENE_W, availH / SCENE_H)
      setFit({ scale, left: (w - SCENE_W * scale) / 2, top: safeTop + (availH - SCENE_H * scale) / 2 })
    }
    update()
    const ro = new ResizeObserver(update)
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  // view diff → 飘字 + 离场残影 + 新卡入场音
  useEffect(() => {
    const prev = prevViewRef.current
    const pos = posRef.current
    if (prev) {
      const fl = diffFloaters(prev, view, pos)
      if (fl.length > 0) {
        setFloaters((cur) => [...cur, ...fl])
        for (const f of fl) later(() => setFloaters((cur) => cur.filter((x) => x.key !== f.key)), FLOAT_MS)
      }
      // 场上（非牌堆）消失的卡 → 残影淡出；新入场的卡 → 出牌音
      const nextIds = new Set(cards.map((c) => c.id))
      const gone = prevLayoutRef.current.filter((c) => c.kind !== 'pile' && !nextIds.has(c.id))
      if (gone.length > 0) {
        setGhosts((cur) => [...cur, ...gone])
        for (const g of gone) later(() => setGhosts((cur) => cur.filter((x) => x.id !== g.id)), GHOST_MS)
      }
      const prevIds = new Set(prevLayoutRef.current.map((c) => c.id))
      const arrived = cards.filter((c) => c.kind !== 'pile' && !prevIds.has(c.id))
      if (arrived.length > 0) playSfx('play')
    }
    prevViewRef.current = view
    prevLayoutRef.current = cards
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [view])

  // 语义事件 → 突进/抖动/箭头/状态弹入 + 音效
  useEffect(() => {
    if (!events || events.length === 0) return
    const { fx, arrows: ar } = eventsToFx(events, posRef.current)
    for (const e of events) {
      if (e.type === 'damage') {
        if (e.src && e.src !== e.dst) playSfx('attack')
        playSfx('damage')
      } else if (e.type === 'heal') {
        playSfx('heal')
      } else if (e.type === 'threat') {
        playSfx('threat')
      } else if (e.type === 'thwart') {
        playSfx('thwart')
      } else if (e.type === 'status') {
        playSfx('status')
      }
    }
    if (ar.length > 0) {
      setArrows((cur) => [...cur, ...ar])
      for (const a of ar) later(() => setArrows((cur) => cur.filter((x) => x.key !== a.key)), ARROW_MS)
    }
    if (fx.size > 0) {
      setCardFx((cur) => {
        const m = new Map(cur)
        for (const [id, f] of fx) m.set(id, f)
        return m
      })
      for (const id of fx.keys()) later(() => setCardFx((cur) => {
        const m = new Map(cur)
        m.delete(id)
        return m
      }), CARD_FX_MS)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [events])

  // 当前问题的高亮映射：sourceId → {发光类, 选项id}；已选中的另计序号。
  // 回合主菜单（含 end_turn）不高亮——那些操作随时可做，无需视觉聚焦。
  const highlight = useMemo(() => {
    const m = new Map<string, { cls: string; choiceId: string }>()
    if (!question) return m
    if (question.choices.some((c) => c.kind === 'end_turn')) return m
    for (const c of question.choices) {
      if (!c.sourceId || c.disabled) continue
      const cls = hlClass(c.kind)
      if (!cls) continue
      if (!m.has(c.sourceId)) m.set(c.sourceId, { cls, choiceId: c.id })
    }
    return m
  }, [question])

  const selOrder = useMemo(() => {
    const m = new Map<string, number>()
    let i = 1
    for (const id of selected ?? []) m.set(id, i++)
    return m
  }, [selected])

  const hlFor = (card: PlacedCard): string => {
    const h = highlight.get(card.id)
    if (!h) return ''
    const sel = selOrder.get(h.choiceId)
    return sel !== undefined ? `hl-selected` : h.cls
  }

  return (
    <div ref={viewportRef} className="board-viewport">
      <div
        className="board-camera"
        style={{ transform: `scale(${fit.scale})`, left: fit.left, top: fit.top }}
      >
        <div className="board-scene">
          {cards.map((c) => (
            <GameCard
              key={c.id}
              card={c}
              onClick={onCardClick}
              fx={cardFx.get(c.id)}
              className={hlFor(c)}
              selOrder={selOrder.get(highlight.get(c.id)?.choiceId ?? '')}
            />
          ))}
          {ghosts.map((c) => (
            <GameCard key={`ghost-${c.id}`} card={c} className="fx-exit" zoom={false} />
          ))}
          {/* 存在高亮目标时，全场压一层暗幕突出可选项 */}
          {highlight.size > 0 && <div className="scene-dim" />}
          {/* 场景快捷按钮（在暗幕之上） */}
          {recoverBtn && (
            <button className="board-action-btn recover" style={{ left: recoverBtn.x, top: recoverBtn.y }} onClick={onRecover}>
              <span className="abi-icon">⟳</span>
              <span className="abi-text">{t('q.recover')}</span>
            </button>
          )}
          {endTurnBtn && (
            <button className="board-action-btn endturn" style={{ left: endTurnBtn.x, top: endTurnBtn.y }} onClick={onEndTurn}>
              <span className="abi-icon">⏭</span>
              <span className="abi-text">{t('q.endTurn')}</span>
            </button>
          )}
          <EffectsLayer floaters={floaters} arrows={arrows} />
        </div>
      </div>
    </div>
  )
}
