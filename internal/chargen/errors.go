package chargen

import "errors"

// errNotOffered reports a decider that answered outside the set the
// procedure put in front of it.
//
// The page decides what a character may choose from - the Advanced Education
// Table is closed below Education 8 (p. 11), the expertise is offered only in
// a weapon already received (p. 22) - and a decider that reaches past the
// offered set is building a character the book does not allow. The engine
// refuses rather than applying it.
var errNotOffered = errors.New("answered outside what was offered")

// errOneAnswer reports a question the engine put to a decider with only one
// legal answer. A Decider method is called only when more than one answer is
// legal, so this is the engine breaking its own contract, not the decider
// answering badly: a question no reader could have decided differently would
// otherwise reach the record as a choice.
var errOneAnswer = errors.New("asked with one answer")

// errNoSuchStrategy reports a word that names none of a flag's strategies.
var errNoSuchStrategy = errors.New("no such strategy")
