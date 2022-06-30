#!/bin/bash -xe

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap
mkdir ${BUILD_DIR}/web
go build -o ${BUILD_DIR}/web/openvpn-web-ui 
