package db

import (
	"database/sql"
	"fmt"
	"time"
)

// CreateImage creates a new image record
func (db *DB) CreateImage(jobID int64, path, originalFilename string) (*Image, error) {
	result, err := db.Exec(`
		INSERT INTO images (job_id, path, original_filename)
		VALUES (?, ?, ?)
	`, jobID, path, originalFilename)
	if err != nil {
		return nil, fmt.Errorf("inserting image: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting image ID: %w", err)
	}

	return db.GetImage(id)
}

// GetImage retrieves an image by ID
func (db *DB) GetImage(id int64) (*Image, error) {
	img := &Image{}
	var startedAt, finishedAt sql.NullString
	err := db.QueryRow(`
		SELECT id, job_id, path, original_filename, status,
		       composite_score, verdict, error, started_at, finished_at
		FROM images WHERE id = ?
	`, id).Scan(
		&img.ID, &img.JobID, &img.Path, &img.OriginalFilename, &img.Status,
		&img.CompositeScore, &img.Verdict, &img.Error, &startedAt, &finishedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("image not found: %d", id)
		}
		return nil, fmt.Errorf("querying image: %w", err)
	}

	if startedAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", startedAt.String)
		img.StartedAt = &t
	}
	if finishedAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", finishedAt.String)
		img.FinishedAt = &t
	}

	return img, nil
}

// ListImagesByJob retrieves all images for a job
func (db *DB) ListImagesByJob(jobID int64) ([]*Image, error) {
	rows, err := db.Query(`
		SELECT id, job_id, path, original_filename, status,
		       composite_score, verdict, error, started_at, finished_at
		FROM images
		WHERE job_id = ?
		ORDER BY id ASC
	`, jobID)
	if err != nil {
		return nil, fmt.Errorf("querying images: %w", err)
	}
	defer rows.Close()

	var images []*Image
	for rows.Next() {
		img := &Image{}
		var startedAt, finishedAt sql.NullString
		err := rows.Scan(
			&img.ID, &img.JobID, &img.Path, &img.OriginalFilename, &img.Status,
			&img.CompositeScore, &img.Verdict, &img.Error, &startedAt, &finishedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning image: %w", err)
		}

		if startedAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", startedAt.String)
			img.StartedAt = &t
		}
		if finishedAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", finishedAt.String)
			img.FinishedAt = &t
		}

		images = append(images, img)
	}

	return images, rows.Err()
}

// UpdateImageStatus updates an image's status and timestamps
func (db *DB) UpdateImageStatus(id int64, status ImageStatus) error {
	now := time.Now().Format("2006-01-02 15:04:05")

	var query string
	if status == ImageStatusRunning {
		query = "UPDATE images SET status = ?, started_at = ? WHERE id = ?"
		_, err := db.Exec(query, status, now, id)
		return err
	} else if status == ImageStatusCompleted || status == ImageStatusFailed {
		query = "UPDATE images SET status = ?, finished_at = ? WHERE id = ?"
		_, err := db.Exec(query, status, now, id)
		return err
	}

	_, err := db.Exec("UPDATE images SET status = ? WHERE id = ?", status, id)
	return err
}

// UpdateImageResult updates an image with processing results
func (db *DB) UpdateImageResult(id int64, compositeScore float64, verdict string) error {
	_, err := db.Exec(`
		UPDATE images
		SET composite_score = ?, verdict = ?, status = ?, finished_at = ?
		WHERE id = ?
	`, compositeScore, verdict, ImageStatusCompleted, time.Now().Format("2006-01-02 15:04:05"), id)
	if err != nil {
		return fmt.Errorf("updating image result: %w", err)
	}
	return nil
}

// UpdateImageError updates an image with an error
func (db *DB) UpdateImageError(id int64, errorMsg string) error {
	_, err := db.Exec(`
		UPDATE images
		SET error = ?, status = ?, finished_at = ?
		WHERE id = ?
	`, errorMsg, ImageStatusFailed, time.Now().Format("2006-01-02 15:04:05"), id)
	if err != nil {
		return fmt.Errorf("updating image error: %w", err)
	}
	return nil
}

// GetQueuedImagesByJob retrieves queued images for a job
func (db *DB) GetQueuedImagesByJob(jobID int64) ([]*Image, error) {
	rows, err := db.Query(`
		SELECT id, job_id, path, original_filename, status,
		       composite_score, verdict, error, started_at, finished_at
		FROM images
		WHERE job_id = ? AND status = ?
		ORDER BY id ASC
	`, jobID, ImageStatusQueued)
	if err != nil {
		return nil, fmt.Errorf("querying queued images: %w", err)
	}
	defer rows.Close()

	var images []*Image
	for rows.Next() {
		img := &Image{}
		var startedAt, finishedAt sql.NullString
		err := rows.Scan(
			&img.ID, &img.JobID, &img.Path, &img.OriginalFilename, &img.Status,
			&img.CompositeScore, &img.Verdict, &img.Error, &startedAt, &finishedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning image: %w", err)
		}

		if startedAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", startedAt.String)
			img.StartedAt = &t
		}
		if finishedAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", finishedAt.String)
			img.FinishedAt = &t
		}

		images = append(images, img)
	}

	return images, rows.Err()
}
