package campaign

import (
	"testing"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	// scenario registry content for the box build smoke test
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/aoa"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/aos"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/civilwar"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/galaxysmostwanted"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/madtansshadow"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/mutantgenesis"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nextevolution"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/riseofredskull"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/sinistermotives"
)

func solo(box string) *State {
	st := New(box, "standard", soloPlayer())
	st.Status = "interlude"
	return st
}

func wonSnap() Snapshot {
	return Snapshot{
		Won: true, HP: []int{8}, KOed: []bool{false}, HeroForm: []bool{true},
		Discard:        [][]string{nil},
		Rescued:        map[int]string{},
		Engaged:        map[int]bool{},
		PowerStoneSlot: -1,
		NoMinions:      true,
	}
}

// MTS: pool rewards and penalties churn between scenarios.
func TestMTSPool(t *testing.T) {
	st := solo("mts")
	snap := wonSnap()
	snap.Experimental = []string{"04073"}
	ApplyVictory(st, snap) // Ebony Maw: landing pad defeated -> Cosmo
	for _, code := range st.Pool {
		if code == mtsCosmo {
			found := true
			_ = found
		} else {
			t.Fatalf("unexpected pool entry %q", code)
		}
	}
	if len(st.Pool) != 1 || st.Pool[0] != mtsCosmo {
		t.Fatalf("pool = %v, want [Cosmo]", st.Pool)
	}
	// Tower Defense in between; the Thanos chapter consumes the pool.
	ApplyVictory(st, wonSnap())
	opts, err := BuildGame(st, 1)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	foundCosmo := false
	for _, c := range opts.Campaign.PoolAllies {
		if c == mtsCosmo {
			foundCosmo = true
		}
	}
	if !foundCosmo {
		t.Fatalf("Cosmo missing from pool allies: %v", opts.Campaign.PoolAllies)
	}
	// Infinity Stones 1B completed -> flagged and milled at the finale.
	snap = wonSnap()
	snap.MainStage = 2
	ApplyVictory(st, snap) // Tower Defense
	ApplyVictory(st, snap) // Thanos
	if !st.Flags["stones"] {
		t.Fatalf("stones flag missing")
	}
	st.Index = 3 // Hela next
	opts, err = BuildGame(st, 2)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	if !opts.Campaign.DiscardTopHalf {
		t.Fatalf("finale must mill half of each deck")
	}
}

// SM: reputation nodes grant boons and queue choices.
func TestSMReputation(t *testing.T) {
	st := solo("sm")
	snap := wonSnap()
	snap.VictoryPoints = 9 // 9 + 5 conditions = 14 reputation
	ApplyVictory(st, snap)
	if st.Counters["reputation"] < 10 {
		t.Fatalf("reputation = %d, want >=10", st.Counters["reputation"])
	}
	if !st.PendingFor(0, ChoiceSMTech) {
		t.Fatalf("pending = %v, want sm-tech", st.PendingChoices)
	}
	if len(st.Players[0].SMTechOffer) != 3 {
		t.Fatalf("tech offer = %v", st.Players[0].SMTechOffer)
	}
	pick := st.Players[0].SMTechOffer[0]
	if err := ApplyChoice(st, 0, ChoiceSMTech, pick); err != nil {
		t.Fatalf("sm-tech: %v", err)
	}
	// Aspect advantage and planning picks are also owed at reputation 12.
	if err := ApplyChoice(st, 0, ChoiceSMAspect, "01088"); err != nil {
		t.Fatalf("sm-aspect: %v", err)
	}
	if err := ApplyChoice(st, 0, ChoiceSMPlan, "01088"); err != nil {
		t.Fatalf("sm-plan: %v", err)
	}
	if st.Players[0].SMShieldTech != pick {
		t.Fatalf("tech not recorded")
	}
	// Osborn attachments were recorded by node 2/12.
	if len(st.SMOsborn) == 0 {
		t.Fatalf("osborn tech missing")
	}
	// Next setup shuffles recorded Osborn tech in.
	opts, err := BuildGame(st, 1)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	for _, code := range st.SMOsborn {
		found := false
		for _, c := range opts.Campaign.PreShuffle {
			if c == code {
				found = true
			}
		}
		if !found {
			t.Fatalf("osborn %q not shuffled in", code)
		}
	}
}

// MG: roles gate the chapter-1 game and the Future Past deck churns.
func TestMGRolesAndFuturePast(t *testing.T) {
	st := solo("mg")
	st.MGFuturePast = FuturePastSeed()
	st.AddPending(0, ChoiceMGRole)
	if _, err := BuildGame(st, 1); err == nil {
		t.Fatalf("chapter 1 must wait for role picks")
	}
	if err := ApplyChoice(st, 0, ChoiceMGRole, "brawler"); err != nil {
		t.Fatalf("role: %v", err)
	}
	opts, err := BuildGame(st, 1)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	if len(opts.Campaign.RoleUpgrades) != 1 {
		t.Fatalf("role upgrade missing: %v", opts.Campaign.RoleUpgrades)
	}
	for _, codes := range opts.Campaign.RoleUpgrades {
		for _, code := range codes {
			found := false
			for _, u := range mgRoles["brawler"] {
				if u == code {
					found = true
				}
			}
			if !found {
				t.Fatalf("upgrade %q not in the brawler set", code)
			}
		}
	}
	// Future Past cards defeated into the victory display leave the pool.
	snap := wonSnap()
	snap.VictoryDisplayCodes = []string{"32166", "32167"}
	ApplyVictory(st, snap)
	if len(st.MGFuturePast) != 3 {
		t.Fatalf("future past = %v, want 3 left", st.MGFuturePast)
	}
}

// NX: the chosen scheme earns its environment and haunts the deck.
func TestNXSchemeLoop(t *testing.T) {
	st := solo("nx")
	QueueNXScheme(st)
	if !st.PendingFor(0, ChoiceNXScheme) {
		t.Fatalf("scheme pick not queued")
	}
	if err := ApplyChoice(st, 0, ChoiceNXScheme, "40190a"); err != nil {
		t.Fatalf("nx scheme: %v", err)
	}
	opts, err := BuildGame(st, 1)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	if opts.Campaign.PlayerSideScheme != "40190a" {
		t.Fatalf("scheme not in play: %q", opts.Campaign.PlayerSideScheme)
	}
	if len(opts.Campaign.PreShuffle) != 1 || opts.Campaign.PreShuffle[0] != "40199" {
		t.Fatalf("paired encounter card missing: %v", opts.Campaign.PreShuffle)
	}
	// Winning with the environment in play earns it.
	snap := wonSnap()
	snap.EnvironmentCodes = []string{"40190b"}
	ApplyVictory(st, snap)
	if len(st.NXEnvEarned) != 1 {
		t.Fatalf("environment not earned: %v", st.NXEnvEarned)
	}
	if !st.PendingFor(0, ChoiceNXScheme) {
		t.Fatalf("next scheme pick not queued: %v", st.PendingChoices)
	}
	// The earned environment (and the Malice card) persist.
	if err := ApplyChoice(st, 0, ChoiceNXScheme, "40191a"); err != nil {
		t.Fatalf("resolve scheme pick: %v", err)
	}
	opts, err = BuildGame(st, 2)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	foundEnv := false
	for _, e := range opts.Campaign.StartEnvironments {
		if e == "40190b" {
			foundEnv = true
		}
	}
	if !foundEnv {
		t.Fatalf("earned environment missing from setup: %v", opts.Campaign.StartEnvironments)
	}
}

// AoA: missions and Overseers rotate and get struck; the finale is
// decided by Protect the Professor.
func TestAOMissions(t *testing.T) {
	st := solo("aoa")
	opts, err := BuildGame(st, 1)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	if opts.Campaign.MissionScheme == "" || opts.Campaign.MissionOverseer == "" {
		t.Fatalf("mission setup incomplete")
	}
	if !opts.Campaign.MissionTeam {
		t.Fatalf("mission team missing")
	}
	foundAlly := false
	for _, code := range opts.Campaign.HandFetch {
		if code == "ally" {
			foundAlly = true
		}
	}
	if !foundAlly {
		t.Fatalf("ally fetch missing")
	}
	snap := wonSnap()
	snap.MinionCodes = []string{st.AOOverseer}
	ApplyVictory(st, snap)
	if len(st.AOMissionLog) != 1 {
		t.Fatalf("mission not struck")
	}
	if len(st.AOOverseerLog) != 0 {
		t.Fatalf("overseer in play must not be struck: %v", st.AOOverseerLog)
	}
	// Finale: professor not saved = lost.
	st.Index = 4
	st.AOMission = aoaProfessor
	snap = wonSnap()
	snap.SideSchemeBaseCodes = []string{aoaProfessor}
	ApplyVictory(st, snap)
	if st.Status != "lost" {
		t.Fatalf("status = %s, want lost", st.Status)
	}
}

// AoS: the mole combo stays consistent and evidence eliminates combos.
func TestAOSMystery(t *testing.T) {
	st := solo("aos")
	PrepareAOSEvidence(st)
	if len(st.AOImEnvelope) != 3 || len(st.AOShieldEnvelope) != 6 {
		t.Fatalf("envelopes wrong: %v / %v", st.AOImEnvelope, st.AOShieldEnvelope)
	}
	// Gaining all six S.H.I.E.L.D. envelope cards leaves exactly one
	// combination — the mole's.
	for i := 0; i < 6; i++ {
		st.AOEvidence = append(st.AOEvidence, st.AOShieldEnvelope[i])
	}
	remaining := aosCombosRemaining(st)
	total := 0
	var moleMember string
	for member, combos := range remaining {
		total += len(combos)
		for _, combo := range combos {
			if combo == st.AOImEnvelope {
				moleMember = member
			}
		}
	}
	if total != 1 || moleMember == "" {
		t.Fatalf("remaining combos = %d, mole member = %q", total, moleMember)
	}
	if !aosMemberIsMole(st, moleMember) {
		t.Fatalf("deduced member is not the mole")
	}
	// Chapter 1 counters: 2 per member, recorded at victory.
	opts, err := BuildGame(st, 1)
	if err != nil {
		t.Fatalf("BuildGame: %v", err)
	}
	if opts.Campaign.BoardCounters["50181a"] != 2 {
		t.Fatalf("board counters = %v", opts.Campaign.BoardCounters)
	}
	snap := wonSnap()
	snap.EnvCountersByCode = map[string]int{"50181a": 0, "50182a": 3, "50183a": 1}
	ApplyVictory(st, snap)
	if st.AOCounters["50181a"] != 0 || st.AOCounters["50182a"] != 3 {
		t.Fatalf("counters not carried: %v", st.AOCounters)
	}
	if len(st.AOEvidence) != 6 {
		t.Fatalf("evidence = %v", st.AOEvidence)
	}
}

// BuildGame produces engine-loadable options for every box.
func TestAllBoxesBuild(t *testing.T) {
	for _, box := range []string{"rrs", "gmw", "mts", "sm", "mg", "nx", "aoa", "aos"} {
		st := solo(box)
		opts, err := BuildGame(st, 42)
		if err != nil {
			t.Fatalf("%s: BuildGame: %v", box, err)
		}
		if _, err := engine.NewGame(opts); err != nil {
			t.Fatalf("%s: NewGame: %v", box, err)
		}
	}
}
