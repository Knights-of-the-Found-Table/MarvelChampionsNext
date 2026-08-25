import { useRef } from 'react'
import type { Choice, Question } from '../api'
import { CardImage, useCardZoom } from '../cards'
import { useT } from '../i18n'
import { useChoiceLabel, useEngineMsg } from '../i18n/labels'

function usePromptText(q: Question): string {
  const em = useEngineMsg()
  return em({ key: q.promptKey, args: q.promptArgs, text: q.prompt ?? '' })
}

interface Props {
  current: Question
  selected: Set<string>
  onPick: (c: Choice) => void
  onBack?: () => void
  onConfirm: () => void
  // 默认折叠：棋盘点卡是主交互，本面板作为"全部可选操作"的调试清单。
  open: boolean
  onToggle: () => void
}

function ChoiceButton({
  choice,
  selected,
  onPick,
}: {
  choice: Choice
  selected: boolean
  onPick: (c: Choice) => void
}) {
  const ref = useRef<HTMLButtonElement | null>(null)
  const zoom = useCardZoom(choice.cardCode ?? '', ref)
  const choiceLabel = useChoiceLabel()
  return (
    <button
      ref={ref}
      className={`choice ${kindClass(choice.kind)} ${selected ? 'selected' : ''}`}
      disabled={choice.disabled}
      onClick={() => onPick(choice)}
      onMouseEnter={choice.cardCode ? zoom.onEnter : undefined}
      onMouseLeave={choice.cardCode ? zoom.hide : undefined}
    >
      {choice.cardCode && <CardImage code={choice.cardCode} size="xs" zoom={false} />}
      <span>{choiceLabel(choice)}</span>
      {choice.cardCode && zoom.overlay}
    </button>
  )
}

export default function QuestionPanel({ current, selected, onPick, onBack, onConfirm, open, onToggle }: Props) {
  const t = useT()
  const promptText = usePromptText(current)

  const isMulti = current.type === 'choose_n'
  const need = current.n ?? 1
  // 图标需求/贡献全部来自服务端结构化字段（question.payIcons /
  // choice.icons），绝不解析渲染后的提示或标签文本。
  const requiredIcons = current.payIcons ?? []
  const selectedIcons = new Map<string, number>()
  if (requiredIcons.length > 0) {
    for (const c of current.choices) {
      if (!selected.has(c.id)) continue
      for (const icon of c.icons ?? []) selectedIcons.set(icon, (selectedIcons.get(icon) ?? 0) + 1)
    }
  }
  const missingIcons = requiredIcons.filter(({ icon, n }) => (selectedIcons.get(icon) ?? 0) < n)

  // 折叠条：提示 + 选项数，点击展开完整清单
  if (!open) {
    return (
      <div className="question-collapsed" onClick={onToggle}>
        <span className="q-toggle">▸</span>
        <span className="q-prompt">{promptText}</span>
        <span className="muted">
          {t('q.choiceCount', { n: current.choices.length })}
        </span>
        {isMulti && selected.size > 0 && (
          <span className="q-selected-count">{selected.size}</span>
        )}
      </div>
    )
  }

  return (
    <div className="question card">
      <div className="row space-between">
        <strong>{promptText}</strong>
        <div className="row">
          {onBack && (
            <button className="linklike" onClick={onBack}>
              {t('q.back')}
            </button>
          )}
          <button className="linklike" onClick={onToggle}>
            {t('q.collapse')}
          </button>
          {isMulti && (
            <span className="muted">
              {t('q.selected', { n: selected.size })}
              {need > 1 ? ` ${t('q.need', { n: need })}` : ''}
            </span>
          )}
          {requiredIcons.length > 0 && (
            <div className={`payment-icons ${missingIcons.length > 0 ? 'missing' : ''}`}>
              {requiredIcons.map(({ icon, n }) => {
                const got = selectedIcons.get(icon) ?? 0
                const wild = selectedIcons.get('wild') ?? 0
                const met = got >= n || got + wild >= n
                return (
                  <span key={icon} className={`payment-icon ${met ? 'met' : 'missing'}`}>
                    {t(`res.${icon}`)} {Math.min(got + wild, n)}/{n}
                  </span>
                )
              })}
            </div>
          )}
        </div>
      </div>
      {isMulti ? (
        <>
          <div className="choices wrap">
            {current.choices.map((c) => (
              <ChoiceButton key={c.id} choice={c} selected={selected.has(c.id)} onPick={onPick} />
            ))}
          </div>
          <button className="primary" onClick={onConfirm} disabled={selected.size === 0}>
            {t('q.confirm')}
          </button>
        </>
      ) : (
        <div className="choices wrap">
          {current.choices.map((c) => (
            <ChoiceButton key={c.id} choice={c} selected={selected.has(c.id)} onPick={onPick} />
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
