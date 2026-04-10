# SurrealDB Integration Tests

This directory contains integration tests for the SurrealDB exporters (v1 and v2).

## Running Integration Tests Locally

### Prerequisites

1. Install SurrealDB:
   ```bash
   curl -sSf https://install.surrealdb.com | sh
   ```

2. Start SurrealDB in memory mode:
   ```bash
   surreal start --log trace --user root --pass root memory
   ```
   
   Or with persistent storage:
   ```bash
   surreal start --log trace --user root --pass root file://mydatabase.db
   ```

### Running Tests

With SurrealDB running, execute:

```bash
# Run all integration tests
go test -v ./exporters/surrealdb/v1/... -run Integration
go test -v ./exporters/surrealdb/v2/... -run Integration

# Or run all tests including integration tests
go test -v ./exporters/surrealdb/...
```

### Skipping Integration Tests

Integration tests require a running SurrealDB instance. To skip them during regular test runs:

```bash
# Skip integration tests with -short flag
go test -short ./...
```

All integration tests check `testing.Short()` and skip themselves when the `-short` flag is used.

## GitHub Actions

Integration tests run automatically in GitHub Actions:
- **Unit tests**: Run on every push/PR (with `-short` to skip integration tests)
- **Integration tests**: Run in a separate job with SurrealDB installed as a service

See `.github/workflows/ci.yml` for the complete CI configuration.

## Test Structure

### v1 Integration Tests
Located in `exporters/surrealdb/v1/integration_test.go`:
- Basic structure validation
- Spouse relationships
- Parent-child relationships
- Multi-generational queries
- Sibling queries
- Complex family structures

### v2 Integration Tests
Located in `exporters/surrealdb/v2/surrealdb_integration_test.go`:
- Basic structure validation
- Media and note relationships with non-standard IDs
- Family relationships
- Entity type verification

## Configuration

Tests use the following defaults:
- URL: `http://localhost:8000`
- Namespace: `gedcom_test` (v1) / `gedcom_test_v2` (v2)
- Database: `test_db` (v1) / `test_db_v2` (v2)
- Username: `root`
- Password: `root`

These can be customized in the test files if needed.

## Troubleshooting

### SurrealDB Connection Issues

If tests fail with connection errors:

1. Check SurrealDB is running:
   ```bash
   curl http://localhost:8000/health
   ```

2. Check the logs:
   ```bash
   # SurrealDB logs will show in the terminal where you started it
   ```

3. Ensure the port is not in use:
   ```bash
   lsof -i :8000
   ```

### Test Cleanup

Tests automatically clean up after themselves by removing tables. If you need to manually clean:

```bash
surreal sql --ns gedcom_test --db test_db --user root --pass root --endpoint http://localhost:8000
```

Then in the SQL prompt:
```sql
REMOVE TABLE person;
REMOVE TABLE family;
REMOVE TABLE note;
REMOVE TABLE media_object;
REMOVE TABLE child_of;
REMOVE TABLE parent_of;
REMOVE TABLE spouse_of;
REMOVE TABLE has_note;
REMOVE TABLE has_media;
```

