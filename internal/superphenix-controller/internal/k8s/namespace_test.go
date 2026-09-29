package k8s

import (
	"context"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-controller/pkg/api/utils"
	"github.com/super-phenix/superphenix/internal/superphenix-controller/pkg/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateNamespaceSetsPodSecurityLabels(t *testing.T) {
	old := config.K8sClient
	config.K8sClient = fake.NewClientset()
	t.Cleanup(func() { config.K8sClient = old })

	const projectId = "22222222-2222-2222-2222-222222222222"
	if err := CreateNamespaceIfNotExists(context.Background(), "11111111-1111-1111-1111-111111111111", projectId); err != nil {
		t.Fatal(err)
	}

	ns, err := config.K8sClient.CoreV1().Namespaces().Get(context.Background(), utils.GetNamespace(projectId), metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
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
