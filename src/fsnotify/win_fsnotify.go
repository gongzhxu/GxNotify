// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

// Package fsnotify provides a platform-independent interface for file system notifications.
package fsnotify

import (
	"bytes"
	"errors"
	"fmt"
	"log"
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

func (op Op) String() string {
	// Use a buffer for efficient string concatenation
	var buffer bytes.Buffer

	if op&Create == Create {
		buffer.WriteString("|CREATE")
	}
	if op&Remove == Remove {
		buffer.WriteString("|REMOVE")
	}
	if op&Write == Write {
		buffer.WriteString("|WRITE")
	}
	if op&Rename == Rename {
		buffer.WriteString("|RENAME")
	}
	if op&Chmod == Chmod {
		buffer.WriteString("|CHMOD")
	}
	if buffer.Len() == 0 {
		return ""
	}
	return buffer.String()[1:] // Strip leading pipe
}

// String returns a string representation of the event in the form
// "file: REMOVE|WRITE|..."
func (e Event) String() string {
	return fmt.Sprintf("%q: %s", e.Name, e.Op.String())
}

// Common errors that can be reported by a watcher
var ErrEventOverflow = errors.New("fsnotify queue overflow")

type Callback func(watchid string, oldfile string, newfile string, operator string, fileType string)

var watcherMutex sync.Mutex
var watcherMap = make(map[string]*Watcher)

func AddWatcher(watchid string, path string, cb Callback) {
	var watcher *Watcher
	var err error

	watcherMutex.Lock()
	_, ok := watcherMap[watchid]
	if ok {
		return
	} else {
		watcher, err = NewWatcher()
		if err != nil {
			return
		}

		watcherMap[watchid] = watcher
	}
	watcherMutex.Unlock()

	go func() {
		var lastevent Event
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op != Rename {
					if lastevent.Op == Rename && event.Op == Create {
						cb(watchid, lastevent.Name, event.Name, lastevent.Op.String(), "")
					} else {
						cb(watchid, event.Name, "", event.Op.String(), "")
					}
				}

				lastevent = event
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
	}
}

func DelWatcher(watchid string) {
	watcherMutex.Lock()
	defer watcherMutex.Unlock()
	watcher, ok := watcherMap[watchid]
	if ok {
		watcher.Close()
	}
}
