#!/bin/bash

CURR_DIR=`pwd`
SECURE_DOCKER_DAEMON_DIR=$CURR_DIR/secure_docker_daemon
SECURE_DOCKER_PLUGIN_DIR=$CURR_DIR/secure-docker-plugin

git clone ssh://git@gitlab.devtools.intel.com:29418/sst/isecl/secure_docker_daemon.git 2>/dev/null 

cd $SECURE_DOCKER_DAEMON_DIR
git fetch
git checkout v1.0/develop
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
git clone ssh://git@gitlab.devtools.intel.com:29418/sst/isecl/secure-docker-plugin.git 2>/dev/null 
cd $SECURE_DOCKER_PLUGIN_DIR
git fetch
git checkout v1.0/develop
git pull

make

if [ $? == 0 ]; then
  echo "Successfully built secure docker plugin"
fi
