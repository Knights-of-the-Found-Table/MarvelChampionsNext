package engine

import (
	"fmt"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// Card is a physical card instance inside a game. It is uniquely identified
// per game and references its CardDef by code.
type Card struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	// Owner is the player id for player cards, "" for encounter cards.
	Owner PlayerID `json:"owner,omitempty"`
	FaceDown bool  `json:"faceDown,omitempty"`
}

func (c Card) Def() *data.CardDef { return DB.MustLookup(c.Code) }

func (c Card) String() string { return fmt.Sprintf("%s(%s)", c.Code, c.ID) }

// CardList is a ordered zone of cards (deck, hand, discard...).
type CardList []Card

func (l CardList) Codes() []string {
	out := make([]string, len(l))
	for i, c := range l {
		out[i] = c.Code
	}
	return out
}

func (l *CardList) Remove(id string) (Card, bool) {
	for i, c := range *l {
		if c.ID == id {
			*l = append((*l)[:i], (*l)[i+1:]...)
			return c, true
		}
	}
	return Card{}, false
}

func (l *CardList) Find(id string) (Card, bool) {
	for _, c := range *l {
		if c.ID == id {
			return c, true
		}
	}
	return Card{}, false
}

// CostPaid is a recorded resource payment for a played card.
type CostPaid struct {
	CardIDs []string `json:"cardIds"` // cards discarded from hand for resources
	// Icons lists the resource icon types contributed by the paid cards
	// (with multiplicity), e.g. ["energy", "physical"].
	Icons []string `json:"icons,omitempty"`
}

// PaidIcon reports whether the payment included the given icon type.
func (c CostPaid) PaidIcon(icon string) bool {
	for _, i := range c.Icons {
		if i == icon {
			return true
		}
	}
	return false
}
