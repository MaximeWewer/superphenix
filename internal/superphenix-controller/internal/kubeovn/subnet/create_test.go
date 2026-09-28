package subnet

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
	ownProject   = "spx-11111111-1111-1111-1111-111111111111"
	otherProject = "spx-22222222-2222-2222-2222-222222222222"
	ownVpc       = "spx-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	otherVpc     = "spx-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

func vpcWithProject(name, projectID string) *v1.Vpc {
	return &v1.Vpc{ObjectMeta: metav1.ObjectMeta{
		Name:   name,
		Labels: map[string]string{spxId.SpxLabelProjectID: projectID},
	}}
}

// TestCheckVpcOwnership ensures a subnet can only be attached to a VPC of the
// caller's project: Kube-OVN VPCs are cluster-scoped, so the project label is
// the only ownership boundary.
func TestCheckVpcOwnership(t *testing.T) {
	config.KubeOvnClient = testhelper.NewFakeKubeOvnClientset(
		vpcWithProject(ownVpc, ownProject),
		vpcWithProject(otherVpc, otherProject),
		&v1.Vpc{ObjectMeta: metav1.ObjectMeta{Name: "ovn-cluster"}},
	)

	tests := []struct {
		name         string
		vpc          string
		wantErr      bool
		wantNotFound bool
	}{
		{name: "own VPC is accepted", vpc: ownVpc},
		{name: "VPC of another project is rejected", vpc: otherVpc, wantErr: true, wantNotFound: true},
		{name: "default cluster VPC is rejected", vpc: "ovn-cluster", wantErr: true, wantNotFound: true},
		{name: "empty VPC is rejected", vpc: "", wantErr: true, wantNotFound: true},
		{name: "unknown VPC is rejected", vpc: "spx-cccccccc-cccc-cccc-cccc-cccccccccccc", wantErr: true, wantNotFound: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkVpcOwnership(context.Background(), tt.vpc, ownProject)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkVpcOwnership() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantNotFound && !apierrors.IsNotFound(err) {
				t.Errorf("expected a NotFound error, got %v", err)
			}
		})
	}
}

// TestCreateSubnet_ForeignVpcRejected checks that CreateSubnet stops before
// creating anything when the VPC belongs to another project.
func TestCreateSubnet_ForeignVpcRejected(t *testing.T) {
	fake := testhelper.NewFakeKubeOvnClientset(vpcWithProject(otherVpc, otherProject))
	config.KubeOvnClient = fake

	info := &CreateSubnetInfo{}
	info.OrgId = "33333333-3333-3333-3333-333333333333"
	info.ProjectId = "11111111-1111-1111-1111-111111111111"
	info.ResourceLocalId = "44444444-4444-4444-4444-444444444444"
	info.General.VpcEId = otherVpc
	info.Network.Protocol = "IPv4"
	info.Network.IPv4 = "10.200.0.0/24"
	info.NatGateway.Enable = true

	err := info.CreateSubnet(context.Background())
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected a NotFound error, got %v", err)
	}

	subnets, err := fake.KubeovnV1().Subnets().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list subnets: %v", err)
	}
	if len(subnets.Items) != 0 {
		t.Errorf("no subnet must be created in a foreign VPC, found %d", len(subnets.Items))
	}
}
