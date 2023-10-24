SED := "sed -i '' -E"
KIND_CLUSTER_NAME := env_var_or_default("KIND_CLUSTER_NAME", "switchboard")

[private]
default:
  @just --list --unsorted

#--------------------------------------------------------------------------------------------------
# DEVELOPMENT
#--------------------------------------------------------------------------------------------------

# Setup your development environment (requires brew to be installed).
[macos]
setup:
  brew install \
    bats \
    golangci-lint \
    helm \
    helm-docs \
    kind \
    kubectl \
    yq

# Run the Switchboard controller from your host.
run:
  go run cmd/main.go --config dev/config.yaml

#--------------------------------------------------------------------------------------------------
# CI TASKS
#--------------------------------------------------------------------------------------------------

# Generate helm docs in `chart/README.md`.
docs:
  cd chart
  helm-docs
  {{SED}} '/external-dns\.sources/d' README.md
  {{SED}} '/external-dns\.crd/d' README.md
  {{SED}} '/cert-manager\.installCRDs/d' README.md

# Lint the code with `golangci-lint`.
lint:
  golangci-lint run --timeout 10m ./...

# Run unit tests.
unit-test: setup-cluster
	go test ./... -coverprofile cover.out

# Run end-to-end tests locally.
e2e-test IMAGE_NAME IMAGE_TAG: create-cluster
  -just e2e-test-ci {{IMAGE_NAME}} {{IMAGE_TAG}}
  yq -i 'del(.image)' tests/config/switchboard.yaml

# Run end-to-end tests in the CI.
e2e-test-ci IMAGE_NAME IMAGE_TAG:
  yq -i '.image.name = "{{IMAGE_NAME}}" | .image.tag = "{{IMAGE_TAG}}"' \
    tests/config/switchboard.yaml
  bats tests -t

#--------------------------------------------------------------------------------------------------
# CLUSTER
#--------------------------------------------------------------------------------------------------

# Create a local Kubernetes cluster for testing.
create-cluster:
  kind get clusters | grep -q '^{{KIND_CLUSTER_NAME}}$' || \
    kind create cluster --name {{KIND_CLUSTER_NAME}}
  kubectl config use-context kind-{{KIND_CLUSTER_NAME}}

# Set up the currently connected Kubernetes cluster for unit tests.
setup-cluster: create-cluster
  kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.13.9/config/manifests/metallb-native.yaml
  kubectl wait -n metallb-system --for=condition=ready pod -l app=metallb --timeout=240s
  kubectl apply -f https://kind.sigs.k8s.io/examples/loadbalancer/metallb-config.yaml
  kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v2.6/docs/content/reference/dynamic-configuration/traefik.containo.us_ingressroutes.yaml
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.11.0/cert-manager.yaml
  helm upgrade --install --set crd.create=true --wait external-dns oci://registry-1.docker.io/bitnamicharts/external-dns
  kubectl apply -f dev/manifests/ca-secret.yaml
  kubectl apply -f dev/manifests/tls-issuer.yaml

# Tear down a locally running Kubernetes test cluster.
teardown-cluster:
  kind delete cluster --name {{KIND_CLUSTER_NAME}} || :
