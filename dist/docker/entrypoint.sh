#!/bin/bash


source /etc/secret-volume/secrets.txt
export WLA_SERVICE_USERNAME
export WLA_SERVICE_PASSWORD
export BEARER_TOKEN

COMPONENT_NAME=workload-agent
LOG_PATH=/var/log/$COMPONENT_NAME
CONFIG_PATH=/etc/$COMPONENT_NAME
CERTS_PATH=$CONFIG_PATH/certs
CERTDIR_TRUSTEDJWTCERTS=$CERTS_PATH/trustedjwt
CERTDIR_TRUSTEDCAS=$CERTS_PATH/trustedca
RUN_PATH=/var/run/$COMPONENT_NAME

if [ ! -f $CONFIG_PATH/.setup_done ]; then
  for directory in $LOG_PATH $CONFIG_PATH $CERTS_PATH $CERTDIR_TRUSTEDJWTCERTS $CERTDIR_TRUSTEDCAS; do
    mkdir -p $directory
    if [ $? -ne 0 ]; then
      echo "Cannot create directory: $directory"
      exit 1
    fi
    chmod 700 $directory
    chmod g+s $directory
  done

  wlagent setup all
  if [ $? -ne 0 ]; then
    exit 1
  fi
  touch $CONFIG_PATH/.setup_done
fi

if [ ! -z "$SETUP_TASK" ]; then
  IFS=',' read -ra ADDR <<< "$SETUP_TASK"
  for task in "${ADDR[@]}"; do
    wlagent setup $task --force
    if [ $? -ne 0 ]; then
      exit 1
    fi
  done
fi

unset AIK_SECRET
wlagent runservice
