package main

import (
	"github.com/mgutz/ansi"
	"golang.org/x/exp/inotify"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	green  = ansi.ColorFunc("green+h:black")
	yellow = ansi.ColorFunc("yellow+h:black")
	red    = ansi.ColorFunc("red+h:black")
)

func main() {
	restart := make(chan bool)

	go func() {
		for {
			cmd := exec.Command("go", "build", "-o", "tmp-out")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			log.Println(yellow("running go build"))
			err := cmd.Run()
			if err != nil {
				<-restart
				continue
			}

			cmd = exec.Command("./tmp-out", os.Args[1:len(os.Args)]...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			log.Println(green("running compiled go program"))
			cmd.Start()

			<-restart

			log.Println(red("killing process"))
			if err := cmd.Process.Kill(); err != nil {
				log.Fatal(err)
			}
			cmd.Wait()
		}
	}()

	watcher, err := inotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	if err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if err := watcher.Watch(path); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case ev := <-watcher.Event:
			if ev.Mask&inotify.IN_CLOSE_WRITE == inotify.IN_CLOSE_WRITE &&
				strings.HasSuffix(ev.Name, ".go") {
				log.Println("event:", ev)
				restart <- true
			}
		case err := <-watcher.Error:
			log.Println("error:", err)
		}
	}
}
