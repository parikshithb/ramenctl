// SPDX-FileCopyrightText: The RamenDR authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"fmt"
	"slices"

	"github.com/ramendr/ramenctl/pkg/report"
)

type Status string

const (
	Passed  = Status("passed")
	Failed  = Status("failed")
	Skipped = Status("skipped")
)

const (
	ValidateStep = "validate"
	SetupStep    = "setup"
	TestsStep    = "tests"
	CleanupStep  = "cleanup"
)

// A step is a test command step.
type Step struct {
	Name   string  `json:"name"`
	Status Status  `json:"status,omitempty"`
	Config *Config `json:"config,omitempty"`
	Items  []*Step `json:"items,omitempty"`
}

// Summary summaries a test run or clean.
type Summary struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// Test config supporting yaml marshaling, unlike ramen/e2e/types.TestConfig
// See https://github.com/RamenDR/ramen/issues/1968
type Config struct {
	Workload string `json:"workload"`
	Deployer string `json:"deployer"`
	PVCSpec  string `json:"pvcSpec"`
}

// Report created by test sub commands.
type Report struct {
	*report.Report
	Name    string  `json:"name"`
	Steps   []*Step `json:"steps"`
	Summary Summary `json:"summary"`
	Status  Status  `json:"status,omitempty"`
}

func newReport(commandName string) *Report {
	return &Report{
		Report: report.New(),
		Name:   commandName,
	}
}

// AddStep adds a step to the report.
func (r *Report) AddStep(step *Step) {
	if r.findStep(step.Name) != nil {
		panic(fmt.Sprintf("step %q exists", step.Name))
	}
	r.Steps = append(r.Steps, step)

	switch step.Status {
	case Passed, Skipped:
		if r.Status == "" {
			r.Status = Passed
		}
	case Failed:
		r.Status = Failed
	}
}

// AddTest records a completed test. A failed test mark the test step and the report as failed.
func (r *Report) AddTest(t *Test) {
	var step *Step

	// To make it easy to use, we create the tests step automaticlaly when adding the first test.
	if len(r.Steps) == 0 || r.Steps[len(r.Steps)-1].Name != TestsStep {
		step = &Step{Name: TestsStep}
		r.AddStep(step)
	} else {
		step = r.Steps[len(r.Steps)-1]
	}

	step.AddTest(t)

	switch t.Status {
	case Passed:
		r.Summary.Passed++
		if r.Status == "" {
			r.Status = Passed
		}
	case Failed:
		r.Summary.Failed++
		r.Status = Failed
	case Skipped:
		r.Summary.Skipped++
		if r.Status == "" {
			r.Status = Passed
		}
	}
}

// Equal return true if report is equal to other report.
func (r *Report) Equal(o *Report) bool {
	if o == nil {
		return false
	}
	if !r.Report.Equal(o.Report) {
		return false
	}
	if r.Name != o.Name {
		return false
	}
	if r.Status != o.Status {
		return false
	}
	if r.Summary != o.Summary {
		return false
	}
	return slices.EqualFunc(r.Steps, o.Steps, func(a *Step, b *Step) bool {
		return a.Equal(b)
	})
}

func (r *Report) findStep(name string) *Step {
	for _, step := range r.Steps {
		if step.Name == name {
			return step
		}
	}
	return nil
}

// AddTest records a completed test. A failed test marks the step as failed.
func (s *Step) AddTest(t *Test) {
	result := &Step{
		Name:   t.Name(),
		Config: t.Config,
		Status: t.Status,
		Items:  t.Steps,
	}

	s.Items = append(s.Items, result)

	switch t.Status {
	case Passed, Skipped:
		if s.Status == "" {
			s.Status = Passed
		}
	case Failed:
		s.Status = Failed
	}
}

// Equal return true if step is equal to other step.
func (s *Step) Equal(o *Step) bool {
	if o == nil {
		return false
	}
	if s.Name != o.Name {
		return false
	}
	if s.Status != o.Status {
		return false
	}
	if s.Config != o.Config {
		if s.Config == nil || o.Config == nil {
			return false
		}
		if *s.Config != *o.Config {
			return false
		}
	}
	return slices.EqualFunc(s.Items, o.Items, func(a *Step, b *Step) bool {
		if a == nil {
			return b == nil
		}
		return a.Equal(b)
	})
}
