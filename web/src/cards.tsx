// Card image resolution with content-hash cache busting. The manifest is
// generated at image-fetch time (docker build or local dev script).

import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useLang, type Lang } from './i18n'

interface Manifest {
  [code: string]: string
}

// 每个语言一份 manifest：zh 走 /img/cards/zh/（未种图的卡由服务端回退
// 英文卡面），en 走原有路由。manifestLoaded 区分「未加载」与「已加载但
// 为空」（zh 未种图时 manifest 是 404），避免每次挂载重复请求。
let manifests: Record<Lang, Manifest | null> = { en: null, zh: null }
let manifestLoaded: Record<Lang, boolean> = { en: false, zh: false }
let manifestPromises: Record<Lang, Promise<Manifest | null> | null> = { en: null, zh: null }

async function loadManifest(lang: Lang): Promise<Manifest | null> {
  if (manifestLoaded[lang]) return manifests[lang]
  if (manifestPromises[lang]) return manifestPromises[lang]
  manifestPromises[lang] = fetch(
    lang === 'zh' ? '/img/cards/zh/manifest.json' : '/img/cards/manifest.json'
  )
    .then((r) => (r.ok ? (r.json() as Promise<Manifest>) : null))
    .catch(() => null)
  manifests[lang] = await manifestPromises[lang]
  manifestLoaded[lang] = true
  return manifests[lang]
}

export function cardUrl(code: string, lang: Lang, hash?: string): string {
  const h = hash ?? manifests[lang]?.[code]
  const prefix = lang === 'zh' ? '/img/cards/zh/' : '/img/cards/'
  return h ? `${prefix}${code}.${h}.png` : `${prefix}${code}.png`
}

export async function preloadManifest(lang: Lang): Promise<void> {
  await loadManifest(lang)
}

// ---- hover zoom preview -----------------------------------------------------
// Modeled on arkhamhorror.app's CardOverlay: hovering a card shows a large,
// pointer-transparent copy fixed beside it. Placed right of the card, flipped
// to the left when it would overflow the viewport, and clamped vertically.
const ZOOM_W = 300
const ZOOM_H = 420 // 5:7 card aspect
const ZOOM_GAP = 10
const ZOOM_PAD = 10
const ZOOM_DELAY_MS = 100
const coarsePointer =
  typeof window.matchMedia === 'function' &&
  window.matchMedia('(hover: none) and (pointer: coarse)').matches

export function useCardZoom(code: string, anchorRef: React.RefObject<HTMLElement | null>) {
  const lang = useLang()
  const [visible, setVisible] = useState(false)
  const [pos, setPos] = useState({ top: 0, left: 0 })
  const timer = useRef<number | null>(null)

  function position() {
    const rect = anchorRef.current!.getBoundingClientRect()
    let left = rect.right + ZOOM_GAP
    if (left + ZOOM_W > window.innerWidth - ZOOM_PAD) left = rect.left - ZOOM_W - ZOOM_GAP
    left = Math.max(ZOOM_PAD, Math.min(left, window.innerWidth - ZOOM_W - ZOOM_PAD))
    const top = Math.max(ZOOM_PAD, Math.min(rect.top - 40, window.innerHeight - ZOOM_H - ZOOM_PAD))
    return { top, left }
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
      setPos(position())
      setVisible(true)
    }, ZOOM_DELAY_MS)
  }

  function hide() {
    clearTimer()
    setVisible(false)
  }

  // Keep the preview glued to the card while the page scrolls or resizes.
  useEffect(() => {
    if (!visible) return
    const track = () => setPos(position())
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
        <div className="card-zoom" style={pos}>
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

  return { onEnter, hide, overlay }
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
  // 语言切换后加载该语言的 manifest（模块级缓存），就绪后切到带
  // 内容哈希的 URL；manifest 未就绪前用无哈希 URL，服务端照常出图。
  useEffect(() => {
    let alive = true
    loadManifest(lang).then(() => {
      if (alive) setTick((t) => t + 1)
    })
    return () => {
      alive = false
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

function fallbackDataUrl(code: string): string {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="150" height="210">
    <rect width="100%" height="100%" rx="8" fill="#222"/>
    <text x="50%" y="50%" fill="#888" font-family="monospace" font-size="14" text-anchor="middle">${code}</text>
  </svg>`
  return 'data:image/svg+xml;utf8,' + encodeURIComponent(svg)
}
