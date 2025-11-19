# AES-256-GCM Encryption for PII Data

This package provides AES-256-GCM encryption for sensitive personally identifiable information (PII) stored in the database.

## Features

- **AES-256-GCM**: Authenticated encryption with 256-bit keys
- **Automatic GORM Hooks**: Encrypt on save, decrypt on load
- **Tag-based Configuration**: Use `encrypt:"true"` struct tags
- **Base64 Encoding**: Encrypted data stored as base64 strings

## Key Generation

Generate a new encryption key (must be done ONCE during setup):

```bash
# Generate a random 32-byte key (base64-encoded)
go run cmd/tools/generate_encryption_key.go
# Output: dGhpcyBpcyBhIHNlY3JldCBrZXkgZm9yIGVuY3J5cHRpb24h...
```

Store the key in environment variable:

```bash
# .env
ENCRYPTION_KEY_BASE64=dGhpcyBpcyBhIHNlY3JldCBrZXkgZm9yIGVuY3J5cHRpb24h...
```

**WARNING**:
- NEVER commit the encryption key to git
- NEVER rotate the key without re-encrypting existing data
- Store the key in a secure secrets manager (AWS Secrets Manager, HashiCorp Vault)

## Usage with GORM

### 1. Initialize the Encryption Plugin

```go
package main

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/hxuan190/stable_payment_gateway/internal/pkg/crypto"
)

func main() {
	// Load encryption key from environment
	keyBase64 := os.Getenv("ENCRYPTION_KEY_BASE64")
	if keyBase64 == "" {
		log.Fatal("ENCRYPTION_KEY_BASE64 environment variable not set")
	}

	// Create cipher
	cipher, err := crypto.NewAES256GCMFromBase64(keyBase64)
	if err != nil {
		log.Fatalf("Failed to create cipher: %v", err)
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Register encryption plugin
	encryptionPlugin := crypto.NewEncryptionPlugin(cipher)
	if err := db.Use(encryptionPlugin); err != nil {
		log.Fatalf("Failed to register encryption plugin: %v", err)
	}

	log.Println("Encryption plugin registered successfully")
}
```

### 2. Tag Sensitive Fields in Models

```go
package model

import (
	"database/sql"
	"time"
)

type TravelRuleData struct {
	ID        string `gorm:"primaryKey"`
	PaymentID string

	// Encrypted fields (marked with encrypt:"true")
	PayerFullName   string         `gorm:"type:text" encrypt:"true"`
	PayerIDDocument sql.NullString `gorm:"type:text" encrypt:"true"`
	PayerAddress    sql.NullString `gorm:"type:text" encrypt:"true"`

	// Non-encrypted fields
	PayerCountry string `gorm:"type:varchar(2)"`
	CreatedAt    time.Time
}
```

### 3. CRUD Operations

Encryption/decryption happens automatically:

```go
// CREATE - Automatically encrypts before saving
travelRuleData := &model.TravelRuleData{
	ID:            uuid.New().String(),
	PaymentID:     paymentID,
	PayerFullName: "John Doe",              // Will be encrypted in DB
	PayerIDDocument: sql.NullString{
		String: "AB1234567",                 // Will be encrypted in DB
		Valid:  true,
	},
	PayerCountry: "US",                      // NOT encrypted
}

result := db.Create(travelRuleData)
if result.Error != nil {
	log.Fatalf("Failed to create: %v", result.Error)
}

// READ - Automatically decrypts after loading
var loaded model.TravelRuleData
result = db.First(&loaded, "id = ?", travelRuleData.ID)
if result.Error != nil {
	log.Fatalf("Failed to load: %v", result.Error)
}

fmt.Println(loaded.PayerFullName) // "John Doe" (decrypted)

// UPDATE - Automatically encrypts before updating
loaded.PayerFullName = "Jane Doe"
result = db.Save(&loaded)
if result.Error != nil {
	log.Fatalf("Failed to update: %v", result.Error)
}
```

## Database Storage

Encrypted data is stored as base64-encoded strings:

```sql
SELECT payer_full_name, payer_id_document FROM travel_rule_data LIMIT 1;

-- payer_full_name | payer_id_document
-- -----------------+-------------------
-- aGVsbG8gd29ybGQ... | eW91ciBlbmNyeXB0...  (encrypted, unreadable)
```

## Manual Encryption/Decryption

For non-GORM use cases (e.g., raw SQL):

```go
package main

import (
	"fmt"
	"log"

	"github.com/hxuan190/stable_payment_gateway/internal/pkg/crypto"
)

func main() {
	// Load cipher
	cipher, err := crypto.NewAES256GCMFromBase64(keyBase64)
	if err != nil {
		log.Fatal(err)
	}

	// Encrypt
	plaintext := "John Doe"
	ciphertext, err := cipher.Encrypt(plaintext)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Encrypted: %s\n", ciphertext) // Base64 string

	// Decrypt
	decrypted, err := cipher.Decrypt(ciphertext)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decrypted: %s\n", decrypted) // "John Doe"
}
```

## Security Best Practices

1. **Key Management**:
   - Generate key ONCE during initial setup
   - Store in AWS Secrets Manager / HashiCorp Vault (NOT in .env file in production)
   - Rotate key annually (requires re-encrypting all data)

2. **Field Selection**:
   - Encrypt: Full name, ID document number, address, date of birth
   - DO NOT encrypt: UUIDs, payment IDs, timestamps, amounts, countries

3. **Performance**:
   - Encryption adds ~1ms per field
   - For bulk operations (1000+ records), consider batch processing

4. **Backup & Recovery**:
   - ALWAYS backup the encryption key
   - If key is lost, encrypted data is PERMANENTLY unrecoverable
   - Store key in multiple secure locations

5. **Compliance**:
   - GDPR: Right to erasure (delete encrypted data when requested)
   - CCPA: Right to know (decrypt and provide data to user)
   - FATF: Retain encrypted data for 7 years minimum

## Migration Strategy

If you have existing unencrypted data:

```go
// Run migration to encrypt existing data
func MigrateEncryptExistingData(db *gorm.DB, cipher *crypto.AES256GCM) error {
	var records []model.TravelRuleData

	// Load all records
	if err := db.Find(&records).Error; err != nil {
		return err
	}

	// Encrypt each record manually
	for i := range records {
		// Check if already encrypted (encrypted data is base64, always ends with =)
		if isAlreadyEncrypted(records[i].PayerFullName) {
			continue
		}

		// Encrypt
		encrypted, err := cipher.Encrypt(records[i].PayerFullName)
		if err != nil {
			return err
		}

		// Update directly (bypass GORM hooks to avoid double-encryption)
		if err := db.Model(&records[i]).UpdateColumn("payer_full_name", encrypted).Error; err != nil {
			return err
		}
	}

	return nil
}

func isAlreadyEncrypted(value string) bool {
	// Simple heuristic: base64 strings typically have '=' padding
	// More robust: try to decrypt and check for error
	_, err := base64.StdEncoding.DecodeString(value)
	return err == nil
}
```

## Troubleshooting

**Error: "decryption failed"**
- Cause: Data was encrypted with a different key
- Solution: Ensure ENCRYPTION_KEY_BASE64 matches the key used to encrypt

**Error: "encryption key must be 32 bytes"**
- Cause: Invalid key length
- Solution: Regenerate key using `GenerateKeyBase64()`

**Performance degradation**
- Cause: Encrypting non-sensitive fields (e.g., UUIDs, amounts)
- Solution: Only tag truly sensitive PII fields with `encrypt:"true"`

## Testing

```go
package crypto_test

import (
	"testing"

	"github.com/hxuan190/stable_payment_gateway/internal/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	// Generate test key
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	// Create cipher
	cipher, err := crypto.NewAES256GCM(key)
	require.NoError(t, err)

	// Test encryption
	plaintext := "Sensitive Data"
	ciphertext, err := cipher.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	// Test decryption
	decrypted, err := cipher.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}
```

## License

Internal use only - Proprietary
