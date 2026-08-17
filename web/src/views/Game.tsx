import { useCallback, useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { get, post, getToken, GameView } from '../api'
import { CardImage } from '../cards'
import QuestionPanel from '../components/QuestionPanel'

export default function Game() {
  const { id } = useParams<{ id: string }>()
  const gameId = Number(id)
  const [view, setView] = useState<GameView | null>(null)
  const [error, setError] = useState('')
  const wsRef = useRef<WebSocket | null>(null)

  const connect = useCallback(() => {
    const token = getToken() ?? ''
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(`${proto}://${location.host}/api/v1/marvel/games/${gameId}/stream?token=${encodeURIComponent(token)}`)
    ws.onmessage = (ev) => {
      try {
        const data = JSON.parse(ev.data)
        if (data.type === 'state') setView(data.view)
      } catch {
        /* ignore */
      }
    }
    ws.onclose = () => {
      // reconnect after a pause (server restart etc.)
      setTimeout(() => {
        if (wsRef.current) connect()
      }, 2000)
    }
    wsRef.current = ws
  }, [gameId])

  useEffect(() => {
    get<GameView>(`/marvel/games/${gameId}`)
      .then(setView)
      .catch((err) => setError(String(err.message)))
    connect()
    return () => {
      const ws = wsRef.current
      wsRef.current = null
      ws?.close()
    }
  }, [gameId, connect])

  async function answer(paths: string[]) {
    try {
      const v = await post<GameView>(`/marvel/games/${gameId}/answer`, { paths })
      setView(v)
    } catch (err) {
      setError(String((err as Error).message))
      setTimeout(() => setError(''), 3000)
      const fresh = await get<GameView>(`/marvel/games/${gameId}`)
      setView(fresh)
    }
  }

  async function undo() {
    try {
      setView(await post<GameView>(`/marvel/games/${gameId}/undo`))
    } catch (err) {
      setError(String((err as Error).message))
      setTimeout(() => setError(''), 3000)
    }
  }

  if (error && !view) return <p className="error">{error}</p>
  if (!view) return <p className="muted">Loading…</p>

  return (
    <div className="game">
      <header className="game-header">
        <h2>
          {view.name} <span className="muted">· Round {view.round}</span>
        </h2>
        <div className="row">
          {view.over ? (
            <span className={view.won ? 'victory' : 'defeat'}>
              {view.won ? '🏆 Victory' : '💀 Defeat'} — {view.reason}
            </span>
          ) : view.waitingFor ? (
            <span className="muted">Waiting for {view.waitingFor}…</span>
          ) : null}
          <button onClick={undo} disabled={view.over}>
            Undo
          </button>
        </div>
      </header>
      {error && <p className="error">{error}</p>}

      <section className="encounter-zone">
        <div className="row wrap">
          {view.villains?.map((v) => (
            <div key={v.id} className="entity">
              <CardImage code={v.code} />
              <div className="entity-label">
                <strong>{v.name}</strong> <span className="muted">stage {v.stageLabel}</span>
                <div className={`hp ${v.hp <= v.maxHp / 3 ? 'low' : ''}`}>
                  {v.hp}/{v.maxHp} HP
                </div>
                <div className="muted">
                  ATK {v.attack} · SCH {v.scheme}
                  {v.stunned ? ' · stunned' : ''}
                  {v.confused ? ' · confused' : ''}
                  {v.tough ? ' · tough' : ''}
                </div>
              </div>
            </div>
          ))}
          {view.mainScheme && (
            <div className="entity">
              <CardImage code={view.mainScheme.code} />
              <div className="entity-label">
                <strong>{view.mainScheme.name}</strong>
                <div className={`threat ${view.mainScheme.threat >= view.mainScheme.maxThreat - 2 ? 'high' : ''}`}>
                  {view.mainScheme.threat}/{view.mainScheme.maxThreat} threat
                </div>
              </div>
            </div>
          )}
          {view.sideSchemes?.map((s) => (
            <div key={s.id} className="entity">
              <CardImage code={s.code} />
              <div className="entity-label">
                <strong>{s.name}</strong>
                <div className="threat">{s.threat} threat</div>
                {s.crisis && <div className="crisis">crisis</div>}
              </div>
            </div>
          ))}
          {view.minions?.map((m) => (
            <div key={m.id} className="entity">
              <CardImage code={m.code} size="sm" />
              <div className="entity-label">
                <strong>{m.name}</strong>
                <div className="hp">
                  {m.hp}/{m.maxHp} HP
                </div>
                <div className="muted">
                  ATK {m.attack} · SCH {m.scheme}
                  {m.guard ? ' · guard' : ''}
                </div>
              </div>
            </div>
          ))}
        </div>
      </section>

      {view.players.map((p) => (
        <section key={p.id} className={`player-zone ${p.exhausted ? 'exhausted' : ''}`}>
          <div className="row wrap player-identity">
            <div className="entity">
              <CardImage code={p.side === 'hero' ? p.heroCode : p.alterEgo} size="sm" />
              <div className="entity-label">
                <strong>{p.name}</strong>
                {p.firstPlayer && <span className="badge">first</span>}
                <div className={`hp ${p.hp <= p.maxHp / 3 ? 'low' : ''}`}>
                  {p.hp}/{p.maxHp} HP
                </div>
                <div className="muted">
                  deck {p.deckCount}
                  {p.exhausted ? ' · exhausted' : ''}
                  {p.stunned ? ' · stunned' : ''}
                  {p.confused ? ' · confused' : ''}
                  {p.tough ? ' · tough' : ''}
                  {p.encounterDown ? ` · ${p.encounterDown} encounter card(s)` : ''}
                </div>
              </div>
            </div>
            {p.allies?.map((a) => (
              <div key={a.id} className={`entity ${a.exhausted ? 'exhausted' : ''}`}>
                <CardImage code={a.code} size="xs" />
                <div className="entity-label">
                  {a.name}
                  <div className="hp">
                    {a.hp}/{a.maxHp}
                  </div>
                </div>
              </div>
            ))}
            {p.supports?.map((s) => (
              <div key={s.id} className={`entity ${s.exhausted ? 'exhausted' : ''}`}>
                <CardImage code={s.code} size="xs" />
                <div className="entity-label">{s.name}</div>
              </div>
            ))}
            {p.upgrades?.map((u) => (
              <div key={u.id} className="entity">
                <CardImage code={u.code} size="xs" />
                <div className="entity-label">{u.name}</div>
              </div>
            ))}
          </div>
          {p.hand && (
            <div className="hand row wrap">
              {p.hand.map((c) => (
                <div key={c.id} className="hand-card" title={c.name}>
                  <CardImage code={c.code} size="sm" />
                </div>
              ))}
            </div>
          )}
          {!p.hand && p.handSize > 0 && (
            <p className="muted">
              {p.name}: {p.handSize} cards in hand (hidden)
            </p>
          )}
        </section>
      ))}

      <section className="question-zone">
        {view.question ? (
          <QuestionPanel question={view.question} onAnswer={answer} />
        ) : view.over ? (
          <p className={view.won ? 'victory' : 'defeat'}>
            Game over — {view.won ? 'victory' : 'defeat'}: {view.reason}
          </p>
        ) : (
          <p className="muted">{view.waitingFor ? `Waiting for ${view.waitingFor}…` : '…'}</p>
        )}
      </section>

      <details className="log">
        <summary>Game log</summary>
        <pre>{(view.log ?? []).slice().reverse().join('\n')}</pre>
      </details>
    </div>
  )
}
