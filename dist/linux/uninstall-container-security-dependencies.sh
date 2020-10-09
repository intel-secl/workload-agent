#!/bin/bash
echo "Removing container security components"

# remove all running containers
docker rmi -f $(docker images -q)

# unmount and remove the secureoverlay2 layer data
for m in $(mount -t overlay | grep /var/lib/docker/secureoverlay2/ | awk '{print $3}')
do
  umount -d -f -R $m
done
rm -rf /var/lib/docker/secureoverlay2

# purge the stale data
echo y | docker system prune -a

# stop and purge
systemctl stop docker.service
systemctl stop secure-docker-plugin.service
systemctl disable secure-docker-plugin.service
rm /lib/systemd/system/secure-docker-plugin.socket
rm /lib/systemd/system/secure-docker-plugin.service
rm -rf /etc/systemd/system/secure-docker-plugin.service.d/
rm /usr/bin/secure-docker-plugin

#Copy all the vanilla docker daemon binaries and config files from backup to /usr/bin/ and reconfigure the docker.service file to support vanilla docker
cp -f /opt/workload-agent/secure-docker-daemon/backup/docker /usr/bin/
cp -f /opt/workload-agent/secure-docker-daemon/backup/dockerd* /usr/bin/

# restore original daemon.json else remove current version
if [ -f /opt/workload-agent/secure-docker-daemon/backup/daemon.json ]; then
  cp -f /opt/workload-agent/secure-docker-daemon/backup/daemon.json /etc/docker/daemon.json
else
  rm -f /etc/docker/daemon.json
fi

# restore original docker unit file
cp -f /opt/workload-agent/secure-docker-daemon/backup/docker.service /lib/systemd/system/docker.service

# reload changes before restart
systemctl daemon-reload
