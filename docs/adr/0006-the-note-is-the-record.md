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
