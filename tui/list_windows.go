//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/lingtai-tui/internal/processscan"
)

func listMain() {
	opts, err := parseListArgs(os.Args[2:])
	if err != nil {
		listUsageError(err)
	}

	found, err := processscan.FindAllAgentProcesses()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing processes: %v\n", err)
		os.Exit(1)
	}
	procs := listProcsFromAgentProcesses(found, opts.FilterDir, os.Getpid())

	if len(procs) == 0 {
		if opts.JSON {
			printListJSON(os.Stdout, procs, nil, opts)
			return
		}
		if opts.FilterDir != "" {
			fmt.Printf("No lingtai processes running in %s.\n", opts.FilterDir)
		} else {
			fmt.Println("No lingtai processes running.")
		}
		return
	}

	phantomDirs := detectPhantomDirs(procs, opts.FilterDir)
	annotateListProcs(procs)
	procs = collapseListProcsByAgentDir(procs)
	if opts.JSON {
		printListJSON(os.Stdout, procs, phantomDirs, opts)
		return
	}
	printList(os.Stdout, procs, phantomDirs, opts, false)
	fmt.Printf("\n%d process(es) running.\n", len(procs))
	printListWarnings(os.Stdout, phantomDirs, opts.FilterDir)
}

func detectPhantomDirs(procs []listProc, filterDir string) map[string]bool {
	phantomDirs := map[string]bool{}
	if filterDir != "" {
		lingtaiDir := filepath.Join(filterDir, ".lingtai")
		if _, err := os.Stat(lingtaiDir); os.IsNotExist(err) {
			phantomDirs[filterDir] = true
		}
		return phantomDirs
	}

	seen := map[string]bool{}
	for _, p := range procs {
		if p.Project == "" || seen[p.Project] {
			continue
		}
		seen[p.Project] = true
		lingtaiDir := filepath.Join(p.Project, ".lingtai")
		if _, err := os.Stat(lingtaiDir); os.IsNotExist(err) {
			phantomDirs[p.Project] = true
		}
	}
	return phantomDirs
}

func agentDirFromWindowsCommandLine(cmdline string) string {
	agentDir, ok := processscan.ExtractAgentDir(cmdline)
	if !ok {
		return ""
	}
	return agentDir
}
