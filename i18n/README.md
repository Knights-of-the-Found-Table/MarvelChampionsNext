# i18n —— 前后端共享文案的单一事实源

`messages.json` 是本项目**唯一**的文案资源：引擎的日志/提示/选项标签/败因、
前端的界面文字，全部存在这一个文件里。每键 en/zh 成对，漏译在结构上不可
能发生（缺失会导致 Go 进程启动 panic 与 TS 编译失败）。

## 文件格式

```json
{
	"c.hexBoltMilled": {"en":"Hex Bolt discards %s …", "zh":"咒术飞弹：弃置遭遇牌 %s…"},
	"game.round": {"en":"Round %d", "zh":"第 %d 轮"}
}
```

- 键按字典序排列、每键一行；`en`/`zh` 都非空且 zh≠en。
- 占位符只有一种语法：Go printf 动词 `%s` / `%d` / `%%`，语序不同时用显式
  序号 `%[1]s` / `%[2]d`。前端渲染同一套动词（web/src/i18n/format.ts）。
- 卡牌/实体名**绝不写进格式串或作为字符串参数**：调用点传 Card/Entity，
  线上形态是 `{k:"card", code}`，客户端按 code 解析本地化卡名
  （见 internal/engine/i18n.go 的 HOW TO DO I18N）。

## 消费方式

| 端 | 路径 | 说明 |
|---|---|---|
| 引擎 | 根级 `i18n` 包 `//go:embed messages.json` → `All()` | `engine.messages` 由此构建；`Msg.Text` 以 en 渲染 |
| 服务端 API | `/api/v1/locales/manifest`、`/locales/{lang}/{hash}`、`/locales/{lang}` | 内容 hash 寻址 + immutable 长缓存（internal/api/handlers_locales.go） |
| 前端运行时 | `web/src/i18n/catalog.ts` 经 `/locales` 装载 | LangProvider 就绪门控，装载完成后同步读字典 |
| 前端类型 | `web/src/i18n/messages.ts` 的纯类型导入 | 编译期擦除、不进打包体积；强制条目形状 |

## 改动流程

1. 编辑 `i18n/messages.json`：加键/改译文（成对填写）。
2. `go test ./internal/engine -run TestMessage -count=1`
   —— TestMessageCatalogComplete 强制双语完备（非空且 zh≠en）；
   TestMessageArgConsistency 强制 en/zh 动词类型与数量一致。
3. `cd web && npx tsc -b` —— 类型侧写保证 JSON 形状；引用已删键会编译报错。

键名一经发布不再改动。命名空间沿用既有约定：引擎侧 `c.*` `log.*` `m.*`
`q.*` `reason.*`；前端 UI `nav.*` `game.*` `type.*` `res.*` 等。
