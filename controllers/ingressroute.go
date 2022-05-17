package controllers

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	configv1 "github.com/borchero/switchboard/api/v1"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/external-dns/endpoint"
)

const (
	managedByAnnotationKey = "kubernetes.io/managed-by"
	ingressAnnotationKey   = "kubernetes.io/ingress.class"
	ignoreAnnotationKey    = "switchboard.borchero.com/ignore"
)

var (
	hostRegex = regexp.MustCompile(
		"Host\\(`((?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9])`\\)",
	)
)

// IngressRouteReconciler reconciles an IngressRoute object.
type IngressRouteReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Logger        *zap.Logger
	IngressConfig configv1.IngressSet
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IngressRouteReconciler) Reconcile(
	ctx context.Context, req ctrl.Request,
) (ctrl.Result, error) {
	logger := r.Logger.With(zap.String("name", req.String()))

	// First, we retrieve the full resource
	ingressRoute := traefik.IngressRoute{}
	if err := r.Get(ctx, req.NamespacedName, &ingressRoute); err != nil {
		if !apierrs.IsNotFound(err) {
			logger.Error("unable to query for ingress route", zap.Error(err))
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Then, we check if the resource should be processed
	ignore := ingressRoute.Annotations[ignoreAnnotationKey]
	ingressClass := ingressRoute.Annotations[ingressAnnotationKey]
	matchesSelector := r.IngressConfig.Selector == nil ||
		r.IngressConfig.Selector.IngressClass == ingressClass
	if ignore == "true" || !matchesSelector {
		logger.Debug("ignoring ingress route", zap.String("ingressClass", ingressClass))
		return ctrl.Result{}, nil
	}
	logger.Debug("reconciling ingress route")

	// Now, we have to ensure that all the dependent resources exist.
	hosts := r.getHostsFromIngress(ingressRoute)
	if len(hosts) == 0 {
		// If there are no hosts defined on the ingress route, we can skip the remaining steps
		logger.Info("ingress route does not require DNS endpoint or certificate")
		return ctrl.Result{}, nil
	}

	// First, we attempt to update the DNS entries.
	targetIP, err := r.getTargetIP(ctx)
	if err != nil {
		logger.Error("failed to obtain target IP", zap.Error(err))
		return ctrl.Result{}, err
	}
	dnsEndpoint, err := r.createDNSEndpoint(ingressRoute, hosts, targetIP)
	if err != nil {
		logger.Error("failed to obtain DNS endpoint", zap.Error(err))
		return ctrl.Result{}, err
	}
	if err := r.upsertDNSEndpoint(ctx, dnsEndpoint); err != nil {
		logger.Error("failed to upsert DNS endpoint", zap.Error(err))
		return ctrl.Result{}, err
	}
	logger.Debug("successfully reconciled DNS endpoint")

	// Then, we create the TLS certificate if required
	if ingressRoute.Spec.TLS != nil && ingressRoute.Spec.TLS.SecretName != "" {
		certificate, err := r.createTLSCertificate(ingressRoute, hosts)
		if err != nil {
			logger.Error("failed to obtain TLS certificate", zap.Error(err))
			return ctrl.Result{}, err
		}
		if err := r.upsertTLSCertificate(ctx, certificate); err != nil {
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
	if service.GetName() != r.IngressConfig.TargetService.Name ||
		service.GetNamespace() != r.IngressConfig.TargetService.Namespace {
		return []reconcile.Request{}
	}

	// Find all ingress routes that are associated with the target service
	ingresses := traefik.IngressRouteList{}
	if err := r.List(context.TODO(), &ingresses); err != nil {
		r.Logger.Error("failed to list ingress routes upon service change", zap.Error(err))
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(ingresses.Items))
	for i, item := range ingresses.Items {
		requests[i].Name = item.Name
		requests[i].Namespace = item.Namespace
	}
	return requests
}

func (r *IngressRouteReconciler) getHostsFromIngress(route traefik.IngressRoute) []string {
	hosts := map[string]struct{}{}
	// First, we try to get hosts from the domains under the TLS key
	if route.Spec.TLS != nil {
		for _, domain := range route.Spec.TLS.Domains {
			hosts[domain.Main] = struct{}{}
			for _, san := range domain.SANs {
				hosts[san] = struct{}{}
			}
		}
	}

	// If no domains are provided, we parse rules
	if len(hosts) == 0 {
		for _, route := range route.Spec.Routes {
			if route.Kind == "Rule" {
				for _, match := range hostRegex.FindAllStringSubmatch(route.Match, -1) {
					hosts[match[1]] = struct{}{}
				}
			}
		}
	}

	// Map to array
	result := make([]string, 0, len(hosts))
	for host := range hosts {
		result = append(result, host)
	}
	return result
}

func (r *IngressRouteReconciler) getTargetIP(ctx context.Context) (string, error) {
	service := v1.Service{}
	name := types.NamespacedName{
		Name:      r.IngressConfig.TargetService.Name,
		Namespace: r.IngressConfig.TargetService.Namespace,
	}
	if err := r.Get(ctx, name, &service); err != nil {
		return "", fmt.Errorf("failed to query for target service: %w", err)
	}

	// The IP is either the first load balancer IP or the cluster IP
	targetIP := service.Spec.ClusterIP
	lbIngress := service.Status.LoadBalancer.Ingress
	if len(lbIngress) > 0 {
		targetIP = lbIngress[0].IP
	}
	return targetIP, nil
}

func (r *IngressRouteReconciler) createDNSEndpoint(
	route traefik.IngressRoute, hosts []string, targetIP string,
) (endpoint.DNSEndpoint, error) {
	annotations := map[string]string{}
	if ingressClass, ok := route.Annotations[ingressAnnotationKey]; ok {
		annotations[ingressAnnotationKey] = ingressClass
	}

	endpoints := make([]*endpoint.Endpoint, len(hosts))
	for i, host := range hosts {
		endpoints[i] = &endpoint.Endpoint{
			DNSName:    host,
			Targets:    []string{targetIP},
			RecordType: "A",
			RecordTTL:  300,
		}
	}

	dnsEndpoint := endpoint.DNSEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:        route.Name,
			Namespace:   route.Namespace,
			Annotations: annotations,
			Labels: map[string]string{
				managedByAnnotationKey: "switchboard",
			},
		},
		Spec: endpoint.DNSEndpointSpec{
			Endpoints: endpoints,
		},
	}

	if err := ctrl.SetControllerReference(&route, &dnsEndpoint, r.Scheme); err != nil {
		return dnsEndpoint, fmt.Errorf(
			"failed to set controller reference for DNS endpoint: %w", err,
		)
	}
	return dnsEndpoint, nil
}

func (r *IngressRouteReconciler) upsertDNSEndpoint(
	ctx context.Context, dnsEndpoint endpoint.DNSEndpoint,
) error {
	key := types.NamespacedName{
		Name:      dnsEndpoint.Name,
		Namespace: dnsEndpoint.Namespace,
	}

	existingDNSEndpoint := endpoint.DNSEndpoint{}
	if err := r.Get(ctx, key, &existingDNSEndpoint); err != nil {
		if apierrs.IsNotFound(err) {
			if err := r.Create(ctx, &dnsEndpoint); err != nil {
				return fmt.Errorf("failed to create DNS endpoint: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to query for existing DNS endpoint: %w", err)
	}

	dnsEndpoint.Spec.DeepCopyInto(&existingDNSEndpoint.Spec)
	existingDNSEndpoint.Annotations = dnsEndpoint.Annotations
	existingDNSEndpoint.Labels = dnsEndpoint.Labels
	if err := r.Update(ctx, &existingDNSEndpoint); err != nil {
		return fmt.Errorf("failed to update DNS endpoint: %w", err)
	}
	return nil
}

func (r *IngressRouteReconciler) createTLSCertificate(
	route traefik.IngressRoute, hosts []string,
) (certmanager.Certificate, error) {
	certificate := certmanager.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-tls", route.Name),
			Namespace: route.Namespace,
			Labels: map[string]string{
				managedByAnnotationKey: "switchboard",
			},
		},
		Spec: certmanager.CertificateSpec{
			SecretName: route.Spec.TLS.SecretName,
			DNSNames:   hosts,
			IssuerRef: cmmeta.ObjectReference{
				Name: r.IngressConfig.Issuer.Name,
				Kind: r.IngressConfig.Issuer.Kind,
			},
		},
	}

	if err := ctrl.SetControllerReference(&route, &certificate, r.Scheme); err != nil {
		return certificate, fmt.Errorf(
			"failed to set controller reference for TLS certificate: %w", err,
		)
	}
	return certificate, nil
}

func (r *IngressRouteReconciler) upsertTLSCertificate(
	ctx context.Context, certificate certmanager.Certificate,
) error {
	key := types.NamespacedName{
		Name:      certificate.Name,
		Namespace: certificate.Namespace,
	}

	existingCertificate := certmanager.Certificate{}
	if err := r.Get(ctx, key, &existingCertificate); err != nil {
		if apierrs.IsNotFound(err) {
			if err := r.Create(ctx, &certificate); err != nil {
				return fmt.Errorf("failed to create TLS certificate: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to query for existing TLS certificate: %w", err)
	}

	certificate.Spec.DeepCopyInto(&existingCertificate.Spec)
	existingCertificate.Labels = certificate.Labels
	if err := r.Update(ctx, &existingCertificate); err != nil {
		return fmt.Errorf("failed to update TLS certificate: %w", err)
	}
	return nil
}
