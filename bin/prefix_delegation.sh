#!/usr/bin/env bash

SERVER_CONF=/var/snap/openvpn/current/openvpn/server.conf
CURRENT_IPV6_PREFIX_FILE=/var/snap/openvpn/current/ipv6_prefix

case $reason in
    BOUND6|EXPIRE6|REBIND6|REBOOT6|RENEW6)
        logger -t "openvpn-ipv6" "event reason: $reason, new_ip6_prefix: $new_ip6_prefix, old_ip6_prefix: $old_ip6_prefix"
        current_ip6_prefix=""
        if [ -f $CURRENT_IPV6_PREFIX_FILE ]; then
            current_ip6_prefix=$(cat $CURRENT_IPV6_PREFIX_FILE)
        fi
        if [ -n "$new_ip6_prefix" ]; then
            if [ "$new_ip6_prefix" != "$current_ip6_prefix" ]; then
                # enable on new prefix
                ip6_no_prefix=$(echo $new_ip6_prefix | cut -f1 -d'/')
                prefix=$(echo $new_ip6_prefix | cut -f2 -d'/')
                if [ "$prefix" -gt 112 ]; then
                    logger -t "openvpn-ipv6" "ipv6 prefix is greater than 112, not supported: $ip6"
                    exit
                fi
                ip6=$new_ip6_prefix
                if [ "$prefix" -lt 64 ]; then
                    ip6="$ip6_no_prefix/64"
                fi
                logger -t "openvpn-ipv6" "enable ipv6: $ip6"
                sed -i 's@.*server-ipv6.*@server-ipv6 '$ip6'@g' ${SERVER_CONF}
                snap restart openvpn
            fi
        else
            if [ "$old_ip6_prefix" == "$current_ip6_prefix" ]; then
                # disable ipv6
                logger -t "openvpn-ipv6" "disable ipv6"
                sed -i 's@.*server-ipv6.*@#server-ipv6@g' ${SERVER_CONF}
                snap restart openvpn
            fi
        fi
        ;;
esac
