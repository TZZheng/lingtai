//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/lingtai-tui/internal/processscan"
)

func listMain() {
	opts, err := parseListArgs(os.Args[2:])
	if err != nil {
		listUsageError(err)
	}

	found, err := processscan.FindAllAgentProcesses()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running ps: %v\n", err)
		os.Exit(1)
	}
	procs := listProcsFromAgentProcesses(found, opts.FilterDir, os.Getpid())
	for i := range procs {
		procs[i].Uptime = humanUptimeFromEtime(procs[i].Uptime)
	}

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
	printList(os.Stdout, procs, phantomDirs, opts, true)
	fmt.Printf("\n%d process(es) running.\n", len(procs))
	printListWarnings(os.Stdout, phantomDirs, opts.FilterDir)
}

// humanUptimeFromEtime converts a ps etime value ([[dd-]hh:]mm:ss) into the
// human-readable uptime the UPTIME column has always shown ("2d 3h", "1h 2m",
// "4m 9s"). Unparseable values are shown as-is rather than guessed.
func humanUptimeFromEtime(etime string) string {
	secs, ok := parseEtimeSeconds(etime)
	if !ok {
		return etime
	}
	d := time.Duration(secs) * time.Second
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%dd %dh", int(d.Hours())/24, int(d.Hours())%24)
	case d >= time.Hour:
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
}

func parseEtimeSeconds(etime string) (int, bool) {
	etime = strings.TrimSpace(etime)
	days := 0
	if day, rest, ok := strings.Cut(etime, "-"); ok {
		d, err := strconv.Atoi(day)
		if err != nil || d < 0 {
			return 0, false
		}
		days = d
		etime = rest
	}
	parts := strings.Split(etime, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	secs := 0
	for _, part := range parts {
		v, err := strconv.Atoi(part)
		if err != nil || v < 0 {
			return 0, false
		}
		secs = secs*60 + v
	}
	return days*24*60*60 + secs, true
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
