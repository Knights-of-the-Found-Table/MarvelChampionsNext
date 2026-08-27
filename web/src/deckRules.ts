// 组牌器的纯规则助手：例外卡匹配与派系容量。这里是服务端
// data.AspectException.Matches / engine.ValidateDeck 同一份结构化语义的
// TS 镜像——只读目录的结构化字段（traits/resources 都是服务端下发的英文
// 印刷数据，与引擎同源），绝不解析卡面文本。

import type { AspectException, CardInfo } from './api'

export const DECK_MIN = 40
export const DECK_MAX = 50

// 四个可选派系；「池」只对死侍开放，见 poolAllowed。
export const ASPECT_OPTIONS = ['aggression', 'justice', 'leadership', 'protection']
export const POOL = 'pool'

// 能进 40-50 牌组的类型（与引擎 deckCardType 对应：身份双面与责任卡除外）。
export function deckCardType(type: string): boolean {
  return (
    type === 'ally' ||
    type === 'event' ||
    type === 'support' ||
    type === 'upgrade' ||
    type === 'resource' ||
    type === 'player_side_scheme'
  )
}

// data.AspectException.Matches 的镜像：卡牌是否命中该骑手豁免。
export function exceptionMatches(x: AspectException | undefined, c: CardInfo): boolean {
  if (!x) return false
  if (x.cardType && c.type !== x.cardType) return false
  if (x.trait && !(c.traits ?? []).some((tr) => tr.toLowerCase() === x.trait!.toLowerCase())) {
    return false
  }
  if (x.eventTraits?.length) {
    if (c.type !== 'event') return false
    const hit = x.eventTraits.some((et) =>
      (c.traits ?? []).some((tr) => tr.toLowerCase() === et.toLowerCase()),
    )
    if (!hit) return false
  }
  if (x.energyEvents && (c.type !== 'event' || !(c.resources ?? []).includes('energy'))) {
    return false
  }
  return true
}

// 派系容量：""（常规）= 1；蜘蛛女 two_equal = 2；魔士亚当 four_equal = 4（固定全选）。
export function aspectCapacity(mode: string | undefined): number {
  if (mode === 'four_equal') return 4
  if (mode === 'two_equal') return 2
  return 1
}
