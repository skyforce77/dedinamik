package main

import (
	"encoding/json"
	"github.com/coreos/go-systemd/dbus"
	"log"
)

type SystemDPlugin struct {
	Name        string
	ServiceName string
	JobMode string
	JobId int
}

type SystemDPluginFile struct {
	ServiceName string `json:"service"`
	JobMode string `json:"mode"`
}

func (plugin *SystemDPlugin) GetName() string {
	return plugin.Name
}
func (plugin *SystemDPlugin) GetCanSleep() bool {
	return false
}
func (plugin *SystemDPlugin) Start() error {
	return startSystemD(plugin)
}
func (plugin *SystemDPlugin) Stop() error {
	return stopSystemD(plugin)
}
func (plugin *SystemDPlugin) Sleep() error {
	return nil
}
func (plugin *SystemDPlugin) WakeUp() error {
	return nil
}

func createSystemDPlugin(file *PluginFile) ServicePlugin {
	command := SystemDPluginFile{}

	err := json.Unmarshal(file.Config, &command)
	if err != nil {
		return nil
	}

	return &SystemDPlugin{
		file.Name,
		command.ServiceName,
		command.JobMode,
		-1,
	}
}

func startSystemD(plugin *SystemDPlugin) error {
	conn, err := dbus.NewSystemConnection()
	if err != nil {
		panic(err)
	}
	id, err := conn.StartUnit(plugin.ServiceName, plugin.JobMode, nil)
	if err != nil {
		panic(err)
	}
	plugin.JobId = id
	conn.Close()
	log.Println("started " + plugin.Name)
	return nil
}

func stopSystemD(plugin *SystemDPlugin) error {
	conn, err := dbus.NewSystemConnection()
	if err != nil {
		panic(err)
	}
	_, err = conn.StopUnit(plugin.ServiceName, plugin.JobMode, nil)
	if err != nil {
		panic(err)
	}
	conn.Close()
	log.Println("stopped " + plugin.Name)
	return nil
}
