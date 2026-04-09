package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/skyforce77/dedinamik/internal/plugin"
)

type DockerPlugin struct {
	Name          string
	Image         string
	ContainerName string
	Env           []string
	Volumes       []string
	Network       string
	Ports         []string // "host:container" format
	ContainerID   string
}

type DockerPluginFile struct {
	Image         string   `json:"image"`
	ContainerName string   `json:"containerName"`
	Env           []string `json:"env"`
	Volumes       []string `json:"volumes"`
	Network       string   `json:"network"`
	Ports         []string `json:"ports"`
}

func (p *DockerPlugin) GetName() string   { return p.Name }
func (p *DockerPlugin) GetCanSleep() bool { return true }

func (p *DockerPlugin) Start() error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	// Pull image if not present
	_, _, err = cli.ImageInspectWithRaw(ctx, p.Image)
	if err != nil {
		log.Printf("pulling image %s...", p.Image)
		reader, err := cli.ImagePull(ctx, p.Image, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("failed to pull image %s: %w", p.Image, err)
		}
		io.Copy(io.Discard, reader)
		reader.Close()
	}

	// Parse port bindings
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	for _, portMapping := range p.Ports {
		binding, err := nat.ParsePortSpec(portMapping)
		if err != nil {
			return fmt.Errorf("invalid port mapping %s: %w", portMapping, err)
		}
		for _, b := range binding {
			exposedPorts[b.Port] = struct{}{}
			portBindings[b.Port] = append(portBindings[b.Port], b.Binding)
		}
	}

	// Create container
	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image:        p.Image,
			Env:          p.Env,
			ExposedPorts: exposedPorts,
		},
		&container.HostConfig{
			Binds:        p.Volumes,
			PortBindings: portBindings,
			NetworkMode:  container.NetworkMode(p.Network),
		},
		nil, nil, p.ContainerName,
	)
	if err != nil {
		return fmt.Errorf("failed to create container %s: %w", p.ContainerName, err)
	}
	p.ContainerID = resp.ID

	// Start container
	if err := cli.ContainerStart(ctx, p.ContainerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %s: %w", p.ContainerName, err)
	}

	log.Printf("started docker container %s (%s)", p.ContainerName, p.ContainerID[:12])
	return nil
}

func (p *DockerPlugin) Stop() error {
	if p.ContainerID == "" {
		return nil
	}
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	timeout := 30
	stopOpts := container.StopOptions{Timeout: &timeout}
	if err := cli.ContainerStop(ctx, p.ContainerID, stopOpts); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", p.ContainerName, err)
	}
	if err := cli.ContainerRemove(ctx, p.ContainerID, container.RemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", p.ContainerName, err)
	}

	log.Printf("stopped docker container %s", p.ContainerName)
	p.ContainerID = ""
	return nil
}

func (p *DockerPlugin) Sleep() error {
	if p.ContainerID == "" {
		return nil
	}
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	if err := cli.ContainerPause(ctx, p.ContainerID); err != nil {
		return fmt.Errorf("failed to pause container %s: %w", p.ContainerName, err)
	}
	log.Printf("paused docker container %s", p.ContainerName)
	return nil
}

func (p *DockerPlugin) WakeUp() error {
	if p.ContainerID == "" {
		return nil
	}
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	if err := cli.ContainerUnpause(ctx, p.ContainerID); err != nil {
		return fmt.Errorf("failed to unpause container %s: %w", p.ContainerName, err)
	}
	log.Printf("unpaused docker container %s", p.ContainerName)
	return nil
}

func CreateDockerPlugin(file *plugin.PluginFile) plugin.ServicePlugin {
	cfg := DockerPluginFile{}
	if err := json.Unmarshal(file.Config, &cfg); err != nil {
		log.Printf("failed to parse docker plugin config: %v", err)
		return nil
	}

	return &DockerPlugin{
		Name:          file.Name,
		Image:         cfg.Image,
		ContainerName: cfg.ContainerName,
		Env:           cfg.Env,
		Volumes:       cfg.Volumes,
		Network:       cfg.Network,
		Ports:         cfg.Ports,
	}
}
