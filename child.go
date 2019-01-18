package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"syscall"
)

type ChildPlugin struct {
	Name    string
	Command []string
	Home	string
	Freeze  bool
	Child   *os.Process
}

type ChildPluginFile struct {
	Command []string `json:"command"`
	Home	string	`json:"home"`
	Freeze  bool     `json:"freeze"`
}

func (plugin *ChildPlugin) GetName() string {
	return plugin.Name
}
func (plugin *ChildPlugin) GetCanSleep() bool {
	return false
}
func (plugin *ChildPlugin) Start() error {
	return startChild(plugin)
}
func (plugin *ChildPlugin) Stop() error {
	return stopChild(plugin)
}
func (plugin *ChildPlugin) Sleep() error {
	return plugin.Child.Signal(syscall.SIGSTOP)
}
func (plugin *ChildPlugin) WakeUp() error {
	return plugin.Child.Signal(syscall.SIGCONT)
}

func createChildPlugin(file *PluginFile) ServicePlugin {
	command := ChildPluginFile{}

	err := json.Unmarshal(file.Config, &command)
	if err != nil {
		return nil
	}

	return &ChildPlugin{
		file.Name,
		command.Command,
		command.Home,
		command.Freeze,
		nil,
	}
}

func startChild(plugin *ChildPlugin) error {
	stdin, _ := os.Create(fmt.Sprintf("/tmp/%s-stdin.log", plugin.Name))
	stdout, _ := os.Create(fmt.Sprintf("/tmp/%s-stdout.log", plugin.Name))
	stderr, _ := os.Create(fmt.Sprintf("/tmp/%s-stderr.log", plugin.Name))

	args := make([]string, len(plugin.Command))
	copy(args, plugin.Command)
	args[0] = path.Base(args[0])

	process, err := os.StartProcess(
		plugin.Command[0],
		args,
		&os.ProcAttr{
			plugin.Home,
			os.Environ(),
			[]*os.File{stdin, stderr, stdout},
			nil,
		},
	)
	if err != nil {
		panic(err)
	}
	plugin.Child = process
	go func() {
		_, err := process.Wait()
		if err != nil {
			log.Println(err)
		}
		plugin.Child = nil
	}()

	log.Println("started", plugin.Name)
	return nil
}

func stopChild(plugin *ChildPlugin) error {
	if plugin.Child != nil {
		if err := plugin.Child.Kill(); err != nil {
			log.Fatal("failed to kill process: ", err)
		}
	}

	log.Println("stopped", plugin.Name)
	return nil
}
