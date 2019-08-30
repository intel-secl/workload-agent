/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package filewatch

import (
	"fmt"
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher encapsulates fsnotify.Watcher for easier functionality with callbacks
type Watcher struct {
	*fsnotify.Watcher
	mtx      *sync.Mutex
	handlers map[string]func(fsnotify.Event)
}

// NewWatcher creates a new Watcher object
func NewWatcher() (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		Watcher:  watcher,
		mtx:      &sync.Mutex{},
		handlers: make(map[string]func(fsnotify.Event)),
	}, nil
}

// HandleEvent executes a delegate handler function when the specified file is modified on the file system
// The delegate handler is only executed if the current watcher object is watching with Watch()
// HandleDelete is thread safe, protected by a sync.Mutex
func (w *Watcher) HandleEvent(file string, handler func(event fsnotify.Event)) error {
	err := w.Add(file)
	if err != nil {
		return err
	}
	w.mtx.Lock()
	w.handlers[file] = handler
	w.mtx.Unlock()
	return nil
}

// UnhandleEvent unregisters event handler
func (w *Watcher) UnhandleEvent(file string) {
	w.mtx.Lock()
	delete(w.handlers, file)
	w.mtx.Unlock()
}

// Watch will begin watching of file system events in a blocking loop
// Any registered event handlers will be executed
func (w *Watcher) Watch() {
	for {
		select {
		case event, ok := <-w.Events:
			if !ok {
				return
			}
			w.mtx.Lock()
			if h, exists := w.handlers[event.Name]; exists {
				h(event)
			}
			w.mtx.Unlock()
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}

func ExampleWatcherUsage() {
	w, _ := NewWatcher()
	w.HandleEvent("/home/user/foobar.txt", func(event fsnotify.Event) {
		affectedFile := event.Name
		if event.Op&fsnotify.Remove == fsnotify.Remove {
			fmt.Printf("File %s was deleted!\n", affectedFile)
		} else {
			fmt.Printf("File %s was modified, created, ...\n", affectedFile)
		}
	})
	go w.Watch()
}
