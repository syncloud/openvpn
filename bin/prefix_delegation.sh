#!/usr/bin/env bash

SERVER_CONF=/var/snap/openvpn/current/openvpn/server.conf

case $reason in
    BOUND6|EXPIRE6|REBIND6|REBOOT6|RENEW6)
        logger -t "openvpn-ipv6" "event reason: $reason, new_ip6_prefix: $new_ip6_prefix, old_ip6_prefix: $old_ip6_prefix"
        if [ -n "$new_ip6_prefix" ]; then
            if [ "$new_ip6_prefix" != "$old_ip6_prefix" ]; then
                # enable on new prefix
                ip6_no_mask=$(echo $new_ip6_prefix | cut -f1 -d'/')
                ip6="$ip6_no_mask/64"
                logger -t "openvpn-ipv6" "enable ipv6: $ip6"
                sed -i 's@.*server-ipv6.*@server-ipv6 '$ip6'@g' ${SERVER_CONF}
                snap restart openvpn
            fi
        else
            # disable ipv6
            logger -t "openvpn-ipv6" "disable ipv6"
            sed -i 's@.*server-ipv6.*@#server-ipv6@g' ${SERVER_CONF}
            snap restart openvpn
        fi
        ;;
esac
