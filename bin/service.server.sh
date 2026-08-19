#!/bin/bash -e

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )

CONFIG_DIR=/var/snap/openvpn/current/openvpn
SERVER_CONF=${CONFIG_DIR}/server.conf

mkdir -p /dev/net
if [ ! -c /dev/net/tun ]; then
  mknod /dev/net/tun c 10 200
fi

${DIR}/bin/firewall.sh

while [ ! -f ${SERVER_CONF} ]
do
  echo "waiting for ${SERVER_CONF}"
  sleep 1
done

exec ${DIR}/openvpn/sbin/openvpn.sh --config ${SERVER_CONF} --cd ${CONFIG_DIR}
