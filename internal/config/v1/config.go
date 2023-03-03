package v1

import (
	v1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
)

// Config is the Schema for the configs API
type Config struct {
	ControllerConfig `yaml:",inline"`
	Selector         IngressSelector    `yaml:"selector"`
	Integrations     IntegrationConfigs `yaml:"integrations"`
}

//-------------------------------------------------------------------------------------------------

// ControllerConfig provides configuration for the controller.
type ControllerConfig struct {
	Health         HealthConfig         `yaml:"health,omitempty"`
	LeaderElection LeaderElectionConfig `yaml:"leaderElection,omitempty"`
	Metrics        MetricsConfig        `yaml:"metrics,omitempty"`
}

// HealthConfig provides configuration for the controller health checks.
type HealthConfig struct {
	HealthProbeBindAddress string `yaml:"healthProbeBindAddress,omitempty"`
}

// LeaderElectionConfig provides configuration for the leader election.
type LeaderElectionConfig struct {
	LeaderElect       bool   `yaml:"leaderElect,omitempty"`
	ResourceName      string `yaml:"resourceName,omitempty"`
	ResourceNamespace string `yaml:"resourceNamespace,omitempty"`
}

// MetricsConfig provides configuration for the controller metrics.
type MetricsConfig struct {
	BindAddress string `yaml:"bindAddress,omitempty"`
}

//-------------------------------------------------------------------------------------------------

// IngressSelector can be used to limit operations to ingresses with a specific class.
type IngressSelector struct {
	IngressClass *string `yaml:"ingressClass,omitempty"`
}

// IntegrationConfigs describes the configurations for all integrations.
type IntegrationConfigs struct {
	ExternalDNS *ExternalDNSIntegrationConfig `yaml:"externalDNS"`
	CertManager *CertManagerIntegrationConfig `yaml:"certManager"`
}

// ExternalDNSIntegrationConfig describes the configuration for the external-dns integration.
// Exactly one of target and target IPs should be set.
type ExternalDNSIntegrationConfig struct {
	TargetService *ServiceRef `yaml:"targetService,omitempty"`
	TargetIPs     []string    `yaml:"targetIPs,omitempty"`
}

// CertManagerIntegrationConfig describes the configuration for the cert-manager integration.
type CertManagerIntegrationConfig struct {
	Template v1.Certificate `yaml:"certificateTemplate"`
}

// ServiceRef uniquely describes a Kubernetes service.
type ServiceRef struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// IssuerRef uniquely references a cert-manager issuer.
type IssuerRef struct {
	Kind string `yaml:"kind"`
	Name string `yaml:"name"`
}
