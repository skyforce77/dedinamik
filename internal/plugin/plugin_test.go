package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/skyforce77/dedinamik/internal/activity"
)

// mockPlugin implements ServicePlugin for testing.
type mockPlugin struct {
	name     string
	canSleep bool
	started  bool
	stopped  bool
	sleeping bool
}

func (m *mockPlugin) GetName() string   { return m.name }
func (m *mockPlugin) GetCanSleep() bool { return m.canSleep }
func (m *mockPlugin) Start() error      { m.started = true; return nil }
func (m *mockPlugin) Stop() error       { m.stopped = true; return nil }
func (m *mockPlugin) Sleep() error      { m.sleeping = true; return nil }
func (m *mockPlugin) WakeUp() error     { m.sleeping = false; return nil }

func newTestPlugin(name string, canSleep bool) (*MonitoredPlugin, *mockPlugin) {
	mock := &mockPlugin{name: name, canSleep: canSleep}
	mp := &MonitoredPlugin{
		Plugin:          mock,
		State:           PluginStopped,
		LastActivity:    time.Now(),
		TimeBeforeSleep: 5 * time.Minute,
	}
	return mp, mock
}

func TestPluginStateConstants(t *testing.T) {
	if PluginRunning == PluginSleeping {
		t.Error("PluginRunning should not equal PluginSleeping")
	}
	if PluginRunning == PluginStopped {
		t.Error("PluginRunning should not equal PluginStopped")
	}
	if PluginSleeping == PluginStopped {
		t.Error("PluginSleeping should not equal PluginStopped")
	}
}

func TestMonitoredPlugin_PeerConnected(t *testing.T) {
	mp, mock := newTestPlugin("test", false)

	if mp.GetState() != PluginStopped {
		t.Fatalf("expected initial state PluginStopped, got %v", mp.GetState())
	}

	mp.PeerConnected()

	if !mock.started {
		t.Error("expected plugin to be started when PeerConnected is called while stopped")
	}
	if mp.GetState() != PluginRunning {
		t.Errorf("expected state PluginRunning after PeerConnected, got %v", mp.GetState())
	}
	if mp.GetPeers() != 1 {
		t.Errorf("expected 1 peer, got %d", mp.GetPeers())
	}

	// Second connection should not re-trigger Start (already running).
	mock.started = false
	mp.PeerConnected()
	if mp.GetPeers() != 2 {
		t.Errorf("expected 2 peers, got %d", mp.GetPeers())
	}
}

func TestMonitoredPlugin_PeerDisconnected(t *testing.T) {
	mp, _ := newTestPlugin("test", false)
	mp.State = PluginRunning
	mp.Peers = 3

	mp.PeerDisconnected()
	if mp.GetPeers() != 2 {
		t.Errorf("expected 2 peers after disconnect, got %d", mp.GetPeers())
	}

	mp.PeerDisconnected()
	if mp.GetPeers() != 1 {
		t.Errorf("expected 1 peer after second disconnect, got %d", mp.GetPeers())
	}
}

func TestMonitoredPlugin_WantToSleep(t *testing.T) {
	mp, _ := newTestPlugin("test", false)
	mp.State = PluginRunning
	mp.TimeBeforeSleep = 50 * time.Millisecond

	// With peers > 0, should not want to sleep.
	mp.Peers = 1
	mp.LastActivity = time.Now().Add(-1 * time.Hour)
	if mp.WantToSleep() {
		t.Error("should not want to sleep when peers > 0")
	}

	// With 0 peers but recent activity, should not want to sleep.
	mp.Peers = 0
	mp.LastActivity = time.Now()
	if mp.WantToSleep() {
		t.Error("should not want to sleep when last activity is recent")
	}

	// With 0 peers and enough time passed, should want to sleep.
	mp.LastActivity = time.Now().Add(-1 * time.Second)
	if !mp.WantToSleep() {
		t.Error("should want to sleep when peers == 0 and enough time has passed")
	}
}

func TestMonitoredPlugin_StartStop(t *testing.T) {
	mp, mock := newTestPlugin("test", false)

	mp.Start()
	if !mock.started {
		t.Error("expected Start to call plugin.Start")
	}
	if mp.GetState() != PluginRunning {
		t.Errorf("expected PluginRunning after Start, got %v", mp.GetState())
	}

	mp.Stop()
	if !mock.stopped {
		t.Error("expected Stop to call plugin.Stop")
	}
	if mp.GetState() != PluginStopped {
		t.Errorf("expected PluginStopped after Stop, got %v", mp.GetState())
	}
}

func TestMonitoredPlugin_SleepWakeUp(t *testing.T) {
	mp, mock := newTestPlugin("test", true)
	mp.State = PluginRunning

	mp.Sleep()
	if !mock.sleeping {
		t.Error("expected Sleep to call plugin.Sleep")
	}
	if mp.GetState() != PluginSleeping {
		t.Errorf("expected PluginSleeping after Sleep, got %v", mp.GetState())
	}

	mp.WakeUp()
	if mock.sleeping {
		t.Error("expected WakeUp to call plugin.WakeUp")
	}
	if mp.GetState() != PluginRunning {
		t.Errorf("expected PluginRunning after WakeUp, got %v", mp.GetState())
	}
}

func TestMonitoredPlugin_GetAwaitList(t *testing.T) {
	mp, _ := newTestPlugin("test", false)

	// Empty await list.
	list := mp.GetAwaitList()
	if len(list) != 0 {
		t.Errorf("expected empty await list, got %d items", len(list))
	}

	// Non-empty await list.
	mp.AwaitList = []activity.AwaitActivity{
		&activity.SocketAwaitActivity{Connection: "tcp", From: ":8080", To: ":9090"},
	}
	list = mp.GetAwaitList()
	if len(list) != 1 {
		t.Errorf("expected 1 item in await list, got %d", len(list))
	}
}

func TestMonitoredPlugin_IsRunning(t *testing.T) {
	mp, _ := newTestPlugin("test", false)

	if mp.IsRunning() {
		t.Error("expected IsRunning to be false when stopped")
	}

	mp.State = PluginRunning
	if !mp.IsRunning() {
		t.Error("expected IsRunning to be true when running")
	}

	mp.State = PluginSleeping
	if mp.IsRunning() {
		t.Error("expected IsRunning to be false when sleeping")
	}
}

func TestMonitoredPlugin_CanSleep(t *testing.T) {
	mp, mock := newTestPlugin("test", false)

	if mp.CanSleep() {
		t.Error("expected CanSleep to be false")
	}

	mock.canSleep = true
	if !mp.CanSleep() {
		t.Error("expected CanSleep to be true")
	}
}

func TestRegisterPluginType(t *testing.T) {
	// Clean up after test.
	originalTypes := make(map[string]*PluginType)
	for k, v := range PluginTypes {
		originalTypes[k] = v
	}
	defer func() {
		PluginTypes = originalTypes
	}()

	pt := &PluginType{
		FromFile: func(file *PluginFile) ServicePlugin {
			return &mockPlugin{name: file.Name}
		},
	}

	RegisterPluginType("mock", pt)

	registered, ok := PluginTypes["mock"]
	if !ok {
		t.Fatal("expected 'mock' plugin type to be registered")
	}
	if registered != pt {
		t.Error("registered plugin type does not match")
	}

	// Verify FromFile works.
	sp := registered.FromFile(&PluginFile{Name: "myservice"})
	if sp.GetName() != "myservice" {
		t.Errorf("expected name 'myservice', got %q", sp.GetName())
	}
}

func TestLoadPluginFromFile(t *testing.T) {
	// Clean up after test.
	originalTypes := make(map[string]*PluginType)
	for k, v := range PluginTypes {
		originalTypes[k] = v
	}
	defer func() {
		PluginTypes = originalTypes
	}()

	RegisterPluginType("testtype", &PluginType{
		FromFile: func(file *PluginFile) ServicePlugin {
			return &mockPlugin{name: file.Name, canSleep: true}
		},
	})

	t.Run("valid JSON file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.json")

		data := map[string]interface{}{
			"name":     "myserver",
			"type":     "testtype",
			"waitTime": 10,
			"await":    []interface{}{},
		}
		raw, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal test data: %v", err)
		}
		if err := os.WriteFile(path, raw, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		mp, err := LoadPluginFromFile(path)
		if err != nil {
			t.Fatalf("LoadPluginFromFile returned error: %v", err)
		}
		if mp == nil {
			t.Fatal("expected non-nil MonitoredPlugin")
		}
		if mp.GetState() != PluginStopped {
			t.Errorf("expected initial state PluginStopped, got %v", mp.GetState())
		}
		if mp.TimeBeforeSleep != 10*time.Minute {
			t.Errorf("expected TimeBeforeSleep 10m, got %v", mp.TimeBeforeSleep)
		}
		if mp.Plugin == nil {
			t.Fatal("expected Plugin to be set")
		}
		if mp.Plugin.GetName() != "myserver" {
			t.Errorf("expected plugin name 'myserver', got %q", mp.Plugin.GetName())
		}
	})

	t.Run("invalid JSON file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.json")

		if err := os.WriteFile(path, []byte("not valid json{{{"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		_, err := LoadPluginFromFile(path)
		if err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := LoadPluginFromFile("/nonexistent/path/plugin.json")
		if err == nil {
			t.Error("expected error for missing file, got nil")
		}
	})

	t.Run("unknown plugin type", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "unknown.json")

		data := map[string]interface{}{
			"name":     "unknown",
			"type":     "nonexistent",
			"waitTime": 5,
			"await":    []interface{}{},
		}
		raw, _ := json.Marshal(data)
		if err := os.WriteFile(path, raw, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		mp, err := LoadPluginFromFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mp.Plugin != nil {
			t.Error("expected Plugin to be nil for unknown type")
		}
	})
}
