package cmd

import (
	"fmt"

	"github.com/UKHomeOffice/keto/pkg/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Long:  "Print version",
	Run: func(c *cobra.Command, args []string) {
		printVersion()
	},
}

func printVersion() {
	fmt.Printf("%+v\n", version.Get())
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
