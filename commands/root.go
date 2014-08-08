package commands

import "github.com/spf13/cobra"

var RootCommand = &cobra.Command{
	Use:   "reditool",
	Short: "reditool is a multipurpose Redis tool",
	Long:  `Reditool is the swiss army knife of Redis command line interfaces. It slices, dices, and purees your Redis server's information`,
	Run: func(cmd *cobra.Command, args []string) {
		//do something
	},
}

var rediCommand *cobra.Command

func Execute() {
	AddCommands()
	rediCommand.Execute()
}

func AddCommands() {
	RootCommand.AddCommand(version)
	RootCommand.AddCommand(inspectCommand)
	RootCommand.AddCommand(backup)
	RootCommand.AddCommand(cloneCommand)
	RootCommand.AddCommand(sentinelCloneCommand)
}

func init() {
	rediCommand = RootCommand
}
