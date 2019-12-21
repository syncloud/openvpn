#!/bin/bash -e

source ${SNAP_DATA}/openvpn/keys/vars

CA_NAME=LocalCA
SERVER_NAME=server
export KEY_NAME=$CA_NAME
echo "Generating CA cert"
$EASY_RSA/pkitool --initca

export KEY_NAME=$SERVER_NAME
echo "Generating server cert"
$EASY_RSA/pkitool --server $SERVER_NAME
