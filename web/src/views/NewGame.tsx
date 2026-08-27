import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { get, post, Deck, ScenarioInfo } from '../api'
import { errorText } from '../deckIssues'
import { useT, useZhMap } from '../i18n'

// 默认选中的牌组：优先合法牌组，全部不合法时退回第一个。
function defaultDeck(decks: Deck[]): string | null {
  return (decks.find((d) => d.valid !== false) ?? decks[0])?.id ?? null
}

export default function NewGame() {
  const navigate = useNavigate()
  const t = useT()
  const zh = useZhMap()
  const [decks, setDecks] = useState<Deck[]>([])
  const [scenarios, setScenarios] = useState<ScenarioInfo[]>([])
  const [playerCount, setPlayerCount] = useState(1)
  const [deckIds, setDeckIds] = useState<(string | null)[]>([null, null, null, null])
  const [scenarioId, setScenarioId] = useState('')
  const [difficulty, setDifficulty] = useState('standard')
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    get<Deck[]>('/marvel/decks').then((d) => {
      setDecks(d)
      if (d.length > 0) setDeckIds((ids) => ids.map((x, i) => (i === 0 && x === null ? defaultDeck(d) : x)))
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
      const chosen = deckIds.slice(0, playerCount).filter((x): x is string => x !== null)
      const view = await post<{ id: string }>('/marvel/games', {
        deckIds: playerCount === 1 ? chosen : [],
        playerCount,
        scenarioId,
        difficulty,
        name: name || undefined,
      })
      navigate(`/games/${view.id}`)
    } catch (err) {
      setError(errorText(t, zh, err))
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
              setDeckIds((ids) => ids.map((x, i) => (i < n && x === null && decks.length > 0 ? defaultDeck(decks) : x)))
            }}
          >
            {[1, 2, 3, 4].map((n) => (
              <option key={n} value={n}>
                {n === 1 ? t('newgame.playersOne') : t('newgame.playersMany', n)}
              </option>
            ))}
          </select>
        </label>
        {playerCount === 1 ? (
          <label>
            {t('newgame.playerDeck', 1)}
          <select
            value={deckIds[0] ?? ''}
            onChange={(e) =>
              setDeckIds((ids) => ids.map((x, j) => (j === 0 ? e.target.value : x)))
            }
          >
            {decks.length === 0 && <option value="">{t('newgame.importFirst')}</option>}
            {decks.map((d) => (
              <option key={d.id} value={d.id} disabled={d.valid === false}>
                {d.name}
                {d.valid === false ? ` ⚠ ${t('decks.invalid')}` : ''}
              </option>
            ))}
          </select>
          </label>
        ) : (
          <p className="muted">{t('newgame.lobbyHint')}</p>
        )}
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
        <button
          type="submit"
          disabled={
            busy ||
            deckIds.slice(0, playerCount).some((x) => x === null) ||
            !scenarioId ||
            deckIds
              .slice(0, playerCount)
              .some((x) => decks.find((d) => d.id === x)?.valid === false)
          }
        >
          {busy ? t('newgame.creating') : t('newgame.create')}
        </button>
      </form>
    </section>
  )
}
