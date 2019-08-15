// +build darwin

package fsnotify

import (
	"log"
	"os"
	"sync"
	"time"
)

type Callback func(watchid string, oldfile string, newfile string, op string, filetype string)

var noteDescription = map[EventFlags]string{
	ItemCreated:  "create",
	ItemRemoved:  "delete",
	ItemRenamed:  "rename",
	ItemModified: "change",

	ItemIsFile:    "0",
	ItemIsDir:     "1",
	ItemIsSymlink: "0",
}

func NewEeventStream(path string) (*EventStream, error) {
	dev, err := DeviceForPath(path)
	if err != nil {
		log.Fatalf("Failed to retrieve device for path: %v", err)
		return nil, err
	}

	log.Println("Device ID:", dev)
	log.Println(EventIDForDeviceBeforeTime(dev, time.Now()))

	es := &EventStream{
		Paths:   []string{path},
		Latency: 0 * time.Millisecond,
		Device:  dev,
		Flags:   FileEvents | WatchRoot,
		Errors:  make(chan error)}
	return es, err
}

var watcherMutex sync.Mutex
var watcherMap = make(map[string]*EventStream)

func AddWatcher(watchid string, path string, cb Callback) {
	watcherMutex.Lock()
	defer watcherMutex.Unlock()

	_, ok := watcherMap[watchid]
	if ok {
		log.Println(watchid, " exists")
		return
	}

	es, err := NewEeventStream(path)
	if err != nil {
		log.Fatal("NewEeventStream err=", err)
		return
	}

	es.Start()
	watcherMap[watchid] = es

	go func() {
		for {
			select {
			case msg := <-es.Events:
				msglen := len(msg)
				for i := 0; i < msglen; i++ {
					if i+1 < msglen {
						if msg[i].ID+1 == msg[i+1].ID {
							handleEvent2(watchid, cb, msg[i], msg[i+1])
							i++
						} else {
							handleEvent(watchid, cb, msg[i])
						}

					} else {
						handleEvent(watchid, cb, msg[i])
					}
				}
			case err = <-es.Errors:
				if err != nil {
					return
				}
			}
		}

	}()
}

func DelWatcher(watchid string) {
	watcherMutex.Lock()
	defer watcherMutex.Unlock()
	es, ok := watcherMap[watchid]
	if ok {
		es.Stop()
		delete(watcherMap, watchid)
	}
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {

		return false, nil
	}

	return false, err
}

func handleEvent(watchid string, cb Callback, event Event) {

	var filetype string
	if event.Flags&ItemIsDir == ItemIsDir {
		filetype = noteDescription[ItemIsDir]
	} else {
		filetype = noteDescription[ItemIsFile]
	}

	event.Path = "/" + event.Path

	if event.Flags&ItemRenamed == ItemRenamed {
		bool, _ := pathExists(event.Path)
		if bool {
			cb(watchid, event.Path, "", noteDescription[ItemCreated], filetype)
		} else {
			cb(watchid, event.Path, "", noteDescription[ItemRemoved], filetype)
		}
	}

	if event.Flags&ItemCreated == ItemCreated {
		cb(watchid, event.Path, "", noteDescription[ItemCreated], filetype)
	}

	if event.Flags&ItemRemoved == ItemRemoved {
		cb(watchid, event.Path, "", noteDescription[ItemRemoved], filetype)
	}

	if event.Flags&ItemModified == ItemModified {
		cb(watchid, event.Path, "", noteDescription[ItemModified], filetype)
	}
}

func handleEvent2(watchid string, cb Callback, event1 Event, event2 Event) {
	if event1.Flags&ItemRenamed == ItemRenamed && event2.Flags&ItemRenamed == ItemRenamed {
		var filetype string
		if event1.Flags&ItemIsDir == ItemIsDir {
			filetype = noteDescription[ItemIsDir]
		} else {
			filetype = noteDescription[ItemIsFile]
		}

		event1.Path = "/" + event1.Path
		event2.Path = "/" + event2.Path
		cb(watchid, event1.Path, event2.Path, noteDescription[ItemRenamed], filetype)
	}
}
