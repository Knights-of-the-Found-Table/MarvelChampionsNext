// 场上单张卡牌：位置由布局引擎通过 CSS 变量（--x/--y/--rot/--z）给出，
// 外层节点只做位移（transition 自动补间移动动画），内层处理横置旋转与
// hover 缩放。标志物（状态芯片、血量/威胁徽章、计数器等）以绝对定位
// 覆盖在卡面上，数值同时写入 data-* 供 diff 动画层定位飘字。
import { useEffect, useRef, useState } from 'react'
import { cardUrl, fallbackDataUrl, useCardZoom } from '../cards'
import { lname, useLang, useT, useZhMap } from '../i18n'
import type { CardFx } from '../board/fx'
import type { PlacedCard } from '../board/layout'

const PLAYER_COLORS = ['#4a90d9', '#d94a4a', '#d9a04a', '#3fa66a']

interface Props {
  card: PlacedCard
  onClick?: (card: PlacedCard) => void
  className?: string
  zoom?: boolean
  fx?: CardFx
  // 多选时已选中的序号（1 起）
  selOrder?: number
}

export default function GameCard({ card, onClick, className = '', zoom = true, fx, selOrder }: Props) {
  const lang = useLang()
  const zh = useZhMap()
  const safeCode = card.code && card.code !== 'undefined' && card.code !== 'null' ? card.code : ''
  const displayTitle = lname(zh, safeCode, card.title)
  const imgRef = useRef<HTMLDivElement | null>(null)
  const cardZoom = useCardZoom(safeCode, imgRef)
  // 挂载时短暂附加入场动画类，之后移除，避免与后续动效在 animation
  // 属性上冲突（class 移除后 animation 恢复 none，不会重播）。
  const [entering, setEntering] = useState(true)
  useEffect(() => {
    const t = window.setTimeout(() => setEntering(false), 380)
    return () => clearTimeout(t)
  }, [])

  if (card.kind === 'pile') return <Pile card={card} className={className} onClick={onClick ? () => onClick(card) : undefined} />

  const color = card.playerIndex >= 0 ? PLAYER_COLORS[card.playerIndex % 4] : '#8a2020'
  const fxCls = fx
    ? `${fx.shake ? 'fx-shake' : ''} ${fx.statusPop ? 'fx-status' : ''} ${fx.lunge ? 'fx-lunge' : ''}`.trim()
    : ''

  return (
    <div
      ref={imgRef}
      className={`gcard pk-${Math.max(0, card.playerIndex)} k-${card.kind} ${card.mainScheme ? 'is-main-scheme' : ''} ${card.active ? 'is-active-player' : ''} ${entering ? 'fx-entering' : ''} ${card.koed ? 'koed' : ''} ${fxCls} ${className}`}
      style={
        {
          '--x': `${card.x}px`,
          '--y': `${card.y}px`,
          '--w': `${card.w ?? 127}px`,
          '--h': `${card.h ?? 176}px`,
          '--rot': `${card.rotate ?? 0}deg`,
          '--s': card.scale ?? 1,
          '--z': card.z ?? 2,
          '--pc': color,
          '--hp-progress': card.maxHp ? `${Math.max(0, Math.min(1, (card.hp ?? 0) / card.maxHp)) * 360}deg` : '0deg',
          '--threat-progress': card.maxThreat ? `${Math.max(0, Math.min(1, (card.threat ?? 0) / card.maxThreat)) * 100}%` : '0%',
          ...(fx?.lunge
            ? {
                '--lx': `${fx.lunge.dx}px`,
                '--ly': `${fx.lunge.dy}px`,
                '--lrot': `${fx.lunge.rot}deg`,
              }
            : {}),
        } as React.CSSProperties
      }
      data-card-id={card.id}
      data-card-kind={card.kind}
      data-hp={card.hp}
      data-threat={card.threat}
      data-counters={card.counters}
      data-sel={selOrder}
      title={displayTitle}
      onClick={onClick ? () => onClick(card) : undefined}
      onMouseEnter={safeCode && zoom ? cardZoom.onEnter : undefined}
      onMouseLeave={safeCode && zoom ? cardZoom.hide : undefined}
    >
      <div className={`gcard-in ${card.exhausted ? 'exhausted' : ''}`}>
        {safeCode ? (
          <img
            className="gcard-img"
            src={cardUrl(safeCode, lang)}
            alt={card.title}
            draggable={false}
            onError={(e) => {
              const img = e.currentTarget
              if (!img.dataset.fallback) {
                img.dataset.fallback = '1'
                img.src = fallbackDataUrl(safeCode)
              }
            }}
          />
        ) : (
          <div className={`gcard-back placeholder ${card.playerIndex >= 0 ? 'player' : 'encounter'}`}>
            <span aria-hidden="true">★</span>
          </div>
        )}

        {(card.kind === 'hero' || card.kind === 'villain') && (
          <div className="portrait-name">{displayTitle}</div>
        )}

        {/* 状态芯片 */}
        <div className="gcard-tokens">
          {card.stunned && <span className="tok tok-stun" title="Stunned">✳</span>}
          {card.confused && <span className="tok tok-confuse" title="Confused">?</span>}
          {card.tough && <span className="tok tok-tough" title="Tough">◆</span>}
          {card.guard && <span className="tok tok-guard" title="Guard">▲</span>}
          {card.firstPlayer && <span className="tok tok-first" title="First player">★</span>}
          {Array.from({ length: Math.min(card.acceleration ?? 0, 6) }).map((_, i) => (
            <span key={i} className="tok tok-accel" title="Acceleration">⏩</span>
          ))}
        </div>

        {/* 阶段/计数/强化徽章 */}
        <div className="gcard-topright">
          {card.stageLabel && <span className="chip chip-stage">{card.stageLabel}</span>}
          {card.counters !== undefined && card.counters > 0 && (
            <span className="chip chip-count">{card.counters}</span>
          )}
          {card.boosts !== undefined && card.boosts > 0 && (
            <span className="chip chip-boost">+{card.boosts}</span>
          )}
        </div>

        {/* 血量徽章 */}
        {card.hp !== undefined && card.maxHp ? (
          <span className={`hp-badge ${card.hp <= card.maxHp / 3 ? 'low' : ''}`}>
            <span className="badge-icon">✚</span>
            <span>{card.hp}</span>
          </span>
        ) : null}

        {/* 威胁条 */}
        {card.threat !== undefined && (
          <span className={`threat-bar ${card.maxThreat && card.threat >= card.maxThreat - 2 ? 'high' : ''}`}>
            <span className="badge-icon">◆</span>
            <strong>{card.threat}</strong>
            {card.maxThreat ? <small>/{card.maxThreat}</small> : null}
          </span>
        )}

        {/* 攻击/密谋/化解数值 */}
        {(card.attack !== undefined || card.thwart !== undefined) && (
          <div className="gcard-stats">
            {card.attack !== undefined && <span className="stat stat-atk"><span className="stat-icon">⚔</span>{card.attack}</span>}
            {card.thwart !== undefined && <span className="stat stat-thw"><span className="stat-icon">◎</span>{card.thwart}</span>}
            {card.scheme !== undefined && <span className="stat stat-sch"><span className="stat-icon">◆</span>{card.scheme}</span>}
          </div>
        )}

        {/* 危机/危害 */}
        {(card.crisis || (card.hazard ?? 0) > 0) && (
          <div className="gcard-schemetags">
            {card.crisis && <span className="tag tag-crisis">危</span>}
            {(card.hazard ?? 0) > 0 && <span className="tag tag-hazard">☠{card.hazard}</span>}
          </div>
        )}
      </div>
      {safeCode && zoom ? cardZoom.overlay : null}
    </div>
  )
}

// 牌堆：按数量堆叠的背面层（每约 3 张一层，封顶 7 层）+ 数量徽章。
// 弃牌堆有顶牌时顶牌朝上盖在层堆上；空弃牌堆显示蓝色空框 + 数字 0。
function Pile({ card, className = '', onClick }: { card: PlacedCard; className?: string; onClick?: () => void }) {
  const s = card.pileScale ?? 1
  const lang = useLang()
  const t = useT()
  const zh = useZhMap()
  const count = card.count ?? 0
  const layers = Math.max(1, Math.min(7, Math.ceil(count / 3)))
  const safeCode = card.code && card.code !== 'undefined' && card.code !== 'null' ? card.code : ''
  const isEmptyDiscard = card.label === 'discard' && count === 0 && !safeCode
  // 牌库/弃牌堆 title：{玩家名}的牌库 / {玩家名}的弃牌堆
  const displayTitle =
    card.label === 'deck'
      ? t('pile.deckTitle', { name: card.title })
      : card.label === 'discard'
        ? t('pile.discardTitle', { name: card.title })
        : card.label === 'encounter'
          ? t('pile.encounter')
          : lname(zh, card.code, card.title)
  return (
    <div
      className={`gcard pile pk-${Math.max(0, card.playerIndex)} k-pile-${card.label ?? 'deck'} ${className}`}
      style={
        {
          '--x': `${card.x}px`,
          '--y': `${card.y}px`,
          '--w': '127px',
          '--h': '176px',
          '--rot': '0deg',
          '--s': s,
          '--z': card.z ?? 1,
          '--pc': card.playerIndex >= 0 ? PLAYER_COLORS[card.playerIndex % 4] : '#8a2020',
        } as React.CSSProperties
      }
      title={displayTitle}
      onClick={onClick}
    >
      <div className="gcard-in">
        {isEmptyDiscard ? (
          <div className="gcard-back empty-discard" />
        ) : (
          <>
            {Array.from({ length: layers }).map((_, i) => (
              <div
                key={i}
                className={`gcard-back pile-layer ${card.playerIndex >= 0 ? 'player' : 'encounter'}`}
                style={{ translate: `${i * 2}px ${-i * 2.5}px` } as React.CSSProperties}
              />
            ))}
            {safeCode && !card.faceDown ? (
              <img
                className="gcard-img pile-top"
                src={cardUrl(safeCode, lang)}
                alt={card.title}
                draggable={false}
                onError={(e) => {
                  const img = e.currentTarget
                  if (!img.dataset.fallback) {
                    img.dataset.fallback = '1'
                    img.src = fallbackDataUrl(safeCode)
                  }
                }}
              />
            ) : null}
          </>
        )}
        {(count > 0 || card.label === 'discard') && <span className="pile-count">{count}</span>}
      </div>
    </div>
  )
}
