#!/bin/bash -e

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )

LIBS=${DIR}/lib
exec ${DIR}/lib/ld.so --library-path $LIBS ${DIR}/sbin/openvpn "$@"
