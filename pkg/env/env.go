/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package env exposes types to create type `Environment` used to run
// feature tests.
package env

import (
	"context"
	"fmt"
	"log"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/pkg/internal/types"
)

type (
	Environment = types.Environment
	Func        = types.EnvFunc

	actionRole uint8
)

type testEnv struct {
	ctx     context.Context
	cfg     *envconf.Config
	actions []action
}

// New creates a test environment with no config attached.
func New() types.Environment {
	return newTestEnv()
}

// NewWithConfig creates an environment using an Environment Configuration value
func NewWithConfig(cfg *envconf.Config) types.Environment {
	env := newTestEnv()
	env.cfg = cfg
	return env
}

// NewWithContext creates a new environment with the provided context and config.
func NewWithContext(ctx context.Context, cfg *envconf.Config) (types.Environment, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("environment config is nil")
	}
	return &testEnv{ctx: ctx, cfg: cfg}, nil
}

func newTestEnv() *testEnv {
	return &testEnv{
		ctx: context.Background(),
		cfg: envconf.New(),
	}
}

// WithContext returns a new environment with the context set to ctx.
// Argument ctx cannot be nil
func (e *testEnv) WithContext(ctx context.Context) types.Environment {
	if ctx == nil {
		panic("nil context") // this should never happen
	}
	env := &testEnv{
		ctx: ctx,
		cfg: e.cfg,
	}
	env.actions = append(env.actions, e.actions...)
	return env
}

// Setup registers environment operations that are executed once
// prior to the environment being ready and prior to any test.
func (e *testEnv) Setup(funcs ...Func) types.Environment {
	if len(funcs) == 0 {
		return e
	}
	e.actions = append(e.actions, action{role: roleSetup, funcs: funcs})
	return e
}

// BeforeEachTest registers environment funcs that are executed
// before each Env.Test(...)
func (e *testEnv) BeforeEachTest(funcs ...Func) types.Environment {
	if len(funcs) == 0 {
		return e
	}
	e.actions = append(e.actions, action{role: roleBeforeTest, funcs: funcs})
	return e
}

// BeforeEachFeature registers step functions that are executed
// before each Feature is tested during env.Test call.
func (e *testEnv) BeforeEachFeature(funcs ...Func) types.Environment {
	if len(funcs) == 0 {
		return e
	}
	e.actions = append(e.actions, action{role: roleBeforeFeature, funcs: funcs})
	return e
}

// AfterEachFeature registers step functions that are executed
// after each feature is tested during an env.Test call.
func (e *testEnv) AfterEachFeature(funcs ...Func) types.Environment {
	if len(funcs) == 0 {
		return e
	}
	e.actions = append(e.actions, action{role: roleAfterFeature, funcs: funcs})
	return e
}

// AfterEachTest registers environment funcs that are executed
// after each Env.Test(...).
func (e *testEnv) AfterEachTest(funcs ...Func) types.Environment {
	if len(funcs) == 0 {
		return e
	}
	e.actions = append(e.actions, action{role: roleAfterTest, funcs: funcs})
	return e
}

// Test executes a feature test from within a TestXXX function.
//
// Feature setups and teardowns are executed at the same *testing.T
// contextual level as the "test" that invoked this method. Assessments
// are executed as a subtests of the feature.  This approach allows
// features/assessments to be filtered using go test -run flag.
//
// Feature tests will have access to and able to update the context
// passed to it.
//
// BeforeTest and AfterTest operations are executed before and after
// the feature is tested respectively.
func (e *testEnv) Test(t *testing.T, testFeatures ...types.Feature) {
	if e.ctx == nil {
		panic("context not set") // something is terribly wrong.
	}

	if len(testFeatures) == 0 {
		t.Log("No test testFeatures provided, skipping test")
		return
	}

	// execute the beforeTest functions
	beforeTestActions := e.getBeforeTestActions()
	var err error
	for _, action := range beforeTestActions {
		if e.ctx, err = action.run(e.ctx, e.cfg); err != nil {
			t.Fatalf("BeforeEachTest failure: %s", err)
		}
	}

	// execute each feature
	beforeFeatureActions := e.getBeforeFeatureActions()
	afterFeatureActions := e.getAfterFeatureActions()
	for _, feature := range testFeatures {
		// execute beforeFeature actions
		for _, action := range beforeFeatureActions {
			if e.ctx, err = action.run(e.ctx, e.cfg); err != nil {
				t.Fatalf("BeforeEachTest failure: %s", err)
			}
		}

		// execute feature test
		e.ctx = e.execFeature(e.ctx, t, feature)

		// execute beforeFeature actions
		for _, action := range afterFeatureActions {
			if e.ctx, err = action.run(e.ctx, e.cfg); err != nil {
				t.Fatalf("BeforeEachTest failure: %s", err)
			}
		}
	}

	// execute afterTest functions
	afterTestActions := e.getAfterTestActions()
	for _, action := range afterTestActions {
		if e.ctx, err = action.run(e.ctx, e.cfg); err != nil {
			t.Fatalf("AfterEachTest failure: %s", err)
		}
	}
}

// Finish registers funcs that are executed at the end of the
// test suite.
func (e *testEnv) Finish(funcs ...Func) types.Environment {
	if len(funcs) == 0 {
		return e
	}

	e.actions = append(e.actions, action{role: roleFinish, funcs: funcs})
	return e
}

// Run is to launch the test suite from a TestMain function.
// It will run m.Run() and exercise all test functions in the
// package.  This method will all Env.Setup operations prior to
// starting the tests and run all Env.Finish operations after
// before completing the suite.
//
func (e *testEnv) Run(m *testing.M) int {
	if e.ctx == nil {
		panic("context not set") // something is terribly wrong.
	}

	setups := e.getSetupActions()
	// fail fast on setup, upon err exit
	var err error
	for _, setup := range setups {
		// context passed down to each setup
		if e.ctx, err = setup.run(e.ctx, e.cfg); err != nil {
			log.Fatal(err)
		}
	}

	exitCode := m.Run() // exec test suite

	finishes := e.getFinishActions()
	// attempt to gracefully clean up.
	// Upon error, log and continue.
	for _, fin := range finishes {
		// context passed down to each finish step
		if e.ctx, err = fin.run(e.ctx, e.cfg); err != nil {
			log.Println(err)
		}
	}

	return exitCode
}

func (e *testEnv) getActionsByRole(r actionRole) []action {
	if e.actions == nil {
		return nil
	}

	var result []action
	for _, a := range e.actions {
		if a.role == r {
			result = append(result, a)
		}
	}

	return result
}

func (e *testEnv) getSetupActions() []action {
	return e.getActionsByRole(roleSetup)
}

func (e *testEnv) getBeforeTestActions() []action {
	return e.getActionsByRole(roleBeforeTest)
}

func (e *testEnv) getBeforeFeatureActions() []action {
	return e.getActionsByRole(roleBeforeFeature)
}

func (e *testEnv) getAfterFeatureActions() []action {
	return e.getActionsByRole(roleAfterFeature)
}

func (e *testEnv) getAfterTestActions() []action {
	return e.getActionsByRole(roleAfterTest)
}

func (e *testEnv) getFinishActions() []action {
	return e.getActionsByRole(roleFinish)
}

func (e *testEnv) execFeature(ctx context.Context, t *testing.T, f types.Feature) context.Context {
	featName := f.Name()

	// feature-level subtest
	t.Run(featName, func(t *testing.T) {
		if e.cfg.FeatureRegex() != nil && !e.cfg.FeatureRegex().MatchString(featName) {
			t.Skipf(`Skipping feature "%s": name not matched`, featName)
		}

		// setups run at feature-level
		setups := features.GetStepsByLevel(f.Steps(), types.LevelSetup)
		for _, setup := range setups {
			ctx = setup.Func()(ctx, t, e.cfg)
		}

		// assessments run as feature/assessment sub level
		assessments := features.GetStepsByLevel(f.Steps(), types.LevelAssess)

		for _, assess := range assessments {
			t.Run(assess.Name(), func(t *testing.T) {
				if e.cfg.AssessmentRegex() != nil && !e.cfg.AssessmentRegex().MatchString(assess.Name()) {
					t.Skipf(`Skipping assessment "%s": name not matched`, assess.Name())
				}
				ctx = assess.Func()(ctx, t, e.cfg)
			})
		}

		// teardowns run at feature-level
		teardowns := features.GetStepsByLevel(f.Steps(), types.LevelTeardown)
		for _, teardown := range teardowns {
			ctx = teardown.Func()(ctx, t, e.cfg)
		}
	})

	return ctx
}
