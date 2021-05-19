#!/bin/bash

SECURE_DOCKER_DAEMON_BACKUP_PATH="/tmp/wlagent_backup/container-runtime"

echo "Upgrading secure-docker-plugin and secure-docker-daemon"

systemctl stop secure-docker-plugin
systemctl stop docker

echo "Creating backup $SECURE_DOCKER_DAEMON_BACKUP_PATH directory for secure-docker-daemon and secure-docker-plugin binaries"
mkdir -p $SECURE_DOCKER_DAEMON_BACKUP_PATH
echo "Taking backup of secure-docker-daemon and secure-docker-plugin"
cp /usr/bin/secure-docker-plugin $SECURE_DOCKER_DAEMON_BACKUP_PATH
cp /usr/bin/docker $SECURE_DOCKER_DAEMON_BACKUP_PATH
which /usr/bin/dockerd-ce 2>/dev/null
if [ $? -ne 0 ]; then
  cp /usr/bin/dockerd $SECURE_DOCKER_DAEMON_BACKUP_PATH
else
  cp /usr/bin/dockerd-ce $SECURE_DOCKER_DAEMON_BACKUP_PATH
fi

cp -f secure-docker-plugin /usr/bin/
cp -f docker-daemon/docker /usr/bin/
which /usr/bin/dockerd-ce 2>/dev/null
if [ $? -ne 0 ]; then
  cp -f docker-daemon/dockerd-ce /usr/bin/dockerd
else
  cp -f docker-daemon/dockerd-ce /usr/bin/dockerd-ce
fi

echo "Starting secure-docker-plugin"
systemctl start secure-docker-plugin
if [ $? -ne 0 ]; then
  echo "Error while starting secure-docker-plugin"
  exit 1
else
  echo "Upgraded secure-docker-plugin successfully"
fi

echo "Starting secure-docker-daemon"
systemctl start docker
if [ $? -ne 0 ]; then
  echo "Error while starting secure-docker-daemon"
  exit 1
else
  echo "Upgraded secure-docker-daemon successfully"
fi