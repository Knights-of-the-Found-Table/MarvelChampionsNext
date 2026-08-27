// 组牌器第二步：三栏布局（参照主机组牌界面）。左侧身份面板（复用牌组
// 详情的英雄卡样式）+ 派系选择器 + 实时规则校验 + 保存；中间当前牌组
// （英雄必备卡锁定，其余按类型分组增减）；右侧可加卡池（搜索 + 类型
// 筛选）。派系纪律：加入第一张某派系的牌即锁定选择器，卡池自动过滤
// 其他派系；手动改派系会移除不符合的卡（二次确认）。例外匹配与加牌
// 上限用 deckRules.ts 的引擎镜像，最终校验以服务端 /decks/validate
// （engine.ValidateDeck）为准。
import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { allCards, post, CardInfo, DeckIssue } from '../api'
import {
  ASPECT_OPTIONS,
  DECK_MAX,
  DECK_MIN,
  POOL,
  aspectCapacity,
  deckCardType,
  exceptionMatches,
} from '../deckRules'
import { describeDeckIssues } from '../deckIssues'
import { CardImage, useCardZoom } from '../cards'
import { lname, lsubname, useT, useZhMap } from '../i18n'
import { ResourceIcon } from '../components/ResourceIcon'
import { StatIcon } from '../components/StatIcon'

const TYPE_ORDER = ['ally', 'event', 'support', 'upgrade', 'resource', 'player_side_scheme']
// 卡池懒加载：首屏只渲染前 POOL_PAGE 行，滚动到底部按需加载更多。
const POOL_PAGE = 150

interface ConfirmState {
  title: string
  body: string
  action: () => void
}

interface CardRowProps {
  info: CardInfo
  count: number
  // 英雄必备卡：锁定张数，无增减控件。
  locked?: boolean
  canAdd?: boolean
  canRemove?: boolean
  onAdd?: () => void
  onRemove?: () => void
  onClick?: () => void
}

// 牌组行与卡池行共用：左起 数量 × 缩略图 费用徽章 名称 资源图标，
// 右侧锁定标记或 +/− 控件；行悬停放大卡图。
function CardRow({ info, count, locked, canAdd, canRemove, onAdd, onRemove, onClick }: CardRowProps) {
  const rowRef = useRef<HTMLDivElement | null>(null)
  const zoom = useCardZoom(info.code, rowRef)
  const t = useT()
  const zh = useZhMap()
  const ac = `an-${info.aspect || 'hero'}`
  return (
    <div
      className={`deck-row builder-row${onClick && canAdd ? ' clickable' : ''}`}
      ref={rowRef}
      onMouseEnter={zoom.onEnter}
      onMouseLeave={zoom.hide}
      onClick={onClick}
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
            <ResourceIcon icon={r} size={13} />
          </span>
        ))}
      </span>
      {locked ? (
        <span className="builder-lock" title={t('builder.heroSet')}>
          🔒
        </span>
      ) : (
        <span className="builder-controls">
          <button
            type="button"
            className="builder-btn"
            disabled={!canRemove}
            aria-label="-"
            onClick={(e) => {
              e.stopPropagation()
              onRemove?.()
            }}
          >
            −
          </button>
          <button
            type="button"
            className="builder-btn builder-add"
            disabled={!canAdd}
            aria-label="+"
            onClick={(e) => {
              e.stopPropagation()
              onAdd?.()
            }}
          >
            +
          </button>
        </span>
      )}
      {zoom.overlay}
    </div>
  )
}

export default function DeckBuilder() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const t = useT()
  const zh = useZhMap()
  const heroBase = params.get('hero') ?? ''
  const heroCode = /^\d{5}a$/.test(heroBase) ? heroBase : `${heroBase}a`

  const [catalog, setCatalog] = useState<Record<string, CardInfo>>({})
  const [ready, setReady] = useState(false)
  const [name, setName] = useState('')
  const [slots, setSlots] = useState<Record<string, number>>({})
  const [aspects, setAspects] = useState<string[]>([])
  const [issues, setIssues] = useState<DeckIssue[]>([])
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState('all')
  const [poolLimit, setPoolLimit] = useState(POOL_PAGE)
  const [confirm, setConfirm] = useState<ConfirmState | null>(null)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const seeded = useRef('')
  const validateSeq = useRef(0)
  // 首次校验响应落地前不显示「合规」绿板（避免冷缓存时误报）。
  const [validated, setValidated] = useState(false)

  useEffect(() => {
    allCards()
      .then((list) => {
        const byCode: Record<string, CardInfo> = {}
        for (const c of list) byCode[c.code] = c
        setCatalog(byCode)
        setReady(true)
      })
      .catch((err) => setError(String((err as Error).message)))
  }, [])

  // ---- 身份与骑手（印在化身面上的结构化字段）----
  const heroDef = catalog[heroCode]
  const aeCode = /^\d{5}a$/.test(heroCode) ? `${heroCode.slice(0, 5)}b` : ''
  const aeDef = aeCode ? catalog[aeCode] : undefined
  const mode = aeDef?.aspectMode ?? ''
  const exception = aeDef?.aspectException
  const uniqueAll = aeDef?.uniqueAll ?? false
  const capacity = aspectCapacity(mode)
  const poolAllowed = heroDef?.cardSet === 'deadpool' && mode === ''
  const aspectOptions = poolAllowed ? [...ASPECT_OPTIONS, POOL] : [...ASPECT_OPTIONS]
  const heroName = heroDef ? lname(zh, heroDef.code, heroDef.name) : heroCode

  // 英雄套装：必带卡，与引擎同一套过滤（可组类型 + 印刷张数 > 0）。
  const heroSet = useMemo(() => {
    const map: Record<string, CardInfo> = {}
    if (!heroDef) return map
    for (const c of Object.values(catalog)) {
      if (c.cardSet === heroDef.cardSet && c.category === 'player' && deckCardType(c.type) && (c.quantity ?? 0) > 0) {
        map[c.code] = c
      }
    }
    return map
  }, [catalog, heroDef])

  // 所有英雄拥有专属套装的套装代码：别的英雄的套装卡不能进牌组
  // （校验器报 cardIllegal），卡池直接不展示。
  const heroOwnedSets = useMemo(() => {
    const sets = new Set<string>()
    for (const c of Object.values(catalog)) {
      if (c.type === 'hero' && c.cardSet) sets.add(c.cardSet)
    }
    return sets
  }, [catalog])

  // 选定英雄后播种：必备卡按印刷张数入场并锁定；魔士亚当四派系常开。
  useEffect(() => {
    if (!heroDef || seeded.current === heroCode) return
    seeded.current = heroCode
    const seed: Record<string, number> = {}
    for (const code of Object.keys(heroSet)) seed[code] = heroSet[code].quantity ?? 0
    setSlots(seed)
    setAspects(mode === 'four_equal' ? [...ASPECT_OPTIONS] : [])
    setName(t('builder.defaultName', lname(zh, heroCode, heroDef.name)))
    // 种子只在英雄切换时落一次，t/zh 变化不重播。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [heroDef, heroSet, heroCode])

  const entries = useMemo<{ info: CardInfo; count: number }[]>(
    () =>
      Object.entries(slots)
        .map(([code, count]) => ({ info: catalog[code], count }))
        .filter((e) => e.info && e.count > 0),
    [slots, catalog],
  )

  const stats = useMemo(() => {
    let total = 0
    const byType = new Map<string, number>()
    for (const { info, count } of entries) {
      total += count
      byType.set(info.type, (byType.get(info.type) ?? 0) + count)
    }
    return { total, byType }
  }, [entries])

  const setEntries = useMemo(
    () =>
      Object.keys(heroSet)
        .sort()
        .map((code) => ({ info: heroSet[code], count: slots[code] ?? 0 })),
    [heroSet, slots],
  )
  const setTotal = setEntries.reduce((a, e) => a + e.count, 0)

  const grouped = useMemo(() => {
    const g = new Map<string, { info: CardInfo; count: number }[]>()
    for (const e of entries) {
      if (heroSet[e.info.code]) continue
      const list = g.get(e.info.type) ?? []
      list.push(e)
      g.set(e.info.type, list)
    }
    for (const list of g.values()) {
      list.sort((a, b) => (a.info.cost ?? 999) - (b.info.cost ?? 999) || a.info.name.localeCompare(b.info.name))
    }
    return g
  }, [entries, heroSet])
  const typeKeys = [
    ...TYPE_ORDER.filter((k) => grouped.has(k)),
    ...[...grouped.keys()].filter((k) => !TYPE_ORDER.includes(k)).sort(),
  ]
  const customTotal = stats.total - setTotal

  // 按牌名记账：复制上限（唯一 1 / 非唯一 3，魔士亚当全 1）与骑手豁免
  // 卡的 Total/Titles 上限都要用。
  const titleInfo = useMemo(() => {
    const counts = new Map<string, number>()
    const unique = new Set<string>()
    for (const { info, count } of entries) {
      if (heroSet[info.code]) continue
      counts.set(info.name, (counts.get(info.name) ?? 0) + count)
      if (info.unique) unique.add(info.name)
    }
    return { counts, unique }
  }, [entries, heroSet])

  const aspectCovers = useMemo(() => {
    const chosen = new Set(aspects)
    return (info: CardInfo) => info.aspect === 'basic' || (info.aspect !== '' && info.aspect !== undefined && chosen.has(info.aspect))
  }, [aspects])

  const exceptUsage = useMemo(() => {
    let n = 0
    const titles = new Set<string>()
    if (!exception) return { n, titles }
    for (const { info, count } of entries) {
      if (heroSet[info.code]) continue
      if (aspectCovers(info)) continue
      if (!exceptionMatches(exception, info)) continue
      n += count
      titles.add(info.name)
    }
    return { n, titles }
  }, [entries, heroSet, exception, aspectCovers])

  // 实时校验：与服务端同一份规则（engine.ValidateDeck），防抖 250ms。
  // 响应按序号丢弃陈旧结果（图片请求占满连接池时校验响应可能乱序到达）。
  useEffect(() => {
    if (!heroDef) return
    const id = window.setTimeout(() => {
      const seq = ++validateSeq.current
      post<{ valid: boolean; issues: DeckIssue[] }>('/marvel/decks/validate', {
        investigatorCode: heroCode,
        slots,
      })
        .then((r) => {
          if (seq === validateSeq.current) {
            setIssues(r.issues ?? [])
            setValidated(true)
          }
        })
        .catch(() => {})
    }, 250)
    return () => window.clearTimeout(id)
  }, [heroDef, heroCode, slots])

  useEffect(() => {
    setPoolLimit(POOL_PAGE)
  }, [search, typeFilter])

  // ---- 增减卡牌 ----
  function canAdd(info: CardInfo): boolean {
    if (!heroDef || stats.total >= DECK_MAX) return false
    if (heroSet[info.code]) return false
    const count = slots[info.code] ?? 0
    if (count >= (info.quantity ?? 0)) return false
    const titleCap = uniqueAll || info.unique || titleInfo.unique.has(info.name) ? 1 : 3
    if ((titleInfo.counts.get(info.name) ?? 0) >= titleCap) return false
    if (exception && !aspectCovers(info) && exceptionMatches(exception, info)) {
      if (exception.total && exceptUsage.n >= exception.total) return false
      if (exception.titles && !exceptUsage.titles.has(info.name) && exceptUsage.titles.size >= exception.titles) {
        return false
      }
    }
    return true
  }

  function addCard(code: string) {
    const info = catalog[code]
    if (!info || !canAdd(info)) return
    setSlots((s) => ({ ...s, [code]: (s[code] ?? 0) + 1 }))
    // 加入第一张某派系的牌即锁定该派系（骑手豁免卡不触发锁定）。
    const a = info.aspect ?? ''
    const lockable = a !== '' && a !== 'basic' && (a === POOL ? poolAllowed : ASPECT_OPTIONS.includes(a))
    if (lockable && aspects.length < capacity && !exceptionMatches(exception, info)) {
      setAspects((cur) => (cur.includes(a) ? cur : [...cur, a]))
    }
  }

  function removeCard(code: string) {
    if (heroSet[code]) return
    setSlots((s) => {
      const n = (s[code] ?? 0) - 1
      const next = { ...s }
      if (n <= 0) delete next[code]
      else next[code] = n
      return next
    })
  }

  // ---- 派系切换：不属于新派系集的卡全部移除（骑手豁免卡除外）----
  function removalsInto(next: string[]): number {
    const keep = new Set(next)
    let n = 0
    for (const { info, count } of entries) {
      if (heroSet[info.code]) continue
      if (info.aspect === 'basic') continue
      if (keep.has(info.aspect ?? '')) continue
      if (exceptionMatches(exception, info)) continue
      n += count
    }
    return n
  }

  function applyAspects(next: string[]) {
    const keep = new Set(next)
    setAspects(next)
    setSlots((s) => {
      const out: Record<string, number> = {}
      for (const [code, n] of Object.entries(s)) {
        const info = catalog[code]
        if (
          !info ||
          heroSet[code] ||
          info.aspect === 'basic' ||
          keep.has(info.aspect ?? '') ||
          exceptionMatches(exception, info)
        ) {
          out[code] = n
        }
      }
      return out
    })
  }

  function clickAspect(a: string) {
    if (mode === 'four_equal') return
    if (aspects.includes(a)) {
      // 双派系模式下可放弃一个（回到单派系）；单派系模式点当前派系无操作。
      if (mode !== 'two_equal' || aspects.length <= 1) return
      const next = aspects.filter((x) => x !== a)
      const n = removalsInto(next)
      setConfirm({
        title: t('builder.dropAspect', t(`aspect.${a}`)),
        body: n > 0 ? t('builder.changeConfirm', n) : t('builder.changeNone'),
        action: () => applyAspects(next),
      })
      return
    }
    if (aspects.length < capacity) {
      applyAspects([...aspects, a])
      return
    }
    // 容量已满：确认后清掉不属于新派系的卡再切换。
    const n = removalsInto([a])
    setConfirm({
      title: t('builder.switchAspect', t(`aspect.${a}`)),
      body: n > 0 ? t('builder.changeConfirm', n) : t('builder.changeNone'),
      action: () => applyAspects([a]),
    })
  }

  function resetCustom() {
    setConfirm({
      title: t('builder.reset'),
      body: t('builder.resetConfirm'),
      action: () =>
        setSlots(() => {
          const out: Record<string, number> = {}
          for (const code of Object.keys(heroSet)) out[code] = heroSet[code].quantity ?? 0
          return out
        }),
    })
  }

  async function save() {
    setSaving(true)
    setError('')
    try {
      const r = await post<{ id: string }>('/marvel/decks', {
        name: name.trim() || t('builder.defaultName', heroName),
        investigatorCode: heroCode,
        slots,
      })
      navigate(`/decks/${r.id}`)
    } catch (err) {
      setError(String((err as Error).message))
    } finally {
      setSaving(false)
    }
  }

  // ---- 可加卡池：可选派系 + 基础 + 骑手豁免；搜索 + 类型筛选 ----
  const pool = useMemo(() => {
    if (!heroDef) return []
    const chosen = new Set(aspects)
    const needle = search.trim().toLowerCase()
    const out: CardInfo[] = []
    for (const c of Object.values(catalog)) {
      if (c.category !== 'player' || !deckCardType(c.type) || (c.quantity ?? 0) <= 0) continue
      if (c.cardSet && heroOwnedSets.has(c.cardSet)) continue
      const a = c.aspect ?? ''
      if (a === 'basic') {
        // 基础卡恒可加
      } else if (exceptionMatches(exception, c)) {
        // 骑手豁免卡：任意派系
      } else if (a === POOL) {
        if (!poolAllowed || !(chosen.has(a) || aspects.length < capacity)) continue
      } else if (a !== '') {
        if (!(chosen.has(a) || aspects.length < capacity)) continue
      } else {
        // 无派系又不命中豁免：不能组进玩家牌组
        continue
      }
      if (typeFilter !== 'all' && c.type !== typeFilter) continue
      if (needle) {
        const en = `${c.name} ${c.subname ?? ''} ${c.packCode}`.toLowerCase()
        const zhName = zh?.[c.code]?.name ?? ''
        if (!en.includes(needle) && !zhName.includes(needle)) continue
      }
      out.push(c)
    }
    out.sort((x, y) => (x.cost ?? 99) - (y.cost ?? 99) || x.name.localeCompare(y.name))
    return out
  }, [catalog, heroDef, heroOwnedSets, aspects, exception, poolAllowed, capacity, typeFilter, search, zh])

  // ---- 早退（全部 hooks 之后）----
  if (!heroBase) return <Navigate to="/decks/new" replace />
  if (!heroDef) {
    return ready ? (
      <Navigate to="/decks/new" replace />
    ) : (
      <section>
        <p className="muted">{t('deck.loading')}</p>
      </section>
    )
  }

  const statOf = (k: 'hp' | 'attack' | 'thwart' | 'defense' | 'recover' | 'handSize') =>
    heroDef?.[k] ?? aeDef?.[k]
  const heroStats: Array<[string, string, number | undefined]> = [
    ['thwart', t('stat.thwart'), statOf('thwart')],
    ['attack', t('stat.attack'), statOf('attack')],
    ['defense', t('stat.defense'), statOf('defense')],
    ['recover', t('stat.recover'), statOf('recover')],
    ['hand', t('stat.handSize'), statOf('handSize')],
    ['hp', t('stat.hp'), statOf('hp')],
  ]
  let identSub = aeDef ? lname(zh, aeDef.code, aeDef.name) : ''
  if (!identSub) identSub = lsubname(zh, heroCode, heroDef.subname) ?? ''
  if (identSub === heroName) identSub = ''

  return (
    <div className="builder-layout">
      <aside className="builder-side">
        <div className="deck-hero card">
          <div className={`dhp-header an-${heroDef.aspect || 'hero'}`}>
            <strong>
              {heroDef.unique ? '★ ' : ''}
              {heroName}
            </strong>
            {identSub && <span>{identSub}</span>}
          </div>
          <div className="row wrap dhp-stats-strip">
            {heroStats
              .filter(([, , v]) => v != null)
              .map(([stat, label, v]) => (
                <span key={stat} className="dhp-stat" title={label}>
                  <StatIcon stat={stat} size={16} />
                  {v}
                </span>
              ))}
          </div>
          <div className="dhp-body">
            <div className="dhp-faces">
              <div className="dhp-face">
                <CardImage code={heroCode} size="md" />
              </div>
              {aeDef && (
                <div className="dhp-face">
                  <CardImage code={aeCode} size="md" />
                </div>
              )}
            </div>
            <div className="dhp-details">
              <div className="dhp-label">{t('deck.details')}</div>
              <div className="dhp-detail">
                <span className="dhp-detail-key">{t('deck.size')}</span>
                <strong
                  className={
                    stats.total < DECK_MIN
                      ? 'builder-size-low'
                      : stats.total > DECK_MAX
                        ? 'builder-size-high'
                        : 'builder-size-ok'
                  }
                >
                  {stats.total} / {DECK_MIN}–{DECK_MAX}
                </strong>
              </div>
              <div className="dhp-detail">
                <span className="dhp-detail-key">{t('deck.aspects')}</span>
                <div className="builder-aspects">
                  {aspectOptions.map((a) => {
                    const fixed = mode === 'four_equal'
                    return (
                      <button
                        key={a}
                        type="button"
                        className={`aspect-pill ap-${a}${aspects.includes(a) ? ' active' : ''}`}
                        disabled={fixed}
                        onClick={() => clickAspect(a)}
                      >
                        <span className={`dot an-${a}`} />
                        {t(`aspect.${a}`)}
                        {fixed ? ' 🔒' : ''}
                      </button>
                    )
                  })}
                </div>
                <div className="muted builder-mode">
                  {mode ? t(`builder.mode.${mode}`) : t('builder.aspectHint')}
                </div>
              </div>
            </div>
          </div>
        </div>
        {issues.length > 0 ? (
          <div className="deck-issues" role="alert">
            <strong>⚠ {t('deck.issueTitle')}</strong>
            <ul>
              {describeDeckIssues(t, zh, issues).map((s, i) => (
                <li key={i}>{s}</li>
              ))}
            </ul>
            <p className="muted">{t('deck.issuesHint')}</p>
          </div>
        ) : (
          validated && <div className="builder-legal">✓ {t('builder.legal')}</div>
        )}
        <div className="card builder-save">
          <input
            value={name}
            placeholder={t('builder.defaultName', heroName)}
            onChange={(e) => setName(e.target.value)}
          />
          <div className="row">
            <button type="button" className="primary" disabled={saving} onClick={save}>
              {t('builder.save')}
            </button>
            <button type="button" disabled={saving || customTotal === 0} onClick={resetCustom}>
              {t('builder.reset')}
            </button>
          </div>
          {error && <p className="error">{error}</p>}
          <p className="muted deck-back">
            <Link to="/decks/new">{t('builder.back')}</Link>
          </p>
        </div>
      </aside>

      <div className="builder-main">
        <div className="deck-group">
          <h3>
            {t('builder.heroSet')} <span className="badge">{setTotal}</span>
          </h3>
          {setEntries.map((e) => (
            <CardRow key={e.info.code} info={e.info} count={e.count} locked />
          ))}
        </div>
        {typeKeys.map((k) => (
          <div key={k} className="deck-group">
            <h3>
              {t(`type.${k}`)} <span className="badge">{stats.byType.get(k)}</span>
            </h3>
            {grouped.get(k)!.map((e) => (
              <CardRow
                key={e.info.code}
                info={e.info}
                count={e.count}
                canAdd={canAdd(e.info)}
                canRemove
                onAdd={() => addCard(e.info.code)}
                onRemove={() => removeCard(e.info.code)}
              />
            ))}
          </div>
        ))}
        {customTotal === 0 && <p className="muted">{t('builder.emptyDeck')}</p>}
      </div>

      <div className="builder-pool">
        <input
          className="builder-search"
          placeholder={t('builder.search')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <div className="builder-chips">
          <button
            type="button"
            className={`builder-chip${typeFilter === 'all' ? ' active' : ''}`}
            onClick={() => setTypeFilter('all')}
          >
            {t('builder.filterAll')}
          </button>
          {TYPE_ORDER.map((k) => (
            <button
              key={k}
              type="button"
              className={`builder-chip${typeFilter === k ? ' active' : ''}`}
              onClick={() => setTypeFilter(k)}
            >
              {t(`type.${k}`)}
            </button>
          ))}
        </div>
        <div className="muted builder-pool-count">{t('builder.poolCount', pool.length)}</div>
        <div className="builder-pool-list">
          {pool.slice(0, poolLimit).map((c) => (
            <CardRow
              key={c.code}
              info={c}
              count={slots[c.code] ?? 0}
              canAdd={canAdd(c)}
              canRemove={(slots[c.code] ?? 0) > 0}
              onAdd={() => addCard(c.code)}
              onRemove={() => removeCard(c.code)}
              onClick={() => addCard(c.code)}
            />
          ))}
          {pool.length > poolLimit && (
            <button type="button" className="builder-more" onClick={() => setPoolLimit((l) => l + 300)}>
              {t('builder.showMore')}
            </button>
          )}
          {pool.length === 0 && <p className="muted">{t('builder.poolEmpty')}</p>}
        </div>
      </div>

      {confirm && (
        <div className="builder-modal" role="dialog" aria-modal="true" onClick={() => setConfirm(null)}>
          <div className="builder-modal-card card" onClick={(e) => e.stopPropagation()}>
            <h3>{confirm.title}</h3>
            <p>{confirm.body}</p>
            <div className="row" style={{ justifyContent: 'flex-end' }}>
              <button type="button" onClick={() => setConfirm(null)}>
                {t('builder.cancel')}
              </button>
              <button
                type="button"
                className="danger"
                onClick={() => {
                  const act = confirm.action
                  setConfirm(null)
                  act()
                }}
              >
                {t('builder.confirm')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
