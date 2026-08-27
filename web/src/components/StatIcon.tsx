// 印刷属性小图标：按实体卡牌的图记语言手绘的内联 SVG——化解/攻击/防御/
// 恢复共用同一个漫画式爆炸形，仅颜色不同（THW 蓝 / ATK 红 / DEF 绿 /
// REC 金）；手牌是扇形展开的三张玩家卡背（玩家卡背印象色 #00548F）；生
// 命复用伤害代币的黑灰泼溅形并加红点示意。纯展示组件：数字由调用方放在
// 图标旁边，图形按 14–16px 的实际显示尺寸简化，不做印刷版的内芯纹理。
const WHITE = '#fff'

// 12 尖角的漫画爆炸形，轻微不规则（生成脚本一次性算出）。
export const BURST =
  'M20.0 1.7 L23.2 7.9 L28.2 5.7 L27.6 12.4 L35.8 10.9 L32.1 16.8 L36.5 20.0 L30.3 22.8 L35.8 29.1 L28.8 28.8 L28.2 34.3 L22.8 30.3 L20.0 38.3 L16.8 32.1 L11.8 34.3 L12.4 27.6 L4.2 29.2 L7.9 23.2 L3.5 20.0 L9.7 17.2 L4.2 10.8 L11.2 11.2 L11.7 5.7 L17.2 9.7 Z'

const STAT_COLORS: Record<string, string | undefined> = {
  thwart: '#0e76bc',
  attack: '#d22b28',
  defense: '#1ea44e',
  recover: '#eeae27',
}

export const PLAYER_CARD_BLUE = '#00548F'

function StatShapes({ stat }: { stat: string }) {
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
          strokeWidth={4.6}
          strokeLinejoin="round"
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
      // 伤害代币：黑灰不规则泼溅 + 右下红色血点。
      return (
        <>
          <path
            d='M20.0 0.8 L25.0 9.5 L29.5 8.6 L29.9 17.2 L38.6 17.8 L30.7 25.7 L31.9 30.7 L24.0 29.8 L19.1 39.0 L16.4 31.4 L8.1 29.4 L10.9 25.1 L0.8 18.1 L8.7 16.1 L8.5 8.3 L16.3 9.9 Z'
            fill="#49505c"
            stroke={WHITE}
            strokeWidth={4}
            strokeLinejoin="round"
            paintOrder="stroke"
            transform="translate(20 19.6) scale(0.92) translate(-20 -19.6)"
          />
          <circle cx={30.8} cy={31.4} r={4} fill="#ff5f4d" stroke={WHITE} strokeWidth={2.6} paintOrder="stroke" />
        </>
      )
    }
    default:
      return null
  }
}

export function StatIcon({ stat, size = 14 }: { stat: string; size?: number }) {
  return (
    <svg className="res-icon" width={size} height={size} viewBox="0 0 40 40" aria-hidden="true">
      <StatShapes stat={stat} />
    </svg>
  )
}
