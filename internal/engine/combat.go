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
		e.Damage += n
		g.logf("%s takes %d damage (%d/%d)", e.EDef().Name, n, e.Damage, e.MaxHP)
		if e.HP() <= 0 {
			g.Push(VillainDefeated{VillainID: e.ID})
		}
	case *Minion:
		if e.Tough {
			e.Tough = false
			g.logf("%s's tough status card prevents the damage", e.EDef().Name)
			return
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
		e.Damage += n
		g.logf("%s takes %d damage (%d/%d)", e.EDef().Name, n, e.Damage, e.MaxHP)
		if e.HP() <= 0 {
			owner := g.Player(e.Owner)
			g.Delete(e.ID)
			if owner != nil {
				owner.Discard = append(owner.Discard, Card{ID: g.nextCardID(), Code: e.Code, Owner: owner.ID})
			}
			g.logf("%s is destroyed", e.EDef().Name)
		}
	case *Player:
		if e.Tough {
			e.Tough = false
			g.logf("%s's tough status card prevents the damage", e.Name)
			return
		}
		e.Damage += n
		g.logf("%s takes %d damage (HP %d/%d)", e.Name, n, e.HP(), e.MaxHP)
		if e.HP() <= 0 && !e.KOed {
			e.KOed = true
			g.logf("%s is KO'd!", e.Name)
			g.Push(GameOver{Won: false, Reason: fmt.Sprintf("%s was defeated", e.Name)})
		}
	}
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

// retaliateOf returns the printed Retaliate value of an enemy.
func retaliateOf(e Entity) int {
	def := e.EDef()
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
