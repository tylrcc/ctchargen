package rules

import "github.com/philoserf/ctchargen/internal/traveller"

// skillsTranscription is the Acquired Skills Table as p. 11 prints it, with
// the eligibility and the education gate that govern it.
//
// Which table a grid is, and which service a column belongs to, are positions
// in traveller's own order: the tables in traveller.SkillTables order and the
// services in traveller.ServiceNames order. A cell is printed text, today's
// transcription of the page; where it still departs from the print is #339.
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
			{"+1 Strength", "+1 Strength", "+1 Strength", "+1 Strength", "+1 Strength", "+1 Strength"},
			{"+1 Dexterity", "+1 Dexterity", "+1 Dexterity", "+1 Dexterity", "+1 Dexterity", "+1 Dexterity"},
			{"+1 Endurance", "+1 Endurance", "+1 Endurance", "+1 Endurance", "+1 Endurance", "+1 Endurance"},
			{"+1 Social Standing", "Gambling", "Gambling", "Gun Cbt", "+1 Strength", "Blade Cbt"},
			{"+1 Intelligence", "Brawling", "Brawling", "+1 Intelligence", "Blade Cbt", "Brawling"},
			{"+1 Education", "Blade Cbt", "+1 Education", "+1 Education", "Bribery", "-1 Social Standing"},
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

// printedThrow is a throw as p. 10 prints it: a target and its DMs.
type printedThrow struct {
	target string
	dms    []printedDM
}

// printedDM is one DM and the condition the page prints for it.
type printedDM struct {
	dm        int
	condition string
}

// printedGrant is one row of the Rank and Service Skills box (p. 23).
type printedGrant struct {
	service traveller.ServiceName
	rank    int
	grant   string
}

// printedPension is one row of the Annual Retirement Pay table (p. 21).
type printedPension struct {
	terms int
	pay   int64
}

// servicesTranscription is the Prior Service Table and the Table of Ranks as
// p. 10 prints them, a column per service in traveller.ServiceNames order,
// with the Rank and Service Skills box and the retirement pay table.
type servicesTranscription struct {
	enlistment, survival  [len(traveller.ServiceNames)]printedThrow
	commission, promotion [len(traveller.ServiceNames)]*printedThrow
	draft                 [len(traveller.ServiceNames)]int
	reenlist              [len(traveller.ServiceNames)]string
	ranks                 [len(traveller.ServiceNames)][]string
	grants                []printedGrant
	pensions              []printedPension
	perAdditionalTerm     int64
	pensionPaidBy         []traveller.ServiceName
}

// musteringTranscription is the two Mustering Out Tables as p. 9 prints
// them, a column per service, with the two DMs, the benefits' names, and the
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

// printedEffect is one characteristic's cell of an aging band: its
// reduction and the saving throw that prevents it.
type printedEffect struct {
	characteristic string
	reduction      int
	saving         string
}

// printedBand is a column group of the Aging Table, from the term it starts.
type printedBand struct {
	fromTerm int
	effects  []printedEffect
}

// agingTranscription is the Aging Table (p. 9) and the medical crisis
// (pp. 7-8).
type agingTranscription struct {
	lastPrintedTerm                    int
	bands                              []printedBand
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
	enlistment: [len(traveller.ServiceNames)]printedThrow{
		{target: "8+", dms: []printedDM{{dm: 1, condition: "Intelligence 8+"}, {dm: 2, condition: "Education 9+"}}},
		{target: "9+", dms: []printedDM{{dm: 1, condition: "Intelligence 8+"}, {dm: 2, condition: "Strength 8+"}}},
		{target: "5+", dms: []printedDM{{dm: 1, condition: "Dexterity 6+"}, {dm: 2, condition: "Endurance 5+"}}},
		{target: "7+", dms: []printedDM{{dm: 1, condition: "Intelligence 6+"}, {dm: 2, condition: "Strength 8+"}}},
		{target: "7+", dms: []printedDM{{dm: 1, condition: "Strength 7+"}, {dm: 2, condition: "Intelligence 6+"}}},
		{target: "3+"},
	},
	survival: [len(traveller.ServiceNames)]printedThrow{
		{target: "5+", dms: []printedDM{{dm: 2, condition: "Intelligence 7+"}}},
		{target: "6+", dms: []printedDM{{dm: 2, condition: "Endurance 8+"}}},
		{target: "5+", dms: []printedDM{{dm: 2, condition: "Education 6+"}}},
		{target: "7+", dms: []printedDM{{dm: 2, condition: "Endurance 9+"}}},
		{target: "5+", dms: []printedDM{{dm: 2, condition: "Intelligence 7+"}}},
		{target: "5+", dms: []printedDM{{dm: 2, condition: "Intelligence 9+"}}},
	},
	commission: [len(traveller.ServiceNames)]*printedThrow{
		{target: "10+", dms: []printedDM{{dm: 1, condition: "Social Standing 9+"}}},
		{target: "9+", dms: []printedDM{{dm: 1, condition: "Education 7+"}}},
		{target: "5+", dms: []printedDM{{dm: 1, condition: "Endurance 7+"}}},
		nil,
		{target: "4+", dms: []printedDM{{dm: 1, condition: "Intelligence 6+"}}},
		nil,
	},
	promotion: [len(traveller.ServiceNames)]*printedThrow{
		{target: "8+", dms: []printedDM{{dm: 1, condition: "Education 8+"}}},
		{target: "9+", dms: []printedDM{{dm: 1, condition: "Social Standing 8+"}}},
		{target: "6+", dms: []printedDM{{dm: 1, condition: "Education 7+"}}},
		nil,
		{target: "10+", dms: []printedDM{{dm: 1, condition: "Intelligence 9+"}}},
		nil,
	},
	draft:    [len(traveller.ServiceNames)]int{1, 2, 3, 4, 5, 6},
	reenlist: [len(traveller.ServiceNames)]string{"6+", "6+", "7+", "3+", "4+", "5+"},
	ranks: [len(traveller.ServiceNames)][]string{
		{"Ensign", "Lieutenant", "Lt Cmdr", "Commander", "Captain", "Admiral"},
		{"Lieutenant", "Captain", "Force Cmdr", "Lt Colonel", "Colonel", "Brigadier"},
		{"Lieutenant", "Captain", "Major", "Lt Colonel", "Colonel", "General"},
		{},
		{"4th Officer", "3rd Officer", "2nd Officer", "1st Officer", "Captain"},
		{},
	},
	grants: []printedGrant{
		{service: traveller.Navy, rank: 5, grant: "+1 Social Standing"},
		{service: traveller.Navy, rank: 6, grant: "+1 Social Standing"},
		{service: traveller.Marines, rank: 0, grant: "Cutlass"},
		{service: traveller.Marines, rank: 1, grant: "Revolver"},
		{service: traveller.Army, rank: 0, grant: "Rifle"},
		{service: traveller.Army, rank: 1, grant: "Submachine Gun"},
		{service: traveller.Merchants, rank: 4, grant: "Pilot"},
		{service: traveller.Scouts, rank: 0, grant: "Pilot"},
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

// transcribedMustering is Book 1 p. 9 (Mustering Out Tables); pp. 21-22 (what
// each benefit is).
//
// Seven rows, not six: a die of 1-6 plus the +1 a rank of 5 or 6 allows reaches
// 7. The dash rows are the font trap - under text extraction they come out as
// the digit 4, which would give a Scout a seventh benefit he does not have.
var transcribedMustering = musteringTranscription{
	table1: [MusterRows][len(traveller.ServiceNames)]string{
		{"Low Psg", "Low Psg", "Low Psg", "Low Psg", "Low Psg", "Low Psg"},
		{"+1 Intelligence", "+2 Intelligence", "+1 Intelligence", "+2 Intelligence", "+1 Intelligence", "+1 Intelligence"},
		{"+2 Education", "+1 Education", "+2 Education", "+2 Education", "+1 Education", "+1 Education"},
		{"Blade", "Blade", "Gun", "Blade", "Gun", "Gun"},
		{"Travellers'", "Travellers'", "High Psg", "Gun", "Blade", "High Psg"},
		{"High Psg", "High Psg", "Mid Psg", "Scout", "Low Psg", "-"},
		{"+2 Social Standing", "+2 Social Standing", "+1 Social Standing", "-", "Merchant", "-"},
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
// The table's header rows are Term of Service 4 through 14 and Age 34 through
// 74+. Its three column groups are read off the Intelligence row, whose 'no
// effect before age 66' ends where age 66 - term 12 - begins. Education and
// Social Standing are unaffected throughout. The last band is terminal per
// E014: the Age row's 74+ is the only cell in either header carrying a plus, so
// the column it labels is open-ended, and the term row's 14 is simply the term
// that first reaches it.
var transcribedAging = agingTranscription{
	// Book 1 p. 9, the Term of Service header row; E014 reads the column it
	// labels as terminal.
	lastPrintedTerm: 14,
	bands: []printedBand{
		{ // ages 34, 38, 42, 46
			fromTerm: 4,
			effects: []printedEffect{
				{characteristic: "Strength", reduction: 1, saving: "8+"},
				{characteristic: "Dexterity", reduction: 1, saving: "7+"},
				{characteristic: "Endurance", reduction: 1, saving: "8+"},
			},
		},
		{ // ages 50, 54, 58, 62
			fromTerm: 8,
			effects: []printedEffect{
				{characteristic: "Strength", reduction: 1, saving: "9+"},
				{characteristic: "Dexterity", reduction: 1, saving: "8+"},
				{characteristic: "Endurance", reduction: 1, saving: "9+"},
			},
		},
		{ // ages 66, 70, 74+
			fromTerm: 12,
			effects: []printedEffect{
				{characteristic: "Strength", reduction: 2, saving: "9+"},
				{characteristic: "Dexterity", reduction: 2, saving: "9+"},
				{characteristic: "Endurance", reduction: 2, saving: "9+"},
				{characteristic: "Intelligence", reduction: 1, saving: "9+"},
			},
		},
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
