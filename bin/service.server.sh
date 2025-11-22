#!/bin/bash -e

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )

CONFIG_DIR=${SNAP_DATA}/openvpn
SERVER_CONF=${CONFIG_DIR}/server.conf
export LD_LIBRARY_PATH=${DIR}/openvpn/lib
mkdir -p /dev/net
if [ ! -c /dev/net/tun ]; then
  mknod /dev/net/tun c 10 200
fi
echo 1 > /proc/sys/net/ipv4/ip_forward || true
echo 1 > /proc/sys/net/ipv6/conf/all/forwarding || true
# Prefer nftables if available; fallback to iptables
if command -v nft >/dev/null 2>&1; then
    # Ensure ip nat table and POSTROUTING chain exist
    nft list table ip nat >/dev/null 2>&1 || nft add table ip nat || true
    nft list chain ip nat POSTROUTING >/dev/null 2>&1 || nft 'add chain ip nat POSTROUTING { type nat hook postrouting priority 100 ; policy accept ; }'

    # Add masquerade rule for VPN subnet if not present
    if ! nft list chain ip nat POSTROUTING 2>/dev/null | grep -qE 'ip saddr 10\.8\.0\.0/24 .* oifname "eth0".* masquerade'; then
        nft add rule ip nat POSTROUTING ip saddr 10.8.0.0/24 oifname "eth0" counter masquerade
    fi
else
    if ! iptables -t nat -C POSTROUTING -s 10.8.0.0/24 -o eth0 -j MASQUERADE; then
        iptables -t nat -A POSTROUTING -s 10.8.0.0/24 -o eth0 -j MASQUERADE
    fi
fi

#IPV6=$(/snap/platform/current/bin/cli ipv6)
#IPV6_65=$(/snap/platform/current/bin/cli ipv6 prefix 65)
#if ! ip6tables -t nat -A POSTROUTING -s ${IPV6_65} -j SNAT –to ${IPV6} ; then
#    ip6tables -t nat -A POSTROUTING -s ${IPV6_65} -j SNAT –to ${IPV6}
#fi

while [ ! -f ${SERVER_CONF} ]
do
  echo "waiting for ${SERVER_CONF}"
  sleep 1
done
exec $DIR/openvpn/sbin/openvpn.sh --daemon openvpn --config ${SERVER_CONF} --cd ${CONFIG_DIR}
