package rules

import "github.com/philoserf/ctchargen/internal/traveller"

// Every cell is typed as the page prints it: its glyphs, its abbreviations,
// its notation and its dashes. The layouts are the page's where its text
// carries them, as the Aging Table's spans and the DM rows' headings do; where
// only position does, which table a grid is, which service a column belongs
// to and which ship a hull is are keyed by traveller's own types (ADR 0005).
// The service tables (pp. 9, 10 and 11) keep the page's rows as rows: a line
// of their literals is a row the page prints, the services across, so the
// second transcription, which walks each service down its column, reads them
// the other way. The education gate is not a cell: p. 11 says it in prose,
// "education 8 or greater", and printedGate names the characteristic as p. 4
// does.

// skillsTranscription is the Acquired Skills Table as p. 11 prints it, with
// the eligibility and the education gate that govern it.
//
// Which table a grid is, and which service a column belongs to, are positions
// in traveller's own order: the tables in traveller.SkillTables order and the
// services in traveller.ServiceNames order. A cell is the page's text: its
// abbreviations, its notation and its dashes, read off the page one page at
// a time (#339).
type skillsTranscription struct {
	tables        [len(traveller.SkillTables)][Faces][len(traveller.ServiceNames)]string
	eligibility   Eligibility
	educationGate printedGate

	// normalizations is E012's, from what a table prints to the heading
	// its description carries.
	normalizations map[string]string
}

// printedGate is the education gate: the table it opens, and the
// characteristic and threshold as the page prints them.
type printedGate struct {
	table                     traveller.SkillTable
	characteristic, threshold string
}

// transcribedSkills is Book 1 p. 11 (Acquired Skills Table); p. 6 (Basic Skill
// Eligibility); pp. 12-20 (the skill descriptions, whose headings every name
// normalizes to).
//
// Rows are transcribed as the page lays them out: one row of the table, six
// services across. E012 governs the names.
var transcribedSkills = skillsTranscription{
	tables: [len(traveller.SkillTables)][Faces][len(traveller.ServiceNames)]string{
		traveller.PersonalDevelopment: { // Personal Development Table
			{"+1 Stren", "+1 Stren", "+1 Stren", "+1 Stren", "+1 Stren", "+1 Stren"},
			{"+1 Dext", "+1 Dext", "+1 Dext", "+1 Dext", "+1 Dext", "+1 Dext"},
			{"+1 Endur", "+1 Endur", "+1 Endur", "+1 Endur", "+1 Endur", "+1 Endur"},
			{"+1 Social", "Gambling", "Gambling", "Gun Cbt", "+1 Stren", "Blade Cbt"},
			{"+1 Intel", "Brawling", "Brawling", "+1 Intel", "Blade Cbt", "Brawling"},
			{"+1 Educ", "Blade Cbt", "+1 Educ", "+1 Educ", "Bribery", "−1 Social"},
		},
		traveller.ServiceSkills: { // Service Skills Table
			{"Ship's Boat", "ATV", "ATV", "Air/Raft", "Steward", "Forgery"},
			{"Vacc Suit", "Vacc Suit", "Air/Raft", "Vacc Suit", "Vacc Suit", "Gambling"},
			{"Fwd Obsv", "Blade Cbt", "Fwd Obsv", "Navigation", "Blade Cbt", "Brawling"},
			{"Blade Cbt", "Blade Cbt", "Blade Cbt", "Mechanical", "Gun Cbt", "Blade Cbt"},
			{"Gun Cbt", "Gun Cbt", "Gun Cbt", "Electronic", "Electronic", "Gun Cbt"},
			{"Gunnery", "Gun Cbt", "Gun Cbt", "Jack-o-T", "Jack-o-T", "Bribery"},
		},
		traveller.AdvancedEducation: { // Advanced Education Table
			{"Vacc Suit", "ATV", "ATV", "Air/Raft", "Streetwise", "Streetwise"},
			{"Mechanical", "Mechanical", "Mechanical", "Mechanical", "Mechanical", "Mechanical"},
			{"Electronic", "Electronic", "Electronic", "Electronic", "Electronic", "Electronic"},
			{"Engnrng", "Tactics", "Tactics", "Jack-o-T", "Navigation", "Gambling"},
			{"Gunnery", "Blade Cbt", "Blade Cbt", "Gunnery", "Gunnery", "Brawling"},
			{"Jack-o-T", "Gun Cbt", "Gun Cbt", "Medical", "Medical", "Forgery"},
		},
		traveller.AdvancedEducationEight: { // Advanced Education Table (education 8+)
			{"Medical", "Medical", "Medical", "Medical", "Medical", "Medical"},
			{"Navigation", "Tactics", "Tactics", "Navigation", "Navigation", "Forgery"},
			{"Engnrng", "Tactics", "Tactics", "Engnrng", "Engnrng", "Electronics"},
			{"Computer", "Computer", "Computer", "Computer", "Computer", "Computer"},
			{"Pilot", "Leader", "Leader", "Pilot", "Pilot", "Streetwise"},
			{"Admin", "Admin", "Admin", "Jack-o-T", "Admin", "Jack-o-T"},
		},
	},

	// Book 1 p. 6, the Basic Skill Eligibility box
	eligibility: Eligibility{InitialTerm: 2, PerSubsequentTerm: 1, OnCommission: 1, OnPromotion: 1},

	// Book 1 p. 11: 'the four tables if the character is of education 8 or
	// greater'
	educationGate: printedGate{table: traveller.AdvancedEducationEight, characteristic: "Education", threshold: "8+"},

	// E012. Left is what a table prints, right is the heading its description
	// carries on pp. 12-20. Blade Cbt and Gun Cbt resolve to the two weapon
	// categories rather than to skill names: p. 11 requires the specific weapon
	// at once, and p. 25's own skill list records the weapon, not the category.
	normalizations: map[string]string{
		"Blade Cbt":   "Blade Combat",
		"Gun Cbt":     "Gun Combat",
		"Fwd Obsv":    "Forward Observer",
		"Engnrng":     "Engineer",
		"Jack-o-T":    "Jack of all Trades",
		"Admin":       "Administration",
		"Electronics": "Electronic",
		"Rifl3":       "Rifle",
		"SMG":         "Submachine Gun",
	},
}

// perService is one printed row of a service table, the services across in
// traveller.ServiceNames order.
type perService [len(traveller.ServiceNames)]string

// printedThrows is one throw's rows of the Prior Service Table as p. 10
// prints them: a target per service, and under it each DM row, headed by its
// amount ("DM of +1 if") and giving a condition per service.
type printedThrows struct {
	targets perService
	dms     []printedDMRow
}

// printedDMRow is one DM row: its heading as printed, and a condition, or a
// dash, per service.
type printedDMRow struct {
	heading    string
	conditions perService
}

// printedGrant is one entry of the Rank and Service Skills box as p. 23
// prints it: its label ("Navy Captain", "Marine") and its grant ("+1 Social",
// "Cutlass-1").
type printedGrant struct {
	label, grant string
}

// printedPension is one row of the Annual Retirement Pay table (p. 21).
type printedPension struct {
	terms int
	pay   int64
}

// rankRows is the rows the Table of Ranks prints, Rank 1 to Rank 6 (p. 10).
const rankRows = 6

// servicesTranscription is the Prior Service Table and the Table of Ranks as
// p. 10 prints them, the services across in traveller.ServiceNames order,
// with the Rank and Service Skills box and the retirement pay table.
type servicesTranscription struct {
	enlistment, survival  printedThrows
	commission, promotion printedThrows
	draft                 [len(traveller.ServiceNames)]int
	reenlist              perService
	ranks                 [rankRows]perService
	grants                []printedGrant
	pensions              []printedPension
	perAdditionalTerm     int64
	pensionPaidBy         []traveller.ServiceName
}

// musteringTranscription is the two Mustering Out Tables as p. 9 prints
// them, the services across, with the two DMs, the benefits' names, and the
// passage prices and resale rate.
type musteringTranscription struct {
	table1               [MusterRows][len(traveller.ServiceNames)]string
	table2               [MusterRows][len(traveller.ServiceNames)]int64
	table1DMFromRank5or6 int
	table2DMFromGambling int
	table2ModifierFrom   string
	names                map[string]string
	prices               [len(traveller.PassageClasses)]int64
	resalePercent        int
}

// printedSpan is one cell of an Aging Table row as p. 9 prints it: its text,
// and the term columns its dotted leaders run across, from and through.
type printedSpan struct {
	text          string
	from, through int
}

// printedAgingRow is one characteristic's row of the Aging Table: its label
// as printed, and its cells left to right.
type printedAgingRow struct {
	label string
	cells []printedSpan
}

// agingTranscription is the Aging Table (p. 9), its two header rows and its
// six rows, and the medical crisis (pp. 7-8).
type agingTranscription struct {
	terms, ages                        []string
	rows                               []printedAgingRow
	crisisSaving                       string
	crisisRecoversTo, crisisMonthsDice int
}

// printedTitle is one rank of nobility and the Social Standing that confers
// it.
type printedTitle struct {
	socialStanding int
	title          string
}

// transcribedServices is Book 1 p. 10 (Prior Service Table, Table of Ranks); p.
// 21 (Annual Retirement Pay); p. 23 (Rank and Service Skills).
//
// Transcribed from a visual read of the page. Never from extracted text: the
// reprint's font turns a printed minus into a digit and an em dash into a 4.
var transcribedServices = servicesTranscription{
	enlistment: printedThrows{
		targets: perService{"8+", "9+", "5+", "7+", "7+", "3+"},
		dms: []printedDMRow{
			{
				heading:    "DM of +1 if",
				conditions: perService{"Intel 8+", "Intel 8+", "Dext 6+", "Intel 6+", "Stren 7+", "—"},
			},
			{
				heading:    "DM of +2 if",
				conditions: perService{"Educ 9+", "Stren 8+", "Endur 5+", "Stren 8+", "Intel 6+", "—"},
			},
		},
	},

	survival: printedThrows{
		targets: perService{"5+", "6+", "5+", "7+", "5+", "5+"},
		dms: []printedDMRow{
			{
				heading:    "DM of +2 if",
				conditions: perService{"Intel 7+", "Endur 8+", "Educ 6+", "Endur 9+", "Intel 7+", "Intel 9+"},
			},
		},
	},

	commission: printedThrows{
		targets: perService{"10+", "9+", "5+", "—", "4+", "—"},
		dms: []printedDMRow{
			{
				heading:    "DM of +1 if",
				conditions: perService{"Social 9+", "Educ 7+", "Endur 7+", "—", "Intel 6+", "—"},
			},
		},
	},

	promotion: printedThrows{
		targets: perService{"8+", "9+", "6+", "—", "10+", "—"},
		dms: []printedDMRow{
			{
				heading:    "DM of +1 if",
				conditions: perService{"Educ 8+", "Social 8+", "Educ 7+", "—", "Intel 9+", "—"},
			},
		},
	},

	draft:    [len(traveller.ServiceNames)]int{1, 2, 3, 4, 5, 6},
	reenlist: perService{"6+", "6+", "7+", "3+", "4+", "5+"},
	ranks: [rankRows]perService{
		{"Ensign", "Lieutenant", "Lieutenant", "—", "4th Officer", "—"},
		{"Lieutenant", "Captain", "Captain", "—", "3rd Officer", "—"},
		{"Lt Cmdr", "Force Cmdr", "Major", "—", "2nd Officer", "—"},
		{"Commander", "Lt Colonel", "Lt Colonel", "—", "1st Officer", "—"},
		{"Captain", "Colonel", "Colonel", "—", "Captain", "—"},
		{"Admiral", "Brigadier", "General", "—", "—", "—"},
	},
	grants: []printedGrant{
		{label: "Navy Captain", grant: "+1 Social"},
		{label: "Navy Admiral", grant: "+1 Social"},
		{label: "Marine", grant: "Cutlass-1"},
		{label: "Marine Lieutenant", grant: "Revolver-1"},
		{label: "Army", grant: "Rifl3-1"},
		{label: "Army Lieutenant", grant: "SMG-1"},
		{label: "Merchant 1st Officer", grant: "Pilot-1"},
		{label: "Scout", grant: "Pilot-1"},
	},

	// Book 1 p. 21
	pensions: []printedPension{
		{terms: 5, pay: 4000},
		{terms: 6, pay: 6000},
		{terms: 7, pay: 8000},
		{terms: 8, pay: 10000},
	},
	perAdditionalTerm: 2000,
	pensionPaidBy:     []traveller.ServiceName{traveller.Navy, traveller.Marines, traveller.Army, traveller.Merchants},
}

// boxServices is how p. 23's Rank and Service Skills box spells the services
// its labels open with. E012 decides it: the box is "the one place the
// Marines are named in the singular, and the reason the box's entries cannot
// be matched to a service by printed string alone". Other has no entry, and
// the box prints none for it.
var boxServices = map[string]traveller.ServiceName{
	"Navy":     traveller.Navy,
	"Marine":   traveller.Marines,
	"Army":     traveller.Army,
	"Scout":    traveller.Scouts,
	"Merchant": traveller.Merchants,
}

// transcribedMustering is Book 1 p. 9 (Mustering Out Tables); pp. 21-22 (what
// each benefit is).
//
// Seven rows, not six: a die of 1-6 plus the +1 a rank of 5 or 6 allows reaches
// 7. The dash rows are the font trap - under text extraction they come out as
// the digit 4, which would give a Scout a seventh benefit he does not have.
var transcribedMustering = musteringTranscription{
	table1: [MusterRows][len(traveller.ServiceNames)]string{
		{"Low Psg", "Low Psg", "Low Psg", "Low Psg", "Low Psg", "Low Psg"},
		{"+1 Intel", "+2 Intel", "+1 Intel", "+2 Intel", "+1 Intel", "+1 Intel"},
		{"+2 Educ", "+1 Educ", "+2 Educ", "+2 Educ", "+1 Educ", "+1 Educ"},
		{"Blade", "Blade", "Gun", "Blade", "Gun", "Gun"},
		{"Travellers'", "Travellers'", "High Psg", "Gun", "Blade", "High Psg"},
		{"High Psg", "High Psg", "Mid Psg", "Scout", "Low Psg", "—"},
		{"+2 Social", "+2 Social", "+1 Social", "—", "Merchant", "—"},
	},
	table2: [MusterRows][len(traveller.ServiceNames)]int64{
		{1000, 2000, 2000, 20000, 1000, 1000},
		{5000, 5000, 5000, 20000, 5000, 5000},
		{5000, 5000, 10000, 30000, 10000, 10000},
		{10000, 10000, 10000, 30000, 20000, 10000},
		{20000, 20000, 10000, 50000, 20000, 10000},
		{50000, 30000, 20000, 50000, 40000, 50000},
		{50000, 40000, 30000, 50000, 40000, 100000},
	},

	// Book 1 p. 9. The two optional DMs the page prints, and the skill that
	// earns the second. The roll counts and rank thresholds that used to sit
	// here are bare procedural bounds indexing no printed table, so they are
	// constants beside the code that applies them (#43).
	table1DMFromRank5or6: 1,
	table2DMFromGambling: 1,
	table2ModifierFrom:   "Gambling",

	// E012. Table 1 abbreviates; the definitions on pp. 21-23 spell each out.
	// The last two rows are why this list is not optional: Table 1's Scout
	// column row 6 prints Scout and its Merchant column row 7 prints Merchant,
	// the same two strings that table uses as service column headers.
	names: map[string]string{
		"Low Psg":     "Low Passage",
		"Mid Psg":     "Middle Passage",
		"High Psg":    "High Passage",
		"Travellers'": "Travellers' Aid",
		"Scout":       "Scout ship, Type S",
		"Merchant":    "Free Trader, Type A",
	},

	// Book 1 pp. 21-22
	prices: [len(traveller.PassageClasses)]int64{
		traveller.HighPassage:   10000,
		traveller.MiddlePassage: 8000,
		traveller.LowPassage:    1000,
	},
	resalePercent: 90,
}

// transcribedAging is Book 1 p. 9 (Aging Table); p. 7 (when a round is run).
//
// The table prints two header rows, Term of Service 4 through 14 and Age 34
// through 74+, and a row per characteristic. A row's cells are each stretched
// by dotted leaders across a group of term columns; where a group begins is
// read off where each cell's leaders begin, and the lift takes the bands from
// every row's cells together. Education and Social Standing are unaffected
// throughout. The last band is terminal per E014: the Age row's 74+ is the
// only cell in either header carrying a plus, so the column it labels is
// open-ended, and the term row's 14 is simply the term that first reaches it.
var transcribedAging = agingTranscription{
	terms: []string{"4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14"},
	ages:  []string{"34", "38", "42", "46", "50", "54", "58", "62", "66", "70", "74+"},
	rows: []printedAgingRow{
		{label: "Strength", cells: []printedSpan{
			{text: "−1 (8+)", from: 4, through: 7},
			{text: "−1 (9+)", from: 8, through: 11},
			{text: "−2 (9+)", from: 12, through: 14},
		}},
		{label: "Dexterity", cells: []printedSpan{
			{text: "−1 (7+)", from: 4, through: 7},
			{text: "−1 (8+)", from: 8, through: 11},
			{text: "−2 (9+)", from: 12, through: 14},
		}},
		{label: "Endurance", cells: []printedSpan{
			{text: "−1 (8+)", from: 4, through: 7},
			{text: "−1 (9+)", from: 8, through: 11},
			{text: "−2 (9+)", from: 12, through: 14},
		}},
		{label: "Intelligence", cells: []printedSpan{
			{text: "no effect before age 66", from: 4, through: 11},
			{text: "−1 (9+)", from: 12, through: 14},
		}},
		{label: "Education", cells: []printedSpan{
			{text: "unaffected by aging", from: 4, through: 14},
		}},
		{label: "Social Standing", cells: []printedSpan{
			{text: "unaffected by aging", from: 4, through: 14},
		}},
	},

	// Book 1 pp. 7-8; E008, E009
	crisisSaving: "8+", crisisRecoversTo: 1, crisisMonthsDice: 1,
}

// transcribedWeapons is Book 1 p. 12 (Blades and Polearms); p. 13 (Guns).
//
// Both boxes are set in two columns and neither states a reading order.
// Column-major is the only order that keeps blades before polearms - the box's
// own title, read in order - and conventional arms before advanced. Row-major
// interleaves both groups into nonsense.
var transcribedWeapons = [len(traveller.WeaponCategories)][]string{
	traveller.Blade: {
		"Dagger",
		"Blade",
		"Foil",
		"Cutlass",
		"Sword",
		"Broadsword",
		"Spear",
		"Halberd",
		"Pike",
		"Cudgel",
		"Bayonet",
	},
	traveller.Gun: {
		"Body Pistol",
		"Automatic Pistol",
		"Revolver",
		"Carbine",
		"Rifle",
		"Laser Carbine",
		"Laser Rifle",
		"Automatic Rifle",
		"Submachine Gun",
		"Shotgun",
	},
}

// transcribedShips is Book 2 p. 18 (Scout/Courier, Type S); Book 2 p. 19 (Free
// Trader, Type A). Consulted only because Book 1 p. 22 points there.
//
// Only what a character record carries. Book 1 pp. 22-23 already give the
// payment terms, the ten-years-per-repeat rule and the constructive possession
// a scout ship is held under; what these pages add is which ship each benefit
// names, and how large it is.
var transcribedShips = [len(traveller.ShipKinds)]int{
	traveller.ScoutShip:  100,
	traveller.FreeTrader: 200,
}

// transcribedNobility is Book 3 p. 22, consulted only because Book 1 p. 5
// points there.
//
// The book prints each rank as a pair and the pair is kept: p. 8 records that
// the rules require no gender of any character.
var transcribedNobility = []printedTitle{
	{socialStanding: 11, title: "knight/dame"},
	{socialStanding: 12, title: "baron/baroness"},
	{socialStanding: 13, title: "marquis/marchioness"},
	{socialStanding: 14, title: "count/countess"},
	{socialStanding: 15, title: "duke/duchess"},
}
