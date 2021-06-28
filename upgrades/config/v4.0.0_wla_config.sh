#!/bin/bash

COMPONENT_NAME=workload-agent
BINARY_NAME=wlagent

if [ -f "/.container-env" ]; then
  source /etc/secret-volume/secrets.txt
  export BEARER_TOKEN
  ln -sfT /usr/bin/$BINARY_NAME /$BINARY_NAME
fi

echo "Starting $COMPONENT_NAME config upgrade to v4.0.0"
./$BINARY_NAME setup all --force
if [ $? != 0 ]; then
  exit 1
fi
echo "Completed $COMPONENT_NAME config upgrade to v4.0.0"
