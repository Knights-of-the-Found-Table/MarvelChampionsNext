// 选项/提示文本的中文本地化：服务端 choice label 是英文（"Tenacity (cost
// 2)"、"Recover (4)"、"Change to Captain Marvel"），这里按规则翻译操作词，
// 卡名用 zh 卡牌映射（choice.cardCode → 中文名）替换英文前缀。
// 英文界面原样返回。
import type { Choice } from '../api'
import { useLang, useZhMap } from '../i18n'

type ZhMap = Record<string, { name: string }> | null

export function localizeLabel(label: string, cardCode: string | undefined, lang: string, zh: ZhMap): string {
  if (lang !== 'zh') return label
  let out = label

  // 固定句式先翻译（避免被卡名前缀替换吃掉）
  out = out
    .replace(/^Change to .+$/, '变身为其他形态')
    .replace(/^End turn$/, '结束回合')

  // 卡名前缀替换："Tenacity (cost 2)" / "Wraith attacks (2)" /
  // "Rechannel — pay 1…" → 中文名 + 余下部分。只处理带真实后缀边界
  // （" ("、" ["、" —"）的 label；纯短语（已翻译的固定句式）不动。
  const zhName = cardCode ? zh?.[cardCode]?.name : undefined
  if (zhName) {
    const idx = out.search(/\s+[(\[—]/)
    if (idx > 0) out = zhName + out.slice(idx)
  }

  out = out
    .replace(/^Recover \((\d+)\)/, '恢复（$1）')
    .replace(/^Attack \((\d+)\)/, '攻击（$1）')
    .replace(/^Thwart \((\d+)\)/, '化解（$1）')
    .replace(/ \(cost (\d+)\)/, '（费用 $1）')
    .replace(/ from discard/, '（从弃牌堆）')
    .replace(/ attacks \((\d+)\)/, ' 攻击（$1）')
    .replace(/ thwarts \((\d+)\)/, ' 化解（$1）')
    .replace(/^Pay (\d+) resources for (.+?)( \(select cards\))?$/, '为 $2 支付 $1 资源')
    .replace(/\[energy\]/g, '[能量]')
    .replace(/\[mental\]/g, '[精神]')
    .replace(/\[physical\]/g, '[物理]')
    .replace(/\[wild\]/g, '[万用]')
  return out
}

// prompt 本地化（无 cardCode；固定短语 + 支付框架）
export function localizePrompt(prompt: string, lang: string): string {
  if (lang !== 'zh') return prompt
  if (prompt === 'Your turn') return '你的回合'
  if (prompt === 'Choose an enemy') return '选择一个敌人'
  if (prompt === 'Choose a scheme') return '选择一个阴谋'
  return localizeLabel(prompt, undefined, lang, null)
}

export function useChoiceLabel(): (c: Choice) => string {
  const lang = useLang()
  const zh = useZhMap()
  return (c) => localizeLabel(c.label, c.cardCode, lang, zh)
}
