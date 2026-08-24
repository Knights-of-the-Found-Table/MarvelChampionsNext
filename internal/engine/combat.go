package engine

import (
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
// discard pile when the deck runs out. Reshuffling places an acceleration
// token next to the main scheme (official Encounter Deck rule).
func (g *Game) drawEncounter() (Card, bool) {
	if len(g.EncounterDeck) == 0 {
		if len(g.EncounterDiscard) == 0 {
			return Card{}, false
		}
		g.EncounterDeck = g.EncounterDiscard
		g.EncounterDiscard = nil
		g.shuffle(&g.EncounterDeck)
		if g.MainScheme != nil {
			g.MainScheme.AccelerationTokens++
			g.logf("Encounter deck reshuffled: %s gains an acceleration token", g.MainScheme.EDef().Name)
		} else {
			g.logf("Encounter deck reshuffled")
		}
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
		g.emit(Evt{Type: "threat", Src: source, Dst: schemeID, N: n})
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
		g.emit(Evt{Type: "threat", Src: source, Dst: schemeID, N: n})
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
	// Klyntar Frenzy: threat cannot be removed while a Symbiote enemy is
	// in play.
	if s := g.SideSchemes[schemeID]; s != nil && s.Code == "20024" {
		for _, mn := range g.Minions {
			if mn.EDef().HasTrait("symbiote") {
				g.logf("threat cannot be removed from Klyntar Frenzy while a Symbiote is in play")
				return
			}
		}
		for _, v := range g.Villains {
			if v.EDef().HasTrait("symbiote") {
				g.logf("threat cannot be removed from Klyntar Frenzy while a Symbiote is in play")
				return
			}
		}
	}
	// Total Destruction: threat cannot be removed while Abomination is in
	// play.
	if s := g.SideSchemes[schemeID]; s != nil && s.Code == "10027" {
		for _, mn := range g.Minions {
			if mn.Code == "10026" {
				g.logf("threat cannot be removed from Total Destruction while Abomination is in play")
				return
			}
		}
	}
	// Spiral: while she is in play, threat cannot be removed from the
	// main scheme.
	if g.MainScheme != nil && schemeID == g.MainScheme.ID {
		for _, v := range g.Villains {
			if v != nil && data.BaseCode(v.Code) == "39012" {
				g.logf("threat cannot be removed while Spiral is in play")
				return
			}
		}
	}
	// Grand Larceny: threat cannot be removed while a Criminal minion is
	// in play.
	if s := g.SideSchemes[schemeID]; s != nil && s.Code == "31030" {
		for _, mn := range g.Minions {
			if mn != nil && mn.EDef().HasTrait("criminal") {
				g.logf("threat cannot be removed from Grand Larceny while a Criminal is in play")
				return
			}
		}
	}
	// Back to the Future (40033): only the Cable player can remove threat
	// from it, and only from it (approximation: keyed on the Cable hero's
	// presence; the damage locks are not modeled).
	if s := g.SideSchemes[schemeID]; s != nil && s.Code == "40033" {
		if !source.Is(KindPlayer) || !g.heroIs(source, "40001") {
			g.logf("only the Cable player can remove threat from Back to the Future")
			return
		}
	} else if source.Is(KindPlayer) && g.heroIs(source, "40001") && g.sideSchemeInPlay("40033") {
		g.logf("Cable can only remove threat from Back to the Future while it is in play")
		return
	}
	if adj, ok := g.eventBonusFor(n, source, "threat"); ok {
		n = adj
	}
	if s := g.SideSchemes[schemeID]; s != nil {
		before := s.Threat
		s.Threat -= n
		if s.Threat < 0 {
			s.Threat = 0
		}
		if d := before - s.Threat; d > 0 {
			g.emit(Evt{Type: "thwart", Src: source, Dst: schemeID, N: d})
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
		if d := before - s.Threat; d > 0 {
			g.emit(Evt{Type: "thwart", Src: source, Dst: schemeID, N: d})
		}
		g.logf("%s loses %d threat (now %d/%d)", s.EDef().Name, before-s.Threat, s.Threat, s.MaxThreat)
		if before > 0 && s.Threat == 0 {
			g.Push(SchemeDefeated{Scheme: s.ID})
		}
	}
}

// ---------------------------------------------------------------- damage

// attackValue returns an enemy's current attack total for a defense
// prompt: the villain's modified ATK (boost icons included) or a minion's
// ATK.
func (g *Game) attackValue(id EntityID) int {
	switch e := g.Entity(id).(type) {
	case *Villain:
		n := e.AttackVal + e.BoostCount
		if b := behavior(e.Code); b.EnemyStatBonus != nil {
			if atk, _ := b.EnemyStatBonus(g, e); atk != 0 {
				n += atk
			}
		}
		n += g.attachmentAttackBonus(e.Attachments)
		return n
	case *Minion:
		n := e.AttackVal
		if b := behavior(e.Code); b.EnemyStatBonus != nil {
			if atk, _ := b.EnemyStatBonus(g, e); atk != 0 {
				n += atk
			}
		}
		n += g.attachmentAttackBonus(e.Attachments)
		// Get Nasty (40117): each minion gets +1 ATK.
		if g.sideSchemeInPlay("40117") {
			n++
		}
		return n
	}
	return 0
}

// attachmentDamageMods applies attachment damage modifiers: Titanium
// Exoskeleton caps single-source damage at 2; Impervious reduces each hit
// by 1; Hidden in the Clutter banks the damage on the attachment and
// prevents it (payoff handled by the card's behavior). Returns the damage
// still to apply (0 = fully prevented).
func (g *Game) attachmentDamageMods(list []EntityID, n int, source EntityID) int {
	for _, aid := range list {
		a := g.Attachments[aid]
		if a == nil {
			continue
		}
		switch a.Code {
		case "40091": // Titanium Exoskeleton
			if n > 2 {
				g.logf("Titanium Exoskeleton caps the damage at 2")
				n = 2
			}
		case "40156": // Impervious
			n--
			g.logf("Impervious reduces the damage by 1")
		case "40106": // Hidden in the Clutter
			a.Counters += n
			g.logf("Hidden in the Clutter banks %d damage (%d stored)", n, a.Counters)
			if a.Counters >= 3 {
				g.Delete(aid)
				g.EncounterDiscard = append(g.EncounterDiscard, Card{ID: g.nextCardID(), Code: a.Code})
				g.logf("Hidden in the Clutter bursts — the attached enemy attacks!")
				if source.Is(KindPlayer) && a.Target != "" {
					g.Push(AskAttack{Enemy: a.Target, Player: PlayerID(source)})
				}
			}
			return 0
		}
	}
	return n
}

// sourceIsTiny reports whether the damage source has the Tiny trait
// (Thumbelina's damage reduction).
func (g *Game) sourceIsTiny(src EntityID) bool {
	e := g.Entity(src)
	return e != nil && e.EDef() != nil && e.EDef().HasTrait("Tiny")
}

// attachmentAttackBonus sums attachment-sourced attack bonuses
// (Bolstered by Wrath scaling with villains under Routed, Aerial
// Bombardment, Thrown Object).
func (g *Game) attachmentAttackBonus(list []EntityID) int {
	n := 0
	for _, aid := range list {
		a := g.Attachments[aid]
		if a == nil {
			continue
		}
		switch a.Code {
		case "40082": // Bolstered by Wrath
			if env := routedEnvOf(g); env != nil {
				n += len(env.StoredCards)
			}
		case "40152", "40157": // Aerial Bombardment; Thrown Object stat
			n++
		}
	}
	return n
}

// routedEnvOf finds the Routed environment (Marauders).
func routedEnvOf(g *Game) *Environment {
	for _, env := range g.Environments {
		if env != nil && data.BaseCode(env.Code) == "40081" {
			return env
		}
	}
	return nil
}

// schemeValueOf returns an enemy's scheme value including dynamic bonuses
// (momentum counters, hand-type scaling).
func (g *Game) schemeValueOf(id EntityID) int {
	switch e := g.Entity(id).(type) {
	case *Villain:
		n := e.SchemeVal + e.BoostCount
		if b := behavior(e.Code); b.EnemyStatBonus != nil {
			if _, sch := b.EnemyStatBonus(g, e); sch != 0 {
				n += sch
			}
		}
		return n
	case *Minion:
		n := e.SchemeVal
		if b := behavior(e.Code); b.EnemyStatBonus != nil {
			if _, sch := b.EnemyStatBonus(g, e); sch != 0 {
				n += sch
			}
		}
		return n
	}
	return 0
}

// destroyAlly removes a defeated ally from play and discards its card.
func (g *Game) destroyAlly(id EntityID) {
	a := g.Allies[id]
	if a == nil {
		return
	}
	owner := g.Player(a.Owner)
	code := a.Code
	g.Delete(id)
	g.cardLeavesPlay(owner, code, a.EDef().Name)
	g.logMajorf("%s is destroyed", a.EDef().Name)
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
	g.logMinorf("event bonus +%d %s", bonus, kind)
	return n + bonus, true
}

func (g *Game) damage(id EntityID, n int, source EntityID) {
	if n <= 0 {
		return
	}
	switch e := g.Entity(id).(type) {
	case *Villain:
		if e == nil {
			return // entity left play mid-resolution
		}
		scen := g.Scenario()
		if scen.VillainUndamageable[e.Stage] {
			g.logf("%s cannot be damaged", e.EDef().Name)
			return
		}
		if b := behavior(e.Code); b.VillainDamageable != nil && !b.VillainDamageable(g, e, n) {
			return
		}
		n = g.attachmentDamageMods(e.Attachments, n, source)
		if n <= 0 {
			return
		}
		// Telekinetic Force Field (40034): attached character takes no
		// damage; absorbing 2+ in one hit burns the attachment out.
		for _, aid := range append([]EntityID(nil), e.Attachments...) {
			a := g.Attachments[aid]
			if a == nil || a.Code != "40034" {
				continue
			}
			g.logf("%s takes no damage (Telekinetic Force Field)", e.EDef().Name)
			if n >= 2 {
				g.Delete(aid)
				g.EncounterDiscard = append(g.EncounterDiscard, Card{ID: g.nextCardID(), Code: a.Code})
				g.logf("Telekinetic Force Field is discarded")
			}
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
		g.emit(Evt{Type: "damage", Src: source, Dst: id, N: n})
		g.logf("%s takes %d damage (%d/%d)", e.EDef().Name, n, e.Damage, e.MaxHP)
		if e.HP() <= 0 {
			g.Push(VillainDefeated{VillainID: e.ID})
		}
	case *Minion:
		if e == nil {
			return
		}
		// Cybernetic Enhancements (38035): the attached minion cannot
		// take damage.
		for _, aid := range e.Attachments {
			if a := g.Attachments[aid]; a != nil && a.Code == "38035" {
				g.logf("%s cannot take damage (Cybernetic Enhancements)", e.EDef().Name)
				return
			}
		}
		// Telekinetic Force Field (40034): same absorption as on villains.
		for _, aid := range append([]EntityID(nil), e.Attachments...) {
			a := g.Attachments[aid]
			if a == nil || a.Code != "40034" {
				continue
			}
			g.logf("%s takes no damage (Telekinetic Force Field)", e.EDef().Name)
			if n >= 2 {
				g.Delete(aid)
				g.EncounterDiscard = append(g.EncounterDiscard, Card{ID: g.nextCardID(), Code: a.Code})
				g.logf("Telekinetic Force Field is discarded")
			}
			return
		}
		if b := behavior(e.Code); b.MinionDamageableSrc != nil && !b.MinionDamageableSrc(g, e, n, source) {
			return
		}
		if b := behavior(e.Code); b.MinionDamageable != nil && !b.MinionDamageable(g, e, n) {
			return
		}
		// Thumbelina (40182): damage from each source reduced by 1 unless
		// the attacker has TINY.
		if e.Code == "40182" && !g.sourceIsTiny(source) {
			n--
			if n <= 0 {
				return
			}
		}
		n = g.attachmentDamageMods(e.Attachments, n, source)
		if n <= 0 {
			return
		}
		// Biomechanical Upgrades: when the attached minion would be
		// defeated, heal all damage instead and discard the attachment.
		if e.HP()-n <= 0 {
			for _, aid := range append([]EntityID(nil), e.Attachments...) {
				a := g.Attachments[aid]
				if a == nil || a.Code != "01185" {
					continue
				}
				e.Damage = 0
				g.Delete(aid)
				g.EncounterDiscard = append(g.EncounterDiscard, Card{ID: g.nextCardID(), Code: a.Code})
				g.logMajorf("%s heals all damage instead of being defeated (Biomechanical Upgrades)", e.EDef().Name)
				return
			}
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
		g.emit(Evt{Type: "damage", Src: source, Dst: id, N: n})
		g.logf("%s takes %d damage (%d/%d)", e.EDef().Name, n, e.Damage, e.MaxHP)
		if e.HP() <= 0 {
			g.Push(MinionDefeated{MinionID: e.ID})
		}
	case *Ally:
		if e == nil {
			return
		}
		if e.Tough {
			e.Tough = false
			return
		}
		if src, ok := g.eventBonusFor(n, source, "damage"); ok {
			n = src
		}
		e.Damage += n
		g.emit(Evt{Type: "damage", Src: source, Dst: id, N: n})
		g.logf("%s takes %d damage (%d/%d)", e.EDef().Name, n, e.Damage, e.MaxHP)
		if e.HP() <= 0 {
			g.Push(AllyDefeated{AllyID: e.ID})
		}
	case *Player:
		if e == nil {
			return
		}
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
		g.emit(Evt{Type: "damage", Src: source, Dst: id, N: n})
		g.logf("%s takes %d damage (HP %d/%d)", e.Name, n, e.HP(), e.MaxHP)
		if e.HP() <= 0 && !e.KOed {
			if g.applyDefeatSave(e) {
				return
			}
			g.eliminatePlayer(e)
		}
	}
}

// eliminatePlayer removes a defeated player from the game (official Player
// Elimination): their permanents are discarded, engaged minions re-engage
// the next clockwise player, the first player token passes on if held, and
// the players only lose once everyone is eliminated.
func (g *Game) eliminatePlayer(p *Player) {
	p.KOed = true
	g.logMajorf("%s is eliminated!", p.Name)
	for _, id := range append([]EntityID(nil), p.Allies...) {
		g.discardControlled(p.ID, id)
	}
	for _, id := range append([]EntityID(nil), p.Supports...) {
		g.discardControlled(p.ID, id)
	}
	for _, id := range append([]EntityID(nil), p.Upgrades...) {
		g.discardControlled(p.ID, id)
	}
	p.Hand = nil
	p.EncounterDown = nil
	next := g.nextActivePlayer(p.ID)
	if next != nil {
		for _, mn := range g.Minions {
			if mn.EngagedWith == p.ID {
				mn.EngagedWith = next.ID
			}
		}
		if p.FirstPlayer {
			p.FirstPlayer = false
			next.FirstPlayer = true
		}
	}
	// A pending question addressed to the eliminated player (their turn
	// menu) is dropped so the game can continue.
	if g.pending != nil && g.pending.Player == p.ID {
		g.pending = nil
		p.EndedTurn = true
		if g.ActiveTurn == p.ID {
			g.ActiveTurn = ""
			g.Push(PlayerTurnEnd{Player: p.ID})
		}
	}
	for _, q := range g.Players {
		if !q.KOed {
			return
		}
	}
	g.Push(GameOver{Won: false, Reason: "All players were eliminated"})
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
	// Identity-level defeat saves (Deadpool's Regeneratin' Degenerate)
	// are consulted last, with a nil upgrade.
	if hook := behavior(p.HeroCode).DefeatSave; hook != nil {
		if hook(g, p, nil) {
			return true
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
		if d := before - e.Damage; d > 0 {
			g.emit(Evt{Type: "heal", Dst: id, N: d})
		}
		g.logf("%s heals %d damage", e.Name, before-e.Damage)
	case *Ally:
		before := e.Damage
		e.Damage -= n
		if e.Damage < 0 {
			e.Damage = 0
		}
		if d := before - e.Damage; d > 0 {
			g.emit(Evt{Type: "heal", Dst: id, N: d})
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
		g.emit(Evt{Type: "status", Dst: id, Status: status, On: true})
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
	n := printedRetaliate(e.EDef())
	// Verna (30038) grants every Inheritor minion retaliate 1.
	if mn, ok := e.(*Minion); ok && mn.EDef().HasTrait("inheritor") && g.minionInPlay("30038") {
		n++
	}
	// Attachments granting retaliate: Head of Steam (X = Juggernaut's
	// momentum), Heavy Armament (+2), Telepathy (+1).
	var ids []EntityID
	switch t := e.(type) {
	case *Villain:
		ids = t.Attachments
	case *Minion:
		ids = t.Attachments
	}
	for _, aid := range ids {
		a := g.Attachments[aid]
		if a == nil {
			continue
		}
		switch a.Code {
		case "40123":
			for _, v := range g.Villains {
				if v != nil && data.BaseCode(v.Code) == "40118" {
					n += v.Counters
				}
			}
		case "40090":
			n += 2
		case "40159":
			n++
		}
	}
	return n
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
