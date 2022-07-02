#!/bin/bash -e

CA_NAME=LocalCA
SERVER_NAME=server
export KEY_NAME=$CA_NAME
echo "Generating CA cert"
/snap/openvpn/current/easy-rsa/easyrsa --batch build-ca nopass

export KEY_NAME=$SERVER_NAME
echo "Generating server cert"
/snap/openvpn/current/easy-rsa/build-key-server $SERVER_NAME
/snap/openvpn/current/easy-rsa/easyrsa --batch build-server-full $SERVER_NAME nopass
h build-server-full $SERVER_NAME nopass
