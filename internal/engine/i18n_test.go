package engine

import (
	"fmt"
	"regexp"
	"testing"
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
	if len(messages) < 200 {
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

// TestLegacyMessageCatalog guards the audited compatibility catalog. New code
// should still prefer keyed T/Tf messages.
func TestLegacyMessageCatalog(t *testing.T) {
	if len(legacyMessages) < 2300 {
		t.Fatalf("legacy catalog unexpectedly small: %d entries", len(legacyMessages))
	}
	for en, zh := range legacyMessages {
		if en == "" || zh == "" {
			t.Errorf("empty legacy translation: %q -> %q", en, zh)
			continue
		}
		enIdx, enTypes, enExplicit := parseVerbs(en)
		zhIdx, zhTypes, zhExplicit := parseVerbs(zh)
		normalize := func(idx []int, types []string, explicit bool) (map[int]string, bool) {
			out := make(map[int]string, len(idx))
			for i, n := range idx {
				if !explicit {
					n = i + 1
				}
				if n < 1 || out[n] != "" {
					return nil, false
				}
				out[n] = types[i]
			}
			return out, true
		}
		enSig, enOK := normalize(enIdx, enTypes, enExplicit)
		zhSig, zhOK := normalize(zhIdx, zhTypes, zhExplicit)
		if !enOK || !zhOK || len(enSig) != len(zhSig) {
			t.Errorf("legacy placeholder mismatch: %q -> %q", en, zh)
			continue
		}
		for n, typ := range enSig {
			if zhSig[n] != typ {
				t.Errorf("legacy placeholder mismatch at arg %d: %q -> %q", n, en, zh)
			}
		}
	}
}

// TestMessageArgConsistency 校验各语言格式串的参数使用:
//   - 用了显式序号（%1$s）就必须全部显式,且恰好覆盖 1..N 各一次;
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

// TestI18nFallback 缺失的键回退键名本身（gettext 惯例）。
func TestI18nFallback(t *testing.T) {
	if got := Tf("no.such.key"); got != "no.such.key" {
		t.Errorf("missing key: %q", got)
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
		if got := Tf(c.key, c.args...); got != c.want {
			t.Errorf("Tf(%q) = %q, want %q", c.key, got, c.want)
		}
	}
	if got := T("q.yourTurn"); got != "你的回合" {
		t.Errorf("T(q.yourTurn) = %q", got)
	}
	if !msgIs("你的回合", "q.yourTurn") || !msgIs("Your turn", "q.yourTurn") {
		t.Errorf("msgIs failed across languages")
	}
}

// TestI18nEnUnchanged 默认语言下输出与迁移前的英文字面量一致。
func TestI18nEnUnchanged(t *testing.T) {
	SetLang(LangEn)
	defer SetLang(LangEn)
	if got := Tf("log.plays", "A", "B"); got != "A plays B" {
		t.Errorf("en drift: %q", got)
	}
	if got := Tf("q.paySelect", 2, "Shield"); got != "Pay 2 resources for Shield (select cards)" {
		t.Errorf("en drift: %q", got)
	}
}

func TestLegacyLogFormatUsesCatalog(t *testing.T) {
	SetLang(LangZh)
	defer SetLang(LangEn)

	g := &Game{}
	g.Logf("%s plays %s", "蜘蛛侠", "网击")
	g.LogMajorf("💀 Defeat: %s", "主线密谋完成")

	if got := g.Log[0].Text; got != "蜘蛛侠 打出 网击" {
		t.Errorf("legacy log format was not localized: %q", got)
	}
	if got := g.Log[1].Text; got != "💀 败北:主线密谋完成" {
		t.Errorf("legacy major log format was not localized: %q", got)
	}
}

func TestLegacyRenderedTextUsesCatalog(t *testing.T) {
	SetLang(LangZh)
	defer SetLang(LangEn)

	cases := map[string]string{
		"ChagallC's turn begins":                              "ChagallC 的回合开始",
		"Boost card revealed: 横冲直撞 (+1)":                      "揭示增效牌 横冲直撞（+1）",
		"犀牛人 takes 3 damage (9/14)":                           "犀牛人 受到 3 点伤害（9/14）",
		"💀 Defeat: The main scheme completed":                 "💀 败北:主线密谋完成",
		"Pay 2 resources for 能量护盾 (select cards)":             "为 能量护盾 支付 2 点资源",
		"Discard 蜘蛛侠":                                         "弃置 蜘蛛侠",
		"Remove 3 threat from 暴力闯入！":                          "从 暴力闯入！ 上移除 3 点威胁",
		"Heal 2 damage from 德拉克斯":                             "为 德拉克斯 治疗 2 点伤害",
		"The Bellerophon enters play with 3 missile counters": "柏勒罗丰号进场并带有 3 个导弹指示物",
		"The Bellerophon — choose a player":                   "柏勒罗丰号——选择一位玩家",
	}
	for input, want := range cases {
		if got := localizeLegacyRenderedText(input); got != want {
			t.Errorf("localizeLegacyRenderedText(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLegacyQuestionUsesCatalog(t *testing.T) {
	SetLang(LangZh)
	defer SetLang(LangEn)

	q := Ask("Your turn",
		Choice{Label: "End turn", Kind: ChoiceEndTurn},
		Choice{Label: "Pass", Kind: ChoicePass}.WithThen(
			Ask("Choose an enemy", Choice{Label: "Done", Kind: ChoicePass}),
		),
	)

	if q.Prompt != "你的回合" || q.Choices[0].Label != "结束回合" || q.Choices[1].Label != "跳过" {
		t.Fatalf("legacy question was not localized: %#v", q)
	}
	if q.Choices[1].Then == nil || q.Choices[1].Then.Prompt != "选择一个敌人" || q.Choices[1].Then.Choices[0].Label != "完成" {
		t.Fatalf("nested legacy question was not localized: %#v", q.Choices[1].Then)
	}
}
