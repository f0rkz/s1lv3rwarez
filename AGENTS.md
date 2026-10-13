# AGENTS.md

## Project intent
Build a small, authentic late-1990s cracktro in Go + Ebitengine. Prefer procedural effects, deterministic behavior, and a tiny dependency surface.

## Working rules
- Keep a fixed 320x200 virtual canvas; present it with nearest-neighbor integer scaling.
- Favor simple data-oriented effects over frameworks or generalized engines.
- Keep audio generated in code unless a tiny embedded asset is clearly better.
- Embed visual assets with `go:embed`.
- Make changes incrementally; run `gofmt ./...` and `go test ./...` after each milestone.
- Avoid unnecessary infrastructure, code generation, or runtime network access.
- Preserve the names `f0rkz`, `sp00nz`, and `S1LV3RWAREZ` exactly when used as crew/history text.
