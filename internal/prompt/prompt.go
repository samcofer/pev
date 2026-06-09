// Package prompt wraps AlecAivazis/survey/v2 with three behavioral modes:
//
//   - interactive (default, requires a TTY) — show the prompt, surface the
//     discovered default, log both question and answer
//   - yes — auto-accept the discovered default, log "(auto-accepted)"
//   - non-interactive — refuse to prompt; if a value isn't already supplied
//     the caller gets back the empty string and the engine SKIPs the check
//     with a "missing or invalid input" reason
//
// Every prompt logs both question text and final answer through logrus so the
// pev log file captures the full Q&A trail (mirrors wbi convention).
package prompt

import (
	"errors"
	"os"

	"github.com/AlecAivazis/survey/v2"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
)

// Mode controls how prompts behave at runtime.
type Mode int

const (
	// ModeInteractive shows survey prompts. The default.
	ModeInteractive Mode = iota
	// ModeYes silently accepts the default for every prompt.
	ModeYes
	// ModeNonInteractive returns the default without showing anything; an
	// empty default returns "" so the engine SKIPs the dependent check.
	ModeNonInteractive
)

// ErrSkipped is returned when the user explicitly typed the magic word "skip"
// at a prompt. Callers translate this into a "missing input" SKIP downstream.
var ErrSkipped = errors.New("skipped by user")

// Driver is the prompt frontend; tests inject a fake Driver to avoid TTY work.
type Driver interface {
	Input(question, defaultValue string) (string, error)
	Confirm(question string, defaultValue bool) (bool, error)
	Select(question string, options []string, defaultValue string) (string, error)
	MultiSelect(question string, options, defaultValues []string) ([]string, error)
	// Password reads a secret with terminal echo turned off. Q/A logs
	// emit the question and the literal "(redacted)" so the secret
	// never lands in pev-log-*.log. In yes / non-interactive mode the
	// returned value is empty — there is no sensible default for a
	// secret, so callers should treat empty as "skip" and SKIP the
	// dependent check.
	Password(question string) (string, error)
}

// surveyDriver is the production Driver implementation backed by survey/v2.
type surveyDriver struct{ mode Mode }

// New returns a Driver. Pass mode based on --non-interactive / --yes flags.
// If mode is ModeInteractive but stdin/stdout aren't a TTY, the driver
// silently downgrades to ModeYes — surveys would error out otherwise and
// we want pev to remain useful when piped or in CI.
func New(mode Mode) Driver {
	if mode == ModeInteractive && !isTerminal() {
		log.Info("stdin or stdout is not a terminal; using --yes mode for prompts")
		mode = ModeYes
	}
	return &surveyDriver{mode: mode}
}

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func logQA(q string, a interface{}) {
	log.WithFields(log.Fields{"question": q, "answer": a}).Info("prompt")
}

// Input asks for free-form text. The discovered defaultValue is shown to the
// user as the inline default and pre-filled if they hit enter.
func (s *surveyDriver) Input(question, defaultValue string) (string, error) {
	switch s.mode {
	case ModeYes, ModeNonInteractive:
		logQA(question, defaultValue)
		return defaultValue, nil
	}
	out := defaultValue
	p := &survey.Input{Message: question, Default: defaultValue}
	if err := survey.AskOne(p, &out); err != nil {
		return "", err
	}
	if out == "skip" {
		logQA(question, "<skipped>")
		return "", ErrSkipped
	}
	logQA(question, out)
	return out, nil
}

// Confirm asks a yes/no question.
func (s *surveyDriver) Confirm(question string, defaultValue bool) (bool, error) {
	switch s.mode {
	case ModeYes, ModeNonInteractive:
		logQA(question, defaultValue)
		return defaultValue, nil
	}
	out := defaultValue
	p := &survey.Confirm{Message: question, Default: defaultValue}
	if err := survey.AskOne(p, &out); err != nil {
		return false, err
	}
	logQA(question, out)
	return out, nil
}

// Select asks the user to pick one option from a list. defaultValue must be
// one of options or it is silently ignored.
func (s *surveyDriver) Select(question string, options []string, defaultValue string) (string, error) {
	switch s.mode {
	case ModeYes, ModeNonInteractive:
		logQA(question, defaultValue)
		return defaultValue, nil
	}
	out := defaultValue
	p := &survey.Select{Message: question, Options: options, Default: defaultValue}
	if err := survey.AskOne(p, &out); err != nil {
		return "", err
	}
	logQA(question, out)
	return out, nil
}

// Password asks for a secret, displaying asterisks instead of the typed
// characters. The Q/A log writes "(redacted)" rather than the actual
// secret. In yes / non-interactive mode the returned value is empty: a
// password has no sensible auto-default, so callers should treat empty
// as "skip" and SKIP the dependent check.
func (s *surveyDriver) Password(question string) (string, error) {
	switch s.mode {
	case ModeYes, ModeNonInteractive:
		logQA(question, "(redacted)")
		return "", nil
	}
	out := ""
	p := &survey.Password{Message: question}
	if err := survey.AskOne(p, &out); err != nil {
		return "", err
	}
	logQA(question, "(redacted)")
	return out, nil
}

// MultiSelect lets the user toggle several options. defaultValues seed the
// initial selection; "All / None" shortcuts are removed to keep selections
// deliberate (matches wbi).
func (s *surveyDriver) MultiSelect(question string, options, defaultValues []string) ([]string, error) {
	switch s.mode {
	case ModeYes, ModeNonInteractive:
		logQA(question, defaultValues)
		return defaultValues, nil
	}
	out := append([]string{}, defaultValues...)
	p := &survey.MultiSelect{Message: question, Options: options, Default: defaultValues}
	if err := survey.AskOne(p, &out, survey.WithRemoveSelectAll(), survey.WithRemoveSelectNone()); err != nil {
		return nil, err
	}
	logQA(question, out)
	return out, nil
}
