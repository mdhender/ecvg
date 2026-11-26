# Project Workflow

Guidance for agents when working in this repository.

## Priorities

- Fix bugs before adding features.
  Before offering to implement a change or work on documentation, check for open bugs (`gh issue list --label bug --state open`).
  If any are open, do not make the offer; propose working on the bugs instead.

## Build and test

- Prefer `go run` over `go build` for your work.
- Do not create build artifacts in the root of the repository - put them in tmp/.

- `go vet` and `go test` Both must pass before committing.

```sh
go vet ./...
go test ./...
```

## Versioning

- Semantic versioning.
  The current version lives in `version.go` as a `semver.Version` literal (`Major`, `Minor`, `Patch`, `PreRelease`).
- **Every commit that changes code must bump the version in `version.go`.**
  Documentation-only commits are encouraged to bump it but are not required to.
- The version bump must be included in the commit for the change it describes.
- Choose the version bump by the change: patch for fixes, minor for new features or any breaking change while below v1.0.0.
- Keep `PreRelease: "alpha"` until told otherwise.

## Git workflow

- Commit directly to `main`.
  No feature branches or pull requests; history is the recovery mechanism.
  This may change after v1.0.0.
- Do not create git tags.
  Versions will be tagged on GitHub once the project achieves v1.0.0.
- Agents are authorized to commit and push each task as it is completed, provided all tests pass and the version has been bumped.
- The remote is `origin`, `github.com/mdhender/ecvg`.
- Agents may create GitHub labels as needed.

## CI

- There is no GitHub CI, and none should be added until the project is ready to release v1.0.0.
  Run the checks above locally instead.
