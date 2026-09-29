// Package subnetaccess decides which internal addresses a project may target
// through the network objects it creates (EIP rules, load balancer endpoints).
//
// A project may target any address of a subnet it owns. A subnet shared with
// it (superphenix.net/allowedProjects) also hosts the owner's workloads, so
// there the project may only target addresses allocated to its own pods and
// VMs, as recorded by the Kube-OVN IP objects.
package subnetaccess

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/super-phenix/superphenix/internal/superphenix-controller/internal/utils"
	k8s "github.com/super-phenix/superphenix/internal/superphenix-controller/pkg/config"

	spxId "github.com/super-phenix/superphenix/pkg/superphenix-id"

	v1 "github.com/kubeovn/kube-ovn/pkg/apis/kubeovn/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// subnetNameLabel is the label Kube-OVN sets on IP objects with their subnet.
const subnetNameLabel = "ovn.kubernetes.io/subnet"

// IsOwnedBy reports whether the project owns the subnet, as opposed to the
// subnet being shared with it.
func IsOwnedBy(subnet *v1.Subnet, projectID string) bool {
	return subnet.GetLabels()[spxId.SpxLabelProjectID] == projectID
}

// IsAccessible reports whether the project owns the subnet or the subnet is
// shared with it.
func IsAccessible(subnet *v1.Subnet, projectID string) bool {
	return IsOwnedBy(subnet, projectID) || utils.IsSharedWithProject(subnet.GetAnnotations(), projectID)
}

// CheckAllocatedToProject returns a BadRequest error unless every ip is
// allocated, on the subnet, to a pod or VM of the project namespace.
func CheckAllocatedToProject(ctx context.Context, subnetName, projectID string, ips []string) error {
	if len(ips) == 0 {
		return nil
	}

	list, err := k8s.KubeOvnClient.KubeovnV1().IPs().List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", subnetNameLabel, subnetName),
	})
	if err != nil {
		return err
	}

	owners := make(map[string]string)
	for _, item := range list.Items {
		for _, addr := range []string{item.Spec.IPAddress, item.Spec.V4IPAddress, item.Spec.V6IPAddress} {
			for _, a := range strings.Split(addr, ",") {
				if parsed := net.ParseIP(strings.TrimSpace(a)); parsed != nil {
					owners[parsed.String()] = item.Spec.Namespace
				}
			}
		}
	}

	for _, ip := range ips {
		parsed := net.ParseIP(strings.TrimSpace(ip))
		if parsed == nil || owners[parsed.String()] != projectID {
			return apierrors.NewBadRequest(fmt.Sprintf(
				"internal IP %q is not allocated to a workload of the project on the shared subnet", ip))
		}
	}
	return nil
}

// CheckTargets validates addresses targeted by a project that are not bound to
// a known subnet (load balancer endpoints): each one must lie in a subnet the
// project can access, and on a shared subnet it must be allocated to the
// project.
func CheckTargets(ctx context.Context, projectID string, ips []string) error {
	if len(ips) == 0 {
		return nil
	}

	subnets, err := k8s.KubeOvnClient.KubeovnV1().Subnets().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ip := range ips {
		var target *v1.Subnet
		for i := range subnets.Items {
			subnet := &subnets.Items[i]
			if !IsAccessible(subnet, projectID) {
				continue
			}
			if ok, err := utils.IsIPInCIDR(subnet.Spec.CIDRBlock, ip); err == nil && ok {
				target = subnet
				// Prefer a subnet the project owns when several match.
				if IsOwnedBy(subnet, projectID) {
					break
				}
			}
		}

		if target == nil {
			return apierrors.NewBadRequest(fmt.Sprintf("IP %q is not in a subnet of the project", ip))
		}
		if !IsOwnedBy(target, projectID) {
			if err := CheckAllocatedToProject(ctx, target.Name, projectID, []string{ip}); err != nil {
				return err
			}
		}
	}
	return nil
}
