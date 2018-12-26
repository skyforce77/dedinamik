package main

import (
	"log"
	"os"
	"os/signal"
	"time"
)

func main() {
	registerPluginTypes()
	plugins := createPlugins()

	monitor(plugins)
	listenActivity(plugins)

	log.Println("started.")

	// Graceful stop
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		select {
		case _ = <-c:
			for _, plugin := range plugins {
				if plugin.GetState() != PluginStopped {
					plugin.Stop()
				}
			}
			os.Exit(0)
		}
	}()

	for {
		time.Sleep(5 * time.Second)
	}
}
