package chargen

import (
	"fmt"
	"slices"

	"github.com/philoserf/ctchargen/internal/dice"
	"github.com/philoserf/ctchargen/internal/rules"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// Ruleset names the text this tool implements. All page cites in a record's
// generation log are to these three books and nothing else.
//
// Implements: The note names its ruleset and the build that wrote it [ADR
// 0002].
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
// It returns nothing, so a watcher cannot change an event's values. An
// event's slices are the record's own, so a write through one changes that
// record and no other: no later generation, and no offer the engine checks
// an answer against.
//
// Implements: The procedure is shown between the questions [Generation record].
func WithObserver(watch func(traveller.Event)) Option {
	return func(r *run) { r.log.watch = watch }
}

// WithBuild names the build that writes the record (ADR 0002). Only the
// command passes it; a record generated in-process by a test passes none
// and carries none, for the reason Character.Build gives.
func WithBuild(build string) Option {
	return func(r *run) { r.char.Build = build }
}

// Auto walks Book 1's character generation procedure, pp. 4-25, as an auto
// run: policy answers every choice point, and the record names it and the
// version it answered as. It rolls dice of its own unless a roller is given.
//
// Implements: A note names the policy that answered it [Generation record].
func Auto(in Inputs, policy Policy, options ...Option) (*Character, error) {
	// The pointer is what marks an auto run: Guided passes nil.
	return generate(in, policy, &policy, options)
}

// Guided walks the same procedure as a guided run: decider answers every
// choice point on a person's behalf, and the record names no policy. Every
// choice is the player's, whatever decider stands for him.
//
// Implements: A guided run's choices are recorded as the player's, not the
// policy's [Choices].
func Guided(in Inputs, decider Decider, options ...Option) (*Character, error) {
	return generate(in, decider, nil, options)
}

// generate is the procedure behind both ways in. Which way a run came in is
// fixed before anything is asked, so nothing a caller passes can make the
// record name one answerer while another answered.
//
// The order within a term is the exposition's, not the worked example's
// (E002): survival, commission, promotion, skills, reenlistment, and then
// the aging round at the end of the term (E006). That order is what every
// fixture's script walks, so it is a fixed choice rather than an incidental
// one.
//
// Implements: A fixture generates one golden, and the same one twice [ADR
// 0001].
// Implements: The worked example reproduces [1:23–25].
func generate(in Inputs, decider Decider, policy *Policy, options []Option) (*Character, error) {
	tables, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("loading the rules: %w", err)
	}

	record := newLog()

	run := &run{
		tables: tables,
		roll:   dice.New(),
		log:    record,
		char:   &Character{Inputs: in, Policy: policy},
		atAge:  traveller.NewAge(traveller.StartingAge),
	}

	for _, option := range options {
		option(run)
	}

	run.decide = logging{to: decider, log: record}

	err = run.generate()
	if err != nil {
		return nil, err
	}

	run.char.Events = record.events
	run.char.Errata = record.stamped()

	return run.char, nil
}

// dead reports whether the character died, read off his departure rather
// than kept beside it (#368).
func (r *run) dead() bool { return traveller.Fatal(r.departure) }

// drafted reports whether the draft is what entered the character in his
// service, read off his enlistment by the fold ServedIn uses (#368). Every
// caller is past enlistment.
func (r *run) drafted() bool {
	var codec serviceCodec

	// A fold over a codec cannot fail: every case only assigns.
	_ = r.enlistment.Fold(&codec)

	return codec.drafted
}

// run is one generation in progress.
type run struct {
	tables *rules.Rules
	roll   Roller
	decide Decider
	log    *log
	char   *Character

	service  rules.Service
	eligible int

	// profile and skills are the character as the procedure has him so far,
	// which every DM and the gambling modifier read mid-run. They are the
	// engine's working state and not the record: each change to them is
	// logged as a fact carrying the value it left, and the finished
	// Character's Profile and Skills are read from those facts (#441).
	profile traveller.Profile
	skills  []traveller.Skill

	// rolledSocial is the Social Standing of p. 4, before anything moved
	// it. E011 turns on whether assessing at the end rather than at 18
	// changes the answer, and that cannot be known from the final value
	// alone. It is working state, as profile is: the Social Standing
	// RolledFact records the same value, and the finished character is read
	// from that.
	rolledSocial int

	// benefits is what mustering out has delivered so far, which the running
	// cash total, a repeat Travellers' Aid, a repeat weapon and a repeat ship
	// read. It is working state, as profile is: each change to it is logged
	// as a fact carrying the value it left, and the finished Character's
	// Benefits are read from those facts (#441).
	benefits Benefits

	// enlistment, terms, rank, atAge and departure are the career so far,
	// which the commission and promotion throws, mustering out, the pension
	// and the check for death read mid-run, and from which each age is
	// advanced. They are working state, as profile is: each change to them is
	// logged as a fact carrying the value it left, or for terms as the step
	// that opens one, and the finished Character's are read from the log
	// (#441).
	enlistment traveller.Enlistment
	terms      int
	rank       traveller.Rank
	atAge      traveller.Age
	departure  traveller.Departure

	// trained counts the results taken off each skills table, which the
	// decider is handed so it can see where the character is thin. It is
	// not on the Character: it is a decision aid, not part of the record.
	trained map[traveller.SkillTable]int
}

// generate walks one character through the procedure, from the six rolls to
// the title, and musters out only a character who lived.
//
// Settled by the page and not a reading: checked as a possible silence at
// milestone 0, and kept so it is not re-litigated as one. Whether a dead
// character musters out. P. 5 ends generation at a failed survival throw ("a
// new character must be generated"), so the departure p. 7's mustering out is
// written for never occurs.
func (r *run) generate() error {
	r.rollProfile()

	err := r.enlist()
	if err != nil {
		return err
	}

	_, served := servedIn(r.enlistment)
	if !served {
		return r.assessTitle()
	}

	err = r.serve()
	if err != nil {
		return err
	}

	if !r.dead() {
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
// Implements: Six 2D rolls, in order: values 2–12 initially [1:4].
func (r *run) rollProfile() {
	r.log.step("characteristics", "p. 4")

	for _, c := range traveller.Characteristics {
		first, second := twoDice(r.roll)

		r.profile[c] = first + second

		if c == traveller.SocialStanding {
			r.rolledSocial = r.profile[c]
		}

		seq := r.log.dice(c.String(), first, second)
		r.log.fact(seq, nil, traveller.RolledFact{Characteristic: c, Value: r.profile[c]})
	}

	r.log.fact(0, nil, traveller.StartedFact{Profile: r.profile, Age: r.atAge})
}

// addSkill records a skill, raising its level if it is already held, and
// returns the level it reached, which the fact logging it carries. P. 12:
// "Each time an already acquired skill is again acquired, the level of the
// skill is increased by 1."
//
// Implements: Skills accumulate as Skill-1, Skill-2, with no cap [1:12].
func (r *run) addSkill(name traveller.SkillName) int {
	for i := range r.skills {
		if r.skills[i].Name == name {
			r.skills[i].Level++

			return r.skills[i].Level
		}
	}

	r.skills = append(r.skills, traveller.Skill{Name: name, Level: 1})

	return 1
}

// has reports whether the character holds a skill at any level. Gambling
// expertise is what allows the +1 on Mustering Out Table 2 (p. 9).
func (r *run) has(name traveller.SkillName) bool {
	return slices.ContainsFunc(r.skills, func(s traveller.Skill) bool { return s.Name == name })
}
