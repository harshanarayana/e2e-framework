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

package helm

import (
	"context"
	"os"
	"path/filepath"
	"sigs.k8s.io/e2e-framework/pkg/external"
	"sigs.k8s.io/e2e-framework/pkg/klient/resources/conditions"
	"sigs.k8s.io/e2e-framework/pkg/klient/types"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/pkg/klient/wait"
)

var curDir, _ = os.Getwd()

func TestHelmChartRepoWorkflow(t *testing.T) {
	feature := features.New("Repo based helm chart workflow").
		Setup(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			manager := external.NewHelmManager(config.KubeconfigFile())
			err := manager.RunRepo(external.WithArgs("add", "nginx-stable", "https://helm.nginx.com/stable"))
			if err != nil {
				t.Fatal("failed to add nginx helm chart repo")
			}
			err = manager.RunRepo(external.WithArgs("update"))
			if err != nil {
				t.Fatal("failed to upgrade helm repo")
			}
			err = manager.RunInstall(external.WithName("nginx"), external.WithNamespace(namespace), external.WithReleaseName("nginx-stable/nginx-ingress"))
			if err != nil {
				t.Fatal("failed to install nginx Helm chart")
			}
			return ctx
		}).
		Assess("Deployment is running successfully", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			deployment := &appsv1.Deployment{
				ObjectMeta: v1.ObjectMeta{
					Name:      "nginx-nginx-ingress",
					Namespace: namespace,
				},
				Spec: appsv1.DeploymentSpec{},
			}
			err := wait.For(conditions.New(config.Client().Resources()).ResourceScaled(deployment, func(object types.Object) int32 {
				return object.(*appsv1.Deployment).Status.ReadyReplicas
			}, 1))
			if err != nil {
				t.Fatal("failed waiting for the Deployment to reach a ready state")
			}
			return ctx
		}).
		Assess("run Chart Tests", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			manager := external.NewHelmManager(config.KubeconfigFile())
			err := manager.RunTest(external.WithArgs("nginx"), external.WithNamespace(namespace))
			if err != nil {
				t.Fatal("failed waiting for the Deployment to reach a ready state")
			}
			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			manager := external.NewHelmManager(config.KubeconfigFile())
			err := manager.RunRepo(external.WithArgs("remove", "nginx-stable"))
			if err != nil {
				t.Fatal("cleanup of the helm repo failed")
			}
			return ctx
		}).Feature()

	testEnv.Test(t, feature)
}

func TestLocalHelmChartWorkflow(t *testing.T) {
	feature := features.New("Local Helm chart workflow").
		Setup(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			manager := external.NewHelmManager(config.KubeconfigFile())
			err := manager.RunInstall(external.WithName("example"), external.WithNamespace(namespace), external.WithChart(filepath.Join(curDir, "testdata", "example_chart")), external.WithWait(), external.WithTimeout("10m"))
			if err != nil {
				t.Fatal("failed to invoke helm install operation due to an error", err)
			}
			return ctx
		}).
		Assess("Deployment Is Running Successfully", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			deployment := &appsv1.Deployment{
				ObjectMeta: v1.ObjectMeta{
					Name:      "example",
					Namespace: namespace,
				},
				Spec: appsv1.DeploymentSpec{},
			}
			err := wait.For(conditions.New(config.Client().Resources()).ResourceScaled(deployment, func(object types.Object) int32 {
				return object.(*appsv1.Deployment).Status.ReadyReplicas
			}, 1))
			if err != nil {
				t.Fatal("failed waiting for the Deployment to reach a ready state")
			}
			return ctx
		}).
		Assess("run Helm Test Workflow", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			manager := external.NewHelmManager(config.KubeconfigFile())
			err := manager.RunTest(external.WithName("example"), external.WithNamespace(namespace))
			if err != nil {
				t.Fatal("failed to perform helm test operation to check if the chart deployment is good")
			}
			return ctx
		}).Feature()

	testEnv.Test(t, feature)
}
