// 热座换人（hot-seat）：一台设备多人轮流操作各自账号视角。
// 原理：换人 = 换 localStorage 里的登录 token + 页面重载。
// 名单只存在这台设备的 localStorage（mc-hotseat-roster），服务器零感知。
// token 有效期 30 天；过期条目在切换时会 401 回登录页，重新登录后可再次加入。
import { useEffect, useRef, useState } from 'react'
import { get, getToken, setToken } from '../api'
import { useT } from '../i18n'

interface SeatEntry {
  username: string
  token: string
  addedAt: number
}

const ROSTER_KEY = 'mc-hotseat-roster'

function loadRoster(): SeatEntry[] {
  try {
    const raw = localStorage.getItem(ROSTER_KEY)
    if (!raw) return []
    const list = JSON.parse(raw)
    if (!Array.isArray(list)) return []
    return list.filter((e) => e && typeof e.username === 'string' && typeof e.token === 'string')
  } catch {
    return []
  }
}

function saveRoster(list: SeatEntry[]) {
  localStorage.setItem(ROSTER_KEY, JSON.stringify(list))
}

export default function HotSeat() {
  const t = useT()
  const [open, setOpen] = useState(false)
  const [roster, setRoster] = useState<SeatEntry[]>(loadRoster)
  const [pending, setPending] = useState<SeatEntry | null>(null)
  const [busy, setBusy] = useState(false)
  const rootRef = useRef<HTMLDivElement | null>(null)
  const current = getToken()

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false)
        setPending(null)
      }
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setOpen(false)
        setPending(null)
      }
    }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  if (!current) return null

  const inRoster = roster.some((e) => e.token === current)

  async function addCurrent() {
    setBusy(true)
    try {
      const me = await get<{ id: number; username: string }>('/whoami')
      const token = getToken()
      if (!token) return
      const next = roster.filter((e) => e.token !== token && e.username !== me.username)
      next.push({ username: me.username, token, addedAt: Date.now() })
      setRoster(next)
      saveRoster(next)
    } finally {
      setBusy(false)
    }
  }

  function switchTo(entry: SeatEntry) {
    if (entry.token === current) return
    setPending(null)
    setToken(entry.token)
    location.reload()
  }

  function remove(entry: SeatEntry) {
    const next = roster.filter((e) => e.token !== entry.token)
    setRoster(next)
    saveRoster(next)
    if (pending?.token === entry.token) setPending(null)
  }

  return (
    <div className="hotseat" ref={rootRef}>
      <button
        type="button"
        className="hotseat-fab"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        title={t('hotseat.title')}
      >
        👥
      </button>
      {open && (
        <div className="hotseat-panel" role="dialog" aria-label={t('hotseat.title')}>
          <div className="hotseat-head">
            <strong>{t('hotseat.title')}</strong>
            {!inRoster && (
              <button type="button" className="linklike" disabled={busy} onClick={addCurrent}>
                {t('hotseat.addCurrent')}
              </button>
            )}
          </div>
          {roster.length === 0 ? (
            <p className="muted">{t('hotseat.empty')}</p>
          ) : (
            <ul className="hotseat-list">
              {roster.map((e) => {
                const isCurrent = e.token === current
                return (
                  <li key={e.token} className={isCurrent ? 'current' : ''}>
                    {pending?.token === e.token ? (
                      <span className="hotseat-confirm">
                        <span>{t('hotseat.confirmSwitch').replace('{name}', e.username)}</span>
                        <button type="button" className="primary" onClick={() => switchTo(e)}>
                          {t('hotseat.switch')}
                        </button>
                        <button type="button" onClick={() => setPending(null)}>
                          {t('hotseat.cancel')}
                        </button>
                      </span>
                    ) : (
                      <>
                        <button
                          type="button"
                          className="hotseat-name"
                          disabled={isCurrent}
                          onClick={() => setPending(e)}
                        >
                          {e.username}
                          {isCurrent && <span className="hotseat-here">{t('hotseat.current')}</span>}
                        </button>
                        {!isCurrent && (
                          <button
                            type="button"
                            className="hotseat-remove"
                            aria-label={t('hotseat.remove')}
                            onClick={() => remove(e)}
                          >
                            ×
                          </button>
                        )}
                      </>
                    )}
                  </li>
                )
              })}
            </ul>
          )}
          <p className="muted hotseat-note">{t('hotseat.note')}</p>
        </div>
      )}
    </div>
  )
}
