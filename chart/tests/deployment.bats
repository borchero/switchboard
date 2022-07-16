#!/usr/bin/env bats

load _helpers

@test "check deployment running" {
    install
    sleep 3

    POD_NAME=$(
        kubectl get pod -l app.kubernetes.io/name=switchboard \
            -o jsonpath="{.items[0].metadata.name}"
    )
    await_pod_ready $POD_NAME
    await_pod_running $POD_NAME
}

function teardown() {
    uninstall
}
