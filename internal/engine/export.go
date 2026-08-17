package engine

import "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"

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

// NextEntityID allocates a fresh entity id (exported for setup hooks).
func (g *Game) NextEntityID(kind string) EntityID { return g.nextEntityID(kind) }

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
