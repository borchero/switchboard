function install() {
    helm dependency build ${BATS_TEST_DIRNAME}/../chart
    helm install \
        --values ${BATS_TEST_DIRNAME}/config/switchboard.yaml \
        switchboard ${BATS_TEST_DIRNAME}/../chart
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

function expect_resource_exists() {
    RESOURCE_TYPE=$1
    RESOURCE_NAME=$2
    kubectl get $RESOURCE_TYPE $RESOURCE_NAME
}

function expect_resource_not_exists() {
    RESOURCE_TYPE=$1
    RESOURCE_NAME=$2
    kubectl get $RESOURCE_TYPE $RESOURCE_NAME && return 1 || return 0
}
