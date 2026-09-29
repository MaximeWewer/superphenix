package group

import (
	"fmt"
	"slices"
	"strings"

	v1PSet "github.com/super-phenix/superphenix/pkg/permify-wrapper/pkg/base/v1/permissionSet"
)

// isAssignablePermissionSet reports whether a permission set can be granted
// through the IAM API: it must be known and must not be internal (spx_*).
// Internal sets such as spx_owner are only granted by the predefined catalog.
func isAssignablePermissionSet(ps string) bool {
	if strings.HasPrefix(ps, v1PSet.InternalPrefix) {
		return false
	}
	_, known := v1PSet.PermissionSetsEntityMap[ps]
	return known
}

// checkRequestedPermissionSets rejects any permission set that the caller is
// not allowed to grant. Internal sets already held by the group (current) are
// kept, so editing a legacy group does not fail, but none can be added.
func checkRequestedPermissionSets(requested, current []string) error {
	var refused []string
	for _, ps := range requested {
		if ps == v1PSet.SpxMember || isAssignablePermissionSet(ps) || slices.Contains(current, ps) {
			continue
		}
		refused = append(refused, ps)
	}
	if len(refused) > 0 {
		return fmt.Errorf("permission sets not allowed: %s", strings.Join(refused, ", "))
	}
	return nil
}

// assignablePermissionSets keeps only the sets that can be granted through the
// IAM API, dropping internal ones such as spx_owner.
func assignablePermissionSets(sets []string) []string {
	result := make([]string, 0, len(sets))
	for _, ps := range sets {
		if isAssignablePermissionSet(ps) {
			result = append(result, ps)
		}
	}
	return result
}
