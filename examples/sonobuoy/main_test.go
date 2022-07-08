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

package sonobuoy

import (
	"context"
	"os"
	"sigs.k8s.io/e2e-framework/pkg/framework"
	"sigs.k8s.io/e2e-framework/pkg/framework/types"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var testenv env.Environment

func TestMain(m *testing.M) {
	testenv = env.New()
	if os.Getenv("SONOBUOY") == "true" {
		// Empty string results in in-cluster config. Perfect if running as a Sonobuoy plugin.
		testenv = env.NewInClusterConfig()
	} else {
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
	}
	os.Exit(testenv.Run(m))
}
