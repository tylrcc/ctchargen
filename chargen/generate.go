package chargen

import (
	"fmt"

	"github.com/philoserf/ctchargen/dice"
	"github.com/philoserf/ctchargen/rules"
	"github.com/philoserf/ctchargen/traveller"
)

// startingAge is where every character begins. P. 4: "All characters begin
// the game the same way, untrained, inexperienced, about 18 years of age."
//
// Implements: All characters begin at 18 [1:4] in Characteristics and the
// record.
const startingAge = 18

// Ruleset names the text this tool implements. All page cites in a record's
// generation log are to these three books and nothing else.
//
// Implements: The record names its ruleset and the build that wrote it
// [Determinism] in The record.
const Ruleset = "Classic Traveller, Books 1-3 (FFE reprint of the 1977 text)"

// An Option is an optional input to a generation: dice of the caller's own,
// or somebody watching the record as it is written.
//
// Options rather than a function per combination, so that the default -
// dice of its own and nobody watching - stays the shortest call, and the engine
// does not grow an entry point every time it gains an optional input.
type Option func(*run)

// WithRoller walks the procedure against a roller of the caller's own, which
// is how a fixture walks a particular path: a career past the Aging Table's
// last printed term, say, which takes a run of 12s no random career throws.
func WithRoller(roller Roller) Option {
	return func(r *run) { r.roll = roller }
}

// WithObserver calls watch with each event as the record is written, which
// is how interactive mode shows the procedure between its questions.
//
// It returns nothing, so a watcher cannot change what it watches.
//
// Implements: The procedure is shown between the questions [FR11] in The
// record.
func WithObserver(watch func(traveller.Event)) Option {
	return func(r *run) { r.log.watch = watch }
}

// WithAnswerer names who the decider stands for, which every choice event
// then records (FR9). A run says the policy answered unless it is told that
// a person did, so a record read back names the kind of thing that made
// each choice rather than assuming the automatic one.
//
// Implements: A guided run's choices are recorded as the player's, not the
// policy's [FR9] in The record.
func WithAnswerer(who traveller.DecidedBy) Option {
	return func(r *run) { r.by = who }
}

// Generate walks Book 1's character generation procedure, pp. 4-25, rolling
// dice of its own unless a roller is given.
//
// The order within a term is the exposition's, not the worked example's
// (E002): survival, commission, promotion, skills, reenlistment, and then
// the aging round at the end of the term (E006). That order is what every
// fixture's script walks, so it is a fixed choice rather than an incidental
// one.
//
// Implements: A fixture generates one golden, and the same one twice
// [Determinism] in The record.
// Implements: The worked example reproduces [1:23–25] in The book's own
// character.
func Generate(in Inputs, decider Decider, options ...Option) (*Character, error) {
	tables, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("loading the rules: %w", err)
	}

	record := newLog()

	run := &run{
		tables: tables,
		roll:   dice.New(),
		log:    record,
		by:     traveller.ByPolicy,
		char: &Character{
			Name: in.Name, Age: traveller.NewAge(startingAge), Inputs: in, Ruleset: Ruleset,
		},
	}

	for _, option := range options {
		option(run)
	}

	run.decide = logging{to: decider, by: run.by, log: record}

	err = run.generate()
	if err != nil {
		return nil, err
	}

	run.char.sortSkills()

	run.char.Events = record.events
	run.char.Errata = record.stamped()

	return run.char, nil
}

// serviceName is the service this run is generating in.
//
// Every caller is past enlistment, so the "did he serve at all" half of
// ServedIn is not in question here; a civilian run returns before any of
// them. The name comes from the Enlistment rather than a field beside it.
func (r *run) serviceName() traveller.ServiceName {
	name, _ := r.char.ServedIn()

	return name
}

// run is one generation in progress.
type run struct {
	tables *rules.Rules
	roll   Roller
	decide Decider
	by     traveller.DecidedBy
	log    *log
	char   *Character

	service  rules.Service
	drafted  bool
	dead     bool
	eligible int

	// rolledSocial is the Social Standing of p. 4, before anything moved
	// it. E011 turns on whether assessing at the end rather than at 18
	// changes the answer, and that cannot be known from the final value
	// alone.
	rolledSocial int

	// trained counts the results taken off each skills table, which the
	// decider is handed so it can see where the character is thin. It is
	// not on the Character: it is a decision aid, not part of the record.
	trained map[traveller.SkillTable]int
}

func (r *run) generate() error {
	r.rollProfile()

	err := r.enlist()
	if err != nil {
		return err
	}

	_, served := r.char.ServedIn()
	if !served {
		return r.assessTitle()
	}

	err = r.serve()
	if err != nil {
		return err
	}

	if !r.dead {
		err := r.musterOut()
		if err != nil {
			return err
		}

		r.pension()
	}

	return r.assessTitle()
}

// rollProfile rolls the six characteristics in the order p. 4 rolls them.
//
// Implements: Six 2D rolls, in order [1:4] in Characteristics and the record.
func (r *run) rollProfile() {
	r.log.step("characteristics", "p. 4")

	for _, c := range traveller.Characteristics {
		first, second := r.roll.TwoDice()

		r.char.Profile[c] = first + second

		if c == traveller.SocialStanding {
			r.rolledSocial = r.char.Profile[c]
		}

		seq := r.log.dice(c.String(), first, second)
		r.log.outcomef(seq, nil, "%v %d", c, first+second)
	}

	r.log.outcomef(0, nil, "UPP %s at age %v", r.char.Profile.UPP(), r.char.Age)
}
