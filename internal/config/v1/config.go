package v1

import (
	v1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
)

// Config is the Schema for the configs API
type Config struct {
	ControllerConfig `json:",inline"`
	Selector         IngressSelector    `json:"selector"`
	Integrations     IntegrationConfigs `json:"integrations"`
}

//-------------------------------------------------------------------------------------------------

// ControllerConfig provides configuration for the controller.
type ControllerConfig struct {
	Health         HealthConfig         `json:"health,omitempty"`
	LeaderElection LeaderElectionConfig `json:"leaderElection,omitempty"`
	Metrics        MetricsConfig        `json:"metrics,omitempty"`
}

// HealthConfig provides configuration for the controller health checks.
type HealthConfig struct {
	HealthProbeBindAddress string `json:"healthProbeBindAddress,omitempty"`
}

// LeaderElectionConfig provides configuration for the leader election.
type LeaderElectionConfig struct {
	LeaderElect       bool   `json:"leaderElect,omitempty"`
	ResourceName      string `json:"resourceName,omitempty"`
	ResourceNamespace string `json:"resourceNamespace,omitempty"`
}

// MetricsConfig provides configuration for the controller metrics.
type MetricsConfig struct {
	BindAddress string `json:"bindAddress,omitempty"`
}

//-------------------------------------------------------------------------------------------------

// IngressSelector can be used to limit operations to ingresses with a specific class.
type IngressSelector struct {
	IngressClass *string `json:"ingressClass,omitempty"`
}

// IntegrationConfigs describes the configurations for all integrations.
type IntegrationConfigs struct {
	ExternalDNS *ExternalDNSIntegrationConfig `json:"externalDNS"`
	CertManager *CertManagerIntegrationConfig `json:"certManager"`
}

// ExternalDNSIntegrationConfig describes the configuration for the external-dns integration.
// Exactly one of target and target IPs should be set.
type ExternalDNSIntegrationConfig struct {
	TargetService *ServiceRef `json:"targetService,omitempty"`
	TargetIPs     []string    `json:"targetIPs,omitempty"`
}

// CertManagerIntegrationConfig describes the configuration for the cert-manager integration.
type CertManagerIntegrationConfig struct {
	Template v1.Certificate `json:"certificateTemplate"`
}

// ServiceRef uniquely describes a Kubernetes service.
type ServiceRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}
