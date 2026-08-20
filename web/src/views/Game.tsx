import { useCallback, useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { get, post, getToken, GameView } from '../api'
import { CardImage } from '../cards'
import QuestionPanel from '../components/QuestionPanel'
import { lname, useT, useZhMap } from '../i18n'

export default function Game() {
  const { id } = useParams<{ id: string }>()
  const gameId = Number(id)
  const t = useT()
  const zh = useZhMap()
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
  if (!view) return <p className="muted">{t('deck.loading')}</p>

  return (
    <div className="game">
      <header className="game-header">
        <h2>
          {view.name} <span className="muted">· {t('game.round', { n: view.round })}</span>
        </h2>
        <div className="row">
          {view.over ? (
            <span className={view.won ? 'victory' : 'defeat'}>
              {t(view.won ? 'game.victory' : 'game.defeat')} — {view.reason}
            </span>
          ) : view.waitingFor ? (
            <span className="muted">{t('game.waitingFor', { name: view.waitingFor })}</span>
          ) : null}
          <button onClick={undo} disabled={view.over}>
            {t('game.undo')}
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
                <strong>{lname(zh, v.code, v.name)}</strong>{' '}
                <span className="muted">{t('game.stage', { label: v.stageLabel })}</span>
                <div className={`hp ${v.hp <= v.maxHp / 3 ? 'low' : ''}`}>
                  {t('game.hp', { hp: v.hp, max: v.maxHp })}
                </div>
                <div className="muted">
                  {t('game.atkSch', { atk: v.attack, sch: v.scheme })}
                  {v.stunned ? ` · ${t('status.stunned')}` : ''}
                  {v.confused ? ` · ${t('status.confused')}` : ''}
                  {v.tough ? ` · ${t('status.tough')}` : ''}
                </div>
              </div>
            </div>
          ))}
          {view.mainScheme && (
            <div className="entity">
              <CardImage code={view.mainScheme.code} />
              <div className="entity-label">
                <strong>{lname(zh, view.mainScheme.code, view.mainScheme.name)}</strong>
                <div className={`threat ${view.mainScheme.threat >= view.mainScheme.maxThreat - 2 ? 'high' : ''}`}>
                  {t('game.threatMax', { n: view.mainScheme.threat, max: view.mainScheme.maxThreat })}
                </div>
              </div>
            </div>
          )}
          {view.sideSchemes?.map((s) => (
            <div key={s.id} className="entity">
              <CardImage code={s.code} />
              <div className="entity-label">
                <strong>{lname(zh, s.code, s.name)}</strong>
                <div className="threat">{t('game.threat', { n: s.threat })}</div>
                {s.crisis && <div className="crisis">{t('game.crisis')}</div>}
              </div>
            </div>
          ))}
          {view.minions?.map((m) => (
            <div key={m.id} className="entity">
              <CardImage code={m.code} size="sm" />
              <div className="entity-label">
                <strong>{lname(zh, m.code, m.name)}</strong>
                <div className="hp">{t('game.hp', { hp: m.hp, max: m.maxHp })}</div>
                <div className="muted">
                  {t('game.atkSch', { atk: m.attack, sch: m.scheme })}
                  {m.guard ? ` · ${t('status.guard')}` : ''}
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
                {p.firstPlayer && <span className="badge">{t('status.first')}</span>}
                <div className={`hp ${p.hp <= p.maxHp / 3 ? 'low' : ''}`}>
                  {t('game.hp', { hp: p.hp, max: p.maxHp })}
                </div>
                <div className="muted">
                  {t('game.deckCount', { n: p.deckCount })}
                  {p.exhausted ? ` · ${t('status.exhausted')}` : ''}
                  {p.stunned ? ` · ${t('status.stunned')}` : ''}
                  {p.confused ? ` · ${t('status.confused')}` : ''}
                  {p.tough ? ` · ${t('status.tough')}` : ''}
                  {p.encounterDown ? ` · ${t('game.encounterCards', { n: p.encounterDown })}` : ''}
                </div>
              </div>
            </div>
            {p.allies?.map((a) => (
              <div key={a.id} className={`entity ${a.exhausted ? 'exhausted' : ''}`}>
                <CardImage code={a.code} size="xs" />
                <div className="entity-label">
                  {lname(zh, a.code, a.name)}
                  <div className="hp">{t('game.hp', { hp: a.hp, max: a.maxHp })}</div>
                </div>
              </div>
            ))}
            {p.supports?.map((s) => (
              <div key={s.id} className={`entity ${s.exhausted ? 'exhausted' : ''}`}>
                <CardImage code={s.code} size="xs" />
                <div className="entity-label">{lname(zh, s.code, s.name)}</div>
              </div>
            ))}
            {p.upgrades?.map((u) => (
              <div key={u.id} className="entity">
                <CardImage code={u.code} size="xs" />
                <div className="entity-label">{lname(zh, u.code, u.name)}</div>
              </div>
            ))}
          </div>
          {p.hand && (
            <div className="hand row wrap">
              {p.hand.map((c) => (
                <div key={c.id} className="hand-card" title={lname(zh, c.code, c.name)}>
                  <CardImage code={c.code} size="sm" />
                </div>
              ))}
            </div>
          )}
          {!p.hand && p.handSize > 0 && (
            <p className="muted">
              {t('game.hiddenHand', { name: p.name, n: p.handSize })}
            </p>
          )}
        </section>
      ))}

      <section className="question-zone">
        {view.question ? (
          <QuestionPanel question={view.question} onAnswer={answer} />
        ) : view.over ? (
          <p className={view.won ? 'victory' : 'defeat'}>
            {t('game.over')} — {t(view.won ? 'game.victory' : 'game.defeat')}: {view.reason}
          </p>
        ) : (
          <p className="muted">{view.waitingFor ? t('game.waitingFor', { name: view.waitingFor }) : '…'}</p>
        )}
      </section>

      <details className="log">
        <summary>{t('game.logTitle')}</summary>
        <pre>{(view.log ?? []).slice().reverse().join('\n')}</pre>
      </details>
    </div>
  )
}
