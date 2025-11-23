#!/bin/bash -e

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )

export CONFIG_DIR=/var/snap/openvpn/current/config/web
echo "${CONFIG_DIR}"
cd $DIR/web
export EASYRSA_VARS_FILE=$DIR/config/easyrsa/vars
exec ./openvpn-web-ui --config=${CONFIG_DIR}
