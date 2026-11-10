# Readings are the authority's record, not the referee's

A reading did two jobs. It is how the authority model records a decision
wherever the page is silent or contradicts itself (CLAUDE.md, Authority rule
5): a `Reading` value in `rules` with its cite and its stamping condition,
held by the gate. And it was shown to the referee: an `[E0xx]` id beside each
outcome it governed in the generation record, `new --readings` to end a note
with each named reading in full, and `ctchargen readings` to list them or
print the ones asked for.

The rc.4 playtest found the second job doing nothing for the referee (#460).
E002 was stamped on nearly every skill line and E006 and E007 on every aging
throw, with nothing telling him what to do with them, and E002, the id seen
most often, changes no outcome (#488). The full text cited a contract he
cannot find (#458) and named a Go type. `readings` was the one place on the
command line where case mattered. And the machinery for withdrawing a reading
had never been used, because no reading had ever been withdrawn (#502).

Decided 2026-11-10 (#460, grilled with #503): referees do not need the
readings. They stay as the authority's internal record: the `Reading` values,
Authority rule 5, the stamping conditions and the gate are unchanged, and the
engine still stamps each erratum on the character it governed, which is what
the stamping conditions hold every reading to. What goes is everything a
referee sees, and what only that served: an outcome no longer carries its
own errata, since nothing reads them once the note names none. No note names
an erratum, on the sheet or in the generation record. The `readings`
subcommand and the `--readings` flag are removed, and so is the withdrawal
machinery, along with the rendering of a reading's text. Where a reading changes something a referee acts on, the
note says so in plain words rather than by id (#483, #489). If a referee
asks for the readings, they come back.

An erratum's number is no longer a promise. No note carries one, so nothing
outside the tree is stamped with a decision that might later change. A
reading whose decision no longer holds is changed or removed like any other
code, and the stamping conditions are held to what it says then.

What is given up: a referee cannot tell from a note which reading governed an
outcome, nor read a reading without the source. The note's plain words and
the source are the two ways to it.

Supersedes in part ADR 0004: its decision that an erratum names one decision
for good, never reused and withdrawn rather than deleted, and its decision
that the tool explains its readings. Its other decisions stand: the readings
are data and so code, and the coverage map is a check.
