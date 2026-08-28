// 场上单张卡牌：位置由布局引擎通过 CSS 变量（--x/--y/--rot/--z）给出，
// 外层节点只做位移（transition 自动补间移动动画），内层处理横置旋转与
// hover 缩放。标志物（状态芯片、血量/威胁徽章、计数器等）以绝对定位
// 覆盖在卡面上，数值同时写入 data-* 供 diff 动画层定位飘字。
import { useEffect, useId, useRef, useState, type ReactNode, type PointerEvent as ReactPointerEvent } from 'react'
import { cardUrl, fallbackDataUrl, useCardZoom } from '../cards'
import { lname, useLang, useT, useZhMap } from '../i18n'
import type { CardFx } from '../board/fx'
import type { PlacedCard } from '../board/layout'

const PLAYER_COLORS = ['#4a90d9', '#d94a4a', '#d9a04a', '#3fa66a']

// 伤害代币图形（不含数字）：对局的血量徽章与牌组详情的生命值图标共用这
// 份黑边 + 炽橙渐变 + 网点定义。渐变/网点 id 必须每实例唯一（同屏多枚
// 代币共享 defs 命名空间）。
export function DamageTokenArt({ gid }: { gid: string }) {
  return (
    <>
      <circle cx="20" cy="20" r="18.6" fill="#171114" />
      <circle cx="20" cy="20" r="14.6" fill={`url(#${gid})`} />
      <circle cx="20" cy="20" r="14.6" fill={`url(#${gid}-d)`} />
      <circle cx="20" cy="20" r="14.6" fill="none" stroke="rgba(255,199,96,0.8)" strokeWidth="1.1" />
    </>
  )
}

// 实体代币风格的数值徽章（参照桌游伤害/威胁代币）：黑边框 + 炽橙底纹 +
// 白色斜体描边数字。hp 展示剩余生命值；threat 把「当前值/要求值」整体印
// 在三角形里（要求值缺席时只印当前值）——串越长字号越小、重心越贴近
// 三角形的宽底。渐变/网点图案的 id 必须每实例唯一（同屏几十个代币共享
// defs 命名空间）。
function StatToken({ kind, value, max }: { kind: 'hp' | 'threat'; value: number; max?: number }) {
  const uid = useId().replace(/[^a-zA-Z0-9]/g, '')
  const gid = `tg-${uid}`
  const label = max ? `${value}/${max}` : `${value}`
  // 两位数缩小字号，避免撑出黑边；带要求值的三角内文再按串长收缩。
  let numSize = value >= 10 ? 13 : 16
  let numY = kind === 'hp' ? 21 : 22.5
  if (kind === 'threat' && max) {
    numSize = label.length <= 3 ? 13 : label.length === 4 ? 10 : 8.2
    numY = label.length <= 3 ? 28 : 28.6
  }
  if (kind === 'hp') {
    return (
      <svg className="stat-token" viewBox="0 0 40 40" aria-hidden="true">
        <defs>
          <radialGradient id={gid} cx="50%" cy="40%" r="68%">
            <stop offset="0%" stopColor="#ffce4d" />
            <stop offset="45%" stopColor="#ff8c1f" />
            <stop offset="80%" stopColor="#e63a1c" />
            <stop offset="100%" stopColor="#7e1007" />
          </radialGradient>
          <pattern id={`${gid}-d`} width="3.6" height="3.6" patternUnits="userSpaceOnUse">
            <circle cx="1.8" cy="1.8" r="0.85" fill="rgba(64,7,2,0.42)" />
          </pattern>
        </defs>
        <DamageTokenArt gid={gid} />
        <text
          className="stat-token-num"
          x="20"
          y={numY}
          textAnchor="middle"
          dominantBaseline="central"
          fontSize={numSize}
        >
          {value}
        </text>
      </svg>
    )
  }
  return (
    <svg className="stat-token" viewBox="0 0 40 40" aria-hidden="true">
      <defs>
        <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#ffbe3d" />
          <stop offset="55%" stopColor="#f59012" />
          <stop offset="100%" stopColor="#d96a08" />
        </linearGradient>
      </defs>
      <polygon
        points="20,3 38,36.5 2,36.5"
        fill="#171114"
        stroke="#171114"
        strokeWidth="5"
        strokeLinejoin="round"
      />
      <polygon
        points="20,8.5 33.2,33 6.8,33"
        fill={`url(#${gid})`}
        stroke="#171114"
        strokeWidth="2.4"
        strokeLinejoin="round"
      />
      {/* 裂纹：桌面威胁代币的碎裂底纹 */}
      <g stroke="#6b3305" strokeWidth="1.1" fill="none" strokeLinecap="round" opacity="0.75">
        <path d="M20 12.5 L17.6 19 L21 24.5" />
        <path d="M17.6 19 L13.8 20.6" />
        <path d="M26.8 25 L23.4 22.4 L25 28.5" />
      </g>
      <text
        className="stat-token-num"
        x="20"
        y={numY}
        textAnchor="middle"
        dominantBaseline="central"
        fontSize={numSize}
        style={max && label.length > 3 ? { strokeWidth: 2.2 } : undefined}
      >
        {label}
      </text>
    </svg>
  )
}

interface Props {
  card: PlacedCard
  onClick?: (card: PlacedCard) => void
  className?: string
  zoom?: boolean
  fx?: CardFx
  // 多选时已选中的序号（1 起）
  selOrder?: number
}

// 数值徽章的语义提示：桌面端 hover/focus，触屏长按 450ms。长按触发后
// 吞掉同一次 click，避免玩家只是看标签却误打出/选择了卡牌。
function TaggedNumber({ className, label, children }: { className: string; label: string; children: ReactNode }) {
  const [touchVisible, setTouchVisible] = useState(false)
  const timer = useRef<number | null>(null)
  const longPressed = useRef(false)

  function clearTimer() {
    if (timer.current !== null) {
      window.clearTimeout(timer.current)
      timer.current = null
    }
  }

  function pointerDown(e: ReactPointerEvent<HTMLSpanElement>) {
    if (e.pointerType !== 'touch') return
    clearTimer()
    longPressed.current = false
    timer.current = window.setTimeout(() => {
      longPressed.current = true
      setTouchVisible(true)
    }, 450)
  }

  function pointerEnd() {
    clearTimer()
    if (longPressed.current) window.setTimeout(() => setTouchVisible(false), 1400)
  }

  useEffect(() => clearTimer, [])

  return (
    <span
      className={`${className} number-tag`}
      data-tag={label}
      data-tag-visible={touchVisible ? 'true' : undefined}
      aria-label={label}
      tabIndex={0}
      onPointerDown={pointerDown}
      onPointerUp={pointerEnd}
      onPointerCancel={pointerEnd}
      onPointerLeave={pointerEnd}
      onContextMenu={(e) => {
        if (longPressed.current) e.preventDefault()
      }}
      onClick={(e) => {
        if (!longPressed.current) return
        e.preventDefault()
        e.stopPropagation()
        longPressed.current = false
      }}
    >
      {children}
    </span>
  )
}

export default function GameCard({ card, onClick, className = '', zoom = true, fx, selOrder }: Props) {
  const lang = useLang()
  const t = useT()
  const zh = useZhMap()
  const safeCode = card.code && card.code !== 'undefined' && card.code !== 'null' ? card.code : ''
  const displayTitle = lname(zh, safeCode, card.title)
  const counterTag = safeCode === '50018' ? t('stat.missileCounter') : t('stat.counter')
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
      <div className="gcard-motion">
        {(card.kind === 'hero' || card.kind === 'villain') && <span className="character-fx" aria-hidden="true" />}
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

        {/* 状态芯片（title 用 status.* 文案） */}
        <div className="gcard-tokens">
          {card.stunned && <span className="tok tok-stun" title={t('status.stunned')}>✳</span>}
          {card.confused && <span className="tok tok-confuse" title={t('status.confused')}>?</span>}
          {card.tough && <span className="tok tok-tough" title={t('status.tough')}>◆</span>}
          {card.guard && <span className="tok tok-guard" title={t('status.guard')}>▲</span>}
          {card.firstPlayer && <span className="tok tok-first" title={t('status.first')}>★</span>}
          {Array.from({ length: Math.min(card.acceleration ?? 0, 6) }).map((_, i) => (
            <span key={i} className="tok tok-accel" title={t('status.acceleration')}>⏩</span>
          ))}
        </div>

        {/* 阶段/计数/强化徽章 */}
        <div className="gcard-topright">
          {card.stageLabel && <TaggedNumber className="chip chip-stage" label={t('stat.stage')}>{card.stageLabel}</TaggedNumber>}
          {card.counters !== undefined && card.counters > 0 && (
            <TaggedNumber className="chip chip-count" label={counterTag}>{card.counters}</TaggedNumber>
          )}
          {card.boosts !== undefined && card.boosts > 0 && (
            <TaggedNumber className="chip chip-boost" label={t('stat.boost')}>+{card.boosts}</TaggedNumber>
          )}
        </div>

        {/* 血量代币（剩余生命值） */}
        {card.hp !== undefined && card.maxHp ? (
          <TaggedNumber className={`hp-badge ${card.hp <= card.maxHp / 3 ? 'low' : ''}`} label={t('stat.hp')}>
            <StatToken kind="hp" value={card.hp} />
          </TaggedNumber>
        ) : null}

        {/* 威胁代币：当前值/要求值整体印在三角形上 */}
        {card.threat !== undefined && (
          <TaggedNumber className={`threat-bar ${card.maxThreat && card.threat >= card.maxThreat - 2 ? 'high' : ''}`} label={t('stat.threat')}>
            <StatToken kind="threat" value={card.threat} max={card.maxThreat} />
          </TaggedNumber>
        )}

        {/* 攻击/密谋/化解数值 */}
        {(card.attack !== undefined || card.thwart !== undefined) && (
          <div className="gcard-stats">
            {card.attack !== undefined && <TaggedNumber className="stat stat-atk" label={t('stat.attack')}><span className="stat-icon">⚔</span>{card.attack}</TaggedNumber>}
            {card.thwart !== undefined && <TaggedNumber className="stat stat-thw" label={t('stat.thwart')}><span className="stat-icon">◎</span>{card.thwart}</TaggedNumber>}
            {card.scheme !== undefined && <TaggedNumber className="stat stat-sch" label={t('stat.scheme')}><span className="stat-icon">◆</span>{card.scheme}</TaggedNumber>}
          </div>
        )}

        {/* 危机/危害 */}
        {(card.crisis || (card.hazard ?? 0) > 0) && (
          <div className="gcard-schemetags">
            {card.crisis && <span className="tag tag-crisis">{t('game.crisis')}</span>}
            {(card.hazard ?? 0) > 0 && <TaggedNumber className="tag tag-hazard" label={t('stat.hazard')}>☠{card.hazard}</TaggedNumber>}
          </div>
        )}
        </div>
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
  const isDiscard = card.label === 'discard' || card.label === 'encounter-discard'
  const isEmptyDiscard = isDiscard && count === 0 && !safeCode
  // 牌库/弃牌堆 title：{玩家名}的牌库 / {玩家名}的弃牌堆
  const displayTitle =
    card.label === 'deck'
      ? t('pile.deckTitle', card.title)
      : card.label === 'discard'
        ? t('pile.discardTitle', card.title)
        : card.label === 'sideDeck'
          ? t('pile.sideDeckTitle', card.title)
          : card.label === 'sideDiscard'
            ? t('pile.sideDiscardTitle', card.title)
            : card.label === 'encounter'
              ? t('pile.encounter')
              : card.label === 'encounter-discard'
                ? t('pile.encounterDiscard')
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
                // 顶牌盖在最上层背面上：跟随层堆的逐层平移，避免与堆错位
                style={{ translate: `${(layers - 1) * 2}px ${-(layers - 1) * 2.5}px` } as React.CSSProperties}
                src={cardUrl(safeCode, lang)}
                alt={card.title}
                draggable={false}
                // 横版卡图（主/支线密谋）在竖直弃牌堆里按实体牌横放：把元素
                // 本身换成旋转后的占位（176×127 居中，.pile-top-landscape），
                // 仅 rotate 90°。不能在 127×176 元素上 scale 放大——那会把
                // 边框/描边一起放大成超出牌堆的大空心框。
                onLoad={(e) => {
                  const img = e.currentTarget
                  img.classList.toggle('pile-top-landscape', img.naturalWidth > img.naturalHeight)
                }}
                onError={(e) => {
                  const img = e.currentTarget
                  if (!img.dataset.fallback) {
                    img.dataset.fallback = '1'
                    img.classList.remove('pile-top-landscape')
                    img.src = fallbackDataUrl(safeCode)
                  }
                }}
              />
            ) : null}
          </>
        )}
        {(count > 0 || isDiscard) && <TaggedNumber className="pile-count" label={t('stat.pileCount')}>{count}</TaggedNumber>}
      </div>
    </div>
  )
}
