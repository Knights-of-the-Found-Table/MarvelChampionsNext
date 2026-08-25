package engine

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

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

// englishMessageKeys lets legacy call sites keep their original English
// format string while still benefiting from the message catalog. New code
// should use T/Tf directly; this reverse index is the compatibility bridge
// while card packages are migrated incrementally.
var englishMessageKeys = func() map[string]string {
	out := make(map[string]string, len(messages))
	for key, byLang := range messages {
		if en := byLang[LangEn]; en != "" {
			out[en] = key
		}
	}
	return out
}()

var formatDirective = regexp.MustCompile(`%(?:\[(\d+)\])?([a-zA-Z])`)

type renderedMessagePattern struct {
	re       *regexp.Regexp
	argOrder []int
	zh       string
	weight   int
}

func compileRenderedPattern(en, zh string) (renderedMessagePattern, bool) {
	locs := formatDirective.FindAllStringSubmatchIndex(en, -1)
	if len(locs) == 0 || zh == "" {
		return renderedMessagePattern{}, false
	}
	var pattern strings.Builder
	pattern.WriteByte('^')
	last, nextArg := 0, 1
	order := make([]int, 0, len(locs))
	for _, loc := range locs {
		pattern.WriteString(regexp.QuoteMeta(en[last:loc[0]]))
		token := en[loc[0]:loc[1]]
		parts := formatDirective.FindStringSubmatch(token)
		arg := nextArg
		if parts[1] != "" {
			arg, _ = strconv.Atoi(parts[1])
		} else {
			nextArg++
		}
		order = append(order, arg)
		switch parts[2] {
		case "d", "i", "u":
			pattern.WriteString(`(-?\d+)`)
		case "f", "e", "E", "g", "G":
			pattern.WriteString(`([-+]?(?:\d+(?:\.\d*)?|\.\d+))`)
		default:
			pattern.WriteString(`(.+?)`)
		}
		last = loc[1]
	}
	pattern.WriteString(regexp.QuoteMeta(en[last:]))
	pattern.WriteByte('$')
	literalWeight := len(en)
	for _, loc := range locs {
		literalWeight -= loc[1] - loc[0]
	}
	return renderedMessagePattern{
		re: regexp.MustCompile(pattern.String()), argOrder: order, zh: zh, weight: literalWeight,
	}, true
}

// Rendered patterns are compiled lazily. Most package tests run in English and
// never need the compatibility matcher; avoiding thousands of regex compiles
// in every test binary keeps the normal engine test suite fast.
var (
	renderedPatternsOnce sync.Once
	renderedPatterns     []renderedMessagePattern
)

func getRenderedMessagePatterns() []renderedMessagePattern {
	renderedPatternsOnce.Do(func() {
		out := make([]renderedMessagePattern, 0, len(messages)+len(legacyMessages))
		add := func(en, zh string) {
			if p, ok := compileRenderedPattern(en, zh); ok {
				out = append(out, p)
			}
		}
		for _, byLang := range messages {
			add(byLang[LangEn], byLang[LangZh])
		}
		for en, zh := range legacyMessages {
			add(en, zh)
		}
		// More specific formats win when two entries can match the same text.
		sort.Slice(out, func(i, j int) bool { return out[i].weight > out[j].weight })
		renderedPatterns = out
	})
	return renderedPatterns
}

type legacyTextRule struct {
	re      *regexp.Regexp
	replace string
}

// genericLegacyTextRules cover high-frequency labels assembled through string
// concatenation (for example "Discard " + a translated card name). Full card-
// specific prose belongs in legacyMessages; these rules stay deliberately
// narrow so names and rules text are never machine-translated at runtime.
var genericLegacyTextRules = []legacyTextRule{
	{regexp.MustCompile(`^Exhaust (.+) → remove (.+) from the game$`), `横置 $1 → 将 $2 移出游戏`},
	{regexp.MustCompile(`^Shuffle in (.+)$`), `将 $1 洗入牌库`},
	{regexp.MustCompile(`^Shuffle (.+) into the encounter deck$`), `将 $1 洗入遭遇牌库`},
	{regexp.MustCompile(`^Add (.+) to your hand$`), `将 $1 加入手牌`},
	{regexp.MustCompile(`^Return (.+) to (?:your )?hand$`), `将 $1 收回手牌`},
	{regexp.MustCompile(`^Return (.+) to your deck$`), `将 $1 放回牌库`},
	{regexp.MustCompile(`^Put (.+) on top(?: of your deck)?$`), `将 $1 置于牌库顶`},
	{regexp.MustCompile(`^Put (.+) into play$`), `将 $1 置入场上`},
	{regexp.MustCompile(`^Remove (\d+) threat from (.+)$`), `从 $2 上移除 $1 点威胁`},
	{regexp.MustCompile(`^Deal (\d+) damage to (.+)$`), `对 $2 造成 $1 点伤害`},
	{regexp.MustCompile(`^Heal (\d+) damage from (.+)$`), `为 $2 治疗 $1 点伤害`},
	{regexp.MustCompile(`^Draw (\d+) cards?$`), `抽 $1 张牌`},
	{regexp.MustCompile(`^Tough on (.+)$`), `给予 $1 坚韧`},
	{regexp.MustCompile(`^Defend with (.+)$`), `使用 $1 防御`},
	{regexp.MustCompile(`^Discard (.+)$`), `弃置 $1`},
	{regexp.MustCompile(`^Ready (.+)$`), `重整 $1`},
	{regexp.MustCompile(`^Exhaust (.+)$`), `横置 $1`},
	{regexp.MustCompile(`^Take (.+)$`), `拿取 $1`},
	{regexp.MustCompile(`^Play (.+)$`), `打出 $1`},
	{regexp.MustCompile(`^Use (.+)$`), `使用 $1`},
	{regexp.MustCompile(`^Spend (.+)$`), `支付 $1`},
	{regexp.MustCompile(`^Attach to (.+)$`), `附加到 $1`},
	{regexp.MustCompile(`^Attach (.+)$`), `附加 $1`},
	{regexp.MustCompile(`^Stun (.+)$`), `使 $1 眩晕`},
	{regexp.MustCompile(`^Confuse (.+)$`), `使 $1 困惑`},
	{regexp.MustCompile(`^Reveal (.+)$`), `揭示 $1`},
	{regexp.MustCompile(`^Tuck (.+)$`), `将 $1 置于牌下`},
	{regexp.MustCompile(`^Control (.+)$`), `控制 $1`},
	{regexp.MustCompile(`^Flip (.+)$`), `翻面 $1`},
	{regexp.MustCompile(`^Remove (.+)$`), `移除 $1`},
	{regexp.MustCompile(`^Heal (.+)$`), `治疗 $1`},
	{regexp.MustCompile(`^Draw (.+)$`), `抽取 $1`},
	{regexp.MustCompile(`^Tough: (.+)$`), `坚韧：$1`},
	{regexp.MustCompile(`^Stun: (.+)$`), `眩晕：$1`},
	{regexp.MustCompile(`^Confuse: (.+)$`), `困惑：$1`},
	{regexp.MustCompile(`^Name: (.+)$`), `宣言：$1`},
	{regexp.MustCompile(`^Search — (.+)$`), `搜寻——$1`},
}

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

// localizeLegacyFormat translates an old English source/format string when it
// already exists in the catalog. It deliberately leaves unknown text intact:
// that makes the bridge safe for player names, card names, and untranslated
// card-specific prose while the audit continues.
func localizeLegacyFormat(format string) string {
	if uiLang == LangEn {
		return format
	}
	if key, ok := englishMessageKeys[format]; ok {
		return msgFormat(key)
	}
	if zh, ok := legacyMessages[format]; ok {
		return zh
	}
	return format
}

// localizeLegacyRenderedText translates catalog messages that were formatted
// before localization (notably text restored from an older save). Captured
// arguments are treated as opaque strings, then recursively localized so a
// nested reason such as "Defeat: The main scheme completed" is fully Chinese.
func localizeLegacyRenderedText(text string) string {
	if uiLang == LangEn || text == "" {
		return text
	}
	if key, ok := englishMessageKeys[text]; ok {
		return msgFormat(key)
	}
	if zh, ok := legacyMessages[text]; ok {
		return zh
	}
	patterns := getRenderedMessagePatterns()
	// Specific catalog formats beat generic action rules. Very broad formats
	// such as "Heal %s" are deferred so they cannot swallow the more useful
	// "Heal 2 damage from <name>" concatenation rule.
	for _, candidate := range patterns {
		if candidate.weight < 8 {
			continue
		}
		if translated, ok := localizeRenderedCandidate(candidate, text); ok {
			return translated
		}
	}
	for _, rule := range genericLegacyTextRules {
		if rule.re.MatchString(text) {
			return rule.re.ReplaceAllString(text, rule.replace)
		}
	}
	for _, candidate := range patterns {
		if candidate.weight >= 8 {
			continue
		}
		if translated, ok := localizeRenderedCandidate(candidate, text); ok {
			return translated
		}
	}
	return text
}

func localizeRenderedCandidate(candidate renderedMessagePattern, text string) (string, bool) {
	matches := candidate.re.FindStringSubmatch(text)
	if matches == nil {
		return "", false
	}
	args := make(map[int]string, len(candidate.argOrder))
	for i, arg := range candidate.argOrder {
		args[arg] = localizeLegacyRenderedText(matches[i+1])
	}
	return renderCapturedFormat(candidate.zh, args), true
}

func renderCapturedFormat(format string, args map[int]string) string {
	locs := formatDirective.FindAllStringSubmatchIndex(format, -1)
	if len(locs) == 0 {
		return format
	}
	var out strings.Builder
	last, nextArg := 0, 1
	for _, loc := range locs {
		out.WriteString(format[last:loc[0]])
		token := format[loc[0]:loc[1]]
		parts := formatDirective.FindStringSubmatch(token)
		arg := nextArg
		if parts[1] != "" {
			arg, _ = strconv.Atoi(parts[1])
		} else {
			nextArg++
		}
		out.WriteString(args[arg])
		last = loc[1]
	}
	out.WriteString(format[last:])
	return out.String()
}

// localizeLegacyQuestion applies the same bridge to prompts and choice labels.
// Ask/AskN receive many choices assembled by card packages, so centralizing the
// compatibility here covers old callers without changing their answer IDs or
// serialized message payloads.
func localizeLegacyQuestion(prompt string, choices []Choice) (string, []Choice) {
	prompt = localizeLegacyRenderedText(prompt)
	for i := range choices {
		choices[i].Label = localizeLegacyRenderedText(choices[i].Label)
		if choices[i].Then != nil {
			localizeQuestionTree(choices[i].Then)
		}
	}
	return prompt, choices
}

func localizeQuestionTree(q *Question) {
	if q == nil {
		return
	}
	q.Prompt, q.Choices = localizeLegacyQuestion(q.Prompt, q.Choices)
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
