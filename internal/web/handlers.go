package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

var templates *template.Template

func init() {
	var err error
	templates, err = template.ParseGlob("web/templates/*.html")
	if err != nil {
		slog.Warn("failed to parse templates", "error", err)
	}
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.db.ListJobs(50)
	if err != nil {
		slog.Error("failed to list jobs", "error", err)
		http.Error(w, "Failed to load jobs", http.StatusInternalServerError)
		return
	}

	data := struct {
		Jobs interface{}
	}{
		Jobs: jobs,
	}

	if err := templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		slog.Error("failed to render dashboard", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(100 << 20); err != nil { // 100 MB max
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	slog.Info("received upload", "file_count", len(files))

	// Placeholder response - actual job creation comes in checkpoint 4
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="upload-success">Received %d file(s). Job creation coming in checkpoint 4.</div>`, len(files))
}

func (s *Server) handleJobDetail(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	job, err := s.db.GetJob(id)
	if err != nil {
		slog.Error("failed to get job", "id", id, "error", err)
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	images, err := s.db.ListImagesByJob(id)
	if err != nil {
		slog.Error("failed to list images", "job_id", id, "error", err)
		http.Error(w, "Failed to load images", http.StatusInternalServerError)
		return
	}

	data := struct {
		Job    interface{}
		Images interface{}
	}{
		Job:    job,
		Images: images,
	}

	if err := templates.ExecuteTemplate(w, "job.html", data); err != nil {
		slog.Error("failed to render job detail", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// handleJobEvents streams SSE events for a job
func (s *Server) handleJobEvents(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	// Verify job exists
	if _, err := s.db.GetJob(id); err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	slog.Info("SSE connection established", "job_id", id)

	// Send heartbeat every 5 seconds
	// Real job events will be wired up in checkpoint 4
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			slog.Info("SSE connection closed", "job_id", id)
			return

		case t := <-ticker.C:
			event := map[string]interface{}{
				"type":      "heartbeat",
				"timestamp": t.Format(time.RFC3339),
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	if err := s.db.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "unhealthy: database ping failed: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "healthy")
}

// renderTemplate is a helper for rendering templates
func renderTemplate(w http.ResponseWriter, name string, data interface{}) error {
	tmpl := filepath.Join("web/templates", name)
	return templates.ExecuteTemplate(w, tmpl, data)
}
