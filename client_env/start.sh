#!/bin/bash -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null && pwd)"
COMMAND="$*"

if [[ -z "${VERSION}" && -f "${DIR}/.version" ]]; then
  VERSION=$(xargs <"${DIR}/.version")
fi

if [[ -z "${VERSION}" ]]; then
  VERSION=latest
fi

IMAGE="ghcr.io/pomo-mondreganto/neo_env:${VERSION}"
CONTAINER_NAME="neo-${VERSION}"

echo "Using image: ${IMAGE}"

OUT=$(docker ps --filter "name=${CONTAINER_NAME}" --format "{{ .Names }}")

if [[ $OUT ]]; then
  echo "Container already exists"
  # shellcheck disable=SC2068
  docker exec -it "${CONTAINER_NAME}" ${COMMAND[@]}
else
  echo "Starting a new container"
  docker run -it \
    --rm \
    --volume "${DIR}":/work \
    --security-opt seccomp=unconfined \
    --security-opt apparmor=unconfined \
    --cap-add=NET_ADMIN \
    --privileged \
    --name "${CONTAINER_NAME}" \
    --hostname "${CONTAINER_NAME}" \
    "${IMAGE}" \
    "${COMMAND}"
fi
