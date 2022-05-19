package controllers

import (
	"context"
	"strings"

	configv1 "github.com/borchero/switchboard/api/v1"
	"github.com/borchero/switchboard/pkg/switchboard"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/external-dns/endpoint"
)

// IngressRouteReconciler reconciles an IngressRoute object.
type IngressRouteReconciler struct {
	client.Client
	scheme   *runtime.Scheme
	logger   *zap.Logger
	selector switchboard.Selector
	target   switchboard.Target
	factory  *switchboard.Factory
	issuer   cmmeta.ObjectReference
}

// NewIngressRouteReconciler creates a new IngressRouteReconciler.
func NewIngressRouteReconciler(
	client client.Client, scheme *runtime.Scheme, logger *zap.Logger, config configv1.IngressSet,
) IngressRouteReconciler {
	return IngressRouteReconciler{
		Client:   client,
		scheme:   scheme,
		logger:   logger,
		selector: switchboard.NewSelector(config.Selector.IngressClass),
		target: switchboard.NewTarget(
			client, config.TargetService.Name, config.TargetService.Namespace,
		),
		factory: switchboard.NewFactory(client, scheme),
		issuer:  cmmeta.ObjectReference{Name: config.Issuer.Name, Kind: config.Issuer.Kind},
	}
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IngressRouteReconciler) Reconcile(
	ctx context.Context, req ctrl.Request,
) (ctrl.Result, error) {
	logger := r.logger.With(zap.String("name", req.String()))

	// First, we retrieve the full resource
	ingressRoute := traefik.IngressRoute{}
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

	// Now, we have to ensure that all the dependent resources exist. For this, we first have to
	// extract all hosts from the ingress.
	hosts := switchboard.NewHostAggregator()
	hosts.ParseTLSHosts(ingressRoute.Spec.TLS)
	hosts.ParseRouteHostsIfRequired(ingressRoute.Spec.Routes)
	if hosts.Len() == 0 {
		// If there are no hosts defined on the ingress route, we can skip the remaining steps
		logger.Info("ingress route does not require DNS endpoint or certificate")
		return ctrl.Result{}, nil
	}

	// First, we attempt to update the DNS entries.
	targetIP, err := r.target.IP(ctx)
	if err != nil {
		logger.Error("failed to obtain target IP", zap.Error(err))
		return ctrl.Result{}, err
	}
	if err := r.factory.UpsertDNSEndpoint(
		ctx, &ingressRoute, hosts.DNSEndpoints(targetIP, 300),
	); err != nil {
		logger.Error("failed to upsert DNS endpoint", zap.Error(err))
		return ctrl.Result{}, err
	}
	logger.Debug("successfully reconciled DNS endpoint")

	// Then, we create the TLS certificate if required
	if ingressRoute.Spec.TLS != nil && ingressRoute.Spec.TLS.SecretName != "" {
		if err := r.factory.UpsertCertificate(
			ctx, &ingressRoute, r.issuer, ingressRoute.Spec.TLS.SecretName, hosts.Hosts(),
		); err != nil {
			if strings.Contains(err.Error(), "the object has been modified") {
				logger.Debug("failed to upsert TLS certificate", zap.Error(err))
			} else {
				logger.Error("failed to upsert TLS certificate", zap.Error(err))
				return ctrl.Result{}, err
			}
		}
		logger.Debug("successfully reconciled TLS certificate")
	}

	// Done
	logger.Info("ingress route is up to date")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&traefik.IngressRoute{}).
		Owns(&certmanager.Certificate{}).
		Owns(&endpoint.DNSEndpoint{}).
		Watches(
			&source.Kind{Type: &v1.Service{}},
			handler.EnqueueRequestsFromMapFunc(r.getAllIngressRoutes),
		).Complete(r)
}

//-------------------------------------------------------------------------------------------------
// UTILITIES
//-------------------------------------------------------------------------------------------------

func (r *IngressRouteReconciler) getAllIngressRoutes(service client.Object) []reconcile.Request {
	// Check whether the service matches the configuration
	if !r.target.Matches(service) {
		return []reconcile.Request{}
	}

	// Find all ingress routes that are associated with the target service
	ingresses := traefik.IngressRouteList{}
	if err := r.List(context.TODO(), &ingresses); err != nil {
		r.logger.Error("failed to list ingress routes upon service change", zap.Error(err))
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(ingresses.Items))
	for i, item := range ingresses.Items {
		requests[i].Name = item.Name
		requests[i].Namespace = item.Namespace
	}
	return requests
}
