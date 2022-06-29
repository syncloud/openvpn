#!/bin/bash -xe

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/build/snap
cd ${DIR}/web
mkdir ${BUILD_DOR}/web
go build -o ${BUILD_DIR}/web/openvpn-web-ui 
