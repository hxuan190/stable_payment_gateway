package kyc

import (
	"context"
	"errors"
	"fmt"

	merchantDomain "github.com/hxuan190/stable_payment_gateway/internal/modules/merchant/domain"
)

var (
	// ErrSumsubNotConfigured is returned when Sumsub credentials are missing
	ErrSumsubNotConfigured = errors.New("Sumsub credentials not configured")
)

// SumsubConfig holds Sumsub API configuration
type SumsubConfig struct {
	AppToken  string
	SecretKey string
	BaseURL   string
}

// SumsubProvider implements KYCProvider using Sumsub API
//
// Sumsub is the recommended KYC provider for PRD v2.2
// Features:
// - Face liveness detection
// - Document verification
// - AML screening
// - PEP checks
//
// Documentation: https://developers.sumsub.com/api-reference/
type SumsubProvider struct {
	config *SumsubConfig
}

// NewSumsubProvider creates a new Sumsub KYC provider
func NewSumsubProvider(config *SumsubConfig) (*SumsubProvider, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if config.AppToken == "" || config.SecretKey == "" {
		return nil, ErrSumsubNotConfigured
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.sumsub.com" // Production URL
	}

	return &SumsubProvider{
		config: config,
	}, nil
}

// CreateApplicant creates a new KYC applicant in Sumsub
//
// API: POST /resources/applicants
// Docs: https://developers.sumsub.com/api-reference/#creating-an-applicant
func (s *SumsubProvider) CreateApplicant(ctx context.Context, fullName, email string) (string, error) {
	// TODO: Implement Sumsub API integration
	// For now, return not implemented error
	return "", fmt.Errorf("Sumsub integration not yet implemented - use MockKYCProvider for development")

	/*
		Example implementation:

		// Create request payload
		payload := map[string]interface{}{
			"externalUserId": uuid.New().String(),
			"fixedInfo": map[string]interface{}{
				"firstName": extractFirstName(fullName),
				"lastName":  extractLastName(fullName),
			},
			"email": email,
		}

		// Make API request
		resp, err := s.makeRequest(ctx, "POST", "/resources/applicants", payload)
		if err != nil {
			return "", fmt.Errorf("failed to create Sumsub applicant: %w", err)
		}

		// Parse response
		var result struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			return "", fmt.Errorf("failed to parse Sumsub response: %w", err)
		}

		return result.ID, nil
	*/
}

// GetApplicantStatus retrieves the current KYC status from Sumsub
//
// API: GET /resources/applicants/{applicantId}/status
// Docs: https://developers.sumsub.com/api-reference/#getting-applicant-status
func (s *SumsubProvider) GetApplicantStatus(ctx context.Context, applicantID string) (merchantDomain.KYCStatus, error) {
	// TODO: Implement Sumsub API integration
	return "", fmt.Errorf("Sumsub integration not yet implemented - use MockKYCProvider for development")

	/*
		Example implementation:

		// Make API request
		resp, err := s.makeRequest(ctx, "GET", fmt.Sprintf("/resources/applicants/%s/status", applicantID), nil)
		if err != nil {
			return "", fmt.Errorf("failed to get Sumsub status: %w", err)
		}

		// Parse response
		var result struct {
			ReviewStatus string `json:"reviewStatus"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			return "", fmt.Errorf("failed to parse Sumsub response: %w", err)
		}

		// Map Sumsub status to our KYCStatus
		switch result.ReviewStatus {
		case "init", "pending":
			return merchantDomain.KYCStatusPending, nil
		case "onHold":
			return merchantDomain.KYCStatusInProgress, nil
		case "completed":
			return merchantDomain.KYCStatusApproved, nil
		case "rejected":
			return merchantDomain.KYCStatusRejected, nil
		default:
			return merchantDomain.KYCStatusPending, nil
		}
	*/
}

// VerifyFaceLiveness triggers face liveness check in Sumsub
//
// API: POST /resources/applicants/{applicantId}/liveness
// Docs: https://developers.sumsub.com/api-reference/#liveness-detection
func (s *SumsubProvider) VerifyFaceLiveness(ctx context.Context, applicantID string) (bool, float64, error) {
	// TODO: Implement Sumsub API integration
	return false, 0, fmt.Errorf("Sumsub integration not yet implemented - use MockKYCProvider for development")

	/*
		Example implementation:

		// Make API request
		resp, err := s.makeRequest(ctx, "GET", fmt.Sprintf("/resources/applicants/%s/liveness", applicantID), nil)
		if err != nil {
			return false, 0, fmt.Errorf("failed to get Sumsub liveness: %w", err)
		}

		// Parse response
		var result struct {
			Status string  `json:"status"`
			Score  float64 `json:"score"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			return false, 0, fmt.Errorf("failed to parse Sumsub response: %w", err)
		}

		passed := result.Status == "passed"
		return passed, result.Score, nil
	*/
}

// GetApplicantData retrieves KYC data from Sumsub
//
// API: GET /resources/applicants/{applicantId}/one
// Docs: https://developers.sumsub.com/api-reference/#getting-applicant-data
func (s *SumsubProvider) GetApplicantData(ctx context.Context, applicantID string) (map[string]interface{}, error) {
	// TODO: Implement Sumsub API integration
	return nil, fmt.Errorf("Sumsub integration not yet implemented - use MockKYCProvider for development")

	/*
		Example implementation:

		// Make API request
		resp, err := s.makeRequest(ctx, "GET", fmt.Sprintf("/resources/applicants/%s/one", applicantID), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get Sumsub data: %w", err)
		}

		// Parse response
		var data map[string]interface{}
		if err := json.Unmarshal(resp, &data); err != nil {
			return nil, fmt.Errorf("failed to parse Sumsub response: %w", err)
		}

		return data, nil
	*/
}

/*
// Helper method for making authenticated Sumsub API requests
func (s *SumsubProvider) makeRequest(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
	// TODO: Implement HMAC signature generation for Sumsub
	// Docs: https://developers.sumsub.com/api-reference/#app-tokens

	var body []byte
	var err error

	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	// Create request
	url := s.config.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Generate signature
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := s.generateSignature(method, path, body, timestamp)

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-App-Token", s.config.AppToken)
	req.Header.Set("X-App-Access-Sig", signature)
	req.Header.Set("X-App-Access-Ts", timestamp)

	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Sumsub API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// generateSignature generates HMAC signature for Sumsub API
func (s *SumsubProvider) generateSignature(method, path string, body []byte, timestamp string) string {
	// Signature = HMAC_SHA256(SecretKey, timestamp + method + path + body)
	message := timestamp + method + path
	if body != nil {
		message += string(body)
	}

	h := hmac.New(sha256.New, []byte(s.config.SecretKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
*/
