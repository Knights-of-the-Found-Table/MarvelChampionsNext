import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { get, GameListItem } from '../api'
import { useT } from '../i18n'

export default function Games() {
  const t = useT()
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

  return (
    <section>
      <h2>{t('games.title')}</h2>
      {error && <p className="error">{error}</p>}
      <p>
        <Link className="button" to="/new">
          {t('games.new')}
        </Link>
      </p>
      {games.length === 0 && <p className="muted">{t('games.empty')}</p>}
      <table className="games-table">
        <thead>
          <tr>
            <th>{t('games.colName')}</th>
            <th>{t('games.colScenario')}</th>
            <th>{t('games.colStatus')}</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {games.map((g) => (
            <tr key={g.id}>
              <td>{g.name}</td>
              <td>{g.scenarioId}</td>
              <td>{t(`status.${g.status}`)}</td>
              <td>
                <Link to={`/games/${g.id}`}>
                  {g.status === 'finished' ? t('games.review') : t('games.open')}
                </Link>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  )
}
