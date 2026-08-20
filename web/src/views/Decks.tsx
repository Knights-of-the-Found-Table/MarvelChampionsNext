import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { get, post, del, Deck } from '../api'
import { CardImage } from '../cards'
import { useT } from '../i18n'

export default function Decks() {
  const navigate = useNavigate()
  const t = useT()
  const [decks, setDecks] = useState<Deck[]>([])
  const [url, setUrl] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

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

  async function importFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = '' // allow re-selecting the same file
    if (!file) return
    setError('')
    setBusy(true)
    try {
      const text = await file.text()
      await post('/marvel/decks', { text })
      await refresh()
    } catch (err) {
      setError(String((err as Error).message))
    } finally {
      setBusy(false)
    }
  }

  async function remove(e: React.MouseEvent, id: number) {
    e.stopPropagation()
    try {
      await del(`/marvel/decks/${id}`)
      await refresh()
    } catch (err) {
      setError(String((err as Error).message))
    }
  }

  return (
    <section>
      <h2>{t('decks.title')}</h2>
      {error && <p className="error">{error}</p>}
      <form className="row" onSubmit={importDeck}>
        <input
          style={{ flex: 1 }}
          placeholder="https://marvelcdb.com/decklist/view/12345/deck-name"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
        />
        <button type="submit" disabled={busy || !url}>
          {t('decks.importUrl')}
        </button>
        <button type="button" disabled={busy} onClick={() => fileRef.current?.click()}>
          {t('decks.importFile')}
        </button>
        <input
          ref={fileRef}
          type="file"
          accept=".txt,.text,text/plain"
          style={{ display: 'none' }}
          onChange={importFile}
        />
      </form>
      {decks.length === 0 && <p className="muted">{t('decks.empty')}</p>}
      <div className="deck-list">
        {decks.map((d) => (
          <div
            key={d.id}
            className="deck card row clickable"
            onClick={() => navigate(`/decks/${d.id}`)}
          >
            <CardImage code={d.investigatorCode} size="sm" />
            <div style={{ flex: 1 }}>
              <strong>{d.name}</strong>
              <div className="muted">
                {t('decks.cardStats', {
                  types: Object.keys(d.slots).length,
                  total: Object.values(d.slots).reduce((a, b) => a + b, 0),
                })}
              </div>
            </div>
            <button className="danger" onClick={(e) => remove(e, d.id)}>
              {t('decks.delete')}
            </button>
          </div>
        ))}
      </div>
    </section>
  )
}
