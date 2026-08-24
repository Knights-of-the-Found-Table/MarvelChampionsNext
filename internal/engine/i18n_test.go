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
