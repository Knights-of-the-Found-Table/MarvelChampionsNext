package engine

// 规则书「PLAYER DECK CUSTOMIZATION RULES」的组牌校验：
//   - 恰好一张身份卡（slots 里混入其他英雄的身份面即违规）；
//   - 牌组 40-50 张（身份卡不计入）；
//   - 英雄套装卡必须按印刷张数整卡在场（套装要求优先于复制上限，
//     例如 Hex Bolt / Always Be Running 各印 4 张）；
//   - 至多一个派系 + 基础卡；身份卡骑手（蜘蛛女双派系、魔士亚当四
//     派系、独眼龙/卡魔拉/希尔/奇迹人/Cable 的例外卡）按 data 层解析
//     好的结构化字段放行；
//   - 非唯一卡按牌名至多 3 张、唯一卡至多 1 张；死侍的「池」卡只在
//     他的牌组里合法。
//
// 校验是建议性的：不合规牌组仍可导入保存（之后支持继续编辑），只是不
// 能用于开始对局。全部只读结构化字段，绝不解析卡面文本（规约见
// data/types.go Keywords 注释）。

import (
	"sort"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// DeckIssue 是一条规则违反。Key 对应前端 i18n/messages.json 的
// deck.issue.* 键；Card/Title 指向违规卡（Title 是稳定英文印名，仅作
// 兜底文案，展示名由前端按 code 本地化）；N/M 携带数量参数。
type DeckIssue struct {
	Key    string `json:"key"`
	Card   string `json:"card,omitempty"`
	Title  string `json:"title,omitempty"`
	Aspect string `json:"aspect,omitempty"`
	N      int    `json:"n,omitempty"`
	M      int    `json:"m,omitempty"`
}

const (
	deckMinSize = 40
	deckMaxSize = 50
)

// heroSetOwners 映射英雄套装代码 → 英雄 base：带这些套装代码的非套装卡
// 属于别的英雄（蛛女那 4 张带派系印刷的套装事件是唯一案例）。
func heroSetOwners() map[string]string {
	owners := map[string]string{}
	for _, d := range DB.All() {
		if d.Type == "hero" && d.Side == "a" {
			owners[d.CardSet] = BaseCodeOf(d.Code)
		}
	}
	return owners
}

// deckCardType 报告该类型卡牌能否进 40-50 牌组：身份双面单独计，责任卡
// 由对局自身发放（引擎组装牌组时同样跳过这三类）。
func deckCardType(t string) bool {
	return t != "hero" && t != "alter_ego" && t != "obligation"
}

// ValidateDeck checks a deck against the rulebook; an empty result means
// the deck is legal. Issues are sorted for stable display and tests.
func ValidateDeck(investigatorCode string, slots map[string]int) []DeckIssue {
	var issues []DeckIssue
	add := func(issue DeckIssue) { issues = append(issues, issue) }

	base := BaseCodeOf(investigatorCode)
	heroDef, ok := DB.Lookup(data.HeroSideCode(base))
	if !ok {
		return []DeckIssue{{Key: "identityUnknown"}}
	}
	// 组牌骑手印在化身面上。
	aeDef, _ := DB.Lookup(data.AlterEgoSideCode(base))
	// 英雄套装归属：别的英雄的专属卡不能进牌组（含蛛女那 4 张带派系
	// 印刷的套装事件——faction 有派系但套装属于别人）。
	heroSets := heroSetOwners()

	// ---- 恰好一张身份卡：slots 里的身份面必须都属于所选英雄 ----
	for _, code := range sortedSlotCodes(slots) {
		d, ok := DB.Lookup(code)
		if !ok || (d.Type != "hero" && d.Type != "alter_ego") {
			continue
		}
		if BaseCodeOf(d.Code) != base {
			add(DeckIssue{Key: "identityMismatch", Card: d.Code, Title: d.EName, M: slots[code]})
		}
	}

	// ---- 英雄套装：套装内每张可组卡按印刷张数整卡在场 ----
	heroSet := map[string]*data.CardDef{}
	for _, d := range DB.InSet(heroDef.CardSet) {
		if d.Category == data.CategoryPlayer && deckCardType(d.Type) && d.Quantity > 0 {
			heroSet[d.Code] = d
		}
	}
	for _, d := range sortedSetDefs(heroSet) {
		n := slots[d.Code]
		switch {
		case n == 0:
			add(DeckIssue{Key: "setMissing", Card: d.Code, Title: d.EName, N: d.Quantity})
		case n != d.Quantity:
			add(DeckIssue{Key: "setCount", Card: d.Code, Title: d.EName, N: d.Quantity, M: n})
		}
	}

	// ---- 牌组张数 ----
	total := 0
	for _, code := range sortedSlotCodes(slots) {
		if d, ok := DB.Lookup(code); ok && deckCardType(d.Type) {
			total += slots[code]
		}
	}
	if total < deckMinSize {
		add(DeckIssue{Key: "tooSmall", N: deckMinSize, M: total})
	}
	if total > deckMaxSize {
		add(DeckIssue{Key: "tooBig", N: deckMaxSize, M: total})
	}

	// ---- 派系纪律 + 复制上限（均只看非套装卡）----
	mode := ""
	var exception *data.AspectException
	if aeDef != nil {
		mode = aeDef.AspectMode
		exception = aeDef.AspectException
	}

	aspectCount := map[string]int{}
	type exCard struct {
		def *data.CardDef
		n   int
	}
	var exCards []exCard
	copies := map[string]int{} // 牌名 -> 总张数
	titleUnique := map[string]bool{}
	rep := map[string]string{} // 牌名 -> 代表 code（排序最小，输出稳定）

	for _, code := range sortedSlotCodes(slots) {
		n := slots[code]
		d, ok := DB.Lookup(code)
		if !ok || !deckCardType(d.Type) || n <= 0 {
			continue
		}
		if _, inSet := heroSet[code]; inSet {
			continue
		}
		if owner := heroSets[d.CardSet]; owner != "" && owner != base {
			add(DeckIssue{Key: "cardIllegal", Card: d.Code, Title: d.EName, M: n})
			continue
		}
		// 派系分类。
		switch d.Aspect {
		case "":
			// faction 不是玩家派系的卡：遭遇/战役模块卡与英雄专属
			// 侧卡组（如召唤牌、Frostbite），不能组进牌组。
			add(DeckIssue{Key: "cardIllegal", Card: d.Code, Title: d.EName, M: n})
			continue
		case "basic":
		case "pool":
			if heroDef.CardSet != "deadpool" {
				add(DeckIssue{Key: "poolWrongHero", Card: d.Code, Title: d.EName, M: n})
				continue
			}
		default:
			aspectCount[d.Aspect] += n
		}
		// 复制上限按牌名累计（基础卡与派系卡都要算）。
		copies[d.EName] += n
		if d.Unique {
			titleUnique[d.EName] = true
		}
		if _, seen := rep[d.EName]; !seen {
			rep[d.EName] = code
		}
	}

	// 身份骑手对派系选择的整体约束。
	present := make([]string, 0, len(aspectCount))
	for a, n := range aspectCount {
		if n > 0 {
			present = append(present, a)
		}
	}
	sort.Strings(present)
	chosenAspect := ""
	switch mode {
	case "two_equal":
		if len(present) > 2 {
			add(DeckIssue{Key: "tooManyAspects", N: 2})
		}
		flagAspectsUnequal(add, aspectCount, present)
	case "four_equal":
		if len(present) != 4 {
			add(DeckIssue{Key: "aspectsUnequal"})
		}
		flagAspectsUnequal(add, aspectCount, present)
	default:
		for _, a := range present {
			if chosenAspect == "" || aspectCount[a] > aspectCount[chosenAspect] {
				chosenAspect = a
			}
		}
	}

	// 逐卡派系检查：模式卡牌（two_equal/four_equal）的合法性由上面的
	// 整体数量校验判定，不再逐卡报；其余模式里非所选派系的卡必须命中
	// 骑手豁免。
	for _, code := range sortedSlotCodes(slots) {
		n := slots[code]
		d, ok := DB.Lookup(code)
		if !ok || !deckCardType(d.Type) || n <= 0 {
			continue
		}
		if _, inSet := heroSet[code]; inSet {
			continue
		}
		if owner := heroSets[d.CardSet]; owner != "" && owner != base {
			continue // 已在分类轮报过 cardIllegal
		}
		switch d.Aspect {
		case "", "basic", "pool":
			continue
		}
		if mode != "" || d.Aspect == chosenAspect {
			continue
		}
		if exception.Matches(d) {
			exCards = append(exCards, exCard{d, n})
			continue
		}
		add(DeckIssue{Key: "wrongAspect", Card: d.Code, Title: d.EName, Aspect: d.Aspect, M: n})
	}

	// 骑手豁免卡的数量上限。
	if exception != nil && len(exCards) > 0 {
		totalCopies := 0
		titles := map[string]bool{}
		for _, e := range exCards {
			totalCopies += e.n
			titles[e.def.EName] = true
		}
		if exception.Total > 0 && totalCopies > exception.Total {
			add(DeckIssue{Key: "exceptCap", N: exception.Total, M: totalCopies})
		}
		if exception.Titles > 0 && len(titles) > exception.Titles {
			add(DeckIssue{Key: "exceptCap", N: exception.Titles, M: len(titles)})
		}
	}

	// 按牌名的复制上限：唯一卡 1 张、非唯一卡 3 张；魔士亚当把全部非
	// 套装卡压到 1 张。
	uniqueAll := aeDef != nil && aeDef.UniqueAll
	for title, n := range copies {
		cap := 3
		if titleUnique[title] || uniqueAll {
			cap = 1
		}
		if n > cap {
			add(DeckIssue{Key: "copyLimit", Card: rep[title], Title: title, N: cap, M: n})
		}
	}

	sort.Slice(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if a.Key != b.Key {
			return a.Key < b.Key
		}
		if a.Card != b.Card {
			return a.Card < b.Card
		}
		return a.Aspect < b.Aspect
	})
	return issues
}

// flagAspectsUnequal reports the aspectsUnequal issue when the present
// aspect copy counts differ (shared by the two_equal/four_equal modes).
func flagAspectsUnequal(add func(DeckIssue), counts map[string]int, present []string) {
	if len(present) < 2 {
		return
	}
	c0 := counts[present[0]]
	for _, a := range present[1:] {
		if counts[a] != c0 {
			add(DeckIssue{Key: "aspectsUnequal"})
			break
		}
	}
}

func sortedSlotCodes(slots map[string]int) []string {
	out := make([]string, 0, len(slots))
	for code := range slots {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

func sortedSetDefs(set map[string]*data.CardDef) []*data.CardDef {
	out := make([]*data.CardDef, 0, len(set))
	for _, d := range set {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}
