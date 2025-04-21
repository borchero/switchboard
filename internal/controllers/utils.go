package controllers

import (
	"fmt"

	configv1 "github.com/borchero/switchboard/internal/config/v1"
	"github.com/borchero/switchboard/internal/ext"
	"github.com/borchero/switchboard/internal/integrations"
	"github.com/borchero/switchboard/internal/k8s"
	"github.com/borchero/switchboard/internal/switchboard"
	traefik "github.com/traefik/traefik/v3/pkg/provider/kubernetes/crd/traefikio/v1alpha1"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

func integrationsFromConfig(
	config configv1.Config, client client.Client,
) ([]integrations.Integration, error) {
	result := make([]integrations.Integration, 0)
	externalDNS := config.Integrations.ExternalDNS
	if config.Integrations.ExternalDNS != nil {
		if (externalDNS.TargetService == nil) == (len(externalDNS.TargetIPs) == 0) {
			return nil, fmt.Errorf(
				"exactly one of `targetService` and `targetIPs` must be set for external-dns",
			)
		}
		if externalDNS.TargetService != nil {
			result = append(result, integrations.NewExternalDNS(
				client, switchboard.NewServiceTarget(
					externalDNS.TargetService.Name,
					externalDNS.TargetService.Namespace,
				),
			))
		} else {
			result = append(result, integrations.NewExternalDNS(
				client, switchboard.NewStaticTarget(externalDNS.TargetIPs...),
			))
		}
	}

	certManager := config.Integrations.CertManager
	if certManager != nil {
		result = append(result, integrations.NewCertManager(client, certManager.Template))
	}
	return result, nil
}

func builderWithIntegrations(
	builder *builder.Builder,
	integrations []integrations.Integration,
	ctrlClient client.Client,
	logger *zap.Logger,
) *builder.Builder {
	// Reconcile whenever an owned resource of one of the integrations is modified
	for _, itg := range integrations {
		builder = builder.Owns(itg.OwnedResource())
	}

	// Watch for dependent resources if required
	for _, itg := range integrations {
		if itg.WatchedObject() != nil {
			var list traefik.IngressRouteList
			enqueue := k8s.EnqueueMapFunc(ctrlClient, logger, itg.WatchedObject(), &list,
				func(list *traefik.IngressRouteList) []client.Object {
					return ext.Map(list.Items, func(v traefik.IngressRoute) client.Object {
						return &v
					})
				},
			)
			builder = builder.Watches(
				itg.WatchedObject(),
				handler.EnqueueRequestsFromMapFunc(enqueue),
			)
		}
	}

	return builder
}
