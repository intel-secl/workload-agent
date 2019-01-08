GITTAG := $(shell git describe --tags --abbrev=0 2> /dev/null)
GITCOMMIT := $(shell git describe --always)
GITCOMMITDATE := $(shell git log -1 --date=short --pretty=format:%cd)
VERSION := $(or ${GITTAG}, v0.0.0)

wlagent:
	env GOOS=linux go build -ldflags "-X intel/isecl/wlagent/version.Version=$(VERSION)-$(GITCOMMIT)" -o out/wlagent

installer: wlagent
	mkdir -p out/wla
	cp dist/linux/install.sh out/wla/install.sh && chmod +x out/wla/install.sh
	cp libvirt/hook.sh out/wla/qemu
	cp out/wlagent out/wla/wlagent
	makeself out/wla out/wla-$(VERSION).bin "Workload Agent $(VERSION)" ./install.sh 

all: installer

clean: 
	rm -rf out/





















# # Makefile is written to be generic and be adapted to different components
# # should only have the modify the top section. 
# COMPONENT ?=workload-agent
# GITTAG := $(shell git describe --tags --abbrev=0 2> /dev/null)
# GITCOMMIT := $(shell git describe --always)
# GITCOMMITDATE := $(shell git log -1 --date=short --pretty=format:%cd)
# VERSION := $(or ${GITTAG}, v0.0.0)
# BUILDTYPE ?=dev
# BUILDID ?= $(GITCOMMIT)
# BIN_DIR_NAME ?=bin
# BUILDOUT_OS_NAME ?=linux
# BINARY_NAME ?=wlagent
# SOURCE_DIR = src
# OUT_DIR = out
# INSTALL_SCRIPT = ./install.sh

# MKDIR_FORCE = mkdir -p

# WORKSPACE ?= $(OUT_DIR)/$(BUILDOUT_OS_NAME)
# BUILDDIR = $(WORKSPACE)/$(COMPONENT)-$(VERSION)
# BUILD_BINARCHIVE_DIR = $(BUILDDIR)-binarchive
# BUILD_BINDIR = $(BUILD_BINARCHIVE_DIR)/$(BIN_DIR_NAME)
# BINARCHIVE_FILE = $(BUILDDIR)/$(COMPONENT)-$(VERSION).zip
# MAKESELF_ARCHIVE_FILE = $(BUILDDIR).bin



# # Section : Building Go code 
# GOBUILD = go build
# BUILD_OPTIONS_LDFLAG = -ldflags '-X "main.component=${COMPONENT}" -X "main.version=${VERSION}" \
# 				-X "main.buildid=${BUILDID}" -X "main.buildtype=${BUILDTYPE}"'
# #BUILD_OPTIONS_LDFLAG = -ldflags '-X "main.component=${COMPONENT}"' 
# BINARY_TARGET = $(BUILD_BINDIR)/$(BINARY_NAME)
# BUILD_OPTIONS_OUTPUTPATH = -o $(abspath $(BINARY_TARGET))

# # recursive version of wildcard function. To match files in subdirectories
# rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard, $d/,$2))

# GOSOURCE := $(call rwildcard,src/,*.go)

# .PHONY: build install clean directories

# # install is the default. Makes a self extracting installer. 
# install: $(MAKESELF_ARCHIVE_FILE)

# #cleans out the directory
# clean:
# 	rm -rf $(BUILDDIR)
# 	rm -rf $(BUILD_BINARCHIVE_DIR)
# 	rm -rf $(OUT_DIR)
# 	rm -rf ./$(BINARY_NAME)


# fresh: clean install

# # Make directories that does not exist
# directories: ${BUILDDIR} ${BUILD_BINDIR}

# #builds the binary
# build: $(BINARY_TARGET)

# wlagent: $(BINARY_TARGET)
# 	cp $(BINARY_TARGET) .

# ${BUILDDIR}:
# 	${MKDIR_FORCE} ${BUILDDIR}

# # Section for Copying files. We are assembling set of files from different
# # directories and having rules to copy them. Will need to add or remove 
# # directories to this list 
# DEST_COPIED_FILES = $(patsubst dist/linux/%,$(BUILDDIR)/%,$(wildcard dist/linux/*))
# DEST_COPIED_FILES += $(patsubst libvirt/%,$(BUILDDIR)/%,$(wildcard libvirt/*))
# DEST_COPIED_FILES += $(patsubst common/bash/%,$(BUILDDIR)/%,$(wildcard common/bash/*))

# # rules to copy files to destination from different directories. Not sure of there is a way
# # to make this into a single rule. 
# $(BUILDDIR)/%:  dist/linux/% | $(BUILDDIR)
# 	cp $< $@
# $(BUILDDIR)/%:  libvirt/% | $(BUILDDIR)
# 	cp $< $@
# $(BUILDDIR)/%:  common/bash/% | $(BUILDDIR)
# 	cp $< $@
# $(BUILDDIR)/%:  % | $(BUILDDIR)
# 	cp $< $@
	

# $(BUILD_BINDIR)	:
# 	$(MKDIR_FORCE) $(BUILD_BINDIR)

# GOBUILD_COMMAND = $(GOBUILD) $(BUILD_OPTIONS_LDFLAG) $(BUILD_OPTIONS_OUTPUTPATH)

# $(BINARY_TARGET): $(GOSOURCE) | $(BUILD_BINDIR)
# 	@echo Building $(BINARY_NAME)
# 	@echo List of source files found $(GOSOURCE)
# 	cd $(SOURCE_DIR); \
# 	$(GOBUILD_COMMAND)


# # Create a zip file that contains binaries that need to be installed into the target system. 
# $(BINARCHIVE_FILE): $(BINARY_TARGET) | $(BUILDDIR) 
# 	tar -cvzf $(BINARCHIVE_FILE) -C $(BUILD_BINARCHIVE_DIR) .

# MAKESELF = $(shell which makeself)
# $(MAKESELF_ARCHIVE_FILE): $(DEST_COPIED_FILES) $(BINARCHIVE_FILE) 
# 	$(MKDIR_FORCE) .tmp
# 	export TMPDIR=.tmp
# 	-chmod +x $(BUILDDIR)/*.sh $(BUILDDIR)/*.bin ||:
# 	$(MAKESELF) --follow --nocomp $(BUILDDIR) $(MAKESELF_ARCHIVE_FILE) $(COMPONENT)-$(VERSION) $(INSTALL_SCRIPT)
# 	rm -rf .tmp
	