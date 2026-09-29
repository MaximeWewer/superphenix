package vm

import (
	"context"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-controller/pkg/config"

	"go.uber.org/mock/gomock"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

const (
	hardeningNamespace = "spx-22222222-2222-2222-2222-222222222222"
	otherNamespace     = "spx-99999999-9999-9999-9999-999999999999"
)

func mockVirtClientVM(t *testing.T) (*kubecli.MockKubevirtClient, *kubecli.MockVirtualMachineInterface) {
	t.Helper()
	ctrl := gomock.NewController(t)
	client := kubecli.NewMockKubevirtClient(ctrl)
	vmIface := kubecli.NewMockVirtualMachineInterface(ctrl)
	orig := config.VirtClient
	config.VirtClient = client
	t.Cleanup(func() { config.VirtClient = orig })
	return client, vmIface
}

func vmWithNetwork(name, namespace, networkName string, labels map[string]string) v1.VirtualMachine {
	return v1.VirtualMachine{
		ObjectMeta: k8smetav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec: v1.VirtualMachineSpec{Template: &v1.VirtualMachineInstanceTemplateSpec{Spec: v1.VirtualMachineInstanceSpec{
			Networks: []v1.Network{{Name: "interface-0", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: networkName}}}},
		}}},
	}
}

func TestIsVMInSubnet(t *testing.T) {
	const subnet = "spx-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	tests := []struct {
		name string
		vms  []v1.VirtualMachine
		want bool
	}{
		{name: "no VM", want: false},
		{name: "VM of the project on the subnet", vms: []v1.VirtualMachine{vmWithNetwork("a", hardeningNamespace, hardeningNamespace+"/"+subnet, nil)}, want: true},
		{name: "VM of another project on the shared subnet", vms: []v1.VirtualMachine{vmWithNetwork("b", otherNamespace, hardeningNamespace+"/"+subnet, nil)}, want: true},
		{name: "VM on another subnet", vms: []v1.VirtualMachine{vmWithNetwork("c", hardeningNamespace, hardeningNamespace+"/spx-other", nil)}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, vmIface := mockVirtClientVM(t)
			client.EXPECT().VirtualMachine(k8smetav1.NamespaceAll).Return(vmIface)
			vmIface.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1.VirtualMachineList{Items: tt.vms}, nil)

			got, err := IsVMInSubnet(context.Background(), hardeningNamespace, subnet)
			if err != nil || got != tt.want {
				t.Errorf("IsVMInSubnet() = %v, %v, want %v", got, err, tt.want)
			}
		})
	}
}
