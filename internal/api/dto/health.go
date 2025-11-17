package dto

// HealthResponse represents the basic health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// StatusResponse represents the detailed system status response
type StatusResponse struct {
	Status   string           `json:"status"`
	Version  string           `json:"version"`
	Services ServiceStatuses  `json:"services"`
	Wallet   *WalletStatus    `json:"wallet,omitempty"`
}

// ServiceStatuses represents the status of all services
type ServiceStatuses struct {
	Database   ServiceStatus `json:"database"`
	Redis      ServiceStatus `json:"redis"`
	Blockchain ServiceStatus `json:"blockchain"`
}

// ServiceStatus represents the status of a single service
type ServiceStatus struct {
	Status  string `json:"status"` // "healthy", "unhealthy", "degraded"
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"` // e.g., "5ms"
}

// WalletStatus represents the hot wallet status
type WalletStatus struct {
	Address      string `json:"address"`
	BalanceUSDT  string `json:"balance_usdt,omitempty"`
	BalanceUSDC  string `json:"balance_usdc,omitempty"`
	BalanceSOL   string `json:"balance_sol,omitempty"`
	LastChecked  string `json:"last_checked,omitempty"`
}
