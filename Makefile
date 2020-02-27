GITTAG := $(shell git describe --tags --abbrev=0 2> /dev/null)
GITCOMMIT := $(shell git describe --always)
GITBRANCH := $(shell git rev-parse --abbrev-ref HEAD)
TIMESTAMP := $(shell date --iso=seconds)
VERSION := $(or ${GITTAG}, v0.0.0)

.PHONY: wlagent, installer, all, clean, vmc-only

wlagent:
	env GOOS=linux go build -ldflags "-X main.Version=$(VERSION)-$(GITCOMMIT) -X main.Branch=$(GITBRANCH) -X main.Time=$(TIMESTAMP)"  -o out/wlagent main.go

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
	makeself out/wla out/workload-agent-$(VERSION).bin "Workload Agent $(VERSION)" ./install.sh 

package: wlagent
	mkdir -p out/wla
	cp dist/linux/install.sh out/wla/install.sh && chmod +x out/wla/install.sh
	cp dist/linux/wlagent.service out/wla/wlagent.service
	cp libvirt/qemu out/wla/qemu && chmod +x out/wla/qemu
	cp out/wlagent out/wla/wlagent && chmod +x out/wla/wlagent
	makeself out/wla out/workload-agent-$(VERSION).bin "Workload Agent $(VERSION)" ./install.sh 

all: deploy-artifact

deploy-artifact: installer
	chmod +x dist/linux/deploy-to-artifactory.sh
	dist/linux/deploy-to-artifactory.sh

clean: 
	rm -rf out/
