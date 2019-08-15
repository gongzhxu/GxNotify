// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

// Package fsnotify provides a platform-independent interface for file system notifications.
package fsnotify

import (
	"log"
	"os"
	"sync"
)

// Event represents a single file system notification.
type Event struct {
	Name string // Relative path to the file or directory.
	Op   Op     // File operation that triggered the event.
}

// Op describes a set of file operations.
type Op uint32

// These are the generalized file operations that can trigger a notification.
const (
	Create Op = 1 << iota
	Write
	Remove
	Rename
	Chmod
)

var noteDescription = map[Op]string{
	Create: "create",
	Remove: "delete",
	Rename: "rename",
	Write:  "change",
}

type Callback func(watchid string, oldfile string, newfile string, op string, filetype string)

var watcherMutex sync.Mutex
var watcherMap = make(map[string]*Watcher)

func AddWatcher(watchid string, path string, cb Callback) {
	watcherMutex.Lock()
	defer watcherMutex.Unlock()

	_, ok := watcherMap[watchid]
	if ok {
		return
	}

	watcher, err := NewWatcher()
	if err != nil {
		return
	}

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
		return
	}

	watcherMap[watchid] = watcher

	go func() {
		var lastevent Event
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				filetype := getFileType(event.Name)

				if event.Op&Rename != Rename {
					if lastevent.Op&Rename == Rename && event.Op&Create == Create {
						cb(watchid, lastevent.Name, event.Name, noteDescription[Rename], filetype)
					} else {
						if lastevent.Op&Create == Create {
							cb(watchid, event.Name, "", noteDescription[Create], filetype)
						}

						if lastevent.Op&Write == Write {
							cb(watchid, event.Name, "", noteDescription[Write], filetype)
						}

						if lastevent.Op&Remove == Remove {
							cb(watchid, event.Name, "", noteDescription[Remove], filetype)
						}
					}
				}

				lastevent = event
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()
}

func DelWatcher(watchid string) {
	watcherMutex.Lock()
	defer watcherMutex.Unlock()
	watcher, ok := watcherMap[watchid]
	if ok {
		watcher.Close()
		delete(watcherMap, watchid)
	}
}

func getFileType(path string) string {
	f, err := os.Stat(path)
	if err != nil {
		return "0"
	}

	if f.IsDir() {
		return "1"
	}

	return "0"
}
