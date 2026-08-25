package nextevolution

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/cardutil"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// victorySideSchemes counts side schemes currently in the victory display
// (Cable's scaling rider).
func victorySideSchemes(g *engine.Game) int {
	n := 0
	for _, c := range g.VictoryDisplay {
		if c.Def().Type == "side_scheme" || c.Def().Type == "player_side_scheme" {
			n++
		}
	}
	return n
}

// deckTopIcons peeks at the top card of the player's deck and returns its
// printed resource icons plus the card itself (Domino's luck riders; the
// actual discard is emitted separately via engine.MillPlayerDeck).
func deckTopIcons(p *engine.Player) (engine.Card, []string, bool) {
	if len(p.Deck) == 0 {
		return engine.Card{}, nil, false
	}
	c := p.Deck[0]
	return c, c.Def().Resources, true
}

// iconCountOf counts a card's printed resource icons.
func iconCountOf(c engine.Card) int { return len(c.Def().Resources) }

// mostCommonHandType counts the cards of the most common card type in the
// player's hand (Stryfe's scaling).
func mostCommonHandType(p *engine.Player) int {
	if p == nil {
		return 0
	}
	counts := map[string]int{}
	for _, c := range p.Hand {
		counts[c.Def().Type]++
	}
	best := 0
	for _, n := range counts {
		if n > best {
			best = n
		}
	}
	return best
}

// handIcons counts resource icons of a given type in the player's hand
// (wild counts for every type).
func handIcons(p *engine.Player, typ string) int {
	n := 0
	for _, c := range p.Hand {
		for _, r := range c.Def().Resources {
			if r == typ || r == "wild" {
				n++
			}
		}
	}
	return n
}

// firstCardOf scans a player's deck and discard pile for a card with the
// given base code and returns it with its location.
func firstCardOf(p *engine.Player, base string) (engine.Card, string, bool) {
	for _, c := range p.Deck {
		if data.BaseCode(c.Code) == base {
			return c, "deck", true
		}
	}
	for _, c := range p.Discard {
		if data.BaseCode(c.Code) == base {
			return c, "discard", true
		}
	}
	return engine.Card{}, "", false
}

// firstCardWhere returns the first card in deck+discard satisfying pred.
func firstCardWhere(p *engine.Player, pred func(*data.CardDef) bool) (engine.Card, string, bool) {
	for _, c := range p.Deck {
		if pred(c.Def()) {
			return c, "deck", true
		}
	}
	for _, c := range p.Discard {
		if pred(c.Def()) {
			return c, "discard", true
		}
	}
	return engine.Card{}, "", false
}

// takeFromZone removes a card from the named zone.
func takeFromZone(p *engine.Player, c engine.Card, zone string) bool {
	switch zone {
	case "deck":
		_, ok := p.Deck.Remove(c.ID)
		return ok
	case "discard":
		_, ok := p.Discard.Remove(c.ID)
		return ok
	}
	return false
}

// routedEnv returns the Routed environment (morlock siege).
func routedEnv(g *engine.Game) *engine.Environment {
	for _, e := range g.Environments {
		if e != nil && engine.BaseCodeOf(e.Code) == "40081" {
			return e
		}
	}
	return nil
}

// villainsUnderRouted counts villains banked under Routed.
func villainsUnderRouted(g *engine.Game) int {
	env := routedEnv(g)
	if env == nil {
		return 0
	}
	return len(env.StoredCards)
}

// marauderChoice builds the shared Marauder "when this attacks you, choose"
// reaction: take the penalty or the enemy gets +2 ATK for this attack.
// Applies to both the villain (40070-40076) and minion (40094-40100)
// printings since they share base codes.
func marauderChoice(g *engine.Game, e engine.Entity, m engine.AskAttack, penaltyLabel string, penalty []engine.Message, atk int) []engine.Message {
	if m.Enemy != e.EID() {
		return nil
	}
	p := g.Player(m.Player)
	if p == nil {
		return nil
	}
	return []engine.Message{engine.AskQuestion{
		Player: p.ID,
		Question: engine.Ask(engine.Tf("c.attacksChoose", e, p.Name),
			engine.Choice{ID: "penalty", Label: engine.S(penaltyLabel), Kind: engine.ChoiceLabel}.Msgs(penalty...),
			engine.Choice{ID: "boost", Label: engine.Tf("c.getsAtkForThisAttack", e, atk), Kind: engine.ChoiceLabel}.
				Msgs(engine.BoostActivation{Enemy: e.EID(), N: atk}),
		)},
	}
}

// revealEncounterFrom pops the top card of the encounter deck.
func revealEncounterFrom(g *engine.Game) (engine.Card, bool) {
	if len(g.EncounterDeck) == 0 {
		g.EncounterDeck = append(g.EncounterDeck, g.EncounterDiscard...)
		g.EncounterDiscard = nil
		g.TLogf("log.encounterReshuffled")
	}
	if len(g.EncounterDeck) == 0 {
		return engine.Card{}, false
	}
	c := g.EncounterDeck[0]
	g.EncounterDeck = g.EncounterDeck[1:]
	return c, true
}

// boostEnemy returns the currently-activating enemy (single-villain
// approximation: the first villain).
func boostEnemy(g *engine.Game) engine.EntityID {
	for id := range g.Villains {
		return id
	}
	return ""
}

// stunBoost confuses/stuns the boost reveal target (first player
// approximation — boost hooks lack activation context).
func stunBoost(g *engine.Game, card engine.Card) []engine.Message {
	if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
		return []engine.Message{engine.StunEntity{Target: p.ID}}
	}
	return nil
}

func confuseBoost(g *engine.Game, card engine.Card) []engine.Message {
	if p := g.Player(cardutil.FirstPlayerID(g)); p != nil {
		return []engine.Message{engine.ConfuseEntity{Target: p.ID}}
	}
	return nil
}

// spawnSideSchemeCard brings a side scheme card into play as an entity with
// starting threat (victory display returns, set-aside Morlock setups...).
func spawnSideSchemeCard(g *engine.Game, code string, threat int) *engine.SideScheme {
	def, ok := engine.DB.Lookup(code)
	if !ok {
		return nil
	}
	maxT := derefOr(def.BaseThreat, 6) + 2*(len(g.Players)-1)
	if threat > maxT {
		maxT = threat
	}
	s := &engine.SideScheme{
		ID:        g.NextEntityID(engine.KindSideScheme),
		Code:      code,
		Threat:    threat,
		MaxThreat: maxT,
	}
	g.SideSchemes[s.ID] = s
	g.TLogf("c.entersPlayThreat", def, threat)
	return s
}

func derefOr(p *int, fallback int) int {
	if p == nil {
		return fallback
	}
	return *p
}
