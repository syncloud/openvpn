#!/bin/bash -xe

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap/openvpn
VERSION=$1

apt update
apt -y install \
  liblzo2-dev \
  libpam-dev \
  net-tools \
  libnl-genl-3-dev \
  libcap-ng-dev \
  liblz4-dev \
  wget \
  unzip

cd ${DIR}/../build
wget https://swupdate.openvpn.org/community/releases/openvpn-${VERSION}.tar.gz  --progress dot:giga -O openvpn-${VERSION}.tar.gz
tar xzf openvpn-${VERSION}.tar.gz
rm openvpn-${VERSION}.tar.gz
cd openvpn-*
./configure --prefix=${BUILD_DIR}
make
make install

export LD_LIBRARY_PATH=${BUILD_DIR}/lib
mkdir -p ${BUILD_DIR}/lib
cp /lib/*/liblzo2.so* ${BUILD_DIR}/lib
cp /lib/*/liblz4.so* ${BUILD_DIR}/lib
cp /lib/*/libnl-genl-3.so* ${BUILD_DIR}/lib
cp /lib/*/libnl-3.so* ${BUILD_DIR}/lib
cp /lib/*/libcap-ng.so* ${BUILD_DIR}/lib
cp /lib/*/libtirpc.so* ${BUILD_DIR}/lib
cp /lib/*/libgssapi_krb5.so* ${BUILD_DIR}/lib
cp /lib/*/libkrb5.so* ${BUILD_DIR}/lib
cp /lib/*/libk5crypto.so* ${BUILD_DIR}/lib
cp /lib/*/libcom_err.so* ${BUILD_DIR}/lib
cp /lib/*/libkrb5support.so* ${BUILD_DIR}/lib
cp /lib/*/libkeyutils.so* ${BUILD_DIR}/lib
cp /usr/lib/*/libcrypt*.so* ${BUILD_DIR}/lib
cp /usr/lib/*/libssl*.so* ${BUILD_DIR}/lib
cp /lib/*/libnsl.so* ${BUILD_DIR}/lib
cp /lib/*/libresolv.so* ${BUILD_DIR}/lib
cp /lib/*/libdl.so* ${BUILD_DIR}/lib
cp /lib/*/libc.so* ${BUILD_DIR}/lib
cp /lib/*/libpthread.so* ${BUILD_DIR}/lib

cp $(readlink -f /lib*/ld-linux-*.so*) ${BUILD_DIR}/lib/ld.so
cp $DIR/bin/openvpn.sh ${BUILD_DIR}/sbin

ldd ${BUILD_DIR}/sbin/openvpn

