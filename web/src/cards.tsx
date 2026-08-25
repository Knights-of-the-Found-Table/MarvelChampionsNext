// Card image resolution with content-hash cache busting. The server
// exposes {code: hash} manifests (/img/cards/manifest.json, zh variant);
// URLs that carry the hash are immutable, so browsers cache them forever.
// The manifest fills up as the server caches images (background prewarm,
// on-demand fetches) and is revalidated here every 30s via ETag, so links
// converge on hashed URLs without a page reload.

import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useLang, type Lang } from './i18n'

interface Manifest {
  [code: string]: string
}

// 每个语言一份 manifest：zh 走 /img/cards/zh/（未种图的卡由服务端用主缓存
// 供图），en 走原有路由。manifestLoaded 区分「未加载」与「已加载但为空」，
// 避免每次挂载重复请求。
let manifests: Record<Lang, Manifest | null> = { en: null, zh: null }
let manifestLoaded: Record<Lang, boolean> = { en: false, zh: false }
let manifestPromises: Record<Lang, Promise<Manifest | null> | null> = { en: null, zh: null }
let manifestEtags: Record<Lang, string | null> = { en: null, zh: null }
const manifestListeners = new Set<() => void>()

function notifyManifests() {
  for (const listener of manifestListeners) listener()
}

async function fetchManifest(lang: Lang): Promise<Manifest | null> {
  const headers: Record<string, string> = {}
  if (manifestEtags[lang]) headers['If-None-Match'] = manifestEtags[lang]
  const url = lang === 'zh' ? '/img/cards/zh/manifest.json' : '/img/cards/manifest.json'
  const r = await fetch(url, { headers })
  if (r.status === 304) return manifests[lang]
  if (!r.ok) return null
  manifestEtags[lang] = r.headers.get('ETag')
  return (await r.json()) as Manifest
}

async function loadManifest(lang: Lang, force = false): Promise<Manifest | null> {
  if (!force && manifestLoaded[lang]) return manifests[lang]
  if (!manifestPromises[lang]) {
    manifestPromises[lang] = fetchManifest(lang)
      .then((m) => {
        manifestPromises[lang] = null
        if (m !== null) {
          const changed = m !== manifests[lang]
          manifests[lang] = m
          manifestLoaded[lang] = true
          if (changed) notifyManifests()
        }
        return m
      })
      .catch(() => {
        manifestPromises[lang] = null
        return null
      })
  }
  return manifestPromises[lang]
}

// 服务端在缓存图片的过程中不断补全 manifest（后台预热、按需抓取），定时
// 用 ETag 再验证，让新出现的哈希无需整页刷新就能用上。
let refreshTimer: number | null = null

function startManifestRefresh() {
  if (refreshTimer !== null) return
  refreshTimer = window.setInterval(() => {
    if (document.hidden) return
    for (const lang of ['en', 'zh'] as Lang[]) {
      if (manifestLoaded[lang]) void loadManifest(lang, true)
    }
  }, 30_000)
}

// zh 路由对未种图的卡回落主缓存供图（同一张图），因此 zh manifest 缺哈希
// 时用 en manifest 的哈希同样正确——它指向的就是实际返回的字节。
function hashFor(code: string, lang: Lang): string | undefined {
  return manifests[lang]?.[code] ?? manifests.en?.[code]
}

export function cardUrl(code: string, lang: Lang, hash?: string): string {
  const h = hash ?? hashFor(code, lang)
  const prefix = lang === 'zh' ? '/img/cards/zh/' : '/img/cards/'
  return h ? `${prefix}${code}.${h}.png` : `${prefix}${code}.png`
}

export async function preloadManifest(lang: Lang): Promise<void> {
  startManifestRefresh()
  await loadManifest(lang)
}

// ---- hover zoom preview -----------------------------------------------------
// Hovering a card shows a large, pointer-transparent copy in a body portal.
// The preview owns the half of the viewport opposite the anchor, so it cannot
// be clipped by the scaled board or feed back into the card's hover hitbox.
const ZOOM_DELAY_MS = 100
const coarsePointer =
  typeof window.matchMedia === 'function' &&
  window.matchMedia('(hover: none) and (pointer: coarse)').matches

export function useCardZoom(code: string, anchorRef: React.RefObject<HTMLElement | null>) {
  const lang = useLang()
  const [visible, setVisible] = useState(false)
  const [side, setSide] = useState<'left' | 'right'>('right')
  const timer = useRef<number | null>(null)

  function position() {
    const rect = anchorRef.current?.getBoundingClientRect()
    if (!rect) return 'right' as const
    return rect.left + rect.width / 2 > window.innerWidth / 2 ? 'left' as const : 'right' as const
  }

  function clearTimer() {
    if (timer.current !== null) {
      clearTimeout(timer.current)
      timer.current = null
    }
  }

  function onEnter() {
    if (coarsePointer) return
    clearTimer()
    timer.current = window.setTimeout(() => {
      setSide(position())
      setVisible(true)
    }, ZOOM_DELAY_MS)
  }

  function show() {
    clearTimer()
    setSide(position())
    setVisible(true)
  }

  function hide() {
    clearTimer()
    setVisible(false)
  }

  // Keep the preview on the side opposite the card while the page moves.
  useEffect(() => {
    if (!visible) return
    const track = () => setSide(position())
    window.addEventListener('scroll', track, true)
    window.addEventListener('resize', track)
    return () => {
      window.removeEventListener('scroll', track, true)
      window.removeEventListener('resize', track)
    }
  }, [visible])

  useEffect(() => clearTimer, [])

  const overlay = visible
    ? createPortal(
        <div className={`card-zoom card-zoom-${side} ${coarsePointer ? 'card-zoom-touch' : ''}`} aria-hidden="true">
          <img
            src={cardUrl(code, lang)}
            alt=""
            onError={(e) => {
              const img = e.currentTarget
              if (!img.dataset.fallback) {
                img.dataset.fallback = '1'
                img.src = fallbackDataUrl(code)
              }
            }}
          />
        </div>,
        document.body
      )
    : null

  return { onEnter, show, hide, overlay }
}

// `zoom={false}` lets a parent row own the preview: it calls useCardZoom
// itself with a wrapper element as the anchor, so hovering anywhere in the
// row (not just the image) shows the overlay beside the thumbnail.
export function CardImage({
  code,
  size = 'md',
  className,
  zoom: zoomEnabled = true,
}: {
  code: string
  size?: 'xs' | 'sm' | 'md' | 'lg'
  className?: string
  zoom?: boolean
}) {
  const widths: Record<string, number> = { xs: 60, sm: 100, md: 160, lg: 220 }
  const imgRef = useRef<HTMLImageElement | null>(null)
  const lang = useLang()
  const [, setTick] = useState(0)
  const zoom = useCardZoom(code, imgRef)
  // 语言切换后加载该语言的 manifest（模块级缓存），就绪后切到带内容
  // 哈希的 URL；manifest 未就绪前用无哈希 URL，服务端照常出图。manifest
  // 有更新（预热补全哈希等）时通过订阅触发重渲染。
  useEffect(() => {
    let alive = true
    const bump = () => {
      if (alive) setTick((t) => t + 1)
    }
    startManifestRefresh()
    manifestListeners.add(bump)
    loadManifest(lang).then(bump)
    return () => {
      alive = false
      manifestListeners.delete(bump)
    }
  }, [lang])
  return (
    <>
      <img
        ref={imgRef}
        className={className}
        src={cardUrl(code, lang)}
        alt={code}
        width={widths[size]}
        loading="lazy"
        onMouseEnter={zoomEnabled ? zoom.onEnter : undefined}
        onMouseLeave={zoomEnabled ? zoom.hide : undefined}
        onError={(e) => {
          const img = e.currentTarget
          if (!img.dataset.fallback) {
            img.dataset.fallback = '1'
            img.src = fallbackDataUrl(code)
          }
        }}
      />
      {zoomEnabled ? zoom.overlay : null}
    </>
  )
}

export function fallbackDataUrl(code: string): string {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="150" height="210">
    <rect width="100%" height="100%" rx="8" fill="#222"/>
    <text x="50%" y="50%" fill="#888" font-family="monospace" font-size="14" text-anchor="middle">${code}</text>
  </svg>`
  return 'data:image/svg+xml;utf8,' + encodeURIComponent(svg)
}
