#!/bin/bash

# Preconditions:
# * http_proxy and https_proxy are already set, if required
# * date and time are synchronized with remote server, if using remote attestation service
# * the mtwilson linux util functions already sourced
#   (for add_package_repository, echo_success, echo_failure)
# * TPM_VERSION is set, for example 1.2 or else it will be auto-detected

# Postconditions:
# * All messages logged to stdout/stderr; caller redirect to logfile as needed

# NOTE:  \cp escapes alias, needed because some systems alias cp to always prompt before override

# Outline:
# 1. Install redhat-lsb-core and other redhat-specific packages
# 2. Install trousers and trousers-devel packages (current is trousers-0.3.13-1.el7.x86_64)
# 3. Install the patched tpm-tools
# 4. Install unzip authbind vim-common packages
# 5. Install java
# 6. Install monit

if [[ ${container} == "docker" ]]; then
    DOCKER=true
else
    DOCKER=false
fi

# source functions file
if [ -f functions ]; then . functions; fi

WORKLOAD_AGENTHOME=${WORKLOAD_AGENTHOME:-/opt/workloadagent}
LOGFILE=${WORKLOAD_AGENTINSTALL_LOG_FILE:-$WORKLOAD_AGENTHOME/logs/install.log}
mkdir -p $(dirname $LOGFILE)

if [ -z "$TPM_VERSION" ]; then
  detect_tpm_version
fi

################################################################################

# 1. Install redhat-lsb-core and other redhat-specific packages
## TODO - removed installing redhat packages. Copy back if needed

# 3. Install trousers and trousers-devel packages (current is trousers-0.3.13-1.el7.x86_64)
## TODO - removed installing trousers. Copy back if needed

# 4. Install the patched tpm-tools
## TODO - removed installing tp-tools. Copy back if needed

# 5. Install unzip authbind vim-common packages
# make sure unzip is installed
WORKLOAD_AGENTYUM_PACKAGES="unzip"
WORKLOAD_AGENTAPT_PACKAGES="unzip"
WORKLOAD_AGENTYAST_PACKAGES="unzip"
WORKLOAD_AGENTZYPPER_PACKAGES="unzip"

##### install prereqs can only be done as root
if [ "$(whoami)" == "root" ]; then
  install_packages "Installer requirements" "WORKLOAD_AGENT"
  if [ $? -ne 0 ]; then echo_failure "Failed to install prerequisites through package installer"; exit 1; fi
else
  echo_warning "Required packages:"
  auto_install_preview "TrustAgent requirements" "WORKLOAD_AGENT"
fi

# 6. Install java
## TODO - removing Install Java package Trust Agent requires java 1.8 or later

# 7. Install monit
monit_required_version=5.5

# detect the packages we have to install
MONIT_PACKAGE=`ls -1 monit-*.tar.gz 2>/dev/null | tail -n 1`

# SCRIPT EXECUTION
monit_clear() {
  #MONIT_HOME=""
  monit=""
}

monit_detect() {
  local monitrc=`ls -1 /etc/monitrc 2>/dev/null | tail -n 1`
  monit=`which monit 2>/dev/null`
}

monit_install() {
if [ "$IS_RPM" != "true" ]; then
  MONIT_YUM_PACKAGES="monit"
fi
  MONIT_APT_PACKAGES="monit"
  MONIT_YAST_PACKAGES=""
  MONIT_ZYPPER_PACKAGES="monit"
  install_packages "Monit" "MONIT"
  if [ $? -ne 0 ]; then echo_failure "Failed to install monit through package installer"; return 1; fi
  monit_clear; monit_detect;
    if [[ -z "$monit" ]]; then
      echo_failure "Unable to auto-install Monit"
      echo "  Monit download URL:"
      echo "  http://www.mmonit.com"
    else
      echo_success "Monit installed in $monit"
    fi
}

monit_src_install() {
  local MONIT_PACKAGE="${1:-monit-5.5-linux-src.tar.gz}"
#  DEVELOPER_YUM_PACKAGES="make gcc openssl libssl-dev"
#  DEVELOPER_APT_PACKAGES="dpkg-dev make gcc openssl libssl-dev"
  DEVELOPER_YUM_PACKAGES="make gcc"
  DEVELOPER_APT_PACKAGES="dpkg-dev make gcc"
  install_packages "Developer tools" "DEVELOPER"
  if [ $? -ne 0 ]; then echo_failure "Failed to install developer tools through package installer"; return 1; fi
  monit_clear; monit_detect;
  if [[ -z "$monit" ]]; then
    if [[ -z "$MONIT_PACKAGE" || ! -f "$MONIT_PACKAGE" ]]; then
      echo_failure "Missing Monit installer: $MONIT_PACKAGE"
      return 1
    fi
    local monitfile=$MONIT_PACKAGE
    echo "Installing $monitfile"
    is_targz=`echo $monitfile | grep ".tar.gz$"`
    is_tgz=`echo $monitfile | grep ".tgz$"`
    if [[ -n "$is_targz" || -n "$is_tgz" ]]; then
      gunzip -c $monitfile | tar xf -
    fi
    local monit_unpacked=`ls -1d monit-* 2>/dev/null`
    local monit_srcdir
    for f in $monit_unpacked
    do
      if [ -d "$f" ]; then
        monit_srcdir="$f"
      fi
    done
    if [[ -n "$monit_srcdir" && -d "$monit_srcdir" ]]; then
      echo "Compiling monit..."
      cd $monit_srcdir
      ./configure --without-pam --without-ssl 2>&1 >/dev/null
      make 2>&1 >/dev/null
      make install  2>&1 >/dev/null
    fi
    monit_clear; monit_detect
    if [[ -z "$monit" ]]; then
      echo_failure "Unable to auto-install Monit"
      echo "  Monit download URL:"
      echo "  http://www.mmonit.com"
    else
      echo_success "Monit installed in $monit"
    fi
  else
    echo "Monit is already installed"
  fi
}

##if [ "$(whoami)" == "root" ] && [ ${DOCKER} != "true" ]; then
## TODO - not installing MONIT package for now. Uncomment if needed
##  monit_install $MONIT_PACKAGE
##else
##  echo_warning "Skipping monit installation"
##fi

if [ -n $result ]; then exit $result; fi