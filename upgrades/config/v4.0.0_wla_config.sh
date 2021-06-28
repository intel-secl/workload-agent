#!/bin/bash

COMPONENT_NAME=workload-agent
BINARY_NAME=wlagent

echo "Starting $COMPONENT_NAME config upgrade to v4.0.0"
./$BINARY_NAME setup all --force
if [ $? != 0 ]; then
  exit 1
fi
echo "Completed $COMPONENT_NAME config upgrade to v4.0.0"
