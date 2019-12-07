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
GO_VERSION=1.11.5
GO_ARCH=armv6l
if [[ ${ARCH} == "x86_64" ]]; then
    GO_ARCH=amd64
fi
GOROOT=${DIR}/go
GOPATH=${DIR}/gopath
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

${BUILD_DIR}/python/bin/pip install -r ${DIR}/requirements.txt

wget https://dl.google.com/go/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz --progress dot:giga
tar xf go${GO_VERSION}.linux-${GO_ARCH}.tar.gz

go version

cp -r ${DIR}/bin ${BUILD_DIR}
cp -r ${DIR}/config ${BUILD_DIR}/config.templates
cp -r ${DIR}/hooks ${BUILD_DIR}

cd ${DIR}/build

go get github.com/beego/bee

wget --progress=dot:giga https://github.com/cyberb/openvpn-web-ui/archive/${OPENVPN_WEBUI_VERSION}.tar.gz
tar xzf ${OPENVPN_WEBUI_VERSION}.tar.gz
cd openvpn-web-ui-${OPENVPN_WEBUI_VERSION}
#go build -o openvpn-web-ui 
${GOPATH}/bin/bee pack -exr='^vendor|^data.db|^build|^README.md|^docs'

mkdir ${DIR}/build/${NAME}/META
echo ${NAME} >> ${DIR}/build/${NAME}/META/app
echo ${VERSION} >> ${DIR}/build/${NAME}/META/version

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
