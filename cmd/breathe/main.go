package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/0xjjjjjj/breathe/internal/config"
	"github.com/0xjjjjjj/breathe/internal/history"
	"github.com/0xjjjjjj/breathe/internal/organizer"
	"github.com/0xjjjjjj/breathe/internal/scanner"
	"github.com/0xjjjjjj/breathe/internal/tui"
)

var (
	cfgFile    string
	jsonOut    bool
	junkOnly   bool
	dryRun     bool
	apply      bool
	plan       bool
	sinceDays  int
	yesFlag    bool
	trashFlag  bool
	patternArg string
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

		if jsonOut {
			return runJSONScan(cfg, absPath)
		}

		return tui.Run(cfg, absPath)
	},
}

func runJSONScan(cfg *config.Config, path string) error {
	tree := scanner.NewTree(path)
	results := make(chan scanner.ScanResult, 1000)

	go scanner.Scan(path, results)

	for r := range results {
		if r.Err != nil {
			continue
		}
		tree.AddEntry(r.Entry)
	}

	matcher := scanner.NewMatcher(cfg.JunkPatterns)
	return tree.ToJSON(os.Stdout, matcher, 3)
}

var organizeCmd = &cobra.Command{
	Use:   "organize [path]",
	Short: "Organize files by type",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := filepath.Join(os.Getenv("HOME"), "Downloads")
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

		matcher := organizer.NewRuleMatcher(cfg.OrganizeRules)
		p, err := matcher.CreatePlan(absPath)
		if err != nil {
			return err
		}

		if plan || jsonOut {
			return outputPlan(p)
		}

		if dryRun {
			exec := organizer.NewExecutor(nil, true)
			return exec.Execute(p)
		}

		if apply {
			db, err := history.Open(config.DataPath())
			if err != nil {
				return err
			}
			defer db.Close()

			exec := organizer.NewExecutor(db, false)
			return exec.Execute(p)
		}

		fmt.Printf("Would organize %d files. Use --plan to see details, --dry-run to preview, or --apply to execute.\n", len(p.Files))
		return nil
	},
}

func outputPlan(p *organizer.Plan) error {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(p)
	}

	for dest, files := range p.ByDest {
		fmt.Printf("\n%s (%d files)\n", dest, len(files))
		for _, f := range files {
			fmt.Printf("  %s\n", filepath.Base(f.Source))
		}
	}
	return nil
}

var historyCmd = &cobra.Command{
	Use:   "history [query]",
	Short: "Search operation history",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := history.Open(config.DataPath())
		if err != nil {
			return err
		}
		defer db.Close()

		var ops []history.Operation

		if len(args) > 0 {
			ops, err = db.Search(args[0])
		} else if sinceDays > 0 {
			since := time.Now().AddDate(0, 0, -sinceDays)
			ops, err = db.Since(since)
		} else {
			since := time.Now().AddDate(0, 0, -7)
			ops, err = db.Since(since)
		}

		if err != nil {
			return err
		}

		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(ops)
		}

		if len(ops) == 0 {
			fmt.Println("No operations found")
			return nil
		}

		for _, op := range ops {
			fmt.Printf("%d | %s | %s | %s",
				op.ID,
				op.Timestamp.Format("2006-01-02 15:04"),
				op.Type,
				filepath.Base(op.SourcePath))
			if op.DestPath != "" {
				fmt.Printf(" -> %s", op.DestPath)
			}
			fmt.Println()
		}

		return nil
	},
}

var undoCmd = &cobra.Command{
	Use:   "undo <id>",
	Short: "Undo an operation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid operation ID: %s", args[0])
		}

		db, err := history.Open(config.DataPath())
		if err != nil {
			return err
		}
		defer db.Close()

		op, err := db.Get(id)
		if err != nil {
			return fmt.Errorf("operation not found: %d", id)
		}

		if !op.Reversible {
			return fmt.Errorf("operation %d is not reversible", id)
		}

		switch op.Type {
		case history.OpMove:
			if err := os.Rename(op.DestPath, op.SourcePath); err != nil {
				return err
			}
			fmt.Printf("Moved %s back to %s\n", op.DestPath, op.SourcePath)
		case history.OpTrash:
			if err := os.Rename(op.DestPath, op.SourcePath); err != nil {
				return err
			}
			fmt.Printf("Restored %s from trash\n", op.SourcePath)
		default:
			return fmt.Errorf("cannot undo operation type: %s", op.Type)
		}

		return nil
	},
}

var cleanCmd = &cobra.Command{
	Use:   "clean <paths...>",
	Short: "Delete files or directories",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !yesFlag {
			return fmt.Errorf("use --yes to confirm deletion")
		}

		db, err := history.Open(config.DataPath())
		if err != nil {
			return err
		}
		defer db.Close()

		cleaner := scanner.NewCleaner(db, trashFlag)

		for _, path := range args {
			absPath, err := filepath.Abs(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "skip %s: %v\n", path, err)
				continue
			}

			if err := cleaner.Delete(absPath); err != nil {
				fmt.Fprintf(os.Stderr, "failed %s: %v\n", path, err)
			} else {
				action := "deleted"
				if trashFlag {
					action = "trashed"
				}
				fmt.Printf("%s %s\n", action, absPath)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/breathe/config.yaml)")

	scanCmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	scanCmd.Flags().BoolVar(&junkOnly, "junk", false, "show only detected junk")
	rootCmd.AddCommand(scanCmd)

	organizeCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would happen")
	organizeCmd.Flags().BoolVar(&apply, "apply", false, "execute the plan")
	organizeCmd.Flags().BoolVar(&plan, "plan", false, "output plan")
	organizeCmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	rootCmd.AddCommand(organizeCmd)

	historyCmd.Flags().IntVar(&sinceDays, "since", 0, "show operations from last N days")
	historyCmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	rootCmd.AddCommand(historyCmd)

	rootCmd.AddCommand(undoCmd)

	cleanCmd.Flags().BoolVar(&yesFlag, "yes", false, "confirm deletion")
	cleanCmd.Flags().BoolVar(&trashFlag, "trash", true, "move to trash instead of permanent delete")
	cleanCmd.Flags().StringVar(&patternArg, "pattern", "", "match junk pattern name")
	rootCmd.AddCommand(cleanCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
