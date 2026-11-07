# ctchargen

A Classic Traveller character generator holding to the © 1977 text of Books
1–3. The language here is the language a referee and the code share.

## Language

**Record**:
The JSON the tool writes for one character, carrying every die thrown and
every choice made. The only way back to a character.
_Avoid_: Output, result, JSON file

**Replay**:
Handing the tool the same inputs and getting the same character back. Not
offered since #200; the record is the only way back to a character.
_Avoid_: Repeatability, reproducibility, regenerate

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
