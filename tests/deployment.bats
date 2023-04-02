#!/usr/bin/env bats

load lib/helpers

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

@test "check resources created" {
    kubectl apply -f ${BATS_TEST_DIRNAME}/resources/ingress.yaml
    # Wait for the Switchboard manager to pick up the changes
    sleep 0.3
    expect_resource_exists dnsendpoint my-ingress
    expect_resource_exists certificate my-ingress-tls
}

@test "check resources deleted" {
    kubectl delete ingressroute my-ingress
    # Wait for the Switchboard manager to pick up the changes
    sleep 0.3
    expect_resource_not_exists dnsendpoint my-ingress
    expect_resource_not_exists certificate my-ingress-tls
}

function teardown_suite() {
    uninstall
}
