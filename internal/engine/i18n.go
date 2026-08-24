package engine

import "fmt"

// Lang 是引擎产出文本（日志、问题提示、选项标签、胜负原因）的界面
// 语言。默认 en：引擎包与测试以英文为基准文本；cmd/server 按 MC_LANG
// 设置。
//
// 文案按消息 ID 查找（参考 go-i18n 的 MessageID 与 golang.org/x/text
// message 的 Printf 目录）：调用点以 Tf(key, args...) 直接用目标语言
// 的格式串格式化，不存在"先英文再翻译"的中间形态。未经翻译的键回退
// en，再回退键名本身（gettext 惯例：缺译显示源语言）。
type Lang string

const (
	LangEn Lang = "en"
	LangZh Lang = "zh"
)

var uiLang = LangEn

// SetLang 设置引擎产出文本的语言（cmd/server 启动时调用）。
func SetLang(l Lang) {
	if l == "" {
		l = LangEn
	}
	uiLang = l
}

// UILang 返回当前产出语言（测试与诊断用）。
func UILang() Lang { return uiLang }

// msgFormat 按 key 取当前语言的格式串；缺失时回退 en，再回退键名。
func msgFormat(key string) string {
	m, ok := messages[key]
	if !ok {
		return key
	}
	if f, ok := m[uiLang]; ok && f != "" {
		return f
	}
	if f, ok := m[LangEn]; ok && f != "" {
		return f
	}
	return key
}

// T 返回 key 对应的固定短语（无参数消息）。
func T(key string) string { return msgFormat(key) }

// Tf 按 key 以当前语言格式化。格式串可用 %[1]s 形式的显式参数序号，
// 让译文自由调整参数顺序；en 条目与迁移前的英文字面量逐字一致，
// 因此默认语言下输出与迁移前完全相同（存量测试不受影响）。
func Tf(key string, args ...any) string {
	if len(args) == 0 {
		return msgFormat(key)
	}
	return fmt.Sprintf(msgFormat(key), args...)
}

// msgIs 判断 s 是否等于 key 在任一语言下的文案。供依赖提示文本做
// 逻辑判断的地方（如 RebuildTurnMenu 识别"Your turn"），使判断在
// 两种语言下都成立。
func msgIs(s, key string) bool {
	if m, ok := messages[key]; ok {
		for _, f := range m {
			if s == f {
				return true
			}
		}
		return false
	}
	return s == key
}
