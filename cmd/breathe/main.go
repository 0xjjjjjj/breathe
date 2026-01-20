package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "breathe",
	Short: "Disk space manager and file organizer",
	Long:  `Let your disk breathe. Scan for space hogs, detect junk, and organize files.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
