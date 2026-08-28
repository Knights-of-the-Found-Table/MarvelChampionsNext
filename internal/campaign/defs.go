// Package campaign implements the meta-campaign layer for The Rise of Red
// Skull and Galaxy's Most Wanted: the scenario chains, the campaign log,
// the between-scenario interludes (upgrade choices, the Market) and the
// per-scenario setup/victory programs from the campaign rulebooks. The
// engine only sees a CampaignSetup struct per game; all cross-game state
// lives here and in the store.
package campaign

// BoxDef describes one campaign box: its scenario chain in play order.
type BoxDef struct {
	Key       string        `json:"key"`
	Name      string        `json:"name"`
	Desc      string        `json:"desc"`
	Scenarios []ScenarioRef `json:"scenarios"`
}

// ScenarioRef names one chapter (engine scenario id + display name).
type ScenarioRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Boxes lists the implemented campaigns. Scenario ids are the engine
// scenario registry ids (first main scheme stage codes).
var Boxes = map[string]*BoxDef{
	"rrs": {
		Key:  "rrs",
		Name: "The Rise of Red Skull",
		Desc: "Hydra strikes Project P.E.G.A.S.U.S. and hunts the Infinity Gem. Five scenarios, fixed identities, Hydra Campaign upgrades and expert persistent damage.",
		Scenarios: []ScenarioRef{
			{ID: "04061", Name: "Crossbones — Attack on Mount Athena"},
			{ID: "04079", Name: "Absorbing Man — None Shall Pass"},
			{ID: "04096", Name: "Taskmaster — Hunting Down Heroes"},
			{ID: "04112", Name: "Zola — The Island of Dr. Zola"},
			{ID: "04128", Name: "Red Skull — The Rise of Red Skull"},
		},
	},
	"aos": {
		Key:  "aos",
		Name: "Agents of S.H.I.E.L.D.",
		Desc: "A mole hides on the S.H.I.E.L.D. Executive Board. Uncover board members' secrets, eliminate evidence combinations and accuse the traitor before Zemo turns the board.",
		Scenarios: []ScenarioRef{
			{ID: "50067a", Name: "Black Widow — The Widow's Web"},
			{ID: "50087a", Name: "Batroc — Infiltrate A.I.M. Island Embassy"},
			{ID: "50104a", Name: "M.O.D.O.K. — Upgrading Adaptoids"},
			{ID: "50130a", Name: "Citizen V — Apprehending Rogue Agents"},
			{ID: "50167a", Name: "Baron Zemo — Zemo's Manipulations"},
		},
	},
	"aoa": {
		Key:  "aoa",
		Name: "Age of Apocalypse",
		Desc: "A dystopian Age of Apocalypse ruled by En Sabah Nur. Allies run side missions against Overseers while X-Force fights through five chapters.",
		Scenarios: []ScenarioRef{
			{ID: "45062", Name: "Unus — Hunting Gene Traitors"},
			{ID: "45085", Name: "The Horsemen of Apocalypse"},
			{ID: "45103", Name: "Apocalypse — The Age of Apocalypse"},
			{ID: "45121", Name: "Dark Beast — Bogus Journey"},
			{ID: "45147", Name: "En Sabah Nur — The Rise of Apocalypse"},
		},
	},
	"nx": {
		Key:  "nx",
		Name: "NeXt Evolution",
		Desc: "Stryfe and Mister Sinister hunt Hope Summers. Pick one campaign player side scheme per chapter — its environment reward persists, its encounter card haunts you.",
		Scenarios: []ScenarioRef{
			{ID: "40077", Name: "Marauders — Mutant Massacre"},
			{ID: "40103", Name: "Marauders — On the Run"},
			{ID: "40121", Name: "Juggernaut — The Unstoppable Juggernaut"},
			{ID: "40139", Name: "Mister Sinister — Sinister Intent"},
			{ID: "40166", Name: "Stryfe — Uncontrollable Power"},
		},
	},
	"mg": {
		Key:  "mg",
		Name: "Mutant Genesis",
		Desc: "The Brotherhood and the Sentinel program strike the X-Men. Choose a team role, earn one-shot Skill upgrades and chase the Future Past deck through time.",
		Scenarios: []ScenarioRef{
			{ID: "32063", Name: "Sabretooth — Stalked by Sabretooth"},
			{ID: "32087", Name: "Sentinels — Night of the Sentinels"},
			{ID: "32112", Name: "Master Mold — The Sentinel Factory"},
			{ID: "32125", Name: "Brotherhood — The Brotherhood Strikes"},
			{ID: "32141", Name: "Magneto — Asteroid M"},
		},
	},
	"sm": {
		Key:  "sm",
		Name: "Sinister Motives",
		Desc: "Sandman, Venom, Mysterio and the Sinister Six overrun New York. The group's reputation unlocks boons and curses on the reputation track.",
		Scenarios: []ScenarioRef{
			{ID: "27064", Name: "Sandman — Hapless Pedestrians"},
			{ID: "27076", Name: "Venom — Leave Us Alone!"},
			{ID: "27087", Name: "Mysterio — Maze of Mirrors"},
			{ID: "27100", Name: "The Sinister Six"},
			{ID: "27116", Name: "Venom Goblin — Skies Over New York"},
		},
	},
	"mts": {
		Key:  "mts",
		Name: "The Mad Titan's Shadow",
		Desc: "Ebony Maw hunts Knowhere while Thanos gathers the Infinity Stones. A shared campaign pool carries rewards and consequences between scenarios.",
		Scenarios: []ScenarioRef{
			{ID: "21074", Name: "Ebony Maw — Attack on Knowhere"},
			{ID: "21098", Name: "Sanctuary II — Under Siege"},
			{ID: "21114", Name: "Thanos — The Infinity Stones"},
			{ID: "21138", Name: "Hela — Odin's Torment"},
			{ID: "21165", Name: "Loki — All Hail King Loki"},
		},
	},
	"gmw": {
		Key:  "gmw",
		Name: "Galaxy's Most Wanted",
		Desc: "The Guardians clash with the Brotherhood of Badoon, raid the Collector's museum and face Ronan. Units buy upgrades from The Market between scenarios.",
		Scenarios: []ScenarioRef{
			{ID: "16057", Name: "Drang — Planetary Invasion"},
			{ID: "16073", Name: "Collector — Infiltrate the Museum"},
			{ID: "16082", Name: "Collector — The Missing Milano"},
			{ID: "16091", Name: "Nebula — The Art of Evasion"},
			{ID: "16106", Name: "Ronan the Accuser — Under Accusation"},
		},
	},
}

// RRS card groups.
var (
	rrsTech         = []string{"04155", "04156", "04157", "04158"}     // Hydra Campaign TECH upgrades
	rrsCond         = []string{"04159a", "04160a", "04161a", "04162a"} // Basic Condition upgrades
	rrsExperimental = []string{"04072", "04073", "04074", "04075"}     // Experimental Weapons
	rrsRescued      = []string{"04097", "04098", "04099"}              // Taskmaster captives
	rrsObligations  = []string{"04163", "04164", "04165", "04166"}     // Expert Campaign obligations
)

// Single-card codes.
const (
	rrsHydraPrison = "04122" // Hydra Prison side scheme (Zola)

	gmwMarketSet    = "the_market" // Campaign – The Market set code
	gmwHeadhunter   = "16183"      // Badoon Headhunter minion
	gmwOnTheHunt    = "16184"      // treachery, marks >= 1
	gmwDeadToRights = "16185"      // treachery, marks >= 2
	gmwHenchman     = "16186"      // minion, marks >= 3
	gmwFugitive     = "16187"      // Fugitive Recovery side scheme
	gmwShipEnv      = "16093"      // Nebula's Ship environment
	gmwPowerStone   = "16126"      // Vandarian Power Stone attachment
	gmwAccused      = "16116"      // "You Stand Accused!" treachery
	gmwPincer       = "16112"      // Pincer Maneuver side scheme
	gmwBlitz        = "16178a"     // Badoon Blitz (scenario 1)
	gmwGallery      = "16179a"     // Gallery of Splendor (scenario 2)
	gmwNoEscape     = "16180a"     // "There is No Escape" (scenario 3)
	gmwGuerrilla    = "16181a"     // Guerrilla Tactics (scenario 4)
	gmwSupremacy    = "16182a"     // Kree Supremacy (scenario 5)
)

// galacticArtifacts side schemes shuffled back in at scenario 4, with
// their printed setup riders (rulebook p.14).
var galacticArtifacts = []string{"16127", "16128", "16129", "16130"}
