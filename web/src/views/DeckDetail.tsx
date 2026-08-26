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

interface DeckEntry {
  info: CardInfo
  count: number
}

function DeckCardRow({ info, count, onPreview }: DeckEntry & { onPreview: (code: string) => void }) {
  const rowRef = useRef<HTMLDivElement | null>(null)
  const zoom = useCardZoom(info.code, rowRef)
  const zh = useZhMap()
  return (
    <div
      className="card row deck-row"
      ref={rowRef}
      onMouseEnter={zoom.onEnter}
      onMouseLeave={zoom.hide}
      onClick={() => onPreview(info.code)}
    >
      <span className="deck-count">{count}x</span>
      <span className="deck-thumb">
        <CardImage code={info.code} size="xs" zoom={false} />
      </span>
      <div style={{ flex: 1 }}>
        <strong>
          {lname(zh, info.code, info.name)}
          {info.subname ? ` (${lsubname(zh, info.code, info.subname)})` : ''}
        </strong>
        <div className="muted">{info.packName ?? info.packCode}</div>
      </div>
      {info.cost != null && <span className="badge">{info.cost}</span>}
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
    const costBuckets = [0, 0, 0, 0, 0, 0] // 0..4, 5+
    const byResource = new Map<string, number>()
    for (const { info, count } of entries) {
      total += count
      byType.set(info.type, (byType.get(info.type) ?? 0) + count)
      const aspect = info.aspect || 'hero'
      byAspect.set(aspect, (byAspect.get(aspect) ?? 0) + count)
      if (info.cost != null) {
        const bucket = Math.min(info.cost, 5)
        costBuckets[bucket] += count
      }
      for (const res of info.resources ?? []) {
        byResource.set(res, (byResource.get(res) ?? 0) + count)
      }
    }
    return { total, byType, byAspect, costBuckets, byResource }
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
  const maxCost = Math.max(1, ...stats.costBuckets)
  const grouped = new Map<string, DeckEntry[]>()
  for (const e of entries) {
    const list = grouped.get(e.info.type) ?? []
    list.push(e)
    grouped.set(e.info.type, list)
  }
  const typeKeys = [
    ...TYPE_ORDER.filter((t) => grouped.has(t)),
    ...[...grouped.keys()].filter((t) => !TYPE_ORDER.includes(t)).sort(),
  ]
  const aspectOrder = ['hero', 'basic', 'aggression', 'justice', 'leadership', 'protection']

  return (
    <section>
      <h2>{deck.name}</h2>
      <div
        className="card row deck-hero"
        ref={heroZoomRef}
        onMouseEnter={heroZoom.onEnter}
        onMouseLeave={heroZoom.hide}
        onClick={() => setPreviewCode(deck.investigatorCode)}
      >
        <CardImage code={deck.investigatorCode} size="lg" zoom={false} />
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '0.6rem' }}>
          <div>
            <strong>{heroName}</strong>
            <span className="muted">
              {' '}{t('deck.heroStats', { total: stats.total, distinct: entries.length })}
            </span>
          </div>
          <div className="row wrap">
            {[...typeKeys.map((key) => [t(`type.${key}`), stats.byType.get(key)!] as const)]
              .map(([label, n]) => (
                <span key={label} className="badge">{label}: {n}</span>
              ))}
          </div>
          <div className="row wrap">
            {aspectOrder
              .filter((a) => stats.byAspect.has(a))
              .map((a) => (
                <span key={a} className="badge">{t(`aspect.${a}`)}: {stats.byAspect.get(a)}</span>
              ))}
          </div>
          <div className="row wrap stat-row">
            {[...stats.byResource.entries()]
              .sort((a, b) => b[1] - a[1])
              .map(([res, n]) => (
                <span key={res} className="muted">
                  {t(`res.${res}`)}: {n}
                </span>
              ))}
          </div>
          <div className="cost-curve">
            {stats.costBuckets.map((n, i) => (
              <div key={i} className="cost-col" title={t('deck.costTitle', { cost: i === 5 ? '5+' : i, n })}>
                <div className="cost-bar" style={{ height: `${Math.round((n / maxCost) * 100)}%` }} />
                <span className="muted">{i === 5 ? '5+' : i}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
      {typeKeys.map((key) => (
        <div key={key}>
          <h3>
            {t(`type.${key}`)}{' '}
            <span className="muted">({stats.byType.get(key)})</span>
          </h3>
          <div className="deck-list">
            {grouped.get(key)!.map((e) => (
              <DeckCardRow key={e.info.code} info={e.info} count={e.count} onPreview={setPreviewCode} />
            ))}
          </div>
        </div>
      ))}
      <p className="muted">
        <Link to="/decks">{t('deck.back')}</Link>
      </p>
      {heroZoom.overlay}
      {previewCode && coarsePointer && <div ref={previewRef} className="preview-anchor" />}
      {previewCode && coarsePointer && previewZoom.overlay}
    </section>
  )
}
