KIND_CLUSTER_NAME ?= switchboard-tests
SED = sed -i '' -E

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

#--------------------------------------------------------------------------------------------------
##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

#--------------------------------------------------------------------------------------------------
##@ Development

.PHONY: generate
generate: controller-gen ## Generate code for custom resources
	$(CONTROLLER_GEN) object paths="./..."

.PHONY: docs
docs: ## Generate helm docs in chart/README.md
	cd $(CURDIR)/chart
	helm-docs
	$(SED) '/external-dns\.sources/d' README.md
	$(SED) '/external-dns\.crd/d' README.md
	$(SED) '/cert-manager\.installCRDs/d' README.md

.PHONY: lint
lint: ## Lint the code with golangci-lint.
	golangci-lint run --exclude-use-default=false -E goimports -E revive --timeout 10m ./...

.PHONY: test
test: ## Run unit tests
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
	go test ./... -coverprofile cover.out

e2e-tests: create-cluster ## Run end-to-end tests
	cd $(CURDIR)/chart && bats tests -t

#--------------------------------------------------------------------------------------------------
##@ Build

.PHONY: build
build: generate ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: generate ## Run a controller from your host.
	go run cmd/main.go --config dev/config.yaml

#--------------------------------------------------------------------------------------------------
##@ Kubernetes

.PHONY: create-cluster
create-cluster: ## Create a local Kubernetes cluster
	kind get clusters | grep -q "^${KIND_CLUSTER_NAME}$$" || \
		kind create cluster --name ${KIND_CLUSTER_NAME}
	kubectl config use-context kind-${KIND_CLUSTER_NAME}

.PHONY: setup-cluster
setup-cluster: create-cluster ## Set up the currently connected Kubernetes cluster
	kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.13.5/config/manifests/metallb-native.yaml
	kubectl wait -n metallb-system --for=condition=ready pod -l app=metallb --timeout=120s
	kubectl apply -f https://kind.sigs.k8s.io/examples/loadbalancer/metallb-config.yaml
	kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v2.6/docs/content/reference/dynamic-configuration/traefik.containo.us_ingressroutes.yaml
	kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.8.0/cert-manager.yaml
	helm repo add bitnami https://charts.bitnami.com/bitnami
	helm upgrade --install --set crd.create=true --wait external-dns bitnami/external-dns
	kubectl apply -f dev/manifests/ca-secret.yaml
	kubectl apply -f dev/manifests/tls-issuer.yaml

.PHONY: teardown-cluster
teardown-cluster: ## Tear down a locally running Kubernetes cluster
	kind delete cluster --name ${KIND_CLUSTER_NAME} || :

#--------------------------------------------------------------------------------------------------
##@ Tool Installation

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0)

#--------------------------------------------------------------------------------------------------
# HELPERS
#--------------------------------------------------------------------------------------------------
# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
