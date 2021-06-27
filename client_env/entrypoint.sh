#!/bin/bash -e

ID_FILE="/.machine-id"
dbus-uuidgen --ensure="${ID_FILE}"

ID=$(cat "${ID_FILE}")
echo "Generated machine id: ${ID}"
echo "${ID}" >/etc/machine-id
echo "${ID}" >/var/lib/dbus/machine-id

# shellcheck disable=SC2068
exec $@
