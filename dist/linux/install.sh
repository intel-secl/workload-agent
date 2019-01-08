#!/bin/bash

# Postconditions:
# * exit with error code 1 only if there was a fatal error:
#####

# WARNING:
# *** do NOT use TABS for indentation, use SPACES
# *** TABS will cause errors in some linux distributions

# WORKLOAD_AGENT install script 
# Outline:
# 1. Check if installer is running as a root
# 2. Load the environment file
# 3. Check if WORKLOAD_AGENT_NOSETUP is set in environment file
# 4. Check if trustagent is intalled
# 5. Load tagent username to a variable
# 6. Load local configurations
# 7. Create application directories
# 8. Copy workload agent installer to workloadagent bin directory and create a symlink
# 9. Call workloadagent setup
# 10. Install and setup libvirt
# 11. Copy isecl-hook script to libvirt hooks directory
# 12. Restart the libvirt service after copying qemu hook

DEFAULT_TRUSTAGENT_USERNAME=tagent

# Log rotate configurations
export LOG_ROTATE_MAX_SIZE=${LOG_ROTATE_MAX_SIZE:-100000}
export LOG_ROTATE_MAX_BACKUPS=${LOG_ROTATE_MAX_BACKUPS:-8}
export LOG_ROTATE_MAX_DAYS=${LOG_ROTATE_MAX_DAYS:-90}

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

# 1. Product installation is only allowed if we are running as root
if [ $EUID -ne 0 ];  then
  echo "Workload agent installation has to run as root. Exiting"
  exit 1
fi

# 2. load installer environment file, if present
if [ -f ~/workloadagent.env ]; then
  echo "Loading environment variables from $(cd ~ && pwd)/workloadagent.env"
  . ~/workloadagent.env
  env_file_exports=$(cat ~/workloadagent.env | grep -E '^[A-Z0-9_]+\s*=' | cut -d = -f 1)
  if [ -n "$env_file_exports" ]; then eval export $env_file_exports; fi
else
  echo "No environment file"
fi

# 3. exit workloadagent setup if WORKLOAD_AGENT_NOSETUP is set
if [ -n "$WORKLOAD_AGENT_NOSETUP" ]; then
  echo "WORKLOAD_AGENT_NOSETUP value is set. So, skipping the workloadagent setup task."
  exit 0
fi

# 4. Check if trustagent is intalled; if not output error
hash tagent 2>/dev/null || 
{
  echo_failure >&2 "Trust agent is not installed. Exiting."; 
  exit 1; 
}


# 5. Use tagent user
#### Using trustagent user here as trustagent needs permissions to access files from workload agent
#### for eg signing binding keys. As tagent is a prerequisite for workloadagent, tagent user can be used here
if [ "$(whoami)" == "root" ]; then
  # create a trustagent user if there isn't already one created
  TRUSTAGENT_USERNAME=${TRUSTAGENT_USERNAME:-$DEFAULT_TRUSTAGENT_USERNAME}
else
  # already running as trustagent user
  TRUSTAGENT_USERNAME=$(whoami)
  if [ ! -w "$TRUSTAGENT_HOME" ] && [ ! -w $(dirname $TRUSTAGENT_HOME) ]; then
    TRUSTAGENT_HOME=$(cd ~ && pwd)
  fi
  echo_warning "Installing as $TRUSTAGENT_USERNAME into $TRUSTAGENT_HOME"  
fi

# 6. Load local configurations
directory_layout() {
export WORKLOAD_AGENT_HOME=/opt/workloadagent
export WORKLOAD_AGENT_CONFIGURATION=${WORKLOAD_AGENT_CONFIGURATION:-/etc/workloadagent}
export TRUST_AGENT_CONFIGURATION=${TRUST_AGENT_CONFIGURATION:-/opt/trustagent/configuration}
export WORKLOAD_AGENT_LOGS=${WORKLOAD_AGENT_LOGS:-/var/log/workloadagent}
export WORKLOAD_AGENT_BIN=${WORKLOAD_AGENT_BIN:-$WORKLOAD_AGENT_HOME/bin}
export INSTALL_LOG_FILE=$WORKLOAD_AGENT_LOGS/install.log
}
directory_layout

mkdir -p $(dirname $INSTALL_LOG_FILE)
if [ $? -ne 0 ]; then
  echo_failure "Cannot create directory: $(dirname $INSTALL_LOG_FILE)"
  exit 1
fi
logfile=$INSTALL_LOG_FILE

# 7. Create application directories (chown will be repeated near end of this script, after setup)
for directory in $WORKLOAD_AGENT_HOME $WORKLOAD_AGENT_CONFIGURATION $WORKLOAD_AGENT_BIN $WORKLOAD_AGENT_LOGS; do
  # mkdir -p will return 0 if directory exists or is a symlink to an existing directory or directory and parents can be created
  mkdir -p $directory 
  if [ $? -ne 0 ]; then
    echo_failure "Cannot create directory: $directory" 2>>$logfile
    exit 1
  fi
  chown -R $TRUSTAGENT_USERNAME:$TRUSTAGENT_USERNAME $directory
  chmod 700 $directory
done

# 8. Copy workload agent installer to workloadagent bin directory and create a symlink
cp -f wlagent $WORKLOAD_AGENT_BIN
chown $TRUSTAGENT_USERNAME:$TRUSTAGENT_USERNAME $WORKLOAD_AGENT_BIN/wlagent
ln -sfT $WORKLOAD_AGENT_BIN/wlagent /usr/local/bin/wlagent

# 9. Call workloadagent setup
wlagent setup | tee $logfile

# Make sure all files created after setup tasks have tagent user ownsership
for directory in $WORKLOAD_AGENT_HOME $WORKLOAD_AGENT_CONFIGURATION $WORKLOAD_AGENT_BIN $WORKLOAD_AGENT_LOGS; do
  chown -R $TRUSTAGENT_USERNAME:$TRUSTAGENT_USERNAME $directory
  chmod 700 $directory
done

# 10. Check if yum packages are already installed; if not install them
yum_packages=(libvirt cryptsetup)
for i in ${yum_packages[*]}
do
  isinstalled=$(rpm -q $i)
  if [ "$isinstalled" == "package $i is not installed" ]; then
    yum -y install $i 2>>$logfile
  fi
else
  echo_warning "Logback configuration not found: $WORKLOAD_AGENT_CONFIGURATION/logback.xml"
fi

chown -R $WORKLOAD_AGENT_USERNAME:$WORKLOAD_AGENT_USERNAME $WORKLOAD_AGENT_HOME
#chmod 755 $WORKLOAD_AGENT_BIN/*

## TODO : migration code - removing for now... add back in if needed
# 18. migrate any old data to the new locations (v1 - v3)  (should be rewritten in java)

# Redefine the variables to the new locations
package_config_filename=$WORKLOAD_AGENT_CONFIGURATION/workloadagent.properties


# 21. copy utilities script file to application folder
mkdir -p "$WORKLOAD_AGENT_HOME"/share/scripts
cp version "$WORKLOAD_AGENT_HOME"/share/scripts/version.sh
cp functions "$WORKLOAD_AGENT_HOME"/share/scripts/functions.sh
chmod -R 700 "$WORKLOAD_AGENT_HOME"/share/scripts
chown -R $WORKLOAD_AGENT_USERNAME:$WORKLOAD_AGENT_USERNAME "$WORKLOAD_AGENT_HOME"/share/scripts
chmod +x $WORKLOAD_AGENT_BIN/*


# 24. create workloadagent-version file
package_version_filename=$WORKLOAD_AGENT_ENV/workloadagent-version
datestr=`date +%Y-%m-%d.%H%M`
touch $package_version_filename
chmod 600 $package_version_filename
chown $WORKLOAD_AGENT_USERNAME:$WORKLOAD_AGENT_USERNAME $package_version_filename
echo "# Installed Trust Agent on ${datestr}" > $package_version_filename
echo "WORKLOAD_AGENT_VERSION=${VERSION}" >> $package_version_filename
echo "WORKLOAD_AGENT_RELEASE=\"${BUILD}\"" >> $package_version_filename

##TODO - workloadagent is not a deamon - at least for now. Don't do any registration
# during a Docker image build, we don't know if 1.2 is going to be used, defer this until Docker startup script.
##if [[ "$(whoami)" == "root" && ${DOCKER} == "false" ]]; then
##  echo "Registering wlagent in start up"
##  register_startup_script $WORKLOAD_AGENT_BIN/wlagent wlagent 21 >>$logfile 2>&1
##  # trousers has N=20 startup number, need to lookup and do a N+1
##else
##  echo_warning "Skipping startup script registration"
##fi

## TODO - removing monit related code - copy over if needed
# 26. configure monit

##TODO : workloadagent runs under the launching user. No need to change the user
# Ensure we have given workloadagent access to its files
##for directory in $WORKLOAD_AGENT_HOME $WORKLOAD_AGENT_CONFIGURATION $WORKLOAD_AGENT_ENV $WORKLOAD_AGENT_REPOSITORY $WORKLOAD_AGENT_VAR $WORKLOAD_AGENT_LOGS; do
##  echo "chown -R $WORKLOAD_AGENT_USERNAME:$WORKLOAD_AGENT_USERNAME $directory" >>$logfile
##  chown -R $WORKLOAD_AGENT_USERNAME:$WORKLOAD_AGENT_USERNAME $directory 2>>$logfile
##done

##TODO - do we need update any system info related to workloadagent. 
##if [[ "$(whoami)" == "root" && ${DOCKER} != "true" ]]; then
##  echo "Updating system information"
##  wlagent update-system-info 2>/dev/null
##else
##  echo_warning "Skipping updating system information"
##fi

# Make the logs dir owned by wlagent user
##chown -R $WORKLOAD_AGENT_USERNAME:$WORKLOAD_AGENT_USERNAME $WORKLOAD_AGENT_LOGS/


# 29. ensure the workloadagent owns all the content created during setup
##for directory in $WORKLOAD_AGENT_HOME $WORKLOAD_AGENT_CONFIGURATION $WORKLOAD_AGENT_JAVA $WORKLOAD_AGENT_BIN $WORKLOAD_AGENT_ENV $WORKLOAD_AGENT_REPOSITORY $WORKLOAD_AGENT_LOGS; do
##  chown -R $WORKLOAD_AGENT_USERNAME:$WORKLOAD_AGENT_USERNAME $directory
##done


# exit workloadagent setup if WORKLOAD_AGENT_NOSETUP is set
if [ -n "$WORKLOAD_AGENT_NOSETUP" ]; then
  echo "WORKLOAD_AGENT_NOSETUP value is set. So, skipping the workloadagent setup task."
  exit 0;
fi

echo_info "Copying the workload agent bin to /usr/local/bin/"
cp $WORKLOAD_AGENT_BIN/wlagent /usr/local/bin/ 

# 33. wlagent setup
wlagent setup 


echo_warning "TODO : Need to install hooks to libvrt - writing configuration directory "
if [ ! -d "/etc/libvirt" ]; then
  echo_warning "libvirt directory not present. Exiting"
  exit 0
fi

mkdir -p "/etc/libvirt/hooks" 
if [ ! -d "/etc/libvirt/hooks" ];  then
  echo_warning "Not able to create hooks directory. Exiting"
  echo 0
fi

# 11. Copy isecl-hook script to libvirt hooks directory. The name of hooks should be qemu
cp -f qemu /etc/libvirt/hooks 

# destination file needs to be called qemu

fill_with_variable_value "qemu" "<AUTOFILL_AT_INSTALL>" "/etc/libvirt/hooks/qemu"

##TODO - any sort of setup tasks after setup
## such as creating the hook for libvrt
# 34. wlagest post-setup
# wlagent post set up taks

## TODO - monit restart if applicable - copy over 
# 34. restart monit

## TODO: commenting out logrotate portion for now.. Fix when logrotation is enabled
########################################################################################################################
# 35. config logrotate 
##mkdir -p /etc/logrotate.d

echo_success "Installation completed."
