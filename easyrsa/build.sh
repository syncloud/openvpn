#!/bin/bash -xe

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap/easyrsa
VERSION=$1

apt update
apt -y install \
  wget \
  unzip

mkdir -p ${DIR}/../build
wget --progress=dot:giga https://github.com/OpenVPN/easy-rsa/releases/download/v${EASY_RSA_VERSION}/EasyRSA-${EASY_RSA_VERSION}.tgz
tar xf EasyRSA-${EASY_RSA_VERSION}.tgz
mv EasyRSA-${EASY_RSA_VERSION} ${BUILD_DIR}/easyrsa
cp -r ${DIR}/../config/easyrsa/vars ${BUILD_DIR}/easyrsa
