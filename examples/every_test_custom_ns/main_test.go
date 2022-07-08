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

package every_test_custom_ns

import (
	"context"
	"fmt"
	"os"
	"sigs.k8s.io/e2e-framework/pkg/framework"
	"sigs.k8s.io/e2e-framework/pkg/framework/types"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var testenv env.Environment

func TestMain(m *testing.M) {
	testenv = env.New()

	// Specifying a run ID so that multiple runs wouldn't collide.
	runID := envconf.RandomName("ns", 4)

	testenv.Setup(
		// Step: creates kind cluster, propagate kind cluster object
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			name := envconf.RandomName("my-cluster", 16)
			cluster := framework.GetProviderGenerator("kind")()
			kubeconfig, err := cluster.Create(framework.WithName(name))
			if err != nil {
				return ctx, err
			}
			// stall a bit to allow most pods to come up
			time.Sleep(time.Second * 10)

			// update environment with kubecofig file
			cfg.WithKubeconfigFile(kubeconfig)

			// propagate cluster value
			return context.WithValue(ctx, "cluster", cluster), nil
		}).Finish(
		// Teardown func: delete kind cluster
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			cluster := ctx.Value("cluster").(types.ClusterProvider) // nil should be tested
			if err := cluster.Destroy(); err != nil {
				return ctx, err
			}
			return ctx, nil
		},
	)

	testenv.BeforeEachTest(func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) {
		return createNSForTest(ctx, cfg, t, runID)
	})
	testenv.AfterEachTest(func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) {
		return deleteNSForTest(ctx, cfg, t, runID)
	})

	os.Exit(testenv.Run(m))
}

// CreateNSForTest creates a random namespace with the runID as a prefix. It is stored in the context
// so that the deleteNSForTest routine can look it up and delete it.
func createNSForTest(ctx context.Context, cfg *envconf.Config, t *testing.T, runID string) (context.Context, error) {
	ns := envconf.RandomName(runID, 10)
	ctx = context.WithValue(ctx, nsKey(t), ns)

	t.Logf("Creating NS %v for test %v", ns, t.Name())
	nsObj := v1.Namespace{}
	nsObj.Name = ns
	return ctx, cfg.Client().Resources().Create(ctx, &nsObj)
}

// DeleteNSForTest looks up the namespace corresponding to the given test and deletes it.
func deleteNSForTest(ctx context.Context, cfg *envconf.Config, t *testing.T, runID string) (context.Context, error) {
	ns := fmt.Sprint(ctx.Value(nsKey(t)))
	t.Logf("Deleting NS %v for test %v", ns, t.Name())

	nsObj := v1.Namespace{}
	nsObj.Name = ns
	return ctx, cfg.Client().Resources().Delete(ctx, &nsObj)
}

func nsKey(t *testing.T) string {
	return "NS-for-%v" + t.Name()
}
