package rules

import "github.com/philoserf/ctchargen/internal/traveller"

// Reading is a decision recorded where the page is silent or contradicts
// itself, with its page cite and the condition under which it governs a
// record.
//
// Every field but the id is prose, kept verbatim with the page's own
// emphasis as inline Markdown, so a renderer or a test can quote it without a
// second transcription of it.
type Reading struct {
	ID    traveller.Erratum
	Title string

	// Pages is the printed pages the reading rests on.
	Pages string

	// PageSays is what the page says, quoted.
	PageSays string

	// Gap is where the page is silent or contradicts itself.
	Gap string

	// Reading is what this tool reads there.
	Reading string

	// Why is the rationale for that reading over another.
	Why string

	// StampedOn is the stamping condition: which records name the erratum.
	StampedOn string

	// Withdrawal is zero for a reading that stands.
	Withdrawal Withdrawal
}

// Withdrawal is why a reading's decision no longer governs new records, and
// what replaced it. A withdrawn reading keeps its erratum, which is never
// reused, and stays among the readings, so a record already stamped with it
// still names one that exists; it is never stamped on a new one.
type Withdrawal struct {
	// Why is the reason, in prose. A withdrawal is one that says why.
	Why string

	// ReplacedBy is the erratum whose reading took this one's place, or zero
	// where none did.
	ReplacedBy traveller.Erratum
}

// Withdrawn is whether the reading's decision no longer governs new records.
func (r Reading) Withdrawn() bool { return r.Withdrawal != Withdrawal{} }

// ReadingOf is the reading an erratum names.
func ReadingOf(e traveller.Erratum) Reading { return readings[e-1] }

// Readings is every reading, in erratum order. Its result type is the count
// check: readings is sized by its literal, so a reading missing, or one past
// the last erratum, does not compile here. TestTheReadings holds each to its
// place.
func Readings() [len(traveller.Errata)]Reading { return readings }

// readings is every reading, the one at index i named by erratum i+1: every
// place the held text is silent, ambiguous or self-contradictory, and the
// reading this tool applies there. A reading applied without one of these is
// the one defect no gate catches (PRD, Testing and the gate); a reading applied silently on a record it
// governed is the same defect wearing a citation.
//
// All of them were decided against a visual read of the held PDFs on
// 2026-09-02, at Book 1 PDF pages 8–31 (printed 2–25) and Book 3 PDF page 27
// (printed 22). Printed page N is PDF page N+6 in Book 1, N+5 in Books 2 and
// 3. Nothing here is inherited from the implementation removed at 41a213a. A
// gap found later may consult that tree's errata as a last resort — the page
// ranks first, these readings second, the old errata third (CLAUDE.md,
// Precedence) — but only as a pointer to a page, and what it yields is
// decided here and gets its own erratum like any other reading.
//
// A rationale that cites "PRD" cites the delivered v1 contract, removed from
// the tree after ec7289f; it is docs/PRD.md at that commit.
//
// An erratum names one decision for good (ADR 0004): it is never reused or
// renumbered, and a reading whose decision no longer holds is withdrawn
// rather than deleted.
var readings = [...]Reading{
	{
		ID:    traveller.E001,
		Title: "The draft is _may_, not _must_",
		Pages: "Book 1 p. 5, both the Enlistment and The Draft paragraphs.",
		PageSays: "Enlistment: \"Only one enlistment attempt is permitted\n" +
			"per character. If rejected for enlistment, he may submit to the draft.\n" +
			"Enlistment or draft is not allowed after age 18.\" The Draft, two paragraphs\n" +
			"later: \"Should an attempt at enlistment fail, the character must submit\n" +
			"to the draft.\"",
		Gap: "The same page states both modalities for the same step. One of\n" +
			"them has to lose.",
		Reading: "_May._ A character rejected for enlistment is offered the draft\n" +
			"and may decline it. Declining ends generation immediately with a complete\n" +
			"record: an 18-year-old civilian, no service, no terms, no skills, no\n" +
			"benefits.",
		Why: "The Enlistment paragraph is where the procedure states what the\n" +
			"character does next; The Draft paragraph opens the section that explains the\n" +
			"draft's mechanism, and its \"must\" is that section's framing sentence rather\n" +
			"than a second statement of the choice. The permissive reading also loses\n" +
			"nothing the restrictive one keeps — a character may always submit — while\n" +
			"the restrictive reading deletes a state the rules describe elsewhere as\n" +
			"ordinary (p. 5, opening ACQUIRING SKILLS AND EXPERTISE: \"In order to acquire\n" +
			"some experience, it is possible to enlist in a service\" — possible, not\n" +
			"required).",
		StampedOn: "Records where the enlistment throw failed — both those that\n" +
			"submitted to the draft and those that declined.",
	},
	{
		ID:    traveller.E002,
		Title: "The per-term order is the exposition's, not the example's",
		Pages: "Book 1 pp. 5–6 (the exposition), p. 24 (the Jamison example).",
		PageSays: "Pp. 5–6 present the term in this order under their\n" +
			"own headings: Terms of Service, Survival, Commissions and Promotions, Skills\n" +
			"and Training, Reenlistment. The Jamison example instead narrates the\n" +
			"reenlistment throw and then the term's skills: \"He signs up for a third term\n" +
			"of service _[reenlistment throw of 4+ required, no DMs, he throws 6]_ and is\n" +
			"accepted. He is eligible for two skills this term…\"",
		Gap: "Two orders for one term, and the record lists a term's throws\n" +
			"in the order they are made, so it cannot be left open.",
		Reading: "The exposition's order governs. Each term runs: survival throw →\n" +
			"commission attempt → promotion attempt → skill eligibilities rolled →\n" +
			"reenlistment throw → aging round (see [E006](#e006--where-the-aging-round-sits-within-the-term)).",
		Why: "The exposition is the rule; the example illustrates it and is\n" +
			"elsewhere explicit that it takes liberties for readability — the same\n" +
			"example batches two aging rounds and says so on p. 25 (\"but is instead being\n" +
			"resolved at this time for simplicity\"). The set of throws is identical under\n" +
			"either order and no outcome changes; only the order the record lists the\n" +
			"throws in differs, which is exactly why the rule text, not the narration,\n" +
			"should fix it.",
		StampedOn: "Records in which at least one term proceeded past its\n" +
			"survival throw. Not stamped on a civilian record, which has no term, nor on\n" +
			"one whose only term ended at its survival throw, where nothing after the\n" +
			"survival throw was drawn and the order is invisible.",
	},
	{
		ID:    traveller.E003,
		Title: "A 12 on the reenlistment throw recurs past term 7",
		Pages: "Book 1 p. 6 (Reenlistment), p. 7 (Retirement), p. 21 (Annual\n" +
			"Retirement Pay).",
		PageSays: "P. 6: \"If the throw is a 12 (exactly), the needs of\n" +
			"the service require that the character serve another term, regardless of his\n" +
			"desires. The reenlistment throw is required to be made during each term of\n" +
			"service.\" P. 7: \"Service beyond the seventh term is normally impossible, and\n" +
			"retirement mandatory. However, persons who throw 12 (exactly) on the final\n" +
			"reenlistment throw must serve an additional term of service.\"",
		Gap: "P. 7 grants \"an additional term\" in the singular, and says\n" +
			"nothing about a second 12 thrown during that additional term.",
		Reading: "The rule recurs without limit. The reenlistment throw is made\n" +
			"during every term including terms past the seventh; a 12 on any of them\n" +
			"grants one further term, and the throw is made again in that term.",
		Why: "P. 6 states without qualification that the throw \"is required to be\n" +
			"made during each term of service\" — the term granted by a 12 is a term of\n" +
			"service, so it carries its own throw, and p. 7's \"the final reenlistment\n" +
			"throw\" is then simply whichever throw turns out to be last. The book also\n" +
			"plainly contemplates careers past the singular reading's ceiling: p. 21's\n" +
			"retirement pay table prints an 8-terms row and then \"Service beyond 8 terms\n" +
			"adds CR 2000 per additional term\", which the singular reading can reach only\n" +
			"once and can never exceed.",
		StampedOn: "Records in which a 12 was thrown on a reenlistment throw at\n" +
			"the end of term **8 or later**. A 12 at the end of term 7 is the case p. 7\n" +
			"prints in its own words; only a second one, thrown during the term that first\n" +
			"12 granted, needs this reading.",
	},
	{
		ID:    traveller.E004,
		Title: "The age of a character who dies in service",
		Pages: "Book 1 p. 5 (Terms of Service, Survival), p. 9 (Aging Table note).",
		PageSays: "P. 5: \"Upon enlistment (or upon being drafted), a\n" +
			"character embarks on a term of service lasting 4 years. This adds 4 years to\n" +
			"the character's age.\" And: \"Failure to successfully achieve the survival\n" +
			"throw results in death; a new character must be generated.\" P. 9's note:\n" +
			"\"Term of service refers to the end of that numbered term\", with term 4 set\n" +
			"against age 34 — so the term arithmetic alone gives an age of 18 + 4 × terms\n" +
			"completed.",
		Gap: "The survival throw is the term's first step and kills the\n" +
			"character part-way through a term the page's own arithmetic only counts when\n" +
			"completed. The record must state an age (PRD FR8) and the page never states\n" +
			"this one.",
		Reading: "The fatal term counts as served. A character who dies during\n" +
			"term N is recorded with N terms of service, and the fatal term's four years\n" +
			"are added to the character's own age at death, including any months already\n" +
			"accrued from an earlier crisis (pp. 7–8), as " +
			"[E008](#e008--the-outcome-of-a-failed-medical-crisis-saving-throw) records a crisis death.\n" +
			"The record names the term and the cause. No aging round is run for the fatal\n" +
			"term (the round sits at the end of the term — [E006](#e006--where-the-aging-round-sits-within-the-term) — " +
			"and the character did not reach it),\n" +
			"and no mustering out occurs.",
		Why: "P. 5 attaches the four years to the term itself — \"a term of service\n" +
			"lasting 4 years. This adds 4 years to the character's age\" — and the term a\n" +
			"character died in is a term entered and never left; counting it keeps the\n" +
			"term and the four years p. 5 attaches to it together. The\n" +
			"alternative — the term uncounted — makes the record say a character died in a\n" +
			"term never served. The years are added to the age the character has, not\n" +
			"recomputed as 18 + 4N: that is the age only when no crisis was survived.\n" +
			"Pp. 7–8 put a recovery's months on the age at once — \"The character ages\n" +
			"(one die equals the number of months in added age) immediately\" — nothing on\n" +
			"the pages takes them off, and [E006](#e006--where-the-aging-round-sits-within-the-term) already lets them " +
			"carry the age past the\n" +
			"term's own arithmetic.",
		StampedOn: "Records ending in death by a failed survival throw.",
	},
	{
		ID:    traveller.E005,
		Title: "When service-wide rank and service skills are granted",
		Pages: "Book 1 p. 23, the Rank and Service Skills box and its text.",
		PageSays: "\"Some skills accrue to a character automatically\n" +
			"(without the necessity of throwing for them, and without using up\n" +
			"eligibility) by virtue of a specific service or a specific rank. … This table\n" +
			"should be consulted during each term of service, and the skills added to the\n" +
			"character as soon as he becomes eligible for them.\" The box's entries are\n" +
			"keyed either to a service alone (Marine, Army, Scout) or to a service and a\n" +
			"rank (Navy Captain, Navy Admiral, Marine Lieutenant, Army Lieutenant,\n" +
			"Merchant 1st Officer).",
		Gap: "For a rank entry, \"as soon as he becomes eligible\" is exact: the\n" +
			"moment the commission or promotion confers the rank, as the Jamison example\n" +
			"shows for Merchant 1st Officer (\"receiving his promotion and his master's\n" +
			"papers (including automatic pilot-1 expertise)\", p. 24). For an entry keyed\n" +
			"to a service alone there is no moment on the page at all; the character is\n" +
			"eligible from the instant he is in the service, and \"during each term\" would\n" +
			"grant the skill again every term.",
		Reading: "A service-wide entry is granted **once**, at enlistment or at\n" +
			"being drafted into that service, before the first term's survival throw. A\n" +
			"rank entry is granted at the moment its rank is conferred, before that term's\n" +
			"skill eligibilities are rolled. Neither is granted a second time.",
		Why: "\"As soon as he becomes eligible\" is the operative clause, and\n" +
			"eligibility for a service-wide entry arrives with the service. The box's\n" +
			"\"consulted during each term\" is an instruction to re-check the table, not a\n" +
			"grant that repeats: the surrounding sentence describes skills that accrue\n" +
			"\"by virtue of a specific service or a specific rank\", and a virtue held\n" +
			"continuously is not earned repeatedly. Granting at entry also puts the skill\n" +
			"where the character can be seen to have it for the whole of the service the\n" +
			"book says confers it.",
		StampedOn: "Records in a service with a service-wide entry — Marines\n" +
			"(Cutlass-1), Army (Rifle-1), Scouts (Pilot-1) — including records that end in\n" +
			"death during the first term.",
	},
	{
		ID:    traveller.E006,
		Title: "Where the aging round sits within the term",
		Pages: "Book 1 p. 7 (Aging), p. 9 (Aging Table note), p. 25 (the example).",
		PageSays: "P. 7: \"There is the possibility of detrimental aging\n" +
			"effects when a character reaches the age of 34, and in 4 year increments\n" +
			"thereafter. When a character turns 34 (when adventuring during the game, or\n" +
			"at the end of the 4th term of service), he is subject to a possible\n" +
			"reduction in his characteristics.\" P. 9's note: \"Term of service refers to\n" +
			"the end of that numbered term.\" P. 25: \"he is subject to 2 rounds of aging\n" +
			"(one round should have been made at the end of term of service 4, but is\n" +
			"instead being resolved at this time for simplicity; the other round is due to\n" +
			"the end of term of service 5).\"",
		Gap: "The pages fix the aging round at the _end of the term_ and fix\n" +
			"it as recurring at the end of every term from the fourth on. They do not fix\n" +
			"its position relative to the reenlistment throw, which p. 6 places \"during\"\n" +
			"the term. The position fixes dice-stream consumption.",
		Reading: "The aging round is the last step of the term, after the\n" +
			"reenlistment throw. Rounds run at the end of term 4 and at the end of every\n" +
			"term thereafter — never batched at mustering out. The round is read off the\n" +
			"table's **Term of Service** row, not its Age row: p. 9's note keys the Age\n" +
			"row to physiological age, which the months added by a medical crisis\n" +
			"(pp. 7–8) can push past the term's own arithmetic, and the term is the\n" +
			"unambiguous index. That row stops at 14; what governs a term past it is\n" +
			"[E014](#e014--the-aging-tables-last-column-is-terminal).",
		Why: "The two pages use different prepositions for the two steps and the\n" +
			"distinction is the only one available: the reenlistment throw is made\n" +
			"\"during each term of service\" (p. 6), the aging round comes \"at the end of\"\n" +
			"the term (p. 7, and p. 9's note reading the table the same way). Refusing the\n" +
			"example's batching is not a reading at all — p. 25 says outright that\n" +
			"batching is a simplification and names the round it displaced.",
		StampedOn: "Records in which at least one aging round was run. Four\n" +
			"completed terms is not the condition: a character killed by the survival\n" +
			"throw of his fourth term has four terms and never reached the round.",
	},
	{
		ID:    traveller.E007,
		Title: "The order of saving throws within an aging round",
		Pages: "Book 1 p. 9, the Aging Table.",
		PageSays: "The table's rows are Strength, Dexterity, Endurance,\n" +
			"Intelligence, Education, Social Standing — the order in which the six are\n" +
			"rolled at p. 4 — with a reduction and a parenthesized saving throw in the\n" +
			"cells that carry one, \"no effect before age 66\" across Intelligence, and\n" +
			"\"unaffected by aging\" across Education and Social Standing. Its note: \"The\n" +
			"negative number is the potential reduction in characteristic if the saving\n" +
			"throw (in parentheses) is not made. Saving throws use two dice.\"",
		Gap: "A round at a given age calls for more than one saving throw. The\n" +
			"page never says in what order they are thrown, and the order fixes\n" +
			"dice-stream consumption.",
		Reading: "Top to bottom in the table's own row order — Strength,\n" +
			"Dexterity, Endurance, Intelligence — throwing only for the characteristics\n" +
			"whose cell prints a reduction at that age, and skipping every cell that does\n" +
			"not. A reduction that brings a characteristic to 0 is resolved **inline**:\n" +
			"the medical-crisis saving throw, and the one die for months where it is\n" +
			"survived, are drawn before the next characteristic's saving throw, not\n" +
			"deferred to the end of the round.",
		Why: "The table's row order is the page's only statement of sequence, and\n" +
			"it is also the order the six characteristics are rolled in at p. 4 and\n" +
			"printed in, in the UPP at p. 8. Any other order would have to be invented. The\n" +
			"crisis resolves inline because p. 7 says so of the recovery it describes —\n" +
			"\"his recovery is made immediately\" — and because deferring it would leave the\n" +
			"character standing at 0 through the rest of a round the table indexes by term\n" +
			"of service ([E006](#e006--where-the-aging-round-sits-within-the-term)), not by\n" +
			"the character's condition.",
		StampedOn: "Records in which at least one aging round was run. Stamped\n" +
			"alongside [E006](#e006--where-the-aging-round-sits-within-the-term).",
	},
	{
		ID:    traveller.E008,
		Title: "The outcome of a failed medical-crisis saving throw",
		Pages: "Book 1 pp. 7–8.",
		PageSays: "\"If, as a result of aging or combat, a characteristic\n" +
			"is reduced to zero, the character is considered to be ill or wounded. A basic\n" +
			"saving throw of 8+ applies (and may be modified by the expertise of attending\n" +
			"medical personnel). If the character survives, his recovery is made\n" +
			"immediately (under slow drug, which speeds up his body chemistry). The\n" +
			"character ages (one die equals the number of months in added age)\n" +
			"immediately, but also returns to play fully recovered. The characteristic\n" +
			"which was reduced to zero automatically becomes one. This process occurs each\n" +
			"time (and for each characteristic) a characteristic is reduced to zero. In\n" +
			"the event that medical care is not available, the character is incapacitated\n" +
			"for the number of months indicated by the die roll.\"",
		Gap: "The passage states the outcome when the throw is made and the\n" +
			"outcome when medical care is absent. It never states the outcome when the\n" +
			"throw is failed.",
		Reading: "Failure is death. Generation ends there, with the record naming\n" +
			"the characteristic that reached zero, the term, and the age at death\n" +
			"(including any months already accrued from an earlier crisis). No mustering\n" +
			"out follows.",
		Why: "\"If the character survives\" is the page's own framing and\n" +
			"presupposes that he might not; a saving throw whose failure had no stated\n" +
			"consequence would not be a saving throw. It is also the same shape the book\n" +
			"gives its other lethal throw — survival, p. 5 — where failure is death\n" +
			"stated plainly, and death is an outcome the record is built to carry (PRD,\n" +
			"Decisions).",
		StampedOn: "Records in which a medical-crisis saving throw was made —\n" +
			"whether it succeeded or failed, since the reading is what put the throw's\n" +
			"failure branch there at all.",
	},
	{
		ID:    traveller.E009,
		Title: "The medical-expertise DM during solo generation",
		Pages: "Book 1 p. 7.",
		PageSays: "\"A basic saving throw of 8+ applies (and may be\n" +
			"modified by the expertise of attending medical personnel).\"",
		Gap: "Character generation has no attending personnel. There is no\n" +
			"crew, no patron, no referee-adjudicated scene — only the character being\n" +
			"generated — so the printed DM has no referent, and the page offers no value\n" +
			"for it.",
		Reading: "No DM applies. The medical-crisis throw is an unmodified 8+ on\n" +
			"two dice, always.",
		Why: "The clause is permissive (\"may be modified\") and names a source of\n" +
			"modification that generation does not contain. Supplying a value would be\n" +
			"inventing a number the page does not print, which the tool's authority model\n" +
			"forbids (PRD, Authority); a character's own Medical skill is not \"attending\n" +
			"medical personnel\" treating him.",
		StampedOn: "Records in which a medical-crisis saving throw was made.\n" +
			"Stamped alongside [E008](#e008--the-outcome-of-a-failed-medical-crisis-saving-throw).",
	},
	{
		ID:    traveller.E010,
		Title: "How far below 1 aging may carry a characteristic",
		Pages: "Book 1 p. 4, pp. 7–8, p. 9.",
		PageSays: "P. 4: \"As a result of various modifications,\n" +
			"characteristic values may ultimately range from 1 to 15. Characteristics (for\n" +
			"player-characters) may never exceed 15, and do not go below 1 except for\n" +
			"calamitous injury or aging.\" Pp. 7–8 name only one sub-1 value: \"If, as a\n" +
			"result of aging or combat, a characteristic is reduced to zero … The\n" +
			"characteristic which was reduced to zero automatically becomes one.\" P. 9's\n" +
			"table prints reductions of −1 and −2.",
		Gap: "P. 4 opens the floor for aging without saying how far, and p. 9\n" +
			"can print a −2 against a characteristic standing at 1. Whether the result is\n" +
			"0 or −1 decides whether the medical crisis of pp. 7–8 fires at all, and\n" +
			"therefore whether the one path that puts months on a character's age is\n" +
			"reachable.",
		Reading: "Aging reductions floor at **0**. A reduction that would carry a\n" +
			"characteristic below 0 stops at 0, and 0 is a medical crisis, resolved per\n" +
			"pp. 7–8 (and [E008](#e008--the-outcome-of-a-failed-medical-crisis-saving-throw),\n" +
			"[E009](#e009--the-medical-expertise-dm-during-solo-generation)). Every other\n" +
			"alteration in the procedure — the skills tables' one negative result, Other's\n" +
			"−1 Social on p. 11, and every characteristic alteration from a table or a\n" +
			"rank — floors at **1**.",
		Why: "Zero is the only value below 1 the rules give any meaning to, and\n" +
			"they give it a complete one: a named state, a saving throw, a recovery, and a\n" +
			"restoration to 1. A negative characteristic appears nowhere in the three\n" +
			"books and would have no rule to resolve it. Clamping at 0 makes the crisis\n" +
			"text apply literally as written, which the alternative does not.",
		StampedOn: "Records in which an aging reduction carried a characteristic\n" +
			"below 1 — which is every reduction that reaches 0, and so every medical\n" +
			"crisis that aging caused.\n" +
			"\n" +
			"An earlier wording of this condition restricted it to a −2 taken against a 1,\n" +
			"on the ground that \"a −1 taken against a 1 reaches 0 under either reading\".\n" +
			"That is arithmetically false and is corrected here: under the floor of 1, a\n" +
			"−1 against a 1 gives **1**, and no crisis follows. The two readings diverge\n" +
			"at exactly the point a reduction would carry a characteristic below 1, which\n" +
			"is the condition above.",
	},
	{
		ID:    traveller.E011,
		Title: "When title eligibility is assessed",
		Pages: "Book 1 p. 5 (Titles); Book 3 p. 22 (Nobility).",
		PageSays: "Book 1 p. 5: \"**Titles:** A character with a Social\n" +
			"Standing of 11 or greater may assume his family's hereditary title. The full\n" +
			"range of titles is given in Book 3. For initial naming, a Social Standing of\n" +
			"11 allows the use of Sir, denoting hereditary knighthood; a Social Standing of 12\n" +
			"allows use of Baron, or prefixing von to the character's surname.\" Book 3\n" +
			"p. 22: \"Persons with social standing of 11 or greater are considered to be\n" +
			"nobility … Nobility have hereditary titles and high standing in their home\n" +
			"communities. … The nobility table indicates the actual designations or titles\n" +
			"accruing to specific social standing values\", the table running from 11\n" +
			"knight/dame to 15 duke/duchess.",
		Gap: "Neither page fixes a moment. Social Standing is not fixed at 18:\n" +
			"it rises on a personal development table and on a Table 1 mustering-out row\n" +
			"(pp. 9, 11), it rises with Navy Captain and Navy Admiral (p. 23), and it\n" +
			"falls on Other's personal development table (p. 11).",
		Reading: "Eligibility is assessed **once, at the end of generation**,\n" +
			"against the character's final Social Standing — after mustering out, and for\n" +
			"every completed generation including one that ended in death. Where a\n" +
			"character who survived generation is eligible, assuming the title is a choice\n" +
			"point, and the record stores both the eligibility and the choice. A character\n" +
			"who died is assessed but not asked: eligibility is a condition of his Social\n" +
			"Standing, which Book 3 states of persons regardless, while assuming a title\n" +
			"is an act a dead man does not perform. His record carries the eligibility and\n" +
			"no assumption.",
		Why: "Book 1 p. 5 points at Book 3 for the substance, and Book 3 states it\n" +
			"as a standing condition of a value, not an act at a moment: persons with the\n" +
			"value _are_ nobility, and titles _accrue to_ values. Reading it against the\n" +
			"final value is what makes that sentence true of the record the tool writes.\n" +
			"The competing anchor — \"For initial naming\" — is about what a name may look\n" +
			"like, not about when a title is held; it sits inside the NAMING section and\n" +
			"governs the naming conventions Sir and von, not the eligibility itself.\n" +
			"\n" +
			"The reading's two visible consequences, both accepted: a mustering-out or\n" +
			"rank alteration **can** confer a title the character did not hold at 18, and\n" +
			"a character killed in service **is** assessed, on the Social Standing he held\n" +
			"when he died.",
		StampedOn: "Records where the reading changed the answer — where\n" +
			"eligibility on the rolled Social Standing of p. 4 and eligibility on the\n" +
			"final Social Standing differ in either direction, and every record ending in\n" +
			"death whose final Social Standing is 11 or greater. A character eligible at\n" +
			"18 and still eligible at the end is assessed the same way under either\n" +
			"reading, and is not stamped.",
	},
	{
		ID:    traveller.E012,
		Title: "Printed names normalized to the descriptions' headings",
		Pages: "Book 1 p. 5; pp. 9–11 (the tables); pp. 12–20 (the skill\n" +
			"descriptions, whose headings are what every name is normalized to);\n" +
			"pp. 21–23.",
		PageSays: "The Acquired Skills Table (p. 11) abbreviates to fit\n" +
			"its columns; the skill descriptions (pp. 12–20) print each skill's name in\n" +
			"full as a heading; the Rank and Service Skills box (p. 23) abbreviates again\n" +
			"and carries at least one reprint typo; the Mustering Out Tables (p. 9)\n" +
			"abbreviate the benefits whose definitions are spelled out on pp. 21–23; and\n" +
			"the six services are spelled more than one way across pp. 5, 9, 10, 11 and 23.",
		Gap: "A skill acquired twice under two spellings is two skills. Only\n" +
			"the descriptions' own headings spell every name once.",
		Reading: "Every name is normalized to its description heading. This is the\n" +
			"one place a printed string is deliberately not reproduced. The full list:\n" +
			"\n" +
			"_Skills_ — table spelling → recorded name:\n" +
			"\n" +
			"| Printed     | Where                                      | Recorded as                                  " +
			"                  |\n" +
			"| ----------- | ------------------------------------------ | " +
			"-------------------------------------------------------------- |\n" +
			"| Fwd Obsv    | p. 11                                      | Forward Observer                             " +
			"                  |\n" +
			"| Engnrng     | p. 11                                      | Engineer                                     " +
			"                  |\n" +
			"| Jack-o-T    | p. 11                                      | Jack of all Trades                           " +
			"                  |\n" +
			"| Admin       | p. 11                                      | Administration                               " +
			"                  |\n" +
			"| Electronics | p. 11, Advanced Education 8+, Other column | Electronic                                   " +
			"                  |\n" +
			"| Blade Cbt   | p. 11                                      | _not a skill name_ — the chosen blade is (p. " +
			"11 bottom, p. 12) |\n" +
			"| Gun Cbt     | p. 11                                      | _not a skill name_ — the chosen gun is (p. " +
			"11 bottom, p. 13)   |\n" +
			"| Rifl3-1     | p. 23 box                                  | Rifle-1                                      " +
			"                  |\n" +
			"| SMG-1       | p. 23 box                                  | Submachine Gun-1                             " +
			"                  |\n" +
			"\n" +
			"_Benefits_ — p. 9 spelling → recorded name, per the definitions on pp. 21–23:\n" +
			"\n" +
			"| Printed     | Recorded as                                   |\n" +
			"| ----------- | --------------------------------------------- |\n" +
			"| Low Psg     | Low Passage                                   |\n" +
			"| Mid Psg     | Middle Passage                                |\n" +
			"| High Psg    | High Passage                                  |\n" +
			"| Travellers' | Travellers' Aid                               |\n" +
			"| Scout       | Scout ship, Type S (p. 23; Book 2 p. 18)      |\n" +
			"| Merchant    | Free Trader, Type A (pp. 22–23; Book 2 p. 19) |\n" +
			"\n" +
			"The last two rows are why this list is not optional. Table 1's Scout column\n" +
			"row 6 prints **Scout** and its Merchant column row 7 prints **Merchant** —\n" +
			"the same two strings the same table uses as service column headers — so a\n" +
			"benefit read by its printed name collides with a service read by its printed\n" +
			"name. Table 1's characteristic rows (+1 Intel, +2 Educ, +2 Social and the\n" +
			"rest) are not normalized here because a characteristic is never recorded as a\n" +
			"printed string: it is an index into the six-key `Profile` of p. 4.\n" +
			"\n" +
			"_Services_ — the six are recorded under the names p. 5 gives them when it\n" +
			"lists them: **Navy, Marines, Army, Scouts, Merchants, Other**. P. 10's Prior\n" +
			"Service Table and Table of Ranks agree with p. 5 (Scouts, Merchants); p. 9's\n" +
			"Table 1 prints Scout and Merchant, p. 9's Table 2 prints Scouts and Merchant,\n" +
			"and p. 11's Acquired Skills Table prints Scouts and Merchant. P. 23's Rank\n" +
			"and Service Skills box keys its entries to **Marine**, Army, Scout and\n" +
			"Merchant — the one place the Marines are named in the singular, and the\n" +
			"reason the box's entries cannot be matched to a service by printed string\n" +
			"alone. P. 5's list is the only place all six appear together.\n" +
			"\n" +
			"Blade Combat and Gun Combat are not recorded as skills because the rules\n" +
			"require the specific weapon at once — p. 11: \"When blade or gun combat is\n" +
			"acquired, the specific weapon in which expertise is achieved must be\n" +
			"specified immediately\" — and the worked example's own skill list records the\n" +
			"weapons (Dagger-1, Cutlass-1, Body Pistol-1, Submachine Gun-1; p. 25). The\n" +
			"weapon named _blade_ on the p. 12 list is a weapon like any other, and p. 12's\n" +
			"warning that vague designation \"defaults to expertise in the weapon named\n" +
			"blade\" describes a player's error, not a result the tool can produce: the\n" +
			"weapon is always chosen explicitly here.",
		Why: "The descriptions are where each skill is defined and named once;\n" +
			"the tables are typesetting. Rifl3-1 is a reprint's broken glyph for Rifle-1 —\n" +
			"no skill or weapon named \"Rifl3\" is defined or listed anywhere in the three\n" +
			"books, and Rifle is on the p. 13 gun list.",
		StampedOn: "No record. A spelling is a transcription, not a reading:\n" +
			"nothing about a character changes with the choice, only what his skill is\n" +
			"called. The entry exists so the list is written down and checkable, not so it\n" +
			"can be cited.",
	},
	{
		ID:    traveller.E013,
		Title: "No promotion throw at the top of the Table of Ranks",
		Pages: "Book 1 p. 6 (Commissions and Promotions), p. 10 (Table of Ranks).",
		PageSays: "P. 6: \"In the same term of service that he is\n" +
			"commissioned, and in each subsequent term of service, a character may attempt\n" +
			"to be promoted. … If a promotion is achieved, the character advances to the\n" +
			"next higher rank in his service. A character is eligible for one promotion\n" +
			"per term of service.\" P. 10's Table of Ranks runs to rank 6 for the Navy,\n" +
			"Marines and Army, to rank 5 for the Merchants, and prints no ranks at all for\n" +
			"Scouts and Other; its text: \"Ranks, commissions, and promotions are\n" +
			"non-existent in the scout and other services.\"",
		Gap: "The page never says what a character already holding his\n" +
			"service's highest printed rank does with his promotion eligibility. Whether\n" +
			"the throw is made is not cosmetic — a throw that is made sits in the record\n" +
			"with its dice, and at the top rank it could change nothing.",
		Reading: "No promotion throw is made once the character holds the highest\n" +
			"rank his service's column prints. The eligibility does not arise, so no dice\n" +
			"are drawn and no skill eligibility can follow from it.",
		Why: "P. 6 defines the whole of a promotion's effect as advancing \"to the\n" +
			"next higher rank in his service\", and the Table of Ranks is what says which\n" +
			"ranks a service has. Where the column ends there is no next higher rank, so\n" +
			"there is nothing for a successful throw to do; making it would draw dice for\n" +
			"an outcome the rules cannot deliver. This is the same shape as the page's\n" +
			"explicit refusal of commission and promotion throws to the Scouts and Other —\n" +
			"where the table prints nothing, nothing is thrown.",
		StampedOn: "Records in which a character reached the highest rank his\n" +
			"service prints and then proceeded past the survival throw of at least one\n" +
			"further term. A character who reached the top rank and then died on the next\n" +
			"term's survival throw never arrived at the promotion step, so the reading\n" +
			"changed nothing and drew nothing.",
	},
	{
		ID:    traveller.E014,
		Title: "The Aging Table's last column is terminal",
		Pages: "Book 1 p. 7 (Aging), p. 9 (the Aging Table).",
		PageSays: "P. 9's table is headed by two rows: **Term of\n" +
			"Service**, running 4, 5, 6, … 14, and **Age**, running 34, 38, 42, … 70,\n" +
			"**74+**. Its note: \"Term of service refers to the end of that numbered term.\n" +
			"Age refers to the first day of the personal (physiological, not\n" +
			"chronological) year.\" P. 7: aging effects arrive at 34 \"and in 4 year\n" +
			"increments thereafter\", with no end named.",
		Gap: "The Term of Service row is a closed list that stops at 14. The\n" +
			"Age row is not — its last cell is open-ended. `Term` has no upper bound\n" +
			"(PRD, Domain model) and a 12 on the reenlistment throw recurs without limit\n" +
			"([E003](#e003--a-12-on-the-reenlistment-throw-recurs-past-term-7)), so a\n" +
			"character can reach term 15, and\n" +
			"[E006](#e006--where-the-aging-round-sits-within-the-term) sends the round to\n" +
			"a row with no cell for him. The Age row would have covered him; the Term row\n" +
			"does not.",
		Reading: "The table's last column is terminal: every term from the 14th on\n" +
			"is read off it. Aging never stops and never changes again after that column.",
		Why: "The two header rows label one set of columns, and the Age row says\n" +
			"in the page's own notation that the last of them is open-ended — **74+**, the\n" +
			"only cell in either row carrying a plus. A column open-ended in one of its\n" +
			"labels is open-ended, and the Term row's 14 is simply the term that first\n" +
			"arrives there. P. 7 puts it beyond doubt from the other direction: increments\n" +
			"continue \"thereafter\" with no terminus, so a character past the table cannot\n" +
			"be a character who has stopped aging.",
		StampedOn: "Records with fifteen or more terms of service — the terms\n" +
			"whose round is read off a column their own number does not print.",
	},
	{
		ID:    traveller.E015,
		Title: "A printed table governs over the worked example's stated result",
		Pages: "Book 1 p. 9 (Mustering Out Table 1), p. 25 (the worked example).",
		PageSays: "P. 9's Table 1 prints, in the Merchant column: row 1\n" +
			"Low Psg, row 2 +1 Intel, row 3 +1 Educ, row 4 Gun, row 5 Blade, row 6 Low\n" +
			"Psg, row 7 Merchant. No row of that column prints a Middle Passage.\n" +
			"\n" +
			"P. 25 musters Jamison out with six rolls on Table 1, each carrying the +1 his\n" +
			"rank of 5 allows: \"he rolls 5 (+1=6) = +1 education; 6 (+1=7) = merchant\n" +
			"ship; 2 (+1=3) = one middle passage; 6 (+1=7) = merchant ship; 6 (+1=7)\n" +
			"=merchant ship; 6 (+1=7) = merchant ship\". Its summary then says \"He also has\n" +
			"one middle passage, worth about CR 8,000.\"",
		Gap: "Two of those six results are not what the table prints for the\n" +
			"row the roll reaches. A 6 reaches row 6, which is Low Psg, not +1 Education;\n" +
			"a 3 reaches row 3, which is +1 Educ, not a Middle Passage. And the Merchant\n" +
			"column has no Middle Passage at any row, so the summary describes a benefit\n" +
			"that column cannot deliver. The other four rolls agree with the table\n" +
			"exactly.",
		Reading: "The printed table governs. Where the worked example's stated\n" +
			"result contradicts the table it is illustrating, the table is the rule and\n" +
			"the example is an illustration of it.",
		Why: "The tables are the rules; the example is prose about them, and the\n" +
			"same example is elsewhere explicit that it takes liberties — it batches two\n" +
			"aging rounds \"for simplicity\" and says so on the same page. The narration\n" +
			"also mislabels the Acquired Skills tables twice on p. 24, calling the\n" +
			"Personal Development table \"Table 7\". An illustration that has already\n" +
			"miscited two tables is not evidence against a third.\n" +
			"\n" +
			"This reading applies E002's principle to a table rather than to an order:\n" +
			"where the exposition and the example disagree, the exposition governs.",
		StampedOn: "No record. No generated character depends on it, because\n" +
			"nothing but the worked example ever claims a result the table does not\n" +
			"print. It is recorded because a reader who reaches p. 25 with p. 9 open will\n" +
			"find the contradiction and needs to know which way this tool read it.\n" +
			"\n" +
			"The visible consequence, for the reproduction of Jamison in `chargen`: the same\n" +
			"six rolls yield the same +1 Education and the same four merchant ships, and\n" +
			"differ in exactly one benefit — a **Low** Passage where p. 25 claims a\n" +
			"Middle one.",
	},
}
