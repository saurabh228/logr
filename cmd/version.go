package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags:
//
//	go build -ldflags "-X github.com/saurabh/logr/cmd.Version=1.2.3"
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the logr version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("logr version %s\n", Version)
	},
}
