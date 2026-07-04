package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:   "assetagent",
		Short: "AI native wealth management agent",
	}

	root.Flags().Bool("version", false, "Print version and exit")
	root.RunE = func(cmd *cobra.Command, args []string) error {
		showVersion, _ := cmd.Flags().GetBool("version")
		if showVersion {
			fmt.Println(version)
			return nil
		}
		return cmd.Help()
	}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
