#!/bin/bash

# this file is to automate the steps that will be performed by buildsever
# on the local machine to make an installer

# to reuse the script, modify , version, name file and path to copy etc

# create a workspace directory under ~/.tmp using version and name
# Start from the build directory.. traverse to main binary directory and build
# make a zip file with contents of bin
# copy all files 
# call the makebin-auto.sh file 

VERSION=0.1
COMPONENTNAME=workloadagent
COMPONENT=$COMPONENTNAME-$VERSION

WORKSPACEDIR=~/.tmp/workspace/$COMPONENTNAME-$VERSION
BUILDOUTDIR=$WORKSPACEDIR/buildout
# make a clean workspace 

rm -rf $WORKSPACEDIR
mkdir -p $WORKSPACEDIR
ls -l $WORKSPACEDIR


# move to the binary directory temporarily to build the binary
cd ../wlagent
go build -o $BUILDOUTDIR/bin/wlagent

# move back to working directory
cd -
# do any other builds and store in the bin directory
tar -cvzf $WORKSPACEDIR/$COMPONENT.zip -C $BUILDOUTDIR .
tar -tvf $WORKSPACEDIR/$COMPONENT.zip

#delete
rm -rf $BUILDOUTDIR

cp ../files/* $WORKSPACEDIR
cp ../etc/* $WORKSPACEDIR

. ./makebin-auto.sh $WORKSPACEDIR

rm -rf $WORKSPACEDIR



