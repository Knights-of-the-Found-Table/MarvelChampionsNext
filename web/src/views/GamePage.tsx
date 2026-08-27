// 对局入口页：先探测对局是否还在大厅（轮询 /lobby，开始后该接口 404），
// 在大厅则渲染邀请界面，否则渲染棋盘对局。房主开始对局后所有访客在
// 轮询周期内自动切换到棋盘。
import { useCallback, useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { get, post, type Deck, type LobbyView } from '../api'
import { errorText } from '../deckIssues'
import { useT, useZhMap } from '../i18n'
import Game from './Game'

export default function GamePage() {
  const { id } = useParams<{ id: string }>()
  const [lobby, setLobby] = useState<LobbyView | null>(null)
  const [gone, setGone] = useState(false)
  const goneRef = useRef(false)

  const poll = useCallback(async () => {
    if (!id || goneRef.current) return
    try {
      setLobby(await get<LobbyView>(`/marvel/games/${id}/lobby`))
    } catch {
      goneRef.current = true
      setLobby(null)
      setGone(true)
    }
  }, [id])

  useEffect(() => {
    goneRef.current = false
    setLobby(null)
    setGone(false)
    void poll()
    const timer = setInterval(() => void poll(), 2500)
    return () => clearInterval(timer)
  }, [poll])

  if (lobby) {
    return (
      <Lobby
        lobby={lobby}
        onRefresh={() => void poll()}
        onStarted={() => {
          goneRef.current = true
          setLobby(null)
          setGone(true)
        }}
      />
    )
  }
  if (!gone) return null
  return <Game />
}

function Lobby({
  lobby,
  onRefresh,
  onStarted,
}: {
  lobby: LobbyView
  onRefresh: () => void
  onStarted: () => void
}) {
  const t = useT()
  const zh = useZhMap()
  const [me, setMe] = useState<{ id: number; username: string } | null>(null)
  const [decks, setDecks] = useState<Deck[]>([])
  const [deckChoice, setDeckChoice] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    get<{ id: number; username: string }>('/whoami').then(setMe).catch(() => {})
    get<Deck[]>('/marvel/decks').then(setDecks).catch(() => {})
  }, [])

  const myId = me ? String(me.id) : ''
  const myEntry = lobby.players.find((p) => p.userId === myId)
  const isHost = !!myEntry?.host
  const joined = !!myEntry?.deck
  const hostEntry = lobby.players.find((p) => p.host)
  const joinedCount = lobby.players.filter((p) => p.deck).length
  const inviteLink = `${window.location.origin}/games/${lobby.id}`

  async function run(fn: () => Promise<unknown>) {
    setError('')
    setBusy(true)
    try {
      await fn()
      onRefresh()
    } catch (err) {
      setError(errorText(t, zh, err))
    } finally {
      setBusy(false)
    }
  }

  async function copyInvite() {
    try {
      await navigator.clipboard.writeText(inviteLink)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // 剪贴板不可用（非安全上下文等）：输入框本身可手动选择复制。
    }
  }

  const deckPicker = (onSubmit: () => void, label: string) => (
    <div className="lobby-picker">
      <select value={deckChoice} onChange={(e) => setDeckChoice(e.target.value)}>
        {decks.length === 0 && <option value="">{t('newgame.importFirst')}</option>}
        {decks.map((d) => (
          <option key={d.id} value={d.id} disabled={d.valid === false}>
            {d.name}
            {d.valid === false ? ` ⚠ ${t('decks.invalid')}` : ''}
          </option>
        ))}
      </select>
      <button className="primary" disabled={busy || deckChoice === ''} onClick={onSubmit}>
        {label}
      </button>
    </div>
  )

  return (
    <section className="lobby">
      <h2>{lobby.name || t('lobby.title')}</h2>
      <p className="muted">
        {lobby.scenarioId} · {lobby.difficulty} · {t('newgame.playersMany', lobby.playerCount)}
      </p>
      {error && <p className="error">{error}</p>}

      <div className="card">
        <strong>{t('lobby.invite')}</strong>
        <div className="lobby-invite">
          <input readOnly value={inviteLink} onFocus={(e) => e.target.select()} />
          <button onClick={() => void copyInvite()}>{copied ? t('lobby.copied') : t('lobby.copy')}</button>
        </div>
      </div>

      <div className="card">
        <strong>{t('lobby.title')}</strong>
        <div className="lobby-rows">
          {lobby.players.map((p) => (
            <div key={p.slot} className="lobby-row">
              <span className="grow">
                {p.username || `#${p.slot}`}
                {p.host && <span className="tag-host">{t('lobby.host')}</span>}
              </span>
              <span className="muted">
                {p.deck ? (
                  <>
                    {p.deck.name}
                    {p.deck.valid === false && (
                      <span className="deck-invalid-badge">⚠ {t('decks.invalid')}</span>
                    )}
                  </>
                ) : (
                  t('lobby.needDeckFirst')
                )}
              </span>
              {isHost && !p.host && (
                <button disabled={busy} onClick={() => void run(() => post(`/marvel/games/${lobby.id}/kick`, { slot: p.slot }))}>
                  {t('lobby.remove')}
                </button>
              )}
            </div>
          ))}
          {Array.from({ length: lobby.openSlots }).map((_, i) => (
            <div key={`open-${i}`} className="lobby-row open">
              <span className="muted">{t('lobby.openSlots')}</span>
            </div>
          ))}
        </div>

        {me && !joined && (
          <div className="lobby-action">
            {deckPicker(
              () => void run(() => post(`/marvel/games/${lobby.id}/join`, { deck: deckChoice })),
              t('lobby.setDeck'),
            )}
          </div>
        )}
        {me && joined && !isHost && (
          <div className="lobby-action">
            <span>{t('lobby.joined')}</span>
            {deckPicker(
              () => void run(() => post(`/marvel/games/${lobby.id}/join`, { deck: deckChoice })),
              t('lobby.changeDeck'),
            )}
          </div>
        )}
        {isHost && (
          <div className="lobby-action">
            <button
              className="primary"
              disabled={busy || !hostEntry?.deck}
              title={hostEntry?.deck ? undefined : t('lobby.needDeckFirst')}
              onClick={() =>
                void run(async () => {
                  await post(`/marvel/games/${lobby.id}/start`, {})
                  onStarted()
                })
              }
            >
              {t(
                'lobby.start',
                joinedCount === 1 ? t('newgame.playersOne') : t('newgame.playersMany', joinedCount),
              )}
            </button>
            {hostEntry?.deck && joinedCount < lobby.playerCount && (
              <span className="muted">{t('lobby.openSlots')}</span>
            )}
          </div>
        )}
      </div>
    </section>
  )
}
