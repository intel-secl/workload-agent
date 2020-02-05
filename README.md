# Workload Agent 

`Workload Agent` is used to launch encrypted workloads on a trusted host.

## Key features
- Start a guest VM
- Stop a quest VM
- Create VM/container trust report
- Fetch a flavor from Workload Service


## System Requirements
- RHEL 7.5/7.6
- Epel 7 Repo
- Proxy settings if applicable

## Software requirements
- git
- makeself
- `go` version >= `go1.11.4` & <= `go1.12.12`
- docker 18.06 or higher

# Step By Step Build Instructions

## Install required shell commands

### Install tools from `yum`
```shell
sudo yum install -y git wget makeself
```

### Install `go` version >= `go1.11.4` & <= `go1.12.12`
The `Workload Agent` requires Go version 1.11.4 that has support for `go modules`. The build was validated with the latest version 1.12.12 of `go`. It is recommended that you use 1.12.12 version of `go`. More recent versions may introduce compatibility issues. You can use the following to install `go`.
```shell
wget https://dl.google.com/go/go1.12.12.linux-amd64.tar.gz
tar -xzf go1.12.12.linux-amd64.tar.gz
sudo mv go /usr/local
export GOROOT=/usr/local/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
```

## Build Workload Agent(WLA)

- Git clone the WLA
- Run scripts to build the WLA

```shell
git clone https://github.com/intel-secl/workload-agent.git
cd workload-agent
make installer
```

# Third Party Dependencies

## WLA

### Direct dependencies

| Name                  | Repo URL                        | Minimum Version Required           |
| ----------------------| --------------------------------| :--------------------------------: |
| logrus                | github.com/sirupsen/logrus      | v1.4.2                             |
| testify               | github.com/stretchr/testify     | v1.3.0                             |
| yaml.v2               | gopkg.in/yaml.v2                | v2.2.2                             |
| fs notify             | github.com/fsnotify/fsnotify    | v1.4.7                             |


### Indirect Dependencies

| Repo URL                          | Minimum version required           |
| ----------------------------------| :--------------------------------: |
| github.com/Gurpartap/logrus-stack | v0.0.0-20170710170904-89c00d8a28f4 |
| github.com/facebookgo/stack       | v0.0.0-20160209184415-751773369052 |
| golang.org/x/net                  | v0.0.0-20190206173232-65e2d4e15006 |

*Note: All dependencies are listed in go.mod*

# Links
# Links
https://01.org/intel-secl/

*Intel® Security Libraries for Data Center v1.6 Release Update:
Due to a recent change in an externally supported repository, namely Docker github; customers may see issues in compiling the latest released version of Intel® SecL-DC v1.6. Intel is working on a resolution and we plan to provide a minor release to address this issue shortly. Regular update to this communication will be shared to customers accordingly.
