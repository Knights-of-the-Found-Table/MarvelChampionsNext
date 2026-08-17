package engine

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// TurnMenu builds the main action menu for a player's turn.
func (g *Game) TurnMenu(p *Player) *Question {
	var choices []Choice

	// Change form (once per turn, not while exhausted).
	if !p.FormChanged && !p.Exhausted {
		var target string
		if p.IsHero() {
			target = p.AlterEgoDef().Name
		} else {
			target = p.HeroDef().Name
		}
		choices = append(choices, Choice{
			ID: "form", Label: "Change to " + target, Kind: ChoiceForm,
			CardCode: otherSideCode(p),
		}.Msgs(ChangeForm{Player: p.ID}))
	}

	// Playable hand cards.
	for _, c := range p.Hand {
		def := c.Def()
		if def.Category != data.CategoryPlayer {
			continue
		}
		switch def.Type {
		case "ally", "support", "upgrade", "event":
		default:
			continue
		}
		cost := deref(def.Cost, 0)
		choice := Choice{
			Label:    fmt.Sprintf("%s (cost %d)", def.Name, cost),
			Kind:     ChoicePlay,
			CardCode: def.Code,
		}
		if cost > 0 {
			choice = choice.WithThen(g.paymentQuestion(p, c, cost))
		} else {
			choice = choice.Msgs(PlayCard{Player: p.ID, Card: c})
		}
		choices = append(choices, choice)
	}

	// Abilities from controlled entities and the identity itself.
	choices = append(choices, g.abilityChoices(p)...)

	// Basic powers.
	if p.IsHero() && !p.Exhausted && !p.Stunned {
		if len(g.Enemies()) > 0 {
			targets := Ask("Choose an enemy", g.enemyChoices(p.AttackStat())...)
			choices = append(choices, Choice{
				ID: "basic-attack", Label: fmt.Sprintf("Attack (%d)", p.AttackStat()),
				Kind: ChoiceBasicPower,
			}.WithThen(targets))
		}
	}
	if p.IsHero() && !p.Exhausted && !p.Confused {
		if len(g.thwartableSchemes()) > 0 {
			targets := Ask("Choose a scheme", g.schemeChoices(p.ThwartStat())...)
			choices = append(choices, Choice{
				ID: "basic-thwart", Label: fmt.Sprintf("Thwart (%d)", p.ThwartStat()),
				Kind: ChoiceBasicPower,
			}.WithThen(targets))
		}
	}
	if !p.IsHero() && !p.Exhausted {
		choices = append(choices, Choice{
			ID: "basic-recover", Label: fmt.Sprintf("Recover (%d)", p.RecoverStat()),
			Kind: ChoiceBasicPower,
		}.Msgs(BasicRecover{Player: p.ID}))
	}

	// Ally basic actions.
	for _, id := range p.Allies {
		a := g.Allies[id]
		if a == nil || a.Exhausted {
			continue
		}
		if len(g.Enemies()) > 0 && !a.Stunned {
			choices = append(choices, Choice{
				ID:    "ally-atk-" + a.ID.String(),
				Label: fmt.Sprintf("%s attacks (%d)", a.EDef().Name, a.AttackVal),
				Kind:  ChoiceBasicPower, SourceID: a.ID,
			}.WithThen(Ask("Choose an enemy", g.enemyChoicesForAlly(a)...)))
		}
		if len(g.thwartableSchemes()) > 0 && !a.Confused {
			choices = append(choices, Choice{
				ID:    "ally-thw-" + a.ID.String(),
				Label: fmt.Sprintf("%s thwarts (%d)", a.EDef().Name, a.ThwartVal),
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

// paymentQuestion builds the resource-payment tree for playing a card.
func (g *Game) paymentQuestion(p *Player, card Card, cost int) *Question {
	q := &Question{
		Type:   "choose_n",
		Prompt: fmt.Sprintf("Pay %d resources for %s (select cards)", cost, card.Def().Name),
	}
	for _, c := range p.Hand {
		if c.ID == card.ID {
			continue
		}
		def := c.Def()
		label := def.Name
		if len(def.Resources) > 0 {
			label += " [" + resourceLabels(def) + "]"
		}
		q.Choices = append(q.Choices, Choice{
			Label: label, Kind: ChoiceResource, CardCode: def.Code,
		}.Msgs(ResourcePayStub{Card: c}))
	}
	q.Validate = fmt.Sprintf("payment:%d", cost)
	q.Context = map[string]any{"cardId": card.ID, "player": p.ID.String()}
	q.assignIDs("")
	return q
}

// ResourcePayStub marks a choice as part of a payment selection; the engine
// collects these and emits a single PlayCard with the payment attached.
type ResourcePayStub struct {
	Card Card
}

func (ResourcePayStub) msg() {}

// abilityPaymentQuestion is the payment flow for costed abilities.
func (g *Game) abilityPaymentQuestion(p *Player, src Entity, idx int, ab Ability) *Question {
	q := &Question{
		Type:   "choose_n",
		Prompt: fmt.Sprintf("Pay %d resources for %s", ab.Cost, ab.Label),
	}
	for _, c := range p.Hand {
		def := c.Def()
		q.Choices = append(q.Choices, Choice{
			Label: def.Name, Kind: ChoiceResource, CardCode: def.Code,
		}.Msgs(ResourcePayStub{Card: c}))
	}
	q.Validate = fmt.Sprintf("payment:%d", ab.Cost)
	q.Context = map[string]any{"abilitySource": src.EID().String(), "abilityIndex": idx, "player": p.ID.String()}
	q.assignIDs("")
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

// thwartableSchemes lists schemes that can currently be thwarted.
func (g *Game) thwartableSchemes() []EntityID {
	var out []EntityID
	if g.MainScheme != nil && !g.MainScheme.Crisis {
		out = append(out, g.MainScheme.ID)
	}
	for id := range g.SideSchemes {
		if !g.SideSchemes[id].Crisis {
			out = append(out, id)
		}
	}
	return out
}

func (g *Game) enemyChoices(dmg int) []Choice {
	var out []Choice
	for _, id := range sortedIDs(g.Villains) {
		v := g.Villains[id]
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — %d/%d HP", v.EDef().Name, v.HP(), v.MaxHP),
			Kind:  ChoiceTarget, SourceID: v.ID, CardCode: v.Code,
		}.Msgs(BasicAttack{Player: g.currentPlayerID(), N: dmg, Target: v.ID}))
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
	consequential := func(target EntityID) []Message {
		return []Message{
			ExhaustEntity{ID: a.ID},
			DamageEntity{Target: target, Damage: a.AttackVal, Source: a.Owner},
			DamageEntity{Target: a.ID, Damage: 1, Source: a.ID},
		}
	}
	for _, id := range sortedIDs(g.Villains) {
		v := g.Villains[id]
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — %d/%d HP", v.EDef().Name, v.HP(), v.MaxHP),
			Kind:  ChoiceTarget, SourceID: v.ID, CardCode: v.Code,
		}.Msgs(consequential(v.ID)...))
	}
	for _, id := range sortedIDs(g.Minions) {
		mn := g.Minions[id]
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — %d/%d HP", mn.EDef().Name, mn.HP(), mn.MaxHP),
			Kind:  ChoiceTarget, SourceID: mn.ID, CardCode: mn.Code,
		}.Msgs(consequential(mn.ID)...))
	}
	return out
}

func (g *Game) schemeChoices(n int) []Choice {
	var out []Choice
	pid := g.currentPlayerID()
	if g.MainScheme != nil && !g.MainScheme.Crisis {
		s := g.MainScheme
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — %d/%d threat", s.EDef().Name, s.Threat, s.MaxThreat),
			Kind:  ChoiceTarget, SourceID: s.ID, CardCode: s.Code,
		}.Msgs(BasicThwart{Player: pid, N: n, Target: s.ID}))
	}
	for _, id := range sortedIDs(g.SideSchemes) {
		s := g.SideSchemes[id]
		if s.Crisis {
			continue
		}
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — %d threat", s.EDef().Name, s.Threat),
			Kind:  ChoiceTarget, SourceID: s.ID, CardCode: s.Code,
		}.Msgs(BasicThwart{Player: pid, N: n, Target: s.ID}))
	}
	return out
}

func (g *Game) schemeChoicesForAlly(a *Ally) []Choice {
	var out []Choice
	consequential := func(target EntityID) []Message {
		return []Message{
			ExhaustEntity{ID: a.ID},
			ThwartScheme{Scheme: target, N: a.ThwartVal, Source: a.Owner},
			DamageEntity{Target: a.ID, Damage: 1, Source: a.ID},
		}
	}
	if g.MainScheme != nil && !g.MainScheme.Crisis {
		s := g.MainScheme
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — %d/%d threat", s.EDef().Name, s.Threat, s.MaxThreat),
			Kind:  ChoiceTarget, SourceID: s.ID, CardCode: s.Code,
		}.Msgs(consequential(s.ID)...))
	}
	for _, id := range sortedIDs(g.SideSchemes) {
		s := g.SideSchemes[id]
		if s.Crisis {
			continue
		}
		out = append(out, Choice{
			Label: fmt.Sprintf("%s — %d threat", s.EDef().Name, s.Threat),
			Kind:  ChoiceTarget, SourceID: s.ID, CardCode: s.Code,
		}.Msgs(consequential(s.ID)...))
	}
	return out
}

// defenderQuestion builds the defense prompt for a villain attack.
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
			ID: "hero-defend",
			Label: fmt.Sprintf("Exhaust %s to defend (+%d DEF)", p.HeroDef().Name, p.DefenseStat()),
			Kind: ChoiceBasicPower,
		}.Msgs(Defends{Defender: p.ID, Against: attackerID}))
	}
	for _, id := range p.Allies {
		a := g.Allies[id]
		if a == nil || a.Exhausted {
			continue
		}
		choices = append(choices, Choice{
			ID:       "ally-defend-" + a.ID.String(),
			Label:    fmt.Sprintf("%s defends (+%d DEF)", a.EDef().Name, a.Defense()),
			Kind:     ChoiceBasicPower, SourceID: a.ID, CardCode: a.Code,
		}.Msgs(Defends{Defender: a.ID, Against: attackerID}))
	}
	return Ask(prompt, choices...)
}

// attackQuestion combines optional hero interrupts with the defense prompt:
// each interrupt choice chains into the defender question.
func (g *Game) attackQuestion(attackerID EntityID, atk int, p *Player, trigger string) *Question {
	defend := g.defenderQuestion(attackerID, atk, p)
	b := behavior(p.HeroCode)
	if b.HeroAbilities == nil {
		return defend
	}
	var interrupts []Choice
	for i, ab := range b.HeroAbilities(g, p) {
		if ab.Trigger != trigger || !ab.usable(g, p.ID, i, p) {
			continue
		}
		var msgs []Message
		if ab.Execute != nil {
			msgs = append(msgs, ab.Execute(g, p.ID)...)
		}
		msgs = append(msgs, RunAbility{Player: p.ID, Source: p.ID, Index: i})
		interrupts = append(interrupts, Choice{
			ID:       fmt.Sprintf("interrupt-%d", i),
			Label:    ab.Label + " (interrupt)",
			Kind:     ChoiceAbility,
			SourceID: p.ID,
			CardCode: p.HeroCode,
		}.Msgs(msgs...).WithThen(defend))
	}
	if len(interrupts) == 0 {
		return defend
	}
	interrupts = append(interrupts, Choice{
		ID: "pass-interrupt", Label: "Continue to defense", Kind: ChoicePass,
	}.WithThen(defend))
	return Ask("Interrupts", interrupts...)
}
