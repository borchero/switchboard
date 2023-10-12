package controllers

import (
	"context"
	"fmt"

	configv1 "github.com/borchero/switchboard/internal/config/v1"
	"github.com/borchero/switchboard/internal/ext"
	"github.com/borchero/switchboard/internal/integrations"
	"github.com/borchero/switchboard/internal/switchboard"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	"go.uber.org/zap"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IngressRouteReconciler reconciles an IngressRoute object.
type IngressRouteReconciler struct {
	client.Client
	logger       *zap.Logger
	selector     switchboard.Selector
	integrations []integrations.Integration
}

// NewIngressRouteReconciler creates a new IngressRouteReconciler.
func NewIngressRouteReconciler(
	client client.Client, logger *zap.Logger, config configv1.Config,
) (IngressRouteReconciler, error) {
	integrations, err := integrationsFromConfig(config, client)
	if err != nil {
		return IngressRouteReconciler{}, fmt.Errorf("failed to initialize integrations: %s", err)
	}
	return IngressRouteReconciler{
		Client:       client,
		logger:       logger,
		selector:     switchboard.NewSelector(config.Selector.IngressClass),
		integrations: integrations,
	}, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IngressRouteReconciler) Reconcile(
	ctx context.Context, req ctrl.Request,
) (ctrl.Result, error) {
	logger := r.logger.With(zap.String("name", req.String()))

	// First, we retrieve the full resource
	var ingressRoute traefik.IngressRoute

	if err := r.Get(ctx, req.NamespacedName, &ingressRoute); err != nil {
		if !apierrs.IsNotFound(err) {
			logger.Error("unable to query for ingress route", zap.Error(err))
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Then, we check if the resource should be processed
	if !r.selector.Matches(ingressRoute.Annotations) {
		logger.Debug("ignoring ingress route")
		return ctrl.Result{}, nil
	}
	logger.Debug("reconciling ingress route")

	// Now, we have to ensure that all the dependent resources exist by calling all integrations.
	// For this, we first have to extract information about the ingress.
	collection, err := switchboard.NewHostCollection().
		WithTLSHostsIfAvailable(ingressRoute.Spec.TLS).
		WithRouteHostsIfRequired(ingressRoute.Spec.Routes)
	if err != nil {
		logger.Error("failed to parse hosts from ingress route", zap.Error(err))
		return ctrl.Result{}, err
	}
	info := integrations.IngressInfo{
		Hosts: collection.Hosts(),
		TLSSecretName: ext.AndThen(ingressRoute.Spec.TLS, func(tls traefik.TLS) string {
			return tls.SecretName
		}),
	}

	// Then, we can run the integrations
	for _, itg := range r.integrations {
		if !r.selector.MatchesIntegration(ingressRoute.Annotations, itg.Name()) {
			// If integration is ignored, skip it
			logger.Debug("ignoring integration", zap.String("integration", itg.Name()))
			continue
		}
		if err := itg.UpdateResource(ctx, &ingressRoute, info); err != nil {
			logger.Error("failed to upsert resource",
				zap.String("integration", itg.Name()), zap.Error(err),
			)
			return ctrl.Result{}, err
		}
		logger.Debug("successfully upserted resource", zap.String("integration", itg.Name()))
	}

	logger.Info("ingress route is up to date")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).For(&traefik.IngressRoute{})
	builder = builderWithIntegrations(builder, r.integrations, r, r.logger)
	return builder.Complete(r)
}
