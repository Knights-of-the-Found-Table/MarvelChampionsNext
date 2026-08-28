// 组牌器的纯规则助手：例外卡匹配与派系容量。这里是服务端
// data.AspectException.Matches / engine.ValidateDeck 同一份结构化语义的
// TS 镜像——只读目录的结构化字段（例外匹配用英文印刷特征 etraits；
// resources 是与语言无关的图标；zh 覆盖部署下 traits 是译文，不参与
// 判断），绝不解析卡面文本。

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

// data.HasTrait/normTrait 的镜像：小写 + 去掉一个结尾句点（"S.H.I.E.L.D."
// 的骑手写作 "s.h.i.e.l.d."，而解析后的特征列表不带结尾句点）。
function normTrait(s: string): string {
  return s.trim().toLowerCase().replace(/\.$/, '')
}

function hasTrait(c: CardInfo, trait: string): boolean {
  const t = normTrait(trait)
  // 匹配优先用英文印刷特征 etraits；zh 覆盖部署下 traits 是译文，
  // 绝不可能等于英文骑手特征，故仅作兜底。
  return [...(c.etraits ?? []), ...(c.traits ?? [])].some((tr) => normTrait(tr) === t)
}

// data.AspectException.Matches 的镜像：卡牌是否命中该骑手豁免。
export function exceptionMatches(x: AspectException | undefined, c: CardInfo): boolean {
  if (!x) return false
  if (x.cardType && c.type !== x.cardType) return false
  if (x.trait && !hasTrait(c, x.trait)) {
    return false
  }
  if (x.eventTraits?.length) {
    if (c.type !== 'event') return false
    const hit = x.eventTraits.some((et) => hasTrait(c, et))
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
