#Copyright (C) 2019 Intel Corporation
#SPDX-License-Identifier: BSD-3-Clause
 
FROM ubuntu:focal

COPY out/wlagent /usr/bin/

COPY dist/docker/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh && touch /.container-env

#Copy upgrade scripts
COPY out/upgrades/container_upgrade.sh /container_upgrade.sh
COPY out/upgrades/config_upgrade.sh /config_upgrade.sh
COPY upgrades/config/* /config/

RUN apt-get update -y && apt-get install -y curl && apt-get install -y libtss2-esys0 dmidecode && \
    apt-get clean && apt-get autoclean && \
    chmod -R +x /container_upgrade.sh /config_upgrade.sh /config /entrypoint.sh && touch /.container-env

CMD ["/entrypoint.sh"]
