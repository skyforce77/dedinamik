package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/skyforce77/dedinamik/internal/activity"
	"github.com/skyforce77/dedinamik/internal/monitor"
	"github.com/skyforce77/dedinamik/internal/plugin"
	"github.com/skyforce77/dedinamik/internal/service"
)

func main() {
	// Register plugin types
	plugin.RegisterPluginType("child", &plugin.PluginType{
		FromFile: service.CreateChildPlugin,
	})
	plugin.RegisterPluginType("systemd", &plugin.PluginType{
		FromFile: service.CreateSystemDPlugin,
	})
	plugin.RegisterPluginType("docker", &plugin.PluginType{
		FromFile: service.CreateDockerPlugin,
	})
	plugin.RegisterPluginType("compose", &plugin.PluginType{
		FromFile: service.CreateComposePlugin,
	})

	// Load plugins from configs directory
	plugins := plugin.CreatePlugins("configs")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Convert to interface slices for monitor and activity
	monitorables := make([]monitor.Monitorable, len(plugins))
	activityPlugins := make([]activity.PluginWithActivity, len(plugins))
	for i, p := range plugins {
		monitorables[i] = p
		activityPlugins[i] = p
	}

	monitor.Start(ctx, monitorables)
	activity.ListenActivity(ctx, activityPlugins)

	log.Println("started.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down...")
	cancel()

	for _, p := range plugins {
		if p.GetState() != plugin.PluginStopped {
			p.Stop()
		}
	}
}
