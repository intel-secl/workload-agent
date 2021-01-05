#!/bin/bash

CURR_DIR=`pwd`
SECURE_DOCKER_DAEMON_DIR=$CURR_DIR/secure-docker-daemon
SECURE_DOCKER_PLUGIN_DIR=$CURR_DIR/secure-docker-plugin

git clone https://gitlab.devtools.intel.com/sst/isecl/secure-docker-daemon.git 2>/dev/null

cd $SECURE_DOCKER_DAEMON_DIR
git fetch
git checkout v3.4/develop
git pull

#Build secure docker daemon

make > /dev/null

if [ $? -ne 0 ]; then
  echo "could not build secure docker daemon"
  exit 1
fi

echo "Successfully built secure docker daemon"

cd $CURR_DIR
##Install secure-docker-daemon plugin
git clone https://github.com/intel-secl/secure-docker-plugin.git 2>/dev/null

cd $SECURE_DOCKER_PLUGIN_DIR
git fetch
git checkout v3.3.0
git pull

make

if [ $? -eq 0 ]; then
  echo "Successfully built secure docker plugin"
fi
