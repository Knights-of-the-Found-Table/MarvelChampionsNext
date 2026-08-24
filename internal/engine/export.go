package engine

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// BaseCodeOf strips an a/b/c side suffix ("01001a" -> "01001").
func BaseCodeOf(code string) string { return data.BaseCode(code) }

// LookupScenarioName returns the display name of a registered scenario.
func LookupScenarioName(id string) string {
	if def, ok := LookupScenario(id); ok {
		return def.Name
	}
	return id
}

// Logf appends a human-readable line to the game log (exported for card
// behavior packages).
func (g *Game) Logf(format string, args ...any) { g.logf(format, args...) }

// LogMinorf appends a minor (bookkeeping) line to the game log.
func (g *Game) LogMinorf(format string, args ...any) { g.logMinorf(format, args...) }

// LogMajorf appends a major (pivotal moment) line to the game log.
func (g *Game) LogMajorf(format string, args ...any) { g.logMajorf(format, args...) }

// DrawEncounter pops the top encounter card, reshuffling when empty
// (exported for card behavior packages).
func (g *Game) DrawEncounter() (Card, bool) { return g.drawEncounter() }

// ShuffleEncounterDeck reshuffles the encounter deck (search effects).
func (g *Game) ShuffleEncounterDeck() {
	g.shuffle(&g.EncounterDeck)
	g.Logf("Encounter deck shuffled")
}

// NextEntityID allocates a fresh entity id (exported for setup hooks).
func (g *Game) NextEntityID(kind string) EntityID { return g.nextEntityID(kind) }

// NextCardID allocates a fresh card id (exported for card behaviors).
func (g *Game) NextCardID() string { return g.nextCardID() }

// ShuffleSideDeck shuffles a player's side deck (Invocation setup).
func (g *Game) ShuffleSideDeck(p *Player) { g.shuffle(&p.SenseDeck) }

// AddEnvironment places an environment entity.
func (g *Game) AddEnvironment(env *Environment) { g.Environments[env.ID] = env }

// EnvironmentByCode finds the first environment matching any code.
func (g *Game) EnvironmentByCode(codes ...string) *Environment {
	for _, env := range g.Environments {
		for _, c := range codes {
			if env.Code == c {
				return env
			}
		}
	}
	return nil
}

// AttackQuestion exposes the interrupt + defense prompt builder.
func (g *Game) AttackQuestion(attackerID EntityID, atk int, p *Player, trigger string) *Question {
	return g.attackQuestion(attackerID, atk, p, trigger)
}

// CustomPaymentQuestion builds a validated payment prompt for card-defined
// flows (Make the Call). Context key "makeCallFrom"/"makeCallCard" routes
// the payment to an AllyEntersPlayFree message.
func (g *Game) CustomPaymentQuestion(p *Player, cost int, prompt string, ctx map[string]any) *Question {
	q := &Question{
		Type:   "choose_n",
		Prompt: prompt,
	}
	q.Choices = g.resourcePayChoices(p, nil, nil)
	q.Validate = fmt.Sprintf("payment:%d", cost)
	if ctx == nil {
		ctx = map[string]any{}
	}
	ctx["player"] = p.ID.String()
	q.Context = ctx
	q.assignIDs("")
	return q
}

// EOwnerIfPlayer resolves a side scheme's reveal "owner" to a player:
// the first player by default (reveal-side effects approximate).
func (s *SideScheme) EOwnerIfPlayer() PlayerID { return "" }

// EOwnerIfPlayer on the game resolves the revealer.
func (g *Game) EOwnerIfPlayer() PlayerID {
	for _, p := range g.Players {
		if p.FirstPlayer {
			return p.ID
		}
	}
	if len(g.Players) > 0 {
		return g.Players[0].ID
	}
	return ""
}

// EntityHasTrait reports an entity's traits including dynamically granted
// ones (Honorary Avenger).
func (g *Game) EntityHasTrait(id EntityID, trait string) bool {
	e := g.Entity(id)
	if e == nil {
		return false
	}
	def := e.EDef()
	if def != nil && def.HasTrait(trait) {
		return true
	}
	switch t := e.(type) {
	case *Player:
		for _, x := range t.ExtraTraits {
			if x == trait {
				return true
			}
		}
	case *Ally:
		for _, x := range t.ExtraTraits {
			if x == trait {
				return true
			}
		}
	}
	return false
}

// HandCard finds a card in a player's hand (exported for card behaviors).
func (g *Game) HandCard(pid PlayerID, cardID string) (Card, bool) {
	if p := g.Player(pid); p != nil {
		return p.Hand.Find(cardID)
	}
	return Card{}, false
}

// AddAlly registers an ally under the owner (exported for tests).
func (g *Game) AddAlly(a *Ally, owner PlayerID) {
	g.Allies[a.ID] = a
	if p := g.Player(owner); p != nil {
		p.Allies = append(p.Allies, a.ID)
	}
}

// AddMinion registers a minion (exported for tests).
func (g *Game) AddMinion(m *Minion, engaged PlayerID) {
	g.Minions[m.ID] = m
	m.EngagedWith = engaged
}

// AddSideScheme registers a side scheme (exported for tests).
func (g *Game) AddSideScheme(s *SideScheme) {
	g.SideSchemes[s.ID] = s
}

// QueueLen returns the number of pending messages in the engine queue
// (exported for tests).
func (g *Game) QueueLen() int { return len(g.queue) }

// AttackValueOf returns an enemy's current attack value (including boost
// icons).
func (g *Game) AttackValueOf(id EntityID) int { return g.attackValue(id) }

// SideSchemeInPlay reports whether a side scheme with the exact code is
// in play (exported for card behaviors).
func (g *Game) SideSchemeInPlay(code string) bool { return g.sideSchemeInPlay(code) }

// AttackActivationPending reports whether a villain's attack question is
// queued (boost-reaction windows, exported for card behaviors).
func AttackActivationPending(g *Game, villain EntityID) bool {
	return g.attackActivationPending(villain)
}
