package db

import (
	"database/sql"
	"fmt"
	"time"
)

// CreateTask creates a new task
func (db *DB) CreateTask(imageID int64, kind TaskKind) (*Task, error) {
	result, err := db.Exec(`
		INSERT INTO tasks (image_id, kind)
		VALUES (?, ?)
	`, imageID, kind)
	if err != nil {
		return nil, fmt.Errorf("inserting task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting task ID: %w", err)
	}

	return db.GetTask(id)
}

// GetTask retrieves a task by ID
func (db *DB) GetTask(id int64) (*Task, error) {
	task := &Task{}
	var startedAt, finishedAt sql.NullString
	err := db.QueryRow(`
		SELECT id, image_id, kind, status, raw_response, parsed_json,
		       error, elapsed_ms, started_at, finished_at
		FROM tasks WHERE id = ?
	`, id).Scan(
		&task.ID, &task.ImageID, &task.Kind, &task.Status,
		&task.RawResponse, &task.ParsedJSON, &task.Error,
		&task.ElapsedMS, &startedAt, &finishedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %d", id)
		}
		return nil, fmt.Errorf("querying task: %w", err)
	}

	if startedAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", startedAt.String)
		task.StartedAt = &t
	}
	if finishedAt.Valid {
		t, _ := time.Parse("2006-01-02 15:04:05", finishedAt.String)
		task.FinishedAt = &t
	}

	return task, nil
}

// ListTasksByImage retrieves all tasks for an image
func (db *DB) ListTasksByImage(imageID int64) ([]*Task, error) {
	rows, err := db.Query(`
		SELECT id, image_id, kind, status, raw_response, parsed_json,
		       error, elapsed_ms, started_at, finished_at
		FROM tasks
		WHERE image_id = ?
		ORDER BY id ASC
	`, imageID)
	if err != nil {
		return nil, fmt.Errorf("querying tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		var startedAt, finishedAt sql.NullString
		err := rows.Scan(
			&task.ID, &task.ImageID, &task.Kind, &task.Status,
			&task.RawResponse, &task.ParsedJSON, &task.Error,
			&task.ElapsedMS, &startedAt, &finishedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}

		if startedAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", startedAt.String)
			task.StartedAt = &t
		}
		if finishedAt.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", finishedAt.String)
			task.FinishedAt = &t
		}

		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// UpdateTaskStatus updates a task's status
func (db *DB) UpdateTaskStatus(id int64, status TaskStatus) error {
	now := time.Now().Format("2006-01-02 15:04:05")

	if status == TaskStatusRunning {
		_, err := db.Exec("UPDATE tasks SET status = ?, started_at = ? WHERE id = ?", status, now, id)
		return err
	} else if status == TaskStatusCompleted || status == TaskStatusFailed {
		_, err := db.Exec("UPDATE tasks SET status = ?, finished_at = ? WHERE id = ?", status, now, id)
		return err
	}

	_, err := db.Exec("UPDATE tasks SET status = ? WHERE id = ?", status, id)
	return err
}

// UpdateTaskResult updates a task with processing results
func (db *DB) UpdateTaskResult(id int64, rawResponse, parsedJSON string, elapsedMS int64) error {
	_, err := db.Exec(`
		UPDATE tasks
		SET raw_response = ?, parsed_json = ?, elapsed_ms = ?,
		    status = ?, finished_at = ?
		WHERE id = ?
	`, rawResponse, parsedJSON, elapsedMS, TaskStatusCompleted,
		time.Now().Format("2006-01-02 15:04:05"), id)
	if err != nil {
		return fmt.Errorf("updating task result: %w", err)
	}
	return nil
}

// UpdateTaskError updates a task with an error
func (db *DB) UpdateTaskError(id int64, errorMsg string) error {
	_, err := db.Exec(`
		UPDATE tasks
		SET error = ?, status = ?, finished_at = ?
		WHERE id = ?
	`, errorMsg, TaskStatusFailed, time.Now().Format("2006-01-02 15:04:05"), id)
	if err != nil {
		return fmt.Errorf("updating task error: %w", err)
	}
	return nil
}
