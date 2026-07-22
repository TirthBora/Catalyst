package dev

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/TirthBora/catalyst/internal/process"
	"github.com/TirthBora/catalyst/internal/project"
	"github.com/TirthBora/catalyst/internal/runner"
	"github.com/TirthBora/catalyst/internal/watcher"
)

func Run() error {
	proj, err := project.Detect()
	if err != nil {
		return err
	}

	cmd := runner.Command(proj)

	manager := process.New()

	if err := manager.Start(cmd); err != nil {
		return err
	}
	w, err := watcher.New()
	if err != nil {
		return err
	}
	defer w.Close()

	if err := w.Watch(proj.Root); err != nil {
		return err
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	for {
		select {
		case file := <-w.Events:
			fmt.Println("Changed:", file)

		case <-sig:
			return manager.Stop()
		}
	}
}
