# Core Structure

```
repoguide-core/
├── contracts/  # shared interfaces plus analysis/AI/store contract types
└── model/      # shared domain models
```

## Dependency Role

- `repoguide-cli` depends on this repo for contracts and shared models only.
- `repoguide-cloud/backend` depends on this repo for contracts and shared models only.
- The umbrella workspace one level up owns `go.work`, Docker Compose, `.env`, `.env.prod`, and integration helpers.

## Verification

```sh
go test ./...
```
