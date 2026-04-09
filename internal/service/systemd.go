package service

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/skyforce77/dedinamik/internal/plugin"
)

type SystemDPlugin struct {
	Name        string
	ServiceName string
	JobMode     string
}

type SystemDPluginFile struct {
	ServiceName string `json:"service"`
	JobMode     string `json:"mode"`
}

func (p *SystemDPlugin) GetName() string {
	return p.Name
}
func (p *SystemDPlugin) GetCanSleep() bool {
	return false
}
func (p *SystemDPlugin) Start() error {
	return startSystemD(p)
}
func (p *SystemDPlugin) Stop() error {
	return stopSystemD(p)
}
func (p *SystemDPlugin) Sleep() error {
	return nil
}
func (p *SystemDPlugin) WakeUp() error {
	return nil
}

func CreateSystemDPlugin(file *plugin.PluginFile) plugin.ServicePlugin {
	command := SystemDPluginFile{}

	err := json.Unmarshal(file.Config, &command)
	if err != nil {
		return nil
	}

	return &SystemDPlugin{
		Name:        file.Name,
		ServiceName: command.ServiceName,
		JobMode:     command.JobMode,
	}
}

func startSystemD(p *SystemDPlugin) error {
	conn, err := dbus.NewSystemConnection()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer conn.Close()
	_, err = conn.StartUnit(p.ServiceName, p.JobMode, nil)
	if err != nil {
		return fmt.Errorf("failed to start unit %s: %w", p.ServiceName, err)
	}
	log.Println("started " + p.Name)
	return nil
}

func stopSystemD(p *SystemDPlugin) error {
	conn, err := dbus.NewSystemConnection()
	if err != nil {
		return fmt.Errorf("failed to connect to systemd: %w", err)
	}
	defer conn.Close()
	_, err = conn.StopUnit(p.ServiceName, p.JobMode, nil)
	if err != nil {
		return fmt.Errorf("failed to stop unit %s: %w", p.ServiceName, err)
	}
	log.Println("stopped " + p.Name)
	return nil
}
