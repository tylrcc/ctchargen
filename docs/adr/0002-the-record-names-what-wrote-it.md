# The record names what wrote it

A record carries `ruleset`, the text it was generated under, and `build`, the
build that wrote it, read from `debug.ReadBuildInfo` as `version` reports it.
The PRD's Determinism section promised both, as provenance for a character
that could be regenerated, and ADR 0001 withdrew that section with the seed.
The two fields stayed, for a reason of their own: a record is the only way
back to a character, so it says which books its page cites are to and which
code a referee is reading or filing a report against. They are provenance
for a person, not stamps a tool verifies. Only the command fills `build`; a
record generated in-process by a test carries none, so a golden does not
differ by machine. Recorded 2026-11-07, for a decision the tree already
carried (#232).
