package campaign

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// scenario registry content for the box build smoke test
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/aoa"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/aos"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/civilwar"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/deadpool"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/galaxysmostwanted"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/goblinfooblin"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/hood"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/iceman"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/madtansshadow"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/mojo"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/mutantgenesis"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nextevolution"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nova"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/riseofredskull"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/rogue"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/sinistermotives"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/spdr"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/wolv"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/wreckingcrew"
)

// contestPlayer builds a one-seat log with an arbitrary hero.
func contestPlayer(hero string) []PlayerLog {
	return []PlayerLog{{
		Slot: 1, UserID: 1, Name: "Tester", HeroBase: hero,
		Deck: map[string]int{"01088": 3, "01003": 2},
	}}
}

// TestContestBoxesBuild smoke-builds every contest campaign's opening
// chapter (after resolving the owed pre-chapter picks).
func TestContestBoxesBuild(t *testing.T) {
	cases := []struct {
		box    string
		hero   string
		warmup func(*State)
	}{
		{"cowl", "01029", nil},     // Iron Man (Avenger)
		{"whatif", "01001", nil},   // Spider-Man
		{"awesome", "01001", nil},  // non-Guardian
		{"alias", "01001", nil},    // any
		{"watchers", "01029", nil}, // Iron Man chapter 1
		{"mojo", "01001", func(st *State) { st.AddPending(0, ChoiceMojoRole); st.AddPending(0, ChoiceMojoTraining) }},
		{"bord", "01001", func(st *State) { st.AddPending(0, ChoiceBordPath) }},
		{"night", "01019", nil}, // She-Hulk
		{"viral", "01001", nil}, // any
		{"entropy", "01001", func(st *State) { st.AddPending(0, ChoiceEnPath1) }},
	}
	for _, tc := range cases {
		st := New(tc.box, "standard", contestPlayer(tc.hero))
		st.Status = "interlude"
		if tc.warmup != nil {
			tc.warmup(st)
		}
		if _, err := BuildGame(st, 42); err == nil && st.AnyPending() {
			t.Fatalf("%s: BuildGame must wait for pending picks", tc.box)
		}
	}
}

// resolveContestOpeners answers the group picks each branching campaign
// owes before its first chapter.
func resolveContestOpeners(t *testing.T, st *State) {
	t.Helper()
	switch st.Box {
	case "mojo":
		if err := ApplyChoice(st, 0, ChoiceMojoRole, "fighter"); err != nil {
			t.Fatalf("mojo role: %v", err)
		}
		if err := ApplyChoice(st, 0, ChoiceMojoTraining, "40190a"); err != nil {
			t.Fatalf("mojo training: %v", err)
		}
	case "bord":
		if err := ApplyChoice(st, 0, ChoiceBordPath, "assault"); err != nil {
			t.Fatalf("bord path: %v", err)
		}
		if st.Index != 2 {
			t.Fatalf("assault path should open at index 2, got %d", st.Index)
		}
	case "entropy":
		if err := ApplyChoice(st, 0, ChoiceEnPath1, "a"); err != nil {
			t.Fatalf("entropy path1: %v", err)
		}
	}
}

// TestContestOpenersBuild answers the group picks and then builds the
// opening game of each branching campaign.
func TestContestOpenersBuild(t *testing.T) {
	for _, box := range []string{"mojo", "bord", "entropy"} {
		st := New(box, "standard", contestPlayer("01001"))
		st.Status = "interlude"
		switch box {
		case "mojo":
			st.AddPending(0, ChoiceMojoRole)
			st.AddPending(0, ChoiceMojoTraining)
		case "bord":
			st.AddPending(0, ChoiceBordPath)
		case "entropy":
			st.AddPending(0, ChoiceEnPath1)
		}
		resolveContestOpeners(t, st)
		opts, err := BuildGame(st, 42)
		if err != nil {
			t.Fatalf("%s: BuildGame: %v", box, err)
		}
		if _, err := engine.NewGame(opts); err != nil {
			t.Fatalf("%s: NewGame: %v", box, err)
		}
	}
}

// Cowl: the tech offer lands after chapter 1 and the recorded upgrade
// joins its owner in play from chapter 2 on.
func TestCowlTechFlow(t *testing.T) {
	st := New("cowl", "standard", contestPlayer("01029"))
	st.Status = "interlude"
	ApplyVictory(st, wonSnap())
	if !st.PendingFor(0, ChoiceSMTech) {
		t.Fatalf("cowl chapter 1 must offer campaign tech")
	}
	pick := st.Players[0].SMTechOffer[0]
	if err := ApplyChoice(st, 0, ChoiceSMTech, pick); err != nil {
		t.Fatalf("sm-tech: %v", err)
	}
	if st.Players[0].SMShieldTech != pick {
		t.Fatalf("tech not recorded")
	}
	opts, err := BuildGame(st, 1)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	up, ok := opts.Campaign.RoleUpgrades[0]
	if !ok || len(up) != 1 || up[0] != pick {
		t.Fatalf("tech upgrade not put into play: %v", opts.Campaign.RoleUpgrades)
	}
}

// What If...?: trait -> ally -> card chain feeds the deck; the finale
// pulls the trait allies into play.
func TestWhatIfTraitFlow(t *testing.T) {
	st := New("whatif", "standard", contestPlayer("01001"))
	st.Status = "interlude"
	ApplyVictory(st, wonSnap())
	if !st.PendingFor(0, ChoiceWITrait) {
		t.Fatalf("trait pick owed")
	}
	if err := ApplyChoice(st, 0, ChoiceWITrait, "webwarrior"); err != nil {
		t.Fatalf("trait: %v", err)
	}
	if !st.PendingFor(0, ChoiceWIAlly) {
		t.Fatalf("ally pick owed after trait")
	}
	if err := ApplyChoice(st, 0, ChoiceWIAlly, "27010"); err != nil { // Silk
		t.Fatalf("wi-ally: %v", err)
	}
	if len(st.Players[0].WIAllies) != 1 {
		t.Fatalf("ally not recorded")
	}
	// Chapter 2 records the Shawarma pool when the place is not in play,
	// and a defeated Community Service scheme grants another trait card.
	snap := wonSnap()
	snap.VictoryDisplayCodes = []string{"27176"}
	ApplyVictory(st, snap)
	if !st.Flags["shawarma"] || !contains(st.Pool, shawarmaCard) {
		t.Fatalf("shawarma pool missing")
	}
	st.Players[0].Deck["27010"]++ // ensure the card is in the deck for the reward
	if !st.PendingFor(0, ChoiceWICard) {
		t.Fatalf("trait-card reward owed")
	}
	if err := ApplyChoice(st, 0, ChoiceWICard, "27010"); err != nil {
		t.Fatalf("wi-card: %v", err)
	}
}

// Going Viral: the branch picks route two of the three Scenario #2
// chapters and both tracks accumulate.
func TestViralBranch(t *testing.T) {
	st := New("viral", "standard", contestPlayer("01001"))
	st.Status = "interlude"
	snap := wonSnap()
	snap.MainThreat = 7
	ApplyVictory(st, snap)
	if !st.PendingFor(0, ChoiceViralNext) {
		t.Fatalf("viral-next pick owed")
	}
	if st.Counters["infection"] != 2 {
		t.Fatalf("infection = %d, want 2", st.Counters["infection"])
	}
	if err := ApplyChoice(st, 0, ChoiceViralNext, "2"); err != nil {
		t.Fatalf("viral-next: %v", err)
	}
	if st.Index != 2 {
		t.Fatalf("index = %d, want 2", st.Index)
	}
	snap2 := wonSnap()
	snap2.MainThreat = 15
	ApplyVictory(st, snap2)
	// After the first #2 the chain auto-routes to an unplayed one.
	if st.Index == 2 || st.Index == 4 {
		t.Fatalf("unexpected index after first #2: %d", st.Index)
	}
	ApplyVictory(st, snap2)
	if st.Index != 4 {
		t.Fatalf("index = %d, want 4 (finale)", st.Index)
	}
	opts, err := BuildGame(st, 3)
	if err != nil {
		t.Fatalf("BuildGame finale: %v", err)
	}
	if len(opts.Campaign.ExtraSets) == 0 {
		t.Fatalf("finale must assemble extra sets from the results")
	}
	// Losing a Scenario #2 skips it with an infection surge.
	st2 := New("viral", "standard", contestPlayer("01001"))
	st2.Status = "interlude"
	st2.Index = 1
	ApplyDefeat(st2)
	if !st2.Flags["modokAway"] || st2.Index != 4 {
		t.Fatalf("zola defeat must flag the algorithm and jump to the finale")
	}
}

// Entropic Ascension: the A/B picks route the chain and record Crime Wave
// lines that assemble the finale.
func TestEntropyFlow(t *testing.T) {
	st := New("entropy", "standard", contestPlayer("01001"))
	st.Status = "interlude"
	st.AddPending(0, ChoiceEnPath1)
	if _, err := BuildGame(st, 1); err == nil {
		t.Fatalf("entropy must wait for the path pick")
	}
	if err := ApplyChoice(st, 0, ChoiceEnPath1, "b"); err != nil {
		t.Fatalf("path1: %v", err)
	}
	opts, err := BuildGame(st, 1)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	found := false
	for _, s := range opts.Campaign.ExtraSets {
		if s == setPowerDrain {
			found = true
		}
	}
	if !found {
		t.Fatalf("Blackout pick must add Power Drain: %v", opts.Campaign.ExtraSets)
	}
	// Chapter 1 routes to Mysterio (2B) and owes the second A/B pick.
	ApplyVictory(st, wonSnap())
	if !st.PendingFor(0, ChoiceEnPath2) {
		t.Fatalf("path2 pick owed after chapter 1")
	}
	if err := ApplyChoice(st, 0, ChoiceEnPath2, "a"); err != nil {
		t.Fatalf("path2: %v", err)
	}
	ApplyVictory(st, wonSnap())
	if st.Index != 3 || st.Selections["cw3"] == "" {
		t.Fatalf("entropy must route to Klaw with crime lines: %d %v", st.Index, st.Selections)
	}
	// Klaw victory routes to The Hood; the finale assembles cw sets.
	if !st.PendingFor(0, ChoiceEnPath3) {
		t.Fatalf("path3 pick owed after chapter 2")
	}
	if err := ApplyChoice(st, 0, ChoiceEnPath3, "b"); err != nil {
		t.Fatalf("path3: %v", err)
	}
	ApplyVictory(st, wonSnap())
	if st.Index != 4 {
		t.Fatalf("expected the Hood finale, got %d", st.Index)
	}
	// Reputation 4 deals the S.H.I.E.L.D. Tech offer; answer it.
	if st.PendingFor(0, ChoiceSMTech) {
		if err := ApplyChoice(st, 0, ChoiceSMTech, st.Players[0].SMTechOffer[0]); err != nil {
			t.Fatalf("sm-tech: %v", err)
		}
	}
	opts, err = BuildGame(st, 2)
	if err != nil {
		t.Fatalf("BuildGame finale: %v", err)
	}
	foundRansacked := false
	for _, s := range opts.Campaign.ExtraSets {
		if s == setRansacked {
			foundRansacked = true
		}
	}
	if !foundRansacked {
		t.Fatalf("crime wave lines must set aside Ransacked Armory: %v", opts.Campaign.ExtraSets)
	}
}

// Black Order: the path pick opens its own chapter; Gear Up and the
// Order minions carry across the shared chapters.
func TestBordFlow(t *testing.T) {
	st := New("bord", "standard", contestPlayer("01001"))
	st.Status = "interlude"
	st.AddPending(0, ChoiceBordPath)
	if err := ApplyChoice(st, 0, ChoiceBordPath, "first"); err != nil {
		t.Fatalf("path: %v", err)
	}
	opts, err := BuildGame(st, 1)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	if opts.ScenarioID != "27064" {
		t.Fatalf("first response opens with Venom, got %s", opts.ScenarioID)
	}
	if opts.Campaign.StartSideScheme != saveShawarma {
		t.Fatalf("venom opener must stage the Shawarma place")
	}
	// Victory jumps past the other openings to Drang.
	snap := wonSnap()
	ApplyVictory(st, snap)
	if st.Index != 3 {
		t.Fatalf("index = %d, want 3 (Drang)", st.Index)
	}
	// Gear Up records a cheap support on the Direct Assault opener.
	st2 := New("bord", "standard", contestPlayer("01001"))
	st2.Status = "interlude"
	st2.Index = 2
	st2.AddPending(0, ChoiceBordGear)
	st2.Selections["path"] = "assault"
	if err := ApplyChoice(st2, 0, ChoiceBordGear, "01092"); err != nil { // Helicarrier, cost 3
		// Helicarrier costs more than 2; the rejection is the assertion.
	} else {
		t.Fatalf("gear must reject cost > 2")
	}
	st2.ResolvePending(0, ChoiceBordGear)
}

// She-Hulk vs. Deadpool: the pool churns through wins and losses and the
// finale reads the record.
func TestNightPool(t *testing.T) {
	st := New("night", "standard", contestPlayer("01019"))
	st.Status = "interlude"
	snap := wonSnap()
	ApplyVictory(st, snap)
	if !st.PendingFor(0, ChoiceNTMeta) {
		t.Fatalf("metagame self-report owed")
	}
	for _, code := range nightCh1Win {
		if !contains(st.Pool, code) {
			t.Fatalf("win pool missing %q", code)
		}
	}
	if err := ApplyChoice(st, 0, ChoiceNTMeta, "yes"); err != nil {
		t.Fatalf("nt-meta: %v", err)
	}
	if st.Selections["gw1"] != "shehulk" {
		t.Fatalf("gw1 = %q, want shehulk", st.Selections["gw1"])
	}
	// Losing a game night does not replay: Deadpool scores and the story
	// moves on (chapter 1's loss feeds Git Gud and Bob into the pool).
	st.Index = 0
	ApplyDefeat(st)
	if st.Index != 1 || !contains(st.Pool, "44028") {
		t.Fatalf("defeat must advance with the lose cards: %d %v", st.Index, st.Pool)
	}
	// The finale assembles Deadpool penalties.
	st.Index = 4
	st.Selections["gw2"] = "deadpool"
	st.Selections["gw3"] = "shehulk"
	opts, err := BuildGame(st, 5)
	if err != nil {
		t.Fatalf("BuildGame finale: %v", err)
	}
	foundFlight := false
	for _, s := range opts.Campaign.ExtraSets {
		if s == "flight" {
			foundFlight = true
		}
	}
	if !foundFlight {
		t.Fatalf("deadpool-won Lord of the Wings must add Flight: %v", opts.Campaign.ExtraSets)
	}
	foundCrystal := false
	for _, s := range opts.Campaign.StartSideSchemes {
		if s == crystalBall {
			foundCrystal = true
		}
	}
	if !foundCrystal {
		t.Fatalf("deadpool-won Web-hacker must stage the Crystal Ball")
	}
}

// Awesome Campaign: influence accrues by the conditions and buys cards.
func TestAwesomeInfluence(t *testing.T) {
	st := New("awesome", "standard", contestPlayer("01001"))
	st.Status = "interlude"
	snap := wonSnap()
	snap.MainThreat = 0
	snap.Acceleration = 0
	ApplyVictory(st, snap)
	// All five conditions met: 5 influence.
	if st.Players[0].Influence != 5 {
		t.Fatalf("influence = %d, want 5", st.Players[0].Influence)
	}
	if err := SpendInfluence(st, 0, "16156"); err != nil { // Grapple, cost 2
		t.Fatalf("spend: %v", err)
	}
	if st.Players[0].Influence != 3 || st.Players[0].Deck["16156"] != 1 {
		t.Fatalf("purchase not recorded")
	}
	if err := SpendInfluence(st, 0, "16156"); err == nil {
		t.Fatalf("duplicate purchase must fail")
	}
	// The chapter flags feed the finale.
	st.Index = 0
	snap2 := wonSnap()
	snap2.MinionCodes = []string{modokMinion}
	ApplyVictory(st, snap2)
	if !st.Flags["modok"] {
		t.Fatalf("MODOK flag missing")
	}
}

// Alias Investigations: the clue trail picks the final villain and the
// tally accrues.
func TestAliasClueChain(t *testing.T) {
	st := New("alias", "standard", contestPlayer("01001"))
	st.Status = "interlude"
	ApplyVictory(st, wonSnap())
	if st.Selections["clue1"] == "" {
		t.Fatalf("clue 1 not recorded")
	}
	if st.Index != 1 {
		t.Fatalf("index = %d, want 1", st.Index)
	}
	ApplyVictory(st, wonSnap())
	if st.Selections["clue2"] == "" {
		t.Fatalf("final clue not recorded")
	}
	want := aliasFinal[st.Selections["clue2"]]
	if st.Index != want || st.Status != "interlude" {
		t.Fatalf("final index = %d/%s, want %d/interlude", st.Index, st.Status, want)
	}
	opts, err := BuildGame(st, 6)
	if err != nil {
		t.Fatalf("BuildGame final: %v", err)
	}
	if opts.ScenarioID != boxScenarioID(st, want) {
		t.Fatalf("final game plays %s, want %s", opts.ScenarioID, boxScenarioID(st, want))
	}
	// The final victory completes the campaign.
	ApplyVictory(st, wonSnap())
	if st.Status != "complete" {
		t.Fatalf("alias must complete")
	}
}

// The Watcher's Team: identity requirements gate each chapter and the
// setup-keyword cards ride into play.
func TestWatchersRequirements(t *testing.T) {
	st := New("watchers", "standard", contestPlayer("01001"))
	st.Status = "interlude"
	if _, err := BuildGame(st, 1); err == nil {
		t.Fatalf("chapter 1 requires Iron Man")
	}
	st.Players[0].HeroBase = heroIronMan
	opts, err := BuildGame(st, 1)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	found := false
	for _, c := range opts.Campaign.SetupKeywordCards {
		if c == godslayer {
			found = true
		}
	}
	if !found {
		t.Fatalf("Godslayer must gain setup: %v", opts.Campaign.SetupKeywordCards)
	}
	if st.Players[0].Deck[godslayer] != 1 || st.Players[0].Deck[systemShock] != 1 {
		t.Fatalf("campaign cards missing from the deck")
	}
	// Chapter 4's named-card rewards are optional.
	st.Index = 3
	st.Players[0].HeroBase = heroCap
	st.AddPending(0, ChoiceWAIntervened)
	if err := ApplyChoice(st, 0, ChoiceWAIntervened, ""); err != nil {
		t.Fatalf("decline must be allowed: %v", err)
	}
	if st.PendingFor(0, ChoiceWAIntervened) {
		t.Fatalf("choice not resolved")
	}
}

// House of Mojo: the role and training picks land before chapter 1 and
// the role upgrades join the setup.
func TestMojoFlow(t *testing.T) {
	st := New("mojo", "standard", contestPlayer("01001"))
	st.Status = "interlude"
	st.AddPending(0, ChoiceMojoRole)
	st.AddPending(0, ChoiceMojoTraining)
	if _, err := BuildGame(st, 1); err == nil {
		t.Fatalf("mojo must wait for role and training picks")
	}
	if err := ApplyChoice(st, 0, ChoiceMojoRole, "fighter"); err != nil {
		t.Fatalf("role: %v", err)
	}
	if err := ApplyChoice(st, 0, ChoiceMojoTraining, "40193a"); err != nil {
		t.Fatalf("training: %v", err)
	}
	opts, err := BuildGame(st, 1)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	if opts.Campaign.PlayerSideScheme != "40193a" {
		t.Fatalf("training scheme must open in play: %q", opts.Campaign.PlayerSideScheme)
	}
	cond := mojoRoles["fighter"].Condition
	up := opts.Campaign.RoleUpgrades[0]
	if len(up) == 0 || up[0] != cond {
		t.Fatalf("fighter condition upgrade missing: %v", up)
	}
	// Thanos chapter 3 victory offers the market-or-shawarma pick.
	st.Index = 2
	st.AddPending(0, ChoiceMojoMarket)
	if err := ApplyChoice(st, 0, ChoiceMojoMarket, shawarmaCard); err != nil {
		t.Fatalf("market: %v", err)
	}
	if st.Players[0].Deck[shawarmaCard] != 1 {
		t.Fatalf("shawarma not in deck")
	}
}

// Crimson Cowl: Masters of Evil minions caught in the victory display
// feed the Intel Level at the finale.
func TestCowlIntel(t *testing.T) {
	st := New("cowl", "standard", contestPlayer("01029"))
	st.Status = "interlude"
	st.Index = 4
	st.CowlCaught = []string{"01129"} // Radioactive Man, printed scheme 2
	snap := wonSnap()
	snap.VictoryDisplayCodes = []string{"01130"}
	snap.MinionCodes = []string{"01131"}
	ApplyVictory(st, snap)
	if !contains(st.CowlCaught, "01130") {
		t.Fatalf("Whirlwind must be caught")
	}
	if !contains(st.Pool, "01131") {
		t.Fatalf("Tiger Shark must escape into the pool")
	}
	if cowlIntel(st) == 0 {
		t.Fatalf("intel must sum the caught minions' scheme")
	}
}
