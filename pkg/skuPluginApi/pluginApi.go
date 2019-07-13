package skuPluginApi

import "github.com/spf13/cobra"

type PluginApi interface {
	InitializeCommands(rootCommand *cobra.Command)
}