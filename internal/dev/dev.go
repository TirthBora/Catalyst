package dev

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TirthBora/catalyst/internal/browser"
	"github.com/TirthBora/catalyst/internal/process"
	"github.com/TirthBora/catalyst/internal/project"
	"github.com/TirthBora/catalyst/internal/runner"
	"github.com/TirthBora/catalyst/internal/watcher"
)

func Run() error {
	proj, err := project.Detect()
	if err != nil {
		return fmt.Errorf("detect project: %w", err)
	}

	cmd := runner.Command(proj)

	manager := process.New()
	if err := manager.Start(cmd); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	// Give the application a moment to start before opening the browser.
	time.Sleep(500 * time.Millisecond)

	// Ignore browser errors in Version 1.
	_ = browser.Open("http://localhost:8080")
	reloader := browser.NewReloader()

	if err := reloader.Start(":35729"); err != nil {
		return fmt.Errorf("start reload serever :%w", err)
	}

	w, err := watcher.New()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer w.Close()

	if err := w.Watch(proj.Root); err != nil {
		return fmt.Errorf("watch project: %w", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	var (
		pending bool
		timer   = time.NewTimer(time.Hour)
	)

	timer.Stop()

	for {
		select {
		case file := <-w.Events:
			fmt.Println("Changed:", file)

			pending = true

			// Safely reset the debounce timer.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}

			timer.Reset(200 * time.Millisecond)

		case <-timer.C:
			if !pending {
				continue
			}

			pending = false

			fmt.Println("Restarting...")

			cmd := runner.Command(proj)

			if err := manager.Restart(cmd); err != nil {
				fmt.Println("Restart failed:", err)
				continue
			}
			reloader.Notify()

		case <-sig:
			fmt.Println("\nShutting down Catalyst...")
			return manager.Stop()
		}
	}
}
