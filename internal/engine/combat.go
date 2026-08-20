package engine

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// ---------------------------------------------------------------- deck utils

func (g *Game) shuffle(l *CardList) {
	cards := *l
	for i := len(cards) - 1; i > 0; i-- {
		j := g.Random(i + 1)
		cards[i], cards[j] = cards[j], cards[i]
	}
	*l = cards
}

func (g *Game) assignCardIDs(l CardList, owner PlayerID) CardList {
	out := make(CardList, len(l))
	for i, c := range l {
		c.ID = g.nextCardID()
		c.Owner = owner
		out[i] = c
	}
	return out
}

// drawEncounter pops the top card of the encounter deck, reshuffling the
// discard pile when the deck runs out.
func (g *Game) drawEncounter() (Card, bool) {
	if len(g.EncounterDeck) == 0 {
		if len(g.EncounterDiscard) == 0 {
			return Card{}, false
		}
		g.EncounterDeck = g.EncounterDiscard
		g.EncounterDiscard = nil
		g.shuffle(&g.EncounterDeck)
		g.logf("Encounter deck reshuffled")
	}
	card := g.EncounterDeck[0]
	g.EncounterDeck = g.EncounterDeck[1:]
	return card, true
}

// PeekEncounterTop returns the top card of the encounter deck without
// drawing it. The encounter discard is reshuffled in first if the deck
// is empty. Used by Loki's self-resolve interrupt.
func (g *Game) PeekEncounterTop() (Card, bool) {
	if len(g.EncounterDeck) == 0 {
		if len(g.EncounterDiscard) == 0 {
			return Card{}, false
		}
		// Reshuffle into a scratch list so the peek doesn't permanently
		// reorder the discard pile.
		scratch := append(CardList(nil), g.EncounterDiscard...)
		g.shuffle(&scratch)
		if len(scratch) == 0 {
			return Card{}, false
		}
		return scratch[0], true
	}
	return g.EncounterDeck[0], true
}

// ---------------------------------------------------------------- threat

func (g *Game) addThreat(schemeID EntityID, n int, source EntityID) {
	if n <= 0 {
		return
	}
	if s := g.SideSchemes[schemeID]; s != nil {
		s.Threat += n
		g.logf("%s gains %d threat (now %d)", s.EDef().Name, n, s.Threat)
		return
	}
	if g.MainScheme != nil && schemeID == g.MainScheme.ID {
		s := g.MainScheme
		// Jennifer Walters' "I Object!" auto-prevents 1 threat per round
		// while in alter-ego form (approximation: auto-use).
		const objectKey = "01019-object"
		if n > 0 && !g.UsedThisRound[objectKey] {
			for _, p := range g.Players {
				if p.HeroCode == "01019a" && !p.IsHero() {
					n--
					g.UsedThisRound[objectKey] = true
					g.logf("Jennifer Walters prevents 1 threat (I Object!)")
					break
				}
			}
		}
		if n <= 0 {
			return
		}
		s.Threat += n
		g.logf("%s gains %d threat (now %d/%d)", s.EDef().Name, n, s.Threat, s.MaxThreat)
		if s.Threat >= s.MaxThreat {
			g.Push(MainSchemeMaxed{Scheme: s.ID})
		}
	}
}

func (g *Game) removeThreat(schemeID EntityID, n int, source EntityID) {
	if n <= 0 {
		return
	}
	if adj, ok := g.eventBonusFor(n, source, "threat"); ok {
		n = adj
	}
	if s := g.SideSchemes[schemeID]; s != nil {
		s.Threat -= n
		if s.Threat < 0 {
			s.Threat = 0
		}
		g.logf("%s loses %d threat (now %d)", s.EDef().Name, n, s.Threat)
		if s.Threat == 0 {
			g.Push(SchemeDefeated{Scheme: s.ID})
		}
		return
	}
	if g.MainScheme != nil && schemeID == g.MainScheme.ID {
		s := g.MainScheme
		before := s.Threat
		s.Threat -= n
		if s.Threat < 0 {
			s.Threat = 0
		}
		g.logf("%s loses %d threat (now %d/%d)", s.EDef().Name, before-s.Threat, s.Threat, s.MaxThreat)
		if before > 0 && s.Threat == 0 {
			g.Push(SchemeDefeated{Scheme: s.ID})
		}
	}
}

// ---------------------------------------------------------------- damage

// destroyAlly removes a defeated ally from play and discards its card.
func (g *Game) destroyAlly(id EntityID) {
	a := g.Allies[id]
	if a == nil {
		return
	}
	owner := g.Player(a.Owner)
	code := a.Code
	g.Delete(id)
	if owner != nil {
		owner.Discard = append(owner.Discard, Card{ID: g.nextCardID(), Code: code, Owner: owner.ID})
	}
	g.logf("%s is destroyed", a.EDef().Name)
}

// eventBonusFor applies a pending Embiggen!/Shrink bonus to a value
// originating from source (a player). ok=false when no bonus applies.
func (g *Game) eventBonusFor(n int, source EntityID, kind string) (int, bool) {
	if source == "" || !source.Is(KindPlayer) {
		return n, false
	}
	var bonus int
	switch kind {
	case "damage":
		bonus = g.EventDamageBonus[source]
	case "threat":
		bonus = g.EventThreatBonus[source]
	default:
		return n, false
	}
	if bonus == 0 {
		return n, false
	}
	switch kind {
	case "damage":
		delete(g.EventDamageBonus, source)
	case "threat":
		delete(g.EventThreatBonus, source)
	}
	g.logf("event bonus +%d %s", bonus, kind)
	return n + bonus, true
}

func (g *Game) damage(id EntityID, n int, source EntityID) {
	if n <= 0 {
		return
	}
	switch e := g.Entity(id).(type) {
	case *Villain:
		scen := g.Scenario()
		if scen.VillainUndamageable[e.Stage] {
			g.logf("%s cannot be damaged", e.EDef().Name)
			return
		}
		if b := behavior(e.Code); b.VillainDamageable != nil && !b.VillainDamageable(g, e, n) {
			return
		}
		if e.Tough {
			e.Tough = false
			g.logf("%s's tough status card prevents the damage", e.EDef().Name)
			return
		}
		if adj, ok := g.eventBonusFor(n, source, "damage"); ok {
			n = adj
		}
		e.Damage += n
		g.logf("%s takes %d damage (%d/%d)", e.EDef().Name, n, e.Damage, e.MaxHP)
		if e.HP() <= 0 {
			g.Push(VillainDefeated{VillainID: e.ID})
		}
	case *Minion:
		if b := behavior(e.Code); b.MinionDamageable != nil && !b.MinionDamageable(g, e, n) {
			return
		}
		if e.Tough {
			e.Tough = false
			g.logf("%s's tough status card prevents the damage", e.EDef().Name)
			return
		}
		if adj, ok := g.eventBonusFor(n, source, "damage"); ok {
			n = adj
		}
		e.Damage += n
		g.logf("%s takes %d damage (%d/%d)", e.EDef().Name, n, e.Damage, e.MaxHP)
		if e.HP() <= 0 {
			g.Push(MinionDefeated{MinionID: e.ID})
		}
	case *Ally:
		if e.Tough {
			e.Tough = false
			return
		}
		if src, ok := g.eventBonusFor(n, source, "damage"); ok {
			n = src
		}
		e.Damage += n
		g.logf("%s takes %d damage (%d/%d)", e.EDef().Name, n, e.Damage, e.MaxHP)
		if e.HP() <= 0 {
			g.Push(AllyDefeated{AllyID: e.ID})
		}
	case *Player:
		// Identity-level prevention (Groot's growth counters).
		if hook := behavior(e.HeroCode).IdentityDamagePrevention; hook != nil {
			if pv := hook(g, e, n); pv > 0 {
				n -= pv
				if n <= 0 {
					return
				}
			}
		}
		// Automatic damage prevention from upgrades (Energy Barrier;
		// approximation: auto-used, reflection hits the first enemy).
		prevented := 0
		for _, id := range e.Upgrades {
			u := g.Upgrades[id]
			if u == nil {
				continue
			}
			hook := behavior(u.Code).DamagePrevention
			if hook == nil {
				continue
			}
			pv, refl := hook(g, u, e, n-prevented)
			prevented += pv
			if refl > 0 {
				// Prefer reflecting at the damage source, else the
				// first enemy.
				target := source
				if !(source.Is(KindVillain) || source.Is(KindMinion)) {
					if enemies := sortedIDs(g.Minions); len(enemies) > 0 {
						target = enemies[0]
					} else if vids := sortedIDs(g.Villains); len(vids) > 0 {
						target = vids[0]
					}
				}
				if target != "" {
					g.Push(DamageEntity{Target: target, Damage: refl, Source: e.ID})
				}
			}
			if prevented >= n {
				break
			}
		}
		if prevented > 0 {
			n -= prevented
			g.logf("%s prevents %d damage", e.Name, prevented)
			if n <= 0 {
				return
			}
		}
		if e.Tough {
			e.Tough = false
			g.logf("%s's tough status card prevents the damage", e.Name)
			return
		}
		e.Damage += n
		g.logf("%s takes %d damage (HP %d/%d)", e.Name, n, e.HP(), e.MaxHP)
		if e.HP() <= 0 && !e.KOed {
			if g.applyDefeatSave(e) {
				return
			}
			e.KOed = true
			g.logf("%s is KO'd!", e.Name)
			g.Push(GameOver{Won: false, Reason: fmt.Sprintf("%s was defeated", e.Name)})
		}
	}
}

// applyDefeatSave lets an upgrade save the identity from defeat (Captain
// America's Helmet); reports whether the defeat was prevented.
func (g *Game) applyDefeatSave(p *Player) bool {
	for _, id := range p.Upgrades {
		u := g.Upgrades[id]
		if u == nil {
			continue
		}
		if hook := behavior(u.Code).DefeatSave; hook != nil {
			if hook(g, p, u) {
				return true
			}
		}
	}
	return false
}

func (g *Game) heal(id EntityID, n int) {
	if n <= 0 {
		return
	}
	switch e := g.Entity(id).(type) {
	case *Player:
		before := e.Damage
		e.Damage -= n
		if e.Damage < 0 {
			e.Damage = 0
		}
		g.logf("%s heals %d damage", e.Name, before-e.Damage)
	case *Ally:
		before := e.Damage
		e.Damage -= n
		if e.Damage < 0 {
			e.Damage = 0
		}
		g.logf("%s heals %d damage", e.EDef().Name, before-e.Damage)
	case *Villain:
		// villains do not heal via generic HealEntity
	}
}

// ---------------------------------------------------------------- status

func (g *Game) setStatus(id EntityID, status string, on bool) {
	switch e := g.Entity(id).(type) {
	case *Player:
		switch status {
		case "stunned":
			e.Stunned = on
		case "confused":
			e.Confused = on
		case "tough":
			e.Tough = on
		}
	case *Villain:
		switch status {
		case "stunned":
			e.Stunned = on
		case "confused":
			e.Confused = on
		case "tough":
			e.Tough = on
		}
	case *Minion:
		switch status {
		case "stunned":
			e.Stunned = on
		case "confused":
			e.Confused = on
		case "tough":
			e.Tough = on
		}
	case *Ally:
		switch status {
		case "stunned":
			e.Stunned = on
		case "confused":
			e.Confused = on
		case "tough":
			e.Tough = on
		}
	}
	if on {
		g.logf("%s gains %s status", id, status)
	}
}

func (g *Game) setExhausted(id EntityID, exhausted bool) {
	switch e := g.Entity(id).(type) {
	case *Player:
		e.Exhausted = exhausted
	case *Ally:
		e.Exhausted = exhausted
	case *Support:
		e.Exhausted = exhausted
	case *Upgrade:
		e.Exhausted = exhausted
	case *Environment:
		e.Exhausted = exhausted
	}
}

// retaliateOf returns the Retaliate value of an enemy or defending
// identity (including upgrade bonuses such as Captain America's Shield).
func retaliateOf(g *Game, e Entity) int {
	if p, ok := e.(*Player); ok {
		return printedRetaliate(p.EDef()) + p.upgradeStats(g).Retaliate
	}
	return printedRetaliate(e.EDef())
}

// printedRetaliate reads the printed Retaliate keyword value.
func printedRetaliate(def *data.CardDef) int {
	for _, k := range def.Keywords {
		if k.Name == "Retaliate" {
			return k.Value
		}
	}
	return 0
}

// Defense returns the ally's printed defense (allies defend with DEF).
func (a *Ally) Defense() int {
	def := a.EDef()
	return deref(def.Defense, 0)
}

var _ = data.BaseCode // keep import during engine bring-up
