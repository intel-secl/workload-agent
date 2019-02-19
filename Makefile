VERSION := v0.0.0
GITCOMMIT := $(shell git describe --always)
GITBRANCH := $(shell git rev-parse --abbrev-ref HEAD)
TIMESTAMP := $(shell date --iso=seconds)

.PHONY: wlagent, installer, all, clean

wlagent:
	env GOOS=linux go build -ldflags "-X main.Version=$(VERSION)-$(GITCOMMIT) -X main.Branch=$(GITBRANCH) -X main.Time=$(TIMESTAMP)"  -o out/wlagent

installer: wlagent
	mkdir -p out/wla
	cp dist/linux/install.sh out/wla/install.sh && chmod +x out/wla/install.sh
	cp libvirt/qemu out/wla/qemu && chmod +x out/wla/qemu
	cp out/wlagent out/wla/wlagent && chmod +x out/wla/wlagent
	cp out/wlagentd out/wla/wlagentd && chmod +x out/wla/wlagentd
	makeself out/wla out/workload-agent-$(VERSION).bin "Workload Agent $(VERSION)" ./install.sh 

all: installer

clean: 
	rm -rf out/
