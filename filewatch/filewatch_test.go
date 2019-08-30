/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package filewatch

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestFileWatch(t *testing.T) {
	f, _ := ioutil.TempFile("", "foobar*.txt")
	name := f.Name()
	f.Close()
	w, _ := NewWatcher()
	go w.Watch()
	notified := make(chan bool)
	timeout := make(chan bool)
	go func() {
		time.Sleep(2 * time.Second)
		timeout <- true
	}()
	w.HandleEvent(name, func(event fsnotify.Event) {
		notified <- true
	})
	os.Remove(name)
	select {
	case <-notified:
	case <-timeout:
		t.Fail()
	}
}

func TestFileWatchOverrideHandler(t *testing.T) {
	f, _ := ioutil.TempFile("", "foobar*.txt")
	name := f.Name()
	f.Close()
	w, _ := NewWatcher()
	go w.Watch()
	stale := make(chan bool)
	notified := make(chan bool)
	timeout := make(chan bool)
	go func() {
		time.Sleep(2 * time.Second)
		timeout <- true
	}()
	w.HandleEvent(name, func(event fsnotify.Event) {
		stale <- true
	})
	w.HandleEvent(name, func(event fsnotify.Event) {
		notified <- true
	})
	os.Remove(name)
	select {
	case <-notified:
	case <-stale:
		t.Fail()
	case <-timeout:
		t.Fail()
	}
}

func TestFileWatchDeleteHandler(t *testing.T) {
	f, _ := ioutil.TempFile("", "foobar*.txt")
	name := f.Name()
	f.Close()
	w, _ := NewWatcher()
	go w.Watch()
	notified := make(chan bool)
	timeout := make(chan bool)
	go func() {
		time.Sleep(100 * time.Millisecond)
		timeout <- true
	}()
	w.HandleEvent(name, func(event fsnotify.Event) {
		notified <- true
	})
	w.UnhandleEvent(name)
	os.Remove(name)
	select {
	case <-notified:
		t.Fail()
	case <-timeout:
	}
}
