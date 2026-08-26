// UI 文案目录的运行时装载器。数据唯一来源是服务端 /locales 分发路由
// （服务端从仓库根 i18n/messages.json 导出，与引擎文案同源）：
//   1. GET /api/v1/locales/manifest 取各语言的内容 hash（no-cache 每次校验）；
//   2. GET /api/v1/locales/{lang}/{hash} 按内容寻址拉目录——URL 不变即内容
//      不变（immutable 长缓存），hash 失配时服务端 404，这里换新 hash 自愈。
// LangProvider 用 ensureCatalog 做就绪门控；装载完成后组件经 uiCatalog()
// 同步读字典。
import { get } from '../api'

export type Lang = 'en' | 'zh'

interface LocaleManifest {
  en: string
  zh: string
}

const catalogs: Record<Lang, Record<string, string> | null> = { en: null, zh: null }
const inflight = new Map<Lang, Promise<void>>()
let manifestPromise: Promise<LocaleManifest> | null = null

function fetchManifest(): Promise<LocaleManifest> {
  manifestPromise ??= get<LocaleManifest>('/locales/manifest').catch((e) => {
    manifestPromise = null // 失败允许重试
    throw e
  })
  return manifestPromise
}

// ensureCatalog 装载某语言目录并缓存（含失败重试）。返回共享 Promise，
// 多处并发调用只发一次请求。
export function ensureCatalog(lang: Lang): Promise<void> {
  const p = inflight.get(lang)
  if (p) return p
  const task = (async () => {
    const manifest = await fetchManifest()
    const h = manifest[lang]
    if (!h) throw new Error(`locale manifest lacks ${lang}`)
    catalogs[lang] = await get<Record<string, string>>(`/locales/${lang}/${h}`)
  })()
  task.catch(() => inflight.delete(lang)) // 失败后可重试
  inflight.set(lang, task)
  return task
}

// uiCatalog 返回已装载的目录（未装载为 null）。LangProvider 门控保证
// 子树渲染时当前语言一定就绪。
export function uiCatalog(lang: Lang): Record<string, string> | null {
  return catalogs[lang]
}
