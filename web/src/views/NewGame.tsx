import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { get, post, Deck, ScenarioInfo, GameView } from '../api'
import { useT } from '../i18n'

export default function NewGame() {
  const navigate = useNavigate()
  const t = useT()
  const [decks, setDecks] = useState<Deck[]>([])
  const [scenarios, setScenarios] = useState<ScenarioInfo[]>([])
  const [playerCount, setPlayerCount] = useState(1)
  const [deckIds, setDeckIds] = useState<(number | null)[]>([null, null, null, null])
  const [scenarioId, setScenarioId] = useState('')
  const [difficulty, setDifficulty] = useState('standard')
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    get<Deck[]>('/marvel/decks').then((d) => {
      setDecks(d)
      if (d.length > 0) setDeckIds((ids) => ids.map((x, i) => (i === 0 && x === null ? d[0].id : x)))
    })
    get<ScenarioInfo[]>('/marvel/scenarios').then((s) => {
      setScenarios(s)
      if (s.length > 0) setScenarioId(s[0].id)
    })
  }, [])

  async function start(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      const chosen = deckIds.slice(0, playerCount).filter((x): x is number => x !== null)
      const view = await post<GameView>('/marvel/games', {
        deckIds: chosen,
        scenarioId,
        difficulty,
        name: name || undefined,
      })
      navigate(`/games/${view.id}`)
    } catch (err) {
      setError(String((err as Error).message))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section>
      <h2>{t('newgame.title')}</h2>
      {error && <p className="error">{error}</p>}
      <form className="card form" onSubmit={start}>
        <label>
          {t('newgame.players')}
          <select
            value={playerCount}
            onChange={(e) => {
              const n = Number(e.target.value)
              setPlayerCount(n)
              setDeckIds((ids) => ids.map((x, i) => (i < n && x === null && decks.length > 0 ? decks[0].id : x)))
            }}
          >
            {[1, 2, 3, 4].map((n) => (
              <option key={n} value={n}>
                {n === 1 ? t('newgame.playersOne') : t('newgame.playersMany', { n })}
              </option>
            ))}
          </select>
        </label>
        {Array.from({ length: playerCount }).map((_, i) => (
          <label key={i}>
            {t('newgame.playerDeck', { n: i + 1 })}
            <select
              value={deckIds[i] ?? ''}
              onChange={(e) =>
                setDeckIds((ids) => ids.map((x, j) => (j === i ? Number(e.target.value) : x)))
              }
            >
              {decks.length === 0 && <option value="">{t('newgame.importFirst')}</option>}
              {decks.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </label>
        ))}
        <label>
          {t('newgame.scenario')}
          <select value={scenarioId} onChange={(e) => setScenarioId(e.target.value)}>
            {scenarios.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
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
          {t('newgame.gameName')}
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder={t('newgame.gameNamePlaceholder')} />
        </label>
        <button type="submit" disabled={busy || deckIds.slice(0, playerCount).some((x) => x === null) || !scenarioId}>
          {busy ? t('newgame.creating') : t('newgame.create')}
        </button>
      </form>
    </section>
  )
}
