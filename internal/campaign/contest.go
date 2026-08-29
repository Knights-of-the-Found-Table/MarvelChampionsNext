package campaign

import (
	"fmt"
	"strings"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/data"
)

// The ten FFG Campaign Contest designs (five winners plus five more
// published designs from the same "New World Hydra" collection). Each has
// its own program file; this file carries the box definitions and the
// shared card-code tables they reference.

// Encounter set codes (marvelcdb card_set_code) referenced by the contest
// campaigns.
const (
	setMoE            = "masters_of_evil"
	setWreckingMod    = "wrecking_crew_modular"
	setUnderAttack    = "under_attack"
	setLegionsHydra   = "legions_of_hydra"
	setHydraAssault   = "hydra_assault"
	setHydraPatrol    = "hydra_patrol"
	setExperimental   = "exper_weapon"
	setTaskmasterMod  = "taskmaster"
	setWeaponMaster   = "weap_master"
	setDoomsday       = "the_doomsday_chair"
	setBrothersGrimm  = "brothers_grimm"
	setBeastyBoys     = "beasty_boys"
	setCrossfire      = "crossfire_crew"
	setSinisterSynd   = "sinister_syndicate"
	setMisterHyde     = "mister_hyde"
	setRansacked      = "ransacked_armory"
	setStreetsMayhem  = "streets_of_mayhem"
	setStateEmergency = "state_of_emergency"
	setDownToEarth    = "down_to_earth"
	setCityInChaos    = "city_in_chaos"
	setSinisterAsslt  = "sinister_assault"
	setOsbornTech     = "osborn_tech"
	setPersonalNight  = "personal_nightmare"
	setGoblinGimmicks = "goblin_gimmicks"
	setRunningIntf    = "running_interference"
	setPowerDrain     = "power_drain"
	setAMessThings    = "a_mess_of_things"
	setInheritors     = "the_inheritors"
	setWhispers       = "whispers_of_paranoia"
	setFrostGiants    = "frost_giants"
	setLegionsHel     = "legions_of_hel"
	setEnchantress    = "enchantress"
	setInfinityGaunt  = "infinity_gauntlet"
	setTowerDefense   = "tower_defense"
	setProjectWideaw  = "project_wideawake"
	setMystique       = "mystique"
	setZeroTolerance  = "zero_tolerance"
	setDystopian      = "dystopian_nightmare"
	setGenosha        = "genosha"
	setSavageLand     = "savage_land"
	setClanAkkaba     = "clan_akkaba"
	setEnSabahNur     = "en_sabah_nur"
	setFourHorsemen   = "four_horsemen"
	setSauron         = "sauron"
	setDeathstrike    = "deathstrike"
	setReavers        = "reavers"
	setZzzax          = "zzzax"
	setIronSpiderSix  = "ironspider_sinister"
	setHopeSummers    = "hope_summers"
	setBlackTom       = "black_tom_cassidy"
	setPowerStone     = "power_stone"
	setShipCommand    = "ship_command"
	setKreeFanatic    = "kree_fanatic"
	setGalacticArt    = "galactic_artifacts"
	setBandBadoon     = "band_of_badoon"
	setBrotherhoodBd  = "brotherhood_of_badoon"
	setKreeMilitant   = "kree_militant"
	setMenagerie      = "menagerie_medley"
	setArmadillo      = "armadillo"
	setTemporal       = "temporal"
	setAnachronauts   = "anachronauts"
	setMojoSitcom     = "sitcom"
	setMojoFantasy    = "fantasy"
	setMojoScifi      = "sci-fi"
	setMojoHorror     = "horror"
	setMojoWestern    = "western"
	setMojoSpiral     = "spiral"
	setMojoCore       = "mojo"
	setMagog          = "magog"
	setHellfire       = "hellfire"
	setGuerrillaTac   = "guerrilla_tactics"
)

// Single card codes used by the contest campaigns.
const (
	quinjet        = "03019"  // Quinjet (Captain America hero pack)
	counterattack  = "08030"  // Counterattack (Black Widow hero pack)
	anticipation   = "25035"  // Anticipation (Valkyrie hero pack)
	interrogation  = "01063"  // Interrogation Room (Core)
	honoraryGuard  = "22035"  // Honorary Guardian (Nebula hero pack)
	helicarrier    = "01092"  // Helicarrier (Core)
	xjetMG         = "32020"  // The X-Jet (Mutant Genesis basic)
	utopiaCyclops  = "33020"  // Utopia (Cyclops hero pack)
	utopiaStorm    = "36024"  // Utopia (Storm hero pack)
	findLostMutant = "45169a" // Find Lost Mutants (AoA mission)
	longshotAlly   = "39071"  // Longshot (MojoMania)
	magnetoSkill   = "32172b" // Magneto ally (data gap; logged no-op)
	metroPD        = "32171b" // Metro PD support (data gap; logged no-op)
	rogueVessel    = "16143"  // Rogue Vessel environment (GMW)
	shawarmaCard   = "21183"  // Shawarma (MTS)
	godslayer      = "18018"  // Godslayer (Gamora hero pack)
	jarnbjorn      = "06019"  // Jarnbjorn (Thor hero pack)
	sorcerSupreme  = "09026"  // The Sorcerer Supreme (Doctor Strange)
	lockAndLoad    = "40019"  // Lock and Load (NeXt Evolution)
	laserCannon    = "04158"  // Laser Cannon (RRS campaign TECH)
	deadpoolAlly   = "40024"  // Deadpool (NeXt Evolution ally)
	systemShock    = "21185"  // System Shock obligation (MTS)
	saveShawarma   = "21182a" // Save the Shawarma Place (MTS)
	unnaturalStorm = "21159"  // Unnatural Storm (MTS side scheme)
	captByHydra    = "04107"  // Captured by Hydra (RRS side scheme)
	modokMinion    = "01184"  // M.O.D.O.K. (Core Doomsday Chair)
	trexMinion     = "11032"  // Tyrannosaurus Rex (Kang)
	childrenAtom   = "49037"  // Children of the Atom (Magneto)
	vibraniumArmor = "01152"  // Vibranium Armor (Core)
	concBlasters   = "01153"  // Concussion Blasters (Core)
	sleepSide      = "04130"  // The Sleeper (RRS)
	zolasAlgorithm = "04163"  // Zola's Algorithm obligation (RRS)
	corruptGuard   = "07008"  // Corrupt Prison Guard (Wrecking Crew)
	deathstrikeMn  = "35034"  // Lady Deathstrike minion (Wolverine)
	scorpionGoblin = "02038"  // Scorpion minion (Green Goblin)
	docOckMinion   = "27158"  // Doctor Octopus (Sinister Assault)
	scorpionSM     = "27162"  // Scorpion (Sinister Assault)
	cardSharkMn    = "39068"  // Card Shark minion (MojoMania)
	roboticEnhance = "27110"  // Robotic Enhancements treachery (SM)
	bioServant     = "04114"  // Ultimate Bio-Servant minion (RRS)
	sauronMn       = "46029"  // Sauron minion (Iceman)
	sauronLives    = "46030"  // Sauron Lives! side scheme (Iceman)
	gameOfMojos    = "39041"  // A Game of Mojo's environment
	mojoRunner     = "39053"  // Mojo Runner environment
	mojoFiles      = "39047"  // The Mojo Files environment
	criminalEnter  = "02006a" // Criminal Enterprise environment (Goblin)
	mojoMiddle     = "39060"  // Mojo in the Middle environment
	monarchEgg     = "16127"  // Hujahdarian Monarch Egg side scheme
	magicalTeapot  = "16128"  // Magical Teapot side scheme
	philStone      = "16129"  // Philosopher's Stone side scheme
	crystalBall    = "16130"  // Crystal Ball side scheme
	radiactiveMan  = "01129"  // Radioactive Man (Core MoE minion)
	madameHydra    = "01181"  // Madame Hydra (Core)
	mystiqueMG     = "32080"  // Mystique (Mutant Genesis)
	nebulaAlly     = "16094"  // Nebula ally (GMW; data gap → logged no-op)
)

// Black Swan has no minion card in the data snapshot; the pool marker
// keeps the log faithful while her setup effect degrades to a no-op
// (same convention as the MTS campaign).
const blackSwanMarker = "pool-black-swan"

// Hero base codes referenced by contest campaign identity requirements.
const (
	heroIronMan   = "01029"
	heroSpiderman = "01001" // Spider-Man (Peter Parker); 27030 also Peter
	heroPeterSM   = "27030"
	heroRogue     = "38001"
	heroCap       = "03001"
	heroSheHulk   = "01019"
)

// rrsRescuedCaptives lists every Taskmaster captive ally in the data
// snapshot (three of the ten printed captives).
var rrsRescuedCaptives = []string{"04097", "04098", "04099"}

// moeMinions lists the Masters of Evil minions in the data snapshot
// (Crimson Cowl's caught/escaped tracking).
var moeMinions = []string{"01129", "01130", "01131", "01132"}

// wreckers lists the four Wrecking Crew villain base codes (Awesome
// Campaign scenario 3 "last villain defeated").
var wreckers = []string{"07002", "07017", "07032", "07046"}

// bordOrderMinions lists the four Black Order minions in the data
// snapshot; Black Swan is the marker above.
var bordOrderMinions = []string{"21085", "21086", "21125", "21126"}

// whatifTraits lists the What If...? trait options (display order).
var whatifTraits = []string{
	"asgard", "avenger", "champion", "guardian", "mystic", "shield", "webwarrior",
}

// whatifTraitSets maps each trait to its modular-set options (the
// recommended picks are first; What If...? rulebook p.8).
var whatifTraitSets = map[string][]string{
	"asgard":     {setEnchantress, setFrostGiants, setLegionsHel, setLegionsHydra, setStateEmergency, setStreetsMayhem},
	"avenger":    {setCrossfire, setMoE, setStateEmergency, setStreetsMayhem, setTowerDefense, setDoomsday},
	"champion":   {setArmadillo, setUnderAttack, setCityInChaos, setDownToEarth, setSinisterAsslt, setZzzax},
	"guardian":   {setAnachronauts, setTowerDefense, setBandBadoon, setKreeMilitant, setMenagerie, setPowerStone},
	"mystic":     {setAnachronauts, setBrothersGrimm, setTemporal, setMisterHyde, setPersonalNight, setWhispers},
	"shield":     {setExperimental, setHydraAssault, setHydraPatrol, setLegionsHydra, setRansacked, setWeaponMaster},
	"webwarrior": {setAMessThings, setGoblinGimmicks, setPowerDrain, setRunningIntf, setSinisterAsslt, setSinisterSynd, setInheritors, setIronSpiderSix},
}

// mojoRoleTable describes a House of Mojo role.
type mojoRoleTable struct {
	Condition string `json:"condition"` // RRS Basic X upgrade (basic side)
	Market    string `json:"market"`    // GMW campaign event (optional pack)
	Skill     string `json:"skill"`     // Mutant Genesis Skill upgrade
}

// mojoRoles maps the House of Mojo roles to their three cards.
var mojoRoles = map[string]mojoRoleTable{
	"fighter":   {"04160a", "16156", "32179"}, // Attack / Grapple / Ferocious Attack
	"tactician": {"04159a", "16157", "32187"}, // Thwart / Wing It / Surprise!
	"protector": {"04162a", "16154", "32178"}, // Defense / Calculate the Odds / Brazen Defense
	"tank":      {"04161a", "16155", "32180"}, // Recovery / Creative Solution / War Cry
}

// awesomeReward is one Guardian Influence shop entry (Awesome p.9).
type awesomeReward struct {
	Code string `json:"code"`
	Cost int    `json:"cost"`
}

// awesomeRewards lists the purchasable cards with their influence costs.
// Codes missing from the data snapshot are filtered out of the shop.
var awesomeRewards = []awesomeReward{
	{"16142", 5},  // Milano (support)
	{"16162", 5},  // Armor Plating
	{"16163", 2},  // Heavy Cannon
	{"16164", 2},  // Hyper Thrusters
	{"16165", 2},  // Reactor Core
	{"16170", 3},  // Cargo Hold
	{"16171", 3},  // Mounted Laser
	{"16172", 3},  // Navigation Column
	{"16173", 3},  // Targeting Screen
	{"04159a", 3}, // Basic Thwart Upgrade
	{"04160a", 3}, // Basic Attack Upgrade
	{"04161a", 3}, // Basic Defense Upgrade
	{"04162a", 3}, // Basic Recovery Upgrade
	{"27182a", 3}, // Compact Darts
	{"27183a", 3}, // Impact-Dampening Suit
	{"27184a", 3}, // Laser Goggles
	{"27185a", 3}, // Propulsion Gauntlet
	{"27186a", 3}, // Retinal Display
	{"27187a", 3}, // Shock Knuckles
	{"27188a", 3}, // Wave Bracers
	{"27189a", 3}, // Wrist Navigator
	{"04155", 4},  // Adrenaline Stims
	{"04156", 4},  // Tactical Scanner
	{"04157", 4},  // Emergency Teleporter
	{"04158", 4},  // Laser Cannon
	{"21187a", 4}, // Norn Stone
	{"16150", 1},  // Brainstorm
	{"16151", 1},  // By Any Means
	{"16152", 1},  // Contingency Plan
	{"16153", 1},  // In Defiance
	{"16169", 1},  // Sure Gamble
	{"16154", 2},  // Calculate the Odds
	{"16155", 2},  // Creative Solution
	{"16156", 2},  // Grapple
	{"16157", 2},  // Wing It
	{"16158", 2},  // Close Call
	{"16159", 2},  // Defy Danger
	{"16160", 2},  // In Harm's Way
	{"16161", 2},  // Take the Fight to Them
	{"16174", 3},  // Grand Strategy
	{"16175", 3},  // Power Unleashed
	{"16176", 3},  // Tried and True
	{"16177", 3},  // Triple Threat
	{"16166", 4},  // Ardent Resolve
	{"16167", 4},  // Onrush
	{"16168", 4},  // Safeguard
	{"21190", 3},  // Lady Sif
	{"21191", 3},  // Fandral
	{"21193", 3},  // Volstagg
}

// Awesome Campaign reward cards that are not in the data snapshot (RRS
// allies Moon Knight / Shang-Chi / White Tiger / Elektra non-captive
// printings); kept here so the log documents the gap.
var awesomeRewardGaps = []string{"04197", "04198", "04199", "04200", "21192"}

// nightPoolAdds lists the Deadpool's Game Night reward-pool additions by
// scenario and outcome (Deadpool pack codes).
var (
	nightPoolStart = []string{"44058", "44053"} // War, Blackout
	nightCh1Win    = []string{"44030", "44055"} // Stick-To-Itiveness, Laser Swords
	nightCh1Lose   = []string{"44028", "44043"} // Git Gud, Bob, Agent of Hydra
	nightCh2Add    = []string{"44057"}          // Tic-Tac-Toe
	nightCh2Picks  = []string{"44018", "44020", "44022", "44047", "44050"}
	nightCh3Picks  = []string{"44019", "44023", "44025", "44026", "44027"}
	nightCh4Picks  = []string{"44013", "44014", "44015", "44016", "44045"}
)

// bordPaths lists the Revenge of the Black Order narrative paths with
// their identity-trait requirements (display only; the path is a group
// decision recorded at campaign start).
var bordPaths = []struct {
	Key   string   `json:"key"`
	Label string   `json:"label"`
	Trait string   `json:"trait"`
	Hero  []string `json:"heroes"`
}{
	{"first", "First Response", "attorney/civilian/genius/mystic/scientist", nil},
	{"specops", "Spec Ops", "android/king/mercenary/s.h.i.e.l.d./soldier/spy/wakanda", nil},
	{"assault", "Direct Assault", "asgard/inhuman/mutant/outlaw", nil},
}

// lookupCard is a small helper for the HeroOK validators.
func lookupCard(code string) (*data.CardDef, bool) {
	return engine.DB.Lookup(code)
}

func init() {
	for key, box := range contestBoxes() {
		Boxes[key] = box
	}
}

// requireTraitHero builds a HeroOK that demands the given trait on the
// hero form (Crimson Cowl: AVENGER).
func requireTraitHero(trait, msg string) func(string) error {
	return func(heroBase string) error {
		for _, side := range []string{"a", "b"} {
			if def, ok := lookupCard(heroBase + side); ok && def.HasTrait(trait) {
				return nil
			}
		}
		return fmt.Errorf("%s", msg)
	}
}

// forbidGuardianHero enforces the Awesome Campaign's no-Guardian rule.
func forbidGuardianHero(heroBase string) error {
	for _, side := range []string{"a", "b"} {
		if def, ok := lookupCard(heroBase + side); ok && def.HasTrait("guardian") {
			return fmt.Errorf("the Awesome Campaign forbids GUARDIAN heroes")
		}
	}
	return nil
}

// contestBoxes builds the ten campaign box definitions.
func contestBoxes() map[string]*BoxDef {
	return map[string]*BoxDef{
		"cowl": {
			Key:  "cowl",
			Name: "The Crimson Cowl Conspiracy",
			Desc: "Contest design by Kurt Hake. The Masters of Evil regroup under a new Crimson Cowl: five chapters across the Core, Red Skull and Hood sets, with recorded S.H.I.E.L.D. Tech, escaped minions and an Intel Level that unlocks the team's gear.",
			Scenarios: []ScenarioRef{
				{ID: "04061", Name: "Crossbones — Attack on Mount Athena"},
				{ID: "01116", Name: "Klaw — Underground Distribution"},
				{ID: "04079", Name: "Absorbing Man — None Shall Pass"},
				{ID: "07001", Name: "Wrecking Crew — Breakout"},
				{ID: "01137", Name: "Ultron — The Crimson Cowl"},
			},
			HeroOK: requireTraitHero("avenger", "the Crimson Cowl Conspiracy needs AVENGER heroes"),
		},
		"whatif": {
			Key:  "whatif",
			Name: "What If...?",
			Desc: "Contest design by Amanda Shagoury. Reality is rewritten across five chapters: record a What If...? trait, rescue trait allies, and pick trait-driven modular sets before the Hood's crime spree ends under Ultron with the Infinity Gauntlet.",
			Scenarios: []ScenarioRef{
				{ID: "02004", Name: "Green Goblin — Risky Business"},
				{ID: "21098", Name: "Corvus & Proxima — Defend the Tower"},
				{ID: "04096", Name: "Taskmaster — Coworker Confinement"},
				{ID: "24004", Name: "The Hood — Leader of Hydra and Many Others"},
				{ID: "01137", Name: "Ultron — Infinity Assembled"},
			},
		},
		"awesome": {
			Key:  "awesome",
			Name: "Awesome Campaign Vol. 1",
			Desc: "Contest design by Steele Hull. Five chapters — Ronan, Absorbing Man, the Wrecking Crew, Red Skull and Thanos — earn Guardian Influence that buys campaign cards before the finale. GUARDIAN heroes are not allowed.",
			Scenarios: []ScenarioRef{
				{ID: "16106", Name: "Ronan — Moonage Daydream"},
				{ID: "04079", Name: "Absorbing Man — Ain't No Mountain High Enough"},
				{ID: "07001", Name: "Wrecking Crew — Escape"},
				{ID: "04128", Name: "Red Skull — Cherry Bomb"},
				{ID: "21114", Name: "Thanos — Spirit in the Sky"},
			},
			HeroOK: forbidGuardianHero,
		},
		"alias": {
			Key:  "alias",
			Name: "Alias Investigations",
			Desc: "Contest winners design by Christian Fecteau. Jessica Jones hunts a kidnapper across three chapters with a persistent Clue Deck; the clue drawn in chapter two selects the final villain.",
			Scenarios: []ScenarioRef{
				{ID: "01097", Name: "Rhino — Fight at the Facility"},
				{ID: "32087", Name: "Sentinel — Brawl at the Mall"},
				{ID: "04096", Name: "The Final Battle — Taskmaster"},
				{ID: "04112", Name: "The Final Battle — Dr. Zola"},
				{ID: "32112", Name: "The Final Battle — Master Mold"},
			},
		},
		"watchers": {
			Key:  "watchers",
			Name: "The Watcher's Team: What If...?",
			Desc: "Contest winners design by Zach Goscha. The Watcher assembles a team across realities: every chapter demands a specific identity (Iron Man, Spider-Man, Rogue, Captain America), so players rebuild their deck between games — use the deck-swap button in the interlude.",
			Scenarios: []ScenarioRef{
				{ID: "21074", Name: "Ebony Maw — Trapped in Time", Requires: "Iron Man"},
				{ID: "40166", Name: "Stryfe — The Sorcerer Supreme", Requires: "Spider-Man (Peter Parker)"},
				{ID: "21138", Name: "Hela — Rogue and the Power of Thor", Requires: "Rogue"},
				{ID: "24004", Name: "The Hood — Vigilante Justice", Requires: "Captain America"},
				{ID: "01137", Name: "Ultron — The Watcher's Team", Requires: "an earlier identity"},
			},
		},
		"mojo": {
			Key:  "mojo",
			Name: "House of Mojo",
			Desc: "Contest winners design by Amanda Shagoury. A Mojo-produced MCU show: pick a role with its own upgrade track, attempt a training player side scheme and survive En Sabah Nur, Magneto, Thanos, the Sentinels and Mojo himself.",
			Scenarios: []ScenarioRef{
				{ID: "45147", Name: "En Sabah Nur — Training for the End of the World"},
				{ID: "32141", Name: "Magneto — Wanda's Vision"},
				{ID: "21114", Name: "Thanos — Balance Will Be Restored"},
				{ID: "32087", Name: "Sentinel — Saving Grace"},
				{ID: "39025", Name: "Mojo — Mojo All Along"},
			},
		},
		"bord": {
			Key:  "bord",
			Name: "Revenge of the Black Order",
			Desc: "Contest winners design by Karl Resch. A branching narrative: the group picks one of three paths (First Response, Spec Ops, Direct Assault), each with its own opening chapter, encounter deck and setups through Drang and Ebony Maw.",
			Scenarios: []ScenarioRef{
				{ID: "27064", Name: "Venom — Krazy Klyntar (First Response)"},
				{ID: "21098", Name: "Corvus & Proxima — Defend It and Hope (Spec Ops)"},
				{ID: "39002", Name: "MaGog — Who Will Fight Me? (Direct Assault)"},
				{ID: "16057", Name: "Drang — Badoon Bombardment"},
				{ID: "21074", Name: "Ebony Maw — Mysteries and Magic"},
			},
		},
		"night": {
			Key:  "night",
			Name: "She-Hulk vs. Deadpool's Game Night",
			Desc: "Contest winners design by Kurt Hake. One player must be She-Hulk. Deadpool hijacks five game nights with a shared 'Pool reward pool, metagame challenges and a Strength of the Alliance track — losses change the finale instead of replaying the chapter.",
			Scenarios: []ScenarioRef{
				{ID: "04128", Name: "Red Skull — Game Night Begins"},
				{ID: "45085", Name: "Horsemen — The Lord of the Wings"},
				{ID: "02004", Name: "Norman Osborn — Android Web-Hacker"},
				{ID: "45147", Name: "En Sabah Nur — Akkaba Horror"},
				{ID: "16082", Name: "Collector — The Game Collection"},
			},
		},
		"viral": {
			Key:  "viral",
			Name: "Going Viral",
			Desc: "Contest design by Henry Borkgren. Ultron's virus spreads through Marvel's cyborgs: rescue captive allies to fill Pym's Antivirus track while the Ultron Infection track punishes every threat left behind. Two of the three second chapters are played.",
			Scenarios: []ScenarioRef{
				{ID: "04096", Name: "Taskmaster — Scenario 1"},
				{ID: "04112", Name: "Zola — Scenario 2A"},
				{ID: "27100", Name: "The Sinister Six — Scenario 2B"},
				{ID: "16091", Name: "Nebula — Scenario 2C"},
				{ID: "01137", Name: "Ultron — Scenario 3"},
			},
		},
		"entropy": {
			Key:  "entropy",
			Name: "Entropic Ascension",
			Desc: "Contest design by Karl Resch. A group reputation track turns every win into upgrades and recurring setup curses, and the recorded Crime Wave lines assemble The Hood's finale from up to seven modular sets.",
			Scenarios: []ScenarioRef{
				{ID: "04096", Name: "Taskmaster — Scenario 1"},
				{ID: "01097", Name: "Rhino — Scenario 2A"},
				{ID: "27087", Name: "Mysterio — Scenario 2B"},
				{ID: "01116", Name: "Klaw — Scenario 3"},
				{ID: "24004", Name: "The Hood — The Final Battle"},
			},
		},
	}
}

// addRoleUpgrade appends a per-player setup upgrade (initializing the
// map on first use).
func addRoleUpgrade(ctx *engine.CampaignSetup, i int, code string) {
	if ctx.RoleUpgrades == nil {
		ctx.RoleUpgrades = map[int][]string{}
	}
	ctx.RoleUpgrades[i] = append(ctx.RoleUpgrades[i], code)
}

// deref reads an optional int field (card costs, printed scheme values).
func deref(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

// SMAllTech lists the Sinister Motives S.H.I.E.L.D. Tech upgrade pool
// (shared by the SM campaign and the contest designs that reuse it).
func SMAllTech() []string { return smTech }

// SMCommunitySchemes lists the Campaign – Community Service side schemes.
func SMCommunitySchemes() []string { return smCommunity }

// WhatIfTraits lists the What If...? trait options.
func WhatIfTraits() []string { return whatifTraits }

// EntSoeOptions lists the State of Emergency side schemes.
func EntSoeOptions() []string { return entSoeOptions }

// ViralNextOptions lists the Scenario #2 chapters still unplayed.
func ViralNextOptions(st *State) []string {
	var out []string
	played := st.Selections["viralPlayed"]
	for _, idx := range []string{"1", "2", "3"} {
		if !strings.Contains(played, idx) {
			out = append(out, idx)
		}
	}
	return out
}

// BoxTables serves the per-box choice tables the UI renders (role cards,
// influence shop, path labels...). Contents vary by campaign.
func BoxTables(st *State) map[string]any {
	tables := map[string]any{}
	switch st.Box {
	case "mojo":
		tables["mojoRoles"] = mojoRoles
		tables["nxAll"] = []string{"40190a", "40191a", "40192a", "40193a", "40194a", "40195a"}
	case "awesome":
		tables["awRewards"] = AwesomeRewards()
	case "bord":
		tables["paths"] = bordPaths
	case "night":
		tables["nightPicks"] = map[string][]string{
			"1": nightCh2Picks, "2": nightCh3Picks, "3": nightCh4Picks,
		}
	case "entropy":
		tables["enPaths"] = map[string][]string{
			"1": {"en1a", "en1b"},
			"2": {"en2a", "en2b"},
			"3": {"en3a", "en3b"},
		}
	case "whatif":
		tables["wiSets"] = whatifTraitSets
	case "watchers":
		tables["requires"] = watchersRequiredNames()
	}
	return tables
}

// watchersRequiredNames renders the required identity of each chapter.
func watchersRequiredNames() []string {
	var out []string
	for _, req := range watchersRequired {
		names := make([]string, 0, len(req))
		for _, base := range req {
			if def, ok := engine.DB.Lookup(base + "a"); ok {
				names = append(names, def.EName)
			} else {
				names = append(names, base)
			}
		}
		out = append(out, strings.Join(names, " / "))
	}
	return out
}
