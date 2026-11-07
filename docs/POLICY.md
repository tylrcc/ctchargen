# POLICY: the auto-mode decision table

2026-09-02. Milestone 0. Companion to `PRD.md`.

**Version 3.** Every record `--auto` writes carries it, in the `policy` field.

Changing what any row below answers is a version bump, because it changes
every character `--auto` makes — and a referee reading a record can
otherwise see only what was answered, not which table answered it. `build`
names the writer and moves for reasons that have nothing to do with the
answers; this moves only when an answer does.

Adding a row for a choice point that did not exist is not a bump: no record
was answered differently under the old table, because the question was never
asked. `docsgate` holds the number here to the constant in the code.

`--auto` answers every question the procedure asks. This document is the
whole of what it answers, one row per `Decider` method. The gate holds the
two to each other in both directions: every row here names a method that
exists, and every method has a row (PRD, Testing and the gate).

## The interface

`Decider` has **one method per choice point**, each typed to its own
alphabet. That is what closes the set: adding a choice point breaks every
implementation at compile time. Interactive mode, auto mode, and the tests
each implement all of it.

```go
type Decider interface {
    // Career
    Service(from []EnlistmentOffer) (ServiceName, error)
    SubmitToDraft() (bool, error)
    AttemptCommission() (bool, error)
    AttemptPromotion() (bool, error)
    ReenlistIntent(from []Intent) (Intent, error)
    AssumeTitle(t Title) (bool, error)

    // Skills
    SkillTable(from []SkillTable) (SkillTable, error)
    Weapon(cat WeaponCategory, from []WeaponName) (WeaponName, error)

    // Mustering out
    MusterTable(from []MusterTable) (MusterTable, error)
    MusterTable1DM() (bool, error)
    MusterTable2DM() (bool, error)
    MusterWeapon(cat WeaponCategory, from []WeaponName,
        received []WeaponName) (WeaponBenefit, error)
}
```

Twelve methods, and the PRD's FR9 sketch elides six of them behind its
ellipsis. Two notes on where this differs from what FR9 prints:

- **`Service` is offered `[]EnlistmentOffer`, not `[]ServiceName`.** FR9
  sketches `Service(from []ServiceName)`. The PRD also says the auto policy
  tie-breaks "by first-listed order in Book 1", and a tie-break presupposes a
  ranking; the only ranking a service has is its enlistment throw and the
  cumulative characteristic DMs it earns (p. 10). An `EnlistmentOffer`
  carries exactly that — the `ServiceName`, the service's printed enlistment
  `Target`, and the DM this character earns against it — so the ranking stays
  a pure function of the offered set and the policy never reaches into the
  character. The shape `from []X` survives; only the element type changes.
  Recorded here rather than applied silently.
- **The remaining six signatures are new**, not sketched. They are fixed
  here so milestone 1 writes them once. Four alphabets come with them that
  the PRD's Domain model does not yet list — `Intent`, `MusterTable`,
  `WeaponBenefit`, `Title` — plus the `EnlistmentOffer` above.

**Every strategy is a pure function of the arguments it is handed** — no
hidden state, no randomness, no I/O, no reach into the character. Where a
strategy needs to know that an option is legal, the engine supplies that by
what it puts in the offered set, never by the strategy inspecting the
character. This is what makes each row testable on its own.

**The offered set is never empty**, and a method with only one legal answer
is never called — a question with one answer is not a choice, and calling it
would put a choice event in the log that no reader could have decided
differently. Where that rule bites, the row says so.

## Selecting strategies

Three flags, each choosing a named strategy over one group of rows:

| Flag       | Governs                                                                                              | Values                            | Default    |
| ---------- | ---------------------------------------------------------------------------------------------------- | --------------------------------- | ---------- |
| `--career` | `Service`, `SubmitToDraft`, `AttemptCommission`, `AttemptPromotion`, `ReenlistIntent`, `AssumeTitle` | `serve`, `retire`, `oneterm`      | `serve`    |
| `--skills` | `SkillTable`, `Weapon`                                                                               | `advanced`, `service`, `personal` | `advanced` |
| `--muster` | `MusterTable`, `MusterTable1DM`, `MusterTable2DM`, `MusterWeapon`                                    | `cash`, `goods`, `spartan`        | `cash`     |

A record generated under any non-default strategy names it in the record's
inputs (PRD, CLI sketch). The default triple — `serve`/`advanced`/`cash` —
is what a bare `--auto` run applies, and a record generated under it names
that too, so no record is silent about the policy that made it.

Ranked preferences below are written best-first. A strategy picks the
first of its ranked options that appears in the offered set; where its
ranking is silent, the tie-break is **first-listed order in Book 1**, which
for services is the Prior Service Table's column order (p. 10) and for skill
tables is the Acquired Skills Table's block order (p. 11).

---

## Career rows — `--career`

### `Service(from []EnlistmentOffer) (ServiceName, error)`

**Asked when** no `--service` flag was given. With `--service`, the
enlistment attempt is forced to the named service and the method is not
called — the throw is still made, and a failed throw still goes to the draft
(PRD, CLI sketch).

**Alphabet.** One `EnlistmentOffer` per service of the Prior Service Table
(p. 10), in book order — Navy, Marines, Army, Scouts, Merchants, Other —
each carrying the service's printed enlistment target and the cumulative DM
this character earns against it.

| Strategy          | Answer                                                                                                                     |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `serve` (default) | The offer whose throw is likeliest to succeed — the 2D probability of meeting its target after its DM. Ties to book order. |
| `retire`          | Same as `serve`.                                                                                                           |
| `oneterm`         | Same as `serve`.                                                                                                           |

The probability is computed from the target and DM the offer carries; it is
not a table of its own and it is not a rule. Two offers of equal probability
are separated by the order the engine listed them in, which is the Prior
Service Table's column order and nothing else.

### `SubmitToDraft() (bool, error)`

**Asked when** the enlistment throw failed. The offer exists at all because
of [E001](ERRATA.md#e001--the-draft-is-may-not-must) — the page prints both
_may_ and _must_, and _may_ governs.

| Strategy          | Answer                                                |
| ----------------- | ----------------------------------------------------- |
| `serve` (default) | yes                                                   |
| `retire`          | yes                                                   |
| `oneterm`         | **no** — generation ends with an 18-year-old civilian |

`oneterm` is the strategy that reaches the civilian record at all, and the
only one whose golden fixture can.

### `AttemptCommission() (bool, error)`

**Asked when** the service has commissions (not Scouts, not Other; p. 6,
p. 10), the character is not yet commissioned, and it is not a draftee's
first term (p. 5).

| Strategy  | Answer |
| --------- | ------ |
| all three | yes    |

Nothing in the rules costs a character anything for a failed commission
throw, and a commission carries a skill eligibility (p. 6). A strategy that
declined would be modelling a preference the book does not price.

### `AttemptPromotion() (bool, error)`

**Asked when** the character is commissioned and holds a rank below the
highest his service prints
([E013](ERRATA.md#e013--no-promotion-throw-at-the-top-of-the-table-of-ranks)).
One attempt per term (p. 6).

| Strategy  | Answer |
| --------- | ------ |
| all three | yes    |

### `ReenlistIntent(from []Intent) (Intent, error)`

**Asked when** the reenlistment throw has been made, was not a 12, and
succeeded, **and** more than one intent is legal. The engine does not ask
otherwise: a 12 forces another term "regardless of his desires" (p. 6), a
failed throw ends the service, and at the end of term 7 or later the only
remaining intent is to leave
([E003](ERRATA.md#e003--a-12-on-the-reenlistment-throw-recurs-past-term-7)).

**Alphabet.** `Continue` | `Discharge` | `Retire`. The engine offers:

| Term just ended | Offered                             |
| --------------- | ----------------------------------- |
| 1–4             | `Continue`, `Discharge`             |
| 5–6             | `Continue`, `Retire`                |
| 7 or later      | _not asked_ — departure is `Retire` |

`Discharge` and `Retire` never coexist: p. 21 makes leaving at the end of the
5th term or later retirement by definition ("is considered to have retired"),
and p. 7 caps voluntary service at seven terms. A `Retire` from the Scouts or
Other is a retirement that pays nothing (p. 21).

| Strategy          | Ranked preference                   |
| ----------------- | ----------------------------------- |
| `serve` (default) | `Continue` > `Retire` > `Discharge` |
| `retire`          | `Retire` > `Continue` > `Discharge` |
| `oneterm`         | `Discharge` > `Retire` > `Continue` |

`serve` takes the full seven terms whenever the throws allow. `retire` leaves
the moment leaving counts as retirement — the end of term 5 (p. 21) — whether
or not that service pays a pension. `oneterm` leaves at the first
opportunity.

### `AssumeTitle(t Title) (bool, error)`

**Asked when** the character survived generation and his final Social
Standing is 11 or greater — once, at the end of generation
([E011](ERRATA.md#e011--when-title-eligibility-is-assessed)). Eligibility is
assessed for the dead too, but they are not asked. `t` is the Book 3 p. 22
title the value confers.

| Strategy  | Answer |
| --------- | ------ |
| all three | yes    |

---

## Skills rows — `--skills`

### `SkillTable(from []SkillTable, taken []int) (SkillTable, error)`

**Asked once per skill eligibility**, before the die is rolled (p. 11:
"must specify the table being consulted prior to the die throw").

**Alphabet.** Personal Development, Service Skills, Advanced Education,
Advanced Education (Education 8+) — the fourth offered only while the
character's Education stands at 8 or greater at the moment of the
designation.

| Strategy             | Ranked preference                                                                    |
| -------------------- | ------------------------------------------------------------------------------------ |
| `advanced` (default) | Advanced Education (8+) > Advanced Education > Service Skills > Personal Development |
| `service`            | Service Skills > Advanced Education (8+) > Advanced Education > Personal Development |
| `personal`           | Personal Development > Service Skills > Advanced Education (8+) > Advanced Education |

**The ranking decides among the tables he has trained on least.** `taken` says
how many results the character has already had off each offered table; the
answer is the highest-ranked table among those with the smallest count.

Version 2 followed the ranking alone, so `advanced` designated Advanced
Education every time it was offered and Personal Development never — ninety
times to none over thirty characters (#34) — and a character generated under
the default never raised a characteristic at all, though p. 11's first table
is how that is done.

Where the weapon draw of `Weapon` admits that the page gives no basis, this
one does not need to: the page gives a basis, in that a character trains where
he is thin, and the strategy still says which way he leans. A first
designation, when every count is zero, is exactly what it was in version 2 —
which is why the rows above still read as they did.

`advanced` is the default because it is the one ranking that makes the
Education 8+ gate visible in a default run: it takes the fourth table the
instant it opens. The gate never closes again once open — p. 9's Aging Table
prints "unaffected by aging" across Education, and the procedure's one
negative table result is Other's −1 Social (p. 11) — so no strategy needs an
answer for an Education that falls back below 8.
`personal` is the ranking that reaches that one negative result.

### `Weapon(cat WeaponCategory, from []WeaponName, vary Vary) (WeaponName, error)`

**Asked when** a skills-table result is Blade Cbt or Gun Cbt, immediately
(p. 11, p. 12, p. 13).

**Alphabet.** The blades and polearms list (p. 12) or the guns list (p. 13),
whichever the category names, in printed order.

| Strategy  | Answer                                                          |
| --------- | --------------------------------------------------------------- |
| all three | A name drawn from the printed list for the category, uniformly. |

**Why drawn and not ranked.** This is the one question the book hands to a
player and gives no basis for. Where p. 11 makes him designate a table before
the throw, or p. 21 decides a departure by the term count, the page supplies
something to decide on and the policy decides on it. Here the page supplies a
list in printed order, and printed order is not a preference — taking the
first is as invented as drawing, and drawing admits it.

Version 1 took the first name every time, which made thirty auto-generated
characters carry twenty-two Body Pistols and seventeen Daggers and nothing
else (#34), though every name on both lists was offered. No strategy reached
it, so no flag could change it either.

The draw comes from the run's own dice, so a fixture that scripts its draws
still walks the same path. It is the only variation any strategy takes.

Repeat receipts take the same weapon again, which raises its level (p. 12:
"Additional acquisitions of expertise in the same weapon increase the present
level by one"). Concentrating rather than diversifying is deliberate: it is
the branch that produces a level above 1 in a default run, and the diversified
branch is reached through `--muster spartan` below.

---

## Mustering-out rows — `--muster`

### `MusterTable(from []MusterTable) (MusterTable, error)`

**Asked once per benefit roll**, before the die (p. 9: "The player must
designate the table before the die is rolled"). The engine offers Table 2
only while fewer than three rolls have been taken on it (p. 9: "A maximum of
three rolls on table 2 are allowed per character"); once the cap is reached
Table 1 is the only option and the method is not called.

| Strategy         | Ranked preference |
| ---------------- | ----------------- |
| `cash` (default) | Table 2 > Table 1 |
| `goods`          | Table 1 > Table 2 |
| `spartan`        | Table 1 > Table 2 |

`cash` reaches the three-roll cap in any run with three or more rolls, which
is the only way the cap is exercised. `goods` is what reaches the ship
benefits, Travellers' Aid, and Table 1's "—" rows.

### `MusterTable1DM() (bool, error)`

**Asked when** the character's final rank is 5 or 6 (p. 9: "Characters with
rank 5 or 6 may add +1 to their rolls on this table") and the roll is on
Table 1.

| Strategy         | Answer |
| ---------------- | ------ |
| `cash` (default) | yes    |
| `goods`          | yes    |
| `spartan`        | **no** |

### `MusterTable2DM() (bool, error)`

**Asked when** the character has Gambling skill (p. 9: "Individuals with
gambling expertise are allowed a DM of +1 on table 2") and the roll is on
Table 2.

| Strategy         | Answer |
| ---------------- | ------ |
| `cash` (default) | yes    |
| `goods`          | yes    |
| `spartan`        | **no** |

`spartan` is the only strategy that declines either DM, and the two
declined branches are reached differently:

- **Table 1's** is reached by a generated golden, `navy-spartan-declines` —
  spartan prefers Table 1, so a spartan character who reaches rank 5 is
  asked and refuses.
- **Table 2's cannot be reached by any strategy.** Spartan is the only one
  that would decline it and also the only one that never rolls on Table 2,
  because it prefers Table 1 and Table 1 is always on offer. The branch is
  held instead by `ctchargen.TestInteractiveOffersTheGamblingModifier`,
  where a person is asked and answers no.

The distinction is worth writing down: a branch no fixture exercises is a
branch nothing tests (PRD, Testing and the gate), and this one is reachable
only from outside the policy.

### `MusterWeapon(cat, from, received) (WeaponBenefit, error)`

**Asked when** a Table 1 roll is a weapon benefit (p. 9's Blade and Gun
rows). `from` is the category's printed list, column-major
([ERRATA.md](ERRATA.md#checked-and-found-determinate)). `received` is the
weapons **of this category** this character has already taken **as
benefits** — doubly narrower than the weapons he holds: p. 22 says
"Expertise may only be taken in a weapon received as a benefit", which
excludes weapons from skills-table results, and it offers the expertise "in
lieu of receiving a second or subsequent weapon of **exactly the same
type**", which keeps it inside the benefit's own category. A gun benefit
cannot be converted into expertise in a blade.

**Alphabet.** `TakeWeapon(name)` — a weapon from the list, whether or not he
already has one — or `TakeExpertise(name)`, +1 in a weapon in `received`.
The expertise option is offered only when `received` is non-empty — that is,
only from the second benefit of that category on.

| Strategy         | Answer                                                                                                                                            |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `cash` (default) | `TakeExpertise` on the first name in `received` when offered; otherwise `TakeWeapon` on the first name in the printed list.                       |
| `goods`          | Same as `cash`.                                                                                                                                   |
| `spartan`        | `TakeWeapon` on the first name in the printed list not already in `received`; if every name is in `received`, the first name in the printed list. |

`cash` and `goods` convert repeats into expertise, which is the branch p. 22's
worked sentence describes. `spartan` diversifies, which is the branch that
fills a character's benefit list with distinct weapons.

---

## Totality

Twelve methods; every one has a row, and every row answers for every offered
set the engine can construct. The policy never returns an error: an offered
set it cannot answer from would mean the engine offered something the rules
do not contain, which is an engine defect and fails there rather than being
absorbed here.

## What this document does not decide

- **Anything a flag on the command line already decides.** `--service`
  forces the enlistment attempt and suppresses `Service`; `--name` supplies
  the name, which is not a choice point (PRD, Decisions).
- **Anything after generation.** Selling a passage at 90% (p. 22), buying
  Travellers' Aid membership for CR 1,000,000 (p. 22), and flying the ship
  are in-play decisions; the record carries the assets and stops.
- **Interactive mode.** It implements the same interface and asks the
  player. The rows above describe only what `--auto` answers.
