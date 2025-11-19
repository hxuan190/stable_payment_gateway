package model

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// StorageClass represents S3 storage class
type StorageClass string

const (
	StorageClassGlacier             StorageClass = "GLACIER"
	StorageClassDeepArchive         StorageClass = "DEEP_ARCHIVE"
	StorageClassIntelligentTiering  StorageClass = "INTELLIGENT_TIERING"
)

// RestoreTier represents Glacier restore speed
type RestoreTier string

const (
	RestoreTierExpedited RestoreTier = "Expedited" // 1-5 hours
	RestoreTierStandard  RestoreTier = "Standard"  // 3-5 hours
	RestoreTierBulk      RestoreTier = "Bulk"      // 5-12 hours
)

// ArchivedRecord stores metadata for data archived to S3 Glacier (PRD v2.2 - Data Retention)
type ArchivedRecord struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	// Original Record Information
	OriginalID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_table_original" json:"original_id"`
	TableName  string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_table_original;index" json:"table_name"`

	// Archive Storage Information
	ArchivePath  string       `gorm:"type:varchar(500);not null;index" json:"archive_path"`
	S3Bucket     string       `gorm:"type:varchar(255);not null" json:"s3_bucket"`
	S3Region     string       `gorm:"type:varchar(50);not null;default:'ap-southeast-1'" json:"s3_region"`
	StorageClass StorageClass `gorm:"type:varchar(50);default:'GLACIER'" json:"storage_class"`

	// Data Integrity
	DataHash              string `gorm:"type:varchar(64);not null;index" json:"data_hash"` // SHA-256
	CompressionAlgorithm  string `gorm:"type:varchar(20);default:'gzip'" json:"compression_algorithm"`
	Encrypted             bool   `gorm:"default:true" json:"encrypted"`
	EncryptionAlgorithm   string `gorm:"type:varchar(50);default:'AES256'" json:"encryption_algorithm"`

	// Archive Size
	OriginalSizeBytes   int64 `gorm:"type:bigint" json:"original_size_bytes"`
	CompressedSizeBytes int64 `gorm:"type:bigint" json:"compressed_size_bytes"`
	ObjectSizeBytes     int64 `gorm:"type:bigint" json:"object_size_bytes"`

	// Archive Lifecycle
	ArchivedAt sql.NullTime   `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;index" json:"archived_at,omitempty"`
	ArchivedBy sql.NullString `gorm:"type:varchar(100)" json:"archived_by,omitempty"`
	ArchiveJobID sql.NullString `gorm:"type:varchar(100)" json:"archive_job_id,omitempty"`

	// Restore Information
	RestoreInitiatedAt sql.NullTime   `gorm:"type:timestamp;index" json:"restore_initiated_at,omitempty"`
	RestoreExpiresAt   sql.NullTime   `gorm:"type:timestamp" json:"restore_expires_at,omitempty"`
	RestoreTier        sql.NullString `gorm:"type:varchar(20)" json:"restore_tier,omitempty"`
	RestoredAt         sql.NullTime   `gorm:"type:timestamp" json:"restored_at,omitempty"`
	RestoreInitiatedBy sql.NullString `gorm:"type:varchar(100)" json:"restore_initiated_by,omitempty"`

	// Metadata
	RecordDataSample      map[string]interface{} `gorm:"type:jsonb" json:"record_data_sample,omitempty"` // Small sample
	Metadata              map[string]interface{} `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`

	// Audit fields
	CreatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name
func (ArchivedRecord) TableName() string {
	return "archived_records"
}

// BeforeCreate hook
func (ar *ArchivedRecord) BeforeCreate() error {
	if ar.ID == uuid.Nil {
		ar.ID = uuid.New()
	}
	return nil
}

// GetCompressionRatio returns the compression ratio as percentage
func (ar *ArchivedRecord) GetCompressionRatio() float64 {
	if ar.OriginalSizeBytes == 0 {
		return 0
	}
	return float64(ar.CompressedSizeBytes) / float64(ar.OriginalSizeBytes) * 100
}

// IsRestoreInProgress returns true if restore is in progress
func (ar *ArchivedRecord) IsRestoreInProgress() bool {
	return ar.RestoreInitiatedAt.Valid && (!ar.RestoredAt.Valid || ar.RestoredAt.Time.After(time.Now()))
}

// IsRestoreExpired returns true if restore has expired
func (ar *ArchivedRecord) IsRestoreExpired() bool {
	return ar.RestoreExpiresAt.Valid && ar.RestoreExpiresAt.Time.Before(time.Now())
}

// InitiateRestore marks this record for Glacier restore
func (ar *ArchivedRecord) InitiateRestore(tier RestoreTier, initiatedBy string) {
	ar.RestoreInitiatedAt = sql.NullTime{Time: time.Now(), Valid: true}
	ar.RestoreTier = sql.NullString{String: string(tier), Valid: true}
	ar.RestoreInitiatedBy = sql.NullString{String: initiatedBy, Valid: true}

	// Set expiry based on tier (Expedited: 1-5 hours, Standard: 3-5 hours, Bulk: 5-12 hours)
	var hours int
	switch tier {
	case RestoreTierExpedited:
		hours = 5
	case RestoreTierStandard:
		hours = 5
	case RestoreTierBulk:
		hours = 12
	default:
		hours = 5
	}
	ar.RestoreExpiresAt = sql.NullTime{Time: time.Now().Add(time.Duration(hours) * time.Hour), Valid: true}
}

// MarkAsRestored marks this record as successfully restored
func (ar *ArchivedRecord) MarkAsRestored() {
	ar.RestoredAt = sql.NullTime{Time: time.Now(), Valid: true}
}

// TransactionHash stores SHA-256 hash chain for transaction immutability
type TransactionHash struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	// Hash Chain Information
	TableName    string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_table_record" json:"table_name"`
	RecordID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_table_record" json:"record_id"`
	DataHash     string    `gorm:"type:varchar(64);not null;index" json:"data_hash"`      // SHA-256 of record
	PreviousHash sql.NullString `gorm:"type:varchar(64);index" json:"previous_hash,omitempty"` // Hash of previous record

	// Merkle Tree
	MerkleRoot  sql.NullString `gorm:"type:varchar(64);index" json:"merkle_root,omitempty"` // Daily Merkle root
	MerkleProof pq.StringArray `gorm:"type:text[]" json:"merkle_proof,omitempty"`           // Merkle proof path

	// Block Information
	BlockNumber sql.NullInt64 `gorm:"type:bigint" json:"block_number,omitempty"`
	BlockDate   time.Time     `gorm:"type:date;not null;index" json:"block_date"`

	// Record Snapshot
	RecordSnapshot map[string]interface{} `gorm:"type:jsonb" json:"record_snapshot,omitempty"`

	// Verification
	Verified   bool         `gorm:"default:false;index" json:"verified"`
	VerifiedAt sql.NullTime `gorm:"type:timestamp" json:"verified_at,omitempty"`

	// Audit fields
	CreatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;index" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name
func (TransactionHash) TableName() string {
	return "transaction_hashes"
}

// BeforeCreate hook
func (th *TransactionHash) BeforeCreate() error {
	if th.ID == uuid.Nil {
		th.ID = uuid.New()
	}
	if th.BlockDate.IsZero() {
		th.BlockDate = time.Now()
	}
	return nil
}

// MarkAsVerified marks this hash as verified
func (th *TransactionHash) MarkAsVerified() {
	th.Verified = true
	th.VerifiedAt = sql.NullTime{Time: time.Now(), Valid: true}
}

// SetMerkleRoot sets the daily Merkle tree root hash
func (th *TransactionHash) SetMerkleRoot(root string) {
	th.MerkleRoot = sql.NullString{String: root, Valid: true}
}

// AddMerkleProof adds a hash to the Merkle proof path
func (th *TransactionHash) AddMerkleProof(proof []string) {
	th.MerkleProof = proof
}

// IsPartOfChain returns true if this hash is linked to previous hash
func (th *TransactionHash) IsPartOfChain() bool {
	return th.PreviousHash.Valid
}

// HasMerkleRoot returns true if daily Merkle root is computed
func (th *TransactionHash) HasMerkleRoot() bool {
	return th.MerkleRoot.Valid
}
