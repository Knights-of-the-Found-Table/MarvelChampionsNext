// Package i18n 持有前后端共享的文案单一事实源 messages.json：
// 一个文件、每键 en/zh 成对，杜绝「漏译」与「两端各改一份」漂移。
//
// 三条消费路径共享同一份嵌入数据：
//   - 服务端：engine.messages 由 All() 构建（Msg.Text 的 en 基准渲染）；
//   - /api/v1/locales/{lang} 及其 manifest/hash 变体：直接导出目录；
//   - 前端：运行时经 /locales 接口拉取（LangProvider 就绪门控保证
//     UI 文案在渲染前可用），类型层面 web/src/i18n/messages.ts 用
//     纯类型导入同一 JSON 派生 MsgKey 完备性。
//
// 修改文案只改本目录的 messages.json：键按字典序排列、每键一行，
// 占位符统一 Go 动词语法（%s %d %[1]s）。测试在 internal/engine
// （TestMessageCatalogComplete / TestMessageArgConsistency）强制
// 双语完备性与动词一致性。
package i18n

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed messages.json
var raw []byte

// Entry 是一条文案的双语内容。
type Entry struct {
	En string `json:"en"`
	Zh string `json:"zh"`
}

var catalog = load()

// load 逐 token 解码 messages.json：除语法错误外，重复键、空 en/zh 都在
// 进程启动时 panic——宁可启动失败也不带着残缺目录上线。
func load() map[string]Entry {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil {
		panic(fmt.Sprintf("i18n: messages.json: %v", err))
	} else if d, ok := tok.(json.Delim); !ok || d != '{' {
		panic("i18n: messages.json: expected object at top level")
	}
	out := make(map[string]Entry, 3000)
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			panic(fmt.Sprintf("i18n: messages.json: %v", err))
		}
		key := keyTok.(string)
		var e Entry
		if err := dec.Decode(&e); err != nil {
			panic(fmt.Sprintf("i18n: messages.json: %s: %v", key, err))
		}
		if _, dup := out[key]; dup {
			panic(fmt.Sprintf("i18n: messages.json: duplicate key %q", key))
		}
		if e.En == "" {
			panic(fmt.Sprintf("i18n: messages.json: %s: empty en", key))
		}
		if e.Zh == "" {
			panic(fmt.Sprintf("i18n: messages.json: %s: empty zh", key))
		}
		out[key] = e
	}
	if _, err := dec.Token(); err != nil { // 收尾 '}'
		panic(fmt.Sprintf("i18n: messages.json: %v", err))
	}
	return out
}

// All 返回完整目录（只读约定，调用方不得修改）。
func All() map[string]Entry { return catalog }
