# Initial scaffold test results — 2026-05-03

## Passing
- `make build` succeeds
- `make run` starts the server on :8080
- `make test`: 9/9 tests pass (data layer)
- `/healthz` returns 200
- All four checkpoints' commits present in git log

## Known issue
- GET / returns HTTP 200 with only 1 byte of body — template not
  rendering. Server logs show no error (likely a swallowed
  template.Execute error). Needs Claude Code to:
  1. Find the dashboard handler in internal/web/
  2. Check template loading path (embed vs filesystem)
  3. Add proper error logging around template execution
  4. Fix the underlying rendering bug

## Not yet tested (blocked by template issue)
- Drag-drop upload flow
- Folder picker
- SSE live updates
- End-to-end fake job processing

## Next session
- Fix template rendering (above)
- Then port real prompts and inference client from Python pipeline


