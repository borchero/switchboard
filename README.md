# Switchboard

![License](https://img.shields.io/github/license/borchero/switchboard)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/switchboard)](https://artifacthub.io/packages/search?repo=switchboard)

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

Switchboard solves these two issues by integrating the Traefik `IngressRoute` CRD with
[cert-manager](https://cert-manager.io) and
[external-dns](https://github.com/kubernetes-sigs/external-dns). Every time the user creates an
`IngressRoute` resource in the cluster, Switchboard performs the following actions:

- If the `IngressRoute` has the field `.spec.tls.secretName` set, it creates a cert-manager
  `Certificate`. A running cert-manager installation will pick up the certificate, issue it, and
  create a secret with the desired name. Traefik will then automatically secure the connection with
  this certificate.
- If any of the routes (`.spec.routes`) of the `IngressRoute` has an entry which references a host
  (e.g. a rule `` Host(`my.example.com`) ``), Switchboard creates a `DNSEndpoint` resource (which
  is a CRD defined by external-dns). Depending on your external-dns configuration, this will create
  a DNS A record in your configured provider, using the rule's host (e.g. `my.example.com`) and the
  external IP of your Traefik service as the value (or the internal IP if it does not have an
  external one).

Note that the `IngressRoute` resources that are processed by Switchboard depend on its configured
**groups** (see below).

_Note: This version of Switchboard is a complete rewrite of Switchboard v0.1 which will not be
maintained anymore. Please refer to the appropriate tags in this repository if you still need to
use it. Be aware that this version of Switchboard provides significantly more functionality while
being considerably more reliable due to its integration with external-dns._

## Installation

Switchboard can be conveniently installed using [Helm](https://helm.sh). For a full installation
guide, consult the [chart repository](https://github.com/borchero/switchboard-chart).

## Example

As outlined above, Switchboard process Traefik `IngressRoute` resources and (optionally) creates
`Certificate` and `DNSEndpoint` resources. For example, you might create the following ingress
route:

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
    # You can also set TLS domains here, overwriting any hosts found in the routes:
    # ---
    # domains:
    #   - main: example.com
    #     sans:
    #       - example.net
    #       - www.example.com
```

As this ingress is TLS-protected, Switchboard creates a certificate:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  # The name is automatically generated from the name of the ingress route
  name: my-ingress-tls
  labels:
    kubernetes.io/managed-by: switchboard
spec:
  # The issuer reference is obtained from Switchboard's global configuration
  issuerRef:
    kind: ClusterIssuer
    name: ca-issuer
  # The DNS names are extracted from the ingress route's hosts
  dnsNames:
    - www.example.com
  # The secret name is copied from the ingress route definition
  secretName: www-tls-certificate
```

Further, it creates a DNS endpoint pointing to your Traefik instance that can be picked up by
`external-dns`:

```yaml
apiVersion: externaldns.k8s.io/v1alpha1
kind: DNSEndpoint
metadata:
  # The name is the same as the ingress's name
  name: my-ingress
  labels:
    kubernetes.io/managed-by: switchboard
spec:
  # The endpoints are automatically obtained from all hostnames in the ingress route's rules
  endpoints:
    - dnsName: www.example.com
      recordTTL: 300
      recordType: A
      targets:
        # The target is the public (or, if unavailable, private) IP address of your Traefik instance
        - 10.96.0.10
```
