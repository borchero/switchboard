# Switchboard Helm Chart

![Type: application](https://img.shields.io/badge/Type-application-informational)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/switchboard)](https://artifacthub.io/packages/search?repo=switchboard)
![License](https://img.shields.io/github/license/borchero/switchboard-chart)

This direcoty contains the Helm chart as well as detailed instructions for deploying the
Switchboard Kubernetes operator. Please read through this repository's root README to understand
how Switchboard works.

## Installation

### Prerequisities

Since Switchboard processes Traefik CRDs, you must make sure that your Kubernetes cluster has all
[Traefik v2](https://github.com/traefik/traefik) CRDs installed.

Depending on the integrations that you want to enable, you further need to have the following
components installed:

- **cert-manager integration**: Requires [cert-manager](https://cert-manager.io) and its CRDs to be
  installed. Can optionally be done by setting `cert-manager.install = true` in this chart.
- **external-dns integration**: Requires
  [external-dns](https://github.com/kubernetes-sigs/external-dns) along with its `DNSEndpoint` CRD
  installed. Can optionally be done by setting `external-dns.install = true` in this chart.

### Install

Switchboard can be installed with Helm version `>= 3.7.0`. For Helm version `< 3.8.0`, you need to
set `HELM_EXPERIMENTAL_OCI=1`. Then you can install the chart directly with the following command:

```bash
helm install switchboard oci://ghcr.io/borchero/charts/switchboard
```

By default, this installs Switchboard with no integrations enabled, i.e. it will not create any
resources. Integrations can be enabled by setting `integrations.<name>.enabled` to `true`. Consult
the configuration options to check which integrations you want to enable.

_Note: You can check your Helm version via `helm version`._

## Values

The following lists all values that may be set when installing this chart (see
[`values.yaml`](./values.yaml) for a more structured overview):

| Key                                              | Type   | Default                          | Description                                                                                                                                                                                                                                                                                                                                                                 |
| ------------------------------------------------ | ------ | -------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| cert-manager.install                             | bool   | `false`                          | Whether the cert-manager chart should be installed. See: https://artifacthub.io/packages/helm/cert-manager/cert-manager                                                                                                                                                                                                                                                     |
| certificateIssuer.create                         | bool   | `false`                          | Whether an ACME certificate issuer should be created for use with cert-manager.                                                                                                                                                                                                                                                                                             |
| certificateIssuer.email                          | string | `nil`                            |                                                                                                                                                                                                                                                                                                                                                                             |
| certificateIssuer.solvers                        | list   | `[]`                             | The solvers to use for verifying that the domain is owned in the ACME challenge. See: https://cert-manager.io/docs/configuration/acme/                                                                                                                                                                                                                                      |
| external-dns.install                             | bool   | `false`                          | Whether the external-dns chart should be installed. If installed manually, make sure to add the `crd` item to the sources. See: https://artifacthub.io/packages/helm/external-dns/external-dns                                                                                                                                                                              |
| image.name                                       | string | `"ghcr.io/borchero/switchboard"` | The switchboard image to use.                                                                                                                                                                                                                                                                                                                                               |
| image.tag                                        | string | `"0.3.0"`                        | The switchboard image tag to use.                                                                                                                                                                                                                                                                                                                                           |
| integrations.certManager.enabled                 | bool   | `false`                          | Whether the cert-manager integration should be enabled. If enabled, `Certificate` resources are created by Switchboard. Setting this to `true` requires specifying an issuer via `integrations.certManager.issuer` or letting the chart create its own issuer by setting `certificateIssuer.create = true` and specifying additional properties for the certificate issuer. |
| integrations.certManager.issuer.kind             | string | `nil`                            | The kind of certificate issuer to use for obtaining TLS certificates.                                                                                                                                                                                                                                                                                                       |
| integrations.certManager.issuer.name             | string | `nil`                            | The name of the certificate issuer to use for obtaining TLS certificates.                                                                                                                                                                                                                                                                                                   |
| integrations.externalDNS.enabled                 | bool   | `false`                          | Whether the external-dns integration should be enabled. If enabled `DNSEndpoint` resources are created by Switchboard. Setting this to `true` requires specifying the target via `integrations.externalDNS.target`.                                                                                                                                                         |
| integrations.externalDNS.targetIPs               | list   | `[]`                             | The static IP addresses that created DNS records should point to. Must not be provided if the target service is set.                                                                                                                                                                                                                                                        |
| integrations.externalDNS.targetService.name      | string | `nil`                            | The name of the (Traefik) service whose IP address should be used for DNS records.                                                                                                                                                                                                                                                                                          |
| integrations.externalDNS.targetService.namespace | string | `nil`                            | The namespace of the (Traefik) service whose IP address should be used for DNS records.                                                                                                                                                                                                                                                                                     |
| metrics.enabled                                  | bool   | `true`                           | Whether the metrics endpoint should be enabled.                                                                                                                                                                                                                                                                                                                             |
| metrics.port                                     | int    | `9090`                           | The port on which Prometheus metrics can be scraped on path `/metrics`.                                                                                                                                                                                                                                                                                                     |
| podAnnotations                                   | object | `{}`                             | Annotations to set on the switchboard pod.                                                                                                                                                                                                                                                                                                                                  |
| podMonitor.create                                | bool   | `false`                          | Whether a PodMonitor should be created which can be used to scrape the metrics endpoint. Ignored if `metrics.enabled` is set to `false`                                                                                                                                                                                                                                     |
| podMonitor.namespace                             | string | `nil`                            | The namespace where the monitor should be created in. Defaults to the release namespace.                                                                                                                                                                                                                                                                                    |
| replicas                                         | int    | `1`                              | The number of manager replicas to use.                                                                                                                                                                                                                                                                                                                                      |
| resources                                        | object | `{}`                             | The resources to use for the operator.                                                                                                                                                                                                                                                                                                                                      |
| selector.ingressClass                            | string | `nil`                            | When set, Switchboard only processes ingress routes with the `kubernetes.io/ingress.class` annotation set to this value.                                                                                                                                                                                                                                                    |
