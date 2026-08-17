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
