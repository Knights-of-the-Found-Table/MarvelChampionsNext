import type { Choice, Question } from '../api'
import { CardImage } from '../cards'
import { useT } from '../i18n'

interface Props {
  current: Question
  selected: Set<string>
  onPick: (c: Choice) => void
  onBack?: () => void
  onConfirm: () => void
}

// 问题面板（受控组件）：导航/多选状态由 Game.tsx 持有，使棋盘点卡与
// 面板按钮共享同一条选择路径。choose_one 直接 pick；choose_n 点选切换，
// 确认后作答。
export default function QuestionPanel({ current, selected, onPick, onBack, onConfirm }: Props) {
  const t = useT()

  const isMulti = current.type === 'choose_n'
  const need = current.n ?? 1

  return (
    <div className="question card">
      <div className="row space-between">
        <strong>{current.prompt || t('q.choose')}</strong>
        <div className="row">
          {onBack && (
            <button className="linklike" onClick={onBack}>
              {t('q.back')}
            </button>
          )}
          {isMulti && (
            <span className="muted">
              {t('q.selected', { n: selected.size })}
              {need > 1 ? ` ${t('q.need', { n: need })}` : ''}
            </span>
          )}
        </div>
      </div>
      {isMulti ? (
        <>
          <div className="choices wrap">
            {current.choices.map((c) => (
              <button
                key={c.id}
                className={`choice ${selected.has(c.id) ? 'selected' : ''}`}
                disabled={c.disabled}
                onClick={() => onPick(c)}
              >
                {c.cardCode && <CardImage code={c.cardCode} size="xs" zoom={false} />}
                <span>{c.label}</span>
              </button>
            ))}
          </div>
          <button className="primary" onClick={onConfirm} disabled={selected.size === 0}>
            {t('q.confirm')}
          </button>
        </>
      ) : (
        <div className="choices wrap">
          {current.choices.map((c) => (
            <button
              key={c.id}
              className={`choice ${kindClass(c.kind)}`}
              disabled={c.disabled}
              onClick={() => onPick(c)}
            >
              {c.cardCode && <CardImage code={c.cardCode} size="xs" zoom={false} />}
              <span>{c.label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

function kindClass(kind?: string): string {
  switch (kind) {
    case 'end_turn':
      return 'end-turn'
    case 'form':
      return 'form'
    case 'play':
      return 'play'
    case 'basic_power':
      return 'power'
    case 'ability':
      return 'ability'
    case 'target':
      return 'target'
    default:
      return ''
  }
}
