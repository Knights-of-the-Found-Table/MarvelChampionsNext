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
  swapCampaignDeck,
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
  const myPendingKinds =
    mySlot >= 0 && st.pendingChoices
      ? Object.entries(st.pendingChoices)
          .filter(([k]) => k.startsWith(mySlot + ':'))
          .map(([, v]) => v)
      : []
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
            {ch.requires && <span className="muted"> — {t('campaigns.flag.requires')} {ch.requires}</span>}
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
          {myPendingKinds.map((myPending) => (
            <div key={myPending}>
              <p>{t('campaigns.choice.' + myPending)}</p>
              {myPending === 'sm-tech' ? (
                <div className="form inline">
                  {(st.players.find((p) => p.slot === mySlot)?.smTechOffer ?? []).map((code) => (
                    <button key={code} disabled={busy} onClick={() => act(() => campaignChoice(id, code))}>
                      {name(code)}
                    </button>
                  ))}
                </div>
              ) : myPending === 'sm-aspect' ? (
                <SMCodePick
                  placeholder={t('campaigns.aspectHint')}
                  deck={st.players.find((p) => p.slot === mySlot)?.deck ?? {}}
                  busy={busy}
                  onPick={(code) => act(() => campaignChoice(id, code))}
                />
              ) : myPending === 'sm-plan' ? (
                <div className="form inline">
                  <select
                    defaultValue=""
                    onChange={(e) => {
                      const code = e.target.value
                      if (code) act(() => campaignChoice(id, code))
                    }}
                  >
                    <option value="">{t('campaigns.pickCard')}</option>
                    {Object.entries(st.players.find((p) => p.slot === mySlot)?.deck ?? {}).map(([code, n]) => (
                      <option key={code} value={code}>
                        {name(code)} ×{n}
                      </option>
                    ))}
                  </select>
                </div>
              ) : myPending === 'mg-role' ? (
                <div className="form inline">
                  {detail.pools.roles.map((role) => (
                    <button key={role} disabled={busy} onClick={() => act(() => campaignChoice(id, role))}>
                      {t('campaigns.role.' + role)}
                    </button>
                  ))}
                </div>
              ) : myPending === 'nx-scheme' ? (
                <div className="form inline">
                  {detail.pools.nx.map((code) => (
                    <button key={code} disabled={busy} onClick={() => act(() => campaignChoice(id, code))}>
                      {name(code)}
                    </button>
                  ))}
                </div>
              ) : myPending === 'aos-accuse' ? (
                <div className="form inline">
                  {detail.pools.aosMembers.map((code) => (
                    <button key={code} disabled={busy} onClick={() => act(() => campaignChoice(id, code))}>
                      {name(code)}
                    </button>
                  ))}
                </div>
              ) : myPending === 'wi-trait' ? (
                <div className="form inline">
                  {(detail.pools.traits ?? []).map((tr) => (
                    <button key={tr} disabled={busy} onClick={() => act(() => campaignChoice(id, tr, 'wi-trait'))}>
                      {t('campaigns.wiTrait.' + tr)}
                    </button>
                  ))}
                </div>
              ) : myPending === 'wi-ally' ? (
                <SMCodePick
                  placeholder="27010"
                  deck={st.players.find((p) => p.slot === mySlot)?.deck ?? {}}
                  busy={busy}
                  onPick={(code) => act(() => campaignChoice(id, code, 'wi-ally'))}
                />
              ) : myPending === 'wi-card' ? (
                <SMCodePick
                  placeholder="27010"
                  deck={st.players.find((p) => p.slot === mySlot)?.deck ?? {}}
                  busy={busy}
                  onPick={(code) => act(() => campaignChoice(id, code, 'wi-card'))}
                />
              ) : myPending === 'aw-ally' ? (
                <SMCodePick
                  placeholder="22001"
                  deck={st.players.find((p) => p.slot === mySlot)?.deck ?? {}}
                  busy={busy}
                  onPick={(code) => act(() => campaignChoice(id, code, 'aw-ally'))}
                />
              ) : myPending === 'aw-identity' ? (
                <DeckPick busy={busy} deck={st.players.find((p) => p.slot === mySlot)?.deck ?? {}} name={name} t={t} onPick={(code) => act(() => campaignChoice(id, code, 'aw-identity'))} />
              ) : myPending === 'mojo-role' ? (
                <div className="form inline">
                  {Object.keys((detail.tables?.mojoRoles as Record<string, unknown>) ?? {}).map((role) => (
                    <button key={role} disabled={busy} onClick={() => act(() => campaignChoice(id, role, 'mojo-role'))}>
                      {t('campaigns.role.' + role)}
                    </button>
                  ))}
                </div>
              ) : myPending === 'mojo-training' || myPending === 'mojo-scheme' ? (
                <div className="form inline">
                  {(detail.pools.allNx ?? []).map((code) => (
                    <button key={code} disabled={busy} onClick={() => act(() => campaignChoice(id, code, myPending))}>
                      {name(code)}
                    </button>
                  ))}
                </div>
              ) : myPending === 'mojo-event' ? (
                <DeckPick busy={busy} deck={st.players.find((p) => p.slot === mySlot)?.deck ?? {}} name={name} t={t} onPick={(code) => act(() => campaignChoice(id, code, 'mojo-event'))} />
              ) : myPending === 'mojo-market' ? (
                <div className="form inline">
                  <button disabled={busy} onClick={() => act(() => campaignChoice(id, '21183', 'mojo-market'))}>
                    {t('campaigns.shawarma')}
                  </button>
                  {(() => {
                    const role = st.players.find((p) => p.slot === mySlot)?.mojoRole
                    const table = (detail.tables?.mojoRoles as Record<string, { market?: string }> | undefined) ?? {}
                    const market = role ? table[role]?.market : undefined
                    return market ? (
                      <button disabled={busy} onClick={() => act(() => campaignChoice(id, market, 'mojo-market'))}>
                        {name(market)}
                      </button>
                    ) : null
                  })()}
                </div>
              ) : myPending === 'bord-path' ? (
                <div className="form inline">
                  {((detail.tables?.paths as Array<{ key: string; label: string }> | undefined) ?? []).map((pth) => (
                    <button key={pth.key} disabled={busy} onClick={() => act(() => campaignChoice(id, pth.key, 'bord-path'))}>
                      {t('campaigns.path.' + pth.key)}
                    </button>
                  ))}
                </div>
              ) : myPending === 'bord-gear' ? (
                <DeckPick busy={busy} deck={st.players.find((p) => p.slot === mySlot)?.deck ?? {}} name={name} t={t} onPick={(code) => act(() => campaignChoice(id, code, 'bord-gear'))} />
              ) : myPending === 'nt-meta' || myPending === 'nt-team' ? (
                <div className="form inline">
                  <button disabled={busy} onClick={() => act(() => campaignChoice(id, 'yes', myPending))}>
                    {t('campaigns.yes')}
                  </button>
                  <button disabled={busy} onClick={() => act(() => campaignChoice(id, '', myPending))}>
                    {t('campaigns.no')}
                  </button>
                </div>
              ) : myPending === 'wa-sight' || myPending === 'wa-portal' || myPending === 'wa-intervened' ? (
                <div className="form inline">
                  <DeckPick busy={busy} deck={st.players.find((p) => p.slot === mySlot)?.deck ?? {}} name={name} t={t} onPick={(code) => act(() => campaignChoice(id, code, myPending))} />
                  <button disabled={busy} onClick={() => act(() => campaignChoice(id, '', myPending))}>
                    {t('campaigns.skip')}
                  </button>
                </div>
              ) : myPending === 'viral-next' ? (
                <div className="form inline">
                  {(detail.pools.viralNext ?? []).map((idx) => (
                    <button key={idx} disabled={busy} onClick={() => act(() => campaignChoice(id, idx, 'viral-next'))}>
                      {t('campaigns.viralNext.' + idx)}
                    </button>
                  ))}
                </div>
              ) : myPending === 'en-path1' || myPending === 'en-path2' || myPending === 'en-path3' ? (
                <div className="form inline">
                  {(() => {
                    const key = myPending.slice(-1)
                    const opts = ((detail.tables?.enPaths as Record<string, string[]> | undefined) ?? {})[key] ?? []
                    return opts.map((opt) => (
                      <button key={opt} disabled={busy} onClick={() => act(() => campaignChoice(id, opt.endsWith('a') ? 'a' : 'b', myPending))}>
                        {t('campaigns.choice.' + opt)}
                      </button>
                    ))
                  })()}
                </div>
              ) : myPending === 'improve' ? (
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
          ))}

          {hasPending && myPendingKinds.length === 0 && <p className="muted">{t('campaigns.waitingChoices')}</p>}
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
          {mySlot >= 0 && detail.status === 'interlude' && (
            <div className="form inline">
              <select value={deckId} onChange={(e) => setDeckId(e.target.value)}>
                {decks.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.name}
                  </option>
                ))}
              </select>
              <button disabled={busy || !deckId} onClick={() => act(() => swapCampaignDeck(id, deckId))}>
                {t('campaigns.swapDeck')}
              </button>
            </div>
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
                  {p.mgRole && (
                    <span>
                      {t('campaigns.role')}:{' '}{t('campaigns.role.' + p.mgRole)}{' '}
                    </span>
                  )}
                  {p.smTech && (
                    <span>
                      {t('campaigns.smTechLabel')}: {name(p.smTech)}{' '}
                    </span>
                  )}
                  {p.wiTrait && (
                    <span>
                      {t('campaigns.wiTraitLabel')}: {t('campaigns.wiTrait.' + p.wiTrait)}{' '}
                    </span>
                  )}
                  {p.wiAllies && p.wiAllies.length > 0 && (
                    <span>
                      {t('campaigns.wiAllies')}: {p.wiAllies.map(name).join(', ')}{' '}
                    </span>
                  )}
                  {p.wiRewards && p.wiRewards.length > 0 && (
                    <span>
                      {t('campaigns.wiRewards')}: {p.wiRewards.map(name).join(', ')}{' '}
                    </span>
                  )}
                  {detail.box === 'awesome' && (
                    <span>
                      {t('campaigns.influence')}: {p.influence ?? 0}{' '}
                    </span>
                  )}
                  {p.awAlly && (
                    <span>
                      {t('campaigns.awAlly')}: {name(p.awAlly)}{' '}
                    </span>
                  )}
                  {p.mojoRole && (
                    <span>
                      {t('campaigns.role')}:{' '}{t('campaigns.role.' + p.mojoRole)}{' '}
                    </span>
                  )}
                  {p.mojoMarket && (
                    <span>
                      {t('campaigns.mojoMarketLabel')}: {name(p.mojoMarket)}{' '}
                    </span>
                  )}
                  {p.mojoScheme && (
                    <span>
                      {t('campaigns.mojoScheme')}: {name(p.mojoScheme)}{' '}
                    </span>
                  )}
                  {p.mojoEvent && (
                    <span>
                      {t('campaigns.mojoEvent')}: {name(p.mojoEvent)}{' '}
                    </span>
                  )}
                  {p.bordObligations && p.bordObligations.length > 0 && (
                    <span>
                      {t('campaigns.bordObligations')}: {p.bordObligations.map(name).join(', ')}{' '}
                    </span>
                  )}
                  {p.bordGear && p.bordGear.length > 0 && (
                    <span>
                      {t('campaigns.bordGear')}: {p.bordGear.map(name).join(', ')}
                    </span>
                  )}
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
            {st.box === 'mts' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {st.pool && st.pool.length > 0 && (
                    <div>
                      {t('campaigns.pool')}: {st.pool.map(name).join(', ')}
                    </div>
                  )}
                  {st.flags?.towerDamaged && <div>{t('campaigns.towerDamaged')}</div>}
                  {st.flags?.stones && <div>{t('campaigns.stonesCompleted')}</div>}
                </td>
              </tr>
            )}
            {st.box === 'sm' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  <div>
                    {t('campaigns.reputation')}: {st.counters?.marked ?? 0}
                  </div>
                  {st.smOsborn && st.smOsborn.length > 0 && (
                    <div>
                      {t('campaigns.osborn')}: {st.smOsborn.map(name).join(', ')}
                    </div>
                  )}
                  {st.smCommunity && st.smCommunity.length > 0 && (
                    <div>
                      {t('campaigns.community')}: {st.smCommunity.map(name).join(', ')}
                    </div>
                  )}
                  {(st.smWaking ?? 0) > 0 && (
                    <div>
                      {t('campaigns.waking')}: {st.smWaking}
                    </div>
                  )}
                  {st.smLastStanding && st.smLastStanding.length > 0 && (
                    <div>
                      {t('campaigns.lastStanding')}: {st.smLastStanding.map(name).join(', ')}
                    </div>
                  )}
                </td>
              </tr>
            )}
            {st.box === 'mg' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {st.mgFuturePast && st.mgFuturePast.length > 0 && (
                    <div>
                      {t('campaigns.futurePast')}: {st.mgFuturePast.map(name).join(', ')}
                    </div>
                  )}
                  {st.mgCaptives && st.mgCaptives.length > 0 && (
                    <div>
                      {t('campaigns.captives')}: {st.mgCaptives.map(name).join(', ')}
                    </div>
                  )}
                  {st.mgRemovedAllies && st.mgRemovedAllies.length > 0 && (
                    <div>
                      {t('campaigns.removedAllies')}: {st.mgRemovedAllies.map(name).join(', ')}
                    </div>
                  )}
                  {st.flags?.jubilee && <div>{t('campaigns.jubilee')}</div>}
                </td>
              </tr>
            )}
            {st.box === 'nx' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {st.nxEnvEarned && st.nxEnvEarned.length > 0 && (
                    <div>
                      {t('campaigns.nxEnvs')}: {st.nxEnvEarned.map(name).join(', ')}
                    </div>
                  )}
                  {st.nxChosen && st.nxChosen.length > 0 && (
                    <div>
                      {t('campaigns.nxChosen')}: {st.nxChosen.map(name).join(', ')}
                    </div>
                  )}
                  {(st.counters?.hopeDamage ?? 0) > 0 && (
                    <div>
                      {t('campaigns.hopeDamage')}: {st.counters?.hopeDamage}
                    </div>
                  )}
                </td>
              </tr>
            )}
            {st.box === 'aoa' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {st.aoMissionLog && st.aoMissionLog.length > 0 && (
                    <div>
                      {t('campaigns.aoMissions')}: {st.aoMissionLog.map(name).join(', ')}
                    </div>
                  )}
                  {st.aoOverseerLog && st.aoOverseerLog.length > 0 && (
                    <div>
                      {t('campaigns.aoOverseers')}: {st.aoOverseerLog.map(name).join(', ')}
                    </div>
                  )}
                </td>
              </tr>
            )}
            {st.box === 'aos' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {st.aoEvidence && st.aoEvidence.length > 0 && (
                    <div>
                      {t('campaigns.aoEvidence')}: {st.aoEvidence.map(name).join(', ')}
                    </div>
                  )}
                  {st.aoCounters && Object.keys(st.aoCounters).length > 0 && (
                    <div>
                      {t('campaigns.aoCounters')}:{' '}
                      {Object.entries(st.aoCounters)
                        .map(([code, n]) => `${name(code)} ${n}`)
                        .join(', ')}
                    </div>
                  )}
                  {st.flags?.accused && (
                    <div>
                      {t('campaigns.aoAccused')}: {st.flags.accusedCorrect ? t('campaigns.aoCorrect') : t('campaigns.aoWrong')}
                    </div>
                  )}
                </td>
              </tr>
            )}
            {(st.box === 'cowl' || st.box === 'night' || st.box === 'whatif' || st.box === 'mojo' || st.box === 'entropy') && st.smCommunity && st.smCommunity.length > 0 && (
              <tr>
                <th>{t('campaigns.community')}</th>
                <td>{st.smCommunity.map(name).join(', ')}</td>
              </tr>
            )}
            {st.box === 'cowl' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {st.cowlCaught && st.cowlCaught.length > 0 && (
                    <div>
                      {t('campaigns.flag.caught')}: {st.cowlCaught.map(name).join(', ')}
                    </div>
                  )}
                  {st.pool && st.pool.length > 0 && (
                    <div>
                      {t('campaigns.flag.escaped')}: {st.pool.map(name).join(', ')}
                    </div>
                  )}
                  <div>
                    {t('campaigns.flag.intel')}: {(st.counters?.intel ?? 0)}
                  </div>
                </td>
              </tr>
            )}
            {st.box === 'whatif' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {st.flags?.shawarma && <div>{t('campaigns.flag.shawarmaPool')}</div>}
                  {st.flags?.towerDamaged && <div>{t('campaigns.towerDamaged')}</div>}
                  {st.flags?.crime && <div>{t('campaigns.flag.crime')}</div>}
                  {st.flags?.dinosaurs && <div>{t('campaigns.flag.dinosaurs')}</div>}
                </td>
              </tr>
            )}
            {st.box === 'awesome' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {st.flags?.modok && <div>{t('campaigns.flag.modok')}</div>}
                  {st.flags?.sleeper && <div>{t('campaigns.flag.sleeper')}</div>}
                  {(st.counters?.delay ?? 0) > 0 && (
                    <div>
                      {t('campaigns.flag.delay')}: {st.counters?.delay}
                    </div>
                  )}
                  {st.selections?.lastVillain && (
                    <div>
                      {t('campaigns.flag.lastVillain')}: {name(st.selections.lastVillain)}
                    </div>
                  )}
                </td>
              </tr>
            )}
            {st.box === 'alias' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {st.selections?.clue1 && (
                    <div>
                      {t('campaigns.flag.clue1')}: {name(st.selections.clue1)}
                    </div>
                  )}
                  {st.selections?.clue2 && (
                    <div>
                      {t('campaigns.flag.clue2')}: {name(st.selections.clue2)}
                    </div>
                  )}
                  {st.victims && st.victims.length > 0 && (
                    <div>
                      {t('campaigns.flag.victims')}: {st.victims.map(name).join(', ')}
                    </div>
                  )}
                  <div>
                    {t('campaigns.flag.wounds')}: {st.counters?.wounds ?? 0}
                  </div>
                  <div>
                    {t('campaigns.flag.tally')}: {st.counters?.tally ?? 0}
                  </div>
                </td>
              </tr>
            )}
            {st.box === 'mojo' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {(['training', 'hellfire', 'advanced1', 'advanced2', 'stronger', 'genosha'] as const).map((f) =>
                    st.flags?.[f] ? <div key={f}>{t('campaigns.flag.' + f)}</div> : null,
                  )}
                </td>
              </tr>
            )}
            {st.box === 'bord' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {st.selections?.path && (
                    <div>
                      {t('campaigns.flag.path')}: {t('campaigns.path.' + st.selections.path)}
                    </div>
                  )}
                  {(['blackDwarf', 'supergiant', 'corvus', 'proxima', 'blackSwan', 'shawarma', 'safehouse', 'breach', 'towerDamaged', 'ls2', 'ls3'] as const).map((f) =>
                    st.flags?.[f] ? <div key={f}>{t('campaigns.flag.' + f)}</div> : null,
                  )}
                </td>
              </tr>
            )}
            {st.box === 'night' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  {(['gw1', 'gw2', 'gw3', 'gw4'] as const).map((k) =>
                    st.selections?.[k] ? (
                      <div key={k}>
                        {t('campaigns.flag.' + k)}: {st.selections[k] === 'shehulk' ? 'She-Hulk' : 'Deadpool'}
                      </div>
                    ) : null,
                  )}
                  <div>
                    {t('campaigns.flag.alliance')}: {st.counters?.alliance ?? 0}
                  </div>
                  <div>
                    {t('campaigns.flag.rewinds')}: {st.counters?.rewinds ?? 0}
                  </div>
                  {st.pool && st.pool.length > 0 && (
                    <div>
                      'Pool: {st.pool.map(name).join(', ')}
                    </div>
                  )}
                </td>
              </tr>
            )}
            {st.box === 'viral' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  <div>
                    {t('campaigns.flag.pym')}: {st.counters?.pym ?? 0}
                  </div>
                  <div>
                    {t('campaigns.flag.infection')}: {st.counters?.infection ?? 0}
                  </div>
                  {st.selections?.viralPlayed && (
                    <div>
                      {t('campaigns.flag.viralPlayed')}: {st.selections.viralPlayed.split(',').map((i) => t('campaigns.viralNext.' + i)).join(', ')}
                    </div>
                  )}
                  {(['zolaStopped', 'zolaAlgorithm', 'sixStopped', 'sixUnited', 'nebulaStopped', 'nebulaAway', 'modokAway', 'scorpionAway'] as const).map((f) =>
                    st.flags?.[f] ? <div key={f}>{t('campaigns.flag.' + f)}</div> : null,
                  )}
                </td>
              </tr>
            )}
            {st.box === 'entropy' && (
              <tr>
                <th>{t('campaigns.campaignLog')}</th>
                <td>
                  <div>
                    {t('campaigns.reputation')}: {st.counters?.marked ?? 0}
                  </div>
                  {([1, 2, 3, 4, 5, 6, 7] as const).map((i) => {
                    const v = st.selections?.['cw' + i]
                    return v ? (
                      <div key={i}>
                        {t('campaigns.flag.cw' + i)}: {v}
                      </div>
                    ) : null
                  })}
                  {([1, 2] as const).map((i) => {
                    const v = st.selections?.['soe' + i]
                    return v ? (
                      <div key={i}>
                        {t('campaigns.flag.soe' + i)}: {name(v)}
                      </div>
                    ) : null
                  })}
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

// DeckPick is a dropdown over one player's deck (campaign picks that name
// a card the player already owns).
function DeckPick(props: {
  deck: Record<string, number>
  busy: boolean
  name: (code: string) => string
  t: (key: string, ...args: Array<string | number>) => string
  onPick: (code: string) => void
}) {
  return (
    <select
      defaultValue=""
      onChange={(e) => {
        const code = e.target.value
        if (code) props.onPick(code)
      }}
    >
      <option value="">{props.t('campaigns.pickCard')}</option>
      {Object.entries(props.deck).map(([code, n]) => (
        <option key={code} value={code}>
          {props.name(code)} ×{n}
        </option>
      ))}
    </select>
  )
}

// SMCodePick is a free-text card picker (aspect advantage: any aspect card
// by code). The server validates the choice.
function SMCodePick(props: {
  placeholder: string
  deck: Record<string, number>
  busy: boolean
  onPick: (code: string) => void
}) {
  const [code, setCode] = useState('')
  return (
    <div className="form inline">
      <input value={code} placeholder={props.placeholder} onChange={(e) => setCode(e.target.value.trim())} />
      <button disabled={props.busy || !code} onClick={() => props.onPick(code)}>
        OK
      </button>
    </div>
  )
}
