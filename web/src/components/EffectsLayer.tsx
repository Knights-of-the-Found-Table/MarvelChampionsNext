// 动效覆盖层：飘字与目标箭头，渲染在场景坐标系内（随场景缩放）。
import type { Arrow, Floater } from '../board/fx'

export default function EffectsLayer({ floaters, arrows }: { floaters: Floater[]; arrows: Arrow[] }) {
  return (
    <>
      {arrows.map((a) => (
        <div
          key={a.key}
          className={`fx-arrow fx-arrow-${a.kind}`}
          style={
            {
              left: a.x,
              top: a.y,
              rotate: `${a.angle}deg`,
              '--len': `${Math.max(a.len, 24)}px`,
            } as React.CSSProperties
          }
        />
      ))}
      {floaters.map((f) => (
        <div
          key={f.key}
          className={`fx-float fx-float-${f.kind}`}
          style={{ left: f.x, top: f.y, '--i': f.index } as React.CSSProperties}
        >
          {f.text}
        </div>
      ))}
    </>
  )
}
