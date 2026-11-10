# The note is the record

The tool wrote a JSON record and rendered a sheet and a transcript from it,
by `new --sheet`, `new --history` or `render`. A referee keeping characters in
a markdown notes vault found the JSON dead weight there: nothing in the vault
reads it, and every sheet pointed at a file it did not link (#398). The
record also carried the service history twice, once as data for the sheet and
once as the events (#401). Decided 2026-11-09 (#398): the generation record
is for a person, not for a tool. `new` and `batch` write one markdown note per
character — the sheet, then the generation record, then what wrote it and
what to keep — and the note is the only way back to a character. The JSON
record, its schema and examples, `render`, `--sheet` and `--history` are
removed.

The cost is that nothing reads a character back: a note cannot be rendered
again in a later build's layout, a batch no longer pipes through `jq`, and
records written by earlier builds are no longer read. A `replay` that re-walks
a record, which ADR 0001 left possible, would need a record a tool can read,
and is not possible from the note as written.

Amends ADR 0001: the record that is the only way back is the note's
generation record. Amends ADR 0002: the ruleset, and the build where the
command wrote the note, are named on its last lines with who answered its
choices, rather than as record fields.

The note's shape is not frozen at `v1.0.0`. The bar #74 set — a referee can
trust what the tool prints, and the record is frozen and supported — keeps
its first half and the command line (#126); a note's headings and lines may
keep moving after `v1.0.0`, as the playtests ask of them. Decided with this
ADR, 2026-11-09.

Narrowed on 2026-11-09 (#399): a note now opens with properties, YAML a
vault's queries stand on, so the property names and their meanings freeze at
`v1.0.0`. Headings and prose stay free.

Finished on 2026-11-09 (#412, #397): dropping the JSON left the service
history stored twice in memory, beside the events. For the service history
and the service attempted, the events are now the one store: each outcome
they are read from carries a typed fact, and its prose is that fact's text.
The sheet's other facts - skills, benefits, characteristics, departure -
are still kept on the character beside the record's prose.

Extended on 2026-11-09 (#441): the characteristics and the skills are read
from the log too. Each initial roll, and each skill taken as expertise at
mustering out, now carries a typed fact, and every fact that changes a
characteristic or gives a skill carries the value the engine reached, so the
fold applies no rule. The engine keeps its own working copy while it runs,
since every DM reads it, but the finished character does not. Benefits,
pension, nobility, enlistment, rank, terms, age and departure are still kept
on the character beside the record's prose.

Extended again on 2026-11-09 (#441): the benefits, the pension and the title
are read from the log too. Each outcome that delivers a benefit - cash,
a passage, Travellers' Aid, a weapon, a ship - and the retirement pay and a
title conferred at the end now carries a typed fact. Each carries the value
the engine reached: the cash total after the roll, a Free Trader as it stands
after a repeat. An outcome that changes nothing - a benefit wasted or lost, a
row the table prints as a dash, a title the end took away - carries none.
The engine keeps its benefits so far as working state, since a repeat and the
cash total read them. Enlistment, rank, terms, age and departure are still
kept on the character beside the record's prose.

Finished on 2026-11-09 (#441): the career is read from the log too, so the
character keeps no fact about him beside it. The draft, the draft declined,
each reenlistment outcome, each death and the line that writes the rolled UPP
now carry a typed fact, and a commission or promotion's fact carries the rank
it conferred beside its name. Each carries the value the engine reached: the
age he started at, the age a term, the fatal term or a crisis's recovery left
him at, and how he left. The terms are read from the steps that open them. The engine
keeps its career so far as working state, since the commission and promotion
throws, mustering out and the pension read it. What the character still
holds beside its log is not a fact about him: its Inputs and its Policy,
which are what the generation was given, its Build, which names what wrote
it, and its Errata, the stamp of every reading that governed it, gathered
from the ids its outcomes name.
