-- Migration 001: Initial schema

CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    status TEXT NOT NULL DEFAULT 'queued',
    source_kind TEXT NOT NULL,
    source_path TEXT NOT NULL,
    model_id TEXT NOT NULL,
    tasks_requested TEXT NOT NULL,
    total_images INTEGER NOT NULL DEFAULT 0,
    completed_images INTEGER NOT NULL DEFAULT 0,
    failed_images INTEGER NOT NULL DEFAULT 0,
    CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    CHECK (source_kind IN ('upload', 'folder_pick', 'watched_folder'))
);

CREATE TABLE IF NOT EXISTS images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    path TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    composite_score REAL,
    verdict TEXT,
    error TEXT,
    started_at TEXT,
    finished_at TEXT,
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    CHECK (status IN ('queued', 'running', 'completed', 'failed'))
);

CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    image_id INTEGER NOT NULL,
    kind TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    raw_response TEXT,
    parsed_json TEXT,
    error TEXT,
    elapsed_ms INTEGER,
    started_at TEXT,
    finished_at TEXT,
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,
    CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    CHECK (kind IN ('quality_rating', 'adjustment_plan', 'crop_suggestions'))
);

CREATE TABLE IF NOT EXISTS watched_folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    enabled INTEGER NOT NULL DEFAULT 1,
    default_model_id TEXT NOT NULL,
    default_tasks TEXT NOT NULL,
    debounce_seconds INTEGER NOT NULL DEFAULT 5,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_images_job_id ON images(job_id);
CREATE INDEX IF NOT EXISTS idx_images_status ON images(status);
CREATE INDEX IF NOT EXISTS idx_tasks_image_id ON tasks(image_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_watched_folders_enabled ON watched_folders(enabled);
