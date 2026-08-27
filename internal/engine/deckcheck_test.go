package engine

// 组牌校验的规则用例。卡牌全部从内嵌卡库按谓词挑选，不硬编码整副清单，
// 只硬编码官方规则语义（英雄套装、派系、复制上限、各身份骑手）。

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// addHeroSet 把英雄套装按印刷张数整卡放进 slots（模拟合法导入）。
func addHeroSet(t *testing.T, slots map[string]int, base string) {
	t.Helper()
	hero, ok := DB.Lookup(data.HeroSideCode(base))
	if !ok {
		t.Fatalf("unknown hero %s", base)
	}
	for _, d := range DB.InSet(hero.CardSet) {
		if d.Category == data.CategoryPlayer && deckCardType(d.Type) && d.Quantity > 0 {
			slots[d.Code] = d.Quantity
		}
	}
}

func deckSize(slots map[string]int) int {
	n := 0
	for code, c := range slots {
		if d, ok := DB.Lookup(code); ok && deckCardType(d.Type) {
			n += c
		}
	}
	return n
}

// padDeck 从满足谓词的卡里补到多 n 张，尊重每牌名上限（perTitle>0 时覆
// 盖默认 3/1），跳过已超限的牌名；池不够时 Fail。
func padDeck(t *testing.T, slots map[string]int, n, perTitle int, pred func(*data.CardDef) bool) {
	t.Helper()
	owners := heroSetOwners()
	used := map[string]int{}
	for code, c := range slots {
		if d, ok := DB.Lookup(code); ok {
			used[d.EName] += c
		}
	}
	got := 0
	for _, d := range DB.All() {
		if got >= n {
			return
		}
		if !deckCardType(d.Type) || d.Quantity <= 0 || !pred(d) {
			continue
		}
		if owner := owners[d.CardSet]; owner != "" {
			continue // 别的英雄的套装卡不进填充池
		}
		cap := perTitle
		if cap <= 0 {
			cap = 3
			if d.Unique {
				cap = 1
			}
		}
		if used[d.EName] >= cap {
			continue
		}
		add := cap - used[d.EName]
		if got+add > n {
			add = n - got
		}
		slots[d.Code] += add
		used[d.EName] += add
		got += add
	}
	if got < n {
		t.Fatalf("card pool exhausted: need %d more cards, got %d", n, got)
	}
}

func aspectPred(aspect string) func(*data.CardDef) bool {
	return func(d *data.CardDef) bool { return d.Aspect == aspect }
}

func findCard(t *testing.T, pred func(*data.CardDef) bool) *data.CardDef {
	t.Helper()
	for _, d := range DB.All() {
		if pred(d) {
			return d
		}
	}
	t.Fatal("no card matches the predicate")
	return nil
}

func hasIssue(t *testing.T, issues []DeckIssue, key string) DeckIssue {
	t.Helper()
	for _, is := range issues {
		if is.Key == key {
			return is
		}
	}
	t.Fatalf("expected issue %q, got %v", key, issues)
	return DeckIssue{}
}

func noIssues(t *testing.T, issues []DeckIssue) {
	t.Helper()
	if len(issues) != 0 {
		t.Fatalf("expected a legal deck, got %v", issues)
	}
}

// legalSpiderManDeck returns a rulebook-legal Spider-Man deck (exact set +
// justice + basics, 40+).
func legalSpiderManDeck(t *testing.T) map[string]int {
	t.Helper()
	slots := map[string]int{}
	addHeroSet(t, slots, "01001")
	padDeck(t, slots, 10, 0, aspectPred("justice"))
	padDeck(t, slots, 40-deckSize(slots), 0, aspectPred("basic"))
	return slots
}

func TestValidateDeckLegal(t *testing.T) {
	noIssues(t, ValidateDeck("01001a", legalSpiderManDeck(t)))
}

func TestValidateDeckTooSmall(t *testing.T) {
	slots := legalSpiderManDeck(t)
	drop := 3
	for code := range slots {
		if drop == 0 {
			break
		}
		if d, ok := DB.Lookup(code); ok && d.Aspect == "basic" && slots[code] > 0 {
			delete(slots, code)
			drop--
		}
	}
	if deckSize(slots) >= 40 {
		t.Fatalf("fixture should shrink below 40, got %d", deckSize(slots))
	}
	is := hasIssue(t, ValidateDeck("01001a", slots), "tooSmall")
	if is.N != deckMinSize || is.M != deckSize(slots) {
		t.Fatalf("tooSmall params: got %+v", is)
	}
}

func TestValidateDeckTooBig(t *testing.T) {
	slots := legalSpiderManDeck(t)
	padDeck(t, slots, 11, 0, aspectPred("justice")) // push past 50
	is := hasIssue(t, ValidateDeck("01001a", slots), "tooBig")
	if is.N != deckMaxSize {
		t.Fatalf("tooBig params: got %+v", is)
	}
}

func TestValidateDeckHeroSetExact(t *testing.T) {
	slots := legalSpiderManDeck(t)
	// 拿掉一张套装卡 → setMissing（N=印刷张数）。
	var victim *data.CardDef
	for code := range slots {
		if d, ok := DB.Lookup(code); ok && d.CardSet == "spider_man" && slots[code] > 1 {
			victim = d
			break
		}
	}
	if victim == nil {
		t.Fatal("no multi-copy hero set card found")
	}
	slots[victim.Code]--
	is := hasIssue(t, ValidateDeck("01001a", slots), "setCount")
	if is.Card != victim.Code || is.N != victim.Quantity || is.M != victim.Quantity-1 {
		t.Fatalf("setCount params: got %+v", is)
	}
	// 整卡移除 → setMissing。
	delete(slots, victim.Code)
	is = hasIssue(t, ValidateDeck("01001a", slots), "setMissing")
	if is.Card != victim.Code || is.N != victim.Quantity {
		t.Fatalf("setMissing params: got %+v", is)
	}
}

func TestValidateDeckForeignHeroSetCard(t *testing.T) {
	slots := legalSpiderManDeck(t)
	// 蛛女的套装事件带派系印刷，仍不得混进别人牌组。
	venomBlast := findCard(t, func(d *data.CardDef) bool {
		return d.CardSet == "spider_woman" && d.Aspect == "aggression"
	})
	slots[venomBlast.Code] = 1
	is := hasIssue(t, ValidateDeck("01001a", slots), "cardIllegal")
	if is.Card != venomBlast.Code {
		t.Fatalf("cardIllegal params: got %+v", is)
	}
}

func TestValidateDeckEncounterCardIllegal(t *testing.T) {
	slots := legalSpiderManDeck(t)
	moonKnight := findCard(t, func(d *data.CardDef) bool {
		return d.Code == "04097" // 遭遇模块的 Moon Knight 盟友
	})
	slots[moonKnight.Code] = 1
	hasIssue(t, ValidateDeck("01001a", slots), "cardIllegal")
}

func TestValidateDeckWrongAspect(t *testing.T) {
	slots := legalSpiderManDeck(t) // 所选派系 justice
	agg := findCard(t, func(d *data.CardDef) bool {
		return d.Aspect == "aggression" && heroSetOwners()[d.CardSet] == "" && !d.Unique
	})
	slots[agg.Code] = 1
	is := hasIssue(t, ValidateDeck("01001a", slots), "wrongAspect")
	if is.Aspect != "aggression" {
		t.Fatalf("wrongAspect params: got %+v", is)
	}
}

func TestValidateDeckCopyLimits(t *testing.T) {
	slots := legalSpiderManDeck(t)
	energy := findCard(t, func(d *data.CardDef) bool {
		return d.EName == "Energy" && d.Aspect == "basic"
	})
	slots[energy.Code] = 4
	is := hasIssue(t, ValidateDeck("01001a", slots), "copyLimit")
	if is.N != 3 || is.M != 4 {
		t.Fatalf("non-unique copyLimit params: got %+v", is)
	}
	// 唯一卡至多 1 张。
	uniq := findCard(t, func(d *data.CardDef) bool {
		return d.Aspect == "basic" && d.Unique && heroSetOwners()[d.CardSet] == ""
	})
	slots[uniq.Code] = 2
	hasIssue(t, ValidateDeck("01001a", slots), "copyLimit")
}

func TestValidateDeckIdentityMismatch(t *testing.T) {
	slots := map[string]int{"01001a": 1} // 蜘蛛侠身份面混进 Wolverine 牌组
	addHeroSet(t, slots, "35001")
	is := hasIssue(t, ValidateDeck("35001a", slots), "identityMismatch")
	if is.Card != "01001a" {
		t.Fatalf("identityMismatch params: got %+v", is)
	}
}

func TestValidateDeckSpiderWomanTwoAspects(t *testing.T) {
	build := func() map[string]int {
		slots := map[string]int{}
		addHeroSet(t, slots, "04031")
		padDeck(t, slots, 8, 0, aspectPred("aggression"))
		padDeck(t, slots, 8, 0, aspectPred("justice"))
		padDeck(t, slots, 40-deckSize(slots), 0, aspectPred("basic"))
		return slots
	}
	noIssues(t, ValidateDeck("04031a", build()))

	// 两个派系数目不等 → aspectsUnequal。
	slots := build()
	padDeck(t, slots, 1, 0, aspectPred("aggression"))
	hasIssue(t, ValidateDeck("04031a", slots), "aspectsUnequal")

	// 三个派系 → tooManyAspects。
	slots = build()
	padDeck(t, slots, 1, 0, aspectPred("leadership"))
	is := hasIssue(t, ValidateDeck("04031a", slots), "tooManyAspects")
	if is.N != 2 {
		t.Fatalf("tooManyAspects params: got %+v", is)
	}
}

func TestValidateDeckAdamWarlock(t *testing.T) {
	build := func() map[string]int {
		slots := map[string]int{}
		addHeroSet(t, slots, "21031")
		for _, a := range []string{"aggression", "justice", "leadership", "protection"} {
			// UniqueAll 把每牌名压到 1 张，填充也要按 1 张上限。
			padDeck(t, slots, 4, 1, aspectPred(a))
		}
		padDeck(t, slots, 40-deckSize(slots), 1, aspectPred("basic"))
		return slots
	}
	noIssues(t, ValidateDeck("21031a", build()))

	// 数量不等 → aspectsUnequal。
	slots := build()
	padDeck(t, slots, 1, 0, aspectPred("justice"))
	hasIssue(t, ValidateDeck("21031a", slots), "aspectsUnequal")

	// UniqueAll：非套装卡至多 1 张。
	slots = build()
	energy := findCard(t, func(d *data.CardDef) bool { return d.EName == "Energy" })
	slots[energy.Code]++
	is := hasIssue(t, ValidateDeck("21031a", slots), "copyLimit")
	if is.N != 1 || is.M != 2 {
		t.Fatalf("warlock copyLimit params: got %+v", is)
	}
}

func TestValidateDeckCyclopsXMenAllies(t *testing.T) {
	build := func() map[string]int {
		slots := map[string]int{}
		addHeroSet(t, slots, "33001")
		padDeck(t, slots, 40-deckSize(slots)-2, 0, aspectPred("leadership"))
		return slots
	}
	// 骑手：任意派系的 X-Men 盟友合法。
	xAlly := findCard(t, func(d *data.CardDef) bool {
		return d.Type == "ally" && d.HasTrait("x-men") && d.Aspect == "aggression" && heroSetOwners()[d.CardSet] == ""
	})
	slots := build()
	slots[xAlly.Code] = 1
	padDeck(t, slots, 40-deckSize(slots), 0, aspectPred("basic"))
	noIssues(t, ValidateDeck("33001a", slots))

	// 非 X-Men 的 aggression 卡仍然违规。
	slots = build()
	agg := findCard(t, func(d *data.CardDef) bool {
		return d.Aspect == "aggression" && d.Type == "ally" && !d.HasTrait("x-men") && heroSetOwners()[d.CardSet] == ""
	})
	slots[agg.Code] = 1
	padDeck(t, slots, 40-deckSize(slots), 0, aspectPred("basic"))
	hasIssue(t, ValidateDeck("33001a", slots), "wrongAspect")
}

func TestValidateDeckCableSideSchemes(t *testing.T) {
	slots := map[string]int{}
	addHeroSet(t, slots, "40001")
	padDeck(t, slots, 40-deckSize(slots)-1, 0, aspectPred("justice"))
	scheme := findCard(t, func(d *data.CardDef) bool {
		return d.Type == "player_side_scheme" && d.Aspect == "aggression"
	})
	slots[scheme.Code] = 1
	noIssues(t, ValidateDeck("40001a", slots))
}

func TestValidateDeckWonderManEnergyEvents(t *testing.T) {
	build := func() map[string]int {
		slots := map[string]int{}
		addHeroSet(t, slots, "58001")
		padDeck(t, slots, 40-deckSize(slots), 0, aspectPred("justice"))
		return slots
	}
	energyEvent := findCard(t, func(d *data.CardDef) bool {
		return d.Type == "event" && d.Aspect == "aggression" && heroSetOwners()[d.CardSet] == "" &&
			len(d.Resources) > 0 && containsString(d.Resources, "energy")
	})
	slots := build()
	slots[energyEvent.Code] = 1
	noIssues(t, ValidateDeck("58001a", slots))

	// 无能量图标的事件不豁免。
	plain := findCard(t, func(d *data.CardDef) bool {
		return d.Type == "event" && d.Aspect == "aggression" && heroSetOwners()[d.CardSet] == "" &&
			!containsString(d.Resources, "energy")
	})
	slots = build()
	slots[plain.Code] = 1
	hasIssue(t, ValidateDeck("58001a", slots), "wrongAspect")
}

func TestValidateDeckGamoraSixEvents(t *testing.T) {
	// 6 张 attack/thwart 事件豁免；第 7 张触发 exceptCap。
	attackEvents := func() []*data.CardDef {
		var out []*data.CardDef
		for _, d := range DB.All() {
			if d.Type == "event" && d.Aspect == "justice" && (d.HasTrait("attack") || d.HasTrait("thwart")) && heroSetOwners()[d.CardSet] == "" {
				out = append(out, d)
			}
		}
		return out
	}
	pool := attackEvents()
	if len(pool) < 7 {
		t.Fatalf("attack/thwart justice event pool too small: %d", len(pool))
	}
	build := func(n int) map[string]int {
		slots := map[string]int{}
		addHeroSet(t, slots, "18001")
		padDeck(t, slots, 40-deckSize(slots)-n, 0, aspectPred("aggression"))
		for _, d := range pool[:n] {
			slots[d.Code]++
		}
		return slots
	}
	noIssues(t, ValidateDeck("18001a", build(6)))
	is := hasIssue(t, ValidateDeck("18001a", build(7)), "exceptCap")
	if is.N != 6 || is.M != 7 {
		t.Fatalf("exceptCap params: got %+v", is)
	}
}

func TestValidateDeckDeadpoolPoolCards(t *testing.T) {
	// Deadpool 牌组里「池」卡合法；别的英雄不行。
	slots := map[string]int{}
	addHeroSet(t, slots, "44001")
	poolAlly := findCard(t, func(d *data.CardDef) bool {
		return d.Aspect == "pool" && !d.Unique && heroSetOwners()[d.CardSet] == ""
	})
	slots[poolAlly.Code] = 1
	padDeck(t, slots, 40-deckSize(slots), 0, aspectPred("aggression"))
	noIssues(t, ValidateDeck("44001a", slots))

	slots2 := legalSpiderManDeck(t)
	slots2[poolAlly.Code] = 1
	is := hasIssue(t, ValidateDeck("01001a", slots2), "poolWrongHero")
	if is.Card != poolAlly.Code {
		t.Fatalf("poolWrongHero params: got %+v", is)
	}
}

func TestValidateDeckUnknownHero(t *testing.T) {
	issues := ValidateDeck("99999x", map[string]int{"01088": 3})
	if len(issues) != 1 || issues[0].Key != "identityUnknown" {
		t.Fatalf("expected identityUnknown, got %v", issues)
	}
}

// 骑手解析：加载期把身份卡文本解析成结构化字段。
func TestDeckRiderParsing(t *testing.T) {
	sw, _ := DB.Lookup("04031b")
	if sw.AspectMode != "two_equal" {
		t.Fatalf("Spider-Woman aspect mode: %q", sw.AspectMode)
	}
	aw, _ := DB.Lookup("21031b")
	if aw.AspectMode != "four_equal" || !aw.UniqueAll {
		t.Fatalf("Adam Warlock riders: mode=%q uniqueAll=%v", aw.AspectMode, aw.UniqueAll)
	}
	cy, _ := DB.Lookup("33001b")
	if cy.AspectException == nil || cy.AspectException.Trait != "x-men" || cy.AspectException.CardType != "ally" {
		t.Fatalf("Cyclops exception: %+v", cy.AspectException)
	}
	cb, _ := DB.Lookup("40001b")
	if cb.AspectException == nil || cb.AspectException.CardType != "player_side_scheme" {
		t.Fatalf("Cable exception: %+v", cb.AspectException)
	}
	wm, _ := DB.Lookup("58001b")
	if wm.AspectException == nil || !wm.AspectException.EnergyEvents {
		t.Fatalf("Wonder Man exception: %+v", wm.AspectException)
	}
	ga, _ := DB.Lookup("18001b")
	if ga.AspectException == nil || ga.AspectException.Total != 6 || len(ga.AspectException.EventTraits) != 2 {
		t.Fatalf("Gamora exception: %+v", ga.AspectException)
	}
	mh, _ := DB.Lookup("50001b")
	if mh.AspectException == nil || mh.AspectException.Titles != 3 || mh.AspectException.Trait != "s.h.i.e.l.d." {
		t.Fatalf("Maria Hill exception: %+v", mh.AspectException)
	}
	// 普通英雄没有骑手。
	sm, _ := DB.Lookup("01001b")
	if sm.AspectMode != "" || sm.AspectException != nil || sm.UniqueAll {
		t.Fatalf("Spider-Man should have no riders: %+v", sm)
	}
	// Pool 派系解析。
	dp, _ := DB.Lookup("44013")
	if dp.Aspect != "pool" {
		t.Fatalf("Dogpool aspect: %q", dp.Aspect)
	}
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
