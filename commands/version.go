package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of reditool",
	Long:  `All software has versions. This is Reditool's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(".1")
	},
}
