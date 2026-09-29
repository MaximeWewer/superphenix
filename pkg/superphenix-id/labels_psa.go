package superphenixId

// Pod Security Admission levels applied to project namespaces.
const (
	PodSecurityEnforceLevel = "baseline"
	PodSecurityAuditLevel   = "restricted"
)

// PodSecurityLabels returns the Pod Security Admission labels of a project
// namespace: baseline is enforced, restricted is audited and warned.
func PodSecurityLabels() map[string]string {
	return map[string]string{
		"pod-security.kubernetes.io/enforce": PodSecurityEnforceLevel,
		"pod-security.kubernetes.io/audit":   PodSecurityAuditLevel,
		"pod-security.kubernetes.io/warn":    PodSecurityAuditLevel,
	}
}
