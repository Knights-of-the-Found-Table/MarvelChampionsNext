import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { get, post, getToken, type Choice, type GameView, type Question } from '../api'
import { lname, useT, useZhMap } from '../i18n'
import { useChoiceLabel } from '../i18n/labels'
import type { GameEvt } from '../board/fx'
import type { PlacedCard } from '../board/layout'
import { initSfx, playSfx, setSfxMuted, sfxSettings } from '../audio/sfx'
import { CardImage } from '../cards'
import Board from '../components/Board'
import QuestionPanel from '../components/QuestionPanel'
import ReportBugButton from '../components/ReportBugButton'
import '../style/board.css'

export default function Game() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const gameId = Number(id)
  const t = useT()
  const choiceLabel = useChoiceLabel()
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

  // 面板默认折叠（棋盘点卡为主交互）；仅当问题没有任何实体关联选项
  // （防御询问、调度等）时自动展开兜底。
  const [panelOpen, setPanelOpen] = useState(false)
  const questionKey = question ? JSON.stringify(question) : ''
  useEffect(() => {
    setStack([])
    setSelected(new Set())
    if (question) {
      const hasEntityChoice = question.choices.some((c) => c.sourceId)
      setPanelOpen(!hasEntityChoice)
    }
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

  // 结束回合/恢复：场景快捷按钮的可用性（根菜单层）
  const endTurnChoice = stack.length === 0 ? current?.choices.find((c) => c.kind === 'end_turn' && !c.disabled) : undefined
  const recoverChoice =
    stack.length === 0 ? current?.choices.find((c) => !c.disabled && c.id === 'basic-recover') : undefined

  // 点场上卡牌：牌堆 → 打开卡牌列表查看器；实体有多个可用选项（英雄牌的
  // 翻面/技能等）→ 弹选择菜单；唯一选项直接执行。
  const [entityMenu, setEntityMenu] = useState<{ card: PlacedCard; choices: Choice[] } | null>(null)

  function onCardClick(card: PlacedCard) {
    if (card.kind === 'pile') {
      if (card.id === 'pile-encounter') {
        void openPile('', 'deck', t('pile.encounter'))
      } else if (card.label === 'deck' || card.label === 'discard') {
        const pid = card.id.replace(/^pile-(deck|discard)-/, '')
        const owner = view?.players.find((p) => p.id === pid)
        void openPile(pid, card.label, `${owner?.name ?? ''} · ${t(card.label === 'deck' ? 'pile.deck' : 'pile.discard')}`)
      }
      return
    }
    if (!current) return
    // 恢复有专门按钮（英雄牌左侧），不再占用英雄牌点击
    const matches = current.choices.filter(
      (ch) => ch.sourceId === card.id && !ch.disabled && ch.id !== 'basic-recover'
    )
    if (matches.length === 0) return
    if (matches.length === 1) {
      pick(matches[0])
      return
    }
    setEntityMenu({ card, choices: matches })
  }

  // 牌堆查看器：牌库列表服务端已洗牌（不泄露抽牌顺序），弃牌堆顶牌优先。
  const [pileModal, setPileModal] = useState<{
    title: string
    cards: Array<{ code: string; name: string }>
  } | null>(null)

  async function openPile(player: string, pile: string, title: string) {
    try {
      const data = await get<{ cards: Array<{ code: string; name: string }> }>(
        `/marvel/games/${gameId}/pile?player=${encodeURIComponent(player)}&pile=${pile}`
      )
      setPileModal({ title, cards: data.cards })
    } catch (err) {
      setError(String((err as Error).message))
      setTimeout(() => setError(''), 3000)
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
      <Board
        view={view}
        events={events}
        question={current}
        selected={selected}
        onCardClick={onCardClick}
        onEndTurn={endTurnChoice ? () => pick(endTurnChoice) : undefined}
        onRecover={recoverChoice ? () => pick(recoverChoice) : undefined}
      />
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
        </div>
        {/* 错误 toast：独立于顶栏，右下角显示 */}
        {error && <div className="error-toast">{error}</div>}
        <div className="hud-controls">
          <ReportBugButton className="hud-report-bug" />
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
        {/* 多选确认条：棋盘点选支付卡后出现 */}
        {isMulti && selected.size > 0 && (
          <div className="confirm-bar">
            <span className="muted">{t('q.selected', { n: selected.size })}</span>
            <button className="primary" onClick={confirmMulti}>
              {t('q.confirm')}
            </button>
            <button onClick={() => setSelected(new Set())}>{t('q.clear')}</button>
          </div>
        )}
        {/* 实体多选项菜单（点英雄牌等出现） */}
        {entityMenu && (
          <div className="entity-menu" onClick={() => setEntityMenu(null)}>
            <div className="entity-menu-body" onClick={(e) => e.stopPropagation()}>
              <div className="row space-between">
                <strong>{entityMenu.card.title}</strong>
                <button className="linklike" onClick={() => setEntityMenu(null)}>
                  {t('pile.close')}
                </button>
              </div>
              <div className="entity-menu-choices">
                {entityMenu.choices.map((c) => (
                  <button
                    key={c.id}
                    className="choice"
                    onClick={() => {
                      setEntityMenu(null)
                      pick(c)
                    }}
                  >
                    {c.cardCode && <CardImage code={c.cardCode} size="xs" zoom={false} />}
                    <span>{choiceLabel(c)}</span>
                  </button>
                ))}
              </div>
            </div>
          </div>
        )}
        {/* 牌堆查看器 */}
        {pileModal && (
          <div className="pile-modal" onClick={() => setPileModal(null)}>
            <div className="pile-modal-body" onClick={(e) => e.stopPropagation()}>
              <div className="row space-between">
                <strong>
                  {pileModal.title}
                  <span className="muted"> · {pileModal.cards.length}</span>
                </strong>
                <button className="linklike" onClick={() => setPileModal(null)}>
                  {t('pile.close')}
                </button>
              </div>
              {pileModal.cards.length === 0 ? (
                <p className="muted">{t('pile.empty')}</p>
              ) : (
                <div className="pile-grid">
                  {pileModal.cards.map((c, i) => (
                    <CardImage key={`${c.code}-${i}`} code={c.code} size="sm" />
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
        {current ? (
          <QuestionPanel
            current={current}
            selected={selected}
            onPick={pick}
            onBack={stack.length > 0 ? back : undefined}
            onConfirm={confirmMulti}
            open={panelOpen}
            onToggle={() => setPanelOpen((v) => !v)}
          />
        ) : view.over ? (
          <div className="question game-over-panel">
            <p className={view.won ? 'victory' : 'defeat'}>
              {t('game.over')} — {t(view.won ? 'game.victory' : 'game.defeat')}: {view.reason}
            </p>
            <button className="primary" onClick={() => navigate('/')}>
              {t('game.backHome')}
            </button>
          </div>
        ) : null}
      </div>
    </div>
  )
}
