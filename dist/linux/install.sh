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

# Do these when installing vm usecase
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
# TODO: ISECL-8364 Resolve flow/steps for using 'env' files when installing workload-agent
if [ -f ~/trustagent.env ]; then
  echo "Loading environment variables from $(cd ~ && pwd)/trustagent.env"
  . ~/trustagent.env
  env_file_exports=$(cat ~/trustagent.env | grep -E '^[A-Z0-9_]+\s*=' | cut -d = -f 1)
  if [ -n "$env_file_exports" ]; then eval export $env_file_exports; fi
else
  echo "trustagent.env not found. Using existing exported variables or default ones"
fi

export LOG_LEVEL=${LOG_LEVEL:-"info"}

auto_install() {
  local component=${1}
  local cprefix=${2}
  local yum_packages=$(eval "echo \$${cprefix}_YUM_PACKAGES")
  # detect available package management tools. start with the less likely ones to differentiate.
  yum -y install $yum_packages
}


# SCRIPT EXECUTION

logRotate_clear() {
  logrotate=""
}

logRotate_detect() {
  local logrotaterc=`ls -1 /etc/logrotate.conf 2>/dev/null | tail -n 1`
  logrotate=`which logrotate 2>/dev/null`
  if [ -z "$logrotate" ] && [ -f "/usr/sbin/logrotate" ]; then
    logrotate="/usr/sbin/logrotate"
  fi
}

logRotate_install() {
  LOGROTATE_YUM_PACKAGES="logrotate"
  if [ "$(whoami)" == "root" ]; then
    auto_install "Log Rotate" "LOGROTATE"
    if [ $? -ne 0 ]; then echo_failure "Failed to install logrotate"; exit -1; fi
  fi
  logRotate_clear; logRotate_detect;
    if [ -z "$logrotate" ]; then
      echo_failure "logrotate is not installed"
    else
      echo  "logrotate installed in $logrotate"
    fi
}

logRotate_install

export LOG_ROTATION_PERIOD=${LOG_ROTATION_PERIOD:-monthly}
export LOG_COMPRESS=${LOG_COMPRESS:-compress}
export LOG_DELAYCOMPRESS=${LOG_DELAYCOMPRESS:-delaycompress}
export LOG_COPYTRUNCATE=${LOG_COPYTRUNCATE:-copytruncate}
export LOG_SIZE=${LOG_SIZE:-100M}
export LOG_OLD=${LOG_OLD:-12}

mkdir -p /etc/logrotate.d

if [ ! -a /etc/logrotate.d/wlagent ]; then
 echo "/var/log/workload-agent/* {
    missingok
        notifempty
        rotate $LOG_OLD
        maxsize $LOG_SIZE
    nodateext
        $LOG_ROTATION_PERIOD
        $LOG_COMPRESS
        $LOG_DELAYCOMPRESS
        $LOG_COPYTRUNCATE
}" > /etc/logrotate.d/wlagent
fi


# Check if trustagent is intalled; if not output error
hash tagent 2>/dev/null || 
{
  echo_failure >&2 "Trust agent is not installed. Exiting."; 
  exit 1; 
}


# Use tagent user
#### Using trustagent user here as trustagent needs permissions to access files from workload agent
#### for eg signing binding keys. As tagent is a prerequisite for workload-agent, tagent user can be used here
if [ "$(whoami)" == "root" ]; then
  # create a trustagent user if there isn't already one created
  TRUSTAGENT_USERNAME=${TRUSTAGENT_USERNAME}
  if [[ -z $TRUSTAGENT_USERNAME ]]; then
    echo "Using default TRUSTAGENT_USERNAME value 'tagent'"
    TRUSTAGENT_USERNAME=tagent
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
export WORKLOAD_AGENT_CA=$WORKLOAD_AGENT_CONFIGURATION/certs/trustedca
export WORKLOAD_AGENT_FLAVORSIGN=$WORKLOAD_AGENT_CONFIGURATION/certs/flavorsign
export WORKLOAD_AGENT_JWT_CERT=$WORKLOAD_AGENT_CONFIGURATION/certs/trustedjwt
export WORKLOAD_AGENT_LOGS=/var/log/workload-agent
export WORKLOAD_AGENT_HOME=/opt/workload-agent
export WORKLOAD_AGENT_BIN=$WORKLOAD_AGENT_HOME/bin
}
directory_layout

echo "Installing workload agent..."

# Create application directories (chown will be repeated near end of this script, after setup)
for directory in $WORKLOAD_AGENT_CONFIGURATION $WORKLOAD_AGENT_CA $WORKLOAD_AGENT_BIN $WORKLOAD_AGENT_LOGS $WORKLOAD_AGENT_FLAVORSIGN $WORKLOAD_AGENT_JWT_CERT; do
  # mkdir -p will return 0 if directory exists or is a symlink to an existing directory or directory and parents can be created
  mkdir -p $directory 
  if [ $? -ne 0 ]; then
    echo_failure "Cannot create directory: $directory"
    exit 1
  fi
  chown -R $TRUSTAGENT_USERNAME:$TRUSTAGENT_USERNAME $directory
  chmod 700 $directory
done

# Copy workload agent installer to workload-agent bin directory and create a symlink
cp -f wlagent $WORKLOAD_AGENT_BIN
chown $TRUSTAGENT_USERNAME:$TRUSTAGENT_USERNAME $WORKLOAD_AGENT_BIN/wlagent
ln -sfT $WORKLOAD_AGENT_BIN/wlagent /usr/local/bin/wlagent

cp -f wlagent.service $WORKLOAD_AGENT_HOME
systemctl enable $WORKLOAD_AGENT_HOME/wlagent.service


# exit workload-agent setup if WORKLOAD_AGENT_NOSETUP is set
if [ $WLA_NOSETUP == "true" ]; then
  echo "WLA_NOSETUP is set. So, skipping the workload-agent setup task."
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
      echo_warning "$1 must be set and exported (empty value is okay)"
      all_env_vars_present=0
      return 1
    fi
  fi

  if [ "${!1:-}" ]; then
    return 0
  else
    echo_warning "$1 must be set and exported"
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

required_vars="MTWILSON_API_URL WLS_API_URL WLA_SERVICE_USERNAME WLA_SERVICE_PASSWORD CMS_TLS_CERT_SHA384 AAS_API_URL CMS_BASE_URL"
for env_var in $required_vars; do
  check_env_var_present $env_var
done

setup_complete=0
# Call workload-agent setup if all the required env variables are set
if [[ $all_env_vars_present -eq 1 ]]; then
  wlagent setup all
  setup_complete=$?
else 
  echo_failure "One or more environment variables are not present. Setup cannot proceed. Aborting..."
  echo_failure "Please export the missing environment variables and run setup again"
  exit 1
fi

# Enable systemd service and start it
systemctl start wlagent

is_docker_installed(){
  which docker 2>/dev/null
  if [ $? -ne 0 ]; then
    echo_failure "Container Security requires Docker 19.03 to be installed on this system, but docker is not installed"
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
systemctl enable secure-docker-plugin.service 2>/dev/null
systemctl start secure-docker-plugin.service

}


#Install secure docker daemon with WA only if WA_WITH_CONTAINER_SECURITY is enabled in workload-agent.env
#and SKIP_SECURE_DOCKER_DAEMON is not enabled in wpm.env
if [ "$WA_WITH_CONTAINER_SECURITY" == "y" ] || [ "$WA_WITH_CONTAINER_SECURITY" == "Y" ] || [ "$WA_WITH_CONTAINER_SECURITY" == "yes" ]; then
  if [ "$SKIP_SECURE_DOCKER_DAEMON" = "y" ] || [ "$SKIP_SECURE_DOCKER_DAEMON" = "Y" ] || [ "$SKIP_SECURE_DOCKER_DAEMON" = "yes" ]; then
    echo "Skipping secure-docker-daemon installation"
  else
    is_docker_installed

    which cryptsetup 2>/dev/null
    if [ $? -ne 0 ]; then
      yum install -y cryptsetup
    fi  
    echo "Installing secure docker daemon"
    systemctl stop docker
    
    mkdir -p $WORKLOAD_AGENT_HOME/secure-docker-daemon/bin
    cp /usr/bin/docker $WORKLOAD_AGENT_HOME/secure-docker-daemon/bin/
    chown -R root:root docker-daemon/
    
    cp -f docker-daemon/docker /usr/bin/
    which /usr/bin/dockerd-ce 2>/dev/null
    if [ $? -ne 0 ]; then
      cp /usr/bin/dockerd $WORKLOAD_AGENT_HOME/secure-docker-daemon/bin/
      cp -f docker-daemon/dockerd-ce /usr/bin/dockerd
    else
      cp /usr/bin/dockerd-ce $WORKLOAD_AGENT_HOME/secure-docker-daemon/bin/
      cp -f docker-daemon/dockerd-ce /usr/bin/dockerd-ce
    fi

    install_secure_docker_plugin

    echo "Starting secure docker engine"
    mkdir -p /etc/docker
    cp daemon.json /etc/docker/ 
    systemctl daemon-reload
    systemctl start docker
    cp uninstall-container-security-dependencies.sh $WORKLOAD_AGENT_HOME/secure-docker-daemon/
  fi
else
  yum_packages=(libvirt cryptsetup)
  for i in ${yum_packages[*]}
  do
    isinstalled=$(rpm -q $i)
    if [ "$isinstalled" == "package $i is not installed" ]; then
      echo_warning "Prerequisite $i is not installed"
    fi
  done

  isinstalled=$(rpm -q libvirt)
  if [ "$isinstalled" == "package $i is not installed" ]; then 
    echo_warning "failed to setup libvirt hook: libvirt not installed"
    echo_warning "setup libvirt hook with $WORKLOAD_AGENT_HOME/qemu"
    cp -f qemu $WORKLOAD_AGENT_HOME
  else
    mkdir -p /etc/libvirt/hooks
    # Copy isecl-hook script to libvirt hooks directory. The name of hooks should be qemu
    cp -f qemu /etc/libvirt/hooks

    # Restart the libvirt service after copying qemu hook and check if it's running
    systemctl restart libvirtd
    isactive=$(systemctl is-active libvirtd)
    if [ ! "$isactive" == "active" ]; then
      echo_warning "libvirtd system service is not active. Exiting"
      exit 0
    fi
    ## TODO: Above - Should we exit if libvirt restart does not work? 
    ## Maybe we should have a separated setup.sh that can just do the setup tasks. 
  fi
fi

if [ $setup_complete -ne 0 ]; then
  echo_failure "Installation completed completed with errors. Please check log file"
  exit 1
fi

echo_success "Installation completed."
