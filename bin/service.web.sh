#!/bin/bash -e

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )

if [[ -z "$1" ]]; then
    echo "usage $0 [start]"
    exit 1
fi

export CONFIG_DIR=${SNAP_DATA}/config/web

echo "${CONFIG_DIR}" | logger -t openvpn-web

case $1 in
start)
    cd $DIR/web
    exec ./openvpn-web-ui --config=${CONFIG_DIR}  | logger -t openvpn-web
    ;;
*)
    echo "not valid command"
    exit 1
    ;;
esac
