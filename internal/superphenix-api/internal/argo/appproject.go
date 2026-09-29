package argo

import (
	"context"
	"fmt"
	"slices"
	"strings"

	logger "github.com/super-phenix/superphenix/pkg/utils/log"

	spxId "github.com/super-phenix/superphenix/pkg/superphenix-id"

	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ensureAppProject creates the project's AppProject, or reconciles the spec of an
// existing one so a prefix/destination change is picked up.
// DefaultClusterResourceWhitelist lists the cluster-scoped kinds the
// Applications of a project may manage: the Kube-OVN network objects and the
// namespace rendered by sfs-iaas, and the Kamaji DataStore of sfs-kaas.
//
// The project's Applications include the tenant GitOps application, whose
// manifests come from a repository the tenant controls. Kinds that grant
// cluster-wide power (RBAC, admission policies and webhooks, CRDs,
// PersistentVolumes, StorageClasses...) must never be listed here: the
// platform provides what the tenant charts need (see the kaas-controller
// chart).
var DefaultClusterResourceWhitelist = []metav1.GroupKind{
	{Group: "", Kind: "Namespace"},
	{Group: "kubeovn.io", Kind: "Vpc"},
	{Group: "kubeovn.io", Kind: "Subnet"},
	{Group: "kubeovn.io", Kind: "VpcNatGateway"},
	{Group: "kubeovn.io", Kind: "IptablesEIP"},
	{Group: "kubeovn.io", Kind: "IptablesFIPRule"},
	{Group: "kubeovn.io", Kind: "IptablesSnatRule"},
	{Group: "kubeovn.io", Kind: "IptablesDnatRule"},
	{Group: "kubeovn.io", Kind: "SwitchLBRule"},
	{Group: "kamaji.clastix.io", Kind: "DataStore"},
}

// clusterResourceWhitelist returns the configured whitelist, or the default.
func (c *Client) clusterResourceWhitelist() []metav1.GroupKind {
	if len(c.opts.ClusterResourceWhitelist) > 0 {
		return c.opts.ClusterResourceWhitelist
	}
	return DefaultClusterResourceWhitelist
}

func (c *Client) ensureAppProject(ctx context.Context, orgaId, projectId string) error {
	log := logger.GetLogger(ctx)
	namespace := c.Namespace(projectId)
	// NOTE: the name uses a literal "spx-" prefix while the namespace follows the
	// configured spxPrefix. Deriving it from the prefix would rename every
	// AppProject in a deployment that overrides spxPrefix, orphaning the old one.
	name := fmt.Sprintf("spx-%s", projectId)

	m := spxId.Metadata{
		OrgId:     orgaId,
		ProjectId: projectId,
	}

	appProject := &v1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.opts.AppProjectNamespace,
			Labels:    m.GetLabels(),
			Finalizers: []string{
				"resources-finalizer.argocd.argoproj.io",
			},
		},
		Spec: v1alpha1.AppProjectSpec{
			// Only Applications deployed in this namespace can use this Project
			SourceNamespaces:         []string{name},
			SourceRepos:              []string{"*"},
			ClusterResourceWhitelist: c.clusterResourceWhitelist(),
			Description:              fmt.Sprintf("Project to deploy Superphénix resources in project spx-%s", projectId),
			//  Only permit applications to deploy to Superphenix clusters in their project namespace
			Destinations: []v1alpha1.ApplicationDestination{
				{Name: "*", Namespace: namespace},
				{Name: "*", Namespace: "velero-system"},
			},
		},
	}

	if _, err := c.apps.AppProjects(c.opts.AppProjectNamespace).Create(ctx, appProject, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			log.Info().Str("namespace", namespace).Msg("app project already exists")
			return c.updateAppProject(ctx, appProject)
		}

		log.Err(err).Str("namespace", namespace).Msg("Error creating app project")
		return err
	}

	return nil
}

func (c *Client) updateAppProject(ctx context.Context, appProject *v1alpha1.AppProject) error {
	log := logger.GetLogger(ctx)

	project, err := c.apps.AppProjects(c.opts.AppProjectNamespace).Get(ctx, appProject.Name, metav1.GetOptions{})
	if err != nil {
		log.Err(err).Str("name", appProject.Name).Msg("Error getting app project")
		return err
	}

	project.Spec.SourceNamespaces = appProject.Spec.SourceNamespaces
	project.Spec.SourceRepos = appProject.Spec.SourceRepos
	project.Spec.ClusterResourceWhitelist = appProject.Spec.ClusterResourceWhitelist
	project.Spec.Description = appProject.Spec.Description
	project.Spec.Destinations = appProject.Spec.Destinations

	if _, err := c.apps.AppProjects(c.opts.AppProjectNamespace).Update(ctx, project, metav1.UpdateOptions{}); err != nil {
		log.Warn().Err(err).Str("name", appProject.Name).Msg("failed to update app project")
		return err
	}
	return nil
}

// ReconcileAppProjects realigns the cluster resource whitelist of every
// existing project AppProject, so that a restriction also applies to projects
// created before it. It returns the number of AppProjects updated.
func (c *Client) ReconcileAppProjects(ctx context.Context) (int, error) {
	log := logger.GetLogger(ctx)
	whitelist := c.clusterResourceWhitelist()

	projects, err := c.apps.AppProjects(c.opts.AppProjectNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: spxId.SpxLabelProjectID,
	})
	if err != nil {
		return 0, err
	}

	updated := 0
	for i := range projects.Items {
		project := &projects.Items[i]
		if !strings.HasPrefix(project.Name, "spx-") || slices.Equal(project.Spec.ClusterResourceWhitelist, whitelist) {
			continue
		}

		project.Spec.ClusterResourceWhitelist = whitelist
		if _, err := c.apps.AppProjects(c.opts.AppProjectNamespace).Update(ctx, project, metav1.UpdateOptions{}); err != nil {
			log.Warn().Err(err).Str("name", project.Name).Msg("Failed to restrict app project")
			continue
		}
		updated++
	}
	return updated, nil
}
