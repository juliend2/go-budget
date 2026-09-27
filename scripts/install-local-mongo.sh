#!/usr/bin/env bash
# Installs a local MongoDB 4.4 into ~/mongodb-budget.
#
# MongoDB 5.0+ requires AVX; this installs 4.4 (no AVX needed) for machines
# whose CPU lacks it. Also installs libssl1.1 (needed by the 4.4 build on
# Debian 12/13) and the database tools (mongodump, mongorestore, ...).
# Idempotent: re-running skips what is already installed.

set -euo pipefail

DEST="${HOME}/mongodb-budget"
MONGO_VERSION="4.4.29"
TOOLS_VERSION="100.12.2"
LIBSSL_URL="https://deb.debian.org/debian/pool/main/o/openssl/libssl1.1_1.1.1w-0+deb11u1_amd64.deb"

mkdir -p "$DEST/bin" "$DEST/lib" "$DEST/data"

if [ ! -x "$DEST/bin/mongod" ]; then
  echo "Downloading mongod ${MONGO_VERSION}..."
  curl -fsSL -o /tmp/mongod.tgz \
    "https://fastdl.mongodb.org/linux/mongodb-linux-x86_64-debian10-${MONGO_VERSION}.tgz"
  tar -C /tmp -xzf /tmp/mongod.tgz \
    "mongodb-linux-x86_64-debian10-${MONGO_VERSION}/bin/mongod"
  mv "/tmp/mongodb-linux-x86_64-debian10-${MONGO_VERSION}/bin/mongod" "$DEST/bin/"
  rm -rf /tmp/mongod.tgz "/tmp/mongodb-linux-x86_64-debian10-${MONGO_VERSION}"
fi

if [ ! -f "$DEST/lib/libssl.so.1.1" ]; then
  echo "Downloading libssl1.1..."
  curl -fsSL -o /tmp/libssl11.deb "$LIBSSL_URL"
  mkdir -p /tmp/ssl11
  dpkg -x /tmp/libssl11.deb /tmp/ssl11
  cp /tmp/ssl11/usr/lib/x86_64-linux-gnu/libssl.so.1.1 "$DEST/lib/"
  cp /tmp/ssl11/usr/lib/x86_64-linux-gnu/libcrypto.so.1.1 "$DEST/lib/"
  rm -rf /tmp/libssl11.deb /tmp/ssl11
fi

if [ ! -x "$DEST/bin/mongodump" ]; then
  echo "Downloading database tools ${TOOLS_VERSION}..."
  curl -fsSL -o /tmp/mongotools.tgz \
    "https://fastdl.mongodb.org/tools/db/mongodb-database-tools-debian12-x86_64-${TOOLS_VERSION}.tgz"
  tar -C /tmp -xzf /tmp/mongotools.tgz
  cp /tmp/mongodb-database-tools-debian12-x86_64-${TOOLS_VERSION}/bin/* "$DEST/bin/"
  rm -rf /tmp/mongotools.tgz "/tmp/mongodb-database-tools-debian12-x86_64-${TOOLS_VERSION}"
fi

echo "Installed in $DEST:"
ls "$DEST/bin" "$DEST/lib"
echo
echo "Start it with:    make mongo-local"
echo "Restore a backup: mongorestore --uri=\"mongodb://localhost:27027\" --nsExclude=\"admin.*\" --gzip --drop <backup-dir>"