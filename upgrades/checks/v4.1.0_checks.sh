#!/bin/bash

if [ -f $WORKLOAD_AGENT_HOME/qemu ] || [ -d $WORKLOAD_AGENT_HOME/secure-docker-daemon ]; then
  echo "v4.1.0 config upgrade is not applicable for docker or vm confidentiality use case, skipping pre-checks"
  exit 0
fi

REQUIRED_CRIO_VERSION=1210 #crio-1.21.0
which crio 2>/dev/null
if [ $? != 0 ]; then
  echo "Prerequisite cri-o v1.21.0 is not installed, please install cri-o v1.21.0 container runtime before proceeding with upgrade, Exiting..."
  exit 1
else
  CRIO_VERSION=$(crio -v | grep -w Version | cut -d':' -f2 | xargs | sed 's/\.//g')
  if [ $CRIO_VERSION -lt $REQUIRED_CRIO_VERSION ]; then
    echo "Prerequisite cri-o installed version: $(crio -v | grep -w Version | cut -d':' -f2 | xargs), please install cri-o >=v1.21.0 container runtime before proceeding with upgrade, Exiting..."
    exit 1
  fi
fi
