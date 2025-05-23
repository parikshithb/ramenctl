// SPDX-FileCopyrightText: The RamenDR authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ramendr/ramen/e2e/types"
	"github.com/ramendr/ramenctl/pkg/command"
)

var testConfig = types.Config{
	PVCSpecs: []types.PVCSpecConfig{
		{Name: "block", StorageClassName: "block-storage"},
		{Name: "file", StorageClassName: "file-storage"},
	},
	Tests: []types.TestConfig{
		{Workload: "deploy", Deployer: "appset", PVCSpec: "block"},
		{Workload: "deploy", Deployer: "appset", PVCSpec: "file"},
		{Workload: "deploy", Deployer: "subscr", PVCSpec: "block"},
		{Workload: "deploy", Deployer: "subscr", PVCSpec: "file"},
		{Workload: "deploy", Deployer: "disapp", PVCSpec: "block"},
		{Workload: "deploy", Deployer: "disapp", PVCSpec: "file"},
	},
}

var testEnv = types.Env{
	Hub: types.Cluster{Name: "hub"},
	C1:  types.Cluster{Name: "c1"},
	C2:  types.Cluster{Name: "c2"},
}

var testOptions Options

var validateFailed = MockBackend{
	ValidateFunc: func(ctx types.Context) error {
		return errors.New("No validate for you!")
	},
}

var validateCanceled = MockBackend{
	ValidateFunc: func(ctx types.Context) error {
		return context.Canceled
	},
}

var setupFailed = MockBackend{
	SetupFunc: func(ctx types.Context) error {
		return errors.New("No setup for you!")
	},
}
var setupCanceled = MockBackend{
	SetupFunc: func(ctx types.Context) error {
		return context.Canceled
	},
}

var cleanupFailed = MockBackend{
	CleanupFunc: func(ctx types.Context) error {
		return errors.New("No cleanup for you!")
	},
}

var cleanupCanceled = MockBackend{
	CleanupFunc: func(ctx types.Context) error {
		return context.Canceled
	},
}

var failoverFailed = MockBackend{
	FailoverFunc: func(ctx types.TestContext) error {
		return errors.New("No failover for you!")
	},
}

var failoverCanceled = MockBackend{
	FailoverFunc: func(ctx types.TestContext) error {
		return context.Canceled
	},
}

var disappFailoverFailed = MockBackend{
	FailoverFunc: func(ctx types.TestContext) error {
		if ctx.Deployer().IsDiscovered() {
			return errors.New("No failover for you!")
		}
		return nil
	},
}

var unprotectFailed = MockBackend{
	UnprotectFunc: func(ctx types.TestContext) error {
		return errors.New("No unprotect for you!")
	},
}

var unprotectCanceled = MockBackend{
	UnprotectFunc: func(ctx types.TestContext) error {
		return context.Canceled
	},
}

var undeployFailed = MockBackend{
	UndeployFunc: func(ctx types.TestContext) error {
		return errors.New("No undeploy for you!")
	},
}

var undeployCanceled = MockBackend{
	UndeployFunc: func(ctx types.TestContext) error {
		return context.Canceled
	},
}

var runFlow = []string{"deploy", "protect", "failover", "relocate", "unprotect", "undeploy"}

func TestRunPassed(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &MockBackend{}, testOptions)

	if err := test.Run(); err != nil {
		t.Fatal(err)
	}

	checkReport(t, test.Report, Passed, Summary{Passed: len(testConfig.Tests)})
	if len(test.Report.Steps) != 3 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Passed)
	setup := test.Report.Steps[1]
	checkStep(t, setup, SetupStep, Passed)
	tests := test.Report.Steps[2]
	checkStep(t, tests, TestsStep, Passed)
	for i, tc := range testConfig.Tests {
		result := tests.Items[i]
		checkTest(t, result, tc, Passed, runFlow...)
	}
}

func TestRunValidateFailed(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &validateFailed, testOptions)

	if err := test.Run(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Failed, Summary{})
	if len(test.Report.Steps) != 1 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Failed)
}

func TestRunValidateCanceled(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &validateCanceled, testOptions)

	if err := test.Run(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Canceled, Summary{})
	if len(test.Report.Steps) != 1 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Canceled)
}

func TestRunSetupFailed(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &setupFailed, testOptions)

	if err := test.Run(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Failed, Summary{})
	if len(test.Report.Steps) != 2 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Passed)
	setup := test.Report.Steps[1]
	checkStep(t, setup, SetupStep, Failed)
}

func TestRunSetupCanceled(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &setupCanceled, testOptions)

	if err := test.Run(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Canceled, Summary{})
	if len(test.Report.Steps) != 2 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Passed)
	setup := test.Report.Steps[1]
	checkStep(t, setup, SetupStep, Canceled)
}

func TestRunTestsFailed(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &failoverFailed, testOptions)

	if err := test.Run(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Failed, Summary{Failed: len(testConfig.Tests)})
	if len(test.Report.Steps) != 3 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Passed)
	setup := test.Report.Steps[1]
	checkStep(t, setup, SetupStep, Passed)
	tests := test.Report.Steps[2]
	checkStep(t, tests, TestsStep, Failed)
	for i, tc := range testConfig.Tests {
		result := tests.Items[i]
		checkTest(t, result, tc, Failed, "deploy", "protect", "failover")
	}
}

func TestRunDisappFailed(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &disappFailoverFailed, testOptions)

	if err := test.Run(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Failed, Summary{Passed: 4, Failed: 2})
	if len(test.Report.Steps) != 3 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Passed)
	setup := test.Report.Steps[1]
	checkStep(t, setup, SetupStep, Passed)
	tests := test.Report.Steps[2]
	checkStep(t, tests, TestsStep, Failed)
	for i, tc := range testConfig.Tests {
		result := tests.Items[i]
		if tc.Deployer == "disapp" {
			checkTest(t, result, tc, Failed, "deploy", "protect", "failover")
		} else {
			checkTest(t, result, tc, Passed, runFlow...)
		}
	}
}

func TestRunTestsCanceled(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &failoverCanceled, testOptions)

	if err := test.Run(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Canceled, Summary{Canceled: len(testConfig.Tests)})
	if len(test.Report.Steps) != 3 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Passed)
	setup := test.Report.Steps[1]
	checkStep(t, setup, SetupStep, Passed)
	tests := test.Report.Steps[2]
	checkStep(t, tests, TestsStep, Canceled)
	for i, tc := range testConfig.Tests {
		result := tests.Items[i]
		checkTest(t, result, tc, Canceled, "deploy", "protect", "failover")
	}
}

func TestCleanPassed(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &MockBackend{}, testOptions)

	if err := test.Clean(); err != nil {
		t.Fatal(err)
	}

	checkReport(t, test.Report, Passed, Summary{Passed: 6})
	if len(test.Report.Steps) != 3 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Passed)
	tests := test.Report.Steps[1]
	checkStep(t, tests, TestsStep, Passed)
	for i, tc := range testConfig.Tests {
		result := tests.Items[i]
		checkTest(t, result, tc, Passed, "cleanup")
	}
	cleanup := test.Report.Steps[2]
	checkStep(t, cleanup, CleanupStep, Passed)
}

func TestCleanValidateFailed(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &validateFailed, testOptions)

	if err := test.Clean(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Failed, Summary{})
	if len(test.Report.Steps) != 1 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Failed)
}

func TestCleanValidateCanceled(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &validateCanceled, testOptions)

	if err := test.Clean(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Canceled, Summary{})
	if len(test.Report.Steps) != 1 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Canceled)
}

func TestCleanUnprotectFailed(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &unprotectFailed, testOptions)

	if err := test.Clean(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Failed, Summary{Failed: len(testConfig.Tests)})
	if len(test.Report.Steps) != 2 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Passed)
	tests := test.Report.Steps[1]
	checkStep(t, tests, TestsStep, Failed)
	for i, tc := range testConfig.Tests {
		result := tests.Items[i]
		checkTest(t, result, tc, Failed, "cleanup")
	}
}

func TestCleanUndeployFailed(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &undeployFailed, testOptions)

	if err := test.Clean(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Failed, Summary{Failed: len(testConfig.Tests)})
	if len(test.Report.Steps) != 2 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Passed)
	tests := test.Report.Steps[1]
	checkStep(t, tests, TestsStep, Failed)
	for i, tc := range testConfig.Tests {
		result := tests.Items[i]
		checkTest(t, result, tc, Failed, "cleanup")
	}
}

func TestCleanUnprotectCanceled(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &unprotectCanceled, testOptions)

	if err := test.Clean(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Canceled, Summary{Canceled: len(testConfig.Tests)})
	if len(test.Report.Steps) != 2 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Passed)
	tests := test.Report.Steps[1]
	checkStep(t, tests, TestsStep, Canceled)
	for i, tc := range testConfig.Tests {
		result := tests.Items[i]
		checkTest(t, result, tc, Canceled, "cleanup")
	}
}

func TestCleanUndeployCanceled(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &undeployCanceled, testOptions)

	if err := test.Clean(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Canceled, Summary{Canceled: len(testConfig.Tests)})
	if len(test.Report.Steps) != 2 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Passed)
	tests := test.Report.Steps[1]
	checkStep(t, tests, TestsStep, Canceled)
	for i, tc := range testConfig.Tests {
		result := tests.Items[i]
		checkTest(t, result, tc, Canceled, "cleanup")
	}
}

func TestCleanCleanupFailed(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &cleanupFailed, testOptions)

	if err := test.Clean(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Failed, Summary{Passed: len(testConfig.Tests)})
	if len(test.Report.Steps) != 3 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Passed)
	tests := test.Report.Steps[1]
	checkStep(t, tests, TestsStep, Passed)
	for i, tc := range testConfig.Tests {
		result := tests.Items[i]
		checkTest(t, result, tc, Passed, "cleanup")
	}
	cleanup := test.Report.Steps[2]
	checkStep(t, cleanup, CleanupStep, Failed)
}

func TestCleanCleanupCanceled(t *testing.T) {
	outputDir := t.TempDir()
	cmd, err := command.ForTest("test-run", &testConfig, &testEnv, outputDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Close()
	test := newCommand(cmd, &cleanupCanceled, testOptions)

	if err := test.Clean(); err == nil {
		t.Fatal("command did not fail")
	}

	checkReport(t, test.Report, Canceled, Summary{Passed: len(testConfig.Tests)})
	if len(test.Report.Steps) != 3 {
		t.Fatalf("unexpected steps %+v", test.Report.Steps)
	}
	validate := test.Report.Steps[0]
	checkStep(t, validate, ValidateStep, Passed)
	tests := test.Report.Steps[1]
	checkStep(t, tests, TestsStep, Passed)
	for i, tc := range testConfig.Tests {
		result := tests.Items[i]
		checkTest(t, result, tc, Passed, "cleanup")
	}
	cleanup := test.Report.Steps[2]
	checkStep(t, cleanup, CleanupStep, Canceled)
}

func checkReport(t *testing.T, report *Report, status Status, summary Summary) {
	if report.Status != status {
		t.Errorf("expected status %q, got %q", status, report.Status)
	}
	if report.Summary != summary {
		t.Errorf("expected summary %+v, got %+v", summary, report.Summary)
	}
	duration := totalDuration(report.Steps)
	if report.Duration != duration {
		t.Fatalf("expected duration %v, got %v", duration, report.Duration)
	}
}

func checkStep(t *testing.T, step *Step, name string, status Status) {
	if name != step.Name {
		t.Fatalf("expected step %q, got %q", name, step.Name)
	}
	if status != step.Status {
		t.Fatalf("expected status %q, got %q", status, step.Status)
	}
	// We cannot check duration since it may be zero on windows.
}

func checkTest(t *testing.T, test *Step, tc types.TestConfig, status Status, flow ...string) {
	name := fmt.Sprintf("%s-%s-%s", tc.Deployer, tc.Workload, tc.PVCSpec)
	if name != test.Name {
		t.Fatalf("expected step %q, got %q", name, test.Name)
	}
	if test.Config == nil || tc != *test.Config {
		t.Fatalf("expected config %+v, got %+v", tc, test.Config)
	}
	if status != test.Status {
		t.Fatalf("expected status %q, got %q", status, test.Status)
	}
	duration := totalDuration(test.Items)
	if test.Duration != duration {
		t.Fatalf("expected duration %v, got %v", duration, test.Duration)
	}
	if len(flow) != len(test.Items) {
		t.Fatalf("test %q steps %+v do not match flow %q", test.Name, test.Items, flow)
	}
	last := len(flow) - 1
	for i, name := range flow[:last] {
		checkStep(t, test.Items[i], name, Passed)
	}
	checkStep(t, test.Items[last], flow[last], test.Status)
}

func totalDuration(steps []*Step) float64 {
	var total float64
	for _, step := range steps {
		total += step.Duration
	}
	return total
}
