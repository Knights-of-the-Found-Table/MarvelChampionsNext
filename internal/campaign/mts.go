package campaign

import (
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// The Mad Titan's Shadow campaign. The shared POOL accumulates campaign
// cards from per-scenario results; later setups pull from it. Codes that
// are missing from the data snapshot (Black Swan, Odin's Armor) are kept
// as named markers so the log stays faithful, and their setup effects
// degrade to no-ops.
var (
	mtsLandingPad     = "21180a" // Secure the Landing Pad
	mtsSecurityBreach = "21181"  // side scheme penalty
	mtsShawarmaPlace  = "21182a" // Save the Shawarma Place
	mtsShawarma       = "21183"  // resource reward
	mtsHackComputer   = "21184a" // Hack Sanctuary's Computer
	mtsSystemShock    = "21185"  // obligation penalty
	mtsNornStones     = "21186a" // Find the Norn Stones
	mtsNornStone      = "21187a" // upgrade reward
	mtsSummonedBack   = "21188"  // treachery
	mtsDungeons       = "21189a" // Open the Dungeons
	mtsCosmo          = "17020"  // ally reward (earliest printing)
	mtsTowerDamaged   = "21100"  // damaged Avengers Tower side
	mtsBlackSwan      = "pool-black-swan"
)

// mtsVictory implements the per-scenario victory steps of the campaign
// rulebook (pages 7-25).
func mtsVictory(st *State, snap Snapshot) {
	living := func(i int) bool { return !snap.KOed[i] }
	inPlay := func(code string) bool { return contains(snap.SideSchemeBaseCodes, code) }
	inVictory := func(code string) bool { return contains(snap.VictoryDisplayCodes, code) }
	switch st.Index {
	case 0: // Ebony Maw
		if !inPlay(mtsLandingPad) {
			st.Pool = appendUnique(st.Pool, mtsCosmo)
		}
		if snap.MainStage > 1 { // Attack on Knowhere 1B was completed
			st.Pool = appendUnique(st.Pool, mtsSecurityBreach)
		}
	case 1: // Tower Defense
		if !inPlay(mtsShawarmaPlace) {
			st.Pool = appendUnique(st.Pool, mtsShawarma)
		}
		// Black Swan cannot reach the victory display in this data
		// snapshot, so the "if NOT in the victory display" branch always
		// fires; her marker's setup effect is a documented no-op.
		st.Pool = appendUnique(st.Pool, mtsBlackSwan)
		if contains(snap.EnvironmentCodes, mtsTowerDamaged) {
			st.Flags["towerDamaged"] = true
		}
	case 2: // Thanos
		if !inVictory("21184b") { // Defensive Protocols (flip side)
			st.Pool = appendUnique(st.Pool, mtsSystemShock)
		}
		if snap.MainStage > 1 { // The Infinity Stones 1B was completed
			st.Flags["stones"] = true
		}
	case 3: // Hela
		if !inPlay(mtsNornStones) {
			st.Pool = appendUnique(st.Pool, mtsNornStone)
		}
		// "Odin's Armor in the victory display": the card is not in the
		// data snapshot, so this branch cannot fire (documented gap).
	}
	st.recordExpertHP(snap, living)
	st.Advance()
}

// mtsSetup applies the campaign setups (pages 7-25). Pool card copies
// that join player decks ride the deck snapshots directly.
func mtsSetup(st *State, ctx *engine.CampaignSetup, opts *engine.NewGameOptions) {
	has := func(code string) bool { return contains(st.Pool, code) }
	addDeck := func(code string) {
		for i := range st.Players {
			st.Players[i].Deck[code]++
		}
	}
	shuffleIn := func(code string) { ctx.PreShuffle = append(ctx.PreShuffle, code) }
	switch st.Index {
	case 0: // Ebony Maw
		ctx.StartSideScheme = mtsLandingPad
		shuffleIn(mtsSecurityBreach)
	case 1: // Tower Defense
		ctx.StartSideScheme = mtsShawarmaPlace
		if has(mtsSecurityBreach) {
			shuffleIn(mtsSecurityBreach)
		}
	case 2: // Thanos
		ctx.StartSideScheme = mtsHackComputer
		if has(mtsCosmo) {
			ctx.PoolAllies = append(ctx.PoolAllies, mtsCosmo)
		}
		if has(mtsSecurityBreach) {
			shuffleIn(mtsSecurityBreach)
		}
		if has(mtsShawarma) {
			addDeck(mtsShawarma)
		}
		// Black Swan's "into play engaged with the first player" has no
		// card in the data snapshot — recorded in the pool, no effect.
		_ = has(mtsBlackSwan)
		if st.Flags["towerDamaged"] {
			ctx.StartDamageEachPlayer = 3
		}
	case 3: // Hela
		ctx.StartSideScheme = mtsNornStones
		shuffleIn(mtsSummonedBack)
		if has(mtsShawarma) {
			addDeck(mtsShawarma)
		}
		if has(mtsSystemShock) {
			addDeck(mtsSystemShock)
		}
		if st.Flags["stones"] {
			ctx.DiscardTopHalf = true
		}
	case 4: // Loki
		ctx.StartSideScheme = mtsDungeons
		shuffleIn(mtsSummonedBack)
		if has(mtsShawarma) {
			addDeck(mtsShawarma)
		}
		if has(mtsSystemShock) {
			addDeck(mtsSystemShock)
		}
		if st.Flags["stones"] {
			ctx.DiscardTopHalf = true
		}
		if has(mtsNornStone) {
			ctx.PoolUpgrades = append(ctx.PoolUpgrades, mtsNornStone)
		}
		if has(mtsSecurityBreach) {
			shuffleIn(mtsSecurityBreach)
		}
	}
	// Expert heal: place an acceleration token on the main scheme.
	// (Applied per player in BuildGame via HealNext.)
	if st.IsExpert() && st.Index >= 1 {
		for i := range st.Players {
			if st.Players[i].HealNext {
				ctx.MainSchemeAcceleration++
			}
		}
	}
}
