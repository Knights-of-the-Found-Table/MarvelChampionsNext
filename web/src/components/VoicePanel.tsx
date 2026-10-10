import { useState } from 'react'
import { useT } from '../i18n'
import { useVoice } from '../voice'

interface Props {
  gameId: string
}

// 语音挂件：与聊天挂件并排的 🎙️ 气泡。mesh 与信令全部在 useVoice /
// VoiceSession 里，这里只做面板 UI：加入/离开、静音、在场名单与说话指示。
export default function VoicePanel({ gameId }: Props) {
  const t = useT()
  const [open, setOpen] = useState(false)
  const v = useVoice(gameId)
  const live = v.status === 'live'
  const joined = live || v.status === 'connecting'
  const roster = v.self ? [v.self, ...v.peers] : v.peers

  return (
    <div className="voice-widget">
      {open && (
        <div className="voice-panel" role="dialog" aria-label={t('voice.title')}>
          <div className="chat-head">
            <strong>{t('voice.title')}</strong>
            <button className="linklike" onClick={() => setOpen(false)}>
              {t('pile.close')}
            </button>
          </div>
          {v.errorKey && <p className="voice-error">{t(v.errorKey)}</p>}
          {!joined ? (
            <div className="voice-join">
              <p className="muted">{t('voice.joinHint')}</p>
              <button className="primary" onClick={() => void v.join()}>
                {t('voice.join')}
              </button>
            </div>
          ) : (
            <>
              <div className="voice-roster">
                {roster.length === 0 ? (
                  <p className="muted">{t('voice.connecting')}</p>
                ) : (
                  roster.map((p) => (
                    <div
                      key={p.id}
                      className={`voice-chip ${v.speaking.has(p.id) ? 'speaking' : ''} ${p.spectator ? 'spectator' : ''}`}
                    >
                      <span className="voice-badge" aria-hidden="true">
                        {p.spectator ? '👁' : '🎙'}
                      </span>
                      <span className="voice-name">
                        {p.id === v.self?.id ? `${p.name} · ${t('voice.you')}` : p.name}
                      </span>
                      {p.muted && (
                        <span className="voice-muted-mark" title={t('voice.muted')}>
                          🔇
                        </span>
                      )}
                    </div>
                  ))
                )}
                {live && v.peers.length === 0 && <p className="muted voice-alone">{t('voice.empty')}</p>}
              </div>
              <div className="voice-controls">
                <button onClick={v.toggleMute} disabled={!live}>
                  {v.muted ? `🔇 ${t('voice.unmute')}` : `🎙 ${t('voice.mute')}`}
                </button>
                <button className="voice-leave" onClick={v.leave} disabled={!live}>
                  {t('voice.leave')}
                </button>
              </div>
            </>
          )}
        </div>
      )}
      <button
        className={`chat-fab voice-fab ${joined ? 'on' : ''} ${v.speaking.size > 0 ? 'talking' : ''}`}
        onClick={() => setOpen((s) => !s)}
        aria-label={t('voice.title')}
        title={t('voice.title')}
      >
        {/* 内联 SVG 麦克风，与聊天气泡同风格。 */}
        <svg viewBox="0 0 24 24" width="22" height="22" fill="currentColor" aria-hidden="true">
          <path d="M12 3a3 3 0 0 0-3 3v6a3 3 0 0 0 6 0V6a3 3 0 0 0-3-3Z" />
          <path d="M6 11a1 1 0 0 1 1 1 5 5 0 0 0 10 0 1 1 0 0 1 2 0 7 7 0 0 1-6 6.93V21a1 1 0 0 1-2 0v-2.07A7 7 0 0 1 5 12a1 1 0 0 1 1-1Z" />
        </svg>
        {joined && v.muted && <span className="voice-fab-muted">🔇</span>}
      </button>
    </div>
  )
}
