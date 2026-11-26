# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

The project guidance lives in two files shared with other coding agents; keep them as the single source of truth and edit them rather than duplicating their content here.

- @AGENTS.md: project overview, commands, code conventions, and the planned data model.
- @WORKFLOW.md: priorities, versioning, git workflow, and CI policy.

## Points that are easy to miss

- Every commit that changes code must bump the version in `version.go` in the same commit (patch for fixes, minor for features or breaking changes); keep `PreRelease: "alpha"`.
- `go vet ./...` and `go test ./...` must both pass before committing.
- Commit directly to `main` and push to `origin` (`github.com/mdhender/ecvg`) as each task is completed. Don't create tags.
- Prefer `go run` over `go build`; any build output goes in `tmp/`, never the repository root.
- Use the `agent` environment when code needs `internal/dotenv`.
