package subnetaccess

import (
	"context"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-controller/internal/gc/resources/testhelper"
	"github.com/super-phenix/superphenix/internal/superphenix-controller/pkg/config"

	spxId "github.com/super-phenix/superphenix/pkg/superphenix-id"

	v1 "github.com/kubeovn/kube-ovn/pkg/apis/kubeovn/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	guest        = "spx-11111111-1111-1111-1111-111111111111"
	owner        = "spx-22222222-2222-2222-2222-222222222222"
	stranger     = "spx-33333333-3333-3333-3333-333333333333"
	guestSubnet  = "spx-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	sharedSubnet = "spx-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	otherSubnet  = "spx-cccccccc-cccc-cccc-cccc-cccccccccccc"
)

func subnet(name, project, cidr string, sharedWith string) *v1.Subnet {
	s := &v1.Subnet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{spxId.SpxLabelProjectID: project}},
		Spec:       v1.SubnetSpec{CIDRBlock: cidr},
	}
	if sharedWith != "" {
		s.Annotations = map[string]string{spxId.SpxAnnotationAllowedProjects: sharedWith}
	}
	return s
}

func ip(name, subnetName, namespace, v4, v6 string) *v1.IP {
	return &v1.IP{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{subnetNameLabel: subnetName}},
		Spec:       v1.IPSpec{Namespace: namespace, Subnet: subnetName, V4IPAddress: v4, V6IPAddress: v6},
	}
}

func setup(t *testing.T) {
	t.Helper()
	fake := testhelper.NewFakeKubeOvnClientset()
	ctx := context.Background()
	for _, s := range []*v1.Subnet{
		subnet(guestSubnet, guest, "10.1.0.0/24", ""),
		subnet(sharedSubnet, owner, "10.2.0.0/24,fd00:2::/64", guest),
		subnet(otherSubnet, stranger, "10.3.0.0/24", ""),
	} {
		if _, err := fake.KubeovnV1().Subnets().Create(ctx, s, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, i := range []*v1.IP{
		ip("guest-vm", sharedSubnet, guest, "10.2.0.10", "fd00:2::10"),
		ip("owner-vm", sharedSubnet, owner, "10.2.0.20", ""),
		ip("elsewhere", otherSubnet, guest, "10.2.0.30", ""),
	} {
		if _, err := fake.KubeovnV1().IPs().Create(ctx, i, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	config.KubeOvnClient = fake
}

func TestCheckAllocatedToProject(t *testing.T) {
	setup(t)
	tests := []struct {
		name    string
		ips     []string
		wantErr bool
	}{
		{name: "own workload v4", ips: []string{"10.2.0.10"}},
		{name: "own workload v6", ips: []string{"fd00:2:0:0::10"}},
		{name: "no address", ips: nil},
		{name: "owner workload", ips: []string{"10.2.0.20"}, wantErr: true},
		{name: "unallocated address", ips: []string{"10.2.0.99"}, wantErr: true},
		{name: "allocation recorded on another subnet", ips: []string{"10.2.0.30"}, wantErr: true},
		{name: "one refused among allowed", ips: []string{"10.2.0.10", "10.2.0.20"}, wantErr: true},
		{name: "invalid address", ips: []string{"not-an-ip"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckAllocatedToProject(context.Background(), sharedSubnet, guest, tt.ips)
			if tt.wantErr {
				if !apierrors.IsBadRequest(err) {
					t.Errorf("err = %v, want BadRequest", err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckTargets(t *testing.T) {
	setup(t)
	tests := []struct {
		name    string
		ips     []string
		wantErr bool
	}{
		{name: "any address of an owned subnet", ips: []string{"10.1.0.200"}},
		{name: "own workload on a shared subnet", ips: []string{"10.2.0.10"}},
		{name: "owner workload on a shared subnet", ips: []string{"10.2.0.20"}, wantErr: true},
		{name: "subnet of another project", ips: []string{"10.3.0.5"}, wantErr: true},
		{name: "address outside every subnet", ips: []string{"192.168.1.1"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTargets(context.Background(), guest, tt.ips)
			if tt.wantErr {
				if !apierrors.IsBadRequest(err) {
					t.Errorf("err = %v, want BadRequest", err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
