# ctchargen

A Classic Traveller character generator holding to the © 1977 text of Books
1–3. The language here is the language a referee and the code share.

## Language

**Note**:
The markdown the tool writes for one character: its properties, its heading
and summary, the sheet, then the generation record, then what wrote it and
what to keep (ADR 0006).
_Avoid_: Output, result, file

**Record**:
The note's generation record: every die thrown and every choice made, step
by step with its page cite. What makes the note the only way back to a
character.
_Avoid_: Transcript, history, log (for the whole), JSON

**Service history**:
The service a term at a time: what entering it granted, and what each term
conferred, gave and took. What the sheet's service record tells.
_Avoid_: Career (the policy's career strategy), log (the events)

**Auto run**:
A generation the policy answers, every choice point of it: `new --auto`, and
every batch, which has nobody else to ask. Its
note names the policy, the version of what it answers and its strategies,
whether or not the policy was asked anything.
_Avoid_: Automatic mode, batch (a batch is many auto runs)

**Batch**:
One run of `ctchargen batch`, and the notes it writes, kept together in a
folder of their own.
_Avoid_: Roster (the fixture roster)

**Member**:
One character in a batch.
_Avoid_: Entry

**Position**:
A member's place in its batch, the number its note's name opens with.
_Avoid_: Index, slot

**Guided run**:
A generation a person answers, every choice point of it. Its note names no
policy, and says the player made every choice.
_Avoid_: Interactive mode (for the note's fact), manual run

**Replay**:
Handing the tool the same inputs and getting the same character back. Not
offered since #200; the record is the only way back to a character.
_Avoid_: Repeatability, reproducibility, regenerate

**Offer**:
What a choice point puts in front of a decider: two or more legal
alternatives, the decider's own to keep.
_Avoid_: Options, menu, choices (for the set)

**Transcription**:
A printed table typed out by hand from a visual read of the page, twice and
independently, so a misread cell cannot pass from one copy to the other: once
as the page prints it, with its glyphs, abbreviations, notation and dashes,
and once in the domain's own words.
_Avoid_: Data file, table data, JSON

**Lift**:
Turning a transcription's printed text into the domain's values, and refusing
what the page could not have printed.
_Avoid_: Parse, decode

**Reading**:
A decision recorded where the page is silent or contradicts itself, with its
page cite and the condition under which it governs a record.
_Avoid_: Erratum (for the decision), ruling, interpretation

**Erratum**:
The id a record carries to say which readings governed it. An erratum names
one decision for good: its reading may be reworded, but a changed decision is
a new erratum.
_Avoid_: Reading (for the id), flag, tag

**Withdrawn reading**:
A reading whose decision no longer governs new records. It keeps its erratum,
is still explained on records it was stamped on, says why, and names the
erratum that replaced it, if any.
_Avoid_: Retired, superseded

**Stamping**:
Marking a record with an erratum because its reading's condition holds for
that record.
_Avoid_: Tagging, applying, flagging

**Script**:
The dice and draws of one path, in the order the procedure asks for them:
what a fixture or a walk states, and a scripted roller plays.
_Avoid_: Seed, sequence (for the whole)

**Fixture**:
A scripted sequence of dice and draws that walks the procedure down one chosen
path, named for the path it reaches, whose note is checked in as a golden.
_Avoid_: Seed, magic number, case

**Walk**:
The command tests' scripted path: dice and draws extracted from a record the
tool wrote, with no golden. The test that drives it is its proof.
_Avoid_: Fixture (a fixture has a golden), replay

**Golden**:
The note a fixture generates, checked in and regenerated never hand-edited.
_Avoid_: Snapshot, expected output, baseline

**Fold**:
A type that reads one of the sums into a value: it implements the sum's
cases interface, one method per case, and each method only records what its
case says, so folding into it cannot fail. Its name ends in `Fold`, and one
that reads its sum into prose ends in `WordsFold`: `serviceFold`,
`sheetFold`, `gainWordsFold`. A type that handles the cases to act on them,
and so can fail, is not a fold, and is named for what it does: applying a
table result, a benefit or a weapon to the run in progress, or a test's
handler that refuses a case.
_Avoid_: Codec, describer, recorder (for the type)
