GITTAG := $(shell git describe --tags --abbrev=0 2> /dev/null)
GITCOMMIT := $(shell git describe --always)
GITCOMMITDATE := $(shell git log -1 --date=short --pretty=format:%cd)
VERSION := $(or ${GITTAG}, v0.0.0)

wlagent:
	env GOOS=linux go build -ldflags "-X intel/isecl/wlagent/version.Version=$(VERSION)-$(GITCOMMIT)" -o out/wlagent

installer: wlagent
	mkdir -p out/wla
	cp dist/linux/install.sh out/wla/install.sh && chmod +x out/wla/install.sh
	cp libvirt/qemu out/wla/qemu
	cp out/wlagent out/wla/wlagent
	makeself out/wla out/workload-agent-$(VERSION).bin "Workload Agent $(VERSION)" ./install.sh 

all: installer

clean: 
	rm -rf out/
