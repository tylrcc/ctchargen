package rules

import "errors"

// The two things that can go wrong here.
var (
	// errMalformed reports that an embedded table will not lift. It is a
	// build defect rather than a runtime condition: the tables ship inside
	// the binary, so a table that does not lift never lifted for anyone.
	// Every lift error wraps this and then says which cell.
	errMalformed = errors.New("malformed rules table")

	// errNoSuchRow reports a roll that the printed table has no row for -
	// a skills table asked for a seventh face, a mustering out table asked
	// for an eighth row, a draft die outside one through six.
	errNoSuchRow = errors.New("no such row")
)
