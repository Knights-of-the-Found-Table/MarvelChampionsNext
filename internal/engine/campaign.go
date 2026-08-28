package engine

import (
	"sort"
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// CampaignSetup carries the meta-campaign modifications applied to a single
// scenario game (The Rise of Red Skull / Galaxy's Most Wanted campaign
// mode). The campaign layer fills it from the campaign log before calling
// NewGame; the engine applies it during game start. Keeping the plumbing
// here (instead of importing a campaign package) preserves the engine's
// save/resume/replay round-trip: the struct serializes with the game.
type CampaignSetup struct {
	// ExtraSets lists encounter set codes appended to the encounter deck
	// beyond the scenario's own (e.g. the Badoon Headhunter modular).
	ExtraSets []string
	// PreShuffle lists specific card codes shuffled into the encounter
	// deck (experimental weapons recorded in the campaign log, On the
	// Hunt, Galactic Artifacts side schemes, Fugitive Recovery…).
	PreShuffle []string
	// StartSideScheme is a campaign side scheme revealed during setup
	// (Badoon Blitz 16178a and friends).
	StartSideScheme string
	// SideSchemeThreat places extra threat on a revealed campaign side
	// scheme (Ronan: Pincer Maneuver scales with evasion counters).
	SideSchemeThreat map[string]int
	// MainSchemeThreat places threat on the main scheme during setup
	// (Red Skull: delay counters; Ronan expert: +1).
	MainSchemeThreat int
	// ShipEvasion places evasion counters on Nebula's Ship (16093).
	ShipEvasion int
	// VillainTough gives the villain a tough status card (Crystal Ball).
	VillainTough bool
	// VillainBoostFacedown gives the villain a facedown boost card
	// (Philosopher's Stone).
	VillainBoostFacedown bool
	// FirstPlayerEncounterFacedown deals the first player a facedown
	// encounter card (Magical Teapot).
	FirstPlayerEncounterFacedown bool
	// CollectionHandCard moves one card of each player's choice from
	// their hand into The Collection (Infiltrate the Museum expert).
	CollectionHandCard bool
	// RemoveFromGame removes these card codes from every player's deck,
	// hand and discard during setup, then (with DrawUpAfterRemove) tops
	// each player up to their hand size (Escape the Museum).
	RemoveFromGame    []string
	DrawUpAfterRemove bool
	// DealEncounter deals a facedown encounter card to the listed player
	// indexes (Red Skull expert: players recorded as engaged with an
	// enemy deal themselves an encounter card).
	DealEncounter []int
	// DealCard deals the given card code facedown to the listed player
	// index (Ronan: "You Stand Accused!" for Power Stone controllers).
	// The card is pulled from the encounter deck or discard pile.
	DealCard map[int]string
	// MillMinionEngage mills the encounter deck until a minion appears
	// and puts it into play engaged with the first player (Brotherhood of
	// Badoon expert).
	MillMinionEngage bool
	// MillRevealAttachment mills the encounter deck until an attachment
	// appears and reveals it (Escape the Museum / Nebula expert
	// techniques).
	MillRevealAttachment bool
}

// SpawnUpgrade puts an upgrade into play under owner without a payment
// window (setup-keyword cards, campaign obligations drawn from a deck).
func (g *Game) SpawnUpgrade(code string, owner PlayerID) *Upgrade {
	def, ok := DB.Lookup(code)
	if !ok {
		return nil
	}
	u := &Upgrade{ID: g.nextEntityID(KindUpgrade), Code: def.Code, Owner: owner}
	g.Upgrades[u.ID] = u
	if p := g.Player(owner); p != nil {
		p.Upgrades = append(p.Upgrades, u.ID)
	}
	g.tlogf("log.putsIntoPlay", g.Player(owner).Name, def)
	if b := behavior(def.Code); b != nil && b.OnPlay != nil {
		g.Push(b.OnPlay(g, u)...)
	}
	return u
}

// setupKeywordCards finds player cards in the deck whose printed rules
// begin with the setup keyword ("A card with the setup keyword begins the
// game in play").
func setupKeywordCards(p *Player) []Card {
	var out []Card
	kept := make(CardList, 0, len(p.Deck))
	for _, c := range p.Deck {
		def, ok := DB.Lookup(c.Code)
		if ok && def.Category == data.CategoryPlayer && hasKeyword(def, "setup") {
			out = append(out, c)
			continue
		}
		kept = append(kept, c)
	}
	p.Deck = kept
	return out
}

func hasKeyword(def *data.CardDef, name string) bool {
	for _, k := range def.Keywords {
		if strings.EqualFold(strings.TrimSpace(k.Name), name) {
			return true
		}
	}
	return false
}

// enterSetupCards resolves the setup keyword: campaign upgrades (Basic
// Condition / Tech upgrades) join their identity before the first round.
func (g *Game) enterSetupCards() {
	for _, p := range g.Players {
		for _, c := range setupKeywordCards(p) {
			switch def := DB.MustLookup(c.Code); def.Type {
			case "upgrade":
				g.SpawnUpgrade(c.Code, p.ID)
			case "support":
				g.SpawnSupport(c.Code, p.ID)
			case "ally":
				g.Push(AllyEntersPlayFree{Player: p.ID, Card: c, FromOwner: p.ID})
			default:
				// Only in-play types are legal setup cards; anything else
				// returns to the deck.
				p.Deck = append(p.Deck, c)
			}
		}
	}
}

// applyCampaignStart runs the campaign setup instructions after the normal
// scenario setup (hands dealt, mulligans taken, hero setup done). Direct
// mutations happen before the queue drains; reveals ride messages so the
// standard reveal pipeline (enter play, surge-free) applies.
func (g *Game) applyCampaignStart(c *CampaignSetup) []Message {
	if c == nil {
		return nil
	}
	var out []Message
	// Threat on the main scheme (Red Skull: delay counters).
	if c.MainSchemeThreat > 0 && g.MainScheme != nil {
		g.MainScheme.Threat += c.MainSchemeThreat
		g.TLogf("c.campaignPlacesThreatOnTheMainScheme", c.MainSchemeThreat)
	}
	// Evasion counters on Nebula's Ship (Monarch Egg / Guerrilla Tactics
	// feed the same environment).
	if c.ShipEvasion > 0 {
		if ship := g.EnvironmentByCode("16093"); ship != nil {
			ship.Counters += c.ShipEvasion
			g.TLogf("c.nebulaSShipGains2EvasionCountersTotal", ship.Counters)
		}
	}
	if c.VillainTough {
		for id := range g.Villains {
			out = append(out, ToughEntity{Target: id})
			break
		}
	}
	if c.VillainBoostFacedown {
		if c2, ok := g.drawEncounter(); ok {
			for id := range g.Villains {
				if v := g.Villains[id]; v != nil {
					v.BoostCards = append(v.BoostCards, c2)
					g.TLogf("log.dealsFacedown", v)
				}
				break
			}
		}
	}
	// Facedown encounter cards (Magical Teapot, Red Skull expert).
	deal := func(i int) {
		if i < 0 || i >= len(g.Players) {
			return
		}
		p := g.Players[i]
		if c2, ok := g.drawEncounter(); ok {
			p.EncounterDown = append(p.EncounterDown, c2)
			g.TLogf("log.dealsFacedown", p.Name)
		}
	}
	for _, i := range c.DealEncounter {
		deal(i)
	}
	if c.DealCard != nil {
		idxs := make([]int, 0, len(c.DealCard))
		for i := range c.DealCard {
			idxs = append(idxs, i)
		}
		sort.Ints(idxs)
		for _, i := range idxs {
			code := c.DealCard[i]
			if i < 0 || i >= len(g.Players) {
				continue
			}
			for _, zone := range []*CardList{&g.EncounterDeck, &g.EncounterDiscard} {
				found := false
				for j, card := range *zone {
					if card.Code == code || data.BaseCode(card.Code) == code {
						c2 := card
						*zone = append((*zone)[:j:j], (*zone)[j+1:]...)
						g.Players[i].EncounterDown = append(g.Players[i].EncounterDown, c2)
						g.TLogf("log.dealsFacedown", g.Players[i].Name)
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
	}
	if c.FirstPlayerEncounterFacedown {
		for i, p := range g.Players {
			if p.FirstPlayer {
				deal(i)
				break
			}
		}
	}
	// Mill effects.
	millUntil := func(match func(Card) bool) (Card, bool) {
		for len(g.EncounterDeck) > 0 {
			card := g.EncounterDeck[0]
			g.EncounterDeck = g.EncounterDeck[1:]
			if match(card) {
				return card, true
			}
			g.EncounterDiscard = append(g.EncounterDiscard, card)
		}
		return Card{}, false
	}
	if c.MillMinionEngage {
		if card, ok := millUntil(func(c Card) bool { return c.Def().Type == "minion" }); ok {
			out = append(out, RevealEncounterCard{Player: firstPlayerID(g), Card: card})
		}
	}
	if c.MillRevealAttachment {
		if card, ok := millUntil(func(c Card) bool { return c.Def().Type == "attachment" }); ok {
			out = append(out, RevealEncounterCard{Player: firstPlayerID(g), Card: card})
		}
	}
	// Remove recorded cards from the game (Escape the Museum), then top
	// the players back up.
	if len(c.RemoveFromGame) > 0 {
		remove := map[string]bool{}
		for _, code := range c.RemoveFromGame {
			remove[data.BaseCode(code)] = true
		}
		for _, p := range g.Players {
			kept := make(CardList, 0, len(p.Deck))
			for _, card := range p.Deck {
				if remove[data.BaseCode(card.Code)] {
					continue
				}
				kept = append(kept, card)
			}
			p.Deck = kept
			kept = make(CardList, 0, len(p.Hand))
			for _, card := range p.Hand {
				if remove[data.BaseCode(card.Code)] {
					continue
				}
				kept = append(kept, card)
			}
			p.Hand = kept
			kept = make(CardList, 0, len(p.Discard))
			for _, card := range p.Discard {
				if remove[data.BaseCode(card.Code)] {
					continue
				}
				kept = append(kept, card)
			}
			p.Discard = kept
			if c.DrawUpAfterRemove {
				if n := p.HandSize(g) - len(p.Hand); n > 0 {
					g.Push(DrawCards{Player: p.ID, N: n})
				}
			}
		}
	}
	// One card from each hand into The Collection (Infiltrate the Museum
	// expert). The engine's Collection intercept normally asks via the
	// leave-play path; setup runs outside a turn, so pick the leftmost
	// card deterministically and log it.
	if c.CollectionHandCard {
		for _, p := range g.Players {
			if len(p.Hand) == 0 {
				continue
			}
			card := p.Hand[0]
			p.Hand = p.Hand[1:]
			card.ID = g.nextCardID()
			g.Collection = append(g.Collection, card)
			g.Logf("%s is placed into The Collection", card.Def().Name)
		}
	}
	var out2 []Message
	if c.StartSideScheme != "" {
		if def, ok := DB.Lookup(c.StartSideScheme); ok {
			code := data.BaseCode(def.Code)
			out2 = append(out2, RevealEncounterCard{Player: firstPlayerID(g), Card: Card{ID: g.nextCardID(), Code: def.Code}})
			if n := c.SideSchemeThreat[code]; n > 0 {
				out2 = append(out2, CampaignSideThreat{Code: code, N: n})
			}
		}
	}
	return append(out, out2...)
}

// applyCampaignDeck assembles encounter-deck additions before the shuffle:
// extra sets and specific pre-shuffled cards.
func (g *Game) applyCampaignDeck(c *CampaignSetup) {
	if c == nil {
		return
	}
	for _, set := range c.ExtraSets {
		g.EncounterDeck = append(g.EncounterDeck, EncounterSetCards(set)...)
	}
	for _, code := range c.PreShuffle {
		g.EncounterDeck = append(g.EncounterDeck, Card{Code: code})
	}
}

// firstPlayerID returns the first player's id (fallback: the first seat).
func firstPlayerID(g *Game) PlayerID {
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
