// tempcleaner.go
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Cleaner struct {
	Config string
	io.ReadCloser
	*sync.WaitGroup
	log             io.WriteCloser
	runConcurrently bool
}

type ConfigEntry struct {
	Dir   string
	Ndays int
	*Cleaner
}

func NewConfigEntry(dir string, ndays int, cleaner *Cleaner) *ConfigEntry {
	return &ConfigEntry{dir, ndays, cleaner}
}

func NewCleaner(config string) *Cleaner {
	return &Cleaner{Config: config, WaitGroup: new(sync.WaitGroup)}

}

func (cleaner *Cleaner) clean() {
	var err error
	cleaner.ReadCloser, err = os.Open(cleaner.Config)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer cleaner.Close()
	for {
		entry := cleaner.ReadExpireEntry()
		if entry != nil {
			if cleaner.runConcurrently {
				go entry.process()
			} else {
				entry.process()
			}
		} else {
			break
		}
	}
	cleaner.Wait()
}

func (entry *ConfigEntry) process() {
	entry.Add(1)
	processDir(entry.Dir, entry.Ndays*86400)
	entry.Done()
}

func (cleaner *Cleaner) ReadExpireEntry() *ConfigEntry {
	var dir string
	var ndays int

	n, err := fmt.Fscanf(cleaner, "%s\t%d\n", &dir, &ndays)
	if err != nil || n != 2 {
		return nil
	}
	return NewConfigEntry(dir, ndays, cleaner)
}

func isFileExpiried(fileInfo os.FileInfo, nseconds int) bool {
	now := time.Now()
	diff := now.Sub(fileInfo.ModTime())
	return diff.Seconds() > float64(nseconds)
}

func processDir(dirName string, nseconds int) {
	dir, err := os.Open(dirName)
	if err != nil {
		fmt.Println(err)
		return
	}

	infos, err := dir.Readdir(0)
	dir.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, info := range infos {
		expiried := isFileExpiried(info, nseconds)
		absname := filepath.Join(dirName, info.Name())
		if expiried {
			defer func() {
				fmt.Println("Deleted", absname)
				if err := os.Remove(absname); err != nil {
					fmt.Println(err)
				}
			}()
		}
		if info.IsDir() {
			processDir(absname, nseconds)
		}
	}
}

var ConfigFile string = *flag.String("config", "tempcleaner.conf", "config file")
var RunConcurrently bool = *flag.Bool("concurrently", true, "process entries concurrently")

func main() {
	start := time.Now()
	flag.Parse()
	cleaner := NewCleaner(ConfigFile)
	cleaner.runConcurrently = RunConcurrently
	cleaner.clean()
	elapsed := time.Now().Sub(start)
	fmt.Println("Elapsed", elapsed)
}
