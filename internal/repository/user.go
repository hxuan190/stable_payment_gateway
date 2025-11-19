package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/shopspring/decimal"
)

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrUserAlreadyExists is returned when a user with the same email already exists
	ErrUserAlreadyExists = errors.New("user with this email already exists")
	// ErrInvalidUserID is returned when the user ID is invalid
	ErrInvalidUserID = errors.New("invalid user ID")
)

// UserRepository handles database operations for users
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// Create inserts a new user into the database
func (r *UserRepository) Create(user *model.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	// Generate UUID if not provided
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	query := `
		INSERT INTO users (
			id, full_name, email, phone_number, date_of_birth,
			nationality, document_type, document_number, document_issued_date, document_expiry_date,
			kyc_status, kyc_applicant_id, kyc_submitted_at, kyc_verified_at,
			face_liveness_passed, face_liveness_score, face_liveness_verified_at,
			is_pep, is_sanctioned, sanction_list_checked_at,
			risk_level, risk_score, risk_assessed_at,
			total_payment_count, total_payment_volume_usd,
			last_payment_at, last_login_at,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30
		)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	err := r.db.QueryRow(
		query,
		user.ID,
		user.FullName,
		user.Email,
		user.PhoneNumber,
		user.DateOfBirth,
		user.Nationality,
		user.DocumentType,
		user.DocumentNumber,
		user.DocumentIssuedDate,
		user.DocumentExpiryDate,
		user.KYCStatus,
		user.KYCApplicantID,
		user.KYCSubmittedAt,
		user.KYCVerifiedAt,
		user.FaceLivenessPassed,
		user.FaceLivenessScore,
		user.FaceLivenessVerifiedAt,
		user.IsPEP,
		user.IsSanctioned,
		user.SanctionListCheckedAt,
		user.RiskLevel,
		user.RiskScore,
		user.RiskAssessedAt,
		user.TotalPaymentCount,
		user.TotalPaymentVolumeUSD,
		user.LastPaymentAt,
		user.LastLoginAt,
		user.Metadata,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation
		if err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id uuid.UUID) (*model.User, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidUserID
	}

	query := `
		SELECT
			id, full_name, email, phone_number, date_of_birth,
			nationality, document_type, document_number, document_issued_date, document_expiry_date,
			kyc_status, kyc_applicant_id, kyc_submitted_at, kyc_verified_at, kyc_rejection_reason,
			face_liveness_passed, face_liveness_score, face_liveness_verified_at,
			is_pep, is_sanctioned, sanction_list_checked_at,
			risk_level, risk_score, risk_assessed_at,
			total_payment_count, total_payment_volume_usd,
			last_payment_at, last_login_at,
			metadata, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &model.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PhoneNumber,
		&user.DateOfBirth,
		&user.Nationality,
		&user.DocumentType,
		&user.DocumentNumber,
		&user.DocumentIssuedDate,
		&user.DocumentExpiryDate,
		&user.KYCStatus,
		&user.KYCApplicantID,
		&user.KYCSubmittedAt,
		&user.KYCVerifiedAt,
		&user.KYCRejectionReason,
		&user.FaceLivenessPassed,
		&user.FaceLivenessScore,
		&user.FaceLivenessVerifiedAt,
		&user.IsPEP,
		&user.IsSanctioned,
		&user.SanctionListCheckedAt,
		&user.RiskLevel,
		&user.RiskScore,
		&user.RiskAssessedAt,
		&user.TotalPaymentCount,
		&user.TotalPaymentVolumeUSD,
		&user.LastPaymentAt,
		&user.LastLoginAt,
		&user.Metadata,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*model.User, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	query := `
		SELECT
			id, full_name, email, phone_number, date_of_birth,
			nationality, document_type, document_number, document_issued_date, document_expiry_date,
			kyc_status, kyc_applicant_id, kyc_submitted_at, kyc_verified_at, kyc_rejection_reason,
			face_liveness_passed, face_liveness_score, face_liveness_verified_at,
			is_pep, is_sanctioned, sanction_list_checked_at,
			risk_level, risk_score, risk_assessed_at,
			total_payment_count, total_payment_volume_usd,
			last_payment_at, last_login_at,
			metadata, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &model.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PhoneNumber,
		&user.DateOfBirth,
		&user.Nationality,
		&user.DocumentType,
		&user.DocumentNumber,
		&user.DocumentIssuedDate,
		&user.DocumentExpiryDate,
		&user.KYCStatus,
		&user.KYCApplicantID,
		&user.KYCSubmittedAt,
		&user.KYCVerifiedAt,
		&user.KYCRejectionReason,
		&user.FaceLivenessPassed,
		&user.FaceLivenessScore,
		&user.FaceLivenessVerifiedAt,
		&user.IsPEP,
		&user.IsSanctioned,
		&user.SanctionListCheckedAt,
		&user.RiskLevel,
		&user.RiskScore,
		&user.RiskAssessedAt,
		&user.TotalPaymentCount,
		&user.TotalPaymentVolumeUSD,
		&user.LastPaymentAt,
		&user.LastLoginAt,
		&user.Metadata,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetByKYCApplicantID retrieves a user by Sumsub applicant ID
func (r *UserRepository) GetByKYCApplicantID(applicantID string) (*model.User, error) {
	if applicantID == "" {
		return nil, errors.New("KYC applicant ID cannot be empty")
	}

	query := `
		SELECT
			id, full_name, email, phone_number, date_of_birth,
			nationality, document_type, document_number, document_issued_date, document_expiry_date,
			kyc_status, kyc_applicant_id, kyc_submitted_at, kyc_verified_at, kyc_rejection_reason,
			face_liveness_passed, face_liveness_score, face_liveness_verified_at,
			is_pep, is_sanctioned, sanction_list_checked_at,
			risk_level, risk_score, risk_assessed_at,
			total_payment_count, total_payment_volume_usd,
			last_payment_at, last_login_at,
			metadata, created_at, updated_at
		FROM users
		WHERE kyc_applicant_id = $1
	`

	user := &model.User{}
	err := r.db.QueryRow(query, applicantID).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PhoneNumber,
		&user.DateOfBirth,
		&user.Nationality,
		&user.DocumentType,
		&user.DocumentNumber,
		&user.DocumentIssuedDate,
		&user.DocumentExpiryDate,
		&user.KYCStatus,
		&user.KYCApplicantID,
		&user.KYCSubmittedAt,
		&user.KYCVerifiedAt,
		&user.KYCRejectionReason,
		&user.FaceLivenessPassed,
		&user.FaceLivenessScore,
		&user.FaceLivenessVerifiedAt,
		&user.IsPEP,
		&user.IsSanctioned,
		&user.SanctionListCheckedAt,
		&user.RiskLevel,
		&user.RiskScore,
		&user.RiskAssessedAt,
		&user.TotalPaymentCount,
		&user.TotalPaymentVolumeUSD,
		&user.LastPaymentAt,
		&user.LastLoginAt,
		&user.Metadata,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by KYC applicant ID: %w", err)
	}

	return user, nil
}

// Update updates an existing user
func (r *UserRepository) Update(user *model.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}
	if user.ID == uuid.Nil {
		return ErrInvalidUserID
	}

	query := `
		UPDATE users SET
			full_name = $2,
			email = $3,
			phone_number = $4,
			date_of_birth = $5,
			nationality = $6,
			document_type = $7,
			document_number = $8,
			document_issued_date = $9,
			document_expiry_date = $10,
			kyc_status = $11,
			kyc_applicant_id = $12,
			kyc_submitted_at = $13,
			kyc_verified_at = $14,
			kyc_rejection_reason = $15,
			face_liveness_passed = $16,
			face_liveness_score = $17,
			face_liveness_verified_at = $18,
			is_pep = $19,
			is_sanctioned = $20,
			sanction_list_checked_at = $21,
			risk_level = $22,
			risk_score = $23,
			risk_assessed_at = $24,
			total_payment_count = $25,
			total_payment_volume_usd = $26,
			last_payment_at = $27,
			last_login_at = $28,
			metadata = $29,
			updated_at = $30
		WHERE id = $1
		RETURNING updated_at
	`

	user.UpdatedAt = time.Now()

	err := r.db.QueryRow(
		query,
		user.ID,
		user.FullName,
		user.Email,
		user.PhoneNumber,
		user.DateOfBirth,
		user.Nationality,
		user.DocumentType,
		user.DocumentNumber,
		user.DocumentIssuedDate,
		user.DocumentExpiryDate,
		user.KYCStatus,
		user.KYCApplicantID,
		user.KYCSubmittedAt,
		user.KYCVerifiedAt,
		user.KYCRejectionReason,
		user.FaceLivenessPassed,
		user.FaceLivenessScore,
		user.FaceLivenessVerifiedAt,
		user.IsPEP,
		user.IsSanctioned,
		user.SanctionListCheckedAt,
		user.RiskLevel,
		user.RiskScore,
		user.RiskAssessedAt,
		user.TotalPaymentCount,
		user.TotalPaymentVolumeUSD,
		user.LastPaymentAt,
		user.LastLoginAt,
		user.Metadata,
		user.UpdatedAt,
	).Scan(&user.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdateKYCStatus updates the KYC status of a user
func (r *UserRepository) UpdateKYCStatus(id uuid.UUID, status model.KYCStatus, rejectionReason *string) error {
	if id == uuid.Nil {
		return ErrInvalidUserID
	}

	var query string
	var args []interface{}

	if status == model.KYCStatusApproved {
		query = `
			UPDATE users
			SET kyc_status = $2, kyc_verified_at = $3, kyc_rejection_reason = NULL, updated_at = $4
			WHERE id = $1
		`
		now := time.Now()
		args = []interface{}{id, status, sql.NullTime{Time: now, Valid: true}, now}
	} else if status == model.KYCStatusRejected && rejectionReason != nil {
		query = `
			UPDATE users
			SET kyc_status = $2, kyc_rejection_reason = $3, updated_at = $4
			WHERE id = $1
		`
		args = []interface{}{id, status, sql.NullString{String: *rejectionReason, Valid: true}, time.Now()}
	} else {
		query = `
			UPDATE users
			SET kyc_status = $2, updated_at = $3
			WHERE id = $1
		`
		args = []interface{}{id, status, time.Now()}
	}

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update KYC status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// IncrementPaymentStats increments payment count and volume for a user
func (r *UserRepository) IncrementPaymentStats(id uuid.UUID, amountUSD decimal.Decimal) error {
	if id == uuid.Nil {
		return ErrInvalidUserID
	}

	query := `
		UPDATE users
		SET
			total_payment_count = total_payment_count + 1,
			total_payment_volume_usd = total_payment_volume_usd + $2,
			last_payment_at = $3,
			updated_at = $3
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.db.Exec(query, id, amountUSD, now)
	if err != nil {
		return fmt.Errorf("failed to increment payment stats: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// ListByKYCStatus retrieves users by KYC status with pagination
func (r *UserRepository) ListByKYCStatus(status model.KYCStatus, limit, offset int) ([]*model.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, full_name, email, phone_number, date_of_birth,
			nationality, document_type, document_number, document_issued_date, document_expiry_date,
			kyc_status, kyc_applicant_id, kyc_submitted_at, kyc_verified_at, kyc_rejection_reason,
			face_liveness_passed, face_liveness_score, face_liveness_verified_at,
			is_pep, is_sanctioned, sanction_list_checked_at,
			risk_level, risk_score, risk_assessed_at,
			total_payment_count, total_payment_volume_usd,
			last_payment_at, last_login_at,
			metadata, created_at, updated_at
		FROM users
		WHERE kyc_status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users by KYC status: %w", err)
	}
	defer rows.Close()

	users := make([]*model.User, 0)
	for rows.Next() {
		user := &model.User{}
		err := rows.Scan(
			&user.ID,
			&user.FullName,
			&user.Email,
			&user.PhoneNumber,
			&user.DateOfBirth,
			&user.Nationality,
			&user.DocumentType,
			&user.DocumentNumber,
			&user.DocumentIssuedDate,
			&user.DocumentExpiryDate,
			&user.KYCStatus,
			&user.KYCApplicantID,
			&user.KYCSubmittedAt,
			&user.KYCVerifiedAt,
			&user.KYCRejectionReason,
			&user.FaceLivenessPassed,
			&user.FaceLivenessScore,
			&user.FaceLivenessVerifiedAt,
			&user.IsPEP,
			&user.IsSanctioned,
			&user.SanctionListCheckedAt,
			&user.RiskLevel,
			&user.RiskScore,
			&user.RiskAssessedAt,
			&user.TotalPaymentCount,
			&user.TotalPaymentVolumeUSD,
			&user.LastPaymentAt,
			&user.LastLoginAt,
			&user.Metadata,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// ListHighRisk retrieves high-risk users (PEP or sanctioned or high risk level)
func (r *UserRepository) ListHighRisk(limit, offset int) ([]*model.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, full_name, email, phone_number, date_of_birth,
			nationality, document_type, document_number, document_issued_date, document_expiry_date,
			kyc_status, kyc_applicant_id, kyc_submitted_at, kyc_verified_at, kyc_rejection_reason,
			face_liveness_passed, face_liveness_score, face_liveness_verified_at,
			is_pep, is_sanctioned, sanction_list_checked_at,
			risk_level, risk_score, risk_assessed_at,
			total_payment_count, total_payment_volume_usd,
			last_payment_at, last_login_at,
			metadata, created_at, updated_at
		FROM users
		WHERE is_pep = TRUE OR is_sanctioned = TRUE OR risk_level = 'high'
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list high-risk users: %w", err)
	}
	defer rows.Close()

	users := make([]*model.User, 0)
	for rows.Next() {
		user := &model.User{}
		err := rows.Scan(
			&user.ID,
			&user.FullName,
			&user.Email,
			&user.PhoneNumber,
			&user.DateOfBirth,
			&user.Nationality,
			&user.DocumentType,
			&user.DocumentNumber,
			&user.DocumentIssuedDate,
			&user.DocumentExpiryDate,
			&user.KYCStatus,
			&user.KYCApplicantID,
			&user.KYCSubmittedAt,
			&user.KYCVerifiedAt,
			&user.KYCRejectionReason,
			&user.FaceLivenessPassed,
			&user.FaceLivenessScore,
			&user.FaceLivenessVerifiedAt,
			&user.IsPEP,
			&user.IsSanctioned,
			&user.SanctionListCheckedAt,
			&user.RiskLevel,
			&user.RiskScore,
			&user.RiskAssessedAt,
			&user.TotalPaymentCount,
			&user.TotalPaymentVolumeUSD,
			&user.LastPaymentAt,
			&user.LastLoginAt,
			&user.Metadata,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// Count returns the total number of users
func (r *UserRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM users`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}
