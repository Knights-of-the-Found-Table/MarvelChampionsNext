// 资源代币图标：按实体代币的贴纸风格手绘的内联 SVG——彩色主体 + 同色系
// 深色内细节 + 白外描边 + 硬投影，与桌面代币同色。纯展示组件：牌组详情
// 资源点、支付跟踪条，以及日志/提示/卡牌文本里的 [energy] 类印刷标记
// （ResText）共用；未知图标回落灰色菱形。
import type { ReactNode } from 'react'

const RES_COLORS: Record<string, string> = {
  energy: '#e6b848',
  physical: '#d78b71',
  mental: '#b7c5e6',
  wild: '#4fb269',
}

// 内部细节的深色（同色系阴影色）：闪电内芯、圆环/连接管、指缝线、内星。
const RES_DARK: Record<string, string> = {
  energy: '#7b6a20',
  physical: '#7e4023',
  mental: '#5b74a7',
  wild: '#1f7a45',
}

const WILD_STAR =
  'M20 3.5 24.5 12.7 34.7 12 29 20.5 34.7 29 24.5 28.3 20 37.5 15.5 28.3 5.3 29 11 20.5 5.3 12 15.5 12.7Z'

function Shapes({ icon }: { icon: string }) {
  const dark = RES_DARK[icon]
  switch (icon) {
    case 'energy':
      return (
        <>
          <path d="M27.5 3 8 23h8.5L13 37 32 16.5h-9L27.5 3Z" />
          <path d="M22.1 11.9 13.5 20.7h6l-2 8 9.2-9.9h-6.9Z" fill={dark} stroke="none" />
        </>
      )
    case 'mental':
      return (
        <>
          {/* 白色底形先铺出整团 union 描边，再铺深色管/环与彩色芯 */}
          <g stroke="#fff" strokeWidth={8.4} strokeLinecap="round" fill="none">
            <path d="M11.3 10 11 29.8" />
            <path d="M11.3 10 26.9 23.5" />
          </g>
          <g fill="#fff" stroke="none">
            <circle cx={11.3} cy={10} r={8.5} />
            <circle cx={11} cy={29.8} r={6} />
            <circle cx={26.9} cy={23.5} r={10.1} />
          </g>
          <g stroke={dark} strokeWidth={3.8} strokeLinecap="round" fill="none">
            <path d="M11.3 10 11 29.8" />
            <path d="M11.3 10 26.9 23.5" />
          </g>
          <g stroke="currentColor" strokeWidth={1.7} strokeLinecap="round" fill="none">
            <path d="M11.3 10 11 29.8" />
            <path d="M11.3 10 26.9 23.5" />
          </g>
          <g fill="currentColor" stroke={dark} strokeWidth={1.8}>
            <circle cx={11.3} cy={10} r={6.2} />
            <circle cx={11} cy={29.8} r={3.7} />
            <circle cx={26.9} cy={23.5} r={7.8} />
          </g>
        </>
      )
    case 'physical':
      return (
        <g transform="translate(20.5 22.75) scale(0.95) rotate(-26) translate(-20.5 -22.75)">
          {/* 拳面：四指关节圆弧 + 圆角掌底；深色指缝线与拇指折痕 */}
          <path d="M7.5 29.5V16.5a3.25 3.25 0 0 1 6.5 0 3.25 3.25 0 0 1 6.5 0 3.25 3.25 0 0 1 6.5 0 3.25 3.25 0 0 1 6.5 0v13q0 3-3 3H10.5q-3 0-3-3Z" />
          <g stroke={dark} strokeWidth={2} strokeLinecap="round" fill="none">
            <path d="M14 17.6q-.2 3.9.1 7.6" />
            <path d="M20.5 17.6q-.2 3.9.1 7.6" />
            <path d="M27 17.6q-.2 3.9.1 7.6" />
            <path d="M17.2 26.6q6-1.6 11.4.9" />
            <path d="M28.4 27.1q1.6 1.5 1 3.9" />
          </g>
        </g>
      )
    case 'wild':
      return (
        <>
          <path d={WILD_STAR} />
          <path d={WILD_STAR} fill={dark} stroke="none" transform="translate(20 20.5) scale(0.72) translate(-20 -20.5)" />
        </>
      )
    default:
      return null
  }
}

export function ResourceIcon({ icon, size = 14, title }: { icon: string; size?: number; title?: string }) {
  const color = RES_COLORS[icon]
  return (
    <svg
      className="res-icon"
      width={size}
      height={size}
      viewBox="0 0 40 40"
      role={title ? 'img' : undefined}
      aria-hidden={title ? undefined : true}
      style={color ? { color } : undefined}
    >
      {title ? <title>{title}</title> : null}
      <g
        fill={color ?? '#8b98a8'}
        stroke="#fff"
        strokeWidth={6}
        strokeLinejoin="round"
        strokeLinecap="round"
        paintOrder="stroke"
      >
        {color ? <Shapes icon={icon} /> : <path d="M20 6 32.5 20 20 34 7.5 20Z" />}
      </g>
    </svg>
  )
}

// 把成品文本里的资源印刷标记替换成内联图标。两种印刷形态都处理：
// 单标记「[energy]」（卡牌文本）与合写「[energy energy]」（支付标签）。
// 仅做展示层装饰：输入是什么文本由调用方决定（日志片段、卡牌印刷文本
// 等），这里不参与任何游戏逻辑。
const RES_TOKEN_RE = /\[((?:energy|physical|mental|wild)(?: +(?:energy|physical|mental|wild))*)\]/g

export function ResText({ text, size = 13 }: { text: string; size?: number }) {
  if (!text.includes('[')) return <>{text}</>
  const parts = text.split(RES_TOKEN_RE)
  if (parts.length === 1) return <>{text}</>
  const out: ReactNode[] = []
  for (let i = 0; i < parts.length; i++) {
    if (i % 2) {
      for (const icon of parts[i].split(/\s+/)) {
        out.push(<ResourceIcon key={`${i}.${icon}`} icon={icon} size={size} />)
      }
    } else if (parts[i]) out.push(parts[i])
  }
  return <>{out}</>
}
