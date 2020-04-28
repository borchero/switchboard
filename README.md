# Switchboard

![Docker Image Version](https://img.shields.io/docker/v/borchero/switchboard)

Switchboard is a tool that manages DNS zones and their A/CNAME records for arbitrary backends. It
runs as Kubernetes controller and watches for custom resources `DNSZone` and `DNSRecord`.

While [External DNS](https://github.com/kubernetes-sigs/external-dns) is already well-established
and works well for most use-cases, we decided to write our own tool for the following reasons:

- Switchboard does not enforce the usage of the `Ingress` resource. As ingress controllers mature,
  custom resources such as the `IngressRoute` introduced by
  [Traefik 2.0](https://docs.traefik.io/migration/v1-to-v2/) may become more common.
- In its core, Switchboard is designed to work with multiple (overlapping) DNS zones and can
  therefore be used for split horizon configurations with any backend. Most importantly: with
  Google Cloud DNS. Besides, we can use cluster IPs for our internal DNS.
- We have a more fine-grained control over the DNS records. For example, we can easily set the time
  to live for individual records.
- Lastly, Switchboard integrates tightly with [cert-manager](https://cert-manager.io/) to generate
  TLS certificates for DNS records (i.e. a set of domains).

In summary, Switchboard provides a more native integration of external DNS records into Kubernetes
than external-dns is able to do.

**_Caveat: Use this component with care. Although it has been tested for common use cases, there
are no comprehensive tests and performance for large clusters might become an issue._**

## Installation

Switchboard can be installed using Helm as follows:

```
helm repo add borchero https://charts.borchero.com
helm install switchboard borchero/switchboard --version
```

## Resources

In total, Switchboard makes use of 4 CRDs, however, only two of them should ever be created by the
user. Namely, `DNSZone` and `DNSRecord`.

_Note: As Switchboard does not currently employ any webhooks, validation might fail when trying to
create "internal" CRDs manually._

### DNS Zone

A DNS zone models a zone as defined at the cloud provider. Every zone therefore includes a reference
to _exactly one_ backend and authentication credentials (as provided by a secret).

It further provides a way for having a _template IP source_. The IP source might e.g. be a static IP
(as all domains are routed to a public load balancer) or the public IP of a service (if an ingress
controller such as Traefik is used).

DNS zones are cluster-level resources and you must therefore always provide a namespace when
referring to Kubernetes resources from a zone.

The `DNSZone` resource can be specified as follows:

```yaml
apiVersion: switchboard.borchero.com/v1alpha1
kind: DNSZone

metadata:
  name: my-zone

spec:
  # Must specify exactly one backend
  clouddns:
    # Required zone name as specified on the Google Cloud Platform
    zoneName: my-zone-name
    # Credentials for a service account
    credentialsSecret:
      name: secret-name
      namespace: secret-namespace
      key: secret-key
  # May specify *at most one* record template
  recordTemplate:
    # Source the IP of a random node matching a set of labels
    nodeIP:
      matchLabels: {}
      type: ExternalIP | InternalIP
    # Source the IP of a service - either external or internal
    serviceIP:
      name: my-service
      namespace: my-namespace
      type: ExternalIP | ClusterIP
    # Use a static IP (e.g. reserved on the Google Cloud)
    staticIP:
      ip: 50.60.70.80
    # Additionally, the ttl may be set (the default is 300)
    ttl: 300
```

### DNS Record

A DNS record models a set of actual DNS records (referred to as DNS _resources_ in the following),
however, all these resources have the same endpoint. A DNS record may specify multiple subdomains
that get an A entry with the same IP and also CNAME records pointing to these A records.

A DNS record may further be added to multiple zones (which is useful for split horizon
configurations) with different IPs for each zone.

The `DNSRecord` resource can be specified as follows:

```yaml
apiVersion: switchboard.borchero.com/v1alpha1
kind: DNSRecord

metadata:
  name: my-record

spec:
  # The subdomains for the A records (@ matches no subdomain)
  hosts: ["@"]
  # The CNAME records referring to the first A record
  cnames: ["www"]
  # A TLS configuration to automatically obtain certificates for the domain and all cname records
  tls:
    certificateName: my-certificate
    secretName: my-certificate-secret
    issuer:
      kind: Issuer | ClusterIssuer
      name: my-issuer
  # A default TTL for the certificate (applies to all zones if given)
  ttl: 300
  # A set of zones to which this record should be added. It may define the IP source (i.e. the
  # contents of the `recordTemplate` block in the `DNSZone` resource) per zone.
  zones:
    # In this case, we override the default IP
    - name: my-zone
      staticIP:
        ip: 60.70.80.90
    # In this case, we use the default IP as provided by the `recordTemplate` block
    - name: my-other-zone
```

### Internal Resources

Internal resources are created by the Switchboard controller and can therefore be accessed via
`kubectl` or the Kubernetes API. They are _not_ intended to be interacted with.

#### DNS Zone Record

As a `DNSRecord` may be part of multiple zones, this record models a DNS record in a single zone.

#### DNS Resource

As every `DNSRecord` (as defined by Switchboard) may have multiple "aliases" (i.e. A records,
CNAME records), a `DNSResource` models the actual record that is pushed to the backend.

## Backends

Switchboard is designed to work with a multitude of backends that provide name resolution.
Currently, Google Cloud DNS is the only supported backend, but extending Switchboard is easy.

### Cloud DNS

In order to create DNS records for Google Cloud DNS, you should first make sure that you have a
service account with the `roles/dns.admin` role assigned. You should then create a Kubernetes
`Secret` that contains a key for the service account to be referenced by a DNS zone.

_Note: DNS Zones must already exist prior to referencing them via Switchboard. Although automatic
creation would be easy, we consider this to be outside this tool's realm._

## Limitations and Known Issues

- When updating a `DNSZone` to refer to a different backend, old entries are _not_ deleted from the
  original zone. Thus, it is recommended to first delete the old zone and subsequently create a new
  one with the same name.
- When deleting a `DNSZone` it may take up to 60 seconds (but no longer) until deletion is
  finished, depending on the number of records associated with the zone. This results from a
  hard-coded time-limit to attempt deleting all relevant entries from the zone. This may, however,
  be unsuccessful.
