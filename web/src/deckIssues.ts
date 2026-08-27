// 牌组校验问题的本地化描述。服务端只发结构化 DeckIssue（规则键 + 数量
// 参数 + 卡码/英文印名兜底），这里按当前语言组装成可读文案；卡名按 code
// 查译名表（zh）或用服务端印名（en），与 i18n「逻辑不解析文本」的规约
// 一致——展示层只消费结构化字段。
import type { DeckIssue } from './api'
import { lname, type ZhCard } from './i18n'

type TFn = (key: string, ...args: Array<string | number>) => string

export function describeDeckIssues(
  t: TFn,
  zh: Record<string, ZhCard> | null,
  issues: DeckIssue[],
): string[] {
  return issues.map((is) => {
    const card = is.card ? lname(zh, is.card, is.title || is.card) : ''
    const aspect = is.aspect ? t(`aspect.${is.aspect}`) : ''
    switch (is.key) {
      case 'identityUnknown':
        return t('deck.issue.identityUnknown')
      case 'identityMismatch':
        return t('deck.issue.identityMismatch', card)
      case 'setMissing':
        return t('deck.issue.setMissing', card, is.n ?? 1)
      case 'setCount':
        return t('deck.issue.setCount', card, is.n ?? 0, is.m ?? 0)
      case 'tooSmall':
        return t('deck.issue.tooSmall', is.m ?? 0, is.n ?? 40)
      case 'tooBig':
        return t('deck.issue.tooBig', is.m ?? 0, is.n ?? 50)
      case 'wrongAspect':
        return t('deck.issue.wrongAspect', card, aspect)
      case 'cardIllegal':
        return t('deck.issue.cardIllegal', card)
      case 'poolWrongHero':
        return t('deck.issue.poolWrongHero', card)
      case 'tooManyAspects':
        return t('deck.issue.tooManyAspects', is.n ?? 1)
      case 'aspectsUnequal':
        return t('deck.issue.aspectsUnequal')
      case 'exceptCap':
        return t('deck.issue.exceptCap', is.n ?? 0, is.m ?? 0)
      case 'copyLimit':
        return t('deck.issue.copyLimit', card, is.n ?? 0, is.m ?? 0)
      default:
        return is.key
    }
  })
}

// 统一的错误文案：创建对局/入座被服务端拒绝时，把响应体里的 deckIssues
// 渲染成本地化清单；其余错误原样透出 message。
export function errorText(t: TFn, zh: Record<string, ZhCard> | null, err: unknown): string {
  const e = err as { deckIssues?: DeckIssue[]; message?: string }
  if (e?.deckIssues?.length) return describeDeckIssues(t, zh, e.deckIssues).join('\n')
  return String(e?.message ?? err)
}
