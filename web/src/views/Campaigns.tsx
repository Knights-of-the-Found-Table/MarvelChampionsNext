import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { createCampaign, Deck, get, listCampaigns, CampaignSummary } from '../api'
import { useT } from '../i18n'

const BOXES = [
  { key: 'rrs', label: 'campaigns.box.rrs' },
  { key: 'gmw', label: 'campaigns.box.gmw' },
]

export default function Campaigns() {
  const t = useT()
  const navigate = useNavigate()
  const [list, setList] = useState<CampaignSummary[]>([])
  const [decks, setDecks] = useState<Deck[]>([])
  const [box, setBox] = useState('rrs')
  const [difficulty, setDifficulty] = useState('standard')
  const [playerCount, setPlayerCount] = useState(1)
  const [deckId, setDeckId] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    listCampaigns().then(setList).catch(() => {})
    get<Deck[]>('/marvel/decks').then((d) => {
      setDecks(d)
      const first = d.find((x) => x.valid !== false) ?? d[0]
      if (first) setDeckId(first.id)
    })
  }, [])

  async function create(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      const detail = await createCampaign({
        box,
        difficulty,
        playerCount,
        deckId: playerCount === 1 ? deckId : undefined,
      })
      navigate(`/campaigns/${detail.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section>
      <h2>{t('campaigns.title')}</h2>
      {error && <p className="error">{error}</p>}
      <form className="card form" onSubmit={create}>
        <label>
          {t('campaigns.box')}
          <select value={box} onChange={(e) => setBox(e.target.value)}>
            {BOXES.map((b) => (
              <option key={b.key} value={b.key}>
                {t(b.label)}
              </option>
            ))}
          </select>
        </label>
        <label>
          {t('newgame.difficulty')}
          <select value={difficulty} onChange={(e) => setDifficulty(e.target.value)}>
            <option value="standard">{t('newgame.difficultyStandard')}</option>
            <option value="expert">{t('newgame.difficultyExpert')}</option>
          </select>
        </label>
        <label>
          {t('newgame.players')}
          <select value={playerCount} onChange={(e) => setPlayerCount(Number(e.target.value))}>
            {[1, 2, 3, 4].map((n) => (
              <option key={n} value={n}>
                {n === 1 ? t('newgame.playersOne') : t('newgame.playersMany', n)}
              </option>
            ))}
          </select>
        </label>
        {playerCount === 1 && (
          <label>
            {t('newgame.playerDeck', 1)}
            <select value={deckId} onChange={(e) => setDeckId(e.target.value)}>
              {decks.length === 0 && <option value="">{t('newgame.importFirst')}</option>}
              {decks.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </label>
        )}
        <p className="muted">{t('campaigns.createHint')}</p>
        <button className="primary" disabled={busy || (playerCount === 1 && !deckId)}>
          {t('campaigns.create')}
        </button>
      </form>

      <h3>{t('campaigns.list')}</h3>
      {list.length === 0 && <p className="muted">{t('campaigns.empty')}</p>}
      <ul className="card list">
        {list.map((c) => (
          <li key={c.id}>
            <Link to={`/campaigns/${c.id}`}>
              {c.name} — {t('campaigns.status.' + c.status)} ({t('campaigns.chapter', c.index + 1)})
            </Link>
          </li>
        ))}
      </ul>
    </section>
  )
}
