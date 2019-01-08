#!/bin/bash

# Postconditions:
# * exit with error code 1 only if there was a fatal error:
#####
# WORKLOAD_AGENT install script 
# Outline:

# WARNING:
# *** do NOT use TABS for indentation, use SPACES
# *** TABS will cause errors in some linux distributions# TERM_DISPLAY_MODE can be "plain" or "color"
TERM_DISPLAY_MODE=color
TERM_COLOR_GREEN="\\033[1;32m"
TERM_COLOR_CYAN="\\033[1;36m"
TERM_COLOR_RED="\\033[1;31m"
TERM_COLOR_YELLOW="\\033[1;33m"
TERM_COLOR_NORMAL="\\033[0;39m"

WORKLOAD_AGENT_LAYOUT=${WORKLOAD_AGENT_LAYOUT:-home}

# Environment:
# - TERM_DISPLAY_MODE
# - TERM_DISPLAY_GREEN
# - TERM_DISPLAY_NORMAL
echo_success() {
  if [ "$TERM_DISPLAY_MODE" = "color" ]; then echo -en "${TERM_COLOR_GREEN}"; fi
  echo ${@:-"[  OK  ]"}
  if [ "$TERM_DISPLAY_MODE" = "color" ]; then echo -en "${TERM_COLOR_NORMAL}"; fi
  return 0
}

# Environment:
# - TERM_DISPLAY_MODE
# - TERM_DISPLAY_RED
# - TERM_DISPLAY_NORMAL
echo_failure() {
  if [ "$TERM_DISPLAY_MODE" = "color" ]; then echo -en "${TERM_COLOR_RED}"; fi
  echo ${@:-"[FAILED]"}
  if [ "$TERM_DISPLAY_MODE" = "color" ]; then echo -en "${TERM_COLOR_NORMAL}"; fi
  return 1
}

# Environment:
# - TERM_DISPLAY_MODE
# - TERM_DISPLAY_YELLOW
# - TERM_DISPLAY_NORMAL
echo_warning() {
  if [ "$TERM_DISPLAY_MODE" = "color" ]; then echo -en "${TERM_COLOR_YELLOW}"; fi
  echo ${@:-"[WARNING]"}
  if [ "$TERM_DISPLAY_MODE" = "color" ]; then echo -en "${TERM_COLOR_NORMAL}"; fi
  return 1
}


echo_info() {
  if [ "$TERM_DISPLAY_MODE" = "color" ]; then echo -en "${TERM_COLOR_CYAN}"; fi
  echo ${@:-"[INFO]"}
  if [ "$TERM_DISPLAY_MODE" = "color" ]; then echo -en "${TERM_COLOR_NORMAL}"; fi
  return 1
}

############################################################################################################

# Product installation is only allowed if we are running as root
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

# LOCAL CONFIGURATION
directory_layout() {
if [ "$WORKLOAD_AGENT_LAYOUT" == "linux" ]; then
  export WORKLOAD_AGENT_CONFIGURATION=${WORKLOAD_AGENT_CONFIGURATION:-/etc/workloadagent}
  export WORKLOAD_AGENT_REPOSITORY=${WORKLOAD_AGENT_REPOSITORY:-/var/opt/workloadagent}
  export WORKLOAD_AGENT_LOGS=${WORKLOAD_AGENT_LOGS:-/var/log/workloadagent}
elif [ "$WORKLOAD_AGENT_LAYOUT" == "home" ]; then
  export WORKLOAD_AGENT_CONFIGURATION=${WORKLOAD_AGENT_CONFIGURATION:-$WORKLOAD_AGENT_HOME/configuration}
  export WORKLOAD_AGENT_REPOSITORY=${WORKLOAD_AGENT_REPOSITORY:-$WORKLOAD_AGENT_HOME/repository}
  export WORKLOAD_AGENT_LOGS=${WORKLOAD_AGENT_LOGS:-$WORKLOAD_AGENT_HOME/logs}
fi
export WORKLOAD_AGENT_VAR=${WORKLOAD_AGENT_VAR:-$WORKLOAD_AGENT_HOME/var}
export WORKLOAD_AGENT_BIN=${WORKLOAD_AGENT_BIN:-$WORKLOAD_AGENT_HOME/bin}
export WORKLOAD_AGENT_BACKUP=${WORKLOAD_AGENT_BACKUP:-$WORKLOAD_AGENT_REPOSITORY/backup}
export INSTALL_LOG_FILE=$WORKLOAD_AGENT_LOGS/install.log
}

# 3. define application directory layout
directory_layout

# 4. create application directories (chown will be repeated near end of this script, after setup)
for directory in $WORKLOAD_AGENT_HOME $WORKLOAD_AGENT_CONFIGURATION $WORKLOAD_AGENT_ENV $WORKLOAD_AGENT_REPOSITORY $WORKLOAD_AGENT_VAR $WORKLOAD_AGENT_LOGS; do
  # mkdir -p will return 0 if directory exists or is a symlink to an existing directory or directory and parents can be created
  mkdir -p $directory
  if [ $? -ne 0 ]; then
    echo_failure "Cannot create directory: $directory"
    exit 1
  fi
  #chown -R $WORKLOAD_AGENT_USERNAME:$WORKLOAD_AGENT_USERNAME $directory
  chmod 700 $directory
done

cp wlagent $WORKLOAD_AGENT_BIN
ln -s $WORKLOAD_AGENT_BIN/wlagent /usr/local/bin

# 5. wlagent setup
wlagent setup 

sudo yum -y install libvirt

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
cp qemu /etc/libvirt/hooks
#fill_with_variable_value "hook.sh" "<AUTOFILL_AT_INSTALL>" "/etc/libvirt/hooks/qemu"
