package controllers

import (
	"context"
	"fmt"
	"log/slog"

	configv1 "github.com/borchero/switchboard/internal/config/v1"
	"github.com/borchero/switchboard/internal/ext"
	"github.com/borchero/switchboard/internal/integrations"
	"github.com/borchero/switchboard/internal/switchboard"
	traefik "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ingressAdapter abstracts the type-specific operations for different Traefik ingress resources.
type ingressAdapter interface {
	// newResource returns a new empty instance of the ingress resource.
	newResource() client.Object
	// extractInfo extracts IngressInfo from a fetched ingress resource.
	extractInfo(client.Object) (integrations.IngressInfo, error)
	// setupWithManager configures the controller watches for this resource type.
	setupWithManager(mgr ctrl.Manager, r *IngressRouteReconciler) error
}

// IngressRouteReconciler reconciles Traefik ingress route objects, including both IngressRoute
// and IngressRouteTCP resources.
type IngressRouteReconciler struct {
	client.Client
	logger       *slog.Logger
	selector     switchboard.Selector
	integrations []integrations.Integration
	adapter      ingressAdapter
}

func newReconciler(
	client client.Client, logger *slog.Logger, config configv1.Config, adapter ingressAdapter,
) (IngressRouteReconciler, error) {
	itgs, err := integrationsFromConfig(config, client)
	if err != nil {
		return IngressRouteReconciler{}, fmt.Errorf("failed to initialize integrations: %s", err)
	}
	return IngressRouteReconciler{
		Client:       client,
		logger:       logger,
		selector:     switchboard.NewSelector(config.Selector.IngressClass),
		integrations: itgs,
		adapter:      adapter,
	}, nil
}

// NewIngressRouteReconciler creates a new reconciler for IngressRoute resources.
func NewIngressRouteReconciler(
	client client.Client, logger *slog.Logger, config configv1.Config,
) (IngressRouteReconciler, error) {
	return newReconciler(client, logger, config, ingressRouteAdapter{})
}

// NewIngressRouteTCPReconciler creates a new reconciler for IngressRouteTCP resources.
func NewIngressRouteTCPReconciler(
	client client.Client, logger *slog.Logger, config configv1.Config,
) (IngressRouteReconciler, error) {
	return newReconciler(client, logger, config, ingressRouteTCPAdapter{})
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IngressRouteReconciler) Reconcile(
	ctx context.Context, req ctrl.Request,
) (ctrl.Result, error) {
	logger := r.logger.With("name", req.String())

	// First, we retrieve the full resource
	resource := r.adapter.newResource()

	if err := r.Get(ctx, req.NamespacedName, resource); err != nil {
		if !apierrs.IsNotFound(err) {
			logger.Error("unable to query for ingress route", "error", err)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Then, we check if the resource should be processed
	if !r.selector.Matches(resource.GetAnnotations()) {
		logger.Debug("ignoring ingress route")
		return ctrl.Result{}, nil
	}
	logger.Debug("reconciling ingress route")

	// Now, we have to ensure that all the dependent resources exist by calling all integrations.
	// For this, we first have to extract information about the ingress.
	info, err := r.adapter.extractInfo(resource)
	if err != nil {
		logger.Error("failed to parse hosts from ingress route", "error", err)
		return ctrl.Result{}, err
	}

	// Then, we can run the integrations
	for _, itg := range r.integrations {
		if !r.selector.MatchesIntegration(resource.GetAnnotations(), itg.Name()) {
			// If integration is ignored, skip it
			logger.Debug("ignoring integration", "integration", itg.Name())
			continue
		}
		if err := itg.UpdateResource(ctx, resource, info); err != nil {
			logger.Error("failed to upsert resource",
				"integration", itg.Name(), "error", err,
			)
			return ctrl.Result{}, err
		}
		logger.Debug("successfully upserted resource", "integration", itg.Name())
	}

	logger.Info("ingress route is up to date")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return r.adapter.setupWithManager(mgr, r)
}

//-------------------------------------------------------------------------------------------------
// INGRESS ROUTE ADAPTER
//-------------------------------------------------------------------------------------------------

type ingressRouteAdapter struct{}

func (a ingressRouteAdapter) newResource() client.Object {
	return &traefik.IngressRoute{}
}

func (a ingressRouteAdapter) extractInfo(obj client.Object) (integrations.IngressInfo, error) {
	ir := obj.(*traefik.IngressRoute)
	collection, err := switchboard.NewHostCollection().
		WithTLSHostsIfAvailable(ir.Spec.TLS).
		WithRouteHostsIfRequired(ir.Spec.Routes)
	if err != nil {
		return integrations.IngressInfo{}, err
	}
	return integrations.IngressInfo{
		Hosts: collection.Hosts(),
		TLSSecretName: ext.AndThen(ir.Spec.TLS, func(tls traefik.TLS) string {
			return tls.SecretName
		}),
	}, nil
}

func (a ingressRouteAdapter) setupWithManager(
	mgr ctrl.Manager, r *IngressRouteReconciler,
) error {
	var list traefik.IngressRouteList
	builder := ctrl.NewControllerManagedBy(mgr).For(&traefik.IngressRoute{})
	builder = builderWithIntegrations(builder, r.integrations, r, r.logger, &list,
		func(list *traefik.IngressRouteList) []client.Object {
			return ext.Map(list.Items, func(v traefik.IngressRoute) client.Object {
				return &v
			})
		},
	)
	return builder.Complete(r)
}

//-------------------------------------------------------------------------------------------------
// INGRESS ROUTE TCP ADAPTER
//-------------------------------------------------------------------------------------------------

type ingressRouteTCPAdapter struct{}

func (a ingressRouteTCPAdapter) newResource() client.Object {
	return &traefik.IngressRouteTCP{}
}

func (a ingressRouteTCPAdapter) extractInfo(obj client.Object) (integrations.IngressInfo, error) {
	ir := obj.(*traefik.IngressRouteTCP)
	collection, err := switchboard.NewHostCollection().
		WithTLSTCPHostsIfAvailable(ir.Spec.TLS).
		WithRouteTCPHostsIfRequired(ir.Spec.Routes)
	if err != nil {
		return integrations.IngressInfo{}, err
	}
	return integrations.IngressInfo{
		Hosts: collection.Hosts(),
		TLSSecretName: ext.AndThen(ir.Spec.TLS, func(tls traefik.TLSTCP) string {
			return tls.SecretName
		}),
	}, nil
}

func (a ingressRouteTCPAdapter) setupWithManager(
	mgr ctrl.Manager, r *IngressRouteReconciler,
) error {
	var list traefik.IngressRouteTCPList
	builder := ctrl.NewControllerManagedBy(mgr).For(&traefik.IngressRouteTCP{})
	builder = builderWithIntegrations(builder, r.integrations, r, r.logger, &list,
		func(list *traefik.IngressRouteTCPList) []client.Object {
			return ext.Map(list.Items, func(v traefik.IngressRouteTCP) client.Object {
				return &v
			})
		},
	)
	return builder.Complete(r)
}
