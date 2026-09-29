package kaas

import (
	"context"
	"strings"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-controller/pkg/config"

	spxId "github.com/super-phenix/superphenix/pkg/superphenix-id"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestDeleteGroupValidatesNamesAndScopesToTheCluster(t *testing.T) {
	cluster := newCluster()
	labels := cluster.GetLabels()
	labels[spxId.SpxLabelResourceLocalID] = "local-cluster"
	cluster.SetLabels(labels)

	t.Run("invalid group name", func(t *testing.T) {
		client := setFakeDynamicClient(t, cluster)
		for _, group := range []string{"a,b=c", "a b", "grp!=x", ""} {
			err := DeleteGroup(context.Background(), testNamespace, testClusterId, []string{group})
			if !apierrors.IsBadRequest(err) {
				t.Errorf("DeleteGroup(%q) error = %v, want BadRequest", group, err)
			}
		}
		for _, action := range client.Actions() {
			if action.GetVerb() == "delete-collection" {
				t.Errorf("unexpected delete for an invalid group name: %v", action)
			}
		}
	})

	t.Run("selector scoped to the cluster", func(t *testing.T) {
		client := setFakeDynamicClient(t, cluster)
		client.PrependReactor("delete-collection", "*", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, nil
		})
		if err := DeleteGroup(context.Background(), testNamespace, testClusterId, []string{"workers"}); err != nil {
			t.Fatalf("DeleteGroup() error = %v", err)
		}
		found := false
		for _, action := range client.Actions() {
			if dc, ok := action.(k8stesting.DeleteCollectionAction); ok {
				found = true
				selector := dc.GetListRestrictions().Labels.String()
				if !strings.Contains(selector, ClusterLabelKey+"="+testClusterId) {
					t.Errorf("selector %q is not scoped to the cluster", selector)
				}
			}
		}
		if !found {
			t.Fatal("no delete-collection action")
		}
	})
}

func TestGetKubeConfigChecksTheCluster(t *testing.T) {
	old := config.K8sClient
	t.Cleanup(func() { config.K8sClient = old })
	config.K8sClient = fake.NewClientset(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testClusterId + "-kubeconfig", Namespace: testNamespace},
		Data:       map[string][]byte{"value": []byte("kubeconfig")},
	})

	foreign := newCluster()
	foreign.SetLabels(map[string]string{spxId.SpxLabelProjectID: "spx-99999999-9999-9999-9999-999999999999"})
	setFakeDynamicClient(t, foreign)
	if _, err := GetKubeConfig(context.Background(), testNamespace, testClusterId); err == nil {
		t.Error("kubeconfig returned for a cluster of another project")
	}

	setFakeDynamicClient(t, newCluster())
	value, err := GetKubeConfig(context.Background(), testNamespace, testClusterId)
	if err != nil || string(value) != "kubeconfig" {
		t.Errorf("GetKubeConfig() = %q, %v", value, err)
	}
}
