#!/bin/bash -e

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )

if [[ -z "$1" ]]; then
    echo "usage $0 [start]"
    exit 1
fi

CONFIG_DIR=${SNAP_DATA}/config/openvpn
SERVER_CONF=${CONFIG_DIR}/server.conf

export LD_LIBRARY_PATH=${DIR}/openvpn/lib

case $1 in
start)
    mkdir -p /dev/net
    if [ ! -c /dev/net/tun ]; then
      mknod /dev/net/tun c 10 200
    fi
    echo "1" > /proc/sys/net/ipv4/ip_forward
    iptables -t nat -D POSTROUTING 1 || true
    iptables -t nat -A POSTROUTING -s 10.8.0.0/24 -o eth0 -j MASQUERADE
    while [ ! -f ${SERVER_CONF} ]
    do
      echo "waiting for ${SERVER_CONF}"
      sleep 1
    done
    exec $DIR/openvpn/sbin/openvpn --daemon openvpn --config ${SERVER_CONF} --cd ${CONFIG_DIR}
    ;;
*)
    echo "not valid command"
    exit 1
    ;;
esac
