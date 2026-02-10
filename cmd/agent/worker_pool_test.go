package main

import (
	"fmt"
	"testing"
)

// Test StartWorkers function
func TestStartWorkers(t *testing.T) {
	jobs := make(chan Job, 2)

	// Start workers
	StartWorkers(2, jobs)

	// Send jobs
	job1 := func() error {
		return nil
	}
	job2 := func() error {
		return nil
	}

	jobs <- job1
	jobs <- job2

	// Close jobs channel to finish workers
	close(jobs)
}

// Test StartWorkers with invalid number of workers
func TestStartWorkersWithInvalidWorkerCount(t *testing.T) {
	jobs := make(chan Job, 2)

	// Start workers with zero count
	StartWorkers(0, jobs)

	// Send job
	job := func() error {
		return nil
	}

	jobs <- job

	// Close jobs channel to finish workers
	close(jobs)
}

// Test job failure
func TestJobFailure(t *testing.T) {
	jobs := make(chan Job, 1)

	// Start workers
	StartWorkers(1, jobs)

	// Send failing job
	job := func() error {
		return fmt.Errorf("job failed")
	}

	jobs <- job

	// Close jobs channel to finish workers
	close(jobs)
}
