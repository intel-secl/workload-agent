#!/bin/bash

SERVICE_NAME=wlagent
CURRENT_VERSION=v3.6.1
BACKUP_PATH=${BACKUP_PATH:-"/tmp/"}
INSTALLED_EXEC_PATH="/opt/workload-agent/bin/$SERVICE_NAME"
CONFIG_PATH="/etc/workload-agent/"
NEW_EXEC_NAME="$SERVICE_NAME"
LOG_FILE=${LOG_FILE:-"/tmp/$SERVICE_NAME-upgrade.log"}
echo "" > $LOG_FILE

if [ -d "/opt/workload-agent/secure-docker-daemon" ]; then
  ./upgrade-secure-docker-daemon.sh |& tee -a $LOG_FILE
fi

./upgrade.sh -s $SERVICE_NAME -v $CURRENT_VERSION -e $INSTALLED_EXEC_PATH -c $CONFIG_PATH -n $NEW_EXEC_NAME -b $BACKUP_PATH |& tee -a $LOG_FILE
