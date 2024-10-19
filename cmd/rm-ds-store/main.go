package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/mdw-go/must/must"
)

var Version = "dev"

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	flags := flag.NewFlagSet(fmt.Sprintf("%s @ %s", filepath.Base(os.Args[0]), Version), flag.ExitOnError)
	flags.Usage = func() {
		_, _ = fmt.Fprintf(flags.Output(), "Usage of %s:\n", flags.Name())
		_, _ = fmt.Fprintln(flags.Output(), ""+
			"Upon startup, the program will delete any .DS_Store files it finds below $CODEPATH. "+
			"Thereafter, it watches for the creation of new .DS_Store files (and delete them too). "+
			"The program runs until it receives SIGINT, SIGTERM, or SIGKILL. You're welcome!",
		)
		flags.PrintDefaults()
	}
	_ = flags.Parse(os.Args[1:])

	watcher := must.Value(fsnotify.NewWatcher())
	defer must.Defer(watcher.Close)()

	root := filepath.Join(os.Getenv("CODEPATH"), "src")
	err := fs.WalkDir(os.DirFS(root), ".", func(path string, d fs.DirEntry, err error) error {
		path = filepath.Join(root, path)
		if d.Type().IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			must.Void(watcher.Add(path))
		}
		if d.Name() == EVIL {
			Delete(path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	log.Printf("Watching %d directories for creation of new .DS_Store files.", len(watcher.WatchList()))
	for {
		select {
		case event := <-watcher.Events:
			if filepath.Base(event.Name) == EVIL {
				Delete(event.Name)
			} else {
				stat, err := os.Stat(event.Name)
				if os.IsNotExist(err) {
					_ = watcher.Remove(event.Name)
				} else if err == nil && stat.IsDir() {
					_ = watcher.Add(event.Name)
				}
			}

		case sig := <-signals:
			log.Println(sig)
			return
		}
	}
}

func Delete(path string) {
	log.Println("EVIL:", path)
	_ = os.Remove(path)
}

const EVIL = ".DS_Store"
