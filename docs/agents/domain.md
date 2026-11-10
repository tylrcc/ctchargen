# Domain Docs

How the engineering skills should consume this repo's domain documentation
when exploring the codebase.

This repo is a single context: one glossary and one set of ADRs, both under
`docs/`, and both exist.

## Before exploring, read these

- **`docs/GLOSSARY.md`**: the domain's terms, as the code and the documents
  use them.
- **`docs/adr/`**: read the ADRs that touch the area you're about to work in.

The glossary lives in `docs/`, whatever a skill assumes about the root: never
create a `GLOSSARY.md` at the repo root. The `domain-modeling` skill adds a
term to the glossary or an ADR to `docs/adr/` when one is actually resolved;
the `codebase-design` skill supplies the vocabulary for module and interface
design.

## File structure

```
/
├── docs/GLOSSARY.md
├── docs/adr/
│   ├── 0001-random-is-random.md
│   └── …
├── cmd/ctchargen/
└── internal/
```

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor
proposal, a hypothesis, a test name), use the term as defined in
`docs/GLOSSARY.md`. Don't drift to synonyms the glossary explicitly avoids.

If the concept you need isn't in the glossary yet, that's a signal: either
you're inventing language the project doesn't use (reconsider) or there's a
real gap (note it for `domain-modeling`).

## Flag ADR conflicts

If your output contradicts an existing ADR, surface it explicitly rather than
silently overriding:

> _Contradicts ADR 0006 (the note is the record), but worth reopening
> because…_
