package organization

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	operatorv1alpha1 "github.com/super-phenix/superphenix/api/operator/v1alpha1"
	"github.com/super-phenix/superphenix/internal/superphenix-operator/db"
)

const (
	// FinalizerName is the name of the finalizer used to prevent deletion of Organizations that are still referenced by Projects.
	FinalizerName = "operator.superphenix.net/finalizer"
)

// +kubebuilder:rbac:groups=operator.superphenix.net,resources=organizations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.superphenix.net,resources=organizations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.superphenix.net,resources=organizations/finalizers,verbs=update
// +kubebuilder:rbac:groups=operator.superphenix.net,resources=projects,verbs=get;list;watch

// Reconciler reconciles an Organization object.
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// OperatorNamespace is the namespace the operator watches. When empty, all
	// namespaces are reconciled.
	OperatorNamespace string
}

// SetupWithManager registers the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Organization{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
			},
		})).
		Watches(
			&operatorv1alpha1.Project{},
			handler.EnqueueRequestsFromMapFunc(r.findOrganizationsForProject),
			builder.WithPredicates(predicate.Funcs{
				CreateFunc: func(e event.CreateEvent) bool {
					return true
				},
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldProj, ok1 := e.ObjectOld.(*operatorv1alpha1.Project)
					newProj, ok2 := e.ObjectNew.(*operatorv1alpha1.Project)
					if !ok1 || !ok2 {
						return false
					}
					// Trigger if organization reference changed
					if oldProj.Spec.OrganizationRef != newProj.Spec.OrganizationRef {
						return true
					}
					// Trigger if deletion timestamp was set
					if oldProj.DeletionTimestamp.IsZero() && !newProj.DeletionTimestamp.IsZero() {
						return true
					}
					return false
				},
				DeleteFunc: func(e event.DeleteEvent) bool {
					return true
				},
			}),
		).
		Named("organization").
		Complete(r)
}

func (r *Reconciler) findOrganizationsForProject(ctx context.Context, obj client.Object) []ctrl.Request {
	project, ok := obj.(*operatorv1alpha1.Project)
	if !ok {
		return nil
	}

	namespace := project.Spec.OrganizationRef.Namespace
	if namespace == "" {
		namespace = project.Namespace
	}

	return []ctrl.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      project.Spec.OrganizationRef.Name,
				Namespace: namespace,
			},
		},
	}
}

// Reconcile reconciles the state of an Organization resource.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if r.OperatorNamespace != "" && req.Namespace != r.OperatorNamespace {
		return ctrl.Result{}, nil
	}

	org := &operatorv1alpha1.Organization{}
	if err := r.Get(ctx, req.NamespacedName, org); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.V(1).Info("Reconciling Organization", "name", org.Name, "namespace", org.Namespace)

	// Handle Finalizers
	stop, res, err := r.reconcileFinalizers(ctx, org)
	if err != nil || stop {
		return res, err
	}

	// Reconcile Database Binding
	if err := r.reconcileDatabaseBinding(ctx, org); err != nil {
		return ctrl.Result{}, err
	}

	org.Status.ObservedGeneration = org.Generation
	org.Status.DisplayName = org.Name
	if org.Spec.Name != "" {
		org.Status.DisplayName = org.Spec.Name
	}
	if err := r.Status().Update(ctx, org); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcileFinalizers(ctx context.Context, org *operatorv1alpha1.Organization) (bool, ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if !org.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(org, FinalizerName) {
			// Check if any Project references this Organization
			projectList := &operatorv1alpha1.ProjectList{}
			if err := r.List(ctx, projectList); err != nil {
				return true, ctrl.Result{}, err
			}

			referenced := false
			for _, p := range projectList.Items {
				if p.Spec.OrganizationRef.Name == org.Name &&
					(p.Spec.OrganizationRef.Namespace == org.Namespace || (p.Spec.OrganizationRef.Namespace == "" && p.Namespace == org.Namespace)) {
					referenced = true
					break
				}
			}

			if referenced {
				log.Info("Organization is still referenced by one or more Projects, blocking deletion", "name", org.Name)
				return true, ctrl.Result{RequeueAfter: time.Minute}, nil
			}

			controllerutil.RemoveFinalizer(org, FinalizerName)
			if err := r.Update(ctx, org); err != nil {
				return true, ctrl.Result{}, err
			}
		}
		return true, ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(org, FinalizerName) {
		controllerutil.AddFinalizer(org, FinalizerName)
		if err := r.Update(ctx, org); err != nil {
			return true, ctrl.Result{}, err
		}
		return false, ctrl.Result{}, nil
	}

	return false, ctrl.Result{}, nil
}

func (r *Reconciler) reconcileDatabaseBinding(ctx context.Context, org *operatorv1alpha1.Organization) error {
	status := metav1.ConditionFalse
	reason := operatorv1alpha1.ReasonBoundFailed
	message := "Organization not found in Superphenix Database"

	if db.Client != nil {
		found, err := db.OrganizationExists(org.Spec.OrganizationID)
		if err == nil {
			if found {
				status = metav1.ConditionTrue
				reason = operatorv1alpha1.ReasonBoundFound
				message = "Organization is bound to the Superphenix Database"
			}
		} else if err != nil {
			return err
		}
	} else {
		message = "Superphenix Database connection not initialized"
	}

	meta.SetStatusCondition(&org.Status.Conditions, metav1.Condition{
		Type:               operatorv1alpha1.ConditionTypeBound,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: org.Generation,
	})
	return nil
}
