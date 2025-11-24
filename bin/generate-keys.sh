#!/bin/bash -ex
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )

EASYRSA_BIN=$DIR/easyrsa/easyrsa
VARS_FILE=$DIR/config/easyrsa/vars

"$EASYRSA_BIN" --vars="$VARS_FILE" --batch build-ca nopass | logger
"$EASYRSA_BIN" --vars="$VARS_FILE" --batch build-server-full server nopass | logger
