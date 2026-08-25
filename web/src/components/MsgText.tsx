// 结构化消息的 React 渲染：把 labels 层解析出的片段序列铺开。卡名片段
// （{card:{code,name}}）复用场上卡牌同一套 body-portal 卡图预览——悬浮
// 对局记录里的卡名即可看牌。
import { useRef } from 'react'
import type { MsgWire } from '../api'
import { useCardZoom } from '../cards'
import { useMsgParts, type MsgPart } from '../i18n/labels'

// 单个卡名片段：自带锚点 ref，hover 出与 GameCard 一致的 card-zoom 层。
function CardArg({ code, name }: { code: string; name: string }) {
  const ref = useRef<HTMLSpanElement | null>(null)
  const zoom = useCardZoom(code, ref)
  if (!code) return <span>{name}</span>
  return (
    <span ref={ref} className="card-arg" onMouseEnter={zoom.onEnter} onMouseLeave={zoom.hide}>
      {name}
      {zoom.overlay}
    </span>
  )
}

export default function MsgText({ m }: { m: MsgWire | string }) {
  const parts = useMsgParts()(m)
  return (
    <>
      {parts.map((p: MsgPart, i: number) =>
        't' in p ? (
          p.t
        ) : 'card' in p ? (
          <CardArg key={i} code={p.card.code} name={p.card.name} />
        ) : (
          <MsgText key={i} m={p.msg} />
        ),
      )}
    </>
  )
}
