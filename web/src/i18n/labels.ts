// 局内文案（日志/提示/选项标签/败因）的结构化渲染层。
//
// 服务端是语言中立的：它下发 消息键 + 结构化参数（{k:"card", code} 表卡名
// 引用、{k:"i"} 表整数、{k:"msg"} 表嵌套消息），由这里按当前语言渲染：
//   1. GET /api/v1/locales/{lang} 取格式串目录（服务端 Go 目录为唯一事实源）；
//   2. 卡名参数按 code 查当前语言译名（zh 用 zh-cards.json，en 用服务端
//      内嵌的 en 名兜底），正对应「{field: card_name, card_code}」的约定；
//   3. 目录未命中或缺参时回退服务端 en 文本（gettext 缺译惯例）。
// 绝不做「拿英文成品文本再匹配翻译」——服务端已删除该方案，前端同样禁止。
import { useEffect, useState } from 'react'
import { get, type ArgWire, type Choice, type MsgWire } from '../api'
import { useLang, useZhMap, type Lang } from '../i18n'

type Catalog = Record<string, string>

const catalogs: Record<Lang, Catalog | null> = { en: null, zh: null }
const catalogPromises: Record<Lang, Promise<Catalog | null> | null> = { en: null, zh: null }
const catalogListeners = new Set<() => void>()

export function loadCatalog(lang: Lang): Promise<Catalog | null> {
  if (catalogs[lang]) return Promise.resolve(catalogs[lang])
  if (!catalogPromises[lang]) {
    catalogPromises[lang] = get<Catalog>(`/api/v1/locales/${lang}`)
      .then((c) => {
        catalogPromises[lang] = null
        if (c) {
          catalogs[lang] = c
          for (const l of catalogListeners) l()
        }
        return catalogs[lang]
      })
      .catch(() => {
        catalogPromises[lang] = null
        return null
      })
  }
  return catalogPromises[lang]
}

// ---- 动词匹配: %s %d %% 及 Go 显式序号 %[1]s ------------------------------

const VERB = /%(?:\[(\d+)\])?([a-zA-Z%])/g

// ---- 消息渲染 -------------------------------------------------------------

export function asMsg(m: MsgWire | string | undefined | null): MsgWire {
  if (typeof m === 'string') return { text: m }
  return m ?? { text: '' }
}

// cardName 按当前语言解析卡名：zh 查译名表，en/缺译回退 arg.s。
type CardNamer = (code: string, fallback: string) => string

// 消息渲染的中间形态：文本片段、卡名引用（code 可悬浮出卡图预览）或嵌套
// 消息。formatMsg 把它拼成字符串；React 侧（MsgText 组件）按片段类型渲染，
// 卡名片段带 hover 卡图。
export type MsgPart = { t: string } | { card: { code: string; name: string } } | { msg: MsgWire }

function argPart(a: ArgWire | undefined, lang: Lang, cardName: CardNamer): MsgPart {
  if (!a) return { t: '' }
  if (a.k === 'i') return { t: String(a.i ?? 0) }
  if (a.k === 'card') return { card: { code: a.code ?? '', name: cardName(a.code ?? '', a.s ?? '') } }
  if (a.k === 'msg' && a.msg) return { msg: a.msg }
  return { t: a.s ?? '' }
}

function msgParts(m: MsgWire, lang: Lang, cardName: CardNamer): MsgPart[] {
  const fmt = m.key ? catalogs[lang]?.[m.key] : undefined
  if (!fmt) return [{ t: m.text }]
  const args = m.args ?? []
  const parts: MsgPart[] = []
  let next = 0
  let last = 0
  for (const match of fmt.matchAll(VERB)) {
    const idx = match.index ?? 0
    if (idx > last) parts.push({ t: fmt.slice(last, idx) })
    last = idx + match[0].length
    if (match[2] === '%') {
      parts.push({ t: '%' })
      continue
    }
    const explicit = match[1]
    const n = explicit ? parseInt(explicit, 10) - 1 : next++
    // 缺参时保留动词原文（与 sprintf 的回退一致），多出的参数忽略。
    parts.push(args[n] !== undefined ? argPart(args[n], lang, cardName) : { t: match[0] })
  }
  if (last < fmt.length) parts.push({ t: fmt.slice(last) })
  return parts
}

export function formatMsg(m: MsgWire, lang: Lang, cardName: CardNamer): string {
  return msgParts(m, lang, cardName)
    .map((p) => ('t' in p ? p.t : 'card' in p ? p.card.name : formatMsg(p.msg, lang, cardName)))
    .join('')
}

// ---- React 绑定 -----------------------------------------------------------

// 目录异步加载；未就绪时调用方先渲染 en 兜底文本，加载完成后由此触发重渲染。
function useCatalogTick(): void {
  const lang = useLang()
  const [, force] = useState(0)
  useEffect(() => {
    let alive = true
    const bump = () => alive && force((n) => n + 1)
    catalogListeners.add(bump)
    loadCatalog(lang)
    return () => {
      alive = false
      catalogListeners.delete(bump)
    }
  }, [lang])
}

function useCardNamer(): CardNamer {
  const zhMap = useZhMap()
  return (code, fallback) => zhMap?.[code]?.name ?? fallback
}

// useEngineMsg 返回 (m) => string：按当前语言渲染结构化消息。
export function useEngineMsg(): (m: MsgWire | string | undefined | null) => string {
  const lang = useLang()
  const cardName = useCardNamer()
  useCatalogTick()
  return (m) => formatMsg(asMsg(m), lang, cardName)
}

// useMsgParts 返回 (m) => MsgPart[]：与 useEngineMsg 同源，但保留卡名片段
// 的结构，供 React 侧（MsgText）给卡名接上 hover 卡图预览。
export function useMsgParts(): (m: MsgWire | string | undefined | null) => MsgPart[] {
  const lang = useLang()
  const cardName = useCardNamer()
  useCatalogTick()
  return (m) => msgParts(asMsg(m), lang, cardName)
}

export function useChoiceLabel(): (c: Choice) => string {
  const em = useEngineMsg()
  return (c) => em(c.label)
}
