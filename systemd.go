package main

import "log"

type SystemDPlugin struct {
	Name        string
	ServiceName string
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

func startSystemD(plugin *SystemDPlugin) error {
	//TODO
	log.Println("started " + plugin.Name)
	return nil
}

func stopSystemD(plugin *SystemDPlugin) error {
	//TODO
	log.Println("stopped " + plugin.Name)
	return nil
}
