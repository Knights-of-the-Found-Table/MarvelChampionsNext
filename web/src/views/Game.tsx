import { useCallback, useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { get, post, getToken, type Choice, type GameView, type Question } from '../api'
import { lname, useT, useZhMap } from '../i18n'
import type { GameEvt } from '../board/fx'
import type { PlacedCard } from '../board/layout'
import { initSfx, playSfx, setSfxMuted, sfxSettings } from '../audio/sfx'
import Board from '../components/Board'
import QuestionPanel from '../components/QuestionPanel'
import '../style/board.css'

export default function Game() {
  const { id } = useParams<{ id: string }>()
  const gameId = Number(id)
  const t = useT()
  const zh = useZhMap()
  const [view, setView] = useState<GameView | null>(null)
  const [events, setEvents] = useState<GameEvt[]>([])
  const [error, setError] = useState('')
  const [sfxOn, setSfxOn] = useState(() => !sfxSettings().muted)
  const [animOn, setAnimOn] = useState(() => localStorage.getItem('mc-anim-off') !== '1')
  // 问题导航状态提升到这里：棋盘点卡与面板按钮共用同一条路径。
  const [stack, setStack] = useState<Question[]>([])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const wsRef = useRef<WebSocket | null>(null)
  const overRef = useRef(false)

  useEffect(() => {
    initSfx()
  }, [])

  useEffect(() => {
    document.body.classList.toggle('fx-off', !animOn)
    localStorage.setItem('mc-anim-off', animOn ? '' : '1')
  }, [animOn])

  // 终局音效（只播一次）
  useEffect(() => {
    if (view?.over && !overRef.current) {
      overRef.current = true
      playSfx(view.won ? 'victory' : 'defeat')
    }
  }, [view])

  const connect = useCallback(() => {
    const token = getToken() ?? ''
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(`${proto}://${location.host}/api/v1/marvel/games/${gameId}/stream?token=${encodeURIComponent(token)}`)
    ws.onmessage = (ev) => {
      try {
        const data = JSON.parse(ev.data)
        if (data.type === 'state') setView(data.view)
        else if (data.type === 'events') setEvents(data.events ?? [])
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
    setStack([])
    setSelected(new Set())
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

  const question = view?.question ?? null
  const current = stack.length > 0 ? stack[stack.length - 1] : question
  const isMulti = current?.type === 'choose_n'

  // 服务端换了一个新问题（或问题消失）时复位导航状态。每次广播都会
  // 反序列化出新对象，不能按引用比较——用内容做键。
  const questionKey = question ? JSON.stringify(question) : ''
  useEffect(() => {
    setStack([])
    setSelected(new Set())
  }, [questionKey])

  // 选项选择：choose_n 多选切换；choose_one 有子层下钻，叶子直接作答。
  // 棋盘点卡与 QuestionPanel 按钮都走这里。
  function pick(c: Choice) {
    if (c.disabled) return
    playSfx('select')
    if (isMulti) {
      setSelected((prev) => {
        const next = new Set(prev)
        if (next.has(c.id)) next.delete(c.id)
        else next.add(c.id)
        return next
      })
      return
    }
    if (c.then && c.then.choices.length > 0) {
      setStack((s) => [...s, c.then!])
      setSelected(new Set())
      return
    }
    void answer([c.id])
  }

  function back() {
    setStack((s) => s.slice(0, -1))
    setSelected(new Set())
  }

  function confirmMulti() {
    if (selected.size === 0) return
    void answer(Array.from(selected))
  }

  // 点场上卡牌：命中当前问题中 sourceId 对应的选项即视同点击面板按钮。
  // 同一实体有多个选项时（如盟友攻击/化解）取第一个可用项。
  function onCardClick(card: PlacedCard) {
    if (!current) return
    const c = current.choices.find((ch) => ch.sourceId === card.id && !ch.disabled)
    if (!c) return
    pick(c)
  }

  async function undo() {
    try {
      setView(await post<GameView>(`/marvel/games/${gameId}/undo`))
    } catch (err) {
      setError(String((err as Error).message))
      setTimeout(() => setError(''), 3000)
    }
  }

  function toggleSfx() {
    const next = !sfxOn
    setSfxOn(next)
    setSfxMuted(!next)
  }

  if (error && !view) return <p className="error">{error}</p>
  if (!view)
    return (
      <div className="board-page">
        <div className="board-msg">{t('deck.loading')}</div>
      </div>
    )

  return (
    <div className="board-page">
      <Board view={view} events={events} question={current} selected={selected} onCardClick={onCardClick} />
      <div className="board-hud">
        <div className="hud-top">
          <strong>{lname(zh, view.mainScheme?.code ?? '', view.scenario)}</strong>
          <span className="muted">· {t('game.round', { n: view.round })}</span>
          {view.over ? (
            <span className={view.won ? 'victory' : 'defeat'}>
              {t(view.won ? 'game.victory' : 'game.defeat')}
            </span>
          ) : view.waitingFor ? (
            <span className="muted">{t('game.waitingFor', { name: view.waitingFor })}</span>
          ) : null}
          {error && <span className="error">{error}</span>}
        </div>
        <div className="hud-controls">
          <button
            className={`hud-toggle ${sfxOn ? 'on' : ''}`}
            onClick={toggleSfx}
            title={t('game.sfx')}
          >
            {sfxOn ? '🔊' : '🔇'}
          </button>
          <button
            className={`hud-toggle ${animOn ? 'on' : ''}`}
            onClick={() => setAnimOn(!animOn)}
            title={t('game.anim')}
          >
            {animOn ? '✨' : '⏹'}
          </button>
          <button className="hud-undo" onClick={undo} disabled={view.over}>
            {t('game.undo')}
          </button>
        </div>
        <div className="hud-log">
          <details>
            <summary>{t('game.logTitle')}</summary>
            <div className="log-body">
              {(view.log ?? []).slice().reverse().map((e, i) => (
                <div key={i} className={`log-line log-${e.level || 'info'}`}>
                  {e.text}
                </div>
              ))}
            </div>
          </details>
        </div>
        {current ? (
          <QuestionPanel
            current={current}
            selected={selected}
            onPick={pick}
            onBack={stack.length > 0 ? back : undefined}
            onConfirm={confirmMulti}
          />
        ) : view.over ? (
          <div className="question">
            <p className={view.won ? 'victory' : 'defeat'}>
              {t('game.over')} — {t(view.won ? 'game.victory' : 'game.defeat')}: {view.reason}
            </p>
          </div>
        ) : null}
      </div>
    </div>
  )
}
