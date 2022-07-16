KIND_CLUSTER_NAME ?= switchboard-chart-tests
SED = sed -i '' -E

#--------------------------------------------------------------------------------------------------
# DOCS
#--------------------------------------------------------------------------------------------------
docs:
	helm-docs
	$(SED) '/external-dns\.sources/d' README.md
	$(SED) '/external-dns\.crd/d' README.md
	$(SED) '/cert-manager\.installCRDs/d' README.md

#--------------------------------------------------------------------------------------------------
# TESTING
#--------------------------------------------------------------------------------------------------
e2e-tests: create-cluster
	bats $(CURDIR)/tests -t

#--------------------------------------------------------------------------------------------------
# CLUSTER PROVISIONING
#--------------------------------------------------------------------------------------------------
create-cluster:
	kind get clusters | grep -q "^${KIND_CLUSTER_NAME}$$" || \
	kind create cluster \
	--name ${KIND_CLUSTER_NAME} \
	--config $(CURDIR)/tests/kind/config.yaml
	kubectl config use-context kind-${KIND_CLUSTER_NAME}

teardown-cluster:
	kind delete cluster --name ${KIND_CLUSTER_NAME} || :
