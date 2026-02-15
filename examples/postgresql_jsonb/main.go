// Package main demonstrates PostgreSQL JSONB integration with gopantic.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// Account demonstrates PostgreSQL JSONB integration with gopantic
type Account struct {
	ID          string          `json:"id"`
	Name        string          `json:"name" validate:"required,min=2"`
	Email       string          `json:"email" validate:"required,email"`
	MetadataRaw json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// AccountMetadata defines the structure of the metadata JSONB field
type AccountMetadata struct {
	Preferences  Preferences       `json:"preferences"`
	Tags         []string          `json:"tags,omitempty"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`
}

// Preferences nested structure
type Preferences struct {
	Theme    string `json:"theme"`
	Language string `json:"language"`
	Timezone string `json:"timezone"`
}

// GetMetadata provides type-safe access to metadata
func (a *Account) GetMetadata() (*AccountMetadata, error) {
	if len(a.MetadataRaw) == 0 {
		return &AccountMetadata{}, nil
	}

	var metadata AccountMetadata
	if err := json.Unmarshal(a.MetadataRaw, &metadata); err != nil {
		return nil, fmt.Errorf("invalid metadata format: %w", err)
	}

	return &metadata, nil
}

// SetMetadata updates the metadata field
func (a *Account) SetMetadata(metadata *AccountMetadata) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	a.MetadataRaw = metadataJSON
	return nil
}

// AccountRepository demonstrates database operations
type AccountRepository struct {
	db *sql.DB
}

// NewAccountRepository creates a new repository instance
func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// Create validates input and inserts a new account
func (r *AccountRepository) Create(input []byte) (*Account, error) {
	// Parse and validate input with gopantic
	account, err := model.ParseInto[Account](input)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Generate ID and timestamp
	account.ID = generateID()
	account.CreatedAt = time.Now()

	// Insert into PostgreSQL
	query := `
		INSERT INTO accounts (id, name, email, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err = r.db.QueryRow(
		query,
		account.ID,
		account.Name,
		account.Email,
		account.MetadataRaw, // PostgreSQL handles json.RawMessage as JSONB
		account.CreatedAt,
	).Scan(&account.ID)

	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &account, nil
}

// FindByID retrieves an account by ID
func (r *AccountRepository) FindByID(id string) (*Account, error) {
	query := `
		SELECT id, name, email, metadata, created_at
		FROM accounts
		WHERE id = $1
	`

	var account Account
	err := r.db.QueryRow(query, id).Scan(
		&account.ID,
		&account.Name,
		&account.Email,
		&account.MetadataRaw,
		&account.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &account, nil
}

// FindByTag queries accounts using JSONB containment
func (r *AccountRepository) FindByTag(tag string) ([]Account, error) {
	// Use PostgreSQL's JSONB containment operator (@>)
	query := `
		SELECT id, name, email, metadata, created_at
		FROM accounts
		WHERE metadata @> jsonb_build_object('tags', jsonb_build_array($1))
	`

	rows, err := r.db.Query(query, tag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var account Account
		err := rows.Scan(
			&account.ID,
			&account.Name,
			&account.Email,
			&account.MetadataRaw,
			&account.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	return accounts, rows.Err()
}

// UpdateMetadata updates only the metadata JSONB field
func (r *AccountRepository) UpdateMetadata(id string, metadataRaw json.RawMessage) error {
	query := `
		UPDATE accounts
		SET metadata = $1
		WHERE id = $2
	`

	_, err := r.db.Exec(query, metadataRaw, id)
	return err
}

// MergeMetadata partially updates metadata using JSONB merge
func (r *AccountRepository) MergeMetadata(id string, updates json.RawMessage) error {
	query := `
		UPDATE accounts
		SET metadata = COALESCE(metadata, '{}'::jsonb) || $1::jsonb
		WHERE id = $2
	`

	_, err := r.db.Exec(query, updates, id)
	return err
}

// Helper functions

func generateID() string {
	return fmt.Sprintf("acc_%d", time.Now().UnixNano())
}

// Example usage (no actual database connection)
func main() {
	fmt.Println("=== PostgreSQL JSONB Integration with gopantic ===")
	fmt.Println()

	// Example 1: Parse and validate account creation request
	fmt.Println("Example 1: Parse and validate account creation")
	createInput := []byte(`{
		"name": "Alice Johnson",
		"email": "alice@example.com",
		"metadata": {
			"preferences": {
				"theme": "dark",
				"language": "en",
				"timezone": "America/New_York"
			},
			"tags": ["premium", "verified"],
			"custom_fields": {
				"department": "Engineering",
				"level": "Senior"
			}
		}
	}`)

	account, err := model.ParseInto[Account](createInput)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	fmt.Printf("✅ Account parsed and validated:\n")
	fmt.Printf("   Name: %s\n", account.Name)
	fmt.Printf("   Email: %s\n", account.Email)
	fmt.Printf("   Metadata (raw): %s\n", string(account.MetadataRaw))
	fmt.Println()

	// Example 2: Type-safe metadata access
	fmt.Println("Example 2: Type-safe metadata access")
	metadata, err := account.GetMetadata()
	if err != nil {
		log.Fatalf("Failed to parse metadata: %v", err)
	}

	fmt.Printf("✅ Metadata parsed:\n")
	fmt.Printf("   Theme: %s\n", metadata.Preferences.Theme)
	fmt.Printf("   Language: %s\n", metadata.Preferences.Language)
	fmt.Printf("   Tags: %v\n", metadata.Tags)
	fmt.Printf("   Department: %s\n", metadata.CustomFields["department"])
	fmt.Println()

	// Example 3: Validation errors
	fmt.Println("Example 3: Validation catches errors")
	invalidInput := []byte(`{
		"name": "A",
		"email": "invalid-email",
		"metadata": {"tags": ["test"]}
	}`)

	_, err = model.ParseInto[Account](invalidInput)
	if err != nil {
		fmt.Printf("✅ Validation correctly rejected invalid input:\n")
		fmt.Printf("   Error: %v\n", err)
	}
	fmt.Println()

	// Example 4: Updating metadata
	fmt.Println("Example 4: Update metadata")
	updatedMetadata := AccountMetadata{
		Preferences: Preferences{
			Theme:    "light",
			Language: "es",
			Timezone: "Europe/Madrid",
		},
		Tags: []string{"premium", "verified", "active"},
		CustomFields: map[string]string{
			"department": "Engineering",
			"level":      "Staff",
		},
	}

	if err := account.SetMetadata(&updatedMetadata); err != nil {
		log.Fatalf("Failed to update metadata: %v", err)
	}

	fmt.Printf("✅ Metadata updated:\n")
	fmt.Printf("   New raw metadata: %s\n", string(account.MetadataRaw))
	fmt.Println()

	// Example 5: Standalone validation (useful after database fetch)
	fmt.Println("Example 5: Standalone validation")
	// Simulate fetching from database (would have json.RawMessage populated)
	fetchedAccount := Account{
		ID:    "acc_123",
		Name:  "Bob Smith",
		Email: "bob@example.com",
		MetadataRaw: json.RawMessage(`{
			"preferences": {"theme": "dark", "language": "en", "timezone": "UTC"},
			"tags": ["basic"]
		}`),
		CreatedAt: time.Now(),
	}

	// Validate the fetched account
	if err := model.Validate(&fetchedAccount); err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	fmt.Printf("✅ Fetched account validated:\n")
	fmt.Printf("   ID: %s\n", fetchedAccount.ID)
	fmt.Printf("   Name: %s\n", fetchedAccount.Name)
	fmt.Println()

	// Example 6: Partial metadata updates
	fmt.Println("Example 6: Partial metadata update pattern")
	partialUpdate := json.RawMessage(`{"tags": ["premium", "verified", "active", "engaged"]}`)

	fmt.Printf("✅ Partial update (would merge in database):\n")
	fmt.Printf("   Update: %s\n", string(partialUpdate))
	fmt.Printf("   Note: Use MergeMetadata() to merge this with existing metadata\n")
	fmt.Println()

	// Example 7: Two-phase validation
	fmt.Println("Example 7: Two-phase validation (standard library + gopantic)")
	complexInput := []byte(`{
		"name": "Charlie Brown",
		"email": "charlie@example.com",
		"metadata": {
			"preferences": {"theme": "auto", "language": "fr", "timezone": "Europe/Paris"},
			"tags": ["free"],
			"custom_fields": {"trial_ends": "2024-12-31"}
		}
	}`)

	// Phase 1: Standard library unmarshal (handles json.RawMessage perfectly)
	var account2 Account
	if err := json.Unmarshal(complexInput, &account2); err != nil {
		log.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Phase 2: gopantic validation
	if err := model.Validate(&account2); err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	fmt.Printf("✅ Two-phase validation successful:\n")
	fmt.Printf("   Name: %s\n", account2.Name)
	fmt.Printf("   Email: %s\n", account2.Email)
	fmt.Println()

	// Example 8: Database schema
	fmt.Println("Example 8: PostgreSQL schema")
	schema := `
-- Database schema for accounts table
CREATE TABLE accounts (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for JSONB queries
CREATE INDEX idx_account_metadata ON accounts USING gin (metadata);
CREATE INDEX idx_account_tags ON accounts USING gin ((metadata->'tags'));
CREATE INDEX idx_account_theme ON accounts ((metadata->'preferences'->>'theme'));

-- Example queries:
-- Find accounts with 'premium' tag:
SELECT * FROM accounts WHERE metadata @> '{"tags": ["premium"]}';

-- Find accounts with dark theme:
SELECT * FROM accounts WHERE metadata->'preferences'->>'theme' = 'dark';

-- Update metadata (full replace):
UPDATE accounts SET metadata = '{"tags": ["new"]}' WHERE id = 'acc_123';

-- Update metadata (merge):
UPDATE accounts SET metadata = metadata || '{"tags": ["additional"]}' WHERE id = 'acc_123';
`
	fmt.Println(schema)

	fmt.Println("=== Summary ===")
	fmt.Println("✅ gopantic provides seamless PostgreSQL JSONB integration:")
	fmt.Println("   1. json.RawMessage preserves raw JSON for JSONB columns")
	fmt.Println("   2. Validation ensures data quality before database insert")
	fmt.Println("   3. Type-safe access methods for metadata")
	fmt.Println("   4. Works with standard sql.DB and popular ORMs")
	fmt.Println("   5. Supports partial updates via JSONB operators")
}
