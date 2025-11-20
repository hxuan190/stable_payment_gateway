package kyc

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

var (
	// ErrApplicantNotFound is returned when an applicant is not found
	ErrApplicantNotFound = errors.New("applicant not found")
	// ErrInvalidApplicantID is returned when the applicant ID is invalid
	ErrInvalidApplicantID = errors.New("invalid applicant ID")
)

// MockApplicant represents a mock KYC applicant
type MockApplicant struct {
	ID                 string
	FullName           string
	Email              string
	Status             model.KYCStatus
	FaceLivenessPassed bool
	FaceLivenessScore  float64
	CreatedAt          time.Time
	VerifiedAt         *time.Time
	Data               map[string]interface{}
}

// MockKYCProvider is a mock implementation of KYCProvider for testing and development
//
// This simulates Sumsub/Onfido behavior without external API calls
// Useful for:
// - Local development
// - Integration testing
// - Demo environments
type MockKYCProvider struct {
	mu               sync.RWMutex
	applicants       map[string]*MockApplicant
	autoApprove      bool
	autoFaceLiveness bool
	simulateDelay    bool
}

// NewMockKYCProvider creates a new mock KYC provider
//
// Options:
// - autoApprove: if true, applicants are automatically approved
// - autoFaceLiveness: if true, face liveness checks automatically pass
// - simulateDelay: if true, simulates realistic API delays (100-500ms)
func NewMockKYCProvider(autoApprove, autoFaceLiveness, simulateDelay bool) *MockKYCProvider {
	return &MockKYCProvider{
		applicants:       make(map[string]*MockApplicant),
		autoApprove:      autoApprove,
		autoFaceLiveness: autoFaceLiveness,
		simulateDelay:    simulateDelay,
	}
}

// CreateApplicant creates a new KYC applicant
func (m *MockKYCProvider) CreateApplicant(ctx context.Context, fullName, email string) (string, error) {
	if m.simulateDelay {
		m.delay()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate applicant ID (simulates Sumsub format)
	applicantID := fmt.Sprintf("mock-%s", uuid.New().String())

	// Create applicant
	applicant := &MockApplicant{
		ID:        applicantID,
		FullName:  fullName,
		Email:     email,
		Status:    model.KYCStatusPending,
		CreatedAt: time.Now(),
		Data: map[string]interface{}{
			"full_name": fullName,
			"email":     email,
		},
	}

	// Auto-approve if enabled
	if m.autoApprove {
		applicant.Status = model.KYCStatusApproved
		now := time.Now()
		applicant.VerifiedAt = &now
	}

	// Auto face liveness if enabled
	if m.autoFaceLiveness {
		applicant.FaceLivenessPassed = true
		applicant.FaceLivenessScore = 0.95 + rand.Float64()*0.05 // 0.95-1.0
	}

	m.applicants[applicantID] = applicant

	return applicantID, nil
}

// GetApplicantStatus retrieves the current KYC status
func (m *MockKYCProvider) GetApplicantStatus(ctx context.Context, applicantID string) (model.KYCStatus, error) {
	if m.simulateDelay {
		m.delay()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	applicant, exists := m.applicants[applicantID]
	if !exists {
		return "", ErrApplicantNotFound
	}

	return applicant.Status, nil
}

// VerifyFaceLiveness triggers face liveness check
func (m *MockKYCProvider) VerifyFaceLiveness(ctx context.Context, applicantID string) (bool, float64, error) {
	if m.simulateDelay {
		m.delay()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	applicant, exists := m.applicants[applicantID]
	if !exists {
		return false, 0, ErrApplicantNotFound
	}

	// If auto face liveness is enabled, always pass
	if m.autoFaceLiveness {
		applicant.FaceLivenessPassed = true
		applicant.FaceLivenessScore = 0.95 + rand.Float64()*0.05 // 0.95-1.0
		return true, applicant.FaceLivenessScore, nil
	}

	// Otherwise, randomly pass/fail (80% pass rate for testing)
	passed := rand.Float64() > 0.2
	score := 0.0
	if passed {
		score = 0.85 + rand.Float64()*0.15 // 0.85-1.0
	} else {
		score = 0.3 + rand.Float64()*0.4 // 0.3-0.7
	}

	applicant.FaceLivenessPassed = passed
	applicant.FaceLivenessScore = score

	return passed, score, nil
}

// GetApplicantData retrieves KYC data
func (m *MockKYCProvider) GetApplicantData(ctx context.Context, applicantID string) (map[string]interface{}, error) {
	if m.simulateDelay {
		m.delay()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	applicant, exists := m.applicants[applicantID]
	if !exists {
		return nil, ErrApplicantNotFound
	}

	return applicant.Data, nil
}

// UpdateApplicantStatus manually updates an applicant's status (for testing)
func (m *MockKYCProvider) UpdateApplicantStatus(applicantID string, status model.KYCStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	applicant, exists := m.applicants[applicantID]
	if !exists {
		return ErrApplicantNotFound
	}

	applicant.Status = status

	if status == model.KYCStatusApproved {
		now := time.Now()
		applicant.VerifiedAt = &now
	}

	return nil
}

// GetApplicant retrieves an applicant (for testing/debugging)
func (m *MockKYCProvider) GetApplicant(applicantID string) (*MockApplicant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	applicant, exists := m.applicants[applicantID]
	if !exists {
		return nil, ErrApplicantNotFound
	}

	return applicant, nil
}

// Reset clears all applicants (for testing)
func (m *MockKYCProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.applicants = make(map[string]*MockApplicant)
}

// Count returns the number of applicants (for testing)
func (m *MockKYCProvider) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.applicants)
}

// Helper methods

// delay simulates realistic API latency
func (m *MockKYCProvider) delay() {
	// Random delay between 100-500ms
	delay := time.Duration(100+rand.Intn(400)) * time.Millisecond
	time.Sleep(delay)
}
