package argo_test

import (
	"context"
	"slices"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-api/internal/argo"
	"github.com/super-phenix/superphenix/internal/superphenix-api/internal/argo/testhelper"

	spxId "github.com/super-phenix/superphenix/pkg/superphenix-id"

	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

const appProjectNamespace = "superphenix-system"

var wildcard = []metav1.GroupKind{{Group: "*", Kind: "*"}}

func appProject(name string, labels map[string]string, whitelist []metav1.GroupKind) *v1alpha1.AppProject {
	return &v1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: appProjectNamespace, Labels: labels},
		Spec:       v1alpha1.AppProjectSpec{ClusterResourceWhitelist: whitelist},
	}
}

func newProjectClient(t *testing.T, opts argo.Options, projects ...*v1alpha1.AppProject) *argo.Client {
	t.Helper()
	fake := testhelper.NewFakeClientset()
	for _, p := range projects {
		if err := fake.Tracker().Add(p); err != nil {
			t.Fatalf("failed to add object to tracker: %v", err)
		}
	}
	opts.AppProjectNamespace = appProjectNamespace
	return argo.NewClient(fake.ArgoprojV1alpha1(), k8sfake.NewClientset(), opts)
}

func getWhitelist(t *testing.T, c *argo.Client, name string) []metav1.GroupKind {
	t.Helper()
	p, err := c.Apps().AppProjects(appProjectNamespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("AppProject %s not found: %v", name, err)
	}
	return p.Spec.ClusterResourceWhitelist
}

func TestDefaultClusterResourceWhitelistIsRestricted(t *testing.T) {
	t.Parallel()

	for _, gk := range argo.DefaultClusterResourceWhitelist {
		if gk.Group == "*" || gk.Kind == "*" {
			t.Errorf("wildcard in default whitelist: %+v", gk)
		}
		switch gk.Group {
		case "rbac.authorization.k8s.io", "admissionregistration.k8s.io", "apiextensions.k8s.io", "storage.k8s.io":
			t.Errorf("privileged group in default whitelist: %+v", gk)
		}
		if gk.Group == "" && gk.Kind != "Namespace" {
			t.Errorf("unexpected core kind in default whitelist: %+v", gk)
		}
	}
}

func TestCreateAppUsesRestrictedAppProject(t *testing.T) {
	t.Parallel()

	c := newProjectClient(t, argo.Options{})
	if err := c.CreateApp(context.Background(), createInfo("kaas-1", argo.AppSource{RepoURL: "https://example.test"})); err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}

	got := getWhitelist(t, c, "spx-"+testProjectId)
	if !slices.Equal(got, argo.DefaultClusterResourceWhitelist) {
		t.Errorf("whitelist = %v, want the default", got)
	}
}

func TestCreateAppUsesConfiguredWhitelist(t *testing.T) {
	t.Parallel()

	configured := []metav1.GroupKind{{Group: "kubeovn.io", Kind: "Subnet"}}
	c := newProjectClient(t, argo.Options{ClusterResourceWhitelist: configured})
	if err := c.CreateApp(context.Background(), createInfo("kaas-1", argo.AppSource{RepoURL: "https://example.test"})); err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}

	got := getWhitelist(t, c, "spx-"+testProjectId)
	if !slices.Equal(got, configured) {
		t.Errorf("whitelist = %v, want %v", got, configured)
	}
}

func TestReconcileAppProjects(t *testing.T) {
	t.Parallel()

	projectLabel := map[string]string{spxId.SpxLabelProjectID: "spx-" + testProjectId}
	c := newProjectClient(t, argo.Options{},
		appProject("spx-legacy", projectLabel, wildcard),
		appProject("spx-aligned", projectLabel, argo.DefaultClusterResourceWhitelist),
		appProject("default", nil, wildcard),
		appProject("system", projectLabel, wildcard),
	)

	updated, err := c.ReconcileAppProjects(context.Background())
	if err != nil {
		t.Fatalf("ReconcileAppProjects() error = %v", err)
	}
	if updated != 1 {
		t.Errorf("updated = %d, want 1", updated)
	}

	if got := getWhitelist(t, c, "spx-legacy"); !slices.Equal(got, argo.DefaultClusterResourceWhitelist) {
		t.Errorf("spx-legacy whitelist = %v, want the default", got)
	}
	// AppProjects that are not project AppProjects are left untouched.
	for _, name := range []string{"default", "system"} {
		if got := getWhitelist(t, c, name); !slices.Equal(got, wildcard) {
			t.Errorf("%s whitelist = %v, want it untouched", name, got)
		}
	}
}
