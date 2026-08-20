// 页面语言（英语 / 简体中文）：默认跟随浏览器语言，选择持久化到
// localStorage，顶栏可切换。中文时把服务端下发的英文卡名覆盖为译文
// （tools/zh 工具链产出的 zh-cards.json），并让卡图走 zh 图片路由。

import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import zhCardsJson from './i18n/zh-cards.json'
import { en, zh, type MsgKey } from './i18n/strings'

export type Lang = 'en' | 'zh'

const LANG_KEY = 'lang'

export function getInitialLang(): Lang {
  const stored = localStorage.getItem(LANG_KEY)
  if (stored === 'en' || stored === 'zh') return stored
  return typeof navigator !== 'undefined' && navigator.language.toLowerCase().startsWith('zh')
    ? 'zh'
    : 'en'
}

interface LangCtx {
  lang: Lang
  setLang: (l: Lang) => void
}

const LangContext = createContext<LangCtx>({ lang: 'en', setLang: () => {} })

export function LangProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(getInitialLang)
  useEffect(() => {
    document.documentElement.lang = lang
    document.title = lang === 'zh' ? zh.brand : en.brand
  }, [lang])
  return (
    <LangContext.Provider
      value={{
        lang,
        setLang: (l) => {
          setLangState(l)
          localStorage.setItem(LANG_KEY, l)
        },
      }}
    >
      {children}
    </LangContext.Provider>
  )
}

export function useLang(): Lang {
  return useContext(LangContext).lang
}

export function useSetLang() {
  return useContext(LangContext).setLang
}

type Params = Record<string, string | number>

// t(key, params?) 取当前语言的 UI 字符串，{name} 占位符用 params 替换。
// 键可以是任意字符串：字典里没有的键原样返回（沿用原先
// `LABEL[key] ?? key` 的兜底写法，动态键如 type/aspect/res 也用得上）。
// zh 字典仍由 Record<MsgKey, string> 在编译期保证覆盖全部 en 键。
// 消费 context，语言切换时使用它的组件会重渲染。
export function useT() {
  const { lang } = useContext(LangContext)
  return (key: string, params?: Params): string => {
    let s = (lang === 'zh' ? zh : en)[key as MsgKey]
    if (s === undefined) s = key
    if (params) s = s.replace(/\{(\w+)\}/g, (m, k) => (k in params ? String(params[k]) : m))
    return s
  }
}

// ---- 中文卡名覆盖 -------------------------------------------------------
// 服务端始终下发英文卡名；zh 模式下按卡牌 code 查译文，查不到就回退英文。

export interface ZhCard {
  name: string
  subname?: string
}

const zhCards = zhCardsJson as Record<string, ZhCard>

// 返回当前语言的译名表；en 模式下为 null。配合 lname/lsubname 使用，
// 避免在 JSX 的 .map() 回调里调用 hook。
export function useZhMap(): Record<string, ZhCard> | null {
  return useLang() === 'zh' ? zhCards : null
}

export function lname(zhMap: Record<string, ZhCard> | null, code: string, fallback: string): string {
  return zhMap?.[code]?.name ?? fallback
}

export function lsubname(
  zhMap: Record<string, ZhCard> | null,
  code: string,
  fallback?: string
): string | undefined {
  return zhMap?.[code]?.subname ?? fallback
}
