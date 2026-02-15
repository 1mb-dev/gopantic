# PostgreSQL JSONB Integration Example

This example demonstrates how to use gopantic with PostgreSQL JSONB columns for flexible metadata storage.

## Overview

The example shows:
- Parsing and validating JSON input with `json.RawMessage` fields
- Type-safe metadata access via helper methods
- PostgreSQL JSONB storage and queries
- Partial metadata updates using JSONB operators
- Two-phase validation (standard library + gopantic)

## Running the Example

```bash
cd examples/postgresql_jsonb
go run main.go
```

## Key Patterns

### 1. Struct Definition

```go
type Account struct {
    ID          string          `json:"id"`
    Name        string          `json:"name" validate:"required,min=2"`
    Email       string          `json:"email" validate:"required,email"`
    MetadataRaw json.RawMessage `json:"metadata,omitempty"`
    CreatedAt   time.Time       `json:"created_at"`
}
```

### 2. Type-Safe Metadata Access

```go
func (a *Account) GetMetadata() (*AccountMetadata, error) {
    var metadata AccountMetadata
    if err := json.Unmarshal(a.MetadataRaw, &metadata); err != nil {
        return nil, err
    }
    return &metadata, nil
}
```

### 3. Database Operations

```go
// Insert with JSONB
query := `
    INSERT INTO accounts (id, name, email, metadata, created_at)
    VALUES ($1, $2, $3, $4, $5)
`
db.Exec(query, account.ID, account.Name, account.Email, account.MetadataRaw, account.CreatedAt)

// Query using JSONB operators
query := `
    SELECT * FROM accounts
    WHERE metadata @> '{"tags": ["premium"]}'::jsonb
`
```

## PostgreSQL Setup

```sql
-- Create table
CREATE TABLE accounts (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_account_metadata ON accounts USING gin (metadata);
CREATE INDEX idx_account_tags ON accounts USING gin ((metadata->'tags'));
```

## Use Cases

### API Request Validation

```go
func CreateAccount(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)

    // Parse and validate in one step
    account, err := model.ParseInto[Account](body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Save to database
    repo.Create(account)
}
```

### Flexible User Preferences

```go
type UserPreferences struct {
    Theme         string   `json:"theme"`
    Notifications bool     `json:"notifications"`
    Language      string   `json:"language"`
    CustomSettings map[string]interface{} `json:"custom_settings,omitempty"`
}

type User struct {
    ID             string          `json:"id"`
    Email          string          `json:"email" validate:"required,email"`
    PreferencesRaw json.RawMessage `json:"preferences,omitempty"`
}
```

### Multi-Tenant Configuration

```go
type TenantConfig struct {
    TenantID  string          `json:"tenant_id" validate:"required"`
    Name      string          `json:"name" validate:"required"`
    ConfigRaw json.RawMessage `json:"config,omitempty"`
}

// Each tenant can have different config structure
```

## Benefits

1. **Schema Flexibility**: Store arbitrary JSON without schema changes
2. **Type Safety**: Validate required fields while keeping metadata flexible
3. **PostgreSQL Power**: Use JSONB operators for efficient queries
4. **gopantic Integration**: Automatic validation before database operations
5. **Migration Path**: Easy to promote metadata fields to top-level columns later

## See Also

- [Type Reference](https://1mb-dev.github.io/gopantic/reference/types/) - Supported types including json.RawMessage
- [API Reference](https://1mb-dev.github.io/gopantic/reference/api/) - Complete API documentation
