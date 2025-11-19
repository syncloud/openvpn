#!/bin/bash -xe

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}
VERSION=2.6.16
EASY_RSA_VERSION=3.2.4

ARCH=$(uname -m)
DOWNLOAD_URL=https://github.com/syncloud/3rdparty/releases/download
BUILD_DIR=${DIR}/build/snap
mkdir -p $BUILD_DIR

apt update
apt -y install wget unzip

cd ${DIR}/build

wget https://swupdate.openvpn.org/community/releases/openvpn-${VERSION}.tar.gz  --progress dot:giga -O openvpn-${VERSION}.tar.gz
tar xzf openvpn-${VERSION}.tar.gz
rm openvpn-${VERSION}.tar.gz

wget --progress=dot:giga https://github.com/OpenVPN/easy-rsa/releases/download/v${EASY_RSA_VERSION}/EasyRSA-${EASY_RSA_VERSION}.tgz
tar xf EasyRSA-${EASY_RSA_VERSION}.tgz
mv EasyRSA-${EASY_RSA_VERSION} ${BUILD_DIR}/easyrsa
cp -r ${DIR}/config/easyrsa/vars ${BUILD_DIR}/easyrsa
