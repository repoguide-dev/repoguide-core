## Repo Structure


```
repoguide/
├── core/             Shared Go packages (imported by both backend and CLI)
│   ├── model/        Canonical data types: TopicContext, ContextPatchJob, TopicPatchSuggestion, etc.
│   ├── store/        Store interfaces: RepoStore, TopicStore, FeedbackStore, JobStore, SuggestionStore
│   ├── ai/           Claude client + AI operations (topic curator, repo context patcher, feedback classifier)
│   │   ├── topic_curator.go          Curate topic context: propose/accept/reject TopicPatchSuggestions
│   │   ├── repo_context_patcher.go   Patch repo-level context from feedback
│   │   └── prompts/                  System prompts for each AI operation
│   └── services/     JobService: executes ContextPatchJobs (patchTopic, patchRepo, newTopic)
│
├── backend/          Go HTTP server (port 8082)
│   ├── main.go       Entry point: MongoDB connect, auth config, start server
│   ├── auth/         Device-code flow, OAuth (GitHub/GitLab/Google), JWT middleware
│   ├── db/           MongoDB store implementation (implements core/store interfaces)
│   ├── model/        Re-exports core/model types for backend packages
│   ├── server/       HTTP mux + all route handlers (repo_handlers.go registers all /api/* routes)
│   ├── analysis/     Shared session-analysis primitives: metrics, pricing, cache, types, utils
│   ├── ai_analysis/  Claude API integration: prompt building, AI-driven topic extraction
│   └── repoanalysis/ Repo-level bundle build: aggregate file/subsystem/trace stats, clustering
│
├── cli/              Go CLI binary (cobra)
│   ├── main.go       Entry point
│   ├── cmd/          Cobra commands: login, logout, register, sessions, repos, stats, files, mcp, doctor, …
│   │   └── root.go   PersistentPreRun fires background sync on every command
│   └── internal/     Core logic
│       ├── sessions.go / session_parser_*.go   Parse Claude, Codex, Cursor, OpenCode, GitHub Copilot, and Gemini CLI session JSON on disk
│       ├── session_events.go                   Convert parsed sessions → RepoSessionEvents
│       ├── repo_sync.go                        RepoSyncClient: HTTP calls to backend API
│       ├── repo_store.go                       Local ~/.repoguide config (list of tracked repos)
│       ├── sqlitestore/store.go                SQLite store implementation (local mode, implements core/store interfaces)
│       ├── mcp_server.go                       MCP server (stdio JSON-RPC, 7 tools - see below)
│       ├── mcp_activity.go                     Append-only log of MCP tool invocations
│       ├── mcp_clients.go                      Backend HTTP calls for MCP tool responses
│       ├── mcp_render.go                       Render backend MCP responses to markdown text
│       ├── mcp_smoke.go                        Smoke-test harness for MCP server
│       ├── agents.go                           Local topic/file scoring (MCP fallback)
│       └── auth/token.go                       Load/save JWT from local keychain/file
│
└── web/              React landing page (Vite + Tailwind, Vite dev server in Docker compose)
```

## Key Flows

### 1. Background Sync (CLI → Backend)
Every CLI command fires `backgroundSync()` in a goroutine (`cmd/root.go`).
It calls `RepoSyncClient.UploadRepoEvents()` which zips parsed session events
and POSTs them to `POST /api/repos/{repo_id}/events/`.

### 2. MCP Server (`repoguide mcp`)
Runs a stdio JSON-RPC MCP server. Seven tools:

| Tool | Purpose |
|------|---------|
| `repoguide_get_repo_experience` | One-shot entry point: returns recommended tool calls for a task |
| `repoguide_list_topics` | Candidate topics for the current task |
| `repoguide_get_bootstrap_context` | Start-here guidance for a chosen topic |
| `repoguide_get_test_context` | Test strategy for a topic |
| `repoguide_get_search_context` | Search guidance (reliable queries, ambiguous ones) |
| `repoguide_record_feedback` | Record whether RepoGuide context helped; call before ending a non-trivial session |

Each tool tries backend first (`mcp_clients.go`), falls back to local file scan
(`mcp_server.go: listTopics / topicFiles`).

### 3. Topic Patch / Curation Pipeline
`core/services.JobService.patchTopic` runs when a `PatchJobTypeTopicPatch` job is dequeued:
1. Load topic + unprocessed feedbacks
2. Load pending `TopicPatchSuggestion` records (suggestions the model hasn't decided on yet)
3. Call `core/ai.CurateTopicContext` → `TopicCuration` (new suggestions + decisions on pending ones)
4. Persist new suggestions (`SuggestionStore.Save`); auto-apply those with confidence ≥ 5
5. Process decisions: accepted suggestions are applied via `ApplyTopicCuration`; rejected ones are marked rejected
6. Write updated `TopicContext` back to store

`TopicPatchSuggestion` statuses: `pending` → `applied` | `rejected`.

### 4. Repo Analysis (Backend)
`POST /api/repos/{repo_id}/analyze` triggers:
1. Load stored events from MongoDB
2. `repoanalysis.Build()` → `RepoAnalysisBundle` (files, subsystems, traces, test signals, discoverability)
3. `ai_analysis` calls Claude to extract topics and build `TopicContext` + `FileSummary` + `BundleSearchData`
4. Store bundle index in MongoDB; MCP topic/file/search endpoints serve it

### 5. Auth
Device-code flow: CLI polls `GET /auth/device/token` after user visits `GET /auth/device`.
OAuth (GitHub/GitLab/Google) available via `GET /auth/{provider}` → callback → JWT cookie/header.

## Backend API Routes

```
GET    /version                              ← required CLI version (no auth, CORS)
GET    /api/repos
POST   /api/repos
GET    /api/repos/{repo_id}
DELETE /api/repos/{repo_id}
POST   /api/repos/{repo_id}/events/      ← CLI sync target
POST   /api/repos/{repo_id}/analyze      ← triggers AI analysis
GET    /api/repos/{repo_id}/topics
GET    /api/repos/{repo_id}/mcp/topics
GET    /api/repos/{repo_id}/mcp/topics/{topic_id}
GET    /api/repos/{repo_id}/mcp/search
GET    /api/limits
```

## Test Commands

```sh
# CLI
cd cli && go test ./...

# Backend
cd backend && go test ./...

# Frontend
cd web && npm test
```

## Status / Known Gaps

- Repo-context analysis (`repoanalysis/`) and AI analysis (`ai_analysis/`) are
  wired into `POST /api/repos/{repo_id}/analyze` but not yet called automatically
  on event upload.
- The CLI does not trigger backend analysis; it must be triggered manually or
  via a separate job.

---

**Keep this file updated.** When you add, move, or remove packages/files, update
the directory tree above. When you add or remove API routes, update the route
table. When a "Known Gap" is addressed, remove it. The goal is that any agent
reading this file can orient in under a minute without grepping the codebase.
