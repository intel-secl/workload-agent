#Copyright (C) 2019 Intel Corporation
#SPDX-License-Identifier: BSD-3-Clause
 
FROM centos:8

RUN yum install -y tpm2-tss dmidecode

COPY out/wlagent /usr/bin/

COPY dist/docker/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh && touch /.container-env

CMD ["/entrypoint.sh"]
