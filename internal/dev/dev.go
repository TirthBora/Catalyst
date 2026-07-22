package dev

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/TirthBora/catalyst/internal/process"
	"github.com/TirthBora/catalyst/internal/project"
	"github.com/TirthBora/catalyst/internal/runner"
)

func Run() error {
	proj, err := project.Detect()
	if err != nil {
		return err
	}

	cmd := runner.New(proj)

	manager := process.New()

	if err := manager.Start(cmd); err != nil {
		return err
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	<-sig

	return manager.Stop()
}
