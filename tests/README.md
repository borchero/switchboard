# Testing

In the following, a very simple set of commands is described to check for Switchboard's
functionality. It is _not_ intended to be used in a CI pipeline and does _not_ properly test this
component.

## Prerequisites

```
minikube start -p switchboard
kubectl apply --validate=false -f \
    https://github.com/jetstack/cert-manager/releases/download/v0.14.2/cert-manager.yaml
```

## GCP Access

```
kubectl apply -f tests/secret.yaml
```

## Install Switchboard

Without pulling the Docker image:

```
kubectl apply -f deploy/helm/crds
cd source
go run main.go
```

With pulling the Docker image:

```
helm install switchboard deploy/helm
```

## Install Zones

```
kubectl apply -f tests/crds/zone-1.yaml
kubectl apply -f tests/crds/zone-2.yaml
kubectl apply -f tests/crds/record-1.yaml
```

Now, check that in the Google Cloud Console, the following entries are inserted:

- In `borchero-com`, `_switchboardtest.borchero.com` should be set to the internal IP of the Minikube node (`kubectl get node switchboard -o json | jq '.status.addresses[0].address'`). The TTL should be set to 300. Additionally, the CNAME record `_switchboardtest_cname.borchero.com` should point to the previous address.

- In `borchero-com-private`, `_switchboardtest.borchero.com` should be set to the internal IP of the cert-manager service (`kubectl get svc -n cert-manager cert-manager -o json | jq .spec.clusterIP`). The TTL should be set to 7200. Additionally, the CNAME record `_switchboardtest_cname.borchero.com` should point to the previous address.

## Shutdown

First, delete the record and ensure that no more records are set in the Google Cloud Console:

```
kubectl delete dnsrecord my-dns-record
```

Afterwards, we can shutdown minikube.

```
minikube delete -p switchboard
```
