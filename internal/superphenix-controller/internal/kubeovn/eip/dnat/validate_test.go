package dnat

import (
	"context"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-controller/internal/gc/resources/testhelper"
	"github.com/super-phenix/superphenix/internal/superphenix-controller/pkg/config"

	spxId "github.com/super-phenix/superphenix/pkg/superphenix-id"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateDNATRefusesInvalidPortsAndProtocol(t *testing.T) {
	eip := spxId.Metadata{
		OrgId:               "11111111-1111-1111-1111-111111111111",
		ProjectId:           "22222222-2222-2222-2222-222222222222",
		ResourceEffectiveId: "33333333-3333-3333-3333-333333333333",
	}

	for _, item := range []InfoDNAT{
		{ExternalPort: "80 --x", InternalIP: "10.0.0.1", InternalPort: "80", Protocol: "tcp"},
		{ExternalPort: "80", InternalIP: "10.0.0.1", InternalPort: "99999", Protocol: "tcp"},
		{ExternalPort: "80", InternalIP: "10.0.0.1", InternalPort: "80", Protocol: "tcp;x"},
	} {
		config.KubeOvnClient = testhelper.NewFakeKubeOvnClientset()
		err := CreateDNAT(context.Background(), item, eip)
		if !apierrors.IsBadRequest(err) {
			t.Errorf("CreateDNAT(%+v) error = %v, want BadRequest", item, err)
		}
		rules, _ := config.KubeOvnClient.KubeovnV1().IptablesDnatRules().List(context.Background(), metav1.ListOptions{})
		if len(rules.Items) != 0 {
			t.Errorf("CreateDNAT(%+v) created %d rules, want none", item, len(rules.Items))
		}
	}

	config.KubeOvnClient = testhelper.NewFakeKubeOvnClientset()
	valid := InfoDNAT{ExternalPort: "80", InternalIP: "10.0.0.1", InternalPort: "8080", Protocol: "tcp"}
	if err := CreateDNAT(context.Background(), valid, eip); err != nil {
		t.Errorf("CreateDNAT(valid) error = %v", err)
	}
}
