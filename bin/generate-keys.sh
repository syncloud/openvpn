#!/bin/bash -ex

CA_NAME=LocalCA
SERVER_NAME=server
export KEY_NAME=$CA_NAME
echo "Generating CA cert"
/snap/openvpn/current/easyrsa/easyrsa --batch build-ca nopass | logger 

export KEY_NAME=$SERVER_NAME
echo "Generating server cert"
/snap/openvpn/current/easyrsa/easyrsa --batch build-server-full $SERVER_NAME nopass | logger
ls -la /var/snap/openvpn/current/pki | logger