#!/bin/bash -e

VPN_SUBNET=10.8.0.0/24
EXTERNAL_IFACE=${EXTERNAL_IFACE:-eth0}

echo 1 > /proc/sys/net/ipv4/ip_forward || true
echo 1 > /proc/sys/net/ipv6/conf/all/forwarding || true

if command -v nft >/dev/null 2>&1; then
    nft list table ip nat >/dev/null 2>&1 || nft add table ip nat || true
    nft list chain ip nat POSTROUTING >/dev/null 2>&1 || \
      nft 'add chain ip nat POSTROUTING { type nat hook postrouting priority 100 ; policy accept ; }'

    if ! nft list chain ip nat POSTROUTING 2>/dev/null | \
        grep -qE "ip saddr ${VPN_SUBNET//./\\.}.* oifname \"${EXTERNAL_IFACE}\".* masquerade"; then
        nft add rule ip nat POSTROUTING ip saddr ${VPN_SUBNET} oifname "${EXTERNAL_IFACE}" counter masquerade
    fi
else
    if ! iptables -t nat -C POSTROUTING -s ${VPN_SUBNET} -o ${EXTERNAL_IFACE} -j MASQUERADE 2>/dev/null; then
        iptables -t nat -A POSTROUTING -s ${VPN_SUBNET} -o ${EXTERNAL_IFACE} -j MASQUERADE
    fi
fi
