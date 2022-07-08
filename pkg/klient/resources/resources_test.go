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

package resources

import (
	"context"
	"encoding/json"
	projectExample2 "sigs.k8s.io/e2e-framework/pkg/klient/resources/testdata/projectExample"
	types2 "sigs.k8s.io/e2e-framework/pkg/klient/types"
	"testing"
	"time"

	"github.com/vladimirvivien/gexe"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	log "k8s.io/klog/v2"
)

func TestCreate(t *testing.T) {
	res, err := New(cfg)
	if err != nil {
		t.Fatalf("Error creating new resources object: %v", err)
	}

	// create a namespace
	err = res.Create(context.TODO(), namespace)
	if err != nil {
		t.Error("error while creating namespace", err)
	}

	var nsObj corev1.Namespace
	err = res.Get(context.TODO(), namespace.Name, namespace.Name, &nsObj)
	if err != nil {
		t.Error("error while getting the namespace", err)
	}

	if nsObj.Name != namespace.Name {
		t.Error("namespace name mismatch, expected : ", namespace.Name, "obtained :", nsObj.Name)
	}
}

func TestRes(t *testing.T) {
	res, err := New(cfg)
	if err != nil {
		t.Fatalf("Error creating new resources object: %v", err)
	}

	err = res.Create(context.TODO(), dep)
	if err != nil {
		t.Error("error while creating deployment", err)
	}

	actual, err := clientset.AppsV1().Deployments(namespace.Name).Get(context.TODO(), dep.Name, metav1.GetOptions{})
	if err != nil {
		t.Error("error while getting the deployment details", err)
	}

	if actual == dep {
		log.Info("deployment found", dep.Name)
	}

	var depObj appsv1.Deployment
	err = res.Get(context.TODO(), dep.Name, namespace.Name, &depObj)
	if err != nil {
		t.Error("error while getting the deployment", err)
	}

	if depObj.Name != dep.Name {
		t.Error("deployment name mismatch, expected : ", dep.Name, "obtained :", depObj.Name)
	}
}

func TestResNoConfig(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Error("expected error while invoking Res without k8s config")
	}
}

func TestResInvalidConfig(t *testing.T) {
	cfg := &rest.Config{
		Host: "invalid-host",
	}

	_, err := New(cfg)
	if err == nil {
		t.Error("expected panic while invoking Res with invalid k8s config")
	}
}

func TestUpdate(t *testing.T) {
	res, err := New(cfg)
	if err != nil {
		t.Fatalf("Error creating new resources object: %v", err)
	}

	depActual := getDeployment("update-test-dep-name")

	err = res.Create(context.TODO(), depActual)
	if err != nil {
		t.Error("error while creating deployment", err)
	}

	depUpdated := depActual
	depUpdated.ObjectMeta.Labels["test-key"] = "test-val"

	err = res.Update(context.TODO(), depUpdated)
	if err != nil {
		t.Error("error while updating deployment", err)
	}

	var depObj appsv1.Deployment
	err = res.Get(context.TODO(), depUpdated.Name, namespace.Name, &depObj)
	if err != nil {
		t.Error("error while getting the deployment", err)
	}

	val, ok := depObj.Labels["test-key"]
	if !ok {
		t.Error("deployment not updated")
	} else if val != "test-val" {
		t.Error("deployment label value mismatch, expected : ", "test-val", "obtained :", val)
	}
}

func TestDelete(t *testing.T) {
	res, err := New(cfg)
	if err != nil {
		t.Fatalf("Error creating new resources object: %v", err)
	}

	depActual := getDeployment("delete-test-dep-name")

	err = res.Create(context.TODO(), depActual)
	if err != nil {
		t.Error("error while creating deployment", err)
	}

	err = res.Delete(context.TODO(), depActual)
	if err != nil {
		t.Error("error while deleting deployment", err)
	}
}

func TestList(t *testing.T) {
	res, err := New(cfg)
	if err != nil {
		t.Fatalf("Error creating new resources object: %v", err)
	}

	deps := &appsv1.DeploymentList{}
	err = res.List(context.TODO(), deps)
	if err != nil {
		t.Error("error while getting the deployment", err)
	}

	if deps.Items == nil {
		t.Error("error while getting the list of deployments", err)
	}

	hasDep := false
	for _, item := range deps.Items {
		if item.Name == dep.Name && item.Namespace == dep.Namespace {
			hasDep = true
			break
		}
	}

	if !hasDep {
		t.Error("there are no deployment exist", hasDep)
	}
}

func TestPatch(t *testing.T) {
	res, err := New(cfg)
	if err != nil {
		t.Fatalf("Error creating new resources object: %v", err)
	}

	mergePatch, err := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				"ping": "pong",
			},
		},
	})
	if err != nil {
		t.Error("error while json marshalling", err)
	}

	err = res.Patch(context.Background(), dep, types2.Patch{PatchType: types.StrategicMergePatchType, Data: mergePatch})
	if err != nil {
		t.Error("error while patching the deployment", err)
	}

	obj := &appsv1.Deployment{}
	err = res.Get(context.Background(), dep.Name, dep.Namespace, obj)
	if err != nil {
		t.Error("error while getting patched deployment", err)
	}

	if obj.Annotations["ping"] != "pong" {
		t.Error("resource patch not applied correctly.")
	}
}

func TestListAllPods(t *testing.T) {
	res, err := New(cfg)
	if err != nil {
		t.Fatalf("Error creating new resources object: %v", err)
	}

	pods := &corev1.PodList{}
	err = res.List(context.TODO(), pods)
	if err != nil {
		t.Error("error while getting the deployment", err)
	}

	if pods.Items == nil {
		t.Error("error while getting the list of deployments", err)
	}

	t.Logf("pod list contains %d pods", len(pods.Items))
}

func TestGetCRDs(t *testing.T) {
	res, err := New(cfg)
	if err != nil {
		t.Fatalf("Error creating new resources object: %v", err)
	}

	// Register type for the API server.
	e := gexe.New()
	p := e.RunProc(`kubectl apply -f ./testdata/projectExample/resourcedefinition.yaml`)
	if p.Err() != nil {
		t.Fatalf("Failed to register CRD: %v %v", p.Err(), p.Result())
	}
	// Sometimes CRDs need just a bit of time before being ready to use.
	time.Sleep(5 * time.Second)

	// Create one
	p = e.RunProc(`kubectl apply -f ./testdata/projectExample/project.yaml`)
	if p.Err() != nil {
		t.Fatalf("Failed to create a CRD via yaml: %v %v", p.Err(), p.Result())
	}

	// See that we can't list it because we don't know the type.
	ps := &projectExample2.ProjectList{}
	err = res.List(context.TODO(), ps)
	if err == nil {
		t.Error("Expected error while listing custom resources before adding it to scheme, but got none")
	}

	// Register type with klient.
	if err := projectExample2.AddToScheme(res.GetScheme()); err != nil {
		t.Fatalf("Failed to add to resource scheme: %v", err)
	}

	// See that we can after registering it.
	err = res.List(context.TODO(), ps)
	if err != nil {
		t.Error("error while listing custom resources", err)
	}
}
