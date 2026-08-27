// 创建牌组第一步：选择英雄。列出全部英雄身份（a 面），支持中英文名与
// 扩展包搜索；点击进入组牌器 /decks/new/build?hero={base}。
import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { get, HeroInfo } from '../api'
import { CardImage } from '../cards'
import { lname, useT, useZhMap } from '../i18n'

export default function HeroPicker() {
  const navigate = useNavigate()
  const t = useT()
  const zh = useZhMap()
  const [heros, setHeros] = useState<HeroInfo[]>([])
  const [q, setQ] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    get<HeroInfo[]>('/marvel/heroes')
      .then(setHeros)
      .catch((err) => setError(String((err as Error).message)))
  }, [])

  const filtered = useMemo(() => {
    const needle = q.trim().toLowerCase()
    return heros
      .filter((h) => {
        if (!needle) return true
        const en = `${h.name} ${h.alterEgoName ?? ''} ${h.packCode}`.toLowerCase()
        const zhName = zh ? `${zh[h.heroCode]?.name ?? ''} ${zh[h.alterEgoCode]?.name ?? ''}` : ''
        return en.includes(needle) || zhName.includes(needle)
      })
      .sort((a, b) => a.packCode.localeCompare(b.packCode) || a.name.localeCompare(b.name))
  }, [heros, q, zh])

  return (
    <section>
      <h2>{t('decks.create')}</h2>
      {error && <p className="error">{error}</p>}
      <p className="muted">{t('builder.chooseHero')}</p>
      <input
        className="builder-search"
        placeholder={t('builder.searchHero')}
        value={q}
        onChange={(e) => setQ(e.target.value)}
      />
      <div className="hero-grid">
        {filtered.map((h) => (
          <button
            key={h.base}
            className="hero-card card"
            onClick={() => navigate(`/decks/new/build?hero=${h.base}`)}
          >
            <CardImage code={h.heroCode} size="sm" zoom={false} />
            <strong>{lname(zh, h.heroCode, h.name)}</strong>
            {h.alterEgoName && (
              <span className="muted">{lname(zh, h.alterEgoCode, h.alterEgoName)}</span>
            )}
            <span className="muted hero-pack">{h.packCode}</span>
          </button>
        ))}
      </div>
      {heros.length > 0 && filtered.length === 0 && (
        <p className="muted">{t('builder.noHeroMatch')}</p>
      )}
    </section>
  )
}
