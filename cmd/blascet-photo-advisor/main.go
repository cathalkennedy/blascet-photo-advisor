package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/cathal/blascet-photo-advisor/internal/db"
	"github.com/cathal/blascet-photo-advisor/internal/job"
	"github.com/cathal/blascet-photo-advisor/internal/web"
)

func main() {
	var (
		addr        = flag.String("addr", ":8080", "HTTP server address")
		dbPath      = flag.String("db", ".blascet-data/blascet.db", "SQLite database path")
		concurrency = flag.Int("workers", 1, "Number of worker goroutines")
	)
	flag.Parse()

	// Setup logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("blascet-photo-advisor starting")

	// Ensure database directory exists
	dbDir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		slog.Error("failed to create database directory", "error", err)
		os.Exit(1)
	}

	// Open database
	database, err := db.Open(*dbPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	slog.Info("database opened", "path", *dbPath)

	// Create and start worker pool
	workerPool := job.NewWorkerPool(database, *concurrency)
	workerPool.Start()
	defer workerPool.Stop()

	// Create and start HTTP server
	server := web.New(database, workerPool, *addr)
	if err := server.Start(); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
