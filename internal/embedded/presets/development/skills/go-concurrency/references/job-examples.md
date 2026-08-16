# Job Examples

Example implementations of concrete job types.

## Simple Job (Config Only)

A job that just needs configuration to run. No partial progress state.

```go
// internal/jobs/s3_scan.go

package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"myapp/internal/highway"
)

type S3PublicAccessJob struct {
	Region    string `json:"region"`
	OutputDir string `json:"outputDir"`
	
	// Not serialized - created at runtime
	client *s3.Client
}

func NewS3PublicAccessJob(region, outputDir string) *S3PublicAccessJob {
	return &S3PublicAccessJob{
		Region:    region,
		OutputDir: outputDir,
	}
}

func (j *S3PublicAccessJob) ID() string {
	return "s3-public-access-" + j.Region
}

func (j *S3PublicAccessJob) Type() string {
	return "s3-public-access"
}

func (j *S3PublicAccessJob) Run(ctx context.Context, progress chan<- highway.Progress) error {
	// Initialize client
	if err := j.initClient(ctx); err != nil {
		return err
	}

	progress <- highway.Progress{
		JobID:   j.ID(),
		Type:    highway.ProgressTypeSubStatus,
		Message: "Listing buckets...",
	}

	buckets, err := j.listBuckets(ctx)
	if err != nil {
		return fmt.Errorf("list buckets: %w", err)
	}

	results := make([]BucketResult, 0, len(buckets))

	for i, bucket := range buckets {
		progress <- highway.Progress{
			JobID:   j.ID(),
			Message: fmt.Sprintf("Checking %s", bucket),
			Current: int64(i + 1),
			Total:   int64(len(buckets)),
		}

		result := j.checkPublicAccess(ctx, bucket)
		results = append(results, result)
	}

	// Write results
	if err := j.writeResults(results); err != nil {
		return fmt.Errorf("write results: %w", err)
	}

	progress <- highway.Progress{
		JobID: j.ID(),
		Done:  true,
	}

	return nil
}

func (j *S3PublicAccessJob) Marshal() ([]byte, error) {
	return json.Marshal(j)
}

func UnmarshalS3PublicAccess(data []byte) (highway.Job, error) {
	var job S3PublicAccessJob
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// Private methods
func (j *S3PublicAccessJob) initClient(ctx context.Context) error {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(j.Region))
	if err != nil {
		return err
	}
	j.client = s3.NewFromConfig(cfg)
	return nil
}

func (j *S3PublicAccessJob) listBuckets(ctx context.Context) ([]string, error) {
	out, err := j.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	
	var names []string
	for _, b := range out.Buckets {
		names = append(names, *b.Name)
	}
	return names, nil
}

func (j *S3PublicAccessJob) checkPublicAccess(ctx context.Context, bucket string) BucketResult {
	// Check public access block, bucket policy, ACLs...
	// Implementation details...
	return BucketResult{Bucket: bucket}
}

func (j *S3PublicAccessJob) writeResults(results []BucketResult) error {
	// Write to j.OutputDir
	return nil
}

type BucketResult struct {
	Bucket       string `json:"bucket"`
	IsPublic     bool   `json:"isPublic"`
	PublicReason string `json:"publicReason,omitempty"`
}
```

---

## Resumable Job (With Partial Progress)

A job that tracks partial progress for proper resume.

```go
// internal/jobs/http_download.go

package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"myapp/internal/highway"
)

type HTTPDownloadJob struct {
	// Config (always saved)
	URL       string `json:"url"`
	OutputDir string `json:"outputDir"`
	Filename  string `json:"filename"`

	// Discovered during first run (saved for resume)
	TotalSize int64 `json:"totalSize"`
	PartSize  int64 `json:"partSize"`

	// Resume state (which parts are done)
	CompletedParts []int `json:"completedParts"`
}

func NewHTTPDownload(url, outputDir string) *HTTPDownloadJob {
	filename := filepath.Base(url)
	return &HTTPDownloadJob{
		URL:            url,
		OutputDir:      outputDir,
		Filename:       filename,
		PartSize:       10 * 1024 * 1024, // 10MB default
		CompletedParts: []int{},
	}
}

func (j *HTTPDownloadJob) ID() string {
	return "http-" + j.Filename
}

func (j *HTTPDownloadJob) Type() string {
	return "http-download"
}

func (j *HTTPDownloadJob) Run(ctx context.Context, progress chan<- highway.Progress) error {
	// First run: discover total size if not known
	if j.TotalSize == 0 {
		size, err := j.getContentLength(ctx)
		if err != nil {
			return fmt.Errorf("get content length: %w", err)
		}
		j.TotalSize = size
	}

	// Calculate parts
	totalParts := int((j.TotalSize + j.PartSize - 1) / j.PartSize)

	// Create output directory
	if err := os.MkdirAll(j.OutputDir, 0755); err != nil {
		return err
	}

	// Download each part
	for partNum := 0; partNum < totalParts; partNum++ {
		// Skip already completed parts (resume!)
		if j.isPartComplete(partNum) {
			continue
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Calculate byte range
		start := int64(partNum) * j.PartSize
		end := start + j.PartSize - 1
		if end >= j.TotalSize {
			end = j.TotalSize - 1
		}

		// Download this range
		if err := j.downloadRange(ctx, partNum, start, end); err != nil {
			return fmt.Errorf("download part %d: %w", partNum, err)
		}

		// Mark as complete (important: do this AFTER successful download)
		j.CompletedParts = append(j.CompletedParts, partNum)

		// Send progress
		progress <- highway.Progress{
			JobID:   j.ID(),
			Message: fmt.Sprintf("Part %d/%d", len(j.CompletedParts), totalParts),
			Current: int64(len(j.CompletedParts)),
			Total:   int64(totalParts),
		}
	}

	// All parts done, merge them
	if err := j.mergeParts(totalParts); err != nil {
		return fmt.Errorf("merge parts: %w", err)
	}

	progress <- highway.Progress{
		JobID: j.ID(),
		Done:  true,
	}

	return nil
}

func (j *HTTPDownloadJob) Marshal() ([]byte, error) {
	// This includes CompletedParts for proper resume
	return json.Marshal(j)
}

func UnmarshalHTTPDownload(data []byte) (highway.Job, error) {
	var job HTTPDownloadJob
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// Private methods

func (j *HTTPDownloadJob) isPartComplete(partNum int) bool {
	for _, p := range j.CompletedParts {
		if p == partNum {
			return true
		}
	}
	return false
}

func (j *HTTPDownloadJob) getContentLength(ctx context.Context) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", j.URL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid content-length")
	}

	return size, nil
}

func (j *HTTPDownloadJob) downloadRange(ctx context.Context, partNum int, start, end int64) error {
	req, err := http.NewRequestWithContext(ctx, "GET", j.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	partPath := filepath.Join(j.OutputDir, fmt.Sprintf("%s.part%d", j.Filename, partNum))
	f, err := os.Create(partPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func (j *HTTPDownloadJob) mergeParts(totalParts int) error {
	finalPath := filepath.Join(j.OutputDir, j.Filename)
	out, err := os.Create(finalPath)
	if err != nil {
		return err
	}
	defer out.Close()

	for i := 0; i < totalParts; i++ {
		partPath := filepath.Join(j.OutputDir, fmt.Sprintf("%s.part%d", j.Filename, i))
		
		part, err := os.Open(partPath)
		if err != nil {
			return err
		}
		
		_, err = io.Copy(out, part)
		part.Close()
		if err != nil {
			return err
		}

		// Clean up part file
		os.Remove(partPath)
	}

	return nil
}
```

---

## Generic Function Job

When you want to wrap an arbitrary function as a job (less common, but useful for one-offs).

```go
// internal/jobs/func_job.go

package jobs

import (
	"context"
	"encoding/json"

	"myapp/internal/highway"
)

type FuncJob struct {
	id   string
	typ  string
	fn   func(ctx context.Context, progress chan<- highway.Progress) error
	data json.RawMessage // For serialization
}

func NewFuncJob(id, typ string, fn func(ctx context.Context, progress chan<- highway.Progress) error) *FuncJob {
	return &FuncJob{
		id:  id,
		typ: typ,
		fn:  fn,
	}
}

func (j *FuncJob) ID() string   { return j.id }
func (j *FuncJob) Type() string { return j.typ }

func (j *FuncJob) Run(ctx context.Context, progress chan<- highway.Progress) error {
	return j.fn(ctx, progress)
}

func (j *FuncJob) Marshal() ([]byte, error) {
	// Note: Functions can't be serialized!
	// This job type doesn't support resume.
	return json.Marshal(map[string]string{
		"id":   j.id,
		"type": j.typ,
		"note": "FuncJob cannot be resumed - function is not serializable",
	})
}

// Usage:
// job := jobs.NewFuncJob("custom-1", "custom", func(ctx context.Context, p chan<- highway.Progress) error {
//     p <- highway.Progress{JobID: "custom-1", Message: "Doing custom work..."}
//     // ... do work ...
//     p <- highway.Progress{JobID: "custom-1", Done: true}
//     return nil
// })
```

**Warning**: FuncJob cannot be resumed because functions aren't serializable. Use only for one-shot operations.

---

## Job Factory Pattern

When creating jobs from configuration or flags.

```go
// internal/jobs/factory.go

package jobs

import (
	"fmt"
	"strings"

	"myapp/internal/highway"
)

// CreateDownloadJob creates the appropriate job type based on URL scheme
func CreateDownloadJob(url, outputDir string) (highway.Job, error) {
	switch {
	case strings.HasPrefix(url, "s3://"):
		return NewS3Download(url, outputDir), nil
	case strings.HasPrefix(url, "gs://"):
		return NewGCSDownload(url, outputDir), nil
	case strings.HasPrefix(url, "http://"), strings.HasPrefix(url, "https://"):
		return NewHTTPDownload(url, outputDir), nil
	default:
		return nil, fmt.Errorf("unsupported URL scheme: %s", url)
	}
}

// CreateScanJobs creates scan jobs for all services and regions
func CreateScanJobs(services, regions []string, outputDir string) []highway.Job {
	var jobs []highway.Job

	for _, region := range regions {
		for _, svc := range services {
			switch svc {
			case "s3":
				jobs = append(jobs, NewS3PublicAccessJob(region, outputDir))
			case "ec2":
				jobs = append(jobs, NewEC2SecurityGroupJob(region, outputDir))
			case "iam":
				// IAM is global, only need one job
				if region == regions[0] {
					jobs = append(jobs, NewIAMPolicyJob(outputDir))
				}
			}
		}
	}

	return jobs
}
```

---

## Registering All Job Types

In your cmd package, register all job types for resume capability.

```go
// cmd/root.go

package cmd

import (
	"myapp/internal/highway"
	"myapp/internal/jobs"
)

var hw *highway.Highway

func initHighway(workers int, statePath string) {
	hw = highway.New(workers, statePath)

	// Register all job types
	hw.RegisterType("http-download", jobs.UnmarshalHTTPDownload)
	hw.RegisterType("s3-download", jobs.UnmarshalS3Download)
	hw.RegisterType("gcs-download", jobs.UnmarshalGCSDownload)
	hw.RegisterType("s3-public-access", jobs.UnmarshalS3PublicAccess)
	hw.RegisterType("ec2-security-groups", jobs.UnmarshalEC2SecurityGroup)
	hw.RegisterType("iam-policy", jobs.UnmarshalIAMPolicy)
}
```
