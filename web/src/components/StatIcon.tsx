// 印刷属性小图标：按实体卡牌的图记语言手绘的内联 SVG——化解/攻击/防御/
// 恢复共用卡面上那种多尖角的漫画爆炸形，仅颜色不同（THW 蓝 / ATK 红 /
// DEF 绿 / REC 金）；手牌是扇形展开的三张玩家卡背（玩家卡背印象色
// #00548F）；生命直接复用对局的伤害代币图形（DamageTokenArt）。纯展示
// 组件：数字由调用方放在图标旁边。
import { useId } from 'react'
import { DamageTokenArt } from './GameCard'

const WHITE = '#fff'

// 14 尖角的漫画爆炸形，尖长身短、轻微不规则并整体斜置（生成脚本一次性
// 算出）；miter 连接保住尖角不被白描边磨圆。
const BURST =
  'M39.7 17.2 L31.3 21.0 L37.6 25.6 L28.5 25.0 L34.5 33.7 L26.3 29.4 L26.6 37.3 L21.4 29.8 L18.3 39.8 L16.6 30.8 L10.6 35.9 L13.2 27.2 L3.4 31.0 L9.4 24.0 L1.7 22.6 L10.1 19.2 L1.0 13.9 L10.3 14.3 L6.6 7.3 L14.5 11.8 L12.9 1.4 L18.4 8.8 L21.6 1.6 L23.0 10.6 L30.1 2.9 L27.8 11.8 L35.4 9.7 L29.3 16.5 Z'

const STAT_COLORS: Record<string, string | undefined> = {
  thwart: '#0e76bc',
  attack: '#d22b28',
  defense: '#1ea44e',
  recover: '#eeae27',
}

export const PLAYER_CARD_BLUE = '#00548F'

function StatShapes({ stat, gid }: { stat: string; gid: string }) {
  switch (stat) {
    case 'thwart':
    case 'attack':
    case 'defense':
    case 'recover': {
      return (
        <path
          d={BURST}
          fill={STAT_COLORS[stat]}
          stroke={WHITE}
          strokeWidth={3.8}
          strokeLinejoin="miter"
          paintOrder="stroke"
        />
      )
    }
    case 'hand': {
      // 玩家卡背扇面：左右两张斜靠、中间前卡压下；细浅框是卡背印花示意。
      const card = ({ x, y, rot }: { x: number; y: number; rot: number }) => (
        <g key={`${x}.${y}`} transform={`translate(${x} ${y}) rotate(${rot})`}>
          <rect
            x={-6.2}
            y={-8.8}
            width={12.4}
            height={17.6}
            rx={2}
            fill={PLAYER_CARD_BLUE}
            stroke={WHITE}
            strokeWidth={3.6}
            strokeLinejoin="round"
            paintOrder="stroke"
          />
          <rect
            x={-4.7}
            y={-7.2}
            width={9.4}
            height={14.4}
            rx={1}
            fill="none"
            stroke="#4d88bf"
            strokeWidth={1.3}
          />
        </g>
      )
      return (
        <>
          {card({ x: 11, y: 13.5, rot: -30 })}
          {card({ x: 29, y: 13.5, rot: 30 })}
          {card({ x: 20, y: 22, rot: 0 })}
        </>
      )
    }
    case 'hp': {
      // 复用对局的伤害代币：黑边圆牌 + 炽橙渐变 + 网点。
      return <DamageTokenArt gid={gid} />
    }
    default:
      return null
  }
}

export function StatIcon({ stat, size = 14 }: { stat: string; size?: number }) {
  // 渐变/网点 id 每实例唯一：同屏多枚图标共享 defs 命名空间。
  const gid = `st-${useId().replace(/[^a-zA-Z0-9]/g, '')}`
  return (
    <svg className="res-icon" width={size} height={size} viewBox="0 0 40 40" aria-hidden="true">
      <defs>
        <radialGradient id={gid} cx="50%" cy="40%" r="68%">
          <stop offset="0%" stopColor="#ffce4d" />
          <stop offset="45%" stopColor="#ff8c1f" />
          <stop offset="80%" stopColor="#e63a1c" />
          <stop offset="100%" stopColor="#7e1007" />
        </radialGradient>
        <pattern id={`${gid}-d`} width="3.6" height="3.6" patternUnits="userSpaceOnUse">
          <circle cx="1.8" cy="1.8" r="0.85" fill="rgba(64,7,2,0.42)" />
        </pattern>
      </defs>
      <StatShapes stat={stat} gid={gid} />
    </svg>
  )
}
