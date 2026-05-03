# blascet-photo-advisor

A local-first photo evaluation tool for photographers. Point the app at images and get structured evaluations from a local vision-language model: quality ratings, Capture One adjustment plans, and crop suggestions.

## What it does

Photographers can drag-drop images, pick folders, or configure watched folders. For each image, the app sends it to a local vision-language model (running in llama-server or LM Studio) and returns:

- Quality rating
- Capture One adjustment plan
- Crop suggestions

Results are stored in SQLite, browsable in a dashboard, and live-updated via Server-Sent Events as the queue processes.

## Tech Stack

- **Language**: Go 1.26+
- **HTTP Router**: chi (github.com/go-chi/chi/v5)
- **Templating**: html/template (stdlib)
- **Frontend**: HTMX + minimal CSS (no build step)
- **Live Updates**: Server-Sent Events (SSE)
- **Database**: SQLite via modernc.org/sqlite (pure Go, WAL mode)
- **Image Processing**: github.com/disintegration/imaging, github.com/rwcarlsen/goexif
- **File Watching**: github.com/fsnotify/fsnotify
- **Config**: TOML via github.com/BurntSushi/toml
- **Inference**: OpenAI-compatible HTTP endpoint (defaults to http://localhost:1234/v1/chat/completions)
- **Logging**: log/slog (stdlib)

## Prerequisites

- Go 1.23 or later
- A local LLM server with OpenAI-compatible API (e.g., LM Studio, llama.cpp)

## Building

```bash
make build
```

This creates the `blascet-photo-advisor` binary in the project root.

## Running

```bash
make run
```

Or run the binary directly:

```bash
./blascet-photo-advisor
```

## Development

```bash
# Format code
make fmt

# Vet code
make vet

# Run tests
make test

# Build, vet, and test
make all
```

## Current Status

**Checkpoint 4 complete**: Worker pool and job queue operational

- ✅ Project skeleton with all dependencies
- ✅ SQLite schema with migrations (jobs, images, tasks, watched_folders)
- ✅ HTTP server with chi router
- ✅ Dashboard with drag-drop and folder picker
- ✅ Server-Sent Events for live job updates
- ✅ Worker pool with configurable concurrency
- ✅ Job state machine (queued → running → completed/failed)
- ✅ Image processing stub (fake scores for now)
- ✅ Event bus for real-time updates
- ✅ Graceful shutdown handling

**Working features:**
- Upload images via drag-drop or folder picker
- Jobs created with image records
- Workers process images in background (2-5 second stub delay)
- Live progress updates via SSE
- Job and image status tracking
- Dashboard shows recent jobs with progress

**Next steps** (future sessions):
- Watched folder implementation
- Real inference client (LM Studio / llama.cpp integration)
- Prompt templates and JSON schema validation
- EXIF extraction and metadata
- Image thumbnails
- Results viewer with markdown rendering
- XMP sidecar writing for Capture One

## Project Structure

```
blascet-photo-advisor/
├── cmd/blascet-photo-advisor/main.go    # entry point
├── internal/
│   ├── config/                          # TOML loading, defaults
│   ├── db/                              # SQLite schema, migrations, queries
│   ├── job/                             # job state machine, queue, worker pool
│   ├── image/                           # resize, EXIF extraction
│   ├── infer/                           # OpenAI-compatible HTTP client
│   ├── watch/                           # folder watcher with debounce
│   ├── web/                             # HTTP handlers, SSE, templates
│   └── prompts/                         # prompt strings + JSON schemas
├── web/
│   ├── templates/                       # html/template files
│   └── static/                          # CSS, vendored JS
├── testdata/                            # sample images for tests
└── config.example.toml
```

## Licence

TBD
