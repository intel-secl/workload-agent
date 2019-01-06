#!/bin/bash

# Postconditions:
# * exit with error code 1 only if there was a fatal error:
#####

# WARNING:
# *** do NOT use TABS for indentation, use SPACES
# *** TABS will cause errors in some linux distributions

# application defaults (these are not configurable and used only in this script so no need to export)
DEFAULT_WORKLOAD_AGENT_HOME=/opt/workload-agent
DEFAULT_WORKLOAD_AGENT_USERNAME=wlagent

# check if we are running in a docker container or running as root. Product installation is only
# allowed if we are running as root
if [ $EUID -ne 0 ];  then
  echo "Workload agent installation has to run as root. Exiting"
  exit 1
fi

# Deployment phase
# 2. load installer environment file, if present
if [ -f ~/workloadagent.env ]; then
  echo "Loading environment variables from $(cd ~ && pwd)/workloadagent.env"
  . ~/workloadagent.env
  env_file_exports=$(cat ~/workloadagent.env | grep -E '^[A-Z0-9_]+\s*=' | cut -d = -f 1)
  if [ -n "$env_file_exports" ]; then eval export $env_file_exports; fi
else
  echo "No environment file"
fi

# exit workloadagent setup if WORKLOAD_AGENT_NOSETUP is set
if [ -n "$WORKLOAD_AGENT_NOSETUP" ]; then
  echo "WORKLOAD_AGENT_NOSETUP value is set. So, skipping the workloadagent setup task."
  exit 0;
fi

# 33. wlagent setup
wlagent setup 

echo_warning "TODO : Need to install hooks to libvrt - writing configuration directory "
if [ ! -d "/etc/libvirt" ]; then
  echo_warning "libvirt directory not present. Exiting"
  exit 0
fi
mkdir "/etc/libvirt/hooks"

if [ ! -d "/etc/libvirt/hooks" ];  then
  echo_warning "Not able to create hooks directory. Exiting"
  echo 0
fi

# Call function to insert relevant variable values to the hooks script.
# Since libvirt calls the binary, certain environment variables need to be
# populated for the wlagent binary to run. Check the hooks.sh script to
# determine the parameters that need to be passed in. Basically, we need
# the input file, the placeholder (such as <AUTOFILL_AT_INSTALL>) and the
# destination file of the hooks script file

# destination file needs to be called qemu

fill_with_variable_value "hook.sh" "<AUTOFILL_AT_INSTALL>" "/etc/libvirt/hooks/quemu"
