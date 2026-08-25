// 对局文案由服务端消息目录统一本地化。这里保留轻量直通接口，避免
// QuestionPanel 把显示文字再次翻译，也不再伪装成持有一份前端卡牌词典。
import type { Choice } from '../api'

export function localizePrompt(prompt: string, _lang: string): string {
  return prompt
}

export function useChoiceLabel(): (c: Choice) => string {
  return (c) => c.label
}
