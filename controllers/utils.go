package controllers

import (
	configv1 "github.com/borchero/switchboard/pkg/config/v1"
	"github.com/borchero/switchboard/pkg/ext"
	"github.com/borchero/switchboard/pkg/integrations"
	"github.com/borchero/switchboard/pkg/k8s"
	"github.com/borchero/switchboard/pkg/switchboard"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	traefik "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func integrationsFromConfig(
	config configv1.Config, client client.Client,
) []integrations.Integration {
	result := make([]integrations.Integration, 0)
	if config.Integrations.ExternalDNS != nil {
		result = append(result, integrations.NewExternalDNS(
			client, switchboard.NewTarget(
				config.Integrations.ExternalDNS.Target.Name,
				config.Integrations.ExternalDNS.Target.Namespace,
			),
		))
	}
	if config.Integrations.CertManager != nil {
		result = append(result, integrations.NewCertManager(
			client, cmmeta.ObjectReference{
				Kind: config.Integrations.CertManager.Issuer.Kind,
				Name: config.Integrations.CertManager.Issuer.Name,
			},
		))
	}
	return result
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
				&source.Kind{Type: itg.WatchedObject()},
				handler.EnqueueRequestsFromMapFunc(enqueue),
			)
		}
	}

	return builder
}
