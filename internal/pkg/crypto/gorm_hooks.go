package crypto

import (
	"database/sql"
	"fmt"
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// EncryptedString is a custom type for encrypted string fields
// GORM will automatically encrypt on save and decrypt on load
type EncryptedString struct {
	sql.NullString
}

// EncryptedBytes is a custom type for encrypted byte fields
type EncryptedBytes struct {
	Valid bool
	Data  []byte
}

// Scan implements sql.Scanner for EncryptedBytes
func (e *EncryptedBytes) Scan(value interface{}) error {
	if value == nil {
		e.Valid = false
		e.Data = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		e.Data = v
		e.Valid = true
	case string:
		e.Data = []byte(v)
		e.Valid = true
	default:
		return fmt.Errorf("unsupported scan type for EncryptedBytes: %T", value)
	}

	return nil
}

// Value implements driver.Valuer for EncryptedBytes
func (e EncryptedBytes) Value() (interface{}, error) {
	if !e.Valid {
		return nil, nil
	}
	return e.Data, nil
}

// EncryptionPlugin provides automatic encryption/decryption for GORM models
type EncryptionPlugin struct {
	cipher *AES256GCM
}

// NewEncryptionPlugin creates a new GORM encryption plugin
func NewEncryptionPlugin(cipher *AES256GCM) *EncryptionPlugin {
	return &EncryptionPlugin{
		cipher: cipher,
	}
}

// Name returns the plugin name
func (p *EncryptionPlugin) Name() string {
	return "encryption_plugin"
}

// Initialize initializes the plugin
func (p *EncryptionPlugin) Initialize(db *gorm.DB) error {
	// Register BeforeCreate callback for encryption
	err := db.Callback().Create().Before("gorm:create").Register("encryption:before_create", p.beforeCreate)
	if err != nil {
		return fmt.Errorf("failed to register before_create callback: %w", err)
	}

	// Register BeforeUpdate callback for encryption
	err = db.Callback().Update().Before("gorm:update").Register("encryption:before_update", p.beforeUpdate)
	if err != nil {
		return fmt.Errorf("failed to register before_update callback: %w", err)
	}

	// Register AfterFind callback for decryption
	err = db.Callback().Query().After("gorm:after_query").Register("encryption:after_find", p.afterFind)
	if err != nil {
		return fmt.Errorf("failed to register after_find callback: %w", err)
	}

	return nil
}

// beforeCreate encrypts fields before creating a record
func (p *EncryptionPlugin) beforeCreate(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}

	// Encrypt all fields tagged with `encrypt:"true"`
	for _, field := range db.Statement.Schema.Fields {
		if shouldEncrypt, ok := field.Tag.Lookup("encrypt"); ok && shouldEncrypt == "true" {
			p.encryptField(db, field)
		}
	}
}

// beforeUpdate encrypts fields before updating a record
func (p *EncryptionPlugin) beforeUpdate(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}

	// Encrypt all fields tagged with `encrypt:"true"`
	for _, field := range db.Statement.Schema.Fields {
		if shouldEncrypt, ok := field.Tag.Lookup("encrypt"); ok && shouldEncrypt == "true" {
			p.encryptField(db, field)
		}
	}
}

// afterFind decrypts fields after loading a record
func (p *EncryptionPlugin) afterFind(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}

	// Decrypt all fields tagged with `encrypt:"true"`
	for _, field := range db.Statement.Schema.Fields {
		if shouldEncrypt, ok := field.Tag.Lookup("encrypt"); ok && shouldEncrypt == "true" {
			p.decryptField(db, field)
		}
	}
}

// encryptField encrypts a single field
func (p *EncryptionPlugin) encryptField(db *gorm.DB, field *schema.Field) {
	// Get the field value
	fieldValue, isZero := field.ValueOf(db.Statement.Context, db.Statement.ReflectValue)
	if isZero {
		return
	}

	// Handle different field types
	switch v := fieldValue.(type) {
	case string:
		if v == "" {
			return
		}
		encrypted, err := p.cipher.Encrypt(v)
		if err != nil {
			db.AddError(fmt.Errorf("failed to encrypt field %s: %w", field.Name, err))
			return
		}
		field.Set(db.Statement.Context, db.Statement.ReflectValue, encrypted)

	case sql.NullString:
		if !v.Valid || v.String == "" {
			return
		}
		encrypted, err := p.cipher.Encrypt(v.String)
		if err != nil {
			db.AddError(fmt.Errorf("failed to encrypt field %s: %w", field.Name, err))
			return
		}
		field.Set(db.Statement.Context, db.Statement.ReflectValue, sql.NullString{String: encrypted, Valid: true})

	case *string:
		if v == nil || *v == "" {
			return
		}
		encrypted, err := p.cipher.Encrypt(*v)
		if err != nil {
			db.AddError(fmt.Errorf("failed to encrypt field %s: %w", field.Name, err))
			return
		}
		field.Set(db.Statement.Context, db.Statement.ReflectValue, &encrypted)

	case []byte:
		if len(v) == 0 {
			return
		}
		encrypted, err := p.cipher.EncryptBytes(v)
		if err != nil {
			db.AddError(fmt.Errorf("failed to encrypt field %s: %w", field.Name, err))
			return
		}
		field.Set(db.Statement.Context, db.Statement.ReflectValue, encrypted)
	}
}

// decryptField decrypts a single field
func (p *EncryptionPlugin) decryptField(db *gorm.DB, field *schema.Field) {
	// Get the field value
	fieldValue, isZero := field.ValueOf(db.Statement.Context, db.Statement.ReflectValue)
	if isZero {
		return
	}

	// Handle different field types
	switch v := fieldValue.(type) {
	case string:
		if v == "" {
			return
		}
		decrypted, err := p.cipher.Decrypt(v)
		if err != nil {
			db.AddError(fmt.Errorf("failed to decrypt field %s: %w", field.Name, err))
			return
		}
		field.Set(db.Statement.Context, db.Statement.ReflectValue, decrypted)

	case sql.NullString:
		if !v.Valid || v.String == "" {
			return
		}
		decrypted, err := p.cipher.Decrypt(v.String)
		if err != nil {
			db.AddError(fmt.Errorf("failed to decrypt field %s: %w", field.Name, err))
			return
		}
		field.Set(db.Statement.Context, db.Statement.ReflectValue, sql.NullString{String: decrypted, Valid: true})

	case *string:
		if v == nil || *v == "" {
			return
		}
		decrypted, err := p.cipher.Decrypt(*v)
		if err != nil {
			db.AddError(fmt.Errorf("failed to decrypt field %s: %w", field.Name, err))
			return
		}
		field.Set(db.Statement.Context, db.Statement.ReflectValue, &decrypted)

	case []byte:
		if len(v) == 0 {
			return
		}
		decrypted, err := p.cipher.DecryptBytes(string(v))
		if err != nil {
			db.AddError(fmt.Errorf("failed to decrypt field %s: %w", field.Name, err))
			return
		}
		field.Set(db.Statement.Context, db.Statement.ReflectValue, decrypted)
	}
}

// EncryptModel encrypts all fields tagged with `encrypt:"true"` in a model
// This is a utility function for manual encryption outside of GORM hooks
func EncryptModel(cipher *AES256GCM, model interface{}) error {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("model must be a struct, got %T", model)
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Check for encrypt tag
		if encryptTag := fieldType.Tag.Get("encrypt"); encryptTag != "true" {
			continue
		}

		// Encrypt based on field type
		switch field.Kind() {
		case reflect.String:
			if field.String() == "" {
				continue
			}
			encrypted, err := cipher.Encrypt(field.String())
			if err != nil {
				return fmt.Errorf("failed to encrypt field %s: %w", fieldType.Name, err)
			}
			field.SetString(encrypted)
		}
	}

	return nil
}

// DecryptModel decrypts all fields tagged with `encrypt:"true"` in a model
// This is a utility function for manual decryption outside of GORM hooks
func DecryptModel(cipher *AES256GCM, model interface{}) error {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("model must be a struct, got %T", model)
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Check for encrypt tag
		if encryptTag := fieldType.Tag.Get("encrypt"); encryptTag != "true" {
			continue
		}

		// Decrypt based on field type
		switch field.Kind() {
		case reflect.String:
			if field.String() == "" {
				continue
			}
			decrypted, err := cipher.Decrypt(field.String())
			if err != nil {
				return fmt.Errorf("failed to decrypt field %s: %w", fieldType.Name, err)
			}
			field.SetString(decrypted)
		}
	}

	return nil
}
