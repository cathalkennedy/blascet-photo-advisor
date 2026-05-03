package db

import (
	"time"
)

// JobStatus represents the state of a job
type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// SourceKind represents how images were provided to a job
type SourceKind string

const (
	SourceKindUpload        SourceKind = "upload"
	SourceKindFolderPick    SourceKind = "folder_pick"
	SourceKindWatchedFolder SourceKind = "watched_folder"
)

// ImageStatus represents the state of an image
type ImageStatus string

const (
	ImageStatusQueued    ImageStatus = "queued"
	ImageStatusRunning   ImageStatus = "running"
	ImageStatusCompleted ImageStatus = "completed"
	ImageStatusFailed    ImageStatus = "failed"
)

// TaskStatus represents the state of a task
type TaskStatus string

const (
	TaskStatusQueued    TaskStatus = "queued"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// TaskKind represents the type of evaluation task
type TaskKind string

const (
	TaskKindQualityRating   TaskKind = "quality_rating"
	TaskKindAdjustmentPlan  TaskKind = "adjustment_plan"
	TaskKindCropSuggestions TaskKind = "crop_suggestions"
)

// Job represents a batch of images to process
type Job struct {
	ID               int64
	CreatedAt        time.Time
	Status           JobStatus
	SourceKind       SourceKind
	SourcePath       string
	ModelID          string
	TasksRequested   string // JSON array or comma-separated
	TotalImages      int
	CompletedImages  int
	FailedImages     int
}

// Image represents a single image in a job
type Image struct {
	ID               int64
	JobID            int64
	Path             string
	OriginalFilename string
	Status           ImageStatus
	CompositeScore   *float64
	Verdict          *string
	Error            *string
	StartedAt        *time.Time
	FinishedAt       *time.Time
}

// Task represents a single evaluation task for an image
type Task struct {
	ID          int64
	ImageID     int64
	Kind        TaskKind
	Status      TaskStatus
	RawResponse *string
	ParsedJSON  *string
	Error       *string
	ElapsedMS   *int64
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

// WatchedFolder represents a folder being monitored for new images
type WatchedFolder struct {
	ID              int64
	Path            string
	Enabled         bool
	DefaultModelID  string
	DefaultTasks    string // JSON array or comma-separated
	DebounceSeconds int
	CreatedAt       time.Time
}
