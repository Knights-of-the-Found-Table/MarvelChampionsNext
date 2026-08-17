import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { get, post, Deck, ScenarioInfo, GameView } from '../api'

export default function NewGame() {
  const navigate = useNavigate()
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
      <h2>New game</h2>
      {error && <p className="error">{error}</p>}
      <form className="card form" onSubmit={start}>
        <label>
          Players
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
                {n} player{n > 1 ? 's' : ''}
              </option>
            ))}
          </select>
        </label>
        {Array.from({ length: playerCount }).map((_, i) => (
          <label key={i}>
            Player {i + 1} deck
            <select
              value={deckIds[i] ?? ''}
              onChange={(e) =>
                setDeckIds((ids) => ids.map((x, j) => (j === i ? Number(e.target.value) : x)))
              }
            >
              {decks.length === 0 && <option value="">-- import a deck first --</option>}
              {decks.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </label>
        ))}
        <label>
          Scenario
          <select value={scenarioId} onChange={(e) => setScenarioId(e.target.value)}>
            {scenarios.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Difficulty
          <select value={difficulty} onChange={(e) => setDifficulty(e.target.value)}>
            <option value="standard">Standard</option>
            <option value="expert">Expert</option>
          </select>
        </label>
        <label>
          Game name (optional)
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Scenario name by default" />
        </label>
        <button type="submit" disabled={busy || deckIds.slice(0, playerCount).some((x) => x === null) || !scenarioId}>
          {busy ? 'Creating…' : 'Create game'}
        </button>
      </form>
    </section>
  )
}
