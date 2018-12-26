package main

import (
	"github.com/carlescere/scheduler"
	"log"
)

func monitor(plugins []*MonitoredPlugin) {
	log.Println("initializing monitoring")

	_, err := scheduler.Every(1).Seconds().Run(func() {
		for _, plugin := range plugins {
			if plugin.State == PluginRunning && plugin.WantToSleep() {
				if plugin.Plugin.GetCanSleep() {
					plugin.Sleep()
				} else {
					plugin.Stop()
				}
			}
		}
	})

	if err != nil {
		panic(err)
	}
}
