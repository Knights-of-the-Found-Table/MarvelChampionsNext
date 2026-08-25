package engine

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// ┌────────────────────────────────────────────────────────────────────────────┐
// │  如何正确实现 i18n（HOW TO DO I18N）—— 修改本包前必读                          │
// └────────────────────────────────────────────────────────────────────────────┘
//
// 【架构】翻译发生在前端，服务端只产出语言中立的键 + 结构化参数：
//
//	调用点      engine.Tf("c.discards", p.Name, hc)     // hc 是 Card
//	引擎内部    Msg{Key:"c.discards", Args:[{s:"Alice"},{k:"card",code:"01034",s:"…"}], Text:"Alice discards …"}
//	API 载荷    {key:"c.discards", args:[{s:"Alice"},{k:"card",code:"01034"}], text:"…"}
//	前端        用 /api/locales/{lang} 取到的格式串 + 卡牌代码→本地化卡名，渲染最终文本
//
// 服务端不渲染用户语言（Text 只是 en 基准文本，供测试与旧客户端兜底），
// 接口上不存在任何“客户端语言”状态；不同浏览器可各选语言。
//
// 【正确做法】
//  1. 新文案：在 i18n_catalog.go 增加条目（en/zh 都必须填，zh≠en），
//     调用点写 Tf("命名空间.语义键", 参数...)。键名稳定，一经发布不再改动。
//  2. 参数顺序：zh 译文可用 %[1]s、%[2]d 显式序号自由调换语序；
//     en/zh 的动词类型与参数个数必须一致（TestMessageArgConsistency 强制）。
//  3. 卡牌/实体名：把 Card、Entity 或 *data.CardDef 值本身作为参数传入
//     （Argify 自动记为 {k:"card", code:…}），前端按 code 解析本地化名称。
//     不要在格式串里内嵌卡名，也不要传 def.Name 字符串——那会把英文/中文
//     卡名焊死进参数，前端无法按语言解析。
//  4. 未翻译的遗留文案：用 S("plain English") 直通显示原文（gettext 缺译
//     回退惯例）。要翻译它：补目录条目 + 把 S(...) 换成 Tf(key, ...)。
//  5. 日志用 tlogf/tlogMinorf/tlogMajorf（卡牌包用导出的 TLog* 系列）；
//     Logf 仅接受原始格式串，仅供尚未翻译的过渡代码使用。
//
// 【禁止】
//  - 严禁“先产出英文，再用 regex/字符串匹配反向翻译”。本项目历史上两次
//    走过这条路（zhtext.go、i18n.go 运行时桥）均被否决并删除：匹配渲染后
//    的文本 inherently 脆弱（参数值与格式串歧义、旧存档、嵌套消息都会破），
//    且无法表达结构化参数。翻译必须在产出点以键完成。
//  - 严禁依赖渲染后的 Prompt/Label 文本做逻辑判断；用 PromptKey/Choice.ID。
//    同理严禁游戏逻辑解析卡面印刷文本（def.Text 的 regex/子串匹配——见
//    paymentIconSpec/hinderRE/boostSpawnsMinion 的三次教训）：印刷数值与
//    关键词在 data 层加载时解析一次成结构化字段（CardDef.Keywords、
//    BoostEntersPlay 等），逻辑只读字段。支付图标的线上形态是
//    Question.PayIcons / Choice.Icons / ctx["abilityIcons"]，前端绝不
//    反向解析渲染文本。
//  - 严禁把用户语言渲染进存储或 API（旧存档中的 en 文本按原样显示即可）。
//
// 【目录】i18n_catalog.go（手写核心文案）+ i18n_cards_catalog.go（卡牌包批量
// 迁移条目，键空间 c.*）在 init 时合并进 messages；/api/locales 由服务端
// 直接从合并后的目录导出，前端与引擎共享同一份事实源。

// Lang 是目录语言标识。引擎内部渲染（Msg.Text）固定以 en 为基准，
// zh 条目仅供 /api/locales 导出与测试使用。
type Lang string

const (
	LangEn Lang = "en"
	LangZh Lang = "zh"
)

var uiLang = LangEn

// SetLang 仅用于测试与诊断。服务端不要调用：API 产出的 Text 恒为 en，
// 语言选择完全由前端完成。
func SetLang(l Lang) {
	if l == "" {
		l = LangEn
	}
	uiLang = l
}

// UILang 返回当前内部渲染语言（测试用）。
func UILang() Lang { return uiLang }

// ─── 结构化消息参数 ──────────────────────────────────────────────────────────

// Arg 是一条语言中立的参数。Kind:
//   - ""（缺省）: S 为纯文本（玩家名、形态名等，不翻译）
//   - "i":        整数
//   - "card":     Code 为卡牌代码，前端按观战者语言解析本地化卡名；
//     S 仅为 en 兜底显示
//   - "msg":      M 为嵌套消息（如带血量的敌人标签），前端递归渲染
type Arg struct {
	Kind string `json:"k,omitempty"`
	S    string `json:"s,omitempty"`
	I    int    `json:"i,omitempty"`
	Code string `json:"code,omitempty"`
	M    *Msg   `json:"msg,omitempty"`
}

func (a Arg) String() string {
	if a.Kind == "i" {
		return strconv.Itoa(a.I)
	}
	return a.S
}

// value returns the arg in its Sprintf-compatible form: ints stay ints so
// %d verbs render correctly, everything else renders as its display text.
func (a Arg) value() any {
	if a.Kind == "i" {
		return a.I
	}
	return a.S
}

// Argify 把任意实参规整为结构化 Arg。Card/Entity/*CardDef 归一为 card
// 引用（记 code），其余按文本/整数收存。
func Argify(v any) Arg {
	switch x := v.(type) {
	case Arg:
		return x
	case Msg:
		if x.Key == "" {
			return Arg{S: x.Text}
		}
		return Arg{Kind: "msg", S: x.Text, M: &x}
	case Card:
		return Arg{Kind: "card", Code: x.Code, S: x.Def().Name}
	case *Card:
		if x == nil {
			return Arg{}
		}
		return Arg{Kind: "card", Code: x.Code, S: x.Def().Name}
	case Entity:
		return Arg{Kind: "card", Code: x.ECode(), S: x.EDef().Name}
	case nil:
		return Arg{}
	case *data.CardDef:
		if x == nil {
			return Arg{}
		}
		return Arg{Kind: "card", Code: x.Code, S: x.Name}
	case string:
		return Arg{S: x}
	case int:
		return Arg{Kind: "i", I: x}
	case int8:
		return Arg{Kind: "i", I: int(x)}
	case int16:
		return Arg{Kind: "i", I: int(x)}
	case int32:
		return Arg{Kind: "i", I: int(x)}
	case int64:
		return Arg{Kind: "i", I: int(x)}
	case uint:
		return Arg{Kind: "i", I: int(x)}
	case uint8:
		return Arg{Kind: "i", I: int(x)}
	case uint16:
		return Arg{Kind: "i", I: int(x)}
	case uint32:
		return Arg{Kind: "i", I: int(x)}
	case uint64:
		return Arg{Kind: "i", I: int(x)}
	case float32, float64:
		return Arg{S: fmt.Sprint(x)}
	default:
		return Arg{S: fmt.Sprint(v)}
	}
}

// ─── 消息 ────────────────────────────────────────────────────────────────────

// Msg 是一条可本地化消息：目录键 + 结构化参数 + en 基准文本。
// 它是 Ask 提示、Choice/Ability 标签与 GameOver 败因的统一载体。
type Msg struct {
	Key  string `json:"key,omitempty"`
	Args []Arg  `json:"args,omitempty"`
	Text string `json:"text"`
}

func (m Msg) String() string { return m.Text }

// UnmarshalJSON 兼容旧存档/旧客户端里的纯字符串形态。
func (m *Msg) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		m.Text = s
		return nil
	}
	type alias Msg
	var a alias
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	*m = Msg(a)
	return nil
}

// T 返回 key 的当前语言固定短语（无参数、纯字符串场景，如占位名）。
func T(key string) string { return msgFormat(key) }

// Tf 以 key 构造结构化消息：Args 保留类型信息供前端翻译，Text 以引擎
// 基准语言（en）渲染。调用点参数请直接传 Card/Entity 值而非名字字符串。
func Tf(key string, args ...any) Msg {
	m := Msg{Key: key}
	if len(args) == 0 {
		m.Text = msgFormat(key)
		return m
	}
	disp := make([]any, len(args))
	for i, a := range args {
		arg := Argify(a)
		m.Args = append(m.Args, arg)
		disp[i] = arg.value()
	}
	m.Text = fmt.Sprintf(msgFormat(key), disp...)
	return m
}

// S 包装一段暂未进入目录的文本：按原文直通显示（缺译回退）。
// 补翻译时改为 Tf(key, ...) 并在目录中加条目。
func S(text string) Msg { return Msg{Text: text} }

// ─── 目录查找 ────────────────────────────────────────────────────────────────

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

// Messages 导出某语言的完整目录（key→格式串），供 /api/locales 使用。
// 缺失该语言的条目以 en 回退填充，保证前端目录完备。
func Messages(lang Lang) map[string]string {
	out := make(map[string]string, len(messages))
	for k, m := range messages {
		f := m[LangEn]
		if s, ok := m[lang]; ok && s != "" {
			f = s
		}
		out[k] = f
	}
	return out
}
