package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/0xjjjjjj/breathe/internal/config"
	"github.com/0xjjjjjj/breathe/internal/tui"
)

var (
	cfgFile  string
	jsonOut  bool
	junkOnly bool
)

var rootCmd = &cobra.Command{
	Use:   "breathe",
	Short: "Disk space manager and file organizer",
	Long:  `Let your disk breathe. Scan for space hogs, detect junk, and organize files.`,
}

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan directory for disk usage",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			return err
		}

		return tui.Run(cfg, absPath)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/breathe/config.yaml)")
	scanCmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	scanCmd.Flags().BoolVar(&junkOnly, "junk", false, "show only detected junk")
	rootCmd.AddCommand(scanCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
