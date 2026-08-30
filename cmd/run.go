/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <inventory publication name>",
	Short: "Run a container for the specified inventory publication target",
	Long:  `Run a container for the specified inventory publication target.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStdout(), "Running %s\n", args[0])
		return runPublication(args[0])
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runPublication(inventoryPublicationName string) error {
	return nil
}
