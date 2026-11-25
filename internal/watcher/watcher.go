package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"catalyst/internal/websocket"

	"github.com/fsnotify/fsnotify"
)

// ShouldIgnore reports whether a directory name should be excluded from watching.
func ShouldIgnore(name string) bool {
	switch name {
	case ".git", "node_modules", "dist", "build", ".vscode", ".idea", "vendor", ".venv", "__pycache__":
		return true
	default:
		return false
	}
}

type Watcher struct {
	fsWatcher *fsnotify.Watcher
	hub       *websocket.Hub
	rootPath  string
	mu        sync.Mutex
	timers    map[string]*time.Timer
}

func NewWatcher(rootPath string, hub *websocket.Hub) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		fsWatcher: fsw,
		hub:       hub,
		rootPath:  rootPath,
		timers:    make(map[string]*time.Timer),
	}, nil
}

// watchDirRecursively adds dir and all its non-ignored subdirectories to the watcher.
func (w *Watcher) watchDirRecursively(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if ShouldIgnore(info.Name()) {
				return filepath.SkipDir
			}
			return w.fsWatcher.Add(path)
		}
		return nil
	})
}

func (w *Watcher) Start() error {
	if err := w.watchDirRecursively(w.rootPath); err != nil {
		return err
	}

	go w.eventLoop()
	return nil
}

func (w *Watcher) eventLoop() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// If a new directory was created, dynamically add it to the watcher.
			if event.Op&fsnotify.Create != 0 {
				if fi, err := os.Stat(event.Name); err == nil && fi.IsDir() {
					if !ShouldIgnore(fi.Name()) {
						_ = w.watchDirRecursively(event.Name)
					}
					w.debounceEmit(event.Name, "create")
					continue
				}
			}

			var action string
			switch {
			case event.Op&fsnotify.Write != 0:
				action = "write"
			case event.Op&fsnotify.Create != 0:
				action = "create"
			case event.Op&fsnotify.Remove != 0:
				action = "delete"
			case event.Op&fsnotify.Rename != 0:
				action = "rename"
			default:
				continue
			}

			w.debounceEmit(event.Name, action)

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v\n", err)
		}
	}
}

func (w *Watcher) debounceEmit(filename, action string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Ignore temporary files created by code editors
	base := filepath.Base(filename)
	if strings.HasSuffix(filename, "~") || strings.HasSuffix(filename, ".tmp") || strings.HasPrefix(base, ".#") {
		return
	}

	if timer, exists := w.timers[filename]; exists {
		timer.Stop()
	}

	// Wait 100ms before broadcasting to avoid spamming the WebSocket
	w.timers[filename] = time.AfterFunc(100*time.Millisecond, func() {
		w.mu.Lock()
		delete(w.timers, filename)
		w.mu.Unlock()

		relPath, err := filepath.Rel(w.rootPath, filename)
		if err != nil {
			relPath = filename
		}
		log.Printf(" File Changed [%s]: %s\n", action, relPath)
		w.hub.Broadcast("FILE_CHANGED", map[string]string{
			"file":      relPath,
			"action":    action,
			"timestamp": time.Now().Format("15:04:05"),
		})
	})
}

func (w *Watcher) Close() {
	w.mu.Lock()
	for k, t := range w.timers {
		t.Stop()
		delete(w.timers, k)
	}
	w.mu.Unlock()
	w.fsWatcher.Close()
}
