// Card image resolution with content-hash cache busting. The manifest is
// generated at image-fetch time (docker build or local dev script).

import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'

interface Manifest {
  [code: string]: string
}

let manifest: Manifest | null = null
let manifestPromise: Promise<Manifest | null> | null = null

async function loadManifest(): Promise<Manifest | null> {
  if (manifest) return manifest
  if (manifestPromise) return manifestPromise
  manifestPromise = fetch('/img/cards/manifest.json')
    .then((r) => (r.ok ? (r.json() as Promise<Manifest>) : null))
    .catch(() => null)
  manifest = await manifestPromise
  return manifest
}

export function cardUrl(code: string, hash?: string): string {
  const h = hash ?? manifest?.[code]
  return h ? `/img/cards/${code}.${h}.png` : `/img/cards/${code}.png`
}

export async function preloadManifest(): Promise<void> {
  await loadManifest()
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
            src={cardUrl(code)}
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
  const zoom = useCardZoom(code, imgRef)
  return (
    <>
      <img
        ref={imgRef}
        className={className}
        src={cardUrl(code)}
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
