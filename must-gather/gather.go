package mustgather

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/rhobs/observability-operator/must-gather/internal/api"
	"github.com/rhobs/observability-operator/must-gather/internal/client"
	"github.com/rhobs/observability-operator/must-gather/internal/monitoring"
)

// Gather is the top-level must-gather orchestrator.
type Gather struct {
	config *api.Config
	client *client.Client
	logger api.Logger
}

// NewGather creates a new must-gather orchestrator writing to baseCollectionPath.
func NewGather(baseCollectionPath, logFileName string, logWriter io.Writer) (*Gather, error) {
	logger := api.NewLogger(logWriter)

	k8sClient, err := client.NewClient(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	absPath, err := filepath.Abs(baseCollectionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve destination path: %w", err)
	}

	config := &api.Config{
		DestDir:     api.NewPath(absPath),
		LogFileName: logFileName,
		Logger:      logWriter,
	}

	return &Gather{
		config: config,
		client: k8sClient,
		logger: logger,
	}, nil
}

// Run executes the must-gather collection.
func (g *Gather) Run(ctx context.Context) error {
	g.logger.Log("..... observability-operator must-gather started .....")
	g.logger.Log("must-gather logs are located at: '%s'", filepath.Join(g.config.DestDir.String(), g.config.LogFileName))

	if err := g.config.DestDir.MkdirAll(); err != nil {
		return err
	}

	collectors := g.createCollectors()
	results := g.runCollectors(ctx, collectors)
	g.logResults(results)

	var failures []error
	for _, result := range results {
		if result.Error != nil {
			failures = append(failures, fmt.Errorf("%s: %w", result.CollectorName, result.Error))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("one or more collectors failed: %v", failures)
	}

	return nil
}

// createCollectors builds the set of collectors to run.
func (g *Gather) createCollectors() []api.Collector {
	return []api.Collector{
		monitoring.NewCollector(g.client, g.logger, g.config.DestDir),
	}
}

// runCollectors runs all collectors concurrently and returns their results.
func (g *Gather) runCollectors(ctx context.Context, collectors []api.Collector) []api.Result {
	var wg sync.WaitGroup
	resultsChan := make(chan api.Result, len(collectors))

	for _, collector := range collectors {
		wg.Add(1)
		go func(c api.Collector) {
			defer wg.Done()

			start := time.Now()
			err := c.Collect(ctx)
			resultsChan <- api.Result{
				CollectorName: c.Name(),
				Error:         err,
				Duration:      time.Since(start),
			}
		}(collector)
	}

	wg.Wait()
	close(resultsChan)

	results := make([]api.Result, 0, len(collectors))
	for result := range resultsChan {
		results = append(results, result)
	}
	return results
}

// logResults logs a summary of every collector run.
func (g *Gather) logResults(results []api.Result) {
	g.logger.Log("=== Must-gather collection complete ===")
	for _, result := range results {
		if result.Error != nil {
			g.logger.Log("FAILED: %s (took %v): %v", result.CollectorName, result.Duration, result.Error)
		} else {
			g.logger.Log("SUCCESS: %s (took %v)", result.CollectorName, result.Duration)
		}
	}
}
