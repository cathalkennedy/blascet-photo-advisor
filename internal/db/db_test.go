package db

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestMigrations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Verify schema_migrations table exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("schema_migrations table not found: %v", err)
	}

	if count == 0 {
		t.Error("expected at least one migration to be applied")
	}

	// Verify all core tables exist
	tables := []string{"jobs", "images", "tasks", "watched_folders"}
	for _, table := range tables {
		var tableCount int
		err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&tableCount)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)
	dbPath := filepath.Join(tmpDir, "test.db")

	// First open
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first open failed: %v", err)
	}

	var count1 int
	err = db1.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count1)
	if err != nil {
		t.Fatalf("failed to count migrations: %v", err)
	}
	db1.Close()

	// Second open (should not re-run migrations)
	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second open failed: %v", err)
	}
	defer db2.Close()

	var count2 int
	err = db2.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count2)
	if err != nil {
		t.Fatalf("failed to count migrations on second open: %v", err)
	}

	if count1 != count2 {
		t.Errorf("migration count changed: first=%d, second=%d", count1, count2)
	}
}

func TestCreateAndGetJob(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	job, err := db.CreateJob(
		SourceKindUpload,
		"/path/to/images",
		"llama-3.2-vision",
		`["quality_rating","adjustment_plan"]`,
		5,
	)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	if job.ID == 0 {
		t.Error("expected non-zero job ID")
	}

	if job.Status != JobStatusQueued {
		t.Errorf("expected status queued, got %s", job.Status)
	}

	if job.TotalImages != 5 {
		t.Errorf("expected 5 total images, got %d", job.TotalImages)
	}

	// Retrieve the job
	retrieved, err := db.GetJob(job.ID)
	if err != nil {
		t.Fatalf("failed to get job: %v", err)
	}

	if retrieved.ID != job.ID {
		t.Errorf("expected job ID %d, got %d", job.ID, retrieved.ID)
	}

	if retrieved.SourceKind != SourceKindUpload {
		t.Errorf("expected source kind upload, got %s", retrieved.SourceKind)
	}
}

func TestUpdateJobStatus(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	job, err := db.CreateJob(SourceKindUpload, "/test", "model-1", `["quality_rating"]`, 1)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	err = db.UpdateJobStatus(job.ID, JobStatusRunning)
	if err != nil {
		t.Fatalf("failed to update job status: %v", err)
	}

	updated, err := db.GetJob(job.ID)
	if err != nil {
		t.Fatalf("failed to get updated job: %v", err)
	}

	if updated.Status != JobStatusRunning {
		t.Errorf("expected status running, got %s", updated.Status)
	}
}

func TestCreateAndGetImage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	job, _ := db.CreateJob(SourceKindUpload, "/test", "model-1", `["quality_rating"]`, 1)

	img, err := db.CreateImage(job.ID, "/path/to/image.jpg", "image.jpg")
	if err != nil {
		t.Fatalf("failed to create image: %v", err)
	}

	if img.ID == 0 {
		t.Error("expected non-zero image ID")
	}

	if img.JobID != job.ID {
		t.Errorf("expected job ID %d, got %d", job.ID, img.JobID)
	}

	if img.Status != ImageStatusQueued {
		t.Errorf("expected status queued, got %s", img.Status)
	}

	// Retrieve the image
	retrieved, err := db.GetImage(img.ID)
	if err != nil {
		t.Fatalf("failed to get image: %v", err)
	}

	if retrieved.OriginalFilename != "image.jpg" {
		t.Errorf("expected filename image.jpg, got %s", retrieved.OriginalFilename)
	}
}

func TestListImagesByJob(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	job, _ := db.CreateJob(SourceKindUpload, "/test", "model-1", `["quality_rating"]`, 3)

	_, _ = db.CreateImage(job.ID, "/path/1.jpg", "1.jpg")
	_, _ = db.CreateImage(job.ID, "/path/2.jpg", "2.jpg")
	_, _ = db.CreateImage(job.ID, "/path/3.jpg", "3.jpg")

	images, err := db.ListImagesByJob(job.ID)
	if err != nil {
		t.Fatalf("failed to list images: %v", err)
	}

	if len(images) != 3 {
		t.Errorf("expected 3 images, got %d", len(images))
	}
}

func TestCreateAndGetTask(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	job, _ := db.CreateJob(SourceKindUpload, "/test", "model-1", `["quality_rating"]`, 1)
	img, _ := db.CreateImage(job.ID, "/path/image.jpg", "image.jpg")

	task, err := db.CreateTask(img.ID, TaskKindQualityRating)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if task.ID == 0 {
		t.Error("expected non-zero task ID")
	}

	if task.ImageID != img.ID {
		t.Errorf("expected image ID %d, got %d", img.ID, task.ImageID)
	}

	if task.Kind != TaskKindQualityRating {
		t.Errorf("expected kind quality_rating, got %s", task.Kind)
	}

	if task.Status != TaskStatusQueued {
		t.Errorf("expected status queued, got %s", task.Status)
	}
}

func TestCreateWatchedFolder(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	wf, err := db.CreateWatchedFolder(
		"/path/to/watch",
		"llama-3.2-vision",
		`["quality_rating"]`,
		10,
	)
	if err != nil {
		t.Fatalf("failed to create watched folder: %v", err)
	}

	if wf.ID == 0 {
		t.Error("expected non-zero watched folder ID")
	}

	if !wf.Enabled {
		t.Error("expected watched folder to be enabled by default")
	}

	if wf.DebounceSeconds != 10 {
		t.Errorf("expected debounce 10s, got %d", wf.DebounceSeconds)
	}
}

func TestListEnabledWatchedFolders(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	wf1, _ := db.CreateWatchedFolder("/path/1", "model-1", `["quality_rating"]`, 5)
	_, _ = db.CreateWatchedFolder("/path/2", "model-1", `["quality_rating"]`, 5)

	// Disable first folder
	db.UpdateWatchedFolderEnabled(wf1.ID, false)

	enabled, err := db.ListEnabledWatchedFolders()
	if err != nil {
		t.Fatalf("failed to list enabled folders: %v", err)
	}

	if len(enabled) != 1 {
		t.Errorf("expected 1 enabled folder, got %d", len(enabled))
	}
}
