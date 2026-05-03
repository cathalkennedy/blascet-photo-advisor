package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cathal/blascet-photo-advisor/internal/db"
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

	// Create job
	job, err := s.db.CreateJob(
		db.SourceKindUpload,
		"upload",
		"llama-3.2-vision", // Default model for now
		`["quality_rating","adjustment_plan","crop_suggestions"]`,
		len(files),
	)
	if err != nil {
		slog.Error("failed to create job", "error", err)
		http.Error(w, "Failed to create job", http.StatusInternalServerError)
		return
	}

	// Create image records
	tmpDir := ".blascet-data/uploads"
	os.MkdirAll(tmpDir, 0755)

	for _, fileHeader := range files {
		// For now, just store the original filename - actual file handling will come later
		imagePath := filepath.Join(tmpDir, fileHeader.Filename)

		_, err := s.db.CreateImage(job.ID, imagePath, fileHeader.Filename)
		if err != nil {
			slog.Error("failed to create image record", "error", err)
			continue
		}
	}

	slog.Info("job created", "job_id", job.ID, "images", len(files))

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="upload-success">Job #%d created with %d image(s). <a href="/jobs/%d">View job</a></div>`,
		job.ID, len(files), job.ID)
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

	// Subscribe to job events
	eventChan := s.workerPool.EventBus().Subscribe(id)
	defer s.workerPool.EventBus().Unsubscribe(id, eventChan)

	// Send heartbeat every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			slog.Info("SSE connection closed", "job_id", id)
			return

		case event := <-eventChan:
			// Send job event
			eventData := map[string]interface{}{
				"type":     string(event.Type),
				"job_id":   event.JobID,
				"image_id": event.ImageID,
			}
			for k, v := range event.Data {
				eventData[k] = v
			}

			data, _ := json.Marshal(eventData)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case t := <-ticker.C:
			// Send heartbeat
			heartbeat := map[string]interface{}{
				"type":      "heartbeat",
				"timestamp": t.Format(time.RFC3339),
			}
			data, _ := json.Marshal(heartbeat)
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
