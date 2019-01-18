package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"
	"time"
)

type PluginState uint8

const (
	PluginRunning PluginState = iota
	PluginSleeping
	PluginStopped
)

type MonitoredPlugin struct {
	Plugin          ServicePlugin
	State           PluginState
	LastActivity    time.Time
	TimeBeforeSleep time.Duration
	Peers           int
	AwaitList       []AwaitActivity

	Mutex *sync.RWMutex
}

func (plugin *MonitoredPlugin) GetPeers() int {
	plugin.Mutex.RLock()
	peers := plugin.Peers
	plugin.Mutex.RUnlock()
	return peers
}
func (plugin *MonitoredPlugin) GetState() PluginState {
	plugin.Mutex.RLock()
	state := plugin.State
	plugin.Mutex.RUnlock()
	return state
}
func (plugin *MonitoredPlugin) PeerConnected() {
	if plugin.GetState() == PluginStopped {
		plugin.Start()
	}
	plugin.Mutex.Lock()
	plugin.Peers += 1
	plugin.LastActivity = time.Now()
	plugin.Mutex.Unlock()
}
func (plugin *MonitoredPlugin) PeerDisconnected() {
	plugin.Mutex.Lock()
	plugin.Peers -= 1
	plugin.LastActivity = time.Now()
	plugin.Mutex.Unlock()
}
func (plugin *MonitoredPlugin) WantToSleep() bool {
	plugin.Mutex.RLock()
	hasToStop := false
	if plugin.Peers <= 0 {
		hasToStop = plugin.LastActivity.Add(plugin.TimeBeforeSleep).Before(time.Now())
	}
	plugin.Mutex.RUnlock()
	return hasToStop
}
func (plugin *MonitoredPlugin) Start() {
	plugin.Mutex.Lock()
	err := plugin.Plugin.Start()
	if err == nil {
		plugin.State = PluginRunning
	}
	plugin.Mutex.Unlock()
}
func (plugin *MonitoredPlugin) Stop() {
	plugin.Mutex.Lock()
	err := plugin.Plugin.Stop()
	if err == nil {
		plugin.State = PluginStopped
	}
	plugin.Mutex.Unlock()
}
func (plugin *MonitoredPlugin) Sleep() {
	plugin.Mutex.Lock()
	err := plugin.Plugin.Sleep()
	if err == nil {
		plugin.State = PluginSleeping
	}
	plugin.Mutex.Unlock()
}
func (plugin *MonitoredPlugin) WakeUp() {
	plugin.Mutex.Lock()
	err := plugin.Plugin.WakeUp()
	if err == nil {
		plugin.State = PluginRunning
	}
	plugin.Mutex.Unlock()
}

type ServicePlugin interface {
	GetName() string
	GetCanSleep() bool

	Start() error
	Stop() error
	Sleep() error
	WakeUp() error
}

type PluginFile struct {
	Name            string            `json:"name"`
	Type            string            `json:"type"`
	TimeBeforeSleep int               `json:"waitTime"`
	AwaitList       []json.RawMessage `json:"await"`
	Config          json.RawMessage   `json:"config"`
}

type PluginType struct {
	FromFile func(file *PluginFile) ServicePlugin
}

var (
	pluginTypes = make(map[string]*PluginType)
)

func registerPluginTypes() {
	pluginTypes["child"] = &PluginType{createChildPlugin}
	pluginTypes["systemd"] = &PluginType{createSystemDPlugin}
}

func loadPluginFromFile(path string) (*MonitoredPlugin, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("Can't read file", path)
		return nil, err
	}

	var pluginFile PluginFile
	err = json.Unmarshal(raw, &pluginFile)
	if err != nil {
		return nil, err
	}

	awaitList := make([]AwaitActivity, len(pluginFile.AwaitList))
	for i := range awaitList {
		f := &AwaitActivityFile{}
		err = json.Unmarshal(pluginFile.AwaitList[i], f)
		if err != nil {
			return nil, err
		}
		awaitList[i] = f.Create(pluginFile.AwaitList[i])
	}

	monitored := MonitoredPlugin{
		nil,
		PluginStopped,
		time.Now(),
		time.Duration(pluginFile.TimeBeforeSleep) * time.Minute,
		0,
		awaitList,
		new(sync.RWMutex),
	}

	typ := pluginTypes[pluginFile.Type]
	if typ != nil {
		monitored.Plugin = typ.FromFile(&pluginFile)
	}

	return &monitored, nil
}
