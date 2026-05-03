package db

import (
	"database/sql"
	"fmt"
	"time"
)

// CreateJob creates a new job
func (db *DB) CreateJob(sourceKind SourceKind, sourcePath, modelID, tasksRequested string, totalImages int) (*Job, error) {
	result, err := db.Exec(`
		INSERT INTO jobs (source_kind, source_path, model_id, tasks_requested, total_images)
		VALUES (?, ?, ?, ?, ?)
	`, sourceKind, sourcePath, modelID, tasksRequested, totalImages)
	if err != nil {
		return nil, fmt.Errorf("inserting job: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting job ID: %w", err)
	}

	return db.GetJob(id)
}

// GetJob retrieves a job by ID
func (db *DB) GetJob(id int64) (*Job, error) {
	job := &Job{}
	var createdAt string
	err := db.QueryRow(`
		SELECT id, created_at, status, source_kind, source_path, model_id,
		       tasks_requested, total_images, completed_images, failed_images
		FROM jobs WHERE id = ?
	`, id).Scan(
		&job.ID, &createdAt, &job.Status, &job.SourceKind, &job.SourcePath,
		&job.ModelID, &job.TasksRequested, &job.TotalImages,
		&job.CompletedImages, &job.FailedImages,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %d", id)
		}
		return nil, fmt.Errorf("querying job: %w", err)
	}

	job.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return job, nil
}

// ListJobs retrieves all jobs, ordered by creation time descending
func (db *DB) ListJobs(limit int) ([]*Job, error) {
	rows, err := db.Query(`
		SELECT id, created_at, status, source_kind, source_path, model_id,
		       tasks_requested, total_images, completed_images, failed_images
		FROM jobs
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("querying jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job := &Job{}
		var createdAt string
		err := rows.Scan(
			&job.ID, &createdAt, &job.Status, &job.SourceKind, &job.SourcePath,
			&job.ModelID, &job.TasksRequested, &job.TotalImages,
			&job.CompletedImages, &job.FailedImages,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning job: %w", err)
		}
		job.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// UpdateJobStatus updates a job's status
func (db *DB) UpdateJobStatus(id int64, status JobStatus) error {
	_, err := db.Exec("UPDATE jobs SET status = ? WHERE id = ?", status, id)
	if err != nil {
		return fmt.Errorf("updating job status: %w", err)
	}
	return nil
}

// UpdateJobCounts updates the completed and failed image counts
func (db *DB) UpdateJobCounts(id int64, completedImages, failedImages int) error {
	_, err := db.Exec(`
		UPDATE jobs
		SET completed_images = ?, failed_images = ?
		WHERE id = ?
	`, completedImages, failedImages, id)
	if err != nil {
		return fmt.Errorf("updating job counts: %w", err)
	}
	return nil
}

// GetQueuedJobs retrieves all jobs with status 'queued' or 'running'
// (jobs that are active and may have unprocessed images)
func (db *DB) GetQueuedJobs() ([]*Job, error) {
	rows, err := db.Query(`
		SELECT id, created_at, status, source_kind, source_path, model_id,
		       tasks_requested, total_images, completed_images, failed_images
		FROM jobs
		WHERE status IN (?, ?)
		ORDER BY created_at ASC
	`, JobStatusQueued, JobStatusRunning)
	if err != nil {
		return nil, fmt.Errorf("querying queued jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job := &Job{}
		var createdAt string
		err := rows.Scan(
			&job.ID, &createdAt, &job.Status, &job.SourceKind, &job.SourcePath,
			&job.ModelID, &job.TasksRequested, &job.TotalImages,
			&job.CompletedImages, &job.FailedImages,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning job: %w", err)
		}
		job.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}
