package main

import (
	"fsnotify"
	"log"
	"time"
)

func Update(watchid string, oldfile string, newfile string, op string, filetype string) {
	log.Printf("watchid=%s, oldfile=%s, newfile=%s, op=%s, filetype=%s\n", watchid, oldfile, newfile, op, filetype)
}

func main() {
	fsnotify.AddWatcher("test", "D:/test", Update)
	fsnotify.AddWatcher("test2", "D:/test2", Update)
	fsnotify.DelWatcher("test2")

	for {
		time.Sleep(time.Duration(5) * time.Second)
	}
}
