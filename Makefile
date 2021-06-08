SHELL:=/bin/bash
GITCOMMIT := $(shell git describe --always)
VERSION := "v4.0.0"
BUILDDATE := $(shell TZ=UTC date +%Y-%m-%dT%H:%M:%S%z)
PROXY_EXISTS := $(shell if [[ "${https_proxy}" || "${http_proxy}" ]]; then echo 1; else echo 0; fi)
DOCKER_PROXY_FLAGS := ""
MONOREPO_GITURL := "ssh://git@gitlab.devtools.intel.com:29418/sst/isecl/intel-secl.git"
MONOREPO_GITBRANCH := "v4.0/develop"

.PHONY: wlagent, installer, all, clean, vmc-only

wlagent:
	export CGO_CFLAGS_ALLOW="-f.*"; \
	env GOOS=linux GOSUMDB=off GOPROXY=direct go build -ldflags "-X main.Version=$(VERSION) -X main.GitHash=$(GITCOMMIT) -X main.BuildDate=$(BUILDDATE)"  -o out/wlagent main.go

installer: wlagent
	mkdir -p out/wla
	cp dist/linux/install.sh out/wla/install.sh && chmod +x out/wla/install.sh
	cp dist/linux/wlagent.service out/wla/wlagent.service
	cp libvirt/qemu out/wla/qemu && chmod +x out/wla/qemu
	cp out/wlagent out/wla/wlagent && chmod +x out/wla/wlagent
	chmod +x dist/linux/build-container-security-dependencies.sh
	dist/linux/build-container-security-dependencies.sh
	cp -rlf secure-docker-daemon/out out/wla/docker-daemon
	rm -rf secure-docker-daemon
	cp -rlf secure-docker-plugin out/
	rm -rf secure-docker-plugin
	cp -r out/secure-docker-plugin/secure-docker-plugin out/wla/
	cp dist/linux/daemon.json out/wla/
	cp -rf out/secure-docker-plugin/artifact out/wla/
	cp dist/linux/uninstall-container-security-dependencies.sh out/wla/uninstall-container-security-dependencies.sh && chmod +x out/wla/uninstall-container-security-dependencies.sh

	git clone --depth 1 -b $(MONOREPO_GITBRANCH) $(MONOREPO_GITURL) tmp_monorepo
	cp -a tmp_monorepo/pkg/lib/common/upgrades/* out/wla/
	rm -rf tmp_monorepo
	cp -a upgrades/* out/wla/
	mv out/wla/build/* out/wla/
	chmod +x out/wla/*.sh

	makeself out/wla out/workload-agent-$(VERSION).bin "Workload Agent $(VERSION)" ./install.sh 

installer-no-docker: wlagent
	mkdir -p out/wla
	cp dist/linux/install.sh out/wla/install.sh && chmod +x out/wla/install.sh
	cp dist/linux/wlagent.service out/wla/wlagent.service
	cp libvirt/qemu out/wla/qemu && chmod +x out/wla/qemu
	cp out/wlagent out/wla/wlagent && chmod +x out/wla/wlagent

	git clone --depth 1 -b $(MONOREPO_GITBRANCH) $(MONOREPO_GITURL) tmp_monorepo
	cp -a tmp_monorepo/pkg/lib/common/upgrades/* out/wla/
	rm -rf tmp_monorepo
	cp -a upgrades/* out/wla/
	mv out/wla/build/* out/wla/
	chmod +x out/wla/*.sh

	makeself out/wla out/workload-agent-$(VERSION).bin "Workload Agent $(VERSION)" ./install.sh 

package: wlagent
	mkdir -p out/wla
	cp dist/linux/install.sh out/wla/install.sh && chmod +x out/wla/install.sh
	cp dist/linux/wlagent.service out/wla/wlagent.service
	cp libvirt/qemu out/wla/qemu && chmod +x out/wla/qemu
	cp out/wlagent out/wla/wlagent && chmod +x out/wla/wlagent
	makeself out/wla out/workload-agent-$(VERSION).bin "Workload Agent $(VERSION)" ./install.sh 

oci-archive: wlagent
ifeq ($(PROXY_EXISTS),1)
	docker build -t isecl/wlagent:$(VERSION) --build-arg http_proxy=${http_proxy} --build-arg https_proxy=${https_proxy} -f dist/docker/Dockerfile .
else
	docker build -t isecl/wlagent:$(VERSION) -f dist/docker/Dockerfile .
endif
	skopeo copy docker-daemon:isecl/wlagent:$(VERSION) oci-archive:out/wlagent-$(VERSION)-$(GITCOMMIT).tar

k8s: oci-archive
	cp -r dist/k8s out/

all: installer

clean: 
	rm -rf out/
