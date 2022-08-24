package controllers

import (
	"context"
	"testing"

	configv1 "github.com/borchero/switchboard/internal/config/v1"
	"github.com/borchero/switchboard/internal/k8tests"
	"github.com/borchero/zeus/pkg/zeus"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationsFromConfig(t *testing.T) {
	// Setup
	scheme := k8tests.NewScheme()
	client := k8tests.NewClient(t, scheme)

	// Test all configurations of integrations
	config := configv1.Config{}
	integrations, err := integrationsFromConfig(config, client, zeus.Logger(context.Background()))
	require.Nil(t, err)
	assert.Len(t, integrations, 0)

	config.Integrations.ExternalDNS = &configv1.ExternalDNSIntegrationConfig{
		TargetService: &configv1.ServiceRef{Name: "my-service", Namespace: "my-namespace"},
	}
	integrations, err = integrationsFromConfig(config, client, zeus.Logger(context.Background()))
	require.Nil(t, err)
	assert.Len(t, integrations, 1)
	assert.Equal(t, "external-dns", integrations[0].Name())

	config.Integrations.ExternalDNS = nil
	config.Integrations.CertManager = &configv1.CertManagerIntegrationConfig{
		Template: certmanager.Certificate{
			Spec: certmanager.CertificateSpec{
				IssuerRef: cmmeta.ObjectReference{
					Kind: "ClusterIssuer",
					Name: "my-issuer",
				},
			},
		},
	}
	integrations, err = integrationsFromConfig(config, client, zeus.Logger(context.Background()))
	require.Nil(t, err)
	assert.Len(t, integrations, 1)
	assert.Equal(t, "cert-manager", integrations[0].Name())

	config.Integrations.ExternalDNS = &configv1.ExternalDNSIntegrationConfig{
		TargetIPs: []string{"127.0.0.1"},
	}
	integrations, err = integrationsFromConfig(config, client, zeus.Logger(context.Background()))
	require.Nil(t, err)
	assert.Len(t, integrations, 2)

	// Must fail if external DNS is not configured correctly
	config.Integrations.ExternalDNS = &configv1.ExternalDNSIntegrationConfig{}
	_, err = integrationsFromConfig(config, client, zeus.Logger(context.Background()))
	require.NotNil(t, err)

	config.Integrations.ExternalDNS = &configv1.ExternalDNSIntegrationConfig{
		TargetIPs: []string{},
	}
	_, err = integrationsFromConfig(config, client, zeus.Logger(context.Background()))
	require.NotNil(t, err)
}
