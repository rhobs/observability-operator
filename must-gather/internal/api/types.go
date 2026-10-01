package api

import (
	"context"
	"io"
	"time"
)

// Config holds the configuration for a must-gather collection run.
type Config struct {
	// DestDir is the root directory where all collected data will be stored.
	DestDir Path

	// LogFileName is the name of the debug log file.
	LogFileName string

	// Logger is where log output should be written.
	Logger io.Writer
}

// Collector defines the interface for all must-gather collectors.
type Collector interface {
	// Collect performs the collection and returns an error if it fails.
	Collect(ctx context.Context) error

	// Name returns the name of this collector.
	Name() string
}

// Result represents the outcome of a single collector run.
type Result struct {
	CollectorName string
	Error         error
	Duration      time.Duration
}
