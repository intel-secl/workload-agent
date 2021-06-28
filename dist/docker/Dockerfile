#Copyright (C) 2019 Intel Corporation
#SPDX-License-Identifier: BSD-3-Clause
 
FROM centos:8

RUN yum install -y tpm2-tss dmidecode

COPY out/wlagent /usr/bin/

COPY dist/docker/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh && touch /.container-env

#Copy upgrade scripts
COPY out/upgrades/container_upgrade.sh /container_upgrade.sh
COPY out/upgrades/config_upgrade.sh /config_upgrade.sh
COPY upgrades/config/* /config/

RUN chmod -R +x /container_upgrade.sh /config_upgrade.sh /config

CMD ["/entrypoint.sh"]
