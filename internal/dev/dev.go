package dev

import (
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
	select {}
}
