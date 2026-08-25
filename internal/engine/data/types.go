// Package data loads the marvelcdb.com card database snapshot embedded at
// build time and exposes normalized card definitions to the engine.
package data

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// CardDef is the normalized, engine-facing view of a marvelcdb card entry.
// Trait, keyword and encounter-set values are plain normalized strings: the
// engine never needs generated enums to support new packs.
type CardDef struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Subname  string `json:"subname,omitempty"`
	PackCode string `json:"packCode"`
	PackName string `json:"packName"`

	// Type is the marvelcdb type_code, see Category for the coarser split.
	Type string `json:"type"`
	// Side is "a" or "b" for double-sided cards (hero/alter-ego, villain
	// personas, environments), empty otherwise.
	Side     string `json:"side,omitempty"`
	LinkedTo string `json:"linkedTo,omitempty"`

	// Category is "player", "encounter" or "other".
	Category string `json:"category"`
	// Aspect is "basic", "aggression", "justice", "leadership" or
	// "protection" for generic player cards, "" for hero-specific and
	// encounter cards.
	Aspect string `json:"aspect,omitempty"`
	// CardSet is the marvelcdb card_set_code (hero set or encounter set).
	CardSet string `json:"cardSet,omitempty"`

	Cost   *int `json:"cost,omitempty"`
	Unique bool `json:"unique"`

	Traits []string `json:"traits,omitempty"`
	// Keywords are the structured form of printed keywords/rules riders
	// (Guard, Retaliate 2, Hinder 3, ...). 游戏逻辑严禁解析 Text 做判断
	// （regex/子串匹配印刷文本一律禁止，参见 engine/i18n.go 的 i18n 规约）：
	// 需要读取印刷信息时，在 normalize 的加载期解析一次成这里的结构化
	// 字段，逻辑只读字段。
	Keywords []Keyword `json:"keywords,omitempty"`

	// Resources lists printed resource icons: energy, physical, mental, wild.
	Resources []string `json:"resources,omitempty"`

	// BoostEntersPlay marks the "Boost: put this card into play" rider:
	// revealed as a boost card, it enters play instead of adding boost
	// icons (parsed at load, never re-matched on Text).
	BoostEntersPlay bool `json:"boostEntersPlay,omitempty"`

	Text      string `json:"text,omitempty"`
	Quantity  int    `json:"quantity,omitempty"`
	DeckLimit int    `json:"deckLimit,omitempty"`

	// Stats (pointers so that "absent" stays distinguishable from 0).
	HP       *int `json:"hp,omitempty"`
	Attack   *int `json:"attack,omitempty"`
	Thwart   *int `json:"thwart,omitempty"`
	Defense  *int `json:"defense,omitempty"`
	Recover  *int `json:"recover,omitempty"`
	Scheme   *int `json:"scheme,omitempty"`
	HandSize *int `json:"handSize,omitempty"`
	Boost    *int `json:"boost,omitempty"`
	Stage    *int `json:"stage,omitempty"`
	// StageLabel keeps the raw stage marker ("I", "2A", "B", ...) for exotic
	// multi-villain layouts where a plain number is not enough.
	StageLabel string `json:"stageLabel,omitempty"`

	BaseThreat       *int `json:"baseThreat,omitempty"`
	EscalationThreat *int `json:"escalationThreat,omitempty"`
	Threat           *int `json:"threat,omitempty"`
	Hazards          int  `json:"hazards,omitempty"`
	Acceleration     int  `json:"acceleration,omitempty"`

	ImageSrc     string `json:"imageSrc,omitempty"`
	BackImageSrc string `json:"backImageSrc,omitempty"`
}

// Keyword is a printed keyword such as Guard, Toughness, Quickstrike,
// Retaliate 2 or Surge. Value is only meaningful for numeric keywords.
type Keyword struct {
	Name  string `json:"name"`
	Value int    `json:"value,omitempty"`
}

func (k Keyword) String() string {
	if k.Value > 0 {
		return fmt.Sprintf("%s %d", k.Name, k.Value)
	}
	return k.Name
}

var playerTypes = map[string]bool{
	"ally": true, "event": true, "resource": true, "support": true,
	"upgrade": true, "player_side_scheme": true, "obligation": true,
}

var encounterTypes = map[string]bool{
	"villain": true, "main_scheme": true, "side_scheme": true,
	"minion": true, "treachery": true, "attachment": true, "environment": true,
}

var aspects = map[string]bool{
	"aggression": true, "justice": true, "leadership": true, "protection": true,
}

// rawCard mirrors the marvelcdb JSON schema; only fields we consume are
// declared, everything else is ignored by the decoder.
type rawCard struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Subname  string `json:"subname"`
	PackCode string `json:"pack_code"`
	PackName string `json:"pack_name"`

	TypeCode     string `json:"type_code"`
	LinkedToCode string `json:"linked_to_code"`
	DoubleSided  bool   `json:"double_sided"`
	FactionCode  string `json:"faction_code"`
	CardSetCode  string `json:"card_set_code"`

	Cost      *int   `json:"cost"`
	IsUnique  bool   `json:"is_unique"`
	Traits    string `json:"traits"`
	Text      string `json:"real_text"`
	Quantity  int    `json:"quantity"`
	DeckLimit int    `json:"deck_limit"`

	ResourceEnergy   *int `json:"resource_energy"`
	ResourcePhysical *int `json:"resource_physical"`
	ResourceMental   *int `json:"resource_mental"`
	ResourceWild     *int `json:"resource_wild"`

	Health   *int   `json:"health"`
	Attack   *int   `json:"attack"`
	Thwart   *int   `json:"thwart"`
	Defense  *int   `json:"defense"`
	Recover  *int   `json:"recover"`
	Scheme   *int   `json:"scheme"`
	HandSize *int   `json:"hand_size"`
	Boost    *int   `json:"boost"`
	Stage    string `json:"stage"`

	BaseThreat         *int `json:"base_threat"`
	EscalationThreat   *int `json:"escalation_threat"`
	Threat             *int `json:"threat"`
	SchemeHazard       *int `json:"scheme_hazard"`
	SchemeAcceleration *int `json:"scheme_acceleration"`

	ImageSrc     string `json:"imagesrc"`
	BackImageSrc string `json:"backimagesrc"`

	LinkedCard *rawCard `json:"linked_card"`
}

var (
	tagRE       = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
	retalRE     = regexp.MustCompile(`^Retaliate (\d+)`)
	hinderValRE = regexp.MustCompile(`Hinder (\d+)`)
	boostSelfRE = regexp.MustCompile(`(?i)boost: put this (?:card|minion) into play`)
	keywordSet  = []string{"Guard", "Toughness", "Quickstrike", "Surge", "Patrol", "Hazards", "ErrandOfMercy"}
	romanRE     = regexp.MustCompile(`^(?i)([IVX]+)(?:([A-Z])|\d)?$`)
	plainNumRE  = regexp.MustCompile(`^(\d+)(?:([A-Z])|\d)?$`)
	letterRE    = regexp.MustCompile(`^([A-Z])(\d)?$`)
)

// parseStage converts marvelcdb stage markers ("1", "I", "2A", "B1") into a
// numeric stage plus the original label. num is nil when the marker cannot be
// reduced to a number.
func parseStage(s string) (num *int, label string) {
	label = strings.TrimSpace(s)
	if label == "" {
		return nil, ""
	}
	if m := romanRE.FindStringSubmatch(label); m != nil {
		if n, ok := romanToInt(m[1]); ok {
			return &n, label
		}
	}
	if m := plainNumRE.FindStringSubmatch(label); m != nil {
		n, _ := strconv.Atoi(m[1])
		return &n, label
	}
	return nil, label
}

func romanToInt(s string) (int, bool) {
	values := map[rune]int{'i': 1, 'v': 5, 'x': 10}
	total, prev := 0, 0
	for _, r := range strings.ToLower(s) {
		v, ok := values[r]
		if !ok {
			return 0, false
		}
		if v > prev {
			total += v - 2*prev
		} else {
			total += v
		}
		prev = v
	}
	return total, total > 0
}

// parseKeywords extracts leading printed keywords ("Guard.", "Retaliate 2.",
// ...) from the card text after stripping formatting tags.
func parseKeywords(text string) []Keyword {
	if text == "" {
		return nil
	}
	clean := strings.TrimSpace(tagRE.ReplaceAllString(text, ""))
	if clean == "" {
		return nil
	}
	var kws []Keyword
	for {
		matched := false
		for _, kw := range keywordSet {
			if strings.HasPrefix(clean, kw+".") {
				kws = append(kws, Keyword{Name: kw})
				clean = strings.TrimSpace(strings.TrimPrefix(clean, kw+"."))
				matched = true
				break
			}
		}
		if !matched {
			if m := retalRE.FindStringSubmatch(clean); m != nil {
				v, _ := strconv.Atoi(m[1])
				kws = append(kws, Keyword{Name: "Retaliate", Value: v})
				clean = strings.TrimSpace(clean[len(m[0]):])
				if strings.HasPrefix(clean, ".") {
					clean = strings.TrimSpace(clean[1:])
				}
				matched = true
			}
		}
		if !matched {
			break
		}
	}
	return kws
}

func parseTraits(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ".") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func normalize(def *CardDef, raw rawCard) {
	def.Code = raw.Code
	def.Name = raw.Name
	def.Subname = raw.Subname
	def.PackCode = raw.PackCode
	def.PackName = raw.PackName
	def.Type = raw.TypeCode
	def.LinkedTo = raw.LinkedToCode
	def.CardSet = raw.CardSetCode
	def.Cost = raw.Cost
	def.Unique = raw.IsUnique
	def.Text = raw.Text
	def.Quantity = raw.Quantity
	def.DeckLimit = raw.DeckLimit

	def.HP = raw.Health
	def.Attack = raw.Attack
	def.Thwart = raw.Thwart
	def.Defense = raw.Defense
	def.Recover = raw.Recover
	def.Scheme = raw.Scheme
	def.HandSize = raw.HandSize
	def.Boost = raw.Boost
	def.Stage, def.StageLabel = parseStage(raw.Stage)
	def.BaseThreat = raw.BaseThreat
	def.EscalationThreat = raw.EscalationThreat
	def.Threat = raw.Threat
	if raw.SchemeHazard != nil && *raw.SchemeHazard > 0 {
		def.Hazards = *raw.SchemeHazard
	}
	if raw.SchemeAcceleration != nil && *raw.SchemeAcceleration > 0 {
		def.Acceleration = *raw.SchemeAcceleration
	}

	def.ImageSrc = raw.ImageSrc
	def.BackImageSrc = raw.BackImageSrc

	def.Traits = parseTraits(raw.Traits)
	def.Keywords = parseKeywords(raw.Text)

	// Riders the game logic keys on, parsed once at load. Hinder is
	// deliberately unanchored: Standard/Expert-mode side schemes print a
	// "Standard Mode Only." preamble before "Hinder 3", so the
	// leading-keyword strip in parseKeywords would miss them.
	if m := hinderValRE.FindStringSubmatch(raw.Text); m != nil {
		v, _ := strconv.Atoi(m[1])
		def.Keywords = append(def.Keywords, Keyword{Name: "Hinder", Value: v})
	}
	// "Boost:</b> Put this card into play" — tags stripped before matching
	// (the literal text carries a bold tag between "Boost:" and "Put").
	def.BoostEntersPlay = boostSelfRE.MatchString(tagRE.ReplaceAllString(raw.Text, ""))

	// Preserve printed resource multiplicity. Basic resource cards such as
	// Energy/Genius/Strength carry two copies of the same icon; collapsing the
	// count here makes each of them pay for only one resource.
	appendResources := func(count *int, icon string) {
		if count == nil {
			return
		}
		for range max(0, *count) {
			def.Resources = append(def.Resources, icon)
		}
	}
	appendResources(raw.ResourceEnergy, "energy")
	appendResources(raw.ResourcePhysical, "physical")
	appendResources(raw.ResourceMental, "mental")
	appendResources(raw.ResourceWild, "wild")

	switch {
	case playerTypes[raw.TypeCode]:
		def.Category = CategoryPlayer
	case encounterTypes[raw.TypeCode]:
		def.Category = CategoryEncounter
	default:
		def.Category = CategoryOther
	}

	if aspects[raw.FactionCode] {
		def.Aspect = raw.FactionCode
	} else if raw.FactionCode == "basic" {
		def.Aspect = "basic"
	}

	if raw.DoubleSided || raw.LinkedToCode != "" || hasSideSuffix(raw.Code) {
		def.Side = sideSuffix(raw.Code)
	}
}

func sideSuffix(code string) string {
	if len(code) == 6 {
		switch code[5] {
		case 'a', 'b', 'c':
			return string(code[5])
		}
	}
	return ""
}

func hasSideSuffix(code string) bool { return sideSuffix(code) != "" }

// Trait reports whether the card carries the given (case-insensitive) trait.
func (c *CardDef) HasTrait(trait string) bool {
	t := strings.ToLower(trait)
	for _, x := range c.Traits {
		if x == t {
			return true
		}
	}
	return false
}

// HasKeyword reports whether the card has the named printed keyword.
func (c *CardDef) HasKeyword(name string) bool {
	for _, k := range c.Keywords {
		if strings.EqualFold(k.Name, name) {
			return true
		}
	}
	return false
}

// KeywordValue returns the printed value of a numeric keyword (Retaliate 2,
// Hinder 3, ...); 0 when the keyword is absent or carries no number.
func (c *CardDef) KeywordValue(name string) int {
	for _, k := range c.Keywords {
		if strings.EqualFold(k.Name, name) {
			return k.Value
		}
	}
	return 0
}

func (c *CardDef) String() string {
	if c.Subname != "" {
		return fmt.Sprintf("%s (%s) [%s]", c.Name, c.Subname, c.Code)
	}
	return fmt.Sprintf("%s [%s]", c.Name, c.Code)
}
