# ctchargen

A Classic Traveller character generator holding to the © 1977 text of Books
1–3. The language here is the language a referee and the code share.

## Language

**Record**:
The JSON the tool writes for one character, carrying every die thrown and
every choice made. The only way back to a character.
_Avoid_: Output, result, JSON file

**Auto run**:
A generation the `--auto` policy answers, every choice point of it. Its
record names the policy, the version of what it answers and its strategies,
whether or not the policy was asked anything.
_Avoid_: Automatic mode, batch (a batch is many auto runs)

**Guided run**:
A generation a person answers, every choice point of it. Its record names no
policy, and that absence is what says the player made every choice.
_Avoid_: Interactive mode (for the record's fact), manual run

**Replay**:
Handing the tool the same inputs and getting the same character back. Not
offered since #200; the record is the only way back to a character.
_Avoid_: Repeatability, reproducibility, regenerate

**Offer**:
What a choice point puts in front of a decider: two or more legal
alternatives, the decider's own to keep.
_Avoid_: Options, menu, choices (for the set)

**Reading**:
A decision recorded where the page is silent or contradicts itself, with its
page cite and the condition under which it governs a record.
_Avoid_: Erratum (for the decision), ruling, interpretation

**Erratum**:
The id a record carries to say which readings governed it.
_Avoid_: Reading (for the id), flag, tag

**Stamping**:
Marking a record with an erratum because its reading's condition holds for
that record.
_Avoid_: Tagging, applying, flagging

**Fixture**:
A scripted sequence of dice and draws that walks the procedure down one chosen
path, named for the path it reaches.
_Avoid_: Seed, magic number, case

**Golden**:
The record a fixture generates, checked in and regenerated never hand-edited.
_Avoid_: Snapshot, expected output, baseline
