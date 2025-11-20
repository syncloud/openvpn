#!/bin/bash -e

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )

export CONFIG_DIR=${SNAP_DATA}/config/web
echo "${CONFIG_DIR}" | logger -t openvpn-web
cd $DIR/web
exec ./openvpn-web-ui --config=${CONFIG_DIR}  | logger -t openvpn-web
