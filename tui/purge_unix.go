//go:build !windows

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/anthropics/lingtai-tui/internal/processscan"
)

func purgeMain() {
	// Optional dir filter from os.Args[2]
	var filterDir string
	if len(os.Args) > 2 {
		filterDir, _ = filepath.Abs(os.Args[2])
	}

	found, err := processscan.FindAllAgentProcesses()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running ps: %v\n", err)
		os.Exit(1)
	}
	procs := purgeProcsFromAgentProcesses(found, filterDir, os.Getpid())

	if len(procs) == 0 {
		if filterDir != "" {
			fmt.Printf("No lingtai processes found in %s.\n", filterDir)
		} else {
			fmt.Println("No lingtai processes found.")
		}
		return
	}

	// List matching processes
	scope := "ALL"
	if filterDir != "" {
		scope = filterDir
	}
	fmt.Printf("%-8s %-30s %s\n", "PID", "AGENT", "DIRECTORY")
	for _, p := range procs {
		fmt.Printf("%-8d %-30s %s\n", p.pid, p.agent, p.dir)
	}
	fmt.Printf("\n%d process(es) in %s. Kill all? [y/N] ", len(procs), scope)

	// Wait for confirmation
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("Aborted.")
		return
	}

	// SIGTERM first
	for _, p := range procs {
		if proc, err := os.FindProcess(p.pid); err == nil {
			proc.Signal(syscall.SIGTERM)
		}
	}
	time.Sleep(2 * time.Second)

	// SIGKILL survivors
	killed := 0
	for _, p := range procs {
		if proc, err := os.FindProcess(p.pid); err == nil {
			if proc.Signal(syscall.Signal(0)) == nil {
				proc.Signal(syscall.SIGKILL)
			}
		}
		killed++
	}

	fmt.Printf("Purged %d process(es).\n", killed)
}
