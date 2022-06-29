#!/bin/bash -xe

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

PREFIX=${DIR}/build/snap/openvpn
NAME=openvpn

apt update
apt -y install liblzo2-dev libpam-dev

cd ${DIR}/build/openvpn-*
./configure --prefix=${PREFIX}
make
make install

export LD_LIBRARY_PATH=${PREFIX}/lib
cp --remove-destination /lib/$(dpkg-architecture -q DEB_HOST_GNU_TYPE)/liblzo2.so* ${PREFIX}/lib
cp --remove-destination /usr/lib/$(dpkg-architecture -q DEB_HOST_GNU_TYPE)/libcrypt*.so* ${PREFIX}/lib
cp --remove-destination /usr/lib/$(dpkg-architecture -q DEB_HOST_GNU_TYPE)/libssl*.so* ${PREFIX}/lib

ldd ${PREFIX}/sbin/openvpn
