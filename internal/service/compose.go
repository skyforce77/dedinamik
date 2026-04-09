package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/skyforce77/dedinamik/internal/plugin"
)

type ComposePlugin struct {
	Name        string
	ComposeFile string
	ProjectName string
}

type ComposePluginFile struct {
	ComposeFile string `json:"composeFile"`
	ProjectName string `json:"projectName"`
}

func (p *ComposePlugin) GetName() string    { return p.Name }
func (p *ComposePlugin) GetCanSleep() bool  { return true }

func (p *ComposePlugin) Start() error {
	if err := p.runCompose("up", "-d"); err != nil {
		return fmt.Errorf("failed to start compose stack %s: %w", p.Name, err)
	}
	log.Printf("started compose stack %s", p.Name)
	return nil
}

func (p *ComposePlugin) Stop() error {
	if err := p.runCompose("down"); err != nil {
		return fmt.Errorf("failed to stop compose stack %s: %w", p.Name, err)
	}
	log.Printf("stopped compose stack %s", p.Name)
	return nil
}

func (p *ComposePlugin) Sleep() error {
	if err := p.runCompose("pause"); err != nil {
		return fmt.Errorf("failed to pause compose stack %s: %w", p.Name, err)
	}
	log.Printf("paused compose stack %s", p.Name)
	return nil
}

func (p *ComposePlugin) WakeUp() error {
	if err := p.runCompose("unpause"); err != nil {
		return fmt.Errorf("failed to unpause compose stack %s: %w", p.Name, err)
	}
	log.Printf("unpaused compose stack %s", p.Name)
	return nil
}

func (p *ComposePlugin) runCompose(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	composeFile, err := filepath.Abs(p.ComposeFile)
	if err != nil {
		return fmt.Errorf("invalid compose file path: %w", err)
	}

	baseArgs := []string{"compose", "-f", composeFile}
	if p.ProjectName != "" {
		baseArgs = append(baseArgs, "-p", p.ProjectName)
	}
	baseArgs = append(baseArgs, args...)

	cmd := exec.CommandContext(ctx, "docker", baseArgs...) // #nosec G204 -- args are from trusted config, not user input
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

func CreateComposePlugin(file *plugin.PluginFile) plugin.ServicePlugin {
	cfg := ComposePluginFile{}
	if err := json.Unmarshal(file.Config, &cfg); err != nil {
		log.Printf("failed to parse compose plugin config: %v", err)
		return nil
	}

	return &ComposePlugin{
		Name:        file.Name,
		ComposeFile: cfg.ComposeFile,
		ProjectName: cfg.ProjectName,
	}
}
