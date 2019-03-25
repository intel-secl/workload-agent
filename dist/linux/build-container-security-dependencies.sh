#!/bin/bash

# To fetch the Gurpartap/logrus-stack and facebookgo/stack and also to copy those dependencies to vendor directory, GOPATH need to be set
if [ -z $GOPATH ]; then echo "Please set the GOPATH"; exit 1; fi

CURR_DIR=`pwd`
SECURE_DOCKER_DAEMON_DIR=$CURR_DIR/secure_docker_daemon
DAEMON_DIR=$SECURE_DOCKER_DAEMON_DIR/dcg_security-container-encryption/daemon-output
SECURE_DOCKER_PLUGIN_DIR=$CURR_DIR/secure-docker-plugin
export no_proxy=$no_proxy,gitlab.devtools.intel.com
git clone ssh://git@gitlab.devtools.intel.com:29418/sst/isecl/secure_docker_daemon.git 2>/dev/null 

cd $SECURE_DOCKER_DAEMON_DIR
git fetch
git checkout v1.0/develop
git pull

#Build secure docker daemon
#Dependencies Gurpartap and facbookgo repos need to be manually copied to vendor directory.
cd dcg_security-container-encryption
go get -u github.com/Gurpartap/logrus-stack
go get -u github.com/facebookgo/stack
mkdir -p vendor/github.com/Gurpartap/logrus-stack  2>/dev/null
mkdir -p  vendor/github.com/facebookgo/stack 2>/dev/null
logrus=`find $GOPATH/pkg/mod/github.com/\!gurpartap -type d | grep "stack" | head -n 1`
stack=`find $GOPATH/pkg/mod/github.com/facebookgo -type d | grep "stack" | head -n 1`

if [ -d $logrus ]; then
  cp -r $logrus/* vendor/github.com/Gurpartap/logrus-stack/
  sed -i 's/sirupsen/Sirupsen/' vendor/github.com/Gurpartap/logrus-stack/logrus-stack-hook.go
fi

if [ -d $stack ]; then
  cp -r $stack/*  vendor/github.com/facebookgo/stack/
fi


make > /dev/null

if [ $? -ne 0 ]; then
  echo "could not build secure docker daemon"
  exit 1
fi
  
#Copy docker daemon binaries to single output directory daemon-output
mkdir $DAEMON_DIR 2>/dev/null

echo "Copying secure docker daemon binaries to daemon-output directory"
cp bundles/17.06.0-dev/binary-client/docker-17.06.0-dev $DAEMON_DIR/docker
cd bundles/17.06.0-dev/binary-daemon
cp docker-containerd docker-runc docker-containerd-ctr docker-containerd-shim docker-init docker-proxy dockerd-17.06.0-dev $DAEMON_DIR
mv $DAEMON_DIR/dockerd-17.06.0-dev $DAEMON_DIR/dockerd

cd $CURR_DIR
##Install secure-docker-daemon plugin
git clone ssh://git@gitlab.devtools.intel.com:29418/sst/isecl/secure-docker-plugin.git 2>/dev/null
cd $SECURE_DOCKER_PLUGIN_DIR
git fetch
git checkout v1.0/develop
git pull
rm go.sum

make

if [ $? == 0 ]; then
  echo "Successfully build secure docker daemon and secure docker plugin"
fi