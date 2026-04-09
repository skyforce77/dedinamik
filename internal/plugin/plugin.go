package plugin

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/skyforce77/dedinamik/internal/activity"
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
	AwaitList       []activity.AwaitActivity

	Mutex sync.RWMutex
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
	} else {
		log.Printf("failed to %s plugin: %v", "start", err)
	}
	plugin.Mutex.Unlock()
}

func (plugin *MonitoredPlugin) Stop() {
	plugin.Mutex.Lock()
	err := plugin.Plugin.Stop()
	if err == nil {
		plugin.State = PluginStopped
	} else {
		log.Printf("failed to %s plugin: %v", "stop", err)
	}
	plugin.Mutex.Unlock()
}

func (plugin *MonitoredPlugin) Sleep() {
	plugin.Mutex.Lock()
	err := plugin.Plugin.Sleep()
	if err == nil {
		plugin.State = PluginSleeping
	} else {
		log.Printf("failed to %s plugin: %v", "sleep", err)
	}
	plugin.Mutex.Unlock()
}

func (plugin *MonitoredPlugin) WakeUp() {
	plugin.Mutex.Lock()
	err := plugin.Plugin.WakeUp()
	if err == nil {
		plugin.State = PluginRunning
	} else {
		log.Printf("failed to %s plugin: %v", "wakeup", err)
	}
	plugin.Mutex.Unlock()
}

// GetAwaitList implements activity.PluginWithActivity.
func (plugin *MonitoredPlugin) GetAwaitList() []activity.AwaitActivity {
	return plugin.AwaitList
}

// IsRunning implements monitor.Monitorable.
func (plugin *MonitoredPlugin) IsRunning() bool {
	return plugin.GetState() == PluginRunning
}

// CanSleep implements monitor.Monitorable.
func (plugin *MonitoredPlugin) CanSleep() bool {
	return plugin.Plugin.GetCanSleep()
}

// GetStateName implements monitor.Monitorable.
func (plugin *MonitoredPlugin) GetStateName() string {
	switch plugin.GetState() {
	case PluginRunning:
		return "running"
	case PluginSleeping:
		return "sleeping"
	default:
		return "stopped"
	}
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

var PluginTypes = make(map[string]*PluginType)

func RegisterPluginType(name string, pt *PluginType) {
	PluginTypes[name] = pt
}

func LoadPluginFromFile(path string) (*MonitoredPlugin, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- path comes from trusted config directory walk
	if err != nil {
		return nil, fmt.Errorf("can't read file %s: %w", path, err)
	}

	var pluginFile PluginFile
	err = json.Unmarshal(raw, &pluginFile)
	if err != nil {
		return nil, err
	}

	awaitList := make([]activity.AwaitActivity, len(pluginFile.AwaitList))
	for i := range awaitList {
		f := &activity.AwaitActivityFile{}
		err = json.Unmarshal(pluginFile.AwaitList[i], f)
		if err != nil {
			return nil, err
		}
		awaitList[i] = f.Create(pluginFile.AwaitList[i])
	}

	monitored := MonitoredPlugin{
		State:           PluginStopped,
		LastActivity:    time.Now(),
		TimeBeforeSleep: time.Duration(pluginFile.TimeBeforeSleep) * time.Minute,
		AwaitList:       awaitList,
	}

	typ := PluginTypes[pluginFile.Type]
	if typ != nil {
		monitored.Plugin = typ.FromFile(&pluginFile)
	}

	return &monitored, nil
}
