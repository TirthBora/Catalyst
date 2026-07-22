package process

import (
	"fmt"
	"os"
	"os/exec"
)

type Manager struct {
	cmd *exec.Cmd
}

func New() *Manager {
	return &Manager{}
}

func (m *Manager) Start(cmd *exec.Cmd) error {
	if m.cmd != nil {
		return fmt.Errorf("process is already running")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return err
	}

	m.cmd = cmd

	go func() {
		_ = cmd.Wait()
		m.cmd = nil
	}()

	return nil
}

func (m *Manager) Stop() error {
	if m.cmd == nil {
		return nil
	}

	err := m.cmd.Process.Kill()
	if err != nil {
		return err
	}

	_ = m.cmd.Wait()
	m.cmd = nil

	return nil
}

func (m *Manager) Restart(cmd *exec.Cmd) error {
	if err := m.Stop(); err != nil {
		return err
	}

	return m.Start(cmd)
}
