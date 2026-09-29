package kaas

import (
	"context"
	"fmt"
	"regexp"

	"github.com/super-phenix/superphenix/pkg/utils/netguard"
)

// The post-install chart fields end up in a shell script executed by a Job in
// the management cluster (sfs-kaas tenant-custom.yaml). They are strictly
// validated here so that no shell metacharacter can ever reach that script.
var (
	// Helm release name: DNS-1123 label, at most 53 characters.
	postInstallChartNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,51}[a-z0-9])?$`)
	// Exact SemVer 2.0 version, with an optional leading "v".
	postInstallChartVersionRegex = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
	// http(s) or oci repository: host, optional port and path, no query,
	// fragment, credentials or whitespace.
	postInstallRepoURLRegex = regexp.MustCompile(`^(https?|oci)://[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?(:[0-9]{1,5})?(/[A-Za-z0-9._~-]+)*/?$`)
	// Kubernetes namespace: DNS-1123 label, at most 63 characters.
	postInstallNamespaceRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
)

const postInstallRepoURLMaxLength = 2048

// validatePostInstallChart checks every user-provided post-install chart
// field. The chart name and repository URL are required; the version and
// namespace are optional (latest version, "default" namespace).
func validatePostInstallChart(spec PostInstallChartSpec) error {
	if !postInstallChartNameRegex.MatchString(spec.ChartName) {
		return fmt.Errorf("post install chart name is invalid, it must match %s", postInstallChartNameRegex.String())
	}

	if len(spec.RepoUrl) > postInstallRepoURLMaxLength || !postInstallRepoURLRegex.MatchString(spec.RepoUrl) {
		return fmt.Errorf("post install chart repository URL is invalid, it must be an http(s) or oci URL without credentials, query or fragment")
	}

	if spec.ChartVersion != "" && !postInstallChartVersionRegex.MatchString(spec.ChartVersion) {
		return fmt.Errorf("post install chart version is invalid, it must be an exact semantic version")
	}

	if spec.Namespace != "" && !postInstallNamespaceRegex.MatchString(spec.Namespace) {
		return fmt.Errorf("post install chart namespace is invalid, it must match %s", postInstallNamespaceRegex.String())
	}

	return nil
}

// postInstallRepoResolver resolves repository hosts; nil means the system
// resolver. Tests replace it.
var postInstallRepoResolver netguard.Resolver

// checkPostInstallRepoHost refuses repositories that are not on a public host:
// the Job that installs the chart runs in the management cluster and would
// otherwise fetch internal services on the user's behalf.
func checkPostInstallRepoHost(ctx context.Context, repoURL string) error {
	if err := netguard.CheckURL(ctx, repoURL, netguard.Options{
		Schemes:  []string{"http", "https", "oci"},
		Resolver: postInstallRepoResolver,
	}); err != nil {
		return fmt.Errorf("post install chart repository URL is not allowed: %w", err)
	}
	return nil
}
