# Display Template

Complete implementation of the terminal display manager for job progress.

## display.go

```go
package display

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/[GITHUB_USER]/[REPO_NAME]/utils"
)

// ANSI standard colors (indices 0-15) adapt to the user's terminal theme.
// Dracula users see Dracula colors, Catppuccin users see Catppuccin, etc.
var (
	colorBlue    = lipgloss.ANSIColor(12) // Bright Blue  - titles, progress fill
	colorGreen   = lipgloss.ANSIColor(10) // Bright Green - running, success
	colorRed     = lipgloss.ANSIColor(9)  // Bright Red   - errors, failures
	colorMagenta = lipgloss.ANSIColor(13) // Bright Magenta - substatus accent
	colorCyan    = lipgloss.ANSIColor(14) // Bright Cyan  - percentages, secondary accent
	colorFg      = lipgloss.ANSIColor(15) // Bright White - primary text
	colorMuted   = lipgloss.ANSIColor(7)  // White        - secondary/dimmed text
	colorChrome  = lipgloss.ANSIColor(8)  // Bright Black - borders, pending, dim UI
)

// ProgressType distinguishes progress bar vs substatus updates
type ProgressType int

const (
	ProgressTypeProgress  ProgressType = iota // Known total, renders progress bar
	ProgressTypeSubStatus                     // Unknown total, renders substatus text
)

// JobStatus represents the execution state of a job
type JobStatus int

const (
	StatusPending JobStatus = iota
	StatusRunning
	StatusCompleted
	StatusFailed
)

// Progress is what jobs send through the progress channel
type Progress struct {
	JobID     string
	Type      ProgressType
	Message   string // Short status shown in brackets next to job ID
	SubStatus string // For SubStatus type: the substatus line
	Current   int64  // For progress: current count
	Total     int64  // For progress: total count
	Extra     string // Extra info shown after percentage
	Done      bool   // Job completed
	Error     error  // Job failed
}

// JobState represents current state of a job for display
type JobState struct {
	ID         string
	Status     JobStatus
	UpdateType ProgressType
	Message    string
	SubStatus  string
	Current    int64
	Total      int64
	Extra      string
}

// Config holds display settings
type Config struct {
	MaxVisibleJobs int           // Max jobs to show in detail (default 5)
	RefreshRate    time.Duration // How often to refresh (default 200ms)
	BoxWidth       int           // Width of the display box (default 72)
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxVisibleJobs: 5,
		RefreshRate:    200 * time.Millisecond,
		BoxWidth:       72,
	}
}

// Display manages the terminal output for job progress
type Display struct {
	config Config

	mu        sync.RWMutex
	jobs      map[string]*JobState
	order     []string // Maintains insertion order
	running   []string
	pending   []string
	completed []string
	failed    []string

	lastLineCount int
	stopCh        chan struct{}
	doneCh        chan struct{}
}

// New creates a new display manager
func New(config Config) *Display {
	return &Display{
		config: config,
		jobs:   make(map[string]*JobState),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// RegisterJob registers a job before it starts (shows as pending)
func (d *Display) RegisterJob(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.jobs[id]; !exists {
		d.jobs[id] = &JobState{
			ID:     id,
			Status: StatusPending,
		}
		d.order = append(d.order, id)
		d.pending = append(d.pending, id)
	}
}

// Update processes a job update
func (d *Display) Update(update Progress) {
	d.mu.Lock()
	defer d.mu.Unlock()

	job, exists := d.jobs[update.JobID]
	if !exists {
		job = &JobState{ID: update.JobID}
		d.jobs[update.JobID] = job
		d.order = append(d.order, update.JobID)
	}

	if update.Done {
		if update.Error != nil {
			job.Status = StatusFailed
			job.Message = update.Error.Error()
			d.removeFromSlice(&d.running, update.JobID)
			d.removeFromSlice(&d.pending, update.JobID)
			d.failed = append(d.failed, update.JobID)
		} else {
			job.Status = StatusCompleted
			job.Message = "Done"
			d.removeFromSlice(&d.running, update.JobID)
			d.removeFromSlice(&d.pending, update.JobID)
			d.completed = append(d.completed, update.JobID)
		}
	} else {
		if job.Status == StatusPending {
			d.removeFromSlice(&d.pending, update.JobID)
			d.running = append(d.running, update.JobID)
		}
		job.Status = StatusRunning
		job.UpdateType = update.Type
		job.Message = update.Message
		job.SubStatus = update.SubStatus
		job.Current = update.Current
		job.Total = update.Total
		job.Extra = update.Extra
	}
}

func (d *Display) removeFromSlice(slice *[]string, id string) {
	for i, v := range *slice {
		if v == id {
			*slice = append((*slice)[:i], (*slice)[i+1:]...)
			return
		}
	}
}

// Start begins the display refresh loop
// In AI mode, prints sequential plain-text lines instead of the interactive TUI
func (d *Display) Start(updates <-chan Progress) {
	if utils.GlobalForAIFlag {
		d.startAI(updates)
		return
	}

	// Process updates in background
	go func() {
		for update := range updates {
			d.Update(update)
		}
	}()

	// Render loop
	go func() {
		defer close(d.doneCh)
		ticker := time.NewTicker(d.config.RefreshRate)
		defer ticker.Stop()

		for {
			select {
			case <-d.stopCh:
				d.clearDisplay()
				d.renderFinal()
				return
			case <-ticker.C:
				d.render()
			}
		}
	}()
}

// startAI processes updates and prints plain-text lines for AI agents
func (d *Display) startAI(updates <-chan Progress) {
	go func() {
		for update := range updates {
			d.Update(update)
			d.printAIUpdate(update)
		}
	}()

	go func() {
		defer close(d.doneCh)
		<-d.stopCh
		d.renderFinal()
	}()
}

// printAIUpdate prints a single progress event as a plain-text line
func (d *Display) printAIUpdate(p Progress) {
	if p.Done {
		if p.Error != nil {
			fmt.Printf("[ERROR] %s: %v\n", p.JobID, p.Error)
		} else {
			fmt.Printf("[OK] %s: Done\n", p.JobID)
		}
		return
	}

	if p.Type == ProgressTypeProgress && p.Total > 0 {
		pct := int(float64(p.Current) / float64(p.Total) * 100)
		extra := ""
		if p.Extra != "" {
			extra = " (" + p.Extra + ")"
		}
		fmt.Printf("[INFO] %s: %s %d%%%s\n", p.JobID, p.Message, pct, extra)
	} else if p.Type == ProgressTypeSubStatus {
		sub := p.SubStatus
		if sub == "" {
			sub = p.Message
		}
		fmt.Printf("[INFO] %s: %s\n", p.JobID, sub)
	}
}

// Stop stops the display and shows final state
func (d *Display) Stop() {
	close(d.stopCh)
	<-d.doneCh
}

func (d *Display) clearDisplay() {
	if d.lastLineCount > 0 {
		for i := 0; i < d.lastLineCount; i++ {
			fmt.Print("\033[A") // Move up
			fmt.Print("\033[K") // Clear line
		}
	}
}

func (d *Display) render() {
	d.mu.RLock()
	defer d.mu.RUnlock()

	d.clearDisplay()

	lines := d.buildDisplay()
	output := strings.Join(lines, "\n")
	fmt.Println(output)

	d.lastLineCount = len(lines)
}

func (d *Display) renderFinal() {
	d.mu.RLock()
	defer d.mu.RUnlock()

	total := len(d.jobs)
	completedCount := len(d.completed)
	failedCount := len(d.failed)

	if utils.GlobalForAIFlag {
		if failedCount == 0 {
			fmt.Printf("[OK] All %d jobs completed successfully\n", total)
		} else {
			fmt.Printf("[OK] %d completed  [ERROR] %d failed\n", completedCount, failedCount)
		}
		return
	}

	successStyle := lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	failStyle := lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(colorMuted)

	if failedCount == 0 {
		fmt.Println(successStyle.Render(fmt.Sprintf("✓ All %d jobs completed successfully", total)))
	} else {
		fmt.Println(successStyle.Render(fmt.Sprintf("✓ %d completed", completedCount)) +
			dimStyle.Render("  ") +
			failStyle.Render(fmt.Sprintf("✗ %d failed", failedCount)))
	}
}

func (d *Display) buildDisplay() []string {
	var lines []string

	total := len(d.jobs)
	boxWidth := d.config.BoxWidth
	innerWidth := boxWidth - 4

	// Styles
	borderStyle := lipgloss.NewStyle().Foreground(colorChrome)
	titleStyle := lipgloss.NewStyle().Foreground(colorBlue).Bold(true)
	runningStyle := lipgloss.NewStyle().Foreground(colorGreen)
	pendingStyle := lipgloss.NewStyle().Foreground(colorChrome)
	jobIDStyle := lipgloss.NewStyle().Foreground(colorFg)
	messageStyle := lipgloss.NewStyle().Foreground(colorMuted)
	substatusStyle := lipgloss.NewStyle().Foreground(colorMagenta)
	progressFillStyle := lipgloss.NewStyle().Foreground(colorBlue)
	progressEmptyStyle := lipgloss.NewStyle().Foreground(colorChrome)
	percentStyle := lipgloss.NewStyle().Foreground(colorCyan)
	extraStyle := lipgloss.NewStyle().Foreground(colorMuted)

	// Top border with title
	title := fmt.Sprintf(" Processing %d jobs ", total)
	leftPad := 2
	rightPad := boxWidth - leftPad - len(title) - 2
	if rightPad < 0 {
		rightPad = 0
	}
	topBorder := borderStyle.Render("┌"+strings.Repeat("─", leftPad)) +
		titleStyle.Render(title) +
		borderStyle.Render(strings.Repeat("─", rightPad)+"┐")
	lines = append(lines, topBorder)

	// Empty line
	lines = append(lines, d.emptyLine(boxWidth, borderStyle))

	// Running jobs
	visibleCount := 0
	for _, jobID := range d.running {
		if visibleCount >= d.config.MaxVisibleJobs {
			break
		}

		job := d.jobs[jobID]
		visibleCount++

		// Line 1: Job ID + [Message]
		msgPart := ""
		if job.Message != "" {
			msgPart = " [" + job.Message + "]"
		}
		maxIDLen := innerWidth - 4 - len(msgPart)
		if maxIDLen < 10 {
			maxIDLen = 10
			msgPart = truncateString(msgPart, innerWidth-4-maxIDLen)
		}
		truncatedID := truncateString(job.ID, maxIDLen)

		jobLine := "  " + runningStyle.Render("●") + " " +
			jobIDStyle.Render(truncatedID) +
			messageStyle.Render(msgPart)
		lines = append(lines, d.padLine(jobLine, boxWidth, borderStyle))

		// Line 2: Progress bar OR Substatus
		if job.UpdateType == ProgressTypeProgress && job.Total > 0 {
			extraInfo := ""
			if job.Extra != "" {
				extraInfo = "  " + job.Extra
			}
			extraLen := len(extraInfo)

			progressWidth := innerWidth - 4 - 1 - 1 - 6 - extraLen
			if progressWidth < 10 {
				progressWidth = 10
				extraInfo = truncateString(extraInfo, innerWidth-4-1-1-6-progressWidth)
			}

			pct := float64(job.Current) / float64(job.Total)
			filled := int(pct * float64(progressWidth))
			empty := progressWidth - filled

			progressLine := "    " +
				progressFillStyle.Render("●"+strings.Repeat("━", filled)) +
				progressEmptyStyle.Render(strings.Repeat(" ", empty)) +
				progressFillStyle.Render("●") +
				" " + percentStyle.Render(fmt.Sprintf("%3d%%", int(pct*100))) +
				extraStyle.Render(extraInfo)
			lines = append(lines, d.padLine(progressLine, boxWidth, borderStyle))
		} else if job.UpdateType == ProgressTypeSubStatus {
			subText := job.SubStatus
			if subText == "" {
				subText = job.Message
			}
			subLine := "    " + substatusStyle.Render("└─ "+truncateString(subText, innerWidth-10))
			lines = append(lines, d.padLine(subLine, boxWidth, borderStyle))
		}

		lines = append(lines, d.emptyLine(boxWidth, borderStyle))
	}

	// Pending jobs
	pendingToShow := d.config.MaxVisibleJobs - visibleCount
	if pendingToShow > 3 {
		pendingToShow = 3
	}
	for i := 0; i < pendingToShow && i < len(d.pending); i++ {
		jobID := d.pending[i]
		truncatedID := truncateString(jobID, innerWidth-14)
		pendingLine := "  " + pendingStyle.Render("○ "+truncatedID+" (queued)")
		lines = append(lines, d.padLine(pendingLine, boxWidth, borderStyle))
	}

	remainingPending := len(d.pending) - pendingToShow
	if remainingPending > 0 {
		moreLine := "    " + pendingStyle.Render(fmt.Sprintf("... and %d more queued", remainingPending))
		lines = append(lines, d.padLine(moreLine, boxWidth, borderStyle))
	}

	lines = append(lines, d.emptyLine(boxWidth, borderStyle))

	// Separator
	sepLine := borderStyle.Render("│  " + strings.Repeat("─", innerWidth-2) + "  │")
	lines = append(lines, sepLine)

	// Summary
	runCount := lipgloss.NewStyle().Foreground(colorGreen).Render(fmt.Sprintf("● %d running", len(d.running)))
	pendCount := lipgloss.NewStyle().Foreground(colorChrome).Render(fmt.Sprintf("○ %d pending", len(d.pending)))
	compCount := lipgloss.NewStyle().Foreground(colorGreen).Render(fmt.Sprintf("✓ %d completed", len(d.completed)))
	failCount := lipgloss.NewStyle().Foreground(colorRed).Render(fmt.Sprintf("✗ %d failed", len(d.failed)))
	summary := "  " + runCount + "  " + pendCount + "  " + compCount + "  " + failCount
	lines = append(lines, d.padLine(summary, boxWidth, borderStyle))

	// Bottom border
	bottomBorder := borderStyle.Render("└" + strings.Repeat("─", boxWidth-2) + "┘")
	lines = append(lines, bottomBorder)

	return lines
}

func (d *Display) emptyLine(boxWidth int, borderStyle lipgloss.Style) string {
	return borderStyle.Render("│") + strings.Repeat(" ", boxWidth-2) + borderStyle.Render("│")
}

func (d *Display) padLine(content string, boxWidth int, borderStyle lipgloss.Style) string {
	visibleLen := lipgloss.Width(content)
	padding := boxWidth - visibleLen - 2

	if padding < 0 {
		padding = 0
	}

	return borderStyle.Render("│") + content + strings.Repeat(" ", padding) + borderStyle.Render("│")
}

func truncateString(s string, maxLen int) string {
	if maxLen <= 3 {
		return s
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
```

---

## Usage Example

```go
package main

import (
	"context"
	"os"
	"os/signal"

	"myapp/internal/display"
	"myapp/internal/highway"
	"myapp/internal/jobs"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Create highway
	hw := highway.New(3, ".myapp-state.json")
	hw.RegisterType("http-download", jobs.UnmarshalHTTPDownload)

	// Create display
	disp := display.New(display.DefaultConfig())

	// Create and register jobs
	urls := []string{
		"https://example.com/file1.zip",
		"https://example.com/file2.zip",
		"https://example.com/file3.zip",
	}
	for _, url := range urls {
		job := jobs.NewHTTPDownload(url, "./downloads")
		disp.RegisterJob(job.ID())
		hw.Submit(job)
	}

	// Start display (consumes progress channel)
	disp.Start(hw.Progress())

	// Run until done or Ctrl+C
	hw.Run(ctx)

	// Stop display (shows final summary)
	disp.Stop()
}
```

---

## Job Progress Examples

### Progress Type (Known Total)

```go
func (j *HTTPDownloadJob) Run(ctx context.Context, progress chan<- Progress) error {
	for chunk := range chunks {
		downloaded += chunk.Size
		
		progress <- Progress{
			JobID:   j.ID(),
			Type:    ProgressTypeProgress,
			Message: "Downloading",
			Current: downloaded,
			Total:   j.TotalSize,
			Extra:   fmt.Sprintf("%dMB/%dMB", downloaded/1024/1024, j.TotalSize/1024/1024),
		}
	}
	
	progress <- Progress{JobID: j.ID(), Done: true}
	return nil
}
```

### SubStatus Type (Unknown Total)

```go
func (j *EC2AuditJob) Run(ctx context.Context, progress chan<- Progress) error {
	for _, instance := range instances {
		progress <- Progress{
			JobID:     j.ID(),
			Type:      ProgressTypeSubStatus,
			Message:   "Auditing",
			SubStatus: fmt.Sprintf("Checking %s", instance.ID),
		}
		
		j.auditInstance(instance)
	}
	
	progress <- Progress{JobID: j.ID(), Done: true}
	return nil
}
```

---

## Customization

### Adjust Visible Jobs

```go
config := display.DefaultConfig()
config.MaxVisibleJobs = 8  // Show more jobs
config.BoxWidth = 80       // Wider box
disp := display.New(config)
```

### Faster/Slower Refresh

```go
config.RefreshRate = 100 * time.Millisecond  // Faster updates
config.RefreshRate = 500 * time.Millisecond  // Slower, less CPU
```
