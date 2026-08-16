# Highway Template

Complete implementation of the Highway pattern.

## highway.go

```go
package highway

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Job is the interface all job types must implement
type Job interface {
	ID() string
	Type() string
	Run(ctx context.Context, progress chan<- Progress) error
	Marshal() ([]byte, error)
}

// JobUnmarshaler recreates a job from serialized state
type JobUnmarshaler func(data []byte) (Job, error)

// ProgressType distinguishes progress bar vs substatus updates
type ProgressType int

const (
	ProgressTypeProgress  ProgressType = iota // Known total, renders progress bar
	ProgressTypeSubStatus                     // Unknown total, renders substatus text
)

// Progress represents a status update from a job
type Progress struct {
	JobID     string       `json:"jobId"`
	Type      ProgressType `json:"type"`
	Message   string       `json:"message,omitempty"`   // Short status next to job ID
	SubStatus string       `json:"subStatus,omitempty"` // Detailed substatus (for SubStatus type)
	Current   int64        `json:"current,omitempty"`
	Total     int64        `json:"total,omitempty"`
	Extra     string       `json:"extra,omitempty"` // Extra info after percentage
	Done      bool         `json:"done,omitempty"`
	Error     error        `json:"-"`
	ErrMsg    string       `json:"error,omitempty"`
}

// Highway is the job execution engine
type Highway struct {
	workers      int
	statePath    string
	unmarshalers map[string]JobUnmarshaler

	mu           sync.Mutex
	pending      []Job
	completed    map[string]bool
	progress     chan Progress
}

// New creates a new Highway with the specified number of workers
func New(workers int, statePath string) *Highway {
	if workers < 1 {
		workers = 1
	}
	return &Highway{
		workers:      workers,
		statePath:    statePath,
		unmarshalers: make(map[string]JobUnmarshaler),
		completed:    make(map[string]bool),
		progress:     make(chan Progress, 100), // Buffered
	}
}

// RegisterType registers an unmarshaler for a job type (required for resume)
func (h *Highway) RegisterType(jobType string, unmarshal JobUnmarshaler) {
	h.unmarshalers[jobType] = unmarshal
}

// Submit adds jobs to the pending queue
func (h *Highway) Submit(jobs ...Job) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pending = append(h.pending, jobs...)
}

// Progress returns a channel that receives progress updates
func (h *Highway) Progress() <-chan Progress {
	return h.progress
}

// Run executes all pending jobs with the configured number of workers
func (h *Highway) Run(ctx context.Context) error {
	jobCh := make(chan Job)
	var wg sync.WaitGroup

	// Start workers (wg.Go handles Add/Done — Go 1.25+)
	for range h.workers {
		wg.Go(func() {
			for job := range jobCh {
				h.executeJob(ctx, job)
			}
		})
	}

	// Feed jobs to workers
	go func() {
		defer close(jobCh)

		h.mu.Lock()
		jobs := h.pending
		h.mu.Unlock()

		for _, job := range jobs {
			// Skip if already completed (resume case)
			if h.isCompleted(job.ID()) {
				continue
			}

			select {
			case <-ctx.Done():
				return // Stop feeding on cancellation
			case jobCh <- job:
			}
		}
	}()

	// Wait for workers to finish or context cancellation
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All jobs completed normally
		close(h.progress)
		h.deleteState()
		return nil
	case <-ctx.Done():
		// Interrupted - wait for feeder to stop and workers to finish
		wg.Wait()
		close(h.progress)
		return h.saveState()
	}
}

func (h *Highway) executeJob(ctx context.Context, job Job) {
	err := job.Run(ctx, h.progress)
	
	if err != nil {
		h.progress <- Progress{
			JobID:  job.ID(),
			Done:   true,
			Error:  err,
			ErrMsg: err.Error(),
		}
		h.markFailed(job.ID())
	} else {
		h.markCompleted(job.ID())
	}
}

func (h *Highway) isCompleted(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.completed[id]
}

func (h *Highway) markCompleted(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.completed[id] = true
}

func (h *Highway) markFailed(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.completed[id] = true // Skip on resume
}

// LoadState loads pending jobs from the state file
func (h *Highway) LoadState() error {
	data, err := os.ReadFile(h.statePath)
	if err != nil {
		return err
	}

	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	// Mark completed jobs
	h.mu.Lock()
	for _, id := range state.Completed {
		h.completed[id] = true
	}
	h.mu.Unlock()

	// Recreate pending jobs
	for _, pj := range state.Pending {
		unmarshal, ok := h.unmarshalers[pj.Type]
		if !ok {
			return fmt.Errorf("unknown job type: %s (register with RegisterType)", pj.Type)
		}

		job, err := unmarshal(pj.Data)
		if err != nil {
			return fmt.Errorf("failed to unmarshal job %s: %w", pj.ID, err)
		}

		h.Submit(job)
	}

	return nil
}

func (h *Highway) saveState() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Get completed IDs
	var completedIDs []string
	for id := range h.completed {
		completedIDs = append(completedIDs, id)
	}

	// Get pending jobs (not yet completed)
	var pendingJobs []persistedJob
	for _, job := range h.pending {
		if h.completed[job.ID()] {
			continue
		}

		data, err := job.Marshal()
		if err != nil {
			continue // Skip jobs that can't be marshaled
		}

		pendingJobs = append(pendingJobs, persistedJob{
			ID:   job.ID(),
			Type: job.Type(),
			Data: data,
		})
	}

	state := persistedState{
		Completed: completedIDs,
		Pending:   pendingJobs,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(h.statePath, data, 0644); err != nil {
		return err
	}

	fmt.Printf("\nState saved to %s\n", h.statePath)
	return nil
}

func (h *Highway) deleteState() {
	os.Remove(h.statePath)
}

// State persistence structures
type persistedState struct {
	Completed []string       `json:"completed"`
	Pending   []persistedJob `json:"pending"`
}

type persistedJob struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
```

---

## Usage Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"myapp/internal/highway"
	"myapp/internal/jobs"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Create highway with 3 workers
	hw := highway.New(3, ".myapp-state.json")

	// Register job types for resume capability
	hw.RegisterType("http-download", jobs.UnmarshalHTTPDownload)
	hw.RegisterType("s3-download", jobs.UnmarshalS3Download)

	// Check for resume
	if _, err := os.Stat(".myapp-state.json"); err == nil {
		fmt.Println("Resuming from saved state...")
		if err := hw.LoadState(); err != nil {
			fmt.Printf("Warning: could not load state: %v\n", err)
		}
	} else {
		// Create new jobs
		hw.Submit(
			jobs.NewHTTPDownload("https://example.com/file1.zip", "./downloads"),
			jobs.NewHTTPDownload("https://example.com/file2.zip", "./downloads"),
			jobs.NewS3Download("s3://bucket/key", "./downloads"),
		)
	}

	// Display progress in background
	go func() {
		for p := range hw.Progress() {
			if p.Error != nil {
				fmt.Printf("✗ %s: %v\n", p.JobID, p.Error)
			} else if p.Done {
				fmt.Printf("✓ %s: complete\n", p.JobID)
			} else if p.Total > 0 {
				pct := float64(p.Current) / float64(p.Total) * 100
				fmt.Printf("⏳ %s: %s (%.1f%%)\n", p.JobID, p.Message, pct)
			} else {
				fmt.Printf("⏳ %s: %s\n", p.JobID, p.Message)
			}
		}
	}()

	// Run until done or Ctrl+C
	if err := hw.Run(ctx); err != nil {
		fmt.Printf("Interrupted: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("All jobs completed!")
}
```

---

## Customization Points

### Different Progress Channel Size

```go
// In New(), change buffer size:
progress: make(chan Progress, 1000),  // Larger buffer for high-volume jobs
```

### Custom State File Location

```go
// Use a fixed location in home directory
statePath := filepath.Join(os.Getenv("HOME"), ".cache", "myapp", "state.json")
hw := highway.New(workers, statePath)
```

### Retry Failed Jobs on Resume

Change `markFailed` to not mark as completed:

```go
func (h *Highway) markFailed(id string) {
	// Don't add to completed - will be retried on resume
	// Optionally track failure count
}
```
