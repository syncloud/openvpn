#!/bin/bash -e

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )

if [[ -z "$1" ]]; then
    echo "usage $0 [start]"
    exit 1
fi

SERVER_CONF=${SNAP_DATA}/openvpn/server.conf

case $1 in
start)
    mkdir -p /dev/net
    if [ ! -c /dev/net/tun ]; then
      mknod /dev/net/tun c 10 200
    fi
    while [ ! -f ${SERVER_CONF} ]
    do
      echo "waiting for ${SERVER_CONF}"
      sleep 1
    done
    exec $DIR/openvpn/sbin/openvpn --config ${SERVER_CONF} --cd ${SNAP_DATA}/openvpn
    ;;
*)
    echo "not valid command"
    exit 1
    ;;
esac
