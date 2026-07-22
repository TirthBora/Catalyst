package dev

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	manager := process.New()

	cmd := runner.Command(proj)
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

			timer.Reset(200 * time.Millisecond)

		case <-timer.C:
			if pending {
				pending = false

				cmd := runner.Command(proj)

				if err := manager.Restart(cmd); err != nil {
					fmt.Println(err)
				}
			}

		case <-sig:
			return manager.Stop()
		}
	}
}
