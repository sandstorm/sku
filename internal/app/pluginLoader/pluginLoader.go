package pluginLoader

import (
	"github.com/sandstorm/sku/pkg/utility"
	"path/filepath"
	"io/ioutil"
	"os"
	"log"
	"plugin"
	"github.com/sandstorm/sku/pkg/skuPluginApi"
	"github.com/sandstorm/sku/internal/app/commands"
)

func Load() {
	skuPathAndFilename := utility.GetSkuExecutableFileName()
	skuPath := filepath.Dir(skuPathAndFilename)
	skuPluginDirectory := filepath.Join(skuPath, "sku_plugins")
	statResult, err := os.Stat(skuPluginDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			// no plugins :)
			return
		}
		log.Fatalf("There was an error calling Stat(): %s", err)
	}

	if !statResult.IsDir() {
		log.Fatalf("There path %s is no directory.", skuPluginDirectory)
	}

	pluginFiles, err := ioutil.ReadDir(skuPluginDirectory)
	if err != nil {
		log.Fatalf("could not ReadDir of folder %s: %s", skuPluginDirectory, err)
	}
	for _, pluginFile := range pluginFiles {
		log.Printf("Finding plugin %s", pluginFile.Name())
		pluginInstance, err := plugin.Open(filepath.Join(skuPluginDirectory, pluginFile.Name()))
		if err != nil {
			log.Fatalf("There was an error opening the plugin %s: %s", pluginFile.Name(), err)
		}

		pluginSymbol, err := pluginInstance.Lookup("Plugin")
		if err != nil {
			log.Fatalf("There was an error readingthe plugin symbol 'Plugin': %s", err)
			os.Exit(1)
		}

		var pluginApi skuPluginApi.PluginApi
		pluginApi, ok := pluginSymbol.(skuPluginApi.PluginApi)
		if !ok {
			log.Fatalf("unexpected type from module symbol")
		}
		pluginApi.InitializeCommands(commands.RootCmd)
	}

}
