#!/bin/bash

if [[ -z $BEARER_TOKEN ]]; then
  echo "BEARER_TOKEN is required for the upgrade to v4.0.0"
  exit 1
fi
