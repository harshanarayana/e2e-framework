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

package env

import (
	"context"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestEnv_New(t *testing.T) {
	e := newTestEnv()
	if e.ctx == nil {
		t.Error("missing default context")
	}

	if len(e.actions) != 0 {
		t.Error("unexpected actions found")
	}

	if e.cfg.Namespace() != "" {
		t.Error("unexpected envconfig.Namespace value")
	}
}

func TestEnv_APIMethods(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T) *testEnv
		roles map[actionRole]int
	}{
		{
			name: "empty actions",
			setup: func(t *testing.T) *testEnv {
				return newTestEnv()
			},
			roles: map[actionRole]int{roleSetup: 0, roleBeforeTest: 0, roleAfterTest: 0, roleFinish: 0},
		},
		{
			name: "setup actions",
			setup: func(t *testing.T) *testEnv {
				env := newTestEnv()
				env.Setup(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					return ctx, nil
				}).Setup(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					return ctx, nil
				})
				return env
			},
			roles: map[actionRole]int{roleSetup: 2, roleBeforeTest: 0, roleAfterTest: 0, roleFinish: 0},
		},
		{
			name: "before actions",
			setup: func(t *testing.T) *testEnv {
				env := newTestEnv()
				env.BeforeEachTest(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					return ctx, nil
				})
				return env
			},
			roles: map[actionRole]int{roleSetup: 0, roleBeforeTest: 1, roleAfterTest: 0, roleFinish: 0},
		},
		{
			name: "after actions",
			setup: func(t *testing.T) *testEnv {
				env := newTestEnv()
				env.AfterEachTest(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					return ctx, nil
				})
				return env
			},
			roles: map[actionRole]int{roleSetup: 0, roleBeforeTest: 0, roleAfterTest: 1, roleFinish: 0},
		},
		{
			name: "finish actions",
			setup: func(t *testing.T) *testEnv {
				env := newTestEnv()
				env.Finish(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					return ctx, nil
				})
				return env
			},
			roles: map[actionRole]int{roleSetup: 0, roleBeforeTest: 0, roleAfterTest: 0, roleFinish: 1},
		},
		{
			name: "all actions",
			setup: func(t *testing.T) *testEnv {
				env := newTestEnv()
				env.Setup(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					return ctx, nil
				}).BeforeEachTest(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					return ctx, nil
				}).AfterEachTest(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					return ctx, nil
				}).Finish(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					return ctx, nil
				})
				return env
			},
			roles: map[actionRole]int{roleSetup: 1, roleBeforeTest: 1, roleAfterTest: 1, roleFinish: 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := test.setup(t)
			for role, count := range test.roles {
				actual := len(env.getActionsByRole(role))
				if actual != count {
					t.Errorf("unexpected number of actions %d for role %d", actual, role)
				}
			}
		})
	}
}

func TestEnv_Test(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		setup    func(*testing.T, context.Context) int
		expected int
	}{
		{
			name:     "feature only",
			ctx:      context.TODO(),
			expected: 42,
			setup: func(t *testing.T, ctx context.Context) (val int) {
				env := newTestEnv()
				f := features.New("test-feat").Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
					val = 42
					return ctx
				})
				env.Test(t, f.Feature())
				return
			},
		},
		{
			name:     "filtered feature",
			ctx:      context.TODO(),
			expected: 42,
			setup: func(t *testing.T, ctx context.Context) (val int) {
				env := NewWithConfig(envconf.New().WithFeatureRegex("test-feat"))
				f := features.New("test-feat").Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
					val = 42
					return ctx
				})
				env.Test(t, f.Feature())

				env2 := NewWithConfig(envconf.New().WithFeatureRegex("skip-me"))
				f2 := features.New("test-feat-2").Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
					val = 42 + 1
					return ctx
				})
				env2.Test(t, f2.Feature())

				return
			},
		},
		{
			name:     "with before-test",
			ctx:      context.TODO(),
			expected: 86,
			setup: func(t *testing.T, ctx context.Context) (val int) {
				env := newTestEnv()
				env.BeforeEachTest(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					val = 44
					return ctx, nil
				})
				f := features.New("test-feat").Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
					val += 42
					return ctx
				})
				env.Test(t, f.Feature())
				return
			},
		},
		{
			name:     "with after-test",
			ctx:      context.TODO(),
			expected: 66,
			setup: func(t *testing.T, ctx context.Context) (val int) {
				env := newTestEnv()
				env.AfterEachTest(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					val -= 20
					return ctx, nil
				}).BeforeEachTest(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					val = 44
					return ctx, nil
				})
				f := features.New("test-feat").Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
					val += 42
					return ctx
				})
				env.Test(t, f.Feature())
				return
			},
		},
		{
			name:     "with before-after-test",
			ctx:      context.TODO(),
			expected: 44,
			setup: func(t *testing.T, ctx context.Context) (val int) {
				env := newTestEnv()
				env.AfterEachTest(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					val = 44
					return ctx, nil
				})
				f := features.New("test-feat").Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
					val = 42 + val
					return ctx
				})
				env.Test(t, f.Feature())
				return
			},
		},
		{
			name:     "filter assessment",
			ctx:      context.TODO(),
			expected: 45,
			setup: func(t *testing.T, ctx context.Context) (val int) {
				val = 42
				env := NewWithConfig(envconf.New().WithAssessmentRegex("add-*"))
				f := features.New("test-feat").
					Assess("add-one", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
						val++
						return ctx
					}).
					Assess("add-two", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
						val += 2
						return ctx
					}).
					Assess("take-one", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
						val--
						return ctx
					})
				env.Test(t, f.Feature())
				return
			},
		},
		{
			name:     "context value propagation with before, during, and after test",
			ctx:      context.TODO(),
			expected: 48,
			setup: func(t *testing.T, ctx context.Context) int {
				env, err := NewWithContext(context.WithValue(ctx, &ctxTestKeyInt{}, 44), envconf.New())
				if err != nil {
					t.Fatal(err)
				}
				env.BeforeEachTest(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					// update before test
					val, ok := ctx.Value(&ctxTestKeyInt{}).(int)
					if !ok {
						t.Fatal("context value was not int")
					}
					val += 2 // 46
					return context.WithValue(ctx, &ctxTestKeyInt{}, val), nil
				})
				env.AfterEachTest(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					// update after the test
					val, ok := ctx.Value(&ctxTestKeyInt{}).(int)
					if !ok {
						t.Fatal("context value was not int")
					}
					val++ // 48
					return context.WithValue(ctx, &ctxTestKeyInt{}, val), nil
				})
				f := features.New("test-feat").Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
					val, ok := ctx.Value(&ctxTestKeyInt{}).(int)
					if !ok {
						t.Fatal("context value was not int")
					}
					val++ // 47

					return context.WithValue(ctx, &ctxTestKeyInt{}, val)
				})

				env.Test(t, f.Feature())
				return env.(*testEnv).ctx.Value(&ctxTestKeyInt{}).(int)
			},
		},
		{
			name:     "no features specified",
			ctx:      context.TODO(),
			expected: 0,
			setup: func(t *testing.T, ctx context.Context) (val int) {
				env := newTestEnv()
				env.Test(t)
				return
			},
		},
		{
			name:     "multiple features",
			ctx:      context.TODO(),
			expected: 84,
			setup: func(t *testing.T, ctx context.Context) (val int) {
				env := newTestEnv()
				f1 := features.New("test-feat-1").
					Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
						val = 42
						return ctx
					})

				f2 := features.New("test-feat-2").
					Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
						val += 42
						return ctx
					})

				env.Test(t, f1.Feature(), f2.Feature())
				return
			},
		},
		{
			name:     "multiple features with before-after-test",
			ctx:      context.TODO(),
			expected: 66,
			setup: func(t *testing.T, ctx context.Context) (val int) {
				env := newTestEnv()
				env.AfterEachTest(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					val = 0
					return ctx, nil
				})
				env.AfterEachTest(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					val = 22 * 3
					return ctx, nil
				})
				f1 := features.New("test-feat-1").Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
					val = 42 + val
					return ctx
				})
				f2 := features.New("test-feat-2").Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
					val = 42 - 40
					return ctx
				})
				env.Test(t, f1.Feature(), f2.Feature())
				return
			},
		},

		{
			name:     "with before-and-after features",
			ctx:      context.TODO(),
			expected: 300,
			setup: func(t *testing.T, ctx context.Context) (val int) {
				env := newTestEnv()
				env.BeforeEachFeature(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					val += 20
					return ctx, nil
				}).AfterEachFeature(func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
					val -= 20
					return ctx, nil
				})
				f1 := features.New("test-feat").Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
					val *= 4
					return ctx
				})
				f2 := features.New("test-feat").Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
					val *= 4
					return ctx
				})
				env.Test(t, f1.Feature(), f2.Feature())
				return
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.setup(t, test.ctx)
			if result != test.expected {
				t.Error("unexpected result: ", result)
			}
		})
	}
}

// This test shows the full context propagation from
// environment setup functions (started in main_test.go) down to
// feature step functions.
func TestEnv_Context_Propagation(t *testing.T) {
	f := features.New("test-context-propagation").
		Assess("assess", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			val, ok := ctx.Value(&ctxTestKeyInt{}).(int)
			if !ok {
				t.Fatal("context value was not int")
			}
			val += 10 // 100
			return context.WithValue(ctx, &ctxTestKeyInt{}, val)
		})

	envForTesting.Test(t, f.Feature())
	// after test will dec by 1

	env, ok := envForTesting.(*testEnv)
	if !ok {
		t.Fatal("wrong type")
	}

	finalVal, ok := env.ctx.Value(&ctxTestKeyInt{}).(int)
	if !ok {
		t.Fatal("wrong type")
	}
	if finalVal != 99 {
		t.Fatalf("unexpected value %d", finalVal)
	}
}
