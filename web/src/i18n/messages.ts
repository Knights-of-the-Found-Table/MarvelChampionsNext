// 共享文案目录的类型侧写。把仓库根的单一事实源 i18n/messages.json 以
// 纯类型导入（typeof import 在编译期完全擦除、不产生任何运行时代码与
// 打包体积）——tsc 在这里强制每个键都是 {en, zh} 成对，前端从此没有
// 自己的第二份字典；运行时数据经 /locales 接口获取（见 catalog.ts）。
type Raw = typeof import('../../../i18n/messages.json')

// 约束检查：任何缺 en 或 zh 的条目都会让编译失败。
type EnsurePairs<T extends Record<string, { en: string; zh: string }>> = T
export type CatalogChecked = EnsurePairs<Raw>

// MsgKey 是文案键全集（引擎命名空间 c./log./m./q./reason.* 与 UI 命名空间
// nav.*/game.*/type.* 等）。动态键（如 `res.${icon}`）仍以 string 传参，
// 由 t() 的未命中回退兜底。
export type MsgKey = keyof Raw
