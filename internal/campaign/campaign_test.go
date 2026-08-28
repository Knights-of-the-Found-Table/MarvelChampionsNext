package campaign

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// scenario registry content for BuildGame
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/galaxysmostwanted"
)

func soloPlayer() []PlayerLog {
	return []PlayerLog{{
		Slot: 1, UserID: 1, Name: "Tester", HeroBase: "01001",
		Deck: map[string]int{"01088": 3, "01003": 2},
	}}
}

func newRRS(t *testing.T) *State {
	t.Helper()
	return New("rrs", "standard", soloPlayer())
}

func wonRRS(hp int, koed bool, hero bool) Snapshot {
	return Snapshot{
		Won: true, HP: []int{hp}, KOed: []bool{koed}, HeroForm: []bool{hero},
		Discard:        [][]string{nil},
		Rescued:        map[int]string{},
		Engaged:        map[int]bool{},
		PowerStoneSlot: -1,
	}
}

// The Rise of Red Skull chain: tech choice, delay counters, rescued
// allies, Hydra Prison removal, and the finale threat from the log.
func TestRRSCampaignFlow(t *testing.T) {
	st := newRRS(t)
	st.Status = "interlude"

	// Scenario 1: experimental weapons recorded, TECH upgrade offered.
	snap := wonRRS(9, false, true)
	snap.Experimental = []string{"04073", "04073"} // duplicate entry dedups
	ApplyVictory(st, snap)
	if len(st.Experimental) != 1 || st.Experimental[0] != "04073" {
		t.Fatalf("experimental = %v", st.Experimental)
	}
	if st.PendingChoices[0] != ChoiceTech {
		t.Fatalf("pending = %v, want tech", st.PendingChoices)
	}
	if err := ApplyChoice(st, 0, "04156"); err != nil {
		t.Fatalf("tech choice: %v", err)
	}
	if st.Players[0].Deck["04156"] != 1 {
		t.Fatalf("tech upgrade not in deck: %v", st.Players[0].Deck)
	}

	// Scenario 2: delay counters recorded, condition upgrade skippable.
	snap = wonRRS(8, false, true)
	snap.DelayCounters = 4
	ApplyVictory(st, snap)
	if st.DelayCounters != 4 {
		t.Fatalf("delay counters = %d", st.DelayCounters)
	}
	if st.PendingChoices[0] != ChoiceCondition {
		t.Fatalf("pending = %v, want condition", st.PendingChoices)
	}
	if err := ApplyChoice(st, 0, ""); err != nil {
		t.Fatalf("skip condition: %v", err)
	}

	// Scenario 3: the rescued captive joins the deck.
	snap = wonRRS(7, false, true)
	snap.Rescued[0] = "04098"
	ApplyVictory(st, snap)
	if st.Players[0].Deck["04098"] != 1 {
		t.Fatalf("rescued ally missing from deck")
	}

	// Scenario 4: Hydra Prison in play removes its allies from the deck.
	snap = wonRRS(6, false, true)
	snap.HydraPrisonInPlay = true
	snap.PrisonAllies = []string{"04098"}
	ApplyVictory(st, snap)
	if len(st.RemovedAllies) != 1 || st.RemovedAllies[0] != "04098" {
		t.Fatalf("removedAllies = %v", st.RemovedAllies)
	}
	if st.Players[0].Deck["04098"] != 0 {
		t.Fatalf("prison ally still in deck")
	}

	// Finale: threat equals the recorded delay counters; winning
	// completes the campaign.
	opts, err := BuildGame(st, 1)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	if opts.ScenarioID != "04128" {
		t.Fatalf("scenario = %s, want the Red Skull", opts.ScenarioID)
	}
	if opts.Campaign.MainSchemeThreat != 4 {
		t.Fatalf("main scheme threat = %d, want 4", opts.Campaign.MainSchemeThreat)
	}
	if len(opts.Campaign.PreShuffle) != 1 || opts.Campaign.PreShuffle[0] != "04073" {
		t.Fatalf("preShuffle = %v, want the recorded weapon", opts.Campaign.PreShuffle)
	}
	ApplyVictory(st, wonRRS(5, false, true))
	if st.Status != "complete" {
		t.Fatalf("status = %s, want complete", st.Status)
	}
}

// Galaxy's Most Wanted unit economy: base + victory values + scenario
// bonuses; the Market spends them and never sells a second copy.
func TestGMWUnitsAndMarket(t *testing.T) {
	st := New("gmw", "standard", soloPlayer())
	st.Status = "interlude"

	snap := Snapshot{
		Won: true, HP: []int{10}, KOed: []bool{false}, HeroForm: []bool{true},
		Discard: [][]string{nil}, Rescued: map[int]string{}, Engaged: map[int]bool{},
		PowerStoneSlot: -1,
		NoMinions:      true,
		MainSchemeCode: "16062b", // stage 1B
		VictoryPoints:  5,        // capped at 3
		HeadhunterDown: true,
	}
	ApplyVictory(st, snap)
	// 1 (alive) + 3 (victory values) + 1 (no minions) + 1 (stage 1B) = 6.
	if got := st.Players[0].Units; got != 6 {
		t.Fatalf("units = %d, want 6", got)
	}
	if !st.Headhunter[0] {
		t.Fatalf("headhunter mark missing")
	}

	// Buying the same card twice fails; cost is deducted once.
	if err := BuyMarket(st, 0, "16150"); err != nil {
		t.Fatalf("buy: %v", err)
	}
	if st.Players[0].Units != 5 {
		t.Fatalf("units after buy = %d, want 5", st.Players[0].Units)
	}
	if err := BuyMarket(st, 0, "16150"); err == nil {
		t.Fatalf("second copy of a Market card was sold")
	}

	// Scenario 3 setup removes the recorded collection cards.
	st.Collection = []string{"01088"}
	st.Index = 2
	st.Status = "interlude"
	opts, err := BuildGame(st, 2)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	if opts.Campaign.StartSideScheme != gmwNoEscape {
		t.Fatalf("side scheme = %s", opts.Campaign.StartSideScheme)
	}
	if len(opts.Campaign.RemoveFromGame) != 1 || opts.Campaign.RemoveFromGame[0] != "01088" {
		t.Fatalf("removeFromGame = %v", opts.Campaign.RemoveFromGame)
	}
	if !opts.Campaign.DrawUpAfterRemove {
		t.Fatalf("players must draw back up")
	}
}

// Expert campaigns record remaining hit points (capped at the printed
// base), eliminate dead players from the victory steps, and heal via the
// obligation/units toggle at the next setup.
func TestExpertPersistence(t *testing.T) {
	st := New("rrs", "expert", soloPlayer())
	st.Status = "interlude"

	// Spider-Man's printed HP is 10; 12 remaining caps to 10.
	snap := wonRRS(12, false, true)
	ApplyVictory(st, snap)
	if st.Players[0].HP != 10 {
		t.Fatalf("hp = %d, want capped 10", st.Players[0].HP)
	}

	// Resolve the outstanding victory choice, then use the heal toggle:
	// it adds a random obligation to the next deck and resets the
	// recorded damage.
	if err := ApplyChoice(st, 0, "04155"); err != nil {
		t.Fatalf("tech choice: %v", err)
	}
	if err := SetHeal(st, 0, true); err != nil {
		t.Fatalf("heal: %v", err)
	}
	opts, err := BuildGame(st, 3)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	if opts.Players[0].StartHP != 0 {
		t.Fatalf("StartHP = %d, want printed value (0)", opts.Players[0].StartHP)
	}
	if len(opts.Players[0].DeckEncounters) != 1 {
		t.Fatalf("heal obligation missing: %v", opts.Players[0].DeckEncounters)
	}
	for _, code := range opts.Players[0].DeckEncounters {
		found := false
		for _, o := range rrsObligations {
			if o == code {
				found = true
			}
		}
		if !found {
			t.Fatalf("unknown obligation %q", code)
		}
	}

	// Eliminated heroes skip the victory steps and rejoin at full health.
	st2 := New("rrs", "expert", soloPlayer())
	st2.Status = "interlude"
	dead := wonRRS(0, true, false)
	dead.Experimental = []string{"04072"}
	ApplyVictory(st2, dead)
	if _, ok := st2.PendingFor(0); ok {
		t.Fatalf("eliminated player was offered a victory choice")
	}
	if st2.Players[0].HP != 0 {
		t.Fatalf("eliminated player hp = %d, want full", st2.Players[0].HP)
	}

	// Losing the Ronan finale on expert loses the campaign.
	st3 := New("gmw", "expert", soloPlayer())
	st3.Index = 4
	st3.Status = "active"
	ApplyDefeat(st3)
	if st3.Status != "lost" {
		t.Fatalf("status = %s, want lost", st3.Status)
	}
	// Standard defeat is a free retry.
	st4 := newRRS(t)
	st4.Status = "active"
	ApplyDefeat(st4)
	if st4.Status != "interlude" || st4.Index != 0 {
		t.Fatalf("retry state = %s/%d", st4.Status, st4.Index)
	}
}

// Nebula's setup: recorded Galactic Artifacts drive the matching riders.
func TestGMWArtifactSetup(t *testing.T) {
	st := New("gmw", "standard", soloPlayer())
	st.Index = 3
	st.Status = "interlude"
	st.Artifacts = []string{"16127", "16129"}
	opts, err := BuildGame(st, 4)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	ctx := opts.Campaign
	if ctx.ShipEvasion != 1 {
		t.Fatalf("ship evasion = %d, want 1 (Monarch Egg)", ctx.ShipEvasion)
	}
	if !ctx.VillainBoostFacedown {
		t.Fatalf("Philosopher's Stone boost missing")
	}
	if ctx.VillainTough || ctx.FirstPlayerEncounterFacedown {
		t.Fatalf("unrecorded artifact effects applied")
	}
	for _, code := range st.Artifacts {
		found := false
		for _, c := range ctx.PreShuffle {
			if c == code {
				found = true
			}
		}
		if !found {
			t.Fatalf("artifact %q not shuffled back in", code)
		}
	}
}

// The engine options carry the campaign context through to NewGame.
func TestBuildGameRuns(t *testing.T) {
	st := New("gmw", "standard", soloPlayer())
	opts, err := BuildGame(st, 5)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	if _, err := engine.NewGame(opts); err != nil {
		t.Fatalf("NewGame: %v", err)
	}
}
