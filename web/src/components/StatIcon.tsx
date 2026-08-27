// 印刷属性小图标：按实体卡牌的图记语言手绘的内联 SVG——化解/攻击/防御/
// 恢复共用卡面上那种八角星形的大尖角爆炸（仅颜色不同：THW 蓝 / ATK 红 /
// DEF 绿 / REC 金）；手牌是扇形展开的三张玩家卡背（玩家卡背印象色
// #00548F）；生命直接复用对局的伤害代币图形（DamageTokenArt）。纯展示
// 组件：数字由调用方放在图标旁边。
import { useId } from 'react'
import { DamageTokenArt } from './GameCard'

const WHITE = '#fff'

// 八尖角的八角星形爆炸（卡面图记正是这种大尖角八角星，谷底深切、尖长
// 不一、整体轻微不正）（生成脚本一次性算出）；miter 连接保住尖角不被
// 白描边磨圆。
const BURST =
  'M19.9 1.2 L21.9 9.6 L26.1 12.2 L33.8 7.6 L27.4 15.2 L28.8 18.5 L38.9 20.0 L29.1 21.4 L28.0 25.4 L33.7 32.7 L26.3 28.2 L21.5 29.8 L19.7 39.4 L18.2 29.8 L15.0 27.8 L7.8 33.5 L11.5 26.4 L10.2 22.3 L0.6 21.0 L10.8 19.0 L12.9 14.6 L7.9 7.3 L14.8 12.0 L18.2 10.7 Z'

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
