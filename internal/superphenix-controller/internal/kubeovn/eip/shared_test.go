package eip

import (
	"context"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-controller/internal/kubeovn/eip/dnat"
	"github.com/super-phenix/superphenix/internal/superphenix-controller/pkg/config"

	v1 "github.com/kubeovn/kube-ovn/pkg/apis/kubeovn/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ownWorkloadIP   = "10.10.0.10"
	ownerWorkloadIP = "10.10.0.20"
)

// fakeSharedAllocations records one address of the shared subnet for the
// project it is shared with, and one for the project that owns it.
func fakeSharedAllocations(t *testing.T) {
	t.Helper()
	for _, item := range []*v1.IP{
		{ObjectMeta: metav1.ObjectMeta{Name: "own-vm", Labels: map[string]string{"ovn.kubernetes.io/subnet": sharedSubnet}},
			Spec: v1.IPSpec{Namespace: ownProject, Subnet: sharedSubnet, V4IPAddress: ownWorkloadIP}},
		{ObjectMeta: metav1.ObjectMeta{Name: "owner-vm", Labels: map[string]string{"ovn.kubernetes.io/subnet": sharedSubnet}},
			Spec: v1.IPSpec{Namespace: otherProject, Subnet: sharedSubnet, V4IPAddress: ownerWorkloadIP}},
	} {
		if _, err := config.KubeOvnClient.KubeovnV1().IPs().Create(context.Background(), item, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestValidateSharedSubnetTargets(t *testing.T) {
	fakeSubnets()
	fakeSharedAllocations(t)

	own, _ := config.KubeOvnClient.KubeovnV1().Subnets().Get(context.Background(), ownSubnet, metav1.GetOptions{})
	shared, _ := config.KubeOvnClient.KubeovnV1().Subnets().Get(context.Background(), sharedSubnet, metav1.GetOptions{})

	tests := []struct {
		name       string
		subnet     *v1.Subnet
		internalIP string
		snat       []string
		dnat       []dnat.InfoDNAT
		wantErr    bool
	}{
		{name: "owned subnet: any address", subnet: own, internalIP: "10.10.0.99", snat: []string{"10.10.0.0/24"}},
		{name: "shared subnet: own FIP target", subnet: shared, internalIP: ownWorkloadIP},
		{name: "shared subnet: owner FIP target", subnet: shared, internalIP: ownerWorkloadIP, wantErr: true},
		{name: "shared subnet: unallocated FIP target", subnet: shared, internalIP: "10.10.0.99", wantErr: true},
		{name: "shared subnet: own single SNAT address", subnet: shared, snat: []string{ownWorkloadIP + "/32"}},
		{name: "shared subnet: SNAT range", subnet: shared, snat: []string{"10.10.0.0/24"}, wantErr: true},
		{name: "shared subnet: owner SNAT address", subnet: shared, snat: []string{ownerWorkloadIP}, wantErr: true},
		{name: "shared subnet: own DNAT target", subnet: shared, dnat: []dnat.InfoDNAT{{InternalIP: ownWorkloadIP}}},
		{name: "shared subnet: owner DNAT target", subnet: shared, dnat: []dnat.InfoDNAT{{InternalIP: ownerWorkloadIP}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSharedSubnetTargets(context.Background(), tt.subnet, ownProject, tt.internalIP, tt.snat, tt.dnat)
			if tt.wantErr && !apierrors.IsBadRequest(err) {
				t.Fatalf("expected a BadRequest error, got %v", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCreateEip_SharedSubnet(t *testing.T) {
	t.Run("owner workload refused before anything is created", func(t *testing.T) {
		fakeSubnets()
		fakeSharedAllocations(t)

		err := newCreateInfo(sharedSubnet, ownerWorkloadIP).CreateEip(context.Background())
		if !apierrors.IsBadRequest(err) {
			t.Fatalf("expected a BadRequest error, got %v", err)
		}
		eips, _ := config.KubeOvnClient.KubeovnV1().IptablesEIPs().List(context.Background(), metav1.ListOptions{})
		if len(eips.Items) != 0 {
			t.Errorf("nothing must be created, found %d EIP", len(eips.Items))
		}
	})

	t.Run("own workload accepted", func(t *testing.T) {
		fakeSubnets()
		fakeSharedAllocations(t)

		if err := newCreateInfo(sharedSubnet, ownWorkloadIP).CreateEip(context.Background()); err != nil {
			t.Fatalf("CreateEip() error = %v", err)
		}
	})
}
