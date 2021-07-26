#!/bin/bash

COMPONENT_NAME=workload-agent
BINARY_NAME=wlagent
WORKLOAD_AGENT_HOME=/opt/workload-agent

# Do nothing for container deployment, container deployment comes with grpc service as default in v4.1.0 container image
if [ -f "/.container-env" ]; then
  exit 0
fi

if [ -f $WORKLOAD_AGENT_HOME/qemu ] || [ -d $WORKLOAD_AGENT_HOME/secure-docker-daemon ]; then
  echo "v4.1.0 config upgrade is not applicable for docker or vm confidentiality use case, skipping config upgrade for VM confidentiality and Container confidentiality with docker use case"
  exit 0
fi

echo "Starting $COMPONENT_NAME config upgrade for Container confidentiality with cri-o use case to v4.1.0"
sed -i "s/runservice/rungrpcservice/g" $WORKLOAD_AGENT_HOME/wlagent.service
if [ $? -ne 0 ]; then
  echo "failed to update $WORKLOAD_AGENT_HOME/wlagent.service service file"
  exit 1
fi
systemctl daemon-reload

echo "Completed $COMPONENT_NAME config upgrade to v4.1.0"
