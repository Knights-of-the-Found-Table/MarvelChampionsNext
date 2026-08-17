import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { get, post, GameListItem } from '../api'

export default function Games() {
  const navigate = useNavigate()
  const [games, setGames] = useState<GameListItem[]>([])
  const [error, setError] = useState('')

  async function refresh() {
    try {
      setGames(await get<GameListItem[]>('/marvel/games'))
    } catch (err) {
      setError(String((err as Error).message))
    }
  }

  useEffect(() => {
    refresh()
  }, [])

  async function join(id: number) {
    try {
      await post(`/marvel/games/${id}/join`, {})
      navigate(`/games/${id}`)
    } catch (err) {
      setError(String((err as Error).message))
    }
  }

  return (
    <section>
      <h2>Games</h2>
      {error && <p className="error">{error}</p>}
      <p>
        <Link className="button" to="/new">
          + New game
        </Link>
      </p>
      {games.length === 0 && <p className="muted">No games yet.</p>}
      <table className="games-table">
        <thead>
          <tr>
            <th>#</th>
            <th>Name</th>
            <th>Scenario</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {games.map((g) => (
            <tr key={g.id}>
              <td>{g.id}</td>
              <td>{g.name}</td>
              <td>{g.scenarioId}</td>
              <td>{g.status}</td>
              <td>
                <Link to={`/games/${g.id}`}>{g.status === 'finished' ? 'Review' : g.status === 'lobby' ? 'Spectate' : 'Open'}</Link>{' '}
                {g.status === 'lobby' && (
                  <button onClick={() => join(g.id)}>Join</button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  )
}
