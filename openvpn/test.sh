#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap/openvpn
ls -la ${BUILD_DIR}/lib
${BUILD_DIR}/sbin/openvpn.sh --help || true
${BUILD_DIR}/sbin/openvpn.sh --version