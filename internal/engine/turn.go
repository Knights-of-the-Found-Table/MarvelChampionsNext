package engine

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// TurnMenu builds the main action menu for a player's turn.
func (g *Game) TurnMenu(p *Player) *Question {
	var choices []Choice

	// Change form (once per turn; the character keeps its ready/exhausted
	// state per the official rules).
	if !p.FormChanged {
		var target string
		if p.IsHero() {
			target = p.AlterEgoDef().Name
		} else {
			target = p.HeroDef().Name
		}
		choices = append(choices, Choice{
			ID: "form", Label: "Change to " + target, Kind: ChoiceForm,
			CardCode: otherSideCode(p), SourceID: p.ID,
		}.Msgs(ChangeForm{Player: p.ID}))
	}

	// Playable hand cards.
	for _, c := range p.Hand {
		def := c.Def()
		if def.Category != data.CategoryPlayer {
			continue
		}
		switch def.Type {
		case "ally", "support", "upgrade", "event", "player_side_scheme":
		default:
			continue
		}
		cost := g.costFor(p, def)
		choice := Choice{
			Label:    fmt.Sprintf("%s (cost %d)", def.Name, cost),
			Kind:     ChoicePlay,
			CardCode: def.Code,
			SourceID: EntityID(c.ID),
		}
		if cost > 0 {
			choice = choice.WithThen(g.paymentQuestion(p, c, cost))
		} else {
			choice = choice.Msgs(PlayCard{Player: p.ID, Card: c})
		}
		choices = append(choices, choice)
	}

	// Allies playable from the discard pile (Lockjaw).
	for _, c := range p.Discard {
		def := c.Def()
		if def.Type != "ally" || !behavior(def.Code).PlayableFromDiscard {
			continue
		}
		cost := g.costFor(p, def)
		choice := Choice{
			Label:    fmt.Sprintf("%s from discard (cost %d)", def.Name, cost),
			Kind:     ChoicePlay,
			CardCode: def.Code,
		}
		if cost > 0 {
			q := &Question{
				Type:   "choose_n",
				Prompt: fmt.Sprintf("Pay %d resources for %s (select cards)", cost, def.Name),
			}
			q.Choices = g.resourcePayChoices(p, &c, def)
			q.Validate = fmt.Sprintf("payment:%d", cost)
			q.Context = map[string]any{"player": p.ID.String(), "playDiscard": c.ID}
			choice = choice.WithThen(q)
		} else {
			choice = choice.Msgs(PlayDiscardAlly{Player: p.ID, Card: c})
		}
		choices = append(choices, choice)
	}

	// Abilities from controlled entities and the identity itself.
	choices = append(choices, g.abilityChoices(p)...)

	// Basic powers.
	if p.IsHero() && !p.Exhausted && !p.Stunned {
		if len(g.Enemies()) > 0 {
			targets := Ask("Choose an enemy", g.enemyChoices(p.AttackStat(g))...)
			choices = append(choices, Choice{
				ID: "basic-attack", Label: fmt.Sprintf("Attack (%d)", p.AttackStat(g)),
				Kind: ChoiceBasicPower, SourceID: p.ID,
			}.WithThen(targets))
		}
	}
	if p.IsHero() && !p.Exhausted && !p.Confused {
		if len(g.thwartableSchemes()) > 0 {
			targets := Ask("Choose a scheme", g.schemeChoices(p.ThwartStat(g))...)
			choices = append(choices, Choice{
				ID: "basic-thwart", Label: fmt.Sprintf("Thwart (%d)", p.ThwartStat(g)),
				Kind: ChoiceBasicPower, SourceID: p.ID,
			}.WithThen(targets))
		}
	}
	if !p.IsHero() && !p.Exhausted {
		choices = append(choices, Choice{
			ID: "basic-recover", Label: fmt.Sprintf("Recover (%d)", p.RecoverStat(g)),
			Kind: ChoiceBasicPower, SourceID: p.ID,
		}.Msgs(BasicRecover{Player: p.ID}))
	}

	// Ally basic actions.
	for _, id := range p.Allies {
		a := g.Allies[id]
		if a == nil || a.Exhausted {
			continue
		}
		// Allies with an additional discard cost cannot attack with an
		// empty hand (Wonder Man).
		attackOK := len(g.Enemies()) > 0 && !a.Stunned
		if attackOK && behavior(a.Code).AllyAttackDiscardCost && len(p.Hand) == 0 {
			attackOK = false
		}
		if attackOK {
			choices = append(choices, Choice{
				ID:    "ally-atk-" + a.ID.String(),
				Label: fmt.Sprintf("%s attacks (%d)", a.EDef().Name, a.AttackVal+a.BonusATK+a.PermATK),
				Kind:  ChoiceBasicPower, SourceID: a.ID,
			}.WithThen(Ask("Choose an enemy", g.enemyChoicesForAlly(a)...)))
		}
		if len(g.thwartableSchemes()) > 0 && !a.Confused {
			choices = append(choices, Choice{
				ID:    "ally-thw-" + a.ID.String(),
				Label: fmt.Sprintf("%s thwarts (%d)", a.EDef().Name, a.ThwartVal+a.BonusTHW),
				Kind:  ChoiceBasicPower, SourceID: a.ID,
			}.WithThen(Ask("Choose a scheme", g.schemeChoicesForAlly(a)...)))
		}
	}

	choices = append(choices, Choice{
		ID: "end-turn", Label: "End turn", Kind: ChoiceEndTurn,
	}.Msgs(PlayerTurnEnd{Player: p.ID}))

	return Ask("Your turn", choices...)
}

func otherSideCode(p *Player) string {
	if p.IsHero() {
		return p.AlterEgoCode
	}
	return p.HeroCode
}

// abilityChoices gathers activated abilities from the identity and
// controlled entities.
func (g *Game) abilityChoices(p *Player) []Choice {
	var out []Choice

	appendAbilities := func(src Entity, abilities []Ability) {
		for i, ab := range abilities {
			if ab.Trigger != "" || !ab.usable(g, src.EID(), i, p) {
				continue // triggered abilities are offered contextually
			}
			choice := Choice{
				ID:       "ability-" + src.EID().String() + "-" + fmt.Sprint(i),
				Label:    ab.Label,
				Kind:     ChoiceAbility,
				SourceID: src.EID(),
				CardCode: src.ECode(),
			}
			if ab.Cost > 0 {
				choice = choice.WithThen(g.abilityPaymentQuestion(p, src, i, ab))
			} else {
				var msgs []Message
				if ab.Exhaust {
					msgs = append(msgs, ExhaustEntity{ID: src.EID()})
				}
				if ab.Execute != nil {
					msgs = append(msgs, ab.Execute(g, src.EID())...)
				}
				msgs = append(msgs, RunAbility{Player: p.ID, Source: src.EID(), Index: i})
				choice = choice.Msgs(msgs...)
			}
			out = append(out, choice)
		}
	}

	// Identity abilities.
	if b := behavior(p.HeroCode); b.HeroAbilities != nil {
		appendAbilities(p, b.HeroAbilities(g, p))
	}

	for _, id := range p.Allies {
		if a := g.Allies[id]; a != nil {
			if hb := behavior(a.Code); hb.Abilities != nil {
				appendAbilities(a, hb.Abilities(g, a))
			}
		}
	}
	for _, id := range p.Supports {
		if s := g.Supports[id]; s != nil {
			if hb := behavior(s.Code); hb.Abilities != nil {
				appendAbilities(s, hb.Abilities(g, s))
			}
		}
	}
	for _, id := range p.Upgrades {
		if u := g.Upgrades[id]; u != nil {
			if hb := behavior(u.Code); hb.Abilities != nil {
				appendAbilities(u, hb.Abilities(g, u))
			}
		}
	}
	return out
}

// ResourcePayStub marks a choice as part of a payment selection; the engine
// collects these and emits a single PlayCard with the payment attached.
type ResourcePayStub struct {
	Card Card
}

func (ResourcePayStub) msg() {}

// AbilityPayStub marks a choice that pays by activating a resource ability
// of an in-play support/upgrade.
type AbilityPayStub struct {
	Source EntityID
	Icon   string
}

func (AbilityPayStub) msg() {}

// resourceProducers lists the player's supports/upgrades whose resource
// ability can currently contribute to paying for targetDef (nil = ability
// payment, no card).
func (g *Game) resourceProducers(p *Player, targetDef *data.CardDef) []Entity {
	var out []Entity
	add := func(id EntityID) {
		e := g.Entity(id)
		if e == nil || e.EExhausted() {
			return
		}
		b := behavior(e.ECode())
		if b.Resource == nil {
			return
		}
		if b.Resource.HeroOnly && !p.IsHero() {
			return
		}
		if b.Resource.EventOnly && (targetDef == nil || targetDef.Type != "event") {
			return
		}
		if b.Resource.UsesCounters {
			switch t := e.(type) {
			case *Support:
				if t.Counters <= 0 {
					return
				}
			case *Upgrade:
				if t.Counters <= 0 {
					return
				}
			case *Ally:
				if t.Counters <= 0 {
					return
				}
			}
		}
		out = append(out, e)
	}
	for _, id := range p.Supports {
		add(id)
	}
	for _, id := range p.Upgrades {
		add(id)
	}
	return out
}

// resourcePayChoices builds payment choices for a player: hand cards plus
// usable resource abilities.
func (g *Game) resourcePayChoices(p *Player, self *Card, targetDef *data.CardDef) []Choice {
	var out []Choice
	for _, c := range p.Hand {
		if self != nil && c.ID == self.ID {
			continue
		}
		def := c.Def()
		label := def.Name
		if len(def.Resources) > 0 {
			label += " [" + resourceLabels(def) + "]"
		}
		out = append(out, Choice{
			Label: label, Kind: ChoiceResource, CardCode: def.Code, SourceID: EntityID(c.ID),
		}.Msgs(ResourcePayStub{Card: c}))
	}
	for _, src := range g.resourceProducers(p, targetDef) {
		ra := behavior(src.ECode()).Resource
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — generate [%s]", src.EDef().Name, ra.Icon),
			Kind:  ChoiceAbility, SourceID: src.EID(), CardCode: src.ECode(),
		}.Msgs(AbilityPayStub{Source: src.EID(), Icon: ra.Icon}))
	}
	return out
}

// paymentQuestion builds the resource-payment tree for playing a card.
func (g *Game) paymentQuestion(p *Player, card Card, cost int) *Question {
	q := &Question{
		Type:   "choose_n",
		Prompt: fmt.Sprintf("Pay %d resources for %s (select cards)", cost, card.Def().Name),
	}
	q.Choices = g.resourcePayChoices(p, &card, card.Def())
	q.Validate = fmt.Sprintf("payment:%d", cost)
	q.Context = map[string]any{"cardId": card.ID, "player": p.ID.String()}
	// ids 由挂载该子树的根问题统一分配（带 "N." 前缀），避免与根层冲突。
	return q
}

// defensePaymentQuestion is the payment flow for a defense event; on
// completion it emits PlayDefenseEvent instead of PlayCard.
func (g *Game) defensePaymentQuestion(p *Player, card Card, cost int, against EntityID) *Question {
	q := g.paymentQuestion(p, card, cost)
	q.Prompt = fmt.Sprintf("Pay %d resources for %s (defense)", cost, card.Def().Name)
	q.Context["defenseAgainst"] = against.String()
	return q
}

// abilityPaymentQuestion is the payment flow for costed abilities.
func (g *Game) abilityPaymentQuestion(p *Player, src Entity, idx int, ab Ability) *Question {
	q := &Question{
		Type:   "choose_n",
		Prompt: fmt.Sprintf("Pay %d resources for %s", ab.Cost, ab.Label),
	}
	q.Choices = g.resourcePayChoices(p, nil, nil)
	q.Validate = fmt.Sprintf("payment:%d", ab.Cost)
	q.Context = map[string]any{"abilitySource": src.EID().String(), "abilityIndex": idx, "player": p.ID.String()}
	return q
}

func resourceLabels(def *data.CardDef) string {
	out := ""
	for _, r := range def.Resources {
		if out != "" {
			out += " "
		}
		out += r
	}
	return out
}

// thwartableSchemes lists schemes that can currently be thwarted. The crisis
// icon on a side scheme blocks threat removal from the main scheme while it
// is in play; crisis side schemes themselves remain thwartable.
func (g *Game) thwartableSchemes() []EntityID {
	var out []EntityID
	if g.MainScheme != nil && !g.crisisInPlay() {
		out = append(out, g.MainScheme.ID)
	}
	for id := range g.SideSchemes {
		out = append(out, id)
	}
	return out
}

// crisisInPlay reports whether any encounter side scheme with the crisis
// icon is in play.
func (g *Game) crisisInPlay() bool {
	for _, s := range g.SideSchemes {
		if s.Crisis && !s.PlayerSide {
			return true
		}
	}
	return false
}

// guardBlocksVillain reports whether pid cannot attack the given villain:
// while a guard minion is engaged with them, they cannot attack villains
// without the guard keyword (official Guard keyword).
func (g *Game) guardBlocksVillain(pid PlayerID, v *Villain) bool {
	if v.EDef().HasKeyword("Guard") {
		return false
	}
	for _, mn := range g.Minions {
		if mn.EngagedWith == pid && mn.Guard {
			return true
		}
	}
	return false
}

func (g *Game) enemyChoices(dmg int) []Choice {
	var out []Choice
	pid := g.currentPlayerID()
	for _, id := range sortedIDs(g.Villains) {
		v := g.Villains[id]
		if g.guardBlocksVillain(pid, v) {
			continue // guard minion engaged: villain cannot be attacked
		}
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — %d/%d HP", v.EDef().Name, v.HP(), v.MaxHP),
			Kind:  ChoiceTarget, SourceID: v.ID, CardCode: v.Code,
		}.Msgs(BasicAttack{Player: pid, N: dmg, Target: v.ID}))
	}
	for _, id := range sortedIDs(g.Minions) {
		mn := g.Minions[id]
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — %d/%d HP", mn.EDef().Name, mn.HP(), mn.MaxHP),
			Kind:  ChoiceTarget, SourceID: mn.ID, CardCode: mn.Code,
		}.Msgs(BasicAttack{Player: g.currentPlayerID(), N: dmg, Target: mn.ID}))
	}
	return out
}

// currentPlayerID is the player whose turn menu built these choices.
func (g *Game) currentPlayerID() PlayerID { return g.ActiveTurn }

func (g *Game) enemyChoicesForAlly(a *Ally) []Choice {
	var out []Choice
	p := g.Player(a.Owner)
	atk := a.AttackVal + a.BonusATK + a.PermATK
	consq := 1 + g.attachedConsequential(a)
	consequential := func(target EntityID) []Message {
		// Elektra-style allies redirect consequential damage to the
		// owner.
		self := a.ID
		if behavior(a.Code).ConsequentialToOwner {
			self = a.Owner
		}
		return []Message{
			ExhaustEntity{ID: a.ID},
			AllyAttackWindow{Ally: a.ID, Target: target},
			DamageEntity{Target: target, Damage: atk, Source: a.Owner},
			DamageEntity{Target: self, Damage: consq, Source: a.ID},
		}
	}
	// Allies whose attack has an additional discard cost (Wonder Man)
	// chain into a card-discard question whose leaves carry the attack.
	needsDiscard := behavior(a.Code).AllyAttackDiscardCost && p != nil && len(p.Hand) > 0
	for _, id := range sortedIDs(g.Villains) {
		v := g.Villains[id]
		if g.guardBlocksVillain(a.Owner, v) {
			continue // guard minion engaged: villain cannot be attacked
		}
		out = append(out, allyAttackChoice(
			fmt.Sprintf("%s — %d/%d HP", v.EDef().Name, v.HP(), v.MaxHP), v.ID, v.Code,
			consequential(v.ID), needsDiscard, a, p))
	}
	for _, id := range sortedIDs(g.Minions) {
		mn := g.Minions[id]
		out = append(out, allyAttackChoice(
			fmt.Sprintf("%s — %d/%d HP", mn.EDef().Name, mn.HP(), mn.MaxHP), mn.ID, mn.Code,
			consequential(mn.ID), needsDiscard, a, p))
	}
	return out
}

// allyAttackChoice builds one enemy target choice for an ally attack,
// routing through the discard-cost question when required.
func allyAttackChoice(label string, target EntityID, code string, attackMsgs []Message, needsDiscard bool, a *Ally, p *Player) Choice {
	c := Choice{
		Label: label, Kind: ChoiceTarget, SourceID: target, CardCode: code,
	}
	if !needsDiscard {
		return c.Msgs(attackMsgs...)
	}
	var picks []Choice
	for _, hc := range p.Hand {
		picks = append(picks, Choice{
			Label: "Discard " + hc.Def().Name, Kind: ChoiceCard, CardCode: hc.Code,
		}.Msgs(append([]Message{DiscardCards{Player: p.ID, Cards: CardList{hc}}}, attackMsgs...)...))
	}
	return c.WithThen(Ask(fmt.Sprintf("Discard a card for %s to attack", a.EDef().Name), picks...))
}

// attachedConsequential sums ConsequentialBonus of upgrades attached to an
// ally (Enraged).
func (g *Game) attachedConsequential(a *Ally) int {
	n := 0
	owner := g.Player(a.Owner)
	if owner == nil {
		return 0
	}
	for _, id := range owner.Upgrades {
		if u := g.Upgrades[id]; u != nil && u.AttachTo == a.ID {
			n += behavior(u.Code).ConsequentialBonus
		}
	}
	return n
}

func (g *Game) schemeChoices(n int) []Choice {
	var out []Choice
	pid := g.currentPlayerID()
	if g.MainScheme != nil && !g.crisisInPlay() {
		s := g.MainScheme
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — %d/%d threat", s.EDef().Name, s.Threat, s.MaxThreat),
			Kind:  ChoiceTarget, SourceID: s.ID, CardCode: s.Code,
		}.Msgs(BasicThwart{Player: pid, N: n, Target: s.ID}))
	}
	for _, id := range sortedIDs(g.SideSchemes) {
		s := g.SideSchemes[id]
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — %d threat", s.EDef().Name, s.Threat),
			Kind:  ChoiceTarget, SourceID: s.ID, CardCode: s.Code,
		}.Msgs(BasicThwart{Player: pid, N: n, Target: s.ID}))
	}
	return out
}

func (g *Game) schemeChoicesForAlly(a *Ally) []Choice {
	var out []Choice
	thw := a.ThwartVal + a.BonusTHW
	consequential := func(target EntityID) []Message {
		self := a.ID
		if behavior(a.Code).ConsequentialToOwner {
			self = a.Owner
		}
		return []Message{
			ExhaustEntity{ID: a.ID},
			ThwartScheme{Scheme: target, N: thw, Source: a.Owner},
			DamageEntity{Target: self, Damage: 1, Source: a.ID},
		}
	}
	if g.MainScheme != nil && !g.crisisInPlay() {
		s := g.MainScheme
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — %d/%d threat", s.EDef().Name, s.Threat, s.MaxThreat),
			Kind:  ChoiceTarget, SourceID: s.ID, CardCode: s.Code,
		}.Msgs(consequential(s.ID)...))
	}
	for _, id := range sortedIDs(g.SideSchemes) {
		s := g.SideSchemes[id]
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — %d threat", s.EDef().Name, s.Threat),
			Kind:  ChoiceTarget, SourceID: s.ID, CardCode: s.Code,
		}.Msgs(consequential(s.ID)...))
	}
	return out
}

// defenderQuestion builds the defense prompt for an enemy attack.
func (g *Game) defenderQuestion(attackerID EntityID, atk int, p *Player) *Question {
	attacker := g.Entity(attackerID)
	name := "enemy"
	if attacker != nil {
		name = attacker.EDef().Name
	}
	prompt := fmt.Sprintf("%s attacks for %d: defend?", name, atk)
	var choices []Choice
	choices = append(choices, Choice{
		ID: "take", Label: "Take the attack", Kind: ChoiceLabel,
	}.Msgs(Defends{Defender: p.ID, Against: attackerID, Undefended: true}))
	if p.IsHero() && !p.Exhausted {
		choices = append(choices, Choice{
			ID:    "hero-defend",
			Label: fmt.Sprintf("Exhaust %s to defend (+%d DEF)", p.HeroDef().Name, p.DefenseStat(g)),
			Kind:  ChoiceBasicPower,
		}.Msgs(Defends{Defender: p.ID, Against: attackerID}))
	}
	for _, id := range p.Allies {
		a := g.Allies[id]
		if a == nil || a.Exhausted {
			continue
		}
		choices = append(choices, Choice{
			ID:    "ally-defend-" + a.ID.String(),
			Label: fmt.Sprintf("%s defends (+%d DEF)", a.EDef().Name, a.Defense()),
			Kind:  ChoiceBasicPower, SourceID: a.ID, CardCode: a.Code,
		}.Msgs(Defends{Defender: a.ID, Against: attackerID}))
	}
	// Defense events playable from hand (Shield Block, Wiggle Room...).
	for _, c := range p.Hand {
		def := c.Def()
		if def.Type != "event" {
			continue
		}
		b := behavior(def.Code)
		if b.DefenseEvent == nil {
			continue
		}
		ec := &EventCard{Code: def.Code, Owner: p.ID}
		if _, _, ok := b.DefenseEvent(g, p, ec, attackerID); !ok {
			continue
		}
		choice := Choice{
			ID: "defense-event-" + c.ID, Label: "Play " + def.Name,
			Kind: ChoicePlay, CardCode: def.Code,
		}
		if cost := deref(def.Cost, 0); cost > 0 {
			choice = choice.WithThen(g.defensePaymentQuestion(p, c, cost, attackerID))
		} else {
			choice = choice.Msgs(PlayDefenseEvent{Player: p.ID, Card: c, Against: attackerID})
		}
		choices = append(choices, choice)
	}
	// In-play upgrade substitute defenses (Bamf!).
	for _, id := range p.Upgrades {
		u := g.Upgrades[id]
		if u == nil {
			continue
		}
		hook := behavior(u.Code).DefenseSubstitute
		if hook == nil {
			continue
		}
		d, extra, ok := hook(g, p, u, attackerID)
		if !ok {
			continue
		}
		d.Via = u.Code
		choice := Choice{
			ID: "defense-sub-" + u.ID.String(), Label: u.EDef().Name + " — defend without exhausting",
			Kind: ChoiceAbility, SourceID: u.ID, CardCode: u.Code,
		}.Msgs(append([]Message{d}, extra...)...)
		choices = append(choices, choice)
	}
	return Ask(prompt, choices...)
}

// attackQuestion combines optional hero/ally interrupts with the defense
// prompt: each interrupt choice chains into the defender question.
func (g *Game) attackQuestion(attackerID EntityID, atk int, p *Player, trigger string) *Question {
	defend := g.defenderQuestion(attackerID, atk, p)
	var interrupts []Choice
	addInterrupt := func(src Entity, i int, ab Ability) {
		var msgs []Message
		if ab.Execute != nil {
			msgs = append(msgs, ab.Execute(g, src.EID())...)
		}
		msgs = append(msgs, RunAbility{Player: p.ID, Source: src.EID(), Index: i})
		interrupts = append(interrupts, Choice{
			ID:       fmt.Sprintf("interrupt-%s-%d", src.EID().Kind(), i),
			Label:    ab.Label + " (interrupt)",
			Kind:     ChoiceAbility,
			SourceID: src.EID(),
			CardCode: src.ECode(),
		}.Msgs(msgs...).WithThen(defend))
	}
	// Identity abilities.
	if b := behavior(p.HeroCode); b.HeroAbilities != nil {
		for i, ab := range b.HeroAbilities(g, p) {
			if ab.Trigger != trigger || !ab.usable(g, p.ID, i, p) {
				continue
			}
			addInterrupt(p, i, ab)
		}
	}
	// Ally triggered abilities (Nova).
	for _, id := range p.Allies {
		a := g.Allies[id]
		if a == nil {
			continue
		}
		if hb := behavior(a.Code); hb.Abilities != nil {
			for i, ab := range hb.Abilities(g, a) {
				if ab.Trigger != trigger || !ab.usable(g, a.ID, i, p) {
					continue
				}
				addInterrupt(a, i, ab)
			}
		}
	}
	// Support triggered abilities (Stand Alone).
	for _, id := range p.Supports {
		s := g.Supports[id]
		if s == nil {
			continue
		}
		if hb := behavior(s.Code); hb.Abilities != nil {
			for i, ab := range hb.Abilities(g, s) {
				if ab.Trigger != trigger || !ab.usable(g, s.ID, i, p) {
					continue
				}
				addInterrupt(s, i, ab)
			}
		}
	}
	if len(interrupts) == 0 {
		return defend
	}
	interrupts = append(interrupts, Choice{
		ID: "pass-interrupt", Label: "Continue to defense", Kind: ChoicePass,
	}.WithThen(defend))
	return Ask("Interrupts", interrupts...)
}
