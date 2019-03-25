VERSION := v1.0
GITCOMMIT := $(shell git describe --always)
GITBRANCH := $(shell git rev-parse --abbrev-ref HEAD)
TIMESTAMP := $(shell date --iso=seconds)

.PHONY: wlagent, installer, all, clean

wlagent:
	env GOOS=linux go build -ldflags "-X main.Version=$(VERSION)-$(GITCOMMIT) -X main.Branch=$(GITBRANCH) -X main.Time=$(TIMESTAMP)"  -o out/wlagent main.go

installer: wlagent
	mkdir -p out/wla
	cp dist/linux/install.sh out/wla/install.sh && chmod +x out/wla/install.sh
	cp dist/linux/workload-agent.service out/wla/workload-agent.service
	cp libvirt/qemu out/wla/qemu && chmod +x out/wla/qemu
	cp out/wlagent out/wla/wlagent && chmod +x out/wla/wlagent
	chmod +x dist/linux/build-container-security-dependencies.sh
	dist/linux/build-container-security-dependencies.sh
	mv secure_docker_daemon out/
	mv out/secure_docker_daemon/dcg_security-container-encryption/daemon-output out/wla/
	mv  secure-docker-plugin out/
	cp -r out/secure-docker-plugin/secure-docker-plugin out/wla/
	cp dist/linux/daemon.json out/wla/
	cp -rf out/secure-docker-plugin/artifact out/wla/
	cp dist/linux/uninstall-container-security-dependencies.sh out/wla/uninstall-container-security-dependencies.sh && chmod +x out/wla/uninstall-container-security-dependencies.sh
	makeself out/wla out/workload-agent-$(VERSION).bin "Workload Agent $(VERSION)" ./install.sh 

all: installer

clean: 
	rm -rf out/
