#!/usr/bin/env bash

SERVER_CONF=/var/snap/openvpn/current/openvpn/server.conf

case ${reason} in
    BOUND6|EXPIRE6|REBIND6|REBOOT6|RENEW6)
        if [[ -n "${new_ip6_prefix}" ]]; then
            if [[ "${new_ip6_prefix}" != "${old_ip6_prefix}" ]]; then
                # enable on new prefix
                logger -t "openvpn-ipv6" "enable ipv6: ${new_ip6_prefix}"
                sed -i 's@.*server-ipv6.*@server-ipv6 '${new_ip6_prefix}'@g' ${SERVER_CONF}
                snap restart openvpn
            fi
        else
            # disable ipv6
            logger -t "openvpn-ipv6" "disable ipv6: ${new_ip6_prefix}"
            sed -i 's@.*server-ipv6.*@#server-ipv6@g' ${SERVER_CONF}
            snap restart openvpn
        fi
        ;;
esac
