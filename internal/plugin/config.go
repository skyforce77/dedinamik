package plugin

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func CreatePlugins(configDir string) []*MonitoredPlugin {
	plugins := make([]*MonitoredPlugin, 0)

	err := filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".json") {
			plugin, err := LoadPluginFromFile(path)
			if err == nil {
				plugins = append(plugins, plugin)
			} else {
				log.Println(err)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal("Can't find plugins folder.")
	}

	return plugins
}
