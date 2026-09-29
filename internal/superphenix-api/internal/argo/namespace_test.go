package argo_test

import (
	"context"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-api/internal/argo"
	"github.com/super-phenix/superphenix/internal/superphenix-api/internal/argo/testhelper"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestCreateAppNamespaceHasPodSecurityLabels(t *testing.T) {
	t.Parallel()

	k8s := k8sfake.NewClientset()
	c := argo.NewClient(testhelper.NewFakeClientset().ArgoprojV1alpha1(), k8s, argo.Options{AppProjectNamespace: "superphenix-system"})
	if err := c.CreateApp(context.Background(), createInfo("kaas-1", argo.AppSource{RepoURL: "https://example.test"})); err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}

	ns, err := k8s.CoreV1().Namespaces().Get(context.Background(), c.Namespace(testProjectId), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("namespace not created: %v", err)
	}
	want := map[string]string{
		"pod-security.kubernetes.io/enforce": "baseline",
		"pod-security.kubernetes.io/audit":   "restricted",
		"pod-security.kubernetes.io/warn":    "restricted",
	}
	for k, v := range want {
		if ns.Labels[k] != v {
			t.Errorf("label %s = %q, want %q", k, ns.Labels[k], v)
		}
	}
}
