package watcher

import (
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	fs     *fsnotify.Watcher
	Events chan string
}

func New() (*Watcher, error) {
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fs:     fs,
		Events: make(chan string),
	}

	go w.run()

	return w, nil
}

func (w *Watcher) run() {
	for {
		select {
		case event, ok := <-w.fs.Events:
			if !ok {
				close(w.Events)
				return
			}

			if filepath.Ext(event.Name) == ".go" {
				w.Events <- event.Name
			}

		case _, ok := <-w.fs.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) Watch(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		switch d.Name() {
		case ".git", "vendor":
			return filepath.SkipDir
		}

		if len(d.Name()) > 0 && d.Name()[0] == '.' {
			return filepath.SkipDir
		}

		return w.fs.Add(path)
	})
}

func (w *Watcher) Close() error {
	return w.fs.Close()
}
