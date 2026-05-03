package job

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/cathal/blascet-photo-advisor/internal/db"
)

// WorkerPool manages a pool of workers that process image jobs
type WorkerPool struct {
	db          *db.DB
	concurrency int
	pollInterval time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	eventBus    *EventBus
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(database *db.DB, concurrency int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	if concurrency < 1 {
		concurrency = 1
	}

	return &WorkerPool{
		db:           database,
		concurrency:  concurrency,
		pollInterval: 2 * time.Second,
		ctx:          ctx,
		cancel:       cancel,
		eventBus:     NewEventBus(),
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	slog.Info("starting worker pool", "concurrency", wp.concurrency)

	for i := 0; i < wp.concurrency; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop stops the worker pool gracefully
func (wp *WorkerPool) Stop() {
	slog.Info("stopping worker pool")
	wp.cancel()
	wp.wg.Wait()
	slog.Info("worker pool stopped")
}

// EventBus returns the event bus for subscribing to job events
func (wp *WorkerPool) EventBus() *EventBus {
	return wp.eventBus
}

// worker is a single worker goroutine
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	slog.Info("worker started", "worker_id", id)

	ticker := time.NewTicker(wp.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-wp.ctx.Done():
			slog.Info("worker stopping", "worker_id", id)
			return

		case t := <-ticker.C:
			slog.Debug("worker tick", "worker_id", id, "time", t.Format(time.RFC3339))
			if err := wp.processNextImage(); err != nil {
				if err != ErrNoWork {
					slog.Error("worker error", "worker_id", id, "error", err)
				} else {
					slog.Debug("no work available", "worker_id", id)
				}
			}
			slog.Debug("worker tick complete, looping", "worker_id", id)
		}
	}
}

var ErrNoWork = fmt.Errorf("no work available")

// processNextImage finds and processes the next queued image
func (wp *WorkerPool) processNextImage() error {
	slog.Debug("claiming next image: querying for active jobs")

	// Find a queued job
	queuedJobs, err := wp.db.GetQueuedJobs()
	if err != nil {
		return fmt.Errorf("getting queued jobs: %w", err)
	}

	if len(queuedJobs) == 0 {
		slog.Debug("no active jobs found")
		return ErrNoWork
	}

	job := queuedJobs[0]
	slog.Debug("found active job", "job_id", job.ID, "job_status", job.Status)

	// Mark job as running if it's still queued
	if job.Status == db.JobStatusQueued {
		if err := wp.db.UpdateJobStatus(job.ID, db.JobStatusRunning); err != nil {
			return fmt.Errorf("updating job status: %w", err)
		}
		wp.eventBus.Publish(Event{
			Type:  EventTypeJobStatusChange,
			JobID: job.ID,
			Data:  map[string]interface{}{"status": db.JobStatusRunning},
		})
	}

	// Get queued images for this job
	images, err := wp.db.GetQueuedImagesByJob(job.ID)
	if err != nil {
		return fmt.Errorf("getting queued images: %w", err)
	}

	if len(images) == 0 {
		slog.Info("job has no queued images, marking as completed", "job_id", job.ID)
		// No more images, mark job as completed
		if err := wp.db.UpdateJobStatus(job.ID, db.JobStatusCompleted); err != nil {
			return fmt.Errorf("updating job status: %w", err)
		}
		wp.eventBus.Publish(Event{
			Type:  EventTypeJobStatusChange,
			JobID: job.ID,
			Data:  map[string]interface{}{"status": db.JobStatusCompleted},
		})
		return ErrNoWork
	}

	// Process the first queued image
	image := images[0]
	slog.Debug("claimed image for processing", "job_id", job.ID, "image_id", image.ID, "filename", image.OriginalFilename)
	return wp.processImage(job, image)
}

// processImage processes a single image (stub implementation)
func (wp *WorkerPool) processImage(job *db.Job, image *db.Image) error {
	slog.Info("processing image", "job_id", job.ID, "image_id", image.ID, "filename", image.OriginalFilename)

	// Mark image as running
	if err := wp.db.UpdateImageStatus(image.ID, db.ImageStatusRunning); err != nil {
		return fmt.Errorf("updating image status: %w", err)
	}

	wp.eventBus.Publish(Event{
		Type:    EventTypeImageStatusChange,
		JobID:   job.ID,
		ImageID: image.ID,
		Data:    map[string]interface{}{"status": db.ImageStatusRunning, "filename": image.OriginalFilename},
	})

	// Stub: sleep for 2-5 seconds to simulate processing
	sleepDuration := time.Duration(2000+rand.Intn(3000)) * time.Millisecond
	time.Sleep(sleepDuration)

	// Stub: generate a fake composite score
	compositeScore := 0.5 + rand.Float64()*0.5 // 0.5 to 1.0

	var verdict string
	if compositeScore >= 0.85 {
		verdict = "excellent"
	} else if compositeScore >= 0.7 {
		verdict = "good"
	} else if compositeScore >= 0.5 {
		verdict = "acceptable"
	} else {
		verdict = "poor"
	}

	// Mark image as completed with fake results
	if err := wp.db.UpdateImageResult(image.ID, compositeScore, verdict); err != nil {
		return fmt.Errorf("updating image result: %w", err)
	}

	wp.eventBus.Publish(Event{
		Type:    EventTypeImageCompleted,
		JobID:   job.ID,
		ImageID: image.ID,
		Data: map[string]interface{}{
			"status":          db.ImageStatusCompleted,
			"filename":        image.OriginalFilename,
			"composite_score": compositeScore,
			"verdict":         verdict,
		},
	})

	slog.Info("image processed", "job_id", job.ID, "image_id", image.ID, "score", compositeScore, "verdict", verdict)

	slog.Debug("updating job progress counts", "job_id", job.ID)
	// Update job counts
	completed, failed := wp.countJobProgress(job.ID)
	if err := wp.db.UpdateJobCounts(job.ID, completed, failed); err != nil {
		return fmt.Errorf("updating job counts: %w", err)
	}

	wp.eventBus.Publish(Event{
		Type:  EventTypeJobProgress,
		JobID: job.ID,
		Data: map[string]interface{}{
			"completed": completed,
			"failed":    failed,
			"total":     job.TotalImages,
		},
	})

	// Check if job is complete
	if completed+failed >= job.TotalImages {
		finalStatus := db.JobStatusCompleted
		if failed > 0 && completed == 0 {
			finalStatus = db.JobStatusFailed
		}

		if err := wp.db.UpdateJobStatus(job.ID, finalStatus); err != nil {
			return fmt.Errorf("updating final job status: %w", err)
		}

		wp.eventBus.Publish(Event{
			Type:  EventTypeJobStatusChange,
			JobID: job.ID,
			Data:  map[string]interface{}{"status": finalStatus},
		})

		slog.Info("job completed", "job_id", job.ID, "status", finalStatus, "completed", completed, "failed", failed)
	} else {
		slog.Debug("job still in progress", "job_id", job.ID, "completed", completed, "failed", failed, "total", job.TotalImages)
	}

	slog.Debug("processImage complete", "job_id", job.ID, "image_id", image.ID)
	return nil
}

// countJobProgress counts completed and failed images for a job
func (wp *WorkerPool) countJobProgress(jobID int64) (completed, failed int) {
	images, err := wp.db.ListImagesByJob(jobID)
	if err != nil {
		slog.Error("failed to count job progress", "job_id", jobID, "error", err)
		return 0, 0
	}

	for _, img := range images {
		if img.Status == db.ImageStatusCompleted {
			completed++
		} else if img.Status == db.ImageStatusFailed {
			failed++
		}
	}

	return completed, failed
}
