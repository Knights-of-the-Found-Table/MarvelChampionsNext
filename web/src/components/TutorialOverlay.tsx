import { useEffect, useMemo, useState } from 'react'
import type { GameView } from '../api'
import { useT } from '../i18n'

interface Props {
  view: GameView | null
  // suppressed 为 true 时（剧情弹窗等更高优先级弹窗激活中）暂时隐藏
  // 教程层——它是 fixed inset:0 的全屏层，不隐藏会吃掉弹窗的触摸事件。
  suppressed?: boolean
}

interface Step {
  id: string
  title: string
  body: string
  when: (view: GameView | null) => boolean
}

// 对局内新手教学：不进规则引擎，只根据当前视图给下一步提示。
export default function TutorialOverlay({ view, suppressed = false }: Props) {
  const t = useT()
  const storageKey = 'mc-tutorial-seen-v1'
  const [active, setActive] = useState(false)
  const [manual, setManual] = useState(false)
  const [stepIndex, setStepIndex] = useState(0)
  const [seen, setSeen] = useState(() => localStorage.getItem(storageKey) === '1')

  const steps = useMemo<Step[]>(
    () => [
      {
        id: 'turn',
        title: t('tutorial.turn.title'),
        body: t('tutorial.turn.body'),
        when: (v) => !!v && !v.over,
      },
      {
        id: 'resources',
        title: t('tutorial.resources.title'),
        body: t('tutorial.resources.body'),
        when: (v) => !!v && !v.over,
      },
      {
        id: 'actions',
        title: t('tutorial.actions.title'),
        body: t('tutorial.actions.body'),
        when: (v) => !!v && !v.over,
      },
      {
        id: 'end-turn',
        title: t('tutorial.endTurn.title'),
        body: t('tutorial.endTurn.body'),
        when: (v) => !!v && !v.over,
      },
      {
        id: 'enemy-phase',
        title: t('tutorial.enemy.title'),
        body: t('tutorial.enemy.body'),
        when: (v) => !!v && !v.over,
      },
      {
        id: 'defense',
        title: t('tutorial.defense.title'),
        body: t('tutorial.defense.body'),
        when: (v) => !!v && !v.over,
      },
    ],
    [t]
  )

  const step = steps[Math.min(stepIndex, steps.length - 1)]

  useEffect(() => {
    if (!seen && view && !view.over) setActive(true)
  }, [seen, view])

  function close(markSeen: boolean) {
    setActive(false)
    setManual(false)
    if (markSeen) {
      setSeen(true)
      localStorage.setItem(storageKey, '1')
    }
  }

  function next() {
    if (stepIndex + 1 >= steps.length) close(true)
    else setStepIndex((n) => n + 1)
  }

  function open() {
    setManual(true)
    setStepIndex(0)
    setActive(true)
  }

  return (
    <>
      <button className="tutorial-toggle" onClick={open} title={t('tutorial.open')}>
        ?
      </button>
      {active && !suppressed && step && (
        <div className="tutorial-overlay" role="dialog" aria-label={t('tutorial.open')}>
          <div className="tutorial-card">
            <div className="tutorial-head">
              <strong>{step.title}</strong>
              <span className="muted">
                {stepIndex + 1}/{steps.length}
              </span>
            </div>
            <p>{step.body}</p>
            <div className="tutorial-actions">
              <button onClick={() => close(true)}>{t('tutorial.skip')}</button>
              <button className="primary" onClick={next}>
                {stepIndex + 1 >= steps.length ? t('tutorial.done') : t('tutorial.next')}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
