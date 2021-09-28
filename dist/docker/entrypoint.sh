#!/bin/bash

SECRETS=/etc/secrets
IFS=$'\r\n' GLOBIGNORE='*' command eval 'secretFiles=($(ls  $SECRETS))'
for i in "${secretFiles[@]}"; do
    export $i=$(cat $SECRETS/$i)
done

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
  cp $CONFIG_PATH/config.yml /tmp/config.yml
  IFS=',' read -ra ADDR <<< "$SETUP_TASK"
  for task in "${ADDR[@]}"; do
    wlagent setup $task --force
    if [ $? -ne 0 ]; then
      cp /tmp/config.yml $CONFIG_PATH/config.yml
      exit 1
    fi
  done
  rm -rf /tmp/config.yml
fi

unset AIK_SECRET
for i in "${secretFiles[@]}"; do
    unset $i
done

wlagent rungrpcservice
