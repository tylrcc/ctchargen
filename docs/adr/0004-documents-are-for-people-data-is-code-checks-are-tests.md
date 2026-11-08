# Documents are for people, data is code, checks are tests

Two files did two jobs. `ERRATA.md` was a document for people that `rules`
also embedded and parsed on every load, kept consistent with the
`traveller.Erratum` constants by a check that existed only because one fact
lived in two places; nothing at run time read what it parsed.
`docs/COVERAGE.md` was a document for people that was also generated output
the gate held to the code, and the marker grammar grew an `in <Section>.`
suffix only to lay it out. Each worked, and each needed machinery to stay
honest. Decided 2026-11-08, as `v1` work so that it lands before the record
freezes: no markdown file is parsed, generated or held to the code by the
gate.

The readings are data, so they are code. Each is a `Reading` value in
`rules`, in an array indexed by its erratum, so a missing or extra reading
does not compile. Its prose is kept verbatim, the page's own emphasis
included as inline markdown. The values are proven identical to `ERRATA.md`
before it goes (#321, #323), and what else it held moves beside the code:
each question it recorded as settled by the page becomes a comment on the
declaration that implements it, and its provenance becomes the readings' own
doc comment. Precedence and the clean room are unchanged; only where the
second rank is written down moves.

From `v1.0.0`, an erratum names one decision for good, a decision being what
its reading decides and when it is stamped. The reading may be reworded under
its erratum, but a changed decision is a new erratum, so a record stamped by
any build means the same to every build that renders it. A reading whose
decision no longer holds is withdrawn, not deleted (#322). It keeps its
erratum, which is never reused; it says why, and names the erratum that
replaced it, if any; it is still explained, marked withdrawn, on records
already stamped with it; and it is never stamped on a new one, which the
stamping harness holds by refusing any golden its predicate fits.

The tool explains its readings (#324). With `--readings`, a transcript keeps
its inline ids and ends with the full text of each reading its record names,
and `ctchargen readings` lists them or prints the ones asked for. The sheet
still leaves them out, for the reason it always has: on a sheet handed to a
player they are codes he does not need.

The coverage map is a check (#325). Every check stays, nothing is committed,
the section suffix goes, and the table is printed on demand.

What is given up: a malformed readings document no longer stops the binary,
since there is no document, and the compiler and the gate hold the readings
instead; and there is no coverage page on GitHub, only the table a run of
the tests prints. The rest of the tree's markdown, from README and CHANGELOG
to these ADRs and the release snapshots, is written for people and tools, and
the gate neither parses, generates nor holds any of it to the code.
