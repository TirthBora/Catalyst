package watcher

import (
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
	return w.fs.Add(root)
}

func (w *Watcher) Close() error {
	return w.fs.Close()
}
