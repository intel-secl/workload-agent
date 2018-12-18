#!/bin/bash

# Preconditions:
# * http_proxy and https_proxy are already set, if required
# * date and time are synchronized with remote server, if using remote attestation service
# * the mtwilson linux util functions already sourced
#   (for add_package_repository, echo_success, echo_failure)
# * WORKLOAD_AGENT_HOME is set, for example /opt/workloadagent
# * WORKLOAD_AGENT_INSTALL_LOG_FILE is set, for example /opt/workloadagent/logs/install.log
# * TPM_VERSION is set, for example 1.2 or else it will be auto-detected

# Postconditions:
# * All messages logged to stdout/stderr; caller redirect to logfile as needed

# NOTE:  \cp escapes alias, needed because some systems alias cp to always prompt before override

# Outline:
# 1. Start tcsd (it already has an init script for next boot, but we need it now)

# source functions file

# TODO : no setup-prereqs here... so just an empty file for now
echo "No setup pre-req tasks currently... so exiting this script for now"