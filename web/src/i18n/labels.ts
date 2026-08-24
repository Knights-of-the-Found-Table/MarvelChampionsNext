// 选项/提示文本直通层。服务端（MC_LANG=zh，默认）已在产出点输出中文
// 的提示、选项标签与日志（internal/engine/zhtext.go），前端不再对英文
// 做反向翻译；本文件仅保留 QuestionPanel 等组件使用的接口形状。
import type { Choice } from '../api'
import { useLang } from '../i18n'

type ZhMap = Record<string, { name: string }> | null

export function localizeLabel(label: string, _cardCode: string | undefined, _lang: string, _zh: ZhMap): string {
  return label
}

export function localizePrompt(prompt: string, _lang: string): string {
  return prompt
}

export function useChoiceLabel(): (c: Choice) => string {
  const lang = useLang()
  const zh = null
  return (c) => localizeLabel(c.label, c.cardCode, lang, zh)
}
