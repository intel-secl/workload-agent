#!/bin/bash

# Postconditions:
# * exit with error code 1 only if there was a fatal error:
#####

# WARNING:
# *** do NOT use TABS for indentation, use SPACES
# *** TABS will cause errors in some linux distributions

# WORKLOAD_AGENT install script 
# Outline:
# Check if installer is running as a root
# Load the environment file
# Check if WORKLOAD_AGENT_NOSETUP is set in environment file
# Check if trustagent is intalled
# Load tagent username to a variable
# Load local configurations
# Create application directories
# Copy workload agent installer to workload-agent bin directory and create a symlink
# Call workload-agent setup
# Install and setup libvirt
# Copy isecl-hook script to libvirt hooks directory
# Restart the libvirt service after copying qemu hook

# TERM_DISPLAY_MODE can be "plain" or "color"
TERM_DISPLAY_MODE=color
TERM_COLOR_GREEN="\\033[1;32m"
TERM_COLOR_CYAN="\\033[1;36m"
TERM_COLOR_RED="\\033[1;31m"
TERM_COLOR_YELLOW="\\033[1;33m"
TERM_COLOR_NORMAL="\\033[0;39m"

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

# Make sure that we are running in the same directory as the install script
cd "$( dirname "$0" )"

# load installer environment file, if present
if [ -f ~/workload-agent.env ]; then
  echo "Loading environment variables from $(cd ~ && pwd)/workload-agent.env"
  . ~/workload-agent.env
  env_file_exports=$(cat ~/workload-agent.env | grep -E '^[A-Z0-9_]+\s*=' | cut -d = -f 1)
  if [ -n "$env_file_exports" ]; then eval export $env_file_exports; fi
else
  echo "workload-agent.env not found. Using existing exported variables or default ones"
fi

export LOG_LEVEL=${LOG_LEVEL:-"info"}


# Check if trustagent is intalled; if not output error
hash tagent 2>/dev/null || 
{
  echo_failure >&2 "Trust agent is not installed. Exiting."; 
  exit 1; 
}


# Check if yum packages are already installed; if not install them
pkg_install_logfile=wlagent_pkg_install.log
yum_packages=(libvirt cryptsetup)
for i in ${yum_packages[*]}
do
  isinstalled=$(rpm -q $i)
  if [ "$isinstalled" == "package $i is not installed" ]; then
    # put logs of install into a temporary file. We will copy this file and
    # delete it later.
    touch $pkg_install_logfile
    echo "Installing $i" >> $pkg_install_logfile
    yum -y install $i | tee -a $pkg_install_logfile
  fi
done
if [ ! -d "/etc/libvirt" ]; then
  echo_failure "libvirt directory not present. Exiting"
  exit 1
fi

mkdir -p "/etc/libvirt/hooks"
if [ ! -d "/etc/libvirt/hooks" ];  then
  echo_failure "Not able to create hooks directory. Exiting"
  exit 1
fi

# Use tagent user
#### Using trustagent user here as trustagent needs permissions to access files from workload agent
#### for eg signing binding keys. As tagent is a prerequisite for workload-agent, tagent user can be used here
if [ "$(whoami)" == "root" ]; then
  # create a trustagent user if there isn't already one created
  TRUSTAGENT_USERNAME=${TRUSTAGENT_USERNAME}
  if [[ -z $TRUSTAGENT_USERNAME ]]; then
    echo_failure "TRUSTAGENT_USERNAME must be exported and not empty"
    exit 1
  fi
  id -u $TRUSTAGENT_USERNAME
  if [[ $? -eq 1 ]]; then
    echo_failure "Cannot find user $TRUSTAGENT_USERNAME. Exiting"
    exit 1
  fi
fi

# Load local configurations
directory_layout() {
export WORKLOAD_AGENT_CONFIGURATION=/etc/workload-agent
export WORKLOAD_AGENT_LOGS=/var/log/workload-agent
export WORKLOAD_AGENT_HOME=/opt/workload-agent
export WORKLOAD_AGENT_BIN=$WORKLOAD_AGENT_HOME/bin
export INSTALL_LOG_FILE=$WORKLOAD_AGENT_LOGS/install.log
}
directory_layout


mkdir -p $(dirname $INSTALL_LOG_FILE)
if [ $? -ne 0 ]; then
  echo_failure "Cannot create directory: $(dirname $INSTALL_LOG_FILE)"
  exit 1
fi
logfile=$INSTALL_LOG_FILE
date >> $logfile
if [ -f $pkg_install_logfile ]; then
  cat $pkg_install_logfile >> $logfile
  rm -rf $pkg_install_logfile
fi

echo "Installing workload agent..." >> $logfile

# Create application directories (chown will be repeated near end of this script, after setup)
for directory in $WORKLOAD_AGENT_CONFIGURATION $WORKLOAD_AGENT_BIN $WORKLOAD_AGENT_LOGS; do
  # mkdir -p will return 0 if directory exists or is a symlink to an existing directory or directory and parents can be created
  mkdir -p $directory 
  if [ $? -ne 0 ]; then
    echo_failure "Cannot create directory: $directory" | tee -a $logfile
    exit 1
  fi
  chown -R $TRUSTAGENT_USERNAME:$TRUSTAGENT_USERNAME $directory
  chmod 700 $directory
done

# Copy workload agent installer to workload-agent bin directory and create a symlink
cp -f wlagent $WORKLOAD_AGENT_BIN
chown $TRUSTAGENT_USERNAME:$TRUSTAGENT_USERNAME $WORKLOAD_AGENT_BIN/wlagent
ln -sfT $WORKLOAD_AGENT_BIN/wlagent /usr/local/bin/wlagent

cp -f workload-agent.service $WORKLOAD_AGENT_HOME
systemctl enable $WORKLOAD_AGENT_HOME/workload-agent.service | tee -a $logfile

# Copy isecl-hook script to libvirt hooks directory. The name of hooks should be qemu
cp -f qemu /etc/libvirt/hooks 

# Restart the libvirt service after copying qemu hook and check if it's running
systemctl restart libvirtd
isactive=$(systemctl is-active libvirtd)
if [ ! "$isactive" == "active" ]; then
  echo_warning "libvirtd system service is not active. Exiting" | tee -a $logfile
  exit 0
fi
## TODO: Above - Should we exit is libvirt restart did not work? 
## Maybe we should have a seperate setup.sh that can just do the setup tasks. 


# exit workload-agent setup if WORKLOAD_AGENT_NOSETUP is set
if [ -n "$WORKLOAD_AGENT_NOSETUP" ]; then
  echo "WORKLOAD_AGENT_NOSETUP is set. So, skipping the workload-agent setup task." | tee -a $logfile
  exit 0
fi



# a global value to indicate if all the needed environment variables are present
# this is initially set to true. The check_env_var_present function would set this
# to false if and of the conditions are not met. This will be used to later decide 
# whether to proceed with the setup
all_env_vars_present=1

# check_env_var_present is used to check if an environment variable that we expect 
# is present. It prints a warning to the console if it does not exist
# Also, sets the the all_env_vars_present to false
# Arguments
#      $1 - var_name - the environment variable name that we are checking
#      $2 - empty_okay - (Optional) - empty_okay implies that environment variable needs
#           to be present - but it is acceptable for it to be empty
#           For most variables that we use, we won't pass it meaning that empty
#           strings are not acceptable
# Return 
#      0 - function succeeds
#      1 - function fauls
check_env_var_present(){
  # check if we were passed in an empty string
  if [[ -z $1 ]]; then return 1; fi

  if [[ $# > 1 ]] && [[ $2 == "true" || $2 == "1" ]]; then
    if [ "${!1:-}" ]; then
      return 0
    else
      echo_warning "$1 must be set and exported (empty value is okay)" | tee -a $logfile
      all_env_vars_present=0
      return 1
    fi
  fi

  if [ "${!1:-}" ]; then
    return 0
  else
    echo_warning "$1 must be set and exported" | tee -a $logfile
    all_env_vars_present=0
    return 1
  fi
}


# Validate the required environment variables for the setup. We are validating this in the 
# binary. However, for someone to figure out what are the ones that need to be set, they can 
# check here

# start with all_env_vars_present=1 and let the the check_env_vars_present() method override
# to false if any of the required vars are not set

all_env_vars_present=1

required_vars="MTWILSON_API_URL MTWILSON_API_USERNAME MTWILSON_API_PASSWORD\
  MTWILSON_TLS_CERT_SHA256 WLS_API_URL WLS_API_USERNAME WLS_API_PASSWORD\
  LOG_LEVEL TRUSTAGENT_CONFIGURATION TRUSTAGENT_USERNAME"
for env_var in $required_vars; do
  check_env_var_present $env_var
done

# Call workload-agent setup if all the required env variables are set
if [[ $all_env_vars_present -eq 1 ]]; then
  wlagent setup | tee -a $logfile
else 
  echo_failure "One or more environment variables are not present. Setup cannot proceed. Aborting..." | tee -a $logfile
  echo_failure "Please export the missing environment variables and run setup again" | tee -a $logfile
  exit 1
fi

# Enable systemd service and start it
systemctl start workload-agent | tee -a $logfile

is_docker_installed(){
  which docker 2>/dev/null
  if [ $? -ne 0 ]; then
    echo "Docker is not installed"
    exit 1
  fi
}

install_secure_docker_plugin(){

mkdir /etc/systemd/system/secure-docker-plugin.service.d 2>1 /dev/null

cat > /etc/systemd/system/secure-docker-plugin.service.d/securedockerplugin.conf <<EOF
[Service]
Environment="INSECURE_SKIP_VERIFY=${INSECURE_SKIP_VERIFY:-true}"
Environment="NO_PROXY=${NO_PROXY}"
Environment="REGISTRY_SCHEME_TYPE=${REGISTRY_SCHEME_TYPE:-https}"
Environment="REGISTRY_USERNAME=${REGISTRY_USERNAME}"
Environment="REGISTRY_PASSWORD=${REGISTRY_PASSWORD}"
Environment="HTTPS_PROXY=${HTTPS_PROXY}"
EOF

cp secure-docker-plugin /usr/bin/
cp artifact/* /lib/systemd/system/

systemctl daemon-reload
systemctl start secure-docker-plugin.service
}


#Install secure docker daemon with WA only if WA_WITH_CONTAINER_SECURITY is enabled in workload-agent.env
if [ "$WA_WITH_CONTAINER_SECURITY" == "y" ] || [ "$WA_WITH_CONTAINER_SECURITY" == "Y" ] || [ "$WA_WITH_CONTAINER_SECURITY" == "yes" ]; then
  is_docker_installed

  which cryptsetup 2>/dev/null
  if [ $? -ne 0 ]; then
    yum install -y cryptsetup
  fi  
  echo "Installing secure docker daemon"
  systemctl stop docker

  mkdir -p $WORKLOAD_AGENT_HOME/secure-docker-daemon/backup
  cp /usr/bin/docker* $WORKLOAD_AGENT_HOME/secure-docker-daemon/backup/
  chown -R root:root docker-daemon/
  cp -f docker-daemon/* /usr/bin/

  install_secure_docker_plugin

  echo "Starting secure docker engine"
  mkdir -p /etc/docker
  cp daemon.json /etc/docker/ 
  systemctl daemon-reload
  systemctl start docker
  cp uninstall-container-security-dependencies.sh $WORKLOAD_AGENT_HOME/secure-docker-daemon/
fi

echo_success "Installation completed." | tee -a $logfile
