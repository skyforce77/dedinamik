package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
)

type ChildPlugin struct {
	Name    string
	Command []string
	Child   *os.Process
}

type ChildPluginFile struct {
	Command []string `json:"command"`
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
	return nil
}
func (plugin *ChildPlugin) WakeUp() error {
	return nil
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
			path.Dir(plugin.Command[0]),
			os.Environ(),
			[]*os.File{stdin, stdout, stderr},
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
		log.Println("stopped")
		plugin.Child = nil
	}()
	return nil
}

func stopChild(plugin *ChildPlugin) error {
	if plugin.Child != nil {
		if err := plugin.Child.Kill(); err != nil {
			log.Fatal("failed to kill process: ", err)
		}
	}
	return nil
}
