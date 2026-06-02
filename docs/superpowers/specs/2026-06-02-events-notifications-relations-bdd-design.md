# Events / Notifications / Relations Pages — BDD Acceptance Tests

## Context

The web module (`internal/modules/web/`) serves Events, Notifications, and Relations pages at `/service/web/*`. These pages have route handlers but lack BDD acceptance specs that exercise the HTTP endpoints end-to-end through the Fiber test app with a real PostgreSQL database.

### Current state

| File | Status |
|------|--------|
| `event_page_spec_test.go` | Does not exist |
| `notifications_page_spec_test.go` | Does not exist |
| `relations_suite_test.go` | Scaffold only — all 6 tests `Skip`ed |
| `event_spec_test.go` | Exists but tests store-level event system only, not web pages |
| `notify_spec_test.go` | Exists but tests command/form types only, not web pages |

### Reference

`tests/specs/view_page_suite_test.go` — fully implemented, uses adapter-stub pattern with real EntClient data seeding.

## Design

### Architecture

Three separate spec files, each following the `view_page_suite_test.go` pattern:

1. **`event_page_spec_test.go` (NEW)** — 9 test cases for 4 event routes
2. **`notifications_page_spec_test.go` (NEW)** — 8 test cases for 3 notification routes
3. **`relations_suite_test.go` (REWRITE)** — 8 test cases for 4 relation routes

### Shared adapter pattern

Each file defines a `webPageAdapter` wrapping `store.Adapter`:
- `GetDB()` delegates to the shared `EntClient` from `lifecycle.go`
- `ParameterGet()` returns a test auth token with `uid`, `topic`, and `scopes`
- Admin variant: scopes `["admin","read","write"]`
- User variant: scopes `["read","write"]`

Lifecycle per `Describe`:
```go
BeforeEach:
  store.Database = adapter
  webmod.InitForE2E(jsonconf)
  webmod.MountForE2E(App)
  seed test data via EntClient

AfterEach:
  store.Database = origDB
  cleanup seeded data
```

### Data seeding

All test data is created directly via `EntClient` fluent API (matching `view_page_suite_test.go`):
- **Events**: 3 `DataEvent` rows (regular + 2 webhook events with `_webhook_method`), `EventFilterCache.Hydrate()`
- **Notifications**: 4 `NotificationRecord` rows across two users, mixed statuses (sent, failed)
- **Relations**: 2 `ResourceLink` rows with app/capability/entity_id fields, 1 `DataEvent` referenced by links

Cleanup uses Ent delete queries filtered by test-specific IDs.

### Event page tests (9 cases)

| Route | Case | Auth | Expected |
|-------|------|------|----------|
| `GET /events` | Admin sees page with tabs/sources/types | admin token | 200, HTML with tab links |
| `GET /events` | Non-admin scope rejected | user token | 403, "Admin access required" |
| `GET /events` | No auth redirects | no cookie | 303 to `/service/web/login` |
| `GET /events/data-events` | Table fragment with seeded rows | admin token | 200, HTML table |
| `GET /events/data-events` | Filter by source | admin token | 200, filtered fragment |
| `GET /events/webhook-logs` | Webhook events only | admin token | 200, HTML webhook table |
| `GET /events/payload/:eventID` | Payload detail for regular event | admin token | 200, HTML with JSON |
| `GET /events/payload/:eventID` | Event not found | admin token | 200, "Event not found" |
| `GET /events/payload/:eventID` | Webhook payload detail | admin token | 200, HTML with headers/body |

### Notification page tests (8 cases)

| Route | Case | Auth | Expected |
|-------|------|------|----------|
| `GET /notifications` | User sees page with records | valid token | 200, HTML page |
| `GET /notifications` | No auth redirects | no cookie | 303 |
| `GET /notifications/list` | Table with user's records | valid token | 200, HTML table |
| `GET /notifications/list` | Pagination with cursor | valid token | 200, next page |
| `GET /notifications/list` | Empty for different user | other user token | empty state |
| `POST /notifications/:id/retry` | Invalid ID format | valid token | 400 |
| `POST /notifications/:id/retry` | Non-existent record | valid token | "Record not found" |
| `POST /notifications/:id/retry` | Non-failed record | valid token | "Only failed" |

Retry success case skipped — requires real notify providers.

### Relations page tests (8 cases)

| Route | Case | Auth | Expected |
|-------|------|------|----------|
| `GET /relations` | Page with search input | valid token | 200, HTML with search |
| `GET /relations` | No auth redirects | no cookie | 303 |
| `GET /relations/tree` | Missing node param | valid token | placeholder text |
| `GET /relations/tree` | Invalid node format | valid token | 400 |
| `GET /relations/tree` | Valid node with relations | valid token | 200, HTML edges |
| `GET /relations/search` | Empty query | valid token | 200, empty body |
| `GET /relations/search` | Finds matching nodes | valid token | 200, results HTML |
| `GET /relations/detail` | Edge detail | valid token | 200, edge metadata |

### Running

```bash
go tool task test:specs -- --label-filter="module,web"
go tool task test:specs:serial   # debugging single file
```

## Implementation

1. Write `event_page_spec_test.go` — 9 cases, admin+user adapters, event seed data
2. Write `notifications_page_spec_test.go` — 8 cases, notify record seed data
3. Rewrite `relations_suite_test.go` — 8 cases, resource link seed data
4. Verify: `go build -tags integration ./tests/specs/...`
5. Run: `go tool ginkgo --label-filter="module,web" --tags integration ./tests/specs/`
