SHELL:=/bin/bash
GITCOMMIT := $(shell git describe --always)
VERSION := "v4.1.0"
BUILDDATE := $(shell TZ=UTC date +%Y-%m-%dT%H:%M:%S%z)
PROXY_EXISTS := $(shell if [[ "${https_proxy}" || "${http_proxy}" ]]; then echo 1; else echo 0; fi)
DOCKER_PROXY_FLAGS := ""
ifeq ($(PROXY_EXISTS),1)
	DOCKER_PROXY_FLAGS = --build-arg http_proxy=${http_proxy} --build-arg https_proxy=${https_proxy}
else
	undefine DOCKER_PROXY_FLAGS
endif
MONOREPO_GITURL := "https://gitlab.devtools.intel.com/sst/isecl/intel-secl.git"
MONOREPO_GITBRANCH := "v4.1/develop"

.PHONY: wlagent, installer, all, clean, vmc-only

wlagent:
	env GOOS=linux GOSUMDB=off GOPROXY=direct go mod tidy
	export CGO_CFLAGS_ALLOW="-f.*"; \
	env GOOS=linux GOSUMDB=off GOPROXY=direct go build -ldflags "-extldflags=-Wl,--allow-multiple-definition -X main.Version=$(VERSION) -X main.GitHash=$(GITCOMMIT) -X main.BuildDate=$(BUILDDATE)"  -o out/wlagent main.go

installer: wlagent download_upgrade_scripts
	mkdir -p out/wla
	cp dist/linux/install.sh out/wla/install.sh && chmod +x out/wla/install.sh
	cp dist/linux/wlagent.service out/wla/wlagent.service
	cp libvirt/qemu out/wla/qemu && chmod +x out/wla/qemu
	cp out/wlagent out/wla/wlagent && chmod +x out/wla/wlagent

	cp -a out/upgrades/* out/wla/
	cp -a upgrades/* out/wla/
	mv out/wla/build/* out/wla/
	chmod +x out/wla/*.sh

	makeself out/wla out/workload-agent-$(VERSION).bin "Workload Agent $(VERSION)" ./install.sh 

download_upgrade_scripts:
	git clone --depth 1 -b $(MONOREPO_GITBRANCH) $(MONOREPO_GITURL) tmp_monorepo
	cp -a tmp_monorepo/pkg/lib/common/upgrades out/
	chmod +x out/upgrades/*.sh
	rm -rf tmp_monorepo

oci-archive: wlagent download_upgrade_scripts
	docker build ${DOCKER_PROXY_FLAGS} -t isecl/wlagent:$(VERSION) -f dist/docker/Dockerfile .
	skopeo copy docker-daemon:isecl/wlagent:$(VERSION) oci-archive:out/wlagent-$(VERSION)-$(GITCOMMIT).tar

k8s: oci-archive
	cp -r dist/k8s out/

all: installer

clean: 
	rm -rf out/
	rm -rf tmp_monorepo
