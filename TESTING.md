# Initial scaffold test results — 2026-05-03

## All four checkpoints verified working

### Checkpoint 1 — skeleton
- `make build`, `make run`, `make test`, `make fmt` all work
- Initial commit present, .gitignore correct
- Module path: github.com/cathal/blascet-photo-advisor

### Checkpoint 2 — data layer
- 9/9 tests pass
- Migrations apply cleanly and are idempotent
- Job, image, task, watched_folder CRUD all verified

### Checkpoint 3 — HTTP server
- Dashboard renders correctly (after template fix this evening)
- /healthz returns 200
- SSE connections establish, hold open, and close cleanly
- Static assets serve

### Checkpoint 4 — job queue + worker
- Drag-drop upload accepted (3 images, job_id=2)
- Worker picked up images from queue
- Stub processor assigned scores and verdicts
- Results written to SQLite
- SSE delivered status updates to the dashboard live

## Bug fixed during testing
- GET / was returning 200 with 1-byte body. Cause: template.Execute
  errors were being silently swallowed. Fixed by adding proper error
  logging around template execution sites, plus the underlying
  rendering issue.

## Ready for next session
- Port prompts from Python pipeline (photo-rate, photo-eval, crop-eval)
- Wire inference client to OpenAI-compatible endpoint
- Replace stub worker with real model calls
- Add EXIF extraction and injection
