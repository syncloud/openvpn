#!/bin/bash -xe

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

if [[ -z "$2" ]]; then
    echo "usage $0 app version"
    exit 1
fi

NAME=$1

ARCH=$(uname -m)
VERSION=$2
OPENVPN_VERSION=2.4.8
OPENVPN_WEBUI_VERSION=master
EASY_RSA_VERSION=2.2.2
GO_VERSION=1.11.5
GO_ARCH=armv6l
if [[ ${ARCH} == "x86_64" ]]; then
    GO_ARCH=amd64
fi
GOROOT=${DIR}/go
export GOPATH=${DIR}/gopath
export PATH=$GOROOT/bin:${GOPATH}/bin:$PATH

rm -rf ${DIR}/build
BUILD_DIR=${DIR}/build/${NAME}
mkdir -p ${BUILD_DIR}

wget --progress=dot:giga https://github.com/syncloud/3rdparty/releases/download/1/nginx-${ARCH}.tar.gz
tar xf nginx-${ARCH}.tar.gz
mv nginx ${BUILD_DIR}

wget --progress=dot:giga https://github.com/syncloud/3rdparty/releases/download/1/openvpn-${ARCH}-${OPENVPN_VERSION}.tar.gz
tar xf openvpn-${ARCH}-${OPENVPN_VERSION}.tar.gz
mv openvpn ${BUILD_DIR}

wget --progress=dot:giga https://github.com/syncloud/3rdparty/releases/download/1/python-${ARCH}.tar.gz
tar xf python-${ARCH}.tar.gz
mv python ${BUILD_DIR}

wget --progress=dot:giga https://github.com/OpenVPN/easy-rsa/releases/download/${EASY_RSA_VERSION}/EasyRSA-${EASY_RSA_VERSION}.tgz
tar xf EasyRSA-${EASY_RSA_VERSION}.tgz
mv EasyRSA-${EASY_RSA_VERSION} ${BUILD_DIR}/easy-rsa

${BUILD_DIR}/python/bin/pip install -r ${DIR}/requirements.txt

wget https://dl.google.com/go/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz --progress dot:giga
tar xf go${GO_VERSION}.linux-${GO_ARCH}.tar.gz

go version

cp -r ${DIR}/bin ${BUILD_DIR}
cp -r ${DIR}/config ${BUILD_DIR}
cp -r ${DIR}/hooks ${BUILD_DIR}
mkdir ${BUILD_DIR}/config/openvpn
cd ${DIR}/build

wget --progress=dot:giga https://github.com/cyberb/openvpn-web-ui/archive/${OPENVPN_WEBUI_VERSION}.tar.gz
tar xzf ${OPENVPN_WEBUI_VERSION}.tar.gz
cd openvpn-web-ui-${OPENVPN_WEBUI_VERSION}
mkdir ${DIR}/build/${NAME}/web
go build -o ${BUILD_DIR}/web/openvpn-web-ui 
cp -r static ${BUILD_DIR}/web
cp -r views ${BUILD_DIR}/web

mkdir ${BUILD_DIR}/META
echo ${NAME} >> ${BUILD_DIR}/META/app
echo ${VERSION} >> ${BUILD_DIR}/META/version

echo "snapping"
SNAP_DIR=${DIR}/build/snap
ARCH=$(dpkg-architecture -q DEB_HOST_ARCH)
rm -rf ${DIR}/*.snap
mkdir ${SNAP_DIR}
cp -r ${BUILD_DIR}/* ${SNAP_DIR}/
cp -r ${DIR}/snap/meta ${SNAP_DIR}/
cp ${DIR}/snap/snap.yaml ${SNAP_DIR}/meta/snap.yaml
echo "version: $VERSION" >> ${SNAP_DIR}/meta/snap.yaml
echo "architectures:" >> ${SNAP_DIR}/meta/snap.yaml
echo "- ${ARCH}" >> ${SNAP_DIR}/meta/snap.yaml

PACKAGE=${NAME}_${VERSION}_${ARCH}.snap
echo ${PACKAGE} > ${DIR}/package.name
mksquashfs ${SNAP_DIR} ${DIR}/${PACKAGE} -noappend -comp xz -no-xattrs -all-root
