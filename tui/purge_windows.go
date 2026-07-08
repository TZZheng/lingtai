//go:build windows

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anthropics/lingtai-tui/internal/processscan"
)

func purgeMain() {
	// Optional dir filter from os.Args[2]
	var filterDir string
	if len(os.Args) > 2 {
		filterDir, _ = filepath.Abs(os.Args[2])
	}

	procs := purgeProcsFromAgentProcesses(processscan.FindAllAgentProcesses(), filterDir, os.Getpid())

	if len(procs) == 0 {
		if filterDir != "" {
			fmt.Printf("No lingtai processes found in %s.\n", filterDir)
		} else {
			fmt.Println("No lingtai processes found.")
		}
		return
	}

	scope := "ALL"
	if filterDir != "" {
		scope = filterDir
	}
	fmt.Printf("%-8s %-30s %s\n", "PID", "AGENT", "DIRECTORY")
	for _, p := range procs {
		fmt.Printf("%-8d %-30s %s\n", p.pid, p.agent, p.dir)
	}
	fmt.Printf("\n%d process(es) in %s. Kill all? [y/N] ", len(procs), scope)

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("Aborted.")
		return
	}

	killed := 0
	for _, p := range procs {
		cmd := exec.Command("taskkill", "/F", "/PID", fmt.Sprint(p.pid))
		if cmd.Run() == nil {
			killed++
		}
	}

	fmt.Printf("Purged %d process(es).\n", killed)
}
