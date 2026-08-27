// 印刷属性小图标：与 ResourceIcon 同一套贴纸语言（彩色主体 + 白外描边 +
// 硬投影），但统一用钢蓝单色——这些是数值语义不是资源语义，避免与四色
// 资源代币混淆。六种：化解（英雄面具）/ 攻击（长剑）/ 防御（盾徽）/
// 恢复（急救十字）/ 手牌（扇形三张）/ 生命（心形）。纯展示组件。
const STAT_COLOR = '#93a4bb'
const STAT_DARK = '#55647a'

function StatShapes({ stat }: { stat: string }) {
  switch (stat) {
    case 'thwart': {
      // 半脸英雄面具：外轮廓带两侧系带缺口，深色双目孔
      return (
        <>
          <path d="M4.5 13.5q15.5-7.5 31 0l-.9 7.2q-.9 9.3-7.5 9.8l-4.4-3h-4.4l-4.4 3q-6.6-.5-7.5-9.8Z" />
          <g fill={STAT_DARK} stroke="none">
            <ellipse cx={13.7} cy={20.4} rx={3.6} ry={2.3} transform="rotate(-8 13.7 20.4)" />
            <ellipse cx={26.3} cy={20.4} rx={3.6} ry={2.3} transform="rotate(8 26.3 20.4)" />
          </g>
        </>
      )
    }
    case 'attack': {
      // 竖直长剑转 45°：剑身 + 护手 + 握柄 + 圆剑首，深色剑脊线
      return (
        <g transform="rotate(45 20 20)">
          <path d="M20 3.2 23.6 8v15.2L20 26.2l-3.6-3V8Z" />
          <rect x={10.6} y={24.6} width={18.8} height={3.6} rx={1.7} />
          <path d="M18.5 28.2h3v4.8h-3Z" />
          <circle cx={20} cy={35.2} r={2.4} />
          <path d="M21.9 8.4v12" stroke={STAT_DARK} strokeWidth={1.7} strokeLinecap="round" fill="none" />
        </g>
      )
    }
    case 'defense': {
      // 盾徽：主体 + 深色内圈勾边
      return (
        <>
          <path d="M20 3.2 33.8 8.4v9q0 12.8-13.8 19.4Q6.2 30.2 6.2 17.4v-9Z" />
          <path
            d="M20 8.5 29.7 12.2v5.2q0 9.1-9.7 14.6-9.7-5.5-9.7-14.6v-5.2Z"
            fill="none"
            stroke={STAT_DARK}
            strokeWidth={2.1}
          />
        </>
      )
    }
    case 'recover': {
      // 急救十字：白色十字压在圆面上
      return (
        <>
          <circle cx={20} cy={20} r={15.6} />
          <path
            d="M16.4 9.6h7.2v6.8h6.8v7.2h-6.8v6.8h-7.2v-6.8H9.6v-7.2h6.8Z"
            fill="#fff"
            stroke="none"
          />
        </>
      )
    }
    case 'hand': {
      // 手牌：左右两张斜靠的暗色底卡衬托中间前卡（绘制顺序即层叠顺序）
      return (
        <>
          <rect x={13.4} y={7} width={13.2} height={18.6} rx={2.3} transform="rotate(-20 20 16.3)" fill={STAT_DARK} />
          <rect x={13.4} y={7} width={13.2} height={18.6} rx={2.3} transform="rotate(20 20 16.3)" fill={STAT_DARK} />
          <rect x={13.4} y={10.4} width={13.2} height={18.6} rx={2.3} />
        </>
      )
    }
    case 'hp': {
      // 心形：左上一道深色高光弧
      return (
        <>
          <path d="M20 34.8C9 27.4 5.2 20.9 5.2 15A7.7 7.7 0 0 1 20 11.2 7.7 7.7 0 0 1 34.8 15c0 5.9-3.8 12.4-14.8 19.8Z" />
          <path
            d="M11.8 15.6q2.7-4.3 6.5-4.2"
            stroke={STAT_DARK}
            strokeWidth={1.9}
            strokeLinecap="round"
            fill="none"
          />
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
      <g
        fill={STAT_COLOR}
        stroke="#fff"
        strokeWidth={5.2}
        strokeLinejoin="round"
        strokeLinecap="round"
        paintOrder="stroke"
      >
        <StatShapes stat={stat} />
      </g>
    </svg>
  )
}
