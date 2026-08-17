import { useEffect, useState } from 'react'
import { get, post, del, Deck } from '../api'
import { CardImage } from '../cards'

export default function Decks() {
  const [decks, setDecks] = useState<Deck[]>([])
  const [url, setUrl] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function refresh() {
    try {
      setDecks(await get<Deck[]>('/marvel/decks'))
    } catch (err) {
      setError(String((err as Error).message))
    }
  }

  useEffect(() => {
    refresh()
  }, [])

  async function importDeck(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      await post('/marvel/decks', { url })
      setUrl('')
      await refresh()
    } catch (err) {
      setError(String((err as Error).message))
    } finally {
      setBusy(false)
    }
  }

  async function remove(id: number) {
    try {
      await del(`/marvel/decks/${id}`)
      await refresh()
    } catch (err) {
      setError(String((err as Error).message))
    }
  }

  return (
    <section>
      <h2>Your decks</h2>
      {error && <p className="error">{error}</p>}
      <form className="row" onSubmit={importDeck}>
        <input
          style={{ flex: 1 }}
          placeholder="https://marvelcdb.com/decklist/view/12345/deck-name"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
        />
        <button type="submit" disabled={busy || !url}>
          Import from marvelcdb
        </button>
      </form>
      {decks.length === 0 && <p className="muted">No decks yet — import one from marvelcdb.</p>}
      <div className="deck-list">
        {decks.map((d) => (
          <div key={d.id} className="deck card row">
            <CardImage code={d.investigatorCode} size="sm" />
            <div style={{ flex: 1 }}>
              <strong>{d.name}</strong>
              <div className="muted">
                {Object.keys(d.slots).length} card types · {Object.values(d.slots).reduce((a, b) => a + b, 0)} cards
              </div>
            </div>
            <button className="danger" onClick={() => remove(d.id)}>
              Delete
            </button>
          </div>
        ))}
      </div>
    </section>
  )
}
