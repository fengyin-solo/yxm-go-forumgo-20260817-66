# Forumgo

Community forum microservice — boards, threads, comments, votes, and moderation reports.

## Features

- **Boards** — create, list, and manage forum categories with slugs and display order
- **Threads** — post discussions with pinning and locking support; view count tracking
- **Comments** — threaded replies with parent/child relationships, soft delete
- **Votes** — upvote/downvote on threads and comments
- **Reports** — user-submitted moderation reports with open/resolved/dismissed workflow

## Architecture

```
cmd/forumgo/
internal/
  model/        — Board, Thread, Comment, Vote, Report entities with Clone/Normalize
  config/       — FORUMGO_* env var configuration
  logger/       — leveled structured logger with child loggers
  auth/         — bearer-token authenticator with 24h expiry
  middleware/   — request ID, structured logging, panic recovery, auth, timeout, rate-limit
  store/        — in-memory stores with atomic JSON-file persistence (write-ahead temp + rename)
  service/      — BoardService, ThreadService, CommentService, VoteService, ReportService
  validator/    — request payload validation
  httpapi/      — REST handler layer, JSON responses, path-segment routing
```

## Building

```bash
go build ./cmd/forumgo
```

## Testing

```bash
go test -race ./...
go vet ./...
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `FORUMGO_ADDR` | `:8080` | HTTP listen address |
| `FORUMGO_DATA_DIR` | `""` | Directory for JSON persistence (empty = in-memory only) |
| `FORUMGO_AUTH_TOKEN` | `""` | Bearer token for authentication (required for non-read endpoints) |
| `FORUMGO_MAX_BODY` | `4096` | Max request body bytes |
| `FORUMGO_TIMEOUT` | `10s` | request timeout |
| `FORUMGO_RATE_LIMIT` | `100` | requests per window per IP |
| `FORUMGO_RATE_WINDOW` | `1m` | rate-limit sliding window |

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/boards` | List boards |
| `POST` | `/api/v1/boards` | Create board |
| `GET` | `/api/v1/boards/by-slug/{slug}` | Get board by slug |
| `GET` | `/api/v1/boards/{id}` | Get board |
| `PUT` | `/api/v1/boards/{id}` | Update board |
| `DELETE` | `/api/v1/boards/{id}` | Delete board |
| `GET` | `/api/v1/boards/{boardID}/threads` | List threads in board |
| `GET` | `/api/v1/threads` | List all threads |
| `POST` | `/api/v1/threads` | Create thread |
| `GET` | `/api/v1/threads/{id}` | Get thread |
| `PUT` | `/api/v1/threads/{id}` | Update thread |
| `DELETE` | `/api/v1/threads/{id}` | Delete thread |
| `GET` | `/api/v1/threads/{threadID}/comments` | List comments in thread |
| `POST` | `/api/v1/comments` | Create comment |
| `GET` | `/api/v1/comments/{id}` | Get comment |
| `PUT` | `/api/v1/comments/{id}` | Update comment |
| `DELETE` | `/api/v1/comments/{id}` | Delete comment |
| `POST` | `/api/v1/threads/{id}/vote` | Vote on thread |
| `POST` | `/api/v1/comments/{id}/vote` | Vote on comment |
| `POST` | `/api/v1/reports` | Submit report |

## Data Model

- **Board** — id, name, slug, description, displayOrder, isLocked, active, timestamps
- **Thread** — id, boardID, authorID, title, body, isPinned, isLocked, viewCount, replyCount, lastReplyAt, timestamps
- **Comment** — id, threadID, parentID, authorID, body, upvotes, downvotes, isDeleted, timestamps
- **Vote** — id, targetType (thread/comment), targetID, userID, value (±1), timestamp
- **Report** — id, reporterID, targetType, targetID, reason, status (open/resolved/dismissed), timestamps

## Persistence

Stores write atomically to temp files then rename, safe for concurrent use. Set `FORUMGO_DATA_DIR` to enable.
