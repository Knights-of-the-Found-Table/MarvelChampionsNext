import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { get, post, type ChatMessage } from '../api'
import { useT } from '../i18n'

interface Props {
  gameId: number
  incoming: ChatMessage | null
}

// 对局聊天挂件：右下气泡，不影响棋盘布局；消息来自服务端 WS 与历史 API。
export default function ChatPanel({ gameId, incoming }: Props) {
  const t = useT()
  const [open, setOpen] = useState(false)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [draft, setDraft] = useState('')
  const [sending, setSending] = useState(false)
  const [unread, setUnread] = useState(0)
  const listRef = useRef<HTMLDivElement | null>(null)
  const openRef = useRef(open)
  openRef.current = open

  useEffect(() => {
    get<{ messages: ChatMessage[] }>(`/marvel/games/${gameId}/chat`)
      .then((data) => setMessages(data.messages ?? []))
      .catch(() => undefined)
  }, [gameId])

  useEffect(() => {
    if (!incoming) return
    setMessages((prev) => (prev.some((m) => m.id === incoming.id) ? prev : [...prev, incoming]))
    if (!openRef.current) setUnread((n) => n + 1)
  }, [incoming])

  useEffect(() => {
    if (open) setUnread(0)
  }, [open])

  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight })
  }, [messages, open])

  async function send(text: string) {
    const clean = text.trim()
    if (!clean || sending) return
    setSending(true)
    try {
      const data = await post<{ message: ChatMessage }>(`/marvel/games/${gameId}/chat`, { text: clean })
      setMessages((prev) => (prev.some((m) => m.id === data.message.id) ? prev : [...prev, data.message]))
      setDraft('')
    } catch {
      // 失败先静默：WebSocket 断线时下一轮同步会恢复；错误 UI 交给全局 toast。
    } finally {
      setSending(false)
    }
  }

  function onKey(ev: KeyboardEvent<HTMLTextAreaElement>) {
    if (ev.key === 'Enter' && !ev.shiftKey) {
      ev.preventDefault()
      void send(draft)
    }
  }

  const phrases = t('chat.phrases').split('|')

  return (
    <div className="chat-widget">
      {open && (
        <div className="chat-panel" role="dialog" aria-label={t('chat.title')}>
          <div className="chat-head">
            <strong>{t('chat.title')}</strong>
            <button className="linklike" onClick={() => setOpen(false)}>
              {t('pile.close')}
            </button>
          </div>
          <div className="chat-list" ref={listRef}>
            {messages.length === 0 ? (
              <p className="muted">{t('chat.empty')}</p>
            ) : (
              messages.map((m) => (
                <div key={m.id} className={`chat-line ${m.spectator ? 'spectator' : ''}`}>
                  <span className="chat-name">{m.name}</span>
                  <span className="chat-text">{m.text}</span>
                </div>
              ))
            )}
          </div>
          <div className="chat-phrases">
            {phrases.map((p) => (
              <button key={p} onClick={() => void send(p)} disabled={sending}>
                {p}
              </button>
            ))}
          </div>
          <div className="chat-compose">
            <textarea
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={onKey}
              maxLength={300}
              placeholder={t('chat.placeholder')}
              rows={2}
            />
            <button className="primary" onClick={() => void send(draft)} disabled={sending || !draft.trim()}>
              {t('chat.send')}
            </button>
          </div>
        </div>
      )}
      <button className="chat-fab" onClick={() => setOpen((v) => !v)} aria-label={t('chat.title')}>
        {/* 内联 SVG 气泡：emoji 字形的光学中心偏移会导致圆钮内对不齐。 */}
        <svg viewBox="0 0 24 24" width="22" height="22" fill="currentColor" aria-hidden="true">
          <path d="M4 2h16a3 3 0 0 1 3 3v9a3 3 0 0 1-3 3h-7.6l-4.7 3.6a1 1 0 0 1-1.7-.8V17H4a3 3 0 0 1-3-3V5a3 3 0 0 1 3-3Z" />
        </svg>
        {unread > 0 && <span className="chat-unread">{unread > 99 ? '99+' : unread}</span>}
      </button>
    </div>
  )
}
