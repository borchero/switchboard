# 1) Add the registry configuration to the kind cluster nodes
REGISTRY_DIR="/etc/containerd/certs.d/localhost:5001"
for node in $(kind get nodes --name switchboard); do
  docker exec "${node}" mkdir -p "${REGISTRY_DIR}"
  cat <<EOF | docker exec -i "${node}" cp /dev/stdin "${REGISTRY_DIR}/hosts.toml"
[host."http://kind-registry:5000"]
EOF
done

# 2) Connect the registry to the cluster
if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "kind-registry")" = 'null' ]; then
  docker network connect "kind" "kind-registry"
fi
