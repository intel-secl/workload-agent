#!/bin/bash

# Postconditions:
# * exit with error code 1 only if there was a fatal error:
#   functions.sh not found (must be adjacent to this file in the package)
#   

# WORKLOAD_AGENT install script
# Outline:
# 1. load application environment variables if already defined from env directory
# 2. load installer environment file, if present
# 3. source the utility script file "functions.sh":  workloadagent-linux-util-3.0-SNAPSHOT.sh
# 4. source the version script file "version"
# 5. define application directory layout
# 6. install pre-required packages
# 7. determine if we are installing as root or non-root, create groups and users accordingly
# 9. remove wlagent from the monit config, stop wlagent and restart monit
# 10. backup current configuration and data, if they exist
# 11. store directory layout in env file
# 12. store workloadagent username in env file
# 13. store log level in env file, if it's set
# 14. If VIRSH_DEFAULT_CONNECT_URI is defined in environment copy it to env directory
# 15. extract workloadagent zip
# 16. symlink wlagent

# 18. migrate any old data to the new locations (v1 - v3)
# 19. setup authbind to allow non-root workloadagent to listen on ports 80 and 443
# 20. create tpm-tools and additional binary symlinks
# 21. copy utilities script file to application folder
# 22. delete existing dependencies from java folder, to prevent duplicate copies
# 23. fix_libcrypto for RHEL
# 24. create workloadagent-version file
# 25. fix_existing_aikcert
# 26. configure monit
# 27. create WORKLOAD_AGENT_TLS_CERT_IP list of system host addresses
# 28. update the extensions cache file
# 29. ensure the workloadagent owns all the content created during setup

# 31. wlagent start
# 32. wlagent setup
# 33. register tpm password with mtwilson
# 34. restart monit
# 35. config logrotate


#####


# WARNING:
# *** do NOT use TABS for indentation, use SPACES
# *** TABS will cause errors in some linux distributions

# application defaults (these are not configurable and used only in this script so no need to export)
DEFAULT_WORKLOAD_AGENT_HOME=/opt/workloadagent
DEFAULT_WORKLOAD_AGENT_USERNAME=wlagent
if [[ ${container} == "docker" ]]; then
    DOCKER=true
else
    DOCKER=false
fi

echo "Dockerized install is: $DOCKER"

# check if we are running in a docker container or running as root. Product installation is only
# allowed if we are running as root
if [ "$(whoami)" != "root" ] && [ $DOCKER != "true" ]; then
  echo "Workload agent installation has to run as root. Exiting"
  exit 1
fi

EXISTING_TAGENT_COMMAND=`which wlagent 2>/dev/null`
if [ -n "$EXISTING_TAGENT_COMMAND" ]; then
  rm -f "$EXISTING_TAGENT_COMMAND"
fi

# default settings
export LOG_ROTATION_PERIOD=${LOG_ROTATION_PERIOD:-monthly}
export LOG_COMPRESS=${LOG_COMPRESS:-compress}
export LOG_DELAYCOMPRESS=${LOG_DELAYCOMPRESS:-delaycompress}
export LOG_COPYTRUNCATE=${LOG_COPYTRUNCATE:-copytruncate}
export LOG_SIZE=${LOG_SIZE:-1G}
export LOG_OLD=${LOG_OLD:-12}
export PROVISION_ATTESTATION=${PROVISION_ATTESTATION:-y}
export WORKLOAD_AGENT_ADMIN_USERNAME=${WORKLOAD_AGENT_ADMIN_USERNAME:-wlagent-admin}
export REGISTER_TPM_PASSWORD=${REGISTER_TPM_PASSWORD:-y}
export WORKLOAD_AGENT_LOGIN_REGISTER=${WORKLOAD_AGENT_LOGIN_REGISTER:-true}
export WORKLOAD_AGENT_HOME=${WORKLOAD_AGENT_HOME:-$DEFAULT_WORKLOAD_AGENT_HOME}
WORKLOAD_AGENT_LAYOUT=${WORKLOAD_AGENT_LAYOUT:-home}

# the env directory is not configurable; it is defined as WORKLOAD_AGENT_HOME/env.d and the
# administrator may use a symlink if necessary to place it anywhere else
export WORKLOAD_AGENT_ENV=$WORKLOAD_AGENT_HOME/env.d

# 1. load application environment variables if already defined from env directory
if [ -d $WORKLOAD_AGENT_ENV ]; then
  WORKLOAD_AGENT_ENV_FILES=$(ls -1 $WORKLOAD_AGENT_ENV/*)
  for env_file in $WORKLOAD_AGENT_ENV_FILES; do
    . $env_file
    env_file_exports=$(cat $env_file | grep -E '^[A-Z0-9_]+\s*=' | cut -d = -f 1)
    if [ -n "$env_file_exports" ]; then eval export $env_file_exports; fi
  done
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


# 3. source the utility script file "functions.sh":  mtwilson-linux-util-3.0-SNAPSHOT.sh
# FUNCTION LIBRARY
## TODO - no function file exists - so don't exits for now
##if [ -f functions ]; then . functions; else echo "Missing file: functions"; exit 1; fi
if [ -f functions ]; then . functions; else echo "Missing file: functions"; fi

# 4. source the version script file "version"
# VERSION INFORMATION
if [ -f version ]; then . version; else echo_warning "Missing file: version"; fi
# The version script is automatically generated at build time and looks like this:
#ARTIFACT=mtwilson-workloadagent-installer
#VERSION=3.0
#BUILD="Fri, 5 Jun 2015 15:55:20 PDT (release-3.0)"


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

# 5. define application directory layout
directory_layout

## TODO - What do we need to do for TPM version in a simulator environment?
## detect_tpm_version

# 6. install pre-required packages
##chmod +x install_prereq.sh
##./install_prereq.sh
##ipResult=$?

#i#f [ "${WORKLOAD_AGENT_SETUP_PREREQS:-yes}" == "yes" ]; then
  # set WORKLOAD_AGENT_REBOOT=no (in workloadagent.env) if you want to ensure it doesn't reboot
  # set WORKLOAD_AGENT_SETUP_PREREQS=no (in workloadagent.env) if you want to skip this step 
##  chmod +x setup_prereqs.sh
##  ./setup_prereqs.sh
##  spResult=$?
##fi

## TODO : determine if there is a reboot required. Commenting for now
## refactor this check after install_prereq.sh and setup_prereq.sh are sorted out
##if [[ $ipResult -eq 255 ]] && [[ $spResult -ne 255 ]]; then
##  echo
##  echo "Trust Agent: A reboot is required. Please reboot and run installer again."
##  echo
##fi
##if [[ $ipResult -eq 255 ]] || [[ $spResult -eq 255 ]]; then
##  mkdir -p "$WORKLOAD_AGENT_HOME/var"
##  touch "$WORKLOAD_AGENT_HOME/var/reboot_required"
##  exit 255
##fi

## TODO : Not sure if we need to create a local user. Uncomment ## if relevant
# 7. determine if we are installing as root or non-root, create groups and users accordingly
##if [ "$(whoami)" == "root" ]; then
  # create a workloadagent user if there isn't already one created
##  WORKLOAD_AGENT_USERNAME=${WORKLOAD_AGENT_USERNAME:-$DEFAULT_WORKLOAD_AGENT_USERNAME}
##  if ! getent passwd $WORKLOAD_AGENT_USERNAME 2>&1 >/dev/null; then
##    useradd --comment "ISecL Workload Agent" --home $WORKLOAD_AGENT_HOME --system --shell /bin/false $WORKLOAD_AGENT_USERNAME
##    usermod --lock $WORKLOAD_AGENT_USERNAME
    # note: to assign a shell and allow login you can run "usermod --shell /bin/bash --unlock $WORKLOAD_AGENT_USERNAME"
##  fi
##else
  # already running as workloadagent user
##  WORKLOAD_AGENT_USERNAME=$(whoami)
##  if [ ! -w "$WORKLOAD_AGENT_HOME" ] && [ ! -w $(dirname $WORKLOAD_AGENT_HOME) ]; then
##    WORKLOAD_AGENT_HOME=$(cd ~ && pwd)
##  fi
##  echo_warning "Installing as $WORKLOAD_AGENT_USERNAME into $WORKLOAD_AGENT_HOME"  
##fi

# 7.a set the  WORKLOAD_AGENT_USERNAME as the current user if not running as root
# This is only relevant for testing in docker environment

WORKLOAD_AGENT_USERNAME=$(whoami)
if [ ! -w "$WORKLOAD_AGENT_HOME" ] && [ ! -w $(dirname $WORKLOAD_AGENT_HOME) ]; then
  WORKLOAD_AGENT_HOME=$(cd ~ && pwd)
else
  echo_warning "Installing as $(whoami) into $WORKLOAD_AGENT_HOME"  
fi

directory_layout


# before we start, clear the install log (directory must already exist; created above)
mkdir -p $(dirname $INSTALL_LOG_FILE)
if [ $? -ne 0 ]; then
  echo_failure "Cannot write to log directory: $(dirname $INSTALL_LOG_FILE)"
  exit 1
fi
date > $INSTALL_LOG_FILE
if [ $? -ne 0 ]; then
  echo_failure "Cannot write to log file: $INSTALL_LOG_FILE"
  exit 1
fi
chown $WORKLOAD_AGENT_USERNAME:$WORKLOAD_AGENT_USERNAME $INSTALL_LOG_FILE
logfile=$INSTALL_LOG_FILE

# 8. create application directories (chown will be repeated near end of this script, after setup)
for directory in $WORKLOAD_AGENT_HOME $WORKLOAD_AGENT_CONFIGURATION $WORKLOAD_AGENT_ENV $WORKLOAD_AGENT_REPOSITORY $WORKLOAD_AGENT_VAR $WORKLOAD_AGENT_LOGS; do
  # mkdir -p will return 0 if directory exists or is a symlink to an existing directory or directory and parents can be created
  mkdir -p $directory
  if [ $? -ne 0 ]; then
    echo_failure "Cannot create directory: $directory"
    exit 1
  fi
  chown -R $WORKLOAD_AGENT_USERNAME:$WORKLOAD_AGENT_USERNAME $directory
  chmod 700 $directory
done

# ensure we have our own wlagent programs in the path
export PATH=$WORKLOAD_AGENT_BIN:$PATH

# ensure that trousers and tpm tools are in the path
##TODO - Are there any paths that need to be added. Comment for now
##export PATH=$PATH:/usr/sbin:/usr/local/sbin

profile_dir=$HOME
if [ "$(whoami)" == "root" ] && [ -n "$WORKLOAD_AGENT_USERNAME" ] && [ "$WORKLOAD_AGENT_USERNAME" != "root" ]; then
  profile_dir=$WORKLOAD_AGENT_HOME
fi
profile_name=$profile_dir/$(basename $(getUserProfileFile))

appendToUserProfileFile "export WORKLOAD_AGENT_HOME=$WORKLOAD_AGENT_HOME" $profile_name

## TODO. Workload Agent is not a deamon. So there is no process for monit
## Comment out code related to monit

# 9. remove wlagent from the monit config, stop wlagent and restart monit
# if there's a monit configuration for workloadagent, remove it to prevent
# monit from trying to restart workloadagent while we are setting up
##if [ "$(whoami)" == "root" ] && [ -f /etc/monit/conf.d/ta.monit ]; then
##  datestr=`date +%Y%m%d.%H%M`
##  backupdir=$WORKLOAD_AGENT_BACKUP/monit.configuration.$datestr
##  mkdir -p $backupdir
##  mv /etc/monit/conf.d/ta.monit $backupdir
##  service monit restart
##fi

# if an existing wlagent is already running, stop it while we install
##existing_wlagent=`which wlagent 2>/dev/null`
##if [ -f "$existing_wlagent" ]; then
##  $existing_wlagent stop
##fi

workloadagent_backup_configuration() {
  if [ -n "$WORKLOAD_AGENT_CONFIGURATION" ] && [ -d "$WORKLOAD_AGENT_CONFIGURATION" ]; then
    mkdir -p $WORKLOAD_AGENT_BACKUP
    if [ $? -ne 0 ]; then
      echo_warning "Cannot create backup directory: $WORKLOAD_AGENT_BACKUP"
      echo_warning "Backup will be stored in /tmp"
      WORKLOAD_AGENT_BACKUP=/tmp
    fi
    datestr=`date +%Y%m%d.%H%M`
    backupdir=$WORKLOAD_AGENT_BACKUP/workloadagent.configuration.$datestr
    cp -r $WORKLOAD_AGENT_CONFIGURATION $backupdir
  fi
}
workloadagent_backup_repository() {
  if [ -n "$WORKLOAD_AGENT_REPOSITORY" ] && [ -d "$WORKLOAD_AGENT_REPOSITORY" ]; then
    mkdir -p $WORKLOAD_AGENT_BACKUP
    if [ $? -ne 0 ]; then
      echo_warning "Cannot create backup directory: $WORKLOAD_AGENT_BACKUP"
      echo_warning "Backup will be stored in /tmp"
      WORKLOAD_AGENT_BACKUP=/tmp
    fi
    datestr=`date +%Y%m%d.%H%M`
    backupdir=$WORKLOAD_AGENT_BACKUP/workloadagent.repository.$datestr
    cp -r $WORKLOAD_AGENT_REPOSITORY $backupdir
  fi
}

# 10. backup current configuration and data, if they exist
workloadagent_backup_configuration
#workloadagent_backup_repository

# 11. store directory layout in env file
echo "# $(date)" > $WORKLOAD_AGENT_ENV/workloadagent-layout
echo "WORKLOAD_AGENT_HOME=$WORKLOAD_AGENT_HOME" >> $WORKLOAD_AGENT_ENV/workloadagent-layout
echo "WORKLOAD_AGENT_CONFIGURATION=$WORKLOAD_AGENT_CONFIGURATION" >> $WORKLOAD_AGENT_ENV/workloadagent-layout
echo "WORKLOAD_AGENT_BIN=$WORKLOAD_AGENT_BIN" >> $WORKLOAD_AGENT_ENV/workloadagent-layout
echo "WORKLOAD_AGENT_REPOSITORY=$WORKLOAD_AGENT_REPOSITORY" >> $WORKLOAD_AGENT_ENV/workloadagent-layout
echo "WORKLOAD_AGENT_LOGS=$WORKLOAD_AGENT_LOGS" >> $WORKLOAD_AGENT_ENV/workloadagent-layout

##TODO : should not need below - but confirm
## 12. store workloadagent username in env file
##echo "# $(date)" > $WORKLOAD_AGENT_ENV/workloadagent-username
##echo "WORKLOAD_AGENT_USERNAME=$WORKLOAD_AGENT_USERNAME" >> $WORKLOAD_AGENT_ENV/workloadagent-username

# 13. store log level in env file, if it's set
if [ -n "$WORKLOAD_AGENT_LOG_LEVEL" ]; then
  echo "# $(date)" > $WORKLOAD_AGENT_ENV/workloadagent-logging
  echo "WORKLOAD_AGENT_LOG_LEVEL=$WORKLOAD_AGENT_LOG_LEVEL" >> $WORKLOAD_AGENT_ENV/workloadagent-logging
fi

# store the auto-exported environment variables in temporary env file
# to make them available after the script uses sudo to switch users;
# we delete that file later
echo "# $(date)" > $WORKLOAD_AGENT_ENV/workloadagent-setup
for env_file_var_name in $env_file_exports
do
  eval env_file_var_value="\$$env_file_var_name"
  echo "export $env_file_var_name='$env_file_var_value'" >> $WORKLOAD_AGENT_ENV/workloadagent-setup
done


## TODO - uncomment if relevant
# save tpm version in trust agent configuration directory
# if we are building a container, defer this until docker run (first setup)
##if [ $DOCKER != "true" ]; then
##    echo -n "$TPM_VERSION" > $WORKLOAD_AGENT_CONFIGURATION/tpm-version
##fi

# 14. If VIRSH_DEFAULT_CONNECT_URI is defined in environment copy it to env directory (likely from ~/.bashrc)
# copy it to our new env folder so it will be available to wlagent on startup
##if [ -n "$LIBVIRT_DEFAULT_URI" ]; then
##  echo "LIBVIRT_DEFAULT_URI=$LIBVIRT_DEFAULT_URI" > $WORKLOAD_AGENT_ENV/virsh
##elif [ -n "$VIRSH_DEFAULT_CONNECT_URI" ]; then
##  echo "VIRSH_DEFAULT_CONNECT_URI=$VIRSH_DEFAULT_CONNECT_URI" > $WORKLOAD_AGENT_ENV/virsh
##fi

cp version $WORKLOAD_AGENT_CONFIGURATION/workloadagent-version

# 15. extract workloadagent zip  (workloadagent-zip-0.1-SNAPSHOT.zip)
echo "Copy workload agent binary"
WORKLOAD_AGENT_ZIPFILE=`ls -1 workloadagent-*.zip 2>/dev/null | head -n 1`
echo tar -xvf $WORKLOAD_AGENT_ZIPFILE -C $WORKLOAD_AGENT_HOME
tar -xvf $WORKLOAD_AGENT_ZIPFILE -C $WORKLOAD_AGENT_HOME

# add bin and sbin directories in workloadagent home directory to path
bin_directories=$(find_subdirectories ${WORKLOAD_AGENT_HOME} bin; find_subdirectories ${WORKLOAD_AGENT_HOME} sbin)
bin_directories_path=$(join_by : ${bin_directories[@]})
for directory in ${bin_directories[@]}; do
  chmod -R 700 $directory
  echo $directory
done
export PATH=$bin_directories_path:$PATH
appendToUserProfileFile "export PATH=${bin_directories_path}:\$PATH" $profile_name

## TODO - there should not be amy library files. Uncomment if relevant
# add lib directories in workloadagent home directory to LD_LIBRARY_PATH variable env file
##lib_directories=$(find_subdirectories ${WORKLOAD_AGENT_HOME}/share lib)
##lib_directories_path=$(join_by : ${lib_directories[@]})
##export LD_LIBRARY_PATH=$lib_directories_path
##echo "export LD_LIBRARY_PATH=${lib_directories_path}" > $WORKLOAD_AGENT_ENV/workloadagent-lib

##TODO - rework as appropriate when we identify the logging mechanism
# update logback.xml with configured workloadagent log directory
if [ -f "$WORKLOAD_AGENT_CONFIGURATION/logback.xml" ]; then
  sed -e "s|<file>.*/workloadagent.log</file>|<file>$WORKLOAD_AGENT_LOGS/workloadagent.log</file>|" $WORKLOAD_AGENT_CONFIGURATION/logback.xml > $WORKLOAD_AGENT_CONFIGURATION/logback.xml.edited
  if [ $? -eq 0 ]; then
    mv $WORKLOAD_AGENT_CONFIGURATION/logback.xml.edited $WORKLOAD_AGENT_CONFIGURATION/logback.xml
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

##if [ ! -a /etc/logrotate.d/workloadagent ]; then
## echo "/opt/workloadagent/logs/workloadagent.log {
##    missingok
##	notifempty
##	rotate $LOG_OLD
##	maxsize $LOG_SIZE
##    nodateext
##	$LOG_ROTATION_PERIOD 
##	$LOG_COMPRESS
##	$LOG_DELAYCOMPRESS
##	$LOG_COPYTRUNCATE
##}" > /etc/logrotate.d/workloadagent
##fi
