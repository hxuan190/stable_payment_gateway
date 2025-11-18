package trmlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client represents a TRM Labs API client
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// ScreeningResponse represents the response from TRM Labs screening API
type ScreeningResponse struct {
	Address        string            `json:"address"`
	Chain          string            `json:"chain"`
	RiskScore      int               `json:"risk_score"` // 0-100
	IsSanctioned   bool              `json:"is_sanctioned"`
	Flags          []string          `json:"flags"` // ["mixer", "sanctions", "darknet", etc]
	Details        map[string]string `json:"details"`
	ScreenedAt     time.Time         `json:"screened_at"`
	ExternalID     string            `json:"external_id,omitempty"`
}

// ScreeningRequest represents the request to TRM Labs screening API
type ScreeningRequest struct {
	Address    string `json:"address"`
	Chain      string `json:"chain"`
	ExternalID string `json:"external_id,omitempty"`
}

// NewClient creates a new TRM Labs client
func NewClient(apiKey string, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.trmlabs.com/public/v1"
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ScreenAddress screens a wallet address against TRM Labs database
// Returns screening result with risk score and flags
func (c *Client) ScreenAddress(ctx context.Context, address string, chain string) (*ScreeningResponse, error) {
	// Validate inputs
	if address == "" {
		return nil, fmt.Errorf("address is required")
	}
	if chain == "" {
		return nil, fmt.Errorf("chain is required")
	}

	// Normalize chain name
	chain = normalizeChainName(chain)

	// Create request payload
	reqBody := ScreeningRequest{
		Address:    address,
		Chain:      chain,
		ExternalID: fmt.Sprintf("screen-%d", time.Now().Unix()),
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/screening/addresses", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
			return nil, fmt.Errorf("TRM Labs API error (status %d): %s", resp.StatusCode, errorResp.Error)
		}
		return nil, fmt.Errorf("TRM Labs API error: status code %d", resp.StatusCode)
	}

	// Parse response
	var screeningResp ScreeningResponse
	if err := json.NewDecoder(resp.Body).Decode(&screeningResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	screeningResp.ScreenedAt = time.Now()

	return &screeningResp, nil
}

// BatchScreenAddresses screens multiple addresses in a single request
func (c *Client) BatchScreenAddresses(ctx context.Context, addresses []string, chain string) (map[string]*ScreeningResponse, error) {
	results := make(map[string]*ScreeningResponse)

	// For MVP, we'll screen addresses one by one
	// In production, use TRM Labs batch API if available
	for _, address := range addresses {
		result, err := c.ScreenAddress(ctx, address, chain)
		if err != nil {
			// Log error but continue with other addresses
			results[address] = &ScreeningResponse{
				Address:    address,
				Chain:      chain,
				RiskScore:  -1, // Indicate error
				ScreenedAt: time.Now(),
			}
			continue
		}
		results[address] = result
	}

	return results, nil
}

// normalizeChainName normalizes chain names to TRM Labs format
func normalizeChainName(chain string) string {
	switch chain {
	case "solana", "Solana", "SOLANA":
		return "solana"
	case "bsc", "BSC", "binance-smart-chain", "BNB Chain":
		return "binance-smart-chain"
	case "ethereum", "Ethereum", "ETHEREUM", "eth", "ETH":
		return "ethereum"
	case "polygon", "Polygon", "POLYGON", "matic":
		return "polygon"
	default:
		return chain
	}
}
