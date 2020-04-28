# This script is used to generate the custom resources YAML files.

set -e

ROOT=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )/..

# 1) Install controller-gen
CONTROLLER_GEN_TMP_DIR=$(mktemp -d)
cd $CONTROLLER_GEN_TMP_DIR
go mod init tmp
go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0
rm -rf $CONTROLLER_GEN_TMP_DIR
CONTROLLER_GEN=$GOBIN/controller-gen

# 2) Generate manifests
cd $ROOT/source
$CONTROLLER_GEN \
    crd:crdVersions=v1beta1,trivialVersions=true \
    object \
    paths="github.com/borchero/switchboard/api/v1alpha1" \
    output:crd:dir=$ROOT/deploy/helm/crds \
    output:object:dir=$ROOT/source/api/v1alpha1

mv $ROOT/deploy/helm/crds/switchboard.borchero.com_dnszones.yaml \
    $ROOT/deploy/helm/crds/dnszones.yaml
mv $ROOT/deploy/helm/crds/switchboard.borchero.com_dnsrecords.yaml \
    $ROOT/deploy/helm/crds/dnsrecords.yaml
mv $ROOT/deploy/helm/crds/switchboard.borchero.com_dnszonerecords.yaml \
    $ROOT/deploy/helm/crds/dnszonerecords.yaml
mv $ROOT/deploy/helm/crds/switchboard.borchero.com_dnsresources.yaml \
    $ROOT/deploy/helm/crds/dnsresources.yaml
