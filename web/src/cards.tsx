// Card image resolution with content-hash cache busting. The manifest is
// generated at image-fetch time (docker build or local dev script).

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
  return `/img/cards/${code}.png${h ? `?v=${h}` : ''}`
}

export async function preloadManifest(): Promise<void> {
  await loadManifest()
}

export function CardImage({
  code,
  size = 'md',
  className,
}: {
  code: string
  size?: 'xs' | 'sm' | 'md' | 'lg'
  className?: string
}) {
  const widths: Record<string, number> = { xs: 60, sm: 100, md: 160, lg: 220 }
  return (
    <img
      className={className}
      src={cardUrl(code)}
      alt={code}
      width={widths[size]}
      loading="lazy"
      onError={(e) => {
        const img = e.currentTarget
        if (!img.dataset.fallback) {
          img.dataset.fallback = '1'
          img.src = fallbackDataUrl(code)
        }
      }}
    />
  )
}

function fallbackDataUrl(code: string): string {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="150" height="210">
    <rect width="100%" height="100%" rx="8" fill="#222"/>
    <text x="50%" y="50%" fill="#888" font-family="monospace" font-size="14" text-anchor="middle">${code}</text>
  </svg>`
  return 'data:image/svg+xml;utf8,' + encodeURIComponent(svg)
}
