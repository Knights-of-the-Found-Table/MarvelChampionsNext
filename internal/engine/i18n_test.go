package engine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

var verbRe = regexp.MustCompile(`%\[(\d+)\]([a-zA-Z])|%([a-zA-Z])`)

// parseVerbs 返回 (按出现顺序的参数序号, 动词类型, 是否显式序号)。
func parseVerbs(f string) ([]int, []string, bool) {
	var idx []int
	var typ []string
	explicit := false
	for _, m := range verbRe.FindAllStringSubmatch(f, -1) {
		if m[1] != "" {
			explicit = true
			var n int
			fmt.Sscanf(m[1], "%d", &n)
			idx = append(idx, n)
			typ = append(typ, m[2])
		} else {
			idx = append(idx, 0) // 位置参数:序号按出现次序
			typ = append(typ, m[3])
		}
	}
	return idx, typ, explicit
}

// TestMessageCatalogComplete 要求每个键都有非空 en/zh 两条,且译文确实
// 与原文不同（防止复制原文当译文）。
func TestMessageCatalogComplete(t *testing.T) {
	if len(messages) < 2000 {
		t.Fatalf("catalog unexpectedly small: %d keys", len(messages))
	}
	for key, m := range messages {
		en, zh := m[LangEn], m[LangZh]
		if en == "" || zh == "" {
			t.Errorf("%s: missing en or zh entry", key)
			continue
		}
		if en == zh {
			t.Errorf("%s: zh identical to en", key)
		}
	}
}

// TestMessageArgConsistency 校验各语言格式串的参数使用:
//   - 用了显式序号（%[1]s）就必须全部显式,且恰好覆盖 1..N 各一次;
//   - 各语言的第 i 个参数类型必须一致（Sprintf 按位置/序号填参,
//     类型错位或数量不匹配会在运行期输出 %!VERB 或乱序参数）。
func TestMessageArgConsistency(t *testing.T) {
	for key, m := range messages {
		perLang := map[Lang][]string{}
		for lang, f := range m {
			idx, typ, explicit := parseVerbs(f)
			if explicit {
				seen := map[int]bool{}
				for _, n := range idx {
					if n < 1 || seen[n] {
						t.Errorf("%s/%s: bad explicit arg index %v", key, lang, idx)
						break
					}
					seen[n] = true
				}
				for i := 1; i <= len(idx); i++ {
					if !seen[i] {
						t.Errorf("%s/%s: explicit indexes %v skip arg %d", key, lang, idx, i)
						break
					}
				}
			} else {
				for i := range idx {
					idx[i] = i + 1
				}
			}
			order := make([]string, len(idx))
			for i, n := range idx {
				order[n-1] = typ[i]
			}
			perLang[lang] = order
		}
		en := perLang[LangEn]
		for lang, order := range perLang {
			if lang == LangEn {
				continue
			}
			if len(order) != len(en) {
				t.Errorf("%s/%s: %d args, en has %d", key, lang, len(order), len(en))
				continue
			}
			for i := range order {
				if order[i] != en[i] {
					t.Errorf("%s/%s: arg %d is %%%s, en uses %%%s", key, lang, i+1, order[i], en[i])
				}
			}
		}
	}
}

func TestResourcePayment(t *testing.T) {
	leadership := &data.CardDef{Code: "leadership-target", Name: "Leadership Card", Type: "event", Aspect: "leadership"}
	justice := &data.CardDef{Code: "justice-target", Name: "Justice Card", Type: "event", Aspect: "justice"}
	// Doubling conditions are structured data parsed at load; hand-built
	// fixtures set them directly.
	power := &data.CardDef{Code: "62017", Name: "The Power of Leadership", Type: "resource", Resources: []string{"wild"}, DoubleForAspect: "leadership"}
	if got := iconCount(power) + powerOfBonus(power, leadership); got != 2 {
		t.Fatalf("The Power of Leadership should pay 2 for Leadership, got %d", got)
	}
	if got := iconCount(power) + powerOfBonus(power, justice); got != 1 {
		t.Fatalf("The Power of Leadership should pay 1 for Justice, got %d", got)
	}
	if got := iconCount(power) + powerOfBonus(power, nil); got != 1 {
		t.Fatalf("The Power of Leadership should pay 1 without a card target, got %d", got)
	}
}

// TestPowerDoublingUnderZhOverlay: the zh overlay replaces Name/Text with
// Chinese translations; doubling keys off DoubleForAspect/DoubleForTrait
// parsed once from the English print text, so localization cannot disable
// it (regression: Mockingbird cost 3 could not be paid with 凝聚力量).
func TestPowerDoublingUnderZhOverlay(t *testing.T) {
	basicAlly := &data.CardDef{Code: "01083", Name: "仿声鸟", EName: "Mockingbird", Type: "ally", Aspect: "basic"}
	powerAll := &data.CardDef{Code: "13024", Name: "凝聚力量", EName: "The Power in All of Us", Type: "resource", Resources: []string{"wild"}, DoubleForAspect: "basic"}
	if got := iconCount(powerAll) + powerOfBonus(powerAll, basicAlly); got != 2 {
		t.Fatalf("The Power in All of Us should pay 2 for a basic card under zh overlay, got %d", got)
	}
	aspectCard := &data.CardDef{Code: "x", Name: "正义卡", EName: "Justice Card", Type: "event", Aspect: "justice"}
	if got := iconCount(powerAll) + powerOfBonus(powerAll, aspectCard); got != 1 {
		t.Fatalf("The Power in All of Us should pay 1 for a Justice card, got %d", got)
	}
	zhLeadership := &data.CardDef{Code: "62017", Name: "领袖之力", EName: "The Power of Leadership", Type: "resource", Resources: []string{"wild"}, DoubleForAspect: "leadership"}
	leadershipTarget := &data.CardDef{Code: "y", Name: "领袖事件", EName: "Leadership Event", Type: "event", Aspect: "leadership"}
	if got := iconCount(zhLeadership) + powerOfBonus(zhLeadership, leadershipTarget); got != 2 {
		t.Fatalf("The Power of Leadership should pay 2 for Leadership under zh overlay, got %d", got)
	}
}

// TestResourceDoubleParsing: every printed doubling resource card parses to
// its structured condition at data load (aspect cards, the Basic (gray)
// card, and the [[AERIAL]]/[[PSIONIC]] trait variants); Self Confidence's
// damage-based wording must NOT parse as a payment-target doubling.
func TestResourceDoubleParsing(t *testing.T) {
	cases := []struct {
		code          string
		aspect, trait string
	}{
		{"01079", "protection", ""},
		{"01055", "aggression", ""},
		{"01062", "justice", ""},
		{"01072", "leadership", ""},
		{"13024", "basic", ""},
		{"46023", "basic", ""},
		{"42022", "", "aerial"},
		{"40028", "", "psionic"},
		{"41021", "", "psionic"},
		{"44025", "", ""}, // Self Confidence: identity-damage condition, not a target doubling
	}
	for _, c := range cases {
		def, ok := DB.Lookup(c.code)
		if !ok {
			t.Fatalf("%s not in DB", c.code)
		}
		if def.DoubleForAspect != c.aspect || def.DoubleForTrait != c.trait {
			t.Errorf("%s: doubleFor=%q/%q, want %q/%q", c.code, def.DoubleForAspect, def.DoubleForTrait, c.aspect, c.trait)
		}
	}
}

// TestI18nFallback 缺失的键回退键名本身（gettext 惯例）。
func TestI18nFallback(t *testing.T) {
	if got := Tf("no.such.key"); got.Text != "no.such.key" {
		t.Errorf("missing key: %q", got.Text)
	}
	if got := T("no.such.key"); got != "no.such.key" {
		t.Errorf("missing key (T): %q", got)
	}
}

// TestI18nZhFormatting 抽查中文产出,含显式序号重排（q.paySelect 的
// en 参数序是 cost、name,中文语序是 name 在前）。
func TestI18nZhFormatting(t *testing.T) {
	SetLang(LangZh)
	defer SetLang(LangEn)

	cases := []struct {
		key  string
		args []any
		want string
	}{
		{"q.paySelect", []any{2, "能量护盾"}, "为 能量护盾 支付 2 点资源"},
		{"log.takesDamage", []any{"绿魔", 4, 6, 10}, "绿魔 受到 4 点伤害（6/10）"},
		{"log.plays", []any{"蜘蛛侠", "网击"}, "蜘蛛侠 打出 网击"},
		{"m.cost", []any{"坚毅", 2}, "坚毅（费用 2）"},
		{"log.round", []any{3}, "── 第 3 轮 ──"},
		{"log.takesStored", []any{"蜘蛛侠", 2, "回声"}, "蜘蛛侠 取回 2 张储存的牌（来自 回声）"},
		{"q.attacksForDefend", []any{"绿魔", 5}, "绿魔 发起 5 点攻击:是否防御?"},
	}
	for _, c := range cases {
		if got := Tf(c.key, c.args...).Text; got != c.want {
			t.Errorf("Tf(%q) = %q, want %q", c.key, got, c.want)
		}
	}
	if got := T("q.yourTurn"); got != "你的回合" {
		t.Errorf("T(q.yourTurn) = %q", got)
	}
}

// TestI18nEnUnchanged 默认语言下 Text 与迁移前的英文字面量一致——
// 这是测试套件与旧存档所依赖的稳定输出。
func TestI18nEnUnchanged(t *testing.T) {
	SetLang(LangEn)
	defer SetLang(LangEn)
	if got := Tf("log.plays", "A", "B"); got.Text != "A plays B" {
		t.Errorf("en drift: %q", got.Text)
	}
	if got := Tf("q.paySelect", 2, "Shield"); got.Text != "Pay 2 resources for Shield (select cards)" {
		t.Errorf("en drift: %q", got.Text)
	}
}

// TestMsgStructure 验证结构化消息:键 + 类型化参数。CardDef/Card/Entity
// 参数记录为 card 引用（前端按 code 解析本地化卡名）,整数记为 i,
// 嵌套 Msg 递归保留结构,en 渲染照常可用。
func TestMsgStructure(t *testing.T) {
	def := &data.CardDef{Code: "01034", Name: "Shield Block"}
	m := Tf("log.plays", "Alice", def)
	if m.Key != "log.plays" || m.Text != "Alice plays Shield Block" {
		t.Fatalf("bad msg: %#v", m)
	}
	if len(m.Args) != 2 {
		t.Fatalf("args: %#v", m.Args)
	}
	if m.Args[0] != (Arg{S: "Alice"}) {
		t.Errorf("plain arg: %#v", m.Args[0])
	}
	if m.Args[1] != (Arg{Kind: "card", Code: "01034", S: "Shield Block"}) {
		t.Errorf("card arg: %#v", m.Args[1])
	}

	n := Tf("c.dealDamageTo", 5, Tf("m.hp", def, 2, 3))
	if len(n.Args) != 2 || n.Args[0] != (Arg{Kind: "i", I: 5}) {
		t.Fatalf("int/nested args: %#v", n.Args)
	}
	nested := n.Args[1]
	if nested.Kind != "msg" || nested.M == nil || nested.M.Key != "m.hp" || nested.M.Text != "Shield Block — 2/3 HP" {
		t.Errorf("nested msg arg: %#v", nested)
	}
	if n.Text != "Deal 5 damage to Shield Block — 2/3 HP" {
		t.Errorf("nested render: %q", n.Text)
	}
}

// TestStructuredLog 验证日志条目同时携带 key/args（供前端按观战者语言
// 渲染）与 en 基准 Text（测试/兜底）。旧存档的纯字符串条目按原样加载。
func TestStructuredLog(t *testing.T) {
	g := &Game{}
	g.tlogf("log.plays", "Alice", &data.CardDef{Code: "01034", Name: "Shield Block"})
	e := g.Log[0]
	if e.Level != LogInfo || e.Key != "log.plays" || e.Text != "Alice plays Shield Block" {
		t.Fatalf("entry: %#v", e)
	}
	if len(e.Args) != 2 || e.Args[1].Code != "01034" {
		t.Fatalf("entry args: %#v", e.Args)
	}

	var legacy LogEntries
	if err := json.Unmarshal([]byte(`["Round 1 begins"]`), &legacy); err != nil {
		t.Fatal(err)
	}
	if len(legacy) != 1 || legacy[0].Text != "Round 1 begins" || legacy[0].Key != "" || legacy[0].Level != LogInfo {
		t.Fatalf("legacy entry: %#v", legacy[0])
	}
}

// TestAskPromptStructure 验证问题提示的 Prompt（en 基准）与
// PromptKey/PromptArgs（前端重渲染）同时就位,子树拷贝不丢失。
func TestAskPromptStructure(t *testing.T) {
	q := Ask(Tf("q.paySelect", 2, &data.CardDef{Code: "01034", Name: "Shield Block"}),
		Choice{ID: "pass", Label: S("Pass"), Kind: ChoicePass})
	if q.Prompt != "Pay 2 resources for Shield Block (select cards)" {
		t.Errorf("prompt: %q", q.Prompt)
	}
	if q.PromptKey != "q.paySelect" || len(q.PromptArgs) != 2 || q.PromptArgs[1].Code != "01034" {
		t.Errorf("prompt structure: %q %#v", q.PromptKey, q.PromptArgs)
	}
	if q.Choices[0].Label.Text != "Pass" {
		t.Errorf("label: %#v", q.Choices[0].Label.Text)
	}

	// 旧存档/旧客户端里的字符串形态标签反序列化为纯文本 Msg。
	var c Choice
	if err := json.Unmarshal([]byte(`{"id":"x","label":"End turn"}`), &c); err != nil {
		t.Fatal(err)
	}
	if c.Label.Text != "End turn" || c.Label.Key != "" {
		t.Fatalf("legacy label: %#v", c.Label.Text)
	}
}

// TestMessagesExport 验证 /api/locales 导出的目录完备:zh 缺失的条目
// 以 en 填充,保证前端总能按 key 取到格式串。
func TestMessagesExport(t *testing.T) {
	for _, lang := range []Lang{LangEn, LangZh} {
		out := Messages(lang)
		if len(out) != len(messages) {
			t.Fatalf("%s: exported %d keys, catalog has %d", lang, len(out), len(messages))
		}
		for k, v := range out {
			if v == "" {
				t.Errorf("%s: empty format for %s", lang, k)
			}
		}
	}
	if Messages(LangZh)["q.yourTurn"] != "你的回合" {
		t.Errorf("zh export wrong: %q", Messages(LangZh)["q.yourTurn"])
	}
}
