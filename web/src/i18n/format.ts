// 与服务端一致的 printf 动词渲染：%s %d %%，及显式序号 %[1]s / %[2]d。
// 前后端共用同一目录（仓库根 i18n/messages.json）意味着占位符语法也只有
// 一种——UI 文案的 t() 与引擎消息的 msgParts 都走这套动词。
const VERB = /%(?:\[(\d+)\])?([a-zA-Z%])/g

// sprint 按位置/显式序号把 args 填进格式串。缺参保留动词原文（与 Go
// sprintf 的容错展示一致），数值与文本实参都按 String() 渲染。
export function sprint(format: string, args: Array<string | number>): string {
  if (args.length === 0) return format
  let next = 0
  let out = ''
  let last = 0
  for (const m of format.matchAll(VERB)) {
    const idx = m.index ?? 0
    out += format.slice(last, idx)
    last = idx + m[0].length
    if (m[2] === '%') {
      out += '%'
      continue
    }
    const n = m[1] ? parseInt(m[1], 10) - 1 : next++
    out += n >= 0 && n < args.length ? String(args[n]) : m[0]
  }
  out += format.slice(last)
  return out
}
