package trmlabs

import (
	"context"
	"strings"
	"time"
)

// MockClient is a mock implementation of TRM Labs client for testing
// and development without requiring actual API credentials
type MockClient struct {
	// SanctionedAddresses is a list of addresses to flag as sanctioned
	SanctionedAddresses []string
	// HighRiskAddresses is a list of addresses to flag as high risk
	HighRiskAddresses []string
	// SimulateError if true, will return errors for testing
	SimulateError bool
}

// NewMockClient creates a new mock TRM Labs client
func NewMockClient() *MockClient {
	return &MockClient{
		SanctionedAddresses: []string{
			// Known sanctioned addresses for testing
			"TornadoCash123456789",
			"SanctionedWallet999",
		},
		HighRiskAddresses: []string{
			// Known high-risk addresses for testing
			"MixerWallet123",
			"DarknetMarket456",
		},
		SimulateError: false,
	}
}

// ScreenAddress implements mock screening logic
func (m *MockClient) ScreenAddress(ctx context.Context, address string, chain string) (*ScreeningResponse, error) {
	if m.SimulateError {
		return nil, ErrAPIUnavailable
	}

	response := &ScreeningResponse{
		Address:      address,
		Chain:        normalizeChainName(chain),
		RiskScore:    0,
		IsSanctioned: false,
		Flags:        []string{},
		Details:      make(map[string]string),
		ScreenedAt:   time.Now(),
	}

	// Check if sanctioned
	for _, sanctioned := range m.SanctionedAddresses {
		if strings.Contains(address, sanctioned) || address == sanctioned {
			response.IsSanctioned = true
			response.RiskScore = 100
			response.Flags = append(response.Flags, "sanctions", "ofac")
			response.Details["reason"] = "Address flagged on OFAC sanctions list"
			return response, nil
		}
	}

	// Check if high risk
	for _, highRisk := range m.HighRiskAddresses {
		if strings.Contains(address, highRisk) || address == highRisk {
			response.RiskScore = 85
			response.Flags = append(response.Flags, "mixer", "high-risk")
			response.Details["reason"] = "Address associated with mixing service"
			return response, nil
		}
	}

	// Check for common risk patterns
	addressLower := strings.ToLower(address)
	if strings.Contains(addressLower, "mixer") {
		response.RiskScore = 70
		response.Flags = append(response.Flags, "mixer")
		response.Details["reason"] = "Address pattern suggests mixer usage"
	} else if strings.Contains(addressLower, "tornado") {
		response.RiskScore = 95
		response.Flags = append(response.Flags, "tornado-cash", "mixer")
		response.Details["reason"] = "Associated with Tornado Cash"
	} else if strings.Contains(addressLower, "darknet") {
		response.RiskScore = 80
		response.Flags = append(response.Flags, "darknet", "high-risk")
		response.Details["reason"] = "Possible darknet marketplace association"
	} else {
		// Safe address
		response.RiskScore = 0
		response.Details["reason"] = "No risk indicators found"
	}

	return response, nil
}

// BatchScreenAddresses implements mock batch screening
func (m *MockClient) BatchScreenAddresses(ctx context.Context, addresses []string, chain string) (map[string]*ScreeningResponse, error) {
	results := make(map[string]*ScreeningResponse)

	for _, address := range addresses {
		result, err := m.ScreenAddress(ctx, address, chain)
		if err != nil {
			results[address] = &ScreeningResponse{
				Address:    address,
				Chain:      chain,
				RiskScore:  -1,
				ScreenedAt: time.Now(),
			}
			continue
		}
		results[address] = result
	}

	return results, nil
}

// AddSanctionedAddress adds an address to the sanctioned list (for testing)
func (m *MockClient) AddSanctionedAddress(address string) {
	m.SanctionedAddresses = append(m.SanctionedAddresses, address)
}

// AddHighRiskAddress adds an address to the high-risk list (for testing)
func (m *MockClient) AddHighRiskAddress(address string) {
	m.HighRiskAddresses = append(m.HighRiskAddresses, address)
}
