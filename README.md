# RepoGuide Core

`repoguide-core` is the contracts-only shared Go module for RepoGuide.

## Contents

- `model/` - shared domain models used by both runtimes
- `contracts/` - shared interfaces and data contracts between runtimes and implementations

## What Moved Out

Concrete implementation code no longer lives in this repo.

- `repoguide-cli` owns the offline/local runtime, local AI client, local analysis, session artifacts, and SQLite-backed services.
- `repoguide-cloud/backend` owns the online/cloud AI, repo analysis, and backend runtime behavior.

## Development

From this repo directly:

```sh
go test ./...
```

From the umbrella workspace:

```sh
go test ./repoguide-core/...
go test ./repoguide-cli/...
go test ./repoguide-cloud/backend/...
```
