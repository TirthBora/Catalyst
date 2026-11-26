package process

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"catalyst/internal/websocket"
)

// State represents the current lifecycle phase of a process.
type State string

const (
	StateRunning State = "running"
	StateStopped State = "stopped"
	StateFailed  State = "failed"
)

// LogLine represents a timestamped output line from stdout or stderr.
type LogLine struct {
	Stream    string    `json:"stream"` // "stdout" or "stderr"
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

// Process tracks a supervised operating system process.
type Process struct {
	ID        string    `json:"id"`
	Command   string    `json:"command"`
	Args      []string  `json:"args"`
	WorkDir   string    `json:"workDir"`
	State     State     `json:"state"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"startedAt"`
	ExitCode  int       `json:"exitCode"`
	Logs      []LogLine `json:"logs"`

	cmd       *exec.Cmd
	stdin     io.WriteCloser
	cancel    context.CancelFunc
	mu        sync.RWMutex
}

// Manager supervises developer processes and streams output.
type Manager struct {
	hub       *websocket.Hub
	processes map[string]*Process
	mu        sync.RWMutex
}

func NewManager(hub *websocket.Hub) *Manager {
	return &Manager{
		hub:       hub,
		processes: make(map[string]*Process),
	}
}

// Start launches a new child process supervised by the manager.
func (m *Manager) Start(id, command string, args []string, workDir string) (*Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, exists := m.processes[id]; exists {
		existing.mu.RLock()
		if existing.State == StateRunning {
			existing.mu.RUnlock()
			return nil, fmt.Errorf("process %q is already running (PID %d)", id, existing.PID)
		}
		existing.mu.RUnlock()
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = workDir

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdinPipe.Close()
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		_ = stdinPipe.Close()
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdinPipe.Close()
		cancel()
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	p := &Process{
		ID:        id,
		Command:   command,
		Args:      args,
		WorkDir:   workDir,
		State:     StateRunning,
		PID:       cmd.Process.Pid,
		StartedAt: time.Now().UTC(),
		Logs:      make([]LogLine, 0, 100),
		cmd:       cmd,
		stdin:     stdinPipe,
		cancel:    cancel,
	}
	m.processes[id] = p

	if m.hub != nil {
		m.hub.Broadcast("PROCESS_STARTED", map[string]interface{}{
			"id":      p.ID,
			"pid":     p.PID,
			"command": p.Command,
		})
	}

	// Capture stdout
	go m.streamOutput(p, "stdout", stdoutPipe)
	// Capture stderr
	go m.streamOutput(p, "stderr", stderrPipe)

	// Wait for process exit in background
	go m.waitProcess(p)

	return p, nil
}

func (m *Manager) streamOutput(p *Process, stream string, r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		line := LogLine{
			Stream:    stream,
			Text:      text,
			Timestamp: time.Now().UTC(),
		}

		p.mu.Lock()
		p.Logs = append(p.Logs, line)
		if len(p.Logs) > 500 {
			p.Logs = p.Logs[len(p.Logs)-500:] // Keep last 500 lines buffer
		}
		p.mu.Unlock()

		if m.hub != nil {
			m.hub.Broadcast("PROCESS_OUTPUT", map[string]interface{}{
				"id":        p.ID,
				"stream":    stream,
				"text":      text,
				"timestamp": line.Timestamp.Format("15:04:05"),
			})
		}
	}
}

func (m *Manager) waitProcess(p *Process) {
	err := p.cmd.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		p.ExitCode = exitErr.ExitCode()
		p.State = StateFailed
	} else if err != nil {
		p.State = StateFailed
		p.ExitCode = -1
	} else {
		p.State = StateStopped
		p.ExitCode = 0
	}

	if m.hub != nil {
		m.hub.Broadcast("PROCESS_STOPPED", map[string]interface{}{
			"id":       p.ID,
			"state":    p.State,
			"exitCode": p.ExitCode,
		})
	}
}

// WriteInput writes input string followed by newline to the process's standard input.
func (p *Process) WriteInput(text string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.State != StateRunning || p.stdin == nil {
		return fmt.Errorf("process is not running")
	}

	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	_, err := p.stdin.Write([]byte(text))
	return err
}

// WriteInput sends interactive text input to a running process standard input pipe.
func (m *Manager) WriteInput(id, text string) error {
	m.mu.RLock()
	p, ok := m.processes[id]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("process %q not found", id)
	}
	return p.WriteInput(text)
}

// Stop gracefully terminates a running process.
func (m *Manager) Stop(id string) error {
	m.mu.RLock()
	p, exists := m.processes[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process %q not found", id)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.State != StateRunning {
		return nil
	}

	// Trigger context cancellation
	p.cancel()

	if p.stdin != nil {
		_ = p.stdin.Close()
	}

	// Platform-aware process termination
	if runtime.GOOS == "windows" && p.cmd != nil && p.cmd.Process != nil {
		// On Windows, taskkill /F /T terminates child process tree to prevent orphan processes
		_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(p.cmd.Process.Pid)).Run()
	} else if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}

	p.State = StateStopped
	return nil
}

// Restart stops and starts a process again.
func (m *Manager) Restart(id string) (*Process, error) {
	m.mu.RLock()
	p, exists := m.processes[id]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("process %q not found", id)
	}

	cmd := p.Command
	args := p.Args
	dir := p.WorkDir

	_ = m.Stop(id)
	time.Sleep(100 * time.Millisecond)

	return m.Start(id, cmd, args, dir)
}

// List returns all currently managed processes.
func (m *Manager) List() []*Process {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*Process, 0, len(m.processes))
	for _, p := range m.processes {
		list = append(list, p)
	}
	return list
}

// Get returns a specific process by ID.
func (m *Manager) Get(id string) (*Process, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.processes[id]
	return p, ok
}

// StopAll terminates all running processes.
func (m *Manager) StopAll() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.processes))
	for id, p := range m.processes {
		p.mu.RLock()
		if p.State == StateRunning {
			ids = append(ids, id)
		}
		p.mu.RUnlock()
	}
	m.mu.RUnlock()

	for _, id := range ids {
		_ = m.Stop(id)
	}
}
