#!/bin/bash -ex

/snap/openvpn/current/easyrsa/easyrsa --batch build-ca nopass | logger
/snap/openvpn/current/easyrsa/easyrsa --batch build-server-full server nopass | logger