// SPDX-FileCopyrightText: The RamenDR authors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"github.com/ramendr/ramenctl/pkg/console"
	"github.com/ramendr/ramenctl/pkg/test"
	"github.com/spf13/cobra"
)

var TestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test disaster recovery flow in your clusters",
}

var TestRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run disaster recovery flow",
	Run: func(c *cobra.Command, args []string) {
		if err := test.Run(outputDir()); err != nil {
			console.Fatal(err)
		}
		console.Completed("Test run completed successfully")
	},
}

var TestCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Delete test artifacts",
	Run: func(c *cobra.Command, args []string) {
		if err := test.Clean(outputDir()); err != nil {
			console.Fatal(err)
		}
		console.Completed("Test cleanup completed successfully")
	},
}

func init() {
	addOutputFlag(TestCmd)
	TestCmd.AddCommand(TestRunCmd, TestCleanCmd)
}
