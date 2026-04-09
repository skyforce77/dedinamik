package service

import (
	"encoding/json"
	"testing"

	"github.com/skyforce77/dedinamik/internal/plugin"
)

func makePluginFile(name, typ string, config interface{}) *plugin.PluginFile {
	raw, _ := json.Marshal(config)
	return &plugin.PluginFile{
		Name:   name,
		Type:   typ,
		Config: raw,
	}
}

// --- Child Plugin ---

func TestCreateChildPlugin(t *testing.T) {
	pf := makePluginFile("mychild", "child", map[string]interface{}{
		"command": []string{"/usr/bin/sleep", "60"},
		"home":    "/tmp",
		"freeze":  true,
	})

	sp := CreateChildPlugin(pf)
	if sp == nil {
		t.Fatal("expected non-nil plugin")
	}

	cp, ok := sp.(*ChildPlugin)
	if !ok {
		t.Fatal("expected *ChildPlugin")
	}

	if cp.Name != "mychild" {
		t.Errorf("Name = %q, want %q", cp.Name, "mychild")
	}
	if len(cp.Command) != 2 || cp.Command[0] != "/usr/bin/sleep" || cp.Command[1] != "60" {
		t.Errorf("Command = %v, want [/usr/bin/sleep 60]", cp.Command)
	}
	if cp.Home != "/tmp" {
		t.Errorf("Home = %q, want %q", cp.Home, "/tmp")
	}
	if !cp.Freeze {
		t.Error("Freeze = false, want true")
	}
}

func TestCreateChildPlugin_InvalidConfig(t *testing.T) {
	pf := &plugin.PluginFile{
		Name:   "bad",
		Type:   "child",
		Config: json.RawMessage(`{invalid json`),
	}

	sp := CreateChildPlugin(pf)
	if sp != nil {
		t.Fatal("expected nil for invalid config")
	}
}

// --- SystemD Plugin ---

func TestCreateSystemDPlugin(t *testing.T) {
	pf := makePluginFile("mysvc", "systemd", map[string]interface{}{
		"service": "nginx.service",
		"mode":    "replace",
	})

	sp := CreateSystemDPlugin(pf)
	if sp == nil {
		t.Fatal("expected non-nil plugin")
	}

	sdp, ok := sp.(*SystemDPlugin)
	if !ok {
		t.Fatal("expected *SystemDPlugin")
	}

	if sdp.Name != "mysvc" {
		t.Errorf("Name = %q, want %q", sdp.Name, "mysvc")
	}
	if sdp.ServiceName != "nginx.service" {
		t.Errorf("ServiceName = %q, want %q", sdp.ServiceName, "nginx.service")
	}
	if sdp.JobMode != "replace" {
		t.Errorf("JobMode = %q, want %q", sdp.JobMode, "replace")
	}
}

func TestCreateSystemDPlugin_InvalidConfig(t *testing.T) {
	pf := &plugin.PluginFile{
		Name:   "bad",
		Type:   "systemd",
		Config: json.RawMessage(`not json`),
	}

	sp := CreateSystemDPlugin(pf)
	if sp != nil {
		t.Fatal("expected nil for invalid config")
	}
}

// --- Docker Plugin ---

func TestCreateDockerPlugin(t *testing.T) {
	pf := makePluginFile("mydocker", "docker", map[string]interface{}{
		"image":         "nginx:latest",
		"containerName": "test-nginx",
		"env":           []string{"FOO=bar", "BAZ=qux"},
		"volumes":       []string{"/host/data:/data"},
		"network":       "bridge",
		"ports":         []string{"8080:80", "8443:443"},
	})

	sp := CreateDockerPlugin(pf)
	if sp == nil {
		t.Fatal("expected non-nil plugin")
	}

	dp, ok := sp.(*DockerPlugin)
	if !ok {
		t.Fatal("expected *DockerPlugin")
	}

	if dp.Name != "mydocker" {
		t.Errorf("Name = %q, want %q", dp.Name, "mydocker")
	}
	if dp.Image != "nginx:latest" {
		t.Errorf("Image = %q, want %q", dp.Image, "nginx:latest")
	}
	if dp.ContainerName != "test-nginx" {
		t.Errorf("ContainerName = %q, want %q", dp.ContainerName, "test-nginx")
	}
	if len(dp.Env) != 2 || dp.Env[0] != "FOO=bar" || dp.Env[1] != "BAZ=qux" {
		t.Errorf("Env = %v, want [FOO=bar BAZ=qux]", dp.Env)
	}
	if len(dp.Volumes) != 1 || dp.Volumes[0] != "/host/data:/data" {
		t.Errorf("Volumes = %v, want [/host/data:/data]", dp.Volumes)
	}
	if dp.Network != "bridge" {
		t.Errorf("Network = %q, want %q", dp.Network, "bridge")
	}
	if len(dp.Ports) != 2 || dp.Ports[0] != "8080:80" || dp.Ports[1] != "8443:443" {
		t.Errorf("Ports = %v, want [8080:80 8443:443]", dp.Ports)
	}
}

func TestCreateDockerPlugin_InvalidConfig(t *testing.T) {
	pf := &plugin.PluginFile{
		Name:   "bad",
		Type:   "docker",
		Config: json.RawMessage(`%%%`),
	}

	sp := CreateDockerPlugin(pf)
	if sp != nil {
		t.Fatal("expected nil for invalid config")
	}
}

// --- Compose Plugin ---

func TestCreateComposePlugin(t *testing.T) {
	pf := makePluginFile("mycompose", "compose", map[string]interface{}{
		"composeFile": "/path/to/docker-compose.yml",
		"projectName": "myproject",
	})

	sp := CreateComposePlugin(pf)
	if sp == nil {
		t.Fatal("expected non-nil plugin")
	}

	cp, ok := sp.(*ComposePlugin)
	if !ok {
		t.Fatal("expected *ComposePlugin")
	}

	if cp.Name != "mycompose" {
		t.Errorf("Name = %q, want %q", cp.Name, "mycompose")
	}
	if cp.ComposeFile != "/path/to/docker-compose.yml" {
		t.Errorf("ComposeFile = %q, want %q", cp.ComposeFile, "/path/to/docker-compose.yml")
	}
	if cp.ProjectName != "myproject" {
		t.Errorf("ProjectName = %q, want %q", cp.ProjectName, "myproject")
	}
}

func TestCreateComposePlugin_InvalidConfig(t *testing.T) {
	pf := &plugin.PluginFile{
		Name:   "bad",
		Type:   "compose",
		Config: json.RawMessage(`{nope`),
	}

	sp := CreateComposePlugin(pf)
	if sp != nil {
		t.Fatal("expected nil for invalid config")
	}
}

// --- GetName ---

func TestChildPlugin_GetName(t *testing.T) {
	p := &ChildPlugin{Name: "testchild"}
	if got := p.GetName(); got != "testchild" {
		t.Errorf("GetName() = %q, want %q", got, "testchild")
	}
}

// --- GetCanSleep ---

func TestDockerPlugin_GetCanSleep(t *testing.T) {
	p := &DockerPlugin{Name: "d"}
	if !p.GetCanSleep() {
		t.Error("DockerPlugin.GetCanSleep() = false, want true")
	}
}

func TestComposePlugin_GetCanSleep(t *testing.T) {
	p := &ComposePlugin{Name: "c"}
	if !p.GetCanSleep() {
		t.Error("ComposePlugin.GetCanSleep() = false, want true")
	}
}

func TestChildPlugin_GetCanSleep(t *testing.T) {
	p := &ChildPlugin{Name: "ch"}
	if p.GetCanSleep() {
		t.Error("ChildPlugin.GetCanSleep() = true, want false")
	}
}
