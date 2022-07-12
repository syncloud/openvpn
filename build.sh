#!/bin/bash -xe

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

PREFIX=${DIR}/build/snap/openvpn
NAME=openvpn

apt update
apt -y install liblzo2-dev libpam-dev net-tools

cd ${DIR}/build/openvpn-*
./configure --prefix=${PREFIX}
make
make install

export LD_LIBRARY_PATH=${PREFIX}/lib
cp /lib/*/liblzo2.so* ${PREFIX}/lib
cp /usr/lib/*/libcrypt*.so* ${PREFIX}/lib
cp /usr/lib/*/libssl*.so* ${PREFIX}/lib
cp /lib/*/libnsl.so* ${PREFIX}/lib
cp /lib/*/libresolv.so* ${PREFIX}/lib
cp /lib/*/libdl.so* ${PREFIX}/lib
cp /lib/*/libc.so* ${PREFIX}/lib
cp /lib/*/libpthread.so* ${PREFIX}/lib
cp $(readlink -f /lib*/ld-linux-*.so*) ${PREFIX}/lib/ld.so
cp $DIR/bin/openvpn.sh ${PREFIX}/sbin

ldd ${PREFIX}/sbin/openvpn

