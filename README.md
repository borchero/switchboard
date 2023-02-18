# Switchboard

![License](https://img.shields.io/github/license/borchero/switchboard)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/switchboard)](https://artifacthub.io/packages/search?repo=switchboard)
[![CI - Application](https://github.com/borchero/switchboard/actions/workflows/ci-application.yml/badge.svg?branch=main)](https://github.com/borchero/switchboard/actions/workflows/ci-application.yml)
[![CI - Chart](https://github.com/borchero/switchboard/actions/workflows/ci-chart.yml/badge.svg?branch=main)](https://github.com/borchero/switchboard/actions/workflows/ci-chart.yml)

Switchboard is a Kubernetes operator that automates the creation of DNS records and TLS
certificates when using [Traefik](https://github.com/traefik/traefik) v2 and its
[`IngressRoute` custom resource](https://doc.traefik.io/traefik/routing/providers/kubernetes-crd/#kind-ingressroute).

Traefik is an amazing reverse proxy and load balancer for Kubernetes, but has two major issues when
using it in production:

- You cannot use Traefik to automatically issue TLS certificates from Let's Encrypt when running
  multiple Traefik instances (see
  [the documentation](https://doc.traefik.io/traefik/providers/kubernetes-crd/#letsencrypt-support-with-the-custom-resource-definition-provider)).
- External tools do not support sourcing hostnames for DNS records from custom resources (including
  the Traefik `IngressRoute` CRD).

Switchboard solves these two issues by integrating the Traefik `IngressRoute` CRD with external
tools (_integrations_):

- [cert-manager](https://cert-manager.io) can be used to create TLS certificates: Switchboard
  automatically creates a `Certificate` resource when an `IngressRoute` has the
  `.spec.tls.secretName` field set. The DNS alt names are taken either from `.spec.tls.domains` or
  (if unavailable) extracted automatically from all rules. The created certificate will then be
  used by Traefik to secure the connection.
- [external-dns](https://github.com/kubernetes-sigs/external-dns) can be used to create DNS A
  records. First, DNS names are extracted from `.spec.tls.domains` and all rules as for the DNS alt
  names. Subsequently, a `DNSEndpoint` resource is created where all DNS names point to your
  Traefik instance. External-dns will pick up the `DNSEndpoint` and add appropriate DNS records in
  your configured provider.

Switchboard allows you to freely choose which integrations you want to use and can, thus, be easily
adopted incrementally.

_Note: This version of Switchboard is a complete rewrite of Switchboard v0.1 which will not be
maintained anymore. Please refer to the appropriate tags in this repository if you still need to
use it. Be aware that this version of Switchboard provides significantly more functionality while
being considerably more reliable due to its integration with external-dns._

## Installation

Switchboard can be conveniently installed using [Helm](https://helm.sh) version `>= 3.8.0`:

```bash
helm install switchboard oci://ghcr.io/borchero/charts/switchboard
```

For a full installation guide, consult the
[Switchboard Helm chart documentation](./chart/README.md).

## Usage

As mentioned above, Switchboard processes Traefik `IngressRoute` resources. Let's assume, we have
the following ingress route which forwards requests to an nginx backend:

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: IngressRoute
metadata:
  name: my-ingress
spec:
  routes:
    - kind: Rule
      match: Host(`www.example.com`) && PathPrefix(`/images`)
      services:
        - name: nginx
  tls:
    secretName: www-tls-certificate
```

Switchboard now automatically extracts information from the ingress route object:

- The ingress route is concerned with a single host, namely `www.example.com`.
- Requests should be TLS-protected and a TLS certificate should be put into the
  `www-tls-certificate` secret.

This information is now passed onto all _integrations_ that Switchboard is configured with.

### Integrations

Integrations are entirely independent of each other. Enabling an integration causes Switchboard to
generate an integration-specific resource (typically a CRD) for each ingress route that it
processes.

Consult the [Switchboard Helm chart documentation](./chart/README.md) for an overview of how to
enable individual integrations.

#### Cert-Manager

The cert-manager integration allows Switchboard to create a `Certificate` resource for an
`IngressRoute` if the ingress (1) specifies `.spec.tls.secretName` and (2) references at least one
host. Using the example ingress route from above, Switchboard creates the following resource:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  # The name is automatically generated from the name of the ingress route.
  name: my-ingress-tls
  labels:
    kubernetes.io/managed-by: switchboard
spec:
  # The issuer reference is obtained from the configuration of the cert-manager integration.
  issuerRef:
    kind: ClusterIssuer
    name: ca-issuer
  dnsNames:
    - www.example.com
  secretName: www-tls-certificate
```

#### External-DNS

The external-dns integration causes Switchboard to create a `DNSEndpoint` resource for an
`IngressRoute` if the ingress references at least one host. Given the example ingress route above,
Switchboard creates the following endpoint:

```yaml
apiVersion: externaldns.k8s.io/v1alpha1
kind: DNSEndpoint
metadata:
  # The name is the same as the ingress's name.
  name: my-ingress
  labels:
    kubernetes.io/managed-by: switchboard
spec:
  endpoints:
    - dnsName: www.example.com
      recordTTL: 300
      recordType: A
      targets:
        # The target is the public (or, if unavailable, private) IP address of your Traefik
        # instance. The Kubernetes service to source the IP from is obtained from the configuration
        # of the external-dns integration.
        - 10.96.0.10
```

### Customization

#### Manually Set Hosts

By default, Switchboard automatically extracts hosts from an ingress route by processing all rules
and extracting hosts from `` Host(`...`) `` blocks. If you want to specify the set of hosts that
are used for TLS certificates and DNS endpoints yourself, set `.spec.tls.domains`, e.g.:

```yaml
spec:
  tls:
    domains:
      - main: example.com
        sans:
          - www.example.com
```

#### Disable Processing of an Ingress Route

By default, Switchboard process all `IngressRoute` objects in your cluster. While you can constrain
Switchboard to only process objects with the `kubernetes.io/ingress.class` annotation set to a
specific value (see the
[Switchboard Helm chart documentation](https://github.com/borchero/switchboard-chart)), you can
also disable processing for individual ingress routes by setting an additional annotation:

```yaml
metadata:
  annotations:
    switchboard.borchero.com/ignore: "all"
```

By setting the `ignore` annotation to `all` (or `true`), Switchboard does not process the ingress
route at all. For more fine-grained control, the value of this annotation can also be set to a
comma-separated list of integrations (possible values `cert-manager`, `external-dns`).

## License

Switchboard is licensed under the [MIT License](./LICENSE).
