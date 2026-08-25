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

// ---- sprintf: %s %d %% 及 Go 显式序号 %[1]s ------------------------------

const VERB = /%(?:\[(\d+)\])?([a-zA-Z%])/g

function sprintf(format: string, args: string[]): string {
  let next = 0
  return format.replace(VERB, (whole, idx: string | undefined, verb: string) => {
    if (verb === '%') return '%'
    const n = idx ? parseInt(idx, 10) - 1 : next++
    const a = args[n]
    return a === undefined ? whole : a
  })
}

// ---- 消息渲染 -------------------------------------------------------------

export function asMsg(m: MsgWire | string | undefined | null): MsgWire {
  if (typeof m === 'string') return { text: m }
  return m ?? { text: '' }
}

// cardName 按当前语言解析卡名：zh 查译名表，en/缺译回退 arg.s。
type CardNamer = (code: string, fallback: string) => string

function renderArg(a: ArgWire, lang: Lang, cardName: CardNamer): string {
  if (a.k === 'i') return String(a.i ?? 0)
  if (a.k === 'card') return cardName(a.code ?? '', a.s ?? '')
  if (a.k === 'msg' && a.msg) return formatMsg(a.msg, lang, cardName)
  return a.s ?? ''
}

export function formatMsg(m: MsgWire, lang: Lang, cardName: CardNamer): string {
  const fmt = m.key ? catalogs[lang]?.[m.key] : undefined
  if (!fmt) return m.text
  const args = (m.args ?? []).map((a) => renderArg(a, lang, cardName))
  return sprintf(fmt, args)
}

// ---- React 绑定 -----------------------------------------------------------

// useEngineMsg 返回 (m) => string：按当前语言渲染结构化消息。目录异步加载，
// 未就绪时先渲染 en 兜底文本，加载完成后触发重渲染。
export function useEngineMsg(): (m: MsgWire | string | undefined | null) => string {
  const lang = useLang()
  const zhMap = useZhMap()
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
  const cardName: CardNamer = (code, fallback) => zhMap?.[code]?.name ?? fallback
  return (m) => formatMsg(asMsg(m), lang, cardName)
}

export function useChoiceLabel(): (c: Choice) => string {
  const em = useEngineMsg()
  return (c) => em(c.label)
}
