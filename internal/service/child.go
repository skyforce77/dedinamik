package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"sync"
	"syscall"

	"github.com/skyforce77/dedinamik/internal/plugin"
)

type ChildPlugin struct {
	Name    string
	Command []string
	Home    string
	Freeze  bool
	Child   *os.Process
	mu      sync.Mutex
}

type ChildPluginFile struct {
	Command []string `json:"command"`
	Home    string   `json:"home"`
	Freeze  bool     `json:"freeze"`
}

func (p *ChildPlugin) GetName() string {
	return p.Name
}
func (p *ChildPlugin) GetCanSleep() bool {
	return false
}
func (p *ChildPlugin) Start() error {
	return startChild(p)
}
func (p *ChildPlugin) Stop() error {
	return stopChild(p)
}
func (p *ChildPlugin) Sleep() error {
	return p.Child.Signal(syscall.SIGSTOP)
}
func (p *ChildPlugin) WakeUp() error {
	return p.Child.Signal(syscall.SIGCONT)
}

func CreateChildPlugin(file *plugin.PluginFile) plugin.ServicePlugin {
	command := ChildPluginFile{}

	err := json.Unmarshal(file.Config, &command)
	if err != nil {
		return nil
	}

	return &ChildPlugin{
		Name:    file.Name,
		Command: command.Command,
		Home:    command.Home,
		Freeze:  command.Freeze,
	}
}

func startChild(p *ChildPlugin) error {
	stdin, err := os.Create(fmt.Sprintf("/tmp/%s-stdin.log", p.Name))
	if err != nil {
		return fmt.Errorf("failed to create stdin log for %s: %w", p.Name, err)
	}
	stdout, err := os.Create(fmt.Sprintf("/tmp/%s-stdout.log", p.Name))
	if err != nil {
		stdin.Close()
		return fmt.Errorf("failed to create stdout log for %s: %w", p.Name, err)
	}
	stderr, err := os.Create(fmt.Sprintf("/tmp/%s-stderr.log", p.Name))
	if err != nil {
		stdin.Close()
		stdout.Close()
		return fmt.Errorf("failed to create stderr log for %s: %w", p.Name, err)
	}

	args := make([]string, len(p.Command))
	copy(args, p.Command)
	args[0] = path.Base(args[0])

	process, err := os.StartProcess(
		p.Command[0],
		args,
		&os.ProcAttr{
			Dir:   p.Home,
			Env:   os.Environ(),
			Files: []*os.File{stdin, stdout, stderr},
		},
	)
	if err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return fmt.Errorf("failed to start %s: %w", p.Name, err)
	}

	stdin.Close()
	stdout.Close()
	stderr.Close()

	p.mu.Lock()
	p.Child = process
	p.mu.Unlock()

	go func() {
		_, err := process.Wait()
		if err != nil {
			log.Println(err)
		}
		p.mu.Lock()
		p.Child = nil
		p.mu.Unlock()
	}()

	log.Println("started", p.Name)
	return nil
}

func stopChild(p *ChildPlugin) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Child != nil {
		if err := p.Child.Kill(); err != nil {
			return fmt.Errorf("failed to kill process %s: %w", p.Name, err)
		}
	}

	log.Println("stopped", p.Name)
	return nil
}
