import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  buyMarket,
  campaignChoice,
  Deck,
  getCampaign,
  get,
  joinCampaign,
  kickCampaign,
  playCampaign,
  setCampaignHeal,
  startCampaign,
} from '../api'
import type { CampaignDetail as CampaignDetailData } from '../api'
import { lname, useT, useZhMap } from '../i18n'

function statusKey(status: string) {
  return 'campaigns.status.' + status
}

export default function CampaignDetail() {
  const { id = '' } = useParams()
  const t = useT()
  const zh = useZhMap()
  const navigate = useNavigate()
  const [detail, setDetail] = useState<CampaignDetailData | null>(null)
  const [decks, setDecks] = useState<Deck[]>([])
  const [deckId, setDeckId] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const reload = useCallback(() => {
    getCampaign(id).then(setDetail).catch((err) => setError(err instanceof Error ? err.message : String(err)))
  }, [id])

  useEffect(() => {
    reload()
    get<Deck[]>('/marvel/decks').then((d) => {
      setDecks(d)
      const first = d.find((x) => x.valid !== false) ?? d[0]
      if (first) setDeckId(first.id)
    })
  }, [reload])

  async function act(fn: () => Promise<CampaignDetailData | { gameId: string }>) {
    setError('')
    setBusy(true)
    try {
      const res = await fn()
      if ('gameId' in res) {
        navigate(`/games/${res.gameId}?campaign=${id}`)
        return
      }
      setDetail(res)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  if (!detail) {
    return (
      <section>
        <h2>{t('campaigns.title')}</h2>
        {error && <p className="error">{error}</p>}
      </section>
    )
  }

  const st = detail.state
  const mySlot = detail.yourSlot
  const myPending = mySlot >= 0 && st.pendingChoices ? st.pendingChoices[String(mySlot)] : undefined
  const hasPending = st.pendingChoices && Object.keys(st.pendingChoices).length > 0
  const name = (code: string) => lname(zh, code, detail.names[code] ?? code)
  const latestGame = detail.games[0]

  return (
    <section>
      <h2>
        {detail.name} · {t('newgame.difficulty' + (detail.difficulty === 'expert' ? 'Expert' : 'Standard'))}
      </h2>
      {detail.desc && <p className="muted">{detail.desc}</p>}
      <p>
        {t(statusKey(detail.status))}
        {st.lastResult && ` — ${t(st.lastResult === 'won' ? 'campaigns.lastWon' : 'campaigns.lastLost')}`}
      </p>
      {error && <p className="error">{error}</p>}

      <ol className="card chapters">
        {detail.chapters.map((ch, i) => (
          <li key={ch.id} className={i === detail.index ? 'current' : i < detail.index ? 'done' : ''}>
            {i + 1}. {ch.name}
          </li>
        ))}
      </ol>

      {detail.status === 'forming' && (
        <div className="card">
          <h3>{t('campaigns.seats')}</h3>
          <ul className="list">
            {Array.from({ length: detail.playerCount }, (_, i) => {
              const seat = detail.seats.find((x) => x.slot === i)
              return (
                <li key={i}>
                  {seat
                    ? `${seat.username} — ${seat.deckName}`
                    : t('campaigns.openSeat')}
                </li>
              )
            })}
          </ul>
          {mySlot < 0 && detail.status === 'forming' && (
            <div className="form inline">
              <select value={deckId} onChange={(e) => setDeckId(e.target.value)}>
                {decks.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.name}
                  </option>
                ))}
              </select>
              <button disabled={busy || !deckId} onClick={() => act(() => joinCampaign(id, deckId))}>
                {t('campaigns.join')}
              </button>
            </div>
          )}
          {detail.host && (
            <div className="form inline">
              {detail.seats
                .filter((x) => x.slot > 0)
                .map((x) => (
                  <button key={x.slot} disabled={busy} onClick={() => act(() => kickCampaign(id, x.slot))}>
                    {t('campaigns.kick')} {x.username}
                  </button>
                ))}
              <button className="primary" disabled={busy || detail.seats.length !== detail.playerCount} onClick={() => act(() => startCampaign(id))}>
                {t('campaigns.start')}
              </button>
            </div>
          )}
        </div>
      )}

      {detail.status === 'interlude' && (
        <div className="card">
          <h3>{t('campaigns.interlude')}</h3>
          {myPending && detail.pools && (
            <div>
              <p>{t('campaigns.choice.' + myPending)}</p>
              {myPending === 'improve' ? (
                st.players
                  .filter((p) => p.slot === mySlot)
                  .map((p) => {
                    const imp = p.condition ? p.condition.slice(0, -1) + 'b' : ''
                    return (
                      <div key={p.slot} className="form inline">
                        <button disabled={busy || !imp} onClick={() => act(() => campaignChoice(id, imp))}>
                          {imp ? name(imp) : t('campaigns.noCondition')}
                        </button>
                        <button disabled={busy} onClick={() => act(() => campaignChoice(id, ''))}>
                          {t('campaigns.skip')}
                        </button>
                      </div>
                    )
                  })
              ) : (
                <div className="form inline">
                  {(myPending === 'tech' ? detail.pools.tech : detail.pools.condition).map((code) => (
                    <button key={code} disabled={busy} onClick={() => act(() => campaignChoice(id, code))}>
                      {name(code)}
                    </button>
                  ))}
                  {myPending === 'condition' && (
                    <button disabled={busy} onClick={() => act(() => campaignChoice(id, ''))}>
                      {t('campaigns.skip')}
                    </button>
                  )}
                </div>
              )}
            </div>
          )}
          {hasPending && !myPending && <p className="muted">{t('campaigns.waitingChoices')}</p>}
          {!hasPending && detail.host && (
            <button className="primary" disabled={busy} onClick={() => act(() => playCampaign(id))}>
              {t('campaigns.play')} — {detail.chapters[detail.index]?.name}
            </button>
          )}
          {!hasPending && !detail.host && <p className="muted">{t('campaigns.hostPlays')}</p>}
          {detail.difficulty === 'expert' && mySlot >= 0 && (
            <p className="form inline">
              <label>
                <input
                  type="checkbox"
                  checked={!!st.players.find((p) => p.slot === mySlot)?.healNext}
                  onChange={(e) => act(() => setCampaignHeal(id, e.target.checked))}
                />
                {t('campaigns.heal')}
              </label>
            </p>
          )}
          {detail.box === 'gmw' && mySlot >= 0 && (
            <div>
              <h4>{t('campaigns.market')}</h4>
              <ul className="list">
                {detail.market.map((mc) => {
                  const bought = st.players.some((p) => p.market?.includes(mc.code))
                  return (
                    <li key={mc.code}>
                      {name(mc.code)} ({t('campaigns.unitCost', mc.cost)}){' '}
                      <button
                        disabled={busy || bought || (st.players.find((p) => p.slot === mySlot)?.units ?? 0) < mc.cost}
                        onClick={() => act(() => buyMarket(id, mc.code))}
                      >
                        {bought ? t('campaigns.bought') : t('campaigns.buy')}
                      </button>
                    </li>
                  )
                })}
              </ul>
            </div>
          )}
        </div>
      )}

      {detail.status === 'active' && latestGame && (
        <div className="card">
          <h3>{t('campaigns.chapterInProgress')}</h3>
          <Link to={`/games/${latestGame.id}?campaign=${id}`}>{latestGame.name}</Link>
        </div>
      )}

      <div className="card">
        <h3>{t('campaigns.log')}</h3>
        <table className="campaign-log">
          <tbody>
            {st.players.map((p) => (
              <tr key={p.slot}>
                <th>{p.name}</th>
                <td>
                  {detail.difficulty === 'expert' && (
                    <span>
                      {t('campaigns.hp')}: {p.hp || '—'}{' '}
                    </span>
                  )}
                  {detail.box === 'gmw' && (
                    <span>
                      {t('campaigns.units')}: {p.units ?? 0}{' '}
                    </span>
                  )}
                  {p.tech && (
                    <span>
                      {t('campaigns.tech')}: {name(p.tech)}{' '}
                    </span>
                  )}
                  {p.condition && (
                    <span>
                      {t('campaigns.condition')}: {name(p.improved && p.condition ? p.condition.slice(0, -1) + 'b' : p.condition)}{' '}
                    </span>
                  )}
                  {p.allies && p.allies.length > 0 && (
                    <span>
                      {t('campaigns.allies')}: {p.allies.map(name).join(', ')}{' '}
                    </span>
                  )}
                  {p.market && p.market.length > 0 && (
                    <span>
                      {t('campaigns.marketCards')}: {p.market.map(name).join(', ')}
                    </span>
                  )}
                  {p.engagedEnemy && <span>{t('campaigns.engaged')}</span>}
                </td>
              </tr>
            ))}
            {st.box === 'rrs' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {st.experimental && st.experimental.length > 0 && (
                    <div>
                      {t('campaigns.experimental')}: {st.experimental.map(name).join(', ')}
                    </div>
                  )}
                  {(st.delayCounters ?? 0) > 0 && (
                    <div>
                      {t('campaigns.delay')}: {st.delayCounters}
                    </div>
                  )}
                  {st.removedAllies && st.removedAllies.length > 0 && (
                    <div>
                      {t('campaigns.removedAllies')}: {st.removedAllies.map(name).join(', ')}
                    </div>
                  )}
                </td>
              </tr>
            )}
            {st.box === 'gmw' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {st.collection && st.collection.length > 0 && (
                    <div>
                      {t('campaigns.collection')}: {st.collection.map(name).join(', ')}
                    </div>
                  )}
                  {st.artifacts && st.artifacts.length > 0 && (
                    <div>
                      {t('campaigns.artifacts')}: {st.artifacts.map(name).join(', ')}
                    </div>
                  )}
                  <div>
                    {t('campaigns.headhunter')}: {st.headhunter?.filter(Boolean).length ?? 0} / 4
                  </div>
                  {(st.powerStone ?? -1) >= 0 && (
                    <div>
                      {t('campaigns.powerStone')}: {st.players.find((p) => p.slot === st.powerStone)?.name}
                    </div>
                  )}
                  <div>
                    {t('campaigns.evasion')}: {st.evasion ?? 0}
                  </div>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <p>
        <Link to="/campaigns">← {t('campaigns.back')}</Link>
      </p>
    </section>
  )
}
