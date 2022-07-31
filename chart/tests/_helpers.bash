function install() {
    # Install Traefik ingress route CRD
    kubectl apply -f https://raw.githubusercontent.com/traefik/traefik/v2.6/docs/content/reference/dynamic-configuration/traefik.containo.us_ingressroutes.yaml
    # Install chart
    helm repo add bitnami https://charts.bitnami.com/bitnami
    helm repo add jetstack https://charts.jetstack.io
    helm dependency build
    helm install \
        --values ${BATS_TEST_DIRNAME}/values/base.yaml \
        switchboard ${BATS_TEST_DIRNAME}/..
}

function uninstall() {
    helm uninstall switchboard
}

function await_pod_ready() {
    POD_NAME=$1

    check() {
        kubectl get pod $1 -o json | \
            jq -r 'select(
                .status.phase == "Running"
                and (
                    [ .status.conditions[] | select(.type == "Ready" and .status == "True") ] 
                    | length
                ) == 1
            )'
    }

    for i in `seq 60`; do
        if [ -n "$(check ${POD_NAME})" ]; then
            echo "${POD_NAME} is ready."
            return 0
        fi
        sleep 2
    done

    echo "${POD_NAME} never became ready."
    return 1
}

function await_pod_running() {
    POD_NAME=$1

    check() {
        kubectl get pod $1 -o json | \
            jq -r 'select(
                .status.phase == "Running"
                and ([ 
                    .status.conditions[]
                    | select(.type == "Initialized" and .status == "True")
                ] | length) == 1
            )'
    }

    for i in `seq 60`; do
        if [ -n "$(check ${POD_NAME})" ]; then
            echo "${POD_NAME} is running."
            return 0
        fi
        sleep 2
    done

    echo "${POD_NAME} never became running."
    return 1
}
