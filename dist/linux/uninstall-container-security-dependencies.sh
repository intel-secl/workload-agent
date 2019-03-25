#!/bin/bash

#Copy all the vanilla docker daemon binaries from backup to /usr/bin/ and reconfigure the docker.service file to support vanilla docker
systemctl stop docker.service
systemctl stop secure-docker-plugin.service
cp -f /opt/workload-agent/secure-docker-daemon/backup/* /usr/bin/
sed -i 's/^ExecStart=.*/ExecStart=\/usr\/bin\/dockerd\ \-H\ unix\:\/\/ /' /lib/systemd/system/docker.service

systemctl stop secure-docker-plugin.service
rm /lib/systemd/system/secure-docker-plugin.socket
rm /lib/systemd/system/secure-docker-plugin.service
rm /usr/bin/secure-docker-plugin
# TODO copy as a backup if the user has already the daemon.json file
rm /etc/docker/daemon.json

systemctl daemon-reload
systemctl start docker.service
