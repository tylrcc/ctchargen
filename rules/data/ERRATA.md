# ERRATA: recorded readings

2026-09-02. Milestone 0. Companion to `PRD.md`.

Every place the held text is silent, ambiguous, or self-contradictory, and
the reading this tool applies there. Each entry carries the printed page,
**the page's own words**, the gap, the reading, why that reading, and the
**stamping condition** — which generated records name the erratum. A reading
applied without an entry here is the one defect no gate catches (PRD,
Testing and the gate); a reading applied silently on a record it governed is
the same defect wearing a citation.

All readings below were decided against a visual read of the held PDFs on
2026-09-02, at Book 1 PDF pages 8–31 (printed 2–25) and Book 3 PDF page 27
(printed 22). Printed page N is PDF page N+6 in Book 1, N+5 in Books 2 and 3.
Nothing here is inherited from the implementation removed at `41a213a`. A
gap found later may consult that tree's errata as a last resort — the page
ranks first, these readings second, the old errata third (`CLAUDE.md`,
Precedence) — but only as a pointer to a page, and what it yields is decided
here and gets its own id like any other reading.

Ids are stable. An id is never reused or renumbered; a withdrawn reading
keeps its heading and says it is withdrawn.

## Index

| Id                                                                            | Reading                                           | Pages      |
| ----------------------------------------------------------------------------- | ------------------------------------------------- | ---------- |
| [E001](#e001--the-draft-is-may-not-must)                                      | The draft is _may_, not _must_                    | 1:5        |
| [E002](#e002--the-per-term-order-is-the-expositions-not-the-examples)         | Per-term order is the exposition's                | 1:5–6, 24  |
| [E003](#e003--a-12-on-the-reenlistment-throw-recurs-past-term-7)              | A 12 on reenlistment recurs past term 7           | 1:6–7, 21  |
| [E004](#e004--the-age-of-a-character-who-dies-in-service)                     | Age of a character who dies in service            | 1:5, 9     |
| [E005](#e005--when-service-wide-rank-and-service-skills-are-granted)          | When service-wide rank/service skills are granted | 1:23       |
| [E006](#e006--where-the-aging-round-sits-within-the-term)                     | Where the aging round sits in the term            | 1:7, 9, 25 |
| [E007](#e007--the-order-of-saving-throws-within-an-aging-round)               | Order of saving throws in an aging round          | 1:9        |
| [E008](#e008--the-outcome-of-a-failed-medical-crisis-saving-throw)            | Outcome of a failed medical-crisis throw          | 1:7–8      |
| [E009](#e009--the-medical-expertise-dm-during-solo-generation)                | Medical-expertise DM during generation            | 1:7        |
| [E010](#e010--how-far-below-1-aging-may-carry-a-characteristic)               | How far below 1 aging may carry a characteristic  | 1:4, 7–9   |
| [E011](#e011--when-title-eligibility-is-assessed)                             | When title eligibility is assessed                | 1:5; 3:22  |
| [E012](#e012--printed-names-normalized-to-the-descriptions-headings)          | Printed names normalized to their headings        | 1:5, 9–23  |
| [E013](#e013--no-promotion-throw-at-the-top-of-the-table-of-ranks)            | No promotion throw at the top of the ranks table  | 1:6, 10    |
| [E014](#e014--the-aging-tables-last-column-is-terminal)                       | The Aging Table's last column is terminal         | 1:7, 9     |
| [E015](#e015--a-printed-table-governs-over-the-worked-examples-stated-result) | A printed table governs over the example's result | 1:9, 25    |

---

### E001 — The draft is _may_, not _must_

**Pages.** Book 1 p. 5, both the Enlistment and The Draft paragraphs.

**What the page says.** Enlistment: "Only one enlistment attempt is permitted
per character. If rejected for enlistment, he **may** submit to the draft.
Enlistment or draft is not allowed after age 18." The Draft, two paragraphs
later: "Should an attempt at enlistment fail, the character **must** submit
to the draft."

**The gap.** The same page states both modalities for the same step. One of
them has to lose.

**Reading.** _May._ A character rejected for enlistment is offered the draft
and may decline it. Declining ends generation immediately with a complete
record: an 18-year-old civilian, no service, no terms, no skills, no
benefits.

**Why.** The Enlistment paragraph is where the procedure states what the
character does next; The Draft paragraph opens the section that explains the
draft's mechanism, and its "must" is that section's framing sentence rather
than a second statement of the choice. The permissive reading also loses
nothing the restrictive one keeps — a character may always submit — while
the restrictive reading deletes a state the rules describe elsewhere as
ordinary (p. 5, opening ACQUIRING SKILLS AND EXPERTISE: "In order to acquire
some experience, it is possible to enlist in a service" — possible, not
required).

**Stamped on.** Records where the enlistment throw failed — both those that
submitted to the draft and those that declined.

---

### E002 — The per-term order is the exposition's, not the example's

**Pages.** Book 1 pp. 5–6 (the exposition), p. 24 (the Jamison example).

**What the page says.** Pp. 5–6 present the term in this order under their
own headings: Terms of Service, Survival, Commissions and Promotions, Skills
and Training, Reenlistment. The Jamison example instead narrates the
reenlistment throw and then the term's skills: "He signs up for a third term
of service _[reenlistment throw of 4+ required, no DMs, he throws 6]_ and is
accepted. He is eligible for two skills this term…"

**The gap.** Two orders for one term, and the order fixes dice-stream
consumption (PRD, Determinism), so it cannot be left open.

**Reading.** The exposition's order governs. Each term runs: survival throw →
commission attempt → promotion attempt → skill eligibilities rolled →
reenlistment throw → aging round (see [E006](#e006--where-the-aging-round-sits-within-the-term)).

**Why.** The exposition is the rule; the example illustrates it and is
elsewhere explicit that it takes liberties for readability — the same
example batches two aging rounds and says so on p. 25 ("but is instead being
resolved at this time for simplicity"). The set of throws is identical under
either order and no outcome changes; only the sequence in which the stream
is drawn differs, which is exactly why the rule text, not the narration,
should fix it.

**Stamped on.** Records in which at least one term proceeded past its
survival throw. Not stamped on a civilian record, which has no term, nor on
one whose only term ended at its survival throw, where nothing after the
survival throw was drawn and the order is invisible.

---

### E003 — A 12 on the reenlistment throw recurs past term 7

**Pages.** Book 1 p. 6 (Reenlistment), p. 7 (Retirement), p. 21 (Annual
Retirement Pay).

**What the page says.** P. 6: "If the throw is a 12 (exactly), the needs of
the service require that the character serve another term, regardless of his
desires. The reenlistment throw is required to be made during each term of
service." P. 7: "Service beyond the seventh term is normally impossible, and
retirement mandatory. However, persons who throw 12 (exactly) on the final
reenlistment throw must serve an additional term of service."

**The gap.** P. 7 grants "**an** additional term" in the singular, and says
nothing about a second 12 thrown during that additional term.

**Reading.** The rule recurs without limit. The reenlistment throw is made
during every term including terms past the seventh; a 12 on any of them
grants one further term, and the throw is made again in that term.

**Why.** P. 6 states without qualification that the throw "is required to be
made during each term of service" — the term granted by a 12 is a term of
service, so it carries its own throw, and p. 7's "the final reenlistment
throw" is then simply whichever throw turns out to be last. The book also
plainly contemplates careers past the singular reading's ceiling: p. 21's
retirement pay table prints an 8-terms row and then "Service beyond 8 terms
adds CR 2000 per additional term", which the singular reading can reach only
once and can never exceed.

**Stamped on.** Records in which a 12 was thrown on a reenlistment throw at
the end of term **8 or later**. A 12 at the end of term 7 is the case p. 7
prints in its own words; only a second one, thrown during the term that first
12 granted, needs this reading.

---

### E004 — The age of a character who dies in service

**Pages.** Book 1 p. 5 (Terms of Service, Survival), p. 9 (Aging Table note).

**What the page says.** P. 5: "Upon enlistment (or upon being drafted), a
character embarks on a term of service lasting 4 years. This adds 4 years to
the character's age." And: "Failure to successfully achieve the survival
throw results in death; a new character must be generated." P. 9's note:
"Term of service refers to the end of that numbered term", with term 4 set
against age 34 — so age is 18 + 4 × terms completed.

**The gap.** The survival throw is the term's first step and kills the
character part-way through a term the page's own arithmetic only counts when
completed. The record must state an age (PRD FR8) and the page never states
this one.

**Reading.** The fatal term counts as served. A character who dies during
term N is recorded with N terms of service and age 18 + 4N, and the record
names the term and the cause. No aging round is run for the fatal term (the
round sits at the end of the term — [E006](#e006--where-the-aging-round-sits-within-the-term) — and the character did not reach it),
and no mustering out occurs.

**Why.** P. 5 attaches the four years to the term itself — "a term of service
lasting 4 years. This adds 4 years to the character's age" — and the term a
character died in is a term he entered and did not leave; counting it keeps
age and terms in the one relation the Aging Table's note fixes, 18 + 4 ×
terms, which is what any reader of the record will assume. The alternative — age
18 + 4(N−1), the term uncounted — makes the record say a character died in a
term he never served.

**Stamped on.** Records ending in death by a failed survival throw.

---

### E005 — When service-wide rank and service skills are granted

**Page.** Book 1 p. 23, the Rank and Service Skills box and its text.

**What the page says.** "Some skills accrue to a character automatically
(without the necessity of throwing for them, and without using up
eligibility) by virtue of a specific service or a specific rank. … This table
should be consulted during each term of service, and the skills added to the
character as soon as he becomes eligible for them." The box's entries are
keyed either to a service alone (Marine, Army, Scout) or to a service and a
rank (Navy Captain, Navy Admiral, Marine Lieutenant, Army Lieutenant,
Merchant 1st Officer).

**The gap.** For a rank entry, "as soon as he becomes eligible" is exact: the
moment the commission or promotion confers the rank, as the Jamison example
shows for Merchant 1st Officer ("receiving his promotion and his master's
papers (including automatic pilot-1 expertise)", p. 24). For an entry keyed
to a service alone there is no moment on the page at all; the character is
eligible from the instant he is in the service, and "during each term" would
grant the skill again every term.

**Reading.** A service-wide entry is granted **once**, at enlistment or at
being drafted into that service, before the first term's survival throw. A
rank entry is granted at the moment its rank is conferred, before that term's
skill eligibilities are rolled. Neither is granted a second time.

**Why.** "As soon as he becomes eligible" is the operative clause, and
eligibility for a service-wide entry arrives with the service. The box's
"consulted during each term" is an instruction to re-check the table, not a
grant that repeats: the surrounding sentence describes skills that accrue
"by virtue of a specific service or a specific rank", and a virtue held
continuously is not earned repeatedly. Granting at entry also puts the skill
where the character can be seen to have it for the whole of the service the
book says confers it.

**Stamped on.** Records in a service with a service-wide entry — Marines
(Cutlass-1), Army (Rifle-1), Scouts (Pilot-1) — including records that end in
death during the first term.

---

### E006 — Where the aging round sits within the term

**Pages.** Book 1 p. 7 (Aging), p. 9 (Aging Table note), p. 25 (the example).

**What the page says.** P. 7: "There is the possibility of detrimental aging
effects when a character reaches the age of 34, and in 4 year increments
thereafter. When a character turns 34 (when adventuring during the game, or
**at the end of the 4th term of service**), he is subject to a possible
reduction in his characteristics." P. 9's note: "Term of service refers to
the end of that numbered term." P. 25: "he is subject to 2 rounds of aging
(one round should have been made at the end of term of service 4, but is
instead being resolved at this time for simplicity; the other round is due to
the end of term of service 5)."

**The gap.** The pages fix the aging round at the _end of the term_ and fix
it as recurring at the end of every term from the fourth on. They do not fix
its position relative to the reenlistment throw, which p. 6 places "during"
the term. The position fixes dice-stream consumption.

**Reading.** The aging round is the last step of the term, after the
reenlistment throw. Rounds run at the end of term 4 and at the end of every
term thereafter — never batched at mustering out. The round is read off the
table's **Term of Service** row, not its Age row: p. 9's note keys the Age
row to physiological age, which the months added by a medical crisis
(pp. 7–8) can push past the term's own arithmetic, and the term is the
unambiguous index. That row stops at 14; what governs a term past it is
[E014](#e014--the-aging-tables-last-column-is-terminal).

**Why.** The two pages use different prepositions for the two steps and the
distinction is the only one available: the reenlistment throw is made
"during each term of service" (p. 6), the aging round comes "at the end of"
the term (p. 7, and p. 9's note reading the table the same way). Refusing the
example's batching is not a reading at all — p. 25 says outright that
batching is a simplification and names the round it displaced.

**Stamped on.** Records in which at least one aging round was run. Four
completed terms is not the condition: a character killed by the survival
throw of his fourth term has four terms and never reached the round.

---

### E007 — The order of saving throws within an aging round

**Page.** Book 1 p. 9, the Aging Table.

**What the page says.** The table's rows are Strength, Dexterity, Endurance,
Intelligence, Education, Social Standing — the order in which the six are
rolled at p. 4 — with a reduction and a parenthesized saving throw in the
cells that carry one, "no effect before age 66" across Intelligence, and
"unaffected by aging" across Education and Social Standing. Its note: "The
negative number is the potential reduction in characteristic if the saving
throw (in parentheses) is not made. Saving throws use two dice."

**The gap.** A round at a given age calls for more than one saving throw. The
page never says in what order they are thrown, and the order fixes
dice-stream consumption.

**Reading.** Top to bottom in the table's own row order — Strength,
Dexterity, Endurance, Intelligence — throwing only for the characteristics
whose cell prints a reduction at that age, and skipping every cell that does
not. A reduction that brings a characteristic to 0 is resolved **inline**:
the medical-crisis saving throw, and the one die for months where it is
survived, are drawn before the next characteristic's saving throw, not
deferred to the end of the round.

**Why.** The table's row order is the page's only statement of sequence, and
it is also the order the six characteristics are rolled in at p. 4 and
printed in in the UPP at p. 8. Any other order would have to be invented. The
crisis resolves inline because p. 7 says so of the recovery it describes —
"his recovery is made immediately" — and because deferring it would leave the
character standing at 0 through the rest of a round the table indexes by term
of service ([E006](#e006--where-the-aging-round-sits-within-the-term)), not by
the character's condition.

**Stamped on.** Records in which at least one aging round was run. Stamped
alongside [E006](#e006--where-the-aging-round-sits-within-the-term).

---

### E008 — The outcome of a failed medical-crisis saving throw

**Pages.** Book 1 pp. 7–8.

**What the page says.** "If, as a result of aging or combat, a characteristic
is reduced to zero, the character is considered to be ill or wounded. A basic
saving throw of 8+ applies (and may be modified by the expertise of attending
medical personnel). **If the character survives**, his recovery is made
immediately (under slow drug, which speeds up his body chemistry). The
character ages (one die equals the number of months in added age)
immediately, but also returns to play fully recovered. The characteristic
which was reduced to zero automatically becomes one. This process occurs each
time (and for each characteristic) a characteristic is reduced to zero. In
the event that medical care is not available, the character is incapacitated
for the number of months indicated by the die roll."

**The gap.** The passage states the outcome when the throw is made and the
outcome when medical care is absent. It never states the outcome when the
throw is failed.

**Reading.** Failure is death. Generation ends there, with the record naming
the characteristic that reached zero, the term, and the age at death
(including any months already accrued from an earlier crisis). No mustering
out follows.

**Why.** "If the character survives" is the page's own framing and
presupposes that he might not; a saving throw whose failure had no stated
consequence would not be a saving throw. It is also the same shape the book
gives its other lethal throw — survival, p. 5 — where failure is death
stated plainly, and death is an outcome the record is built to carry (PRD,
Decisions).

**Stamped on.** Records in which a medical-crisis saving throw was made —
whether it succeeded or failed, since the reading is what put the throw's
failure branch there at all.

---

### E009 — The medical-expertise DM during solo generation

**Page.** Book 1 p. 7.

**What the page says.** "A basic saving throw of 8+ applies (and may be
modified by the expertise of attending medical personnel)."

**The gap.** Character generation has no attending personnel. There is no
crew, no patron, no referee-adjudicated scene — only the character being
generated — so the printed DM has no referent, and the page offers no value
for it.

**Reading.** No DM applies. The medical-crisis throw is an unmodified 8+ on
two dice, always.

**Why.** The clause is permissive ("may be modified") and names a source of
modification that generation does not contain. Supplying a value would be
inventing a number the page does not print, which the tool's authority model
forbids (PRD, Authority); a character's own Medical skill is not "attending
medical personnel" treating him.

**Stamped on.** Records in which a medical-crisis saving throw was made.
Stamped alongside [E008](#e008--the-outcome-of-a-failed-medical-crisis-saving-throw).

---

### E010 — How far below 1 aging may carry a characteristic

**Pages.** Book 1 p. 4, pp. 7–8, p. 9.

**What the page says.** P. 4: "As a result of various modifications,
characteristic values may ultimately range from 1 to 15. Characteristics (for
player-characters) may never exceed 15, and do not go below 1 **except for
calamitous injury or aging**." Pp. 7–8 name only one sub-1 value: "If, as a
result of aging or combat, a characteristic is **reduced to zero** … The
characteristic which was reduced to zero automatically becomes one." P. 9's
table prints reductions of −1 and −2.

**The gap.** P. 4 opens the floor for aging without saying how far, and p. 9
can print a −2 against a characteristic standing at 1. Whether the result is
0 or −1 decides whether the medical crisis of pp. 7–8 fires at all, and
therefore whether the one path that puts months on a character's age is
reachable.

**Reading.** Aging reductions floor at **0**. A reduction that would carry a
characteristic below 0 stops at 0, and 0 is a medical crisis, resolved per
pp. 7–8 (and [E008](#e008--the-outcome-of-a-failed-medical-crisis-saving-throw),
[E009](#e009--the-medical-expertise-dm-during-solo-generation)). Every other
alteration in the procedure — the skills tables' one negative result, Other's
−1 Social on p. 11, and every characteristic alteration from a table or a
rank — floors at **1**.

**Why.** Zero is the only value below 1 the rules give any meaning to, and
they give it a complete one: a named state, a saving throw, a recovery, and a
restoration to 1. A negative characteristic appears nowhere in the three
books and would have no rule to resolve it. Clamping at 0 makes the crisis
text apply literally as written, which the alternative does not.

**Stamped on.** Records in which an aging reduction carried a characteristic
below 1 — which is every reduction that reaches 0, and so every medical
crisis that aging caused.

An earlier wording of this condition restricted it to a −2 taken against a 1,
on the ground that "a −1 taken against a 1 reaches 0 under either reading".
That is arithmetically false and is corrected here: under the floor of 1, a
−1 against a 1 gives **1**, and no crisis follows. The two readings diverge
at exactly the point a reduction would carry a characteristic below 1, which
is the condition above.

---

### E011 — When title eligibility is assessed

**Pages.** Book 1 p. 5 (Titles); Book 3 p. 22 (Nobility).

**What the page says.** Book 1 p. 5: "**Titles:** A character with a Social
Standing of 11 or greater may assume his family's hereditary title. The full
range of titles is given in Book 3. For initial naming, a Social Standing of
11 allows use of Sir, denoting hereditary knighthood; a Social Standing of 12
allows use of Baron, or prefixing von to the character's surname." Book 3
p. 22: "Persons with social standing of 11 or greater **are considered to be
nobility** … Nobility have hereditary titles and high standing in their home
communities. … The nobility table indicates the actual designations or titles
**accruing to specific social standing values**", the table running from 11
knight/dame to 15 duke/duchess.

**The gap.** Neither page fixes a moment. Social Standing is not fixed at 18:
it rises on a personal development table and on a Table 1 mustering-out row
(pp. 9, 11), it rises with Navy Captain and Navy Admiral (p. 23), and it
falls on Other's personal development table (p. 11).

**Reading.** Eligibility is assessed **once, at the end of generation**,
against the character's final Social Standing — after mustering out, and for
every completed generation including one that ended in death. Where a
character who survived generation is eligible, assuming the title is a choice
point, and the record stores both the eligibility and the choice. A character
who died is assessed but not asked: eligibility is a condition of his Social
Standing, which Book 3 states of persons regardless, while assuming a title
is an act a dead man does not perform. His record carries the eligibility and
no assumption.

**Why.** Book 1 p. 5 points at Book 3 for the substance, and Book 3 states it
as a standing condition of a value, not an act at a moment: persons with the
value _are_ nobility, and titles _accrue to_ values. Reading it against the
final value is what makes that sentence true of the record the tool writes.
The competing anchor — "For initial naming" — is about what a name may look
like, not about when a title is held; it sits inside the NAMING section and
governs the naming conventions Sir and von, not the eligibility itself.

The reading's two visible consequences, both accepted: a mustering-out or
rank alteration **can** confer a title the character did not hold at 18, and
a character killed in service **is** assessed, on the Social Standing he held
when he died.

**Stamped on.** Records where the reading changed the answer — where
eligibility on the rolled Social Standing of p. 4 and eligibility on the
final Social Standing differ in either direction, and every record ending in
death whose final Social Standing is 11 or greater. A character eligible at
18 and still eligible at the end is assessed the same way under either
reading, and is not stamped.

---

### E012 — Printed names normalized to the descriptions' headings

**Pages.** Book 1 p. 5; pp. 9–11 (the tables); pp. 12–20 (the skill
descriptions, whose headings are what every name is normalized to);
pp. 21–23.

**What the page says.** The Acquired Skills Table (p. 11) abbreviates to fit
its columns; the skill descriptions (pp. 12–20) print each skill's name in
full as a heading; the Rank and Service Skills box (p. 23) abbreviates again
and carries at least one reprint typo; the Mustering Out Tables (p. 9)
abbreviate the benefits whose definitions are spelled out on pp. 21–23; and
the six services are spelled more than one way across pp. 5, 9, 10, 11 and 23.

**The gap.** A skill acquired twice under two spellings is two skills. Only
the descriptions' own headings spell every name once.

**Reading.** Every name is normalized to its description heading. This is the
one place a printed string is deliberately not reproduced. The full list:

_Skills_ — table spelling → recorded name:

| Printed     | Where                                      | Recorded as                                                    |
| ----------- | ------------------------------------------ | -------------------------------------------------------------- |
| Fwd Obsv    | p. 11                                      | Forward Observer                                               |
| Engnrng     | p. 11                                      | Engineer                                                       |
| Jack-o-T    | p. 11                                      | Jack of all Trades                                             |
| Admin       | p. 11                                      | Administration                                                 |
| Electronics | p. 11, Advanced Education 8+, Other column | Electronic                                                     |
| Blade Cbt   | p. 11                                      | _not a skill name_ — the chosen blade is (p. 11 bottom, p. 12) |
| Gun Cbt     | p. 11                                      | _not a skill name_ — the chosen gun is (p. 11 bottom, p. 13)   |
| Rifl3-1     | p. 23 box                                  | Rifle-1                                                        |
| SMG-1       | p. 23 box                                  | Submachine Gun-1                                               |

_Benefits_ — p. 9 spelling → recorded name, per the definitions on pp. 21–23:

| Printed     | Recorded as                                   |
| ----------- | --------------------------------------------- |
| Low Psg     | Low Passage                                   |
| Mid Psg     | Middle Passage                                |
| High Psg    | High Passage                                  |
| Travellers' | Travellers' Aid                               |
| Scout       | Scout ship, Type S (p. 23; Book 2 p. 18)      |
| Merchant    | Free Trader, Type A (pp. 22–23; Book 2 p. 19) |

The last two rows are why this list is not optional. Table 1's Scout column
row 6 prints **Scout** and its Merchant column row 7 prints **Merchant** —
the same two strings the same table uses as service column headers — so a
benefit read by its printed name collides with a service read by its printed
name. Table 1's characteristic rows (+1 Intel, +2 Educ, +2 Social and the
rest) are not normalized here because a characteristic is never recorded as a
printed string: it is an index into the six-key `Profile` of p. 4.

_Services_ — the six are recorded under the names p. 5 gives them when it
lists them: **Navy, Marines, Army, Scouts, Merchants, Other**. P. 10's Prior
Service Table and Table of Ranks agree with p. 5 (Scouts, Merchants); p. 9's
Table 1 prints Scout and Merchant, p. 9's Table 2 prints Scouts and Merchant,
and p. 11's Acquired Skills Table prints Scouts and Merchant. P. 23's Rank
and Service Skills box keys its entries to **Marine**, Army, Scout and
Merchant — the one place the Marines are named in the singular, and the
reason the box's entries cannot be matched to a service by printed string
alone. P. 5's list is the only place all six appear together.

Blade Combat and Gun Combat are not recorded as skills because the rules
require the specific weapon at once — p. 11: "When blade or gun combat is
acquired, the specific weapon in which expertise is achieved must be
specified immediately" — and the worked example's own skill list records the
weapons (Dagger-1, Cutlass-1, Body Pistol-1, Submachine Gun-1; p. 25). The
weapon named _blade_ on the p. 12 list is a weapon like any other, and p. 12's
warning that vague designation "defaults to expertise in the weapon named
blade" describes a player's error, not a result the tool can produce: the
weapon is always chosen explicitly here.

**Why.** The descriptions are where each skill is defined and named once;
the tables are typesetting. Rifl3-1 is a reprint's broken glyph for Rifle-1 —
no skill or weapon named "Rifl3" is defined or listed anywhere in the three
books, and Rifle is on the p. 13 gun list.

**Stamped on.** No record. A spelling is a transcription, not a reading:
nothing about a character changes with the choice, only what his skill is
called. The entry exists so the list is written down and checkable, not so it
can be cited.

---

### E013 — No promotion throw at the top of the Table of Ranks

**Pages.** Book 1 p. 6 (Commissions and Promotions), p. 10 (Table of Ranks).

**What the page says.** P. 6: "In the same term of service that he is
commissioned, and in each subsequent term of service, a character may attempt
to be promoted. … If a promotion is achieved, the character advances to the
next higher rank in his service. A character is eligible for one promotion
per term of service." P. 10's Table of Ranks runs to rank 6 for the Navy,
Marines and Army, to rank 5 for the Merchants, and prints no ranks at all for
Scouts and Other; its text: "Ranks, commissions, and promotions are
non-existent in the scout and other services."

**The gap.** The page never says what a character already holding his
service's highest printed rank does with his promotion eligibility. Whether
the throw is made is not cosmetic — a throw that is made consumes two dice
from the stream (PRD, Determinism).

**Reading.** No promotion throw is made once the character holds the highest
rank his service's column prints. The eligibility does not arise, so no dice
are drawn and no skill eligibility can follow from it.

**Why.** P. 6 defines the whole of a promotion's effect as advancing "to the
next higher rank in his service", and the Table of Ranks is what says which
ranks a service has. Where the column ends there is no next higher rank, so
there is nothing for a successful throw to do; making it would draw dice for
an outcome the rules cannot deliver. This is the same shape as the page's
explicit refusal of commission and promotion throws to the Scouts and Other —
where the table prints nothing, nothing is thrown.

**Stamped on.** Records in which a character reached the highest rank his
service prints and then proceeded past the survival throw of at least one
further term. A character who reached the top rank and then died on the next
term's survival throw never arrived at the promotion step, so the reading
changed nothing and drew nothing.

---

### E014 — The Aging Table's last column is terminal

**Pages.** Book 1 p. 7 (Aging), p. 9 (the Aging Table).

**What the page says.** P. 9's table is headed by two rows: **Term of
Service**, running 4, 5, 6, … 14, and **Age**, running 34, 38, 42, … 70,
**74+**. Its note: "Term of service refers to the end of that numbered term.
Age refers to the first day of the personal (physiological, not
chronological) year." P. 7: aging effects arrive at 34 "and in 4 year
increments thereafter", with no end named.

**The gap.** The Term of Service row is a closed list that stops at 14. The
Age row is not — its last cell is open-ended. `Term` has no upper bound
(PRD, Domain model) and a 12 on the reenlistment throw recurs without limit
([E003](#e003--a-12-on-the-reenlistment-throw-recurs-past-term-7)), so a
character can reach term 15, and
[E006](#e006--where-the-aging-round-sits-within-the-term) sends the round to
a row with no cell for him. The Age row would have covered him; the Term row
does not.

**Reading.** The table's last column is terminal: every term from the 14th on
is read off it. Aging never stops and never changes again after that column.

**Why.** The two header rows label one set of columns, and the Age row says
in the page's own notation that the last of them is open-ended — **74+**, the
only cell in either row carrying a plus. A column open-ended in one of its
labels is open-ended, and the Term row's 14 is simply the term that first
arrives there. P. 7 puts it beyond doubt from the other direction: increments
continue "thereafter" with no terminus, so a character past the table cannot
be a character who has stopped aging.

**Stamped on.** Records with fifteen or more terms of service — the terms
whose round is read off a column their own number does not print.

---

### E015 — A printed table governs over the worked example's stated result

**Pages.** Book 1 p. 9 (Mustering Out Table 1), p. 25 (the worked example).

**What the page says.** P. 9's Table 1 prints, in the Merchant column: row 1
Low Psg, row 2 +1 Intel, row 3 +1 Educ, row 4 Gun, row 5 Blade, row 6 Low
Psg, row 7 Merchant. No row of that column prints a Middle Passage.

P. 25 musters Jamison out with six rolls on Table 1, each carrying the +1 his
rank of 5 allows: "he rolls 5 (+1=6) = +1 education; 6 (+1=7) = merchant
ship; 2 (+1=3) = one middle passage; 6 (+1=7) = merchant ship; 6 (+1=7)
=merchant ship; 6 (+1=7) = merchant ship". Its summary then says "He also has
one middle passage, worth about CR 8,000."

**The gap.** Two of those six results are not what the table prints for the
row the roll reaches. A 6 reaches row 6, which is Low Psg, not +1 Education;
a 3 reaches row 3, which is +1 Educ, not a Middle Passage. And the Merchant
column has no Middle Passage at any row, so the summary describes a benefit
that column cannot deliver. The other four rolls agree with the table
exactly.

**Reading.** The printed table governs. Where the worked example's stated
result contradicts the table it is illustrating, the table is the rule and
the example is an illustration of it.

**Why.** The tables are the rules; the example is prose about them, and the
same example is elsewhere explicit that it takes liberties — it batches two
aging rounds "for simplicity" and says so on the same page. The narration
also mislabels the Acquired Skills tables twice on p. 24, calling the
Personal Development table "Table 7". An illustration that has already
miscited two tables is not evidence against a third.

This reading applies E002's principle to a table rather than to an order:
where the exposition and the example disagree, the exposition governs.

**Stamped on.** No record. No generated character depends on it, because
nothing but the worked example ever claims a result the table does not
print. It is recorded because a reader who reaches p. 25 with p. 9 open will
find the contradiction and needs to know which way this tool read it.

The visible consequence, for the replay of Jamison in `chargen`: the same
six rolls yield the same +1 Education and the same four merchant ships, and
differ in exactly one benefit — a **Low** Passage where p. 25 claims a
Middle one.

---

## Checked and found determinate

Read at milestone 0, considered as candidates, and **not** errata — the
pages settle them. Recorded so they are not re-litigated as silences.

- **Which throws are two dice.** P. 10: "All rolls except draft are two-die
  throws." The draft is one die (p. 5); so are the skills-table roll (p. 11),
  each mustering-out roll (p. 9) and the medical-crisis recovery months
  (p. 7).
- **Whether a dead character musters out.** P. 5 ends generation at a failed
  survival throw ("a new character must be generated"), so the departure
  p. 7's mustering out is written for never occurs.
- **Retirement pay is earned by the term count, not by the manner of
  leaving.** P. 21: "A character who leaves the service at the end of the 5th
  or later term of service is considered to have retired, and receives
  retirement pay" — so a character forced out by a failed reenlistment throw
  at the end of term 5 or later draws it too. P. 21 also excludes the Scouts
  and Other services from retirement pay entirely, whatever the term count.
- **The Education 8+ gate is tested against the current value.** P. 11:
  "Characters may distribute their rolls over the three tables (the four
  tables if the character is of education 8 or greater), but must specify the
  table being consulted prior to the die throw" — present tense, at the
  moment the table is designated, so an Education raised to 8 mid-career
  opens the fourth table from the next designation on.
- **Aging runs at the end of every term from the fourth, not only at 34.**
  P. 7 ("and in 4 year increments thereafter"), p. 9's table (terms 4 through 14) and p. 25's example (rounds owed for terms 4 and 5) agree.
- **The weapon lists are read column-major.** The Blades and Polearms box
  (p. 12) and the Guns box (p. 13) are each set in two columns, and the page
  never states a reading order — but only one order makes either box
  coherent. Column-major gives blades Dagger, Blade, Foil, Cutlass, Sword,
  Broadsword and then polearms Spear, Halberd, Pike, Cudgel, Bayonet, which
  is the box's own title read in order; and it gives guns Body Pistol,
  Automatic Pistol, Revolver, Carbine, Rifle before Laser Carbine, Laser
  Rifle, Automatic Rifle, Submachine Gun, Shotgun, conventional arms before
  advanced. Row-major interleaves both groups into nonsense. The order
  matters because it fixes what "first in the printed list" names in
  POLICY.md.
- **Mustering-out expertise stays inside the benefit's own category.**
  P. 22: a repeat weapon benefit may be taken as "+1 expertise **in lieu of
  receiving a second or subsequent weapon of exactly the same type**", and
  "Expertise may only be taken in a weapon received as a benefit." The
  expertise substitutes for the weapon that benefit would have delivered, so
  it is taken in a weapon of that benefit's category — the page's worked
  example never leaves one (blade → cutlass → cutlass-1). A gun benefit
  cannot be converted into expertise in a blade.
- **A character may voluntarily begin a seventh term but not an eighth.**
  P. 7: "A character may serve up to 7 terms voluntarily", and "Service beyond
  the seventh term is normally impossible" — only a 12 carries him further
  (see [E003](#e003--a-12-on-the-reenlistment-throw-recurs-past-term-7)).
