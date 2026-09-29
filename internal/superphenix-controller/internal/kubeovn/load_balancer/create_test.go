package loadBalancer

import (
	"context"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-controller/internal/gc/resources/testhelper"
	"github.com/super-phenix/superphenix/internal/superphenix-controller/pkg/config"

	spxId "github.com/super-phenix/superphenix/pkg/superphenix-id"

	v1 "github.com/kubeovn/kube-ovn/pkg/apis/kubeovn/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestCreateLoadBalancer_EndpointIPv4Validation verifies only valid IPv4 endpoints are accepted.
func TestCreateLoadBalancer_EndpointIPv4Validation(t *testing.T) {
	tests := []struct {
		name      string
		endpoints []string
		wantErr   bool
	}{
		{name: "valid IPv4 endpoint", endpoints: []string{"10.0.0.1"}, wantErr: false},
		{name: "multiple valid IPv4 endpoints", endpoints: []string{"10.0.0.1", "192.168.1.2"}, wantErr: false},
		{name: "IPv6 endpoint rejected", endpoints: []string{"fe80::1"}, wantErr: true},
		{name: "IPv6 loopback rejected", endpoints: []string{"::1"}, wantErr: true},
		{name: "garbage endpoint rejected", endpoints: []string{"not-an-ip"}, wantErr: true},
		{name: "empty endpoint rejected", endpoints: []string{""}, wantErr: true},
		{name: "one invalid among valid rejected", endpoints: []string{"10.0.0.1", "bad"}, wantErr: true},
		{name: "endpoint of the project on a shared subnet", endpoints: []string{"10.50.0.10"}, wantErr: false},
		{name: "endpoint of the owner on a shared subnet rejected", endpoints: []string{"10.50.0.20"}, wantErr: true},
		{name: "endpoint in another project's subnet rejected", endpoints: []string{"10.60.0.5"}, wantErr: true},
		{name: "endpoint outside the project subnets rejected", endpoints: []string{"172.16.0.1"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := testhelper.NewFakeKubeOvnClientset()
			config.KubeOvnClient = client
			seedProjectNetwork(t, client, "spx-project")

			info := &CreateLoadBalancerInfo{
				VIP:       "198.18.0.1",
				Endpoints: tt.endpoints,
			}
			info.OrgId = "org"
			info.ProjectId = "project"
			info.ResourceLocalId = "lb"
			info.ResourceEffectiveId = "lb-effective"

			err := info.CreateLoadBalancer(context.Background())
			if tt.wantErr && err == nil {
				t.Errorf("expected an error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

// seedProjectNetwork creates the subnets the endpoint checks look at: one owned
// by the project, one owned by another project and shared with it (with one
// address allocated to each project), and one of another project.
func seedProjectNetwork(t *testing.T, client *testhelper.FakeKubeOvnClientset, project string) {
	t.Helper()
	const other = "spx-99999999-9999-9999-9999-999999999999"
	ctx := context.Background()
	subnets := []*v1.Subnet{
		{ObjectMeta: metav1.ObjectMeta{Name: "own", Labels: map[string]string{spxId.SpxLabelProjectID: project}}, Spec: v1.SubnetSpec{CIDRBlock: "10.0.0.0/16,192.168.1.0/24"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "shared", Labels: map[string]string{spxId.SpxLabelProjectID: other}, Annotations: map[string]string{spxId.SpxAnnotationAllowedProjects: project}}, Spec: v1.SubnetSpec{CIDRBlock: "10.50.0.0/24"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "foreign", Labels: map[string]string{spxId.SpxLabelProjectID: other}}, Spec: v1.SubnetSpec{CIDRBlock: "10.60.0.0/24"}},
	}
	for _, s := range subnets {
		if _, err := client.KubeovnV1().Subnets().Create(ctx, s, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	ips := []*v1.IP{
		{ObjectMeta: metav1.ObjectMeta{Name: "mine", Labels: map[string]string{"ovn.kubernetes.io/subnet": "shared"}}, Spec: v1.IPSpec{Namespace: project, Subnet: "shared", V4IPAddress: "10.50.0.10"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "theirs", Labels: map[string]string{"ovn.kubernetes.io/subnet": "shared"}}, Spec: v1.IPSpec{Namespace: other, Subnet: "shared", V4IPAddress: "10.50.0.20"}},
	}
	for _, i := range ips {
		if _, err := client.KubeovnV1().IPs().Create(ctx, i, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
}
