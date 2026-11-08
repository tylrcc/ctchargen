// Package traveller holds the domain of Classic Traveller character
// generation: the alphabets Book 1 prints, the values it works in, and the
// sums that say "exactly one of".
//
// It imports nothing else in this module, and nothing in it rolls dice,
// reads a table, or marshals JSON. A domain type that needs to know about
// any of those is in the wrong package.
//
// The dividing rule between what is a type here and what is data in the
// rules package: types carry identity, never rule invariants. A service that
// is not one of the six is not a service; a characteristic outside the six
// does not exist. A range a page prints is not a type; whether it is data or
// a constant beside the code is CLAUDE.md's Authority point 6.
//
// Exhaustiveness comes in two strengths here, and the difference matters:
//
//   - The sums (Enlistment, TableResult, BenefitRow, Departure, Event,
//     WeaponBenefit) are folds. Each has a cases interface with one method
//     per case, so adding a case adds a method and every implementation
//     stops compiling. That is the compiler, and nothing can switch it off.
//   - The plain enums are checked by the exhaustive linter, which the gate
//     runs. That is not the compiler: Go does not check switch
//     exhaustiveness, and dropping the linter would drop the guarantee.
//
// Page citations are to the printed pages of the FFE reprint of the (c) 1977
// text. Printed page N is PDF page N+6 in Book 1, N+5 in Books 2 and 3.
//
// Implements: The spellings that gate rests on tell every value of an alphabet
// apart [Choice points] in The record.
package traveller
