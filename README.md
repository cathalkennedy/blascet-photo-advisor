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

**Checkpoint 1 complete**: Project skeleton established

- ✅ Directory structure created
- ✅ Go module initialised (github.com/cathal/blascet-photo-advisor)
- ✅ Dependencies added
- ✅ Stub main.go
- ✅ Makefile with build, run, test, fmt targets
- ✅ README, .gitignore
- ✅ Initial git commit

**Next**: Data layer (SQLite schema, migrations, queries)

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
