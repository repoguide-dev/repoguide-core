## Repo Structure

```
repoguide-core/
├── contracts/  Shared interfaces and analysis/AI contract types
└── model/      Shared domain models
```

## Test Commands

```sh
go test ./...
```

## Notes

- Keep this repo contracts-only. Do not add runtime logic, storage implementations, AI clients, or analysis builders here.
- `repoguide-cli` owns the local/offline implementation.
- `repoguide-cloud/backend` owns the cloud/online implementation.
- If you change package boundaries here, update `README.md` and `STRUCTURE.md` in the same change.
