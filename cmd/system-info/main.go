// Package main contains the entry point for the system-info application.
package main

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
	"nginx-reaper/internal/procps"
	"os"
)

func main() {
	processes, err := process.Processes()
	if err != nil {
		panic(err)
	}
	fmt.Println("Processes information:")
	for _, proc := range processes {
		fmt.Println(procps.NewProcessInfo(proc))
	}
	fmt.Println()

	fmt.Println("Memory information:")
	fmt.Println(procps.NewMemoryInfo(os.Getpid()))
}
