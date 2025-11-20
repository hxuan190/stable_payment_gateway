package blockchain

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"stable_payment_gateway/internal/ports"
	"stable_payment_gateway/internal/shared/events"
)

// ListenerManager manages multiple blockchain listeners
// It provides a unified interface for starting, stopping, and monitoring multiple blockchain listeners
type ListenerManager struct {
	// Map of blockchain type to listener
	listeners map[ports.BlockchainType]*EventBasedListenerAdapter

	// Event bus for publishing events
	eventBus events.EventBus

	// Logger
	logger *logrus.Logger

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// NewListenerManager creates a new blockchain listener manager
func NewListenerManager(eventBus events.EventBus, logger *logrus.Logger) *ListenerManager {
	if logger == nil {
		logger = logrus.New()
	}

	return &ListenerManager{
		listeners: make(map[ports.BlockchainType]*EventBasedListenerAdapter),
		eventBus:  eventBus,
		logger:    logger,
	}
}

// AddListener adds a blockchain listener to the manager
func (m *ListenerManager) AddListener(listener ports.BlockchainListener) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	blockchainType := listener.GetBlockchainType()

	// Check if listener already exists
	if _, exists := m.listeners[blockchainType]; exists {
		return fmt.Errorf("listener for %s already exists", blockchainType)
	}

	// Wrap listener with event-based adapter
	eventBasedListener := NewEventBasedListenerAdapter(listener, m.eventBus)

	m.listeners[blockchainType] = eventBasedListener

	m.logger.WithFields(logrus.Fields{
		"blockchain":      blockchainType,
		"wallet_address":  listener.GetWalletAddress(),
		"supported_tokens": listener.GetSupportedTokens(),
	}).Info("Added blockchain listener")

	return nil
}

// RemoveListener removes a blockchain listener from the manager
func (m *ListenerManager) RemoveListener(blockchainType ports.BlockchainType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	listener, exists := m.listeners[blockchainType]
	if !exists {
		return fmt.Errorf("listener for %s not found", blockchainType)
	}

	// Stop listener if running
	if listener.IsRunning() {
		ctx := context.Background()
		if err := listener.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop listener: %w", err)
		}
	}

	delete(m.listeners, blockchainType)

	m.logger.WithField("blockchain", blockchainType).Info("Removed blockchain listener")

	return nil
}

// StartAll starts all registered blockchain listeners
func (m *ListenerManager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for blockchainType, listener := range m.listeners {
		if listener.IsRunning() {
			m.logger.WithField("blockchain", blockchainType).Warn("Listener already running")
			continue
		}

		if err := listener.Start(ctx); err != nil {
			m.logger.WithFields(logrus.Fields{
				"blockchain": blockchainType,
				"error":      err.Error(),
			}).Error("Failed to start blockchain listener")
			return fmt.Errorf("failed to start %s listener: %w", blockchainType, err)
		}

		m.logger.WithField("blockchain", blockchainType).Info("Started blockchain listener")
	}

	return nil
}

// StopAll stops all running blockchain listeners
func (m *ListenerManager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errors []error

	for blockchainType, listener := range m.listeners {
		if !listener.IsRunning() {
			continue
		}

		if err := listener.Stop(ctx); err != nil {
			m.logger.WithFields(logrus.Fields{
				"blockchain": blockchainType,
				"error":      err.Error(),
			}).Error("Failed to stop blockchain listener")
			errors = append(errors, fmt.Errorf("failed to stop %s listener: %w", blockchainType, err))
			continue
		}

		m.logger.WithField("blockchain", blockchainType).Info("Stopped blockchain listener")
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop some listeners: %v", errors)
	}

	return nil
}

// GetListener retrieves a listener by blockchain type
func (m *ListenerManager) GetListener(blockchainType ports.BlockchainType) (*EventBasedListenerAdapter, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	listener, exists := m.listeners[blockchainType]
	if !exists {
		return nil, fmt.Errorf("listener for %s not found", blockchainType)
	}

	return listener, nil
}

// GetAllListeners returns all registered listeners
func (m *ListenerManager) GetAllListeners() map[ports.BlockchainType]*EventBasedListenerAdapter {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	listeners := make(map[ports.BlockchainType]*EventBasedListenerAdapter, len(m.listeners))
	for k, v := range m.listeners {
		listeners[k] = v
	}

	return listeners
}

// GetHealthStatus returns the health status of all listeners
func (m *ListenerManager) GetHealthStatus() map[ports.BlockchainType]ports.ListenerHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	healthStatus := make(map[ports.BlockchainType]ports.ListenerHealth)
	for blockchainType, listener := range m.listeners {
		healthStatus[blockchainType] = listener.GetListenerHealth()
	}

	return healthStatus
}

// IsAllHealthy checks if all listeners are healthy
func (m *ListenerManager) IsAllHealthy() bool {
	healthStatus := m.GetHealthStatus()

	for _, health := range healthStatus {
		if !health.IsHealthy {
			return false
		}
	}

	return true
}

// GetListenerCount returns the number of registered listeners
func (m *ListenerManager) GetListenerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.listeners)
}
