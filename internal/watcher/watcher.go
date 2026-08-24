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

func (w *Watcher) Start() error {
	// Walk the directory and watch everything
	err := filepath.Walk(w.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			// Skip heavy/system directories to save memory
			if name == ".git" || name == "node_modules" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
			return w.fsWatcher.Add(path)
		}
		return nil
	})
	if err != nil {
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
			// Only care about writes or creates
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				w.debounceEmit(event.Name)
			}
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v\n", err)
		}
	}
}

func (w *Watcher) debounceEmit(filename string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Ignore temporary files created by code editors
	if strings.HasSuffix(filename, "~") || strings.HasSuffix(filename, ".tmp") {
		return
	}

	if timer, exists := w.timers[filename]; exists {
		timer.Stop()
	}

	// Wait 100ms before broadcasting to avoid spamming the WebSocket
	w.timers[filename] = time.AfterFunc(100*time.Millisecond, func() {
		relPath, _ := filepath.Rel(w.rootPath, filename)
		log.Printf(" File Changed: %s\n", relPath)
		w.hub.Broadcast("FILE_CHANGED", map[string]string{
			"file":      relPath,
			"timestamp": time.Now().Format("15:04:05"),
		})
	})
}

func (w *Watcher) Close() {
	w.fsWatcher.Close()
}
