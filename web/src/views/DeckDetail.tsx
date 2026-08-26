// 牌组详情：ArkhamDB 式布局。左侧英雄面板（宽幅色带头部 + 立绘 + 印刷
// 属性/能力文本 + 牌组统计），右侧牌组清单按类型分组双栏排布，每行
// 「数量 × 缩略图 费用徽章 名称 资源图标」，名称按阵营着色。
import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { get, allCards, CardInfo, Deck } from '../api'
import { CardImage, useCardZoom } from '../cards'
import { lname, lsubname, useT, useZhMap } from '../i18n'

function useOutsideClose(active: boolean, close: () => void) {
  useEffect(() => {
    if (!active) return
    const onDown = (ev: PointerEvent) => {
      const target = ev.target as HTMLElement | null
      if (target?.closest('.deck-row, .deck-hero')) return
      close()
    }
    window.addEventListener('pointerdown', onDown)
    return () => window.removeEventListener('pointerdown', onDown)
  }, [active, close])
}

const coarsePointer =
  typeof window.matchMedia === 'function' && window.matchMedia('(hover: none) and (pointer: coarse)').matches

const TYPE_ORDER = ['ally', 'event', 'support', 'upgrade', 'resource', 'player_side_scheme']
const ASPECT_ORDER = ['hero', 'aggression', 'justice', 'leadership', 'protection', 'basic']
const RES_GLYPH: Record<string, string> = { energy: '⚡', physical: '👊', mental: '🧠', wild: '⭐' }

interface DeckEntry {
  info: CardInfo
  count: number
}

// an-* 阵营配色类：名称/费用徽章/色带头部共用。hero 专属卡回落为中性色。
function aspectClass(a?: string): string {
  return `an-${a || 'hero'}`
}

// 卡牌编号：取 code 数字段的后三位（01001a → 1），packName #N 展示用。
function cardNumber(code: string): string {
  const digits = code.replace(/\D/g, '')
  if (digits.length > 2) return String(Number(digits.slice(2)))
  return digits
}

function DeckCardRow({ info, count, onPreview }: DeckEntry & { onPreview: (code: string) => void }) {
  const rowRef = useRef<HTMLDivElement | null>(null)
  const zoom = useCardZoom(info.code, rowRef)
  const zh = useZhMap()
  const t = useT()
  const ac = aspectClass(info.aspect)
  return (
    <div
      className="deck-row"
      role="button"
      ref={rowRef}
      onMouseEnter={zoom.onEnter}
      onMouseLeave={zoom.hide}
      onClick={() => onPreview(info.code)}
    >
      <span className="deck-count">{count}×</span>
      <span className="deck-thumb">
        <CardImage code={info.code} size="xs" zoom={false} />
      </span>
      {info.cost != null ? (
        <span className={`cost-pill ${ac}`}>{info.cost}</span>
      ) : (
        <span className="cost-pill empty" aria-hidden="true" />
      )}
      <span className={`deck-name ${ac}`}>
        {info.unique && <span className="uniq">• </span>}
        {lname(zh, info.code, info.name)}
        {info.subname ? ` (${lsubname(zh, info.code, info.subname)})` : ''}
      </span>
      <span className="deck-pips">
        {(info.resources ?? []).map((r, i) => (
          <span key={i} className="pip" title={t(`res.${r}`)}>
            {RES_GLYPH[r] ?? '◆'}
          </span>
        ))}
      </span>
      {zoom.overlay}
    </div>
  )
}

export default function DeckDetail() {
  const { id } = useParams<{ id: string }>()
  const t = useT()
  const zh = useZhMap()
  const [deck, setDeck] = useState<Deck | null>(null)
  const [catalog, setCatalog] = useState<Record<string, CardInfo>>({})
  const [previewCode, setPreviewCode] = useState<string | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    get<Deck>(`/marvel/decks/${id}`)
      .then(setDeck)
      .catch((err) => setError(String((err as Error).message)))
    allCards().then((list) => {
      const byCode: Record<string, CardInfo> = {}
      for (const c of list) byCode[c.code] = c
      setCatalog(byCode)
    })
  }, [id])

  const entries = useMemo<DeckEntry[]>(() => {
    if (!deck) return []
    return Object.entries(deck.slots)
      .map(([code, count]) => ({ info: catalog[code], count }))
      .filter((e) => e.info)
      .sort((a, b) => a.info.name.localeCompare(b.info.name))
  }, [deck, catalog])

  const stats = useMemo(() => {
    let total = 0
    const byType = new Map<string, number>()
    const byAspect = new Map<string, number>()
    const packs = new Set<string>()
    for (const { info, count } of entries) {
      total += count
      byType.set(info.type, (byType.get(info.type) ?? 0) + count)
      const aspect = info.aspect || 'hero'
      byAspect.set(aspect, (byAspect.get(aspect) ?? 0) + count)
      packs.add(info.packCode)
    }
    return { total, byType, byAspect, packs }
  }, [entries])

  // 所有 hooks 必须在下面的 error/loading 早退 return 之前调用：牌组是
  // 异步加载的，deck=null 的首帧与加载完成的第二帧 hook 数量必须一致，
  // 否则 React 报 "Rendered more hooks than during the previous render"。
  const heroZoomRef = useRef<HTMLDivElement | null>(null)
  const heroZoom = useCardZoom(deck?.investigatorCode ?? '', heroZoomRef)
  const previewRef = useRef<HTMLDivElement | null>(null)
  const previewZoom = useCardZoom(previewCode ?? deck?.investigatorCode ?? '', previewRef)
  useOutsideClose(previewCode !== null, () => setPreviewCode(null))
  useEffect(() => {
    if (previewCode && coarsePointer) previewZoom.show()
    else previewZoom.hide()
    // previewZoom identity changes with code; only react to the requested code.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [previewCode, coarsePointer])

  if (error) return <section><h2>{t('deck.title')}</h2><p className="error">{error}</p></section>
  if (!deck) return <section><p className="muted">{t('deck.loading')}</p></section>

  const hero = catalog[deck.investigatorCode]
  const heroName = hero ? lname(zh, hero.code, hero.name) : deck.investigatorCode
  const heroSub = hero ? lsubname(zh, hero.code, hero.subname ?? '') : ''
  // 扩充需求含英雄本身的扩充。
  const packCount = stats.packs.size + (hero && !stats.packs.has(hero.packCode) ? 1 : 0)

  const grouped = new Map<string, DeckEntry[]>()
  for (const e of entries) {
    // 英雄卡就是左侧面板的身份卡，不再进右侧类型清单。
    if (e.info.type === 'hero') continue
    const list = grouped.get(e.info.type) ?? []
    list.push(e)
    grouped.set(e.info.type, list)
  }
  for (const list of grouped.values()) {
    list.sort((a, b) => {
      const ca = a.info.cost ?? 999
      const cb = b.info.cost ?? 999
      return ca - cb || a.info.name.localeCompare(b.info.name)
    })
  }
  const typeKeys = [
    ...TYPE_ORDER.filter((t) => grouped.has(t)),
    ...[...grouped.keys()].filter((t) => !TYPE_ORDER.includes(t)).sort(),
  ]

  const heroStats: Array<[string, string, number | undefined]> = [
    ['❤️', t('stat.hp'), hero?.hp],
    ['⚔️', t('stat.attack'), hero?.attack],
    ['💬', t('stat.thwart'), hero?.thwart],
    ['🛡️', t('stat.defense'), hero?.defense],
    ['🔄', t('stat.recover'), hero?.recover],
    ['✋', t('stat.handSize'), hero?.handSize],
  ]

  return (
    <div className="deck-layout">
      <aside className="deck-side">
        <div
          className="deck-hero card"
          ref={heroZoomRef}
          onMouseEnter={heroZoom.onEnter}
          onMouseLeave={heroZoom.hide}
          onClick={() => setPreviewCode(deck.investigatorCode)}
        >
          <div className={`dhp-header ${aspectClass(hero?.aspect)}`}>
            <strong>
              {hero?.unique ? '★ ' : ''}
              {heroName}
            </strong>
            {heroSub && <span>{heroSub}</span>}
          </div>
          <div className="dhp-body">
            <div className="dhp-info">
              <div className="dhp-label">{hero ? t(`type.${hero.type}`) : ''}</div>
              <div className="row wrap dhp-stats">
                {heroStats
                  .filter(([, , v]) => v != null)
                  .map(([glyph, label, v]) => (
                    <span key={label} className="dhp-stat" title={label}>
                      <span className="dhp-glyph">{glyph}</span>
                      {v}
                    </span>
                  ))}
              </div>
              {hero?.text && (
                <div className="dhp-text">
                  {hero.text.replace(/<[^>]*>/g, '')}
                </div>
              )}
              {hero && (
                <div className="dhp-pack muted">
                  {hero.packName ?? hero.packCode} #{cardNumber(hero.code)}
                </div>
              )}
            </div>
            <CardImage code={deck.investigatorCode} size="lg" zoom={false} />
          </div>
          <div className="dhp-foot">
            <div className="dhp-foot-cell">
              <span className="muted">{t('deck.size')}</span>
              <strong>{t('deck.heroStats', stats.total, entries.length)}</strong>
            </div>
            <div className="dhp-foot-cell">
              <span className="muted">{t('deck.packs')}</span>
              <strong>{packCount}</strong>
            </div>
          </div>
        </div>
        <p className="muted deck-back">
          <Link to="/decks">{t('deck.back')}</Link>
        </p>
      </aside>

      <div className="deck-main">
        <h2>{deck.name}</h2>
        <div className="row wrap deck-chips">
          {ASPECT_ORDER.filter((a) => stats.byAspect.has(a)).map((a) => (
            <span key={a} className="deck-chip">
              <span className={`dot ${aspectClass(a)}`} />
              {t(`aspect.${a}`)} ({stats.byAspect.get(a)})
            </span>
          ))}
          <span className="deck-chip">{t('deck.heroStats', stats.total, entries.length)}</span>
          <span className="deck-chip">
            {t('deck.packs')} {packCount}
          </span>
        </div>
        <div className="deck-cols">
          {typeKeys.map((key) => (
            <div key={key} className="deck-group">
              <h3>
                {t(`type.${key}`)} <span className="badge">{stats.byType.get(key)}</span>
              </h3>
              {grouped.get(key)!.map((e) => (
                <DeckCardRow key={e.info.code} info={e.info} count={e.count} onPreview={setPreviewCode} />
              ))}
            </div>
          ))}
        </div>
      </div>
      {heroZoom.overlay}
      {previewCode && coarsePointer && <div ref={previewRef} className="preview-anchor" />}
      {previewCode && coarsePointer && previewZoom.overlay}
    </div>
  )
}
