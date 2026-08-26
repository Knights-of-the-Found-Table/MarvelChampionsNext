// 页面语言（英语 / 简体中文）：默认跟随浏览器语言，选择持久化到
// localStorage，顶栏可切换。
//
// 文案数据全部来自服务端 /locales 分发的共享目录（仓库根
// i18n/messages.json 的单一事实源）：LangProvider 用 ensureCatalog 做
// 就绪门控——当前语言目录装载完成前只渲染加载态；就绪后另一语言在后台
// 预取，切换语言即时、不再触发门控。中文时把服务端下发的英文卡名覆盖为
// 译文（tools/zh 工具链产出的 zh-cards.json），并让卡图走 zh 图片路由。

import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import zhCardsJson from './i18n/zh-cards.json'
import { ensureCatalog, uiCatalog, type Lang } from './i18n/catalog'
import { sprint } from './i18n/format'

export type { Lang }

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

function applyDocMeta(lang: Lang) {
  document.documentElement.lang = lang
  document.title = uiCatalog(lang)?.['brand'] ?? 'Marvel Champions'
}

export function LangProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(getInitialLang)
  // 门控态：readyLang === lang 才渲染子树。目录已在内存（例如切回已
  // 装载的语言）时同步放行，不闪加载态。
  const [readyLang, setReadyLang] = useState<Lang | null>(() => (uiCatalog(lang) ? lang : null))
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let alive = true
    const other: Lang = lang === 'zh' ? 'en' : 'zh'
    if (uiCatalog(lang)) {
      applyDocMeta(lang)
      setReadyLang(lang)
      setFailed(false)
      // 后台预取另一语言：切换语言即时生效。
      void ensureCatalog(other).catch(() => {})
      return
    }
    setReadyLang(null)
    setFailed(false)
    ensureCatalog(lang).then(
      () => {
        if (!alive) return
        applyDocMeta(lang)
        setReadyLang(lang)
        void ensureCatalog(other).catch(() => {})
      },
      () => {
        if (alive) setFailed(true)
      },
    )
    return () => {
      alive = false
    }
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
      {readyLang === lang ? (
        children
      ) : failed ? (
        <div className="boot-gate">
          {/* 引导期文案在目录可用之前展示，双语并列是引导页的合法硬编码。 */}
          <p>Failed to load text data / 文案数据加载失败</p>
          <button onClick={() => window.location.reload()}>Retry / 重试</button>
        </div>
      ) : (
        <div className="boot-gate" aria-busy="true" />
      )}
    </LangContext.Provider>
  )
}

export function useLang(): Lang {
  return useContext(LangContext).lang
}

export function useSetLang() {
  return useContext(LangContext).setLang
}

// t(key, ...args) 取当前语言的 UI 字符串。值里的 %s/%d 动词按位置填入
// args（与引擎消息同一套占位符语法，见 i18n/format.ts）。键可以是任意
// 字符串：动态键（`type.${key}` 等）与未命中键原样返回。
// 消费 context，语言切换时使用它的组件会重渲染；LangProvider 门控保证
// 子树渲染时目录已就绪。
export function useT() {
  const { lang } = useContext(LangContext)
  return (key: string, ...args: Array<string | number>): string => {
    const fmt = uiCatalog(lang)?.[key]
    if (fmt === undefined) return key
    return args.length > 0 ? sprint(fmt, args) : fmt
  }
}

// ---- 中文卡名覆盖 -------------------------------------------------------
// 服务端始终下发英文卡名；zh 模式下按卡牌 code 查译名表（构建期内嵌的
// 译名快照），查不到就回退英文。

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
