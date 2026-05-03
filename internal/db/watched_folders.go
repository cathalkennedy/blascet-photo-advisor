package db

import (
	"database/sql"
	"fmt"
	"time"
)

// CreateWatchedFolder creates a new watched folder
func (db *DB) CreateWatchedFolder(path, defaultModelID, defaultTasks string, debounceSeconds int) (*WatchedFolder, error) {
	result, err := db.Exec(`
		INSERT INTO watched_folders (path, default_model_id, default_tasks, debounce_seconds)
		VALUES (?, ?, ?, ?)
	`, path, defaultModelID, defaultTasks, debounceSeconds)
	if err != nil {
		return nil, fmt.Errorf("inserting watched folder: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting watched folder ID: %w", err)
	}

	return db.GetWatchedFolder(id)
}

// GetWatchedFolder retrieves a watched folder by ID
func (db *DB) GetWatchedFolder(id int64) (*WatchedFolder, error) {
	wf := &WatchedFolder{}
	var createdAt string
	var enabled int
	err := db.QueryRow(`
		SELECT id, path, enabled, default_model_id, default_tasks,
		       debounce_seconds, created_at
		FROM watched_folders WHERE id = ?
	`, id).Scan(
		&wf.ID, &wf.Path, &enabled, &wf.DefaultModelID,
		&wf.DefaultTasks, &wf.DebounceSeconds, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("watched folder not found: %d", id)
		}
		return nil, fmt.Errorf("querying watched folder: %w", err)
	}

	wf.Enabled = enabled == 1
	wf.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return wf, nil
}

// ListWatchedFolders retrieves all watched folders
func (db *DB) ListWatchedFolders() ([]*WatchedFolder, error) {
	rows, err := db.Query(`
		SELECT id, path, enabled, default_model_id, default_tasks,
		       debounce_seconds, created_at
		FROM watched_folders
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying watched folders: %w", err)
	}
	defer rows.Close()

	var folders []*WatchedFolder
	for rows.Next() {
		wf := &WatchedFolder{}
		var createdAt string
		var enabled int
		err := rows.Scan(
			&wf.ID, &wf.Path, &enabled, &wf.DefaultModelID,
			&wf.DefaultTasks, &wf.DebounceSeconds, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning watched folder: %w", err)
		}
		wf.Enabled = enabled == 1
		wf.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		folders = append(folders, wf)
	}

	return folders, rows.Err()
}

// ListEnabledWatchedFolders retrieves all enabled watched folders
func (db *DB) ListEnabledWatchedFolders() ([]*WatchedFolder, error) {
	rows, err := db.Query(`
		SELECT id, path, enabled, default_model_id, default_tasks,
		       debounce_seconds, created_at
		FROM watched_folders
		WHERE enabled = 1
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying enabled watched folders: %w", err)
	}
	defer rows.Close()

	var folders []*WatchedFolder
	for rows.Next() {
		wf := &WatchedFolder{}
		var createdAt string
		var enabled int
		err := rows.Scan(
			&wf.ID, &wf.Path, &enabled, &wf.DefaultModelID,
			&wf.DefaultTasks, &wf.DebounceSeconds, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning watched folder: %w", err)
		}
		wf.Enabled = enabled == 1
		wf.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		folders = append(folders, wf)
	}

	return folders, rows.Err()
}

// UpdateWatchedFolderEnabled updates the enabled status of a watched folder
func (db *DB) UpdateWatchedFolderEnabled(id int64, enabled bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := db.Exec("UPDATE watched_folders SET enabled = ? WHERE id = ?", enabledInt, id)
	if err != nil {
		return fmt.Errorf("updating watched folder enabled: %w", err)
	}
	return nil
}

// DeleteWatchedFolder deletes a watched folder
func (db *DB) DeleteWatchedFolder(id int64) error {
	_, err := db.Exec("DELETE FROM watched_folders WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting watched folder: %w", err)
	}
	return nil
}
