import { useState } from 'react'
import type { Choice, Question } from '../api'
import { CardImage } from '../cards'

interface Props {
  question: Question
  onAnswer: (paths: string[]) => void
}

// QuestionPanel renders a question tree. choose_one descends into nested
// Then-questions; choose_n collects a selection and confirms.
export default function QuestionPanel({ question, onAnswer }: Props) {
  const [stack, setStack] = useState<Question[]>([])
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const current = stack.length > 0 ? stack[stack.length - 1] : question

  const isMulti = current.type === 'choose_n'
  const need = current.n ?? 1

  function reset() {
    setStack([])
    setSelected(new Set())
  }

  function pick(c: Choice) {
    if (isMulti) {
      const next = new Set(selected)
      if (next.has(c.id)) next.delete(c.id)
      else if (next.size < need + 4) next.add(c.id) // allow over-selection? keep simple cap
      setSelected(next)
      return
    }
    if (c.then && c.then.choices.length > 0) {
      setStack([...stack, c.then])
      return
    }
    onAnswer([c.id])
    reset()
  }

  function confirmMulti() {
    const paths = Array.from(selected)
    if (paths.length === 0) return
    onAnswer(paths)
    reset()
  }

  function back() {
    setStack(stack.slice(0, -1))
    setSelected(new Set())
  }

  return (
    <div className="question card">
      <div className="row space-between">
        <strong>{current.prompt || 'Choose'}</strong>
        <div className="row">
          {stack.length > 0 && (
            <button className="linklike" onClick={back}>
              ‹ back
            </button>
          )}
          {isMulti && (
            <span className="muted">
              {selected.size} selected
              {need > 1 ? ` (need ≥ ${need})` : ''}
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
                onClick={() => pick(c)}
              >
                {c.cardCode && <CardImage code={c.cardCode} size="xs" />}
                <span>{c.label}</span>
              </button>
            ))}
          </div>
          <button className="primary" onClick={confirmMulti} disabled={selected.size === 0}>
            Confirm
          </button>
        </>
      ) : (
        <div className="choices wrap">
          {current.choices.map((c) => (
            <button
              key={c.id}
              className={`choice ${kindClass(c.kind)}`}
              disabled={c.disabled}
              onClick={() => pick(c)}
            >
              {c.cardCode && <CardImage code={c.cardCode} size="xs" />}
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
