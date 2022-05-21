package controllers

import (
	"testing"

	configv1 "github.com/borchero/switchboard/internal/config/v1"
	"github.com/borchero/switchboard/internal/k8tests"
	"github.com/stretchr/testify/assert"
)

func TestIntegrationsFromConfig(t *testing.T) {
	// Setup
	scheme := k8tests.NewScheme()
	client := k8tests.NewClient(t, scheme)

	// Test all configurations of integrations
	config := configv1.Config{}
	integrations := integrationsFromConfig(config, client)
	assert.Len(t, integrations, 0)

	config.Integrations.ExternalDNS = &configv1.ExternalDNSIntegrationConfig{
		Target: configv1.ServiceRef{Name: "my-service", Namespace: "my-namespace"},
	}
	integrations = integrationsFromConfig(config, client)
	assert.Len(t, integrations, 1)
	assert.Equal(t, "external-dns", integrations[0].Name())

	config.Integrations.ExternalDNS = nil
	config.Integrations.CertManager = &configv1.CertManagerIntegrationConfig{
		Issuer: configv1.IssuerRef{Kind: "ClusterIssuer", Name: "my-issuer"},
	}
	integrations = integrationsFromConfig(config, client)
	assert.Len(t, integrations, 1)
	assert.Equal(t, "cert-manager", integrations[0].Name())

	config.Integrations.ExternalDNS = &configv1.ExternalDNSIntegrationConfig{
		Target: configv1.ServiceRef{Name: "my-service", Namespace: "my-namespace"},
	}
	integrations = integrationsFromConfig(config, client)
	assert.Len(t, integrations, 2)
}
