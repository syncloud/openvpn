#!/bin/bash

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )

if [[ -z "$1" ]]; then
    echo "usage $0 [start]"
    exit 1
fi

case $1 in
start)
    cd $DIR/web
    exec ./openvpn-web-ui --config=${SNAP_DATA}/config/web | logger -t openvpn-web
    ;;
*)
    echo "not valid command"
    exit 1
    ;;
esac
