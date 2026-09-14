package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	mustgather "github.com/rhobs/observability-operator/must-gather"
)

func main() {
	var (
		destDir     string
		logFileName string
	)

	flag.StringVar(&destDir, "dest-dir", "/must-gather", "Destination directory for collecting must-gather data")
	flag.StringVar(&logFileName, "log-file", "gather-debug.log", "Name of the debug log file")
	flag.Parse()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create destination directory: %v\n", err)
		os.Exit(1)
	}

	logFilePath := filepath.Join(destDir, logFileName)
	logFile, err := os.Create(logFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log file: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close log file: %v\n", err)
		}
	}()

	// Tee log output to both stdout and the debug log file.
	logWriter := io.MultiWriter(os.Stdout, logFile)

	gather, err := mustgather.NewGather(destDir, logFileName, logWriter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create gather: %v\n", err)
		os.Exit(1)
	}

	if err := gather.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Must-gather failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Must-gather completed successfully")
}
