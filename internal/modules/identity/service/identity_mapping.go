package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/pkg/cache"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
	"github.com/shopspring/decimal"
)

var (
	// ErrWalletNotRecognized is returned when a wallet cannot be auto-recognized
	ErrWalletNotRecognized = errors.New("wallet not recognized, KYC required")
	// ErrKYCRequired is returned when KYC is required for a new wallet
	ErrKYCRequired = errors.New("KYC verification required")
	// ErrKYCNotVerified is returned when user's KYC is not approved
	ErrKYCNotVerified = errors.New("KYC not verified")
	// ErrUserBlocked is returned when user is sanctioned or flagged
	ErrUserBlocked = errors.New("user is blocked due to sanctions or flags")
)

// KYCProvider defines the interface for KYC verification providers (Sumsub, Onfido, etc.)
type KYCProvider interface {
	// CreateApplicant creates a new KYC applicant
	CreateApplicant(ctx context.Context, fullName, email string) (applicantID string, err error)

	// GetApplicantStatus retrieves the current KYC status
	GetApplicantStatus(ctx context.Context, applicantID string) (status model.KYCStatus, err error)

	// VerifyFaceLiveness triggers face liveness check
	VerifyFaceLiveness(ctx context.Context, applicantID string) (passed bool, score float64, err error)

	// GetApplicantData retrieves KYC data (name, DOB, nationality, etc.)
	GetApplicantData(ctx context.Context, applicantID string) (data map[string]interface{}, err error)
}

// IdentityMappingService handles smart wallet-to-user identity mapping
// (PRD v2.2 - Smart Identity Mapping)
//
// Key Features:
// - One-time KYC with automatic wallet recognition
// - Cache-aside pattern (Redis → DB fallback)
// - 7-day cache TTL for active wallets
// - Face liveness detection for new wallets
type IdentityMappingService struct {
	walletMappingRepo *repository.WalletIdentityMappingRepository
	userRepo          *repository.UserRepository
	cache             *cache.RedisCache
	kycProvider       KYCProvider
	cacheTTL          time.Duration
}

// NewIdentityMappingService creates a new identity mapping service
func NewIdentityMappingService(
	walletMappingRepo *repository.WalletIdentityMappingRepository,
	userRepo *repository.UserRepository,
	cache *cache.RedisCache,
	kycProvider KYCProvider,
) *IdentityMappingService {
	return &IdentityMappingService{
		walletMappingRepo: walletMappingRepo,
		userRepo:          userRepo,
		cache:             cache,
		kycProvider:       kycProvider,
		cacheTTL:          7 * 24 * time.Hour, // 7 days
	}
}

// RecognizeWallet attempts to recognize a wallet address and return user identity
//
// Flow:
// 1. Check Redis cache (fast path)
// 2. If not cached, check database
// 3. If found in DB, cache for 7 days
// 4. If not found, return ErrWalletNotRecognized
//
// Returns:
// - WalletIdentityMapping if recognized
// - ErrWalletNotRecognized if wallet is new/unknown
// - ErrKYCNotVerified if wallet exists but KYC not approved
// - ErrUserBlocked if user is sanctioned or flagged
func (s *IdentityMappingService) RecognizeWallet(ctx context.Context, walletAddress string, blockchain model.Chain) (*model.WalletIdentityMapping, *model.User, error) {
	if walletAddress == "" {
		return nil, nil, errors.New("wallet address cannot be empty")
	}

	// Step 1: Check Redis cache first (fast path)
	cacheKey := s.getCacheKey(walletAddress, blockchain)
	cachedData, err := s.cache.Get(ctx, cacheKey)

	if err == nil && cachedData != "" {
		// Cache hit! Deserialize and return
		var mapping model.WalletIdentityMapping
		if err := json.Unmarshal([]byte(cachedData), &mapping); err == nil {
			// Load full user data
			user, err := s.userRepo.GetByID(mapping.UserID)
			if err != nil {
				// Cache might be stale, invalidate it
				_ = s.cache.Delete(ctx, cacheKey)
				// Continue to DB lookup below
			} else {
				// Cache hit success!
				return &mapping, user, nil
			}
		}
	}

	// Step 2: Cache miss, check database
	mapping, err := s.walletMappingRepo.GetByWalletAddress(walletAddress, blockchain)
	if err != nil {
		if errors.Is(err, repository.ErrWalletMappingNotFound) {
			return nil, nil, ErrWalletNotRecognized
		}
		return nil, nil, fmt.Errorf("failed to get wallet mapping from DB: %w", err)
	}

	// Step 3: Load user data
	user, err := s.userRepo.GetByID(mapping.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user data: %w", err)
	}

	// Step 4: Verify user is allowed to transact
	if user.IsSanctioned {
		return mapping, user, ErrUserBlocked
	}

	if mapping.IsFlagged {
		return mapping, user, ErrUserBlocked
	}

	if !user.IsKYCVerified() {
		return mapping, user, ErrKYCNotVerified
	}

	// Step 5: Cache the mapping for 7 days
	if err := s.cacheWalletMapping(ctx, mapping); err != nil {
		// Log error but don't fail the request
		// Cache failure should not block recognition
	}

	return mapping, user, nil
}

// GetOrCreateWalletMapping gets existing wallet mapping or creates a new one with KYC
//
// This is the main entry point for payment processing:
// 1. Try to recognize wallet (cache → DB)
// 2. If recognized and KYC approved: return user
// 3. If not recognized: trigger KYC flow
// 4. Create wallet mapping after successful KYC
func (s *IdentityMappingService) GetOrCreateWalletMapping(
	ctx context.Context,
	walletAddress string,
	blockchain model.Chain,
	fullName string,
	email string,
) (*model.WalletIdentityMapping, *model.User, bool, error) {
	// Try to recognize wallet first
	mapping, user, err := s.RecognizeWallet(ctx, walletAddress, blockchain)

	if err == nil {
		// Wallet recognized and user is good to go
		return mapping, user, false, nil // false = no new KYC needed
	}

	if !errors.Is(err, ErrWalletNotRecognized) {
		// Some other error occurred (DB error, user blocked, etc.)
		return mapping, user, false, err
	}

	// Wallet not recognized - need KYC
	// For MVP, we return an error indicating KYC is required
	// The caller should handle KYC initiation separately
	return nil, nil, true, ErrKYCRequired // true = new KYC needed
}

// CreateWalletMappingAfterKYC creates a wallet mapping after successful KYC
//
// This should be called after KYC verification is complete (e.g., Sumsub webhook)
func (s *IdentityMappingService) CreateWalletMappingAfterKYC(
	ctx context.Context,
	walletAddress string,
	blockchain model.Chain,
	userID uuid.UUID,
) (*model.WalletIdentityMapping, error) {
	// Verify user exists and has verified KYC
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if !user.IsKYCVerified() {
		return nil, ErrKYCNotVerified
	}

	// Check if mapping already exists
	existingMapping, err := s.walletMappingRepo.GetByWalletAddress(walletAddress, blockchain)
	if err == nil {
		// Mapping already exists - just update it
		existingMapping.KYCStatus = user.KYCStatus
		if err := s.walletMappingRepo.Update(existingMapping); err != nil {
			return nil, fmt.Errorf("failed to update wallet mapping: %w", err)
		}

		// Cache the updated mapping
		_ = s.cacheWalletMapping(ctx, existingMapping)

		return existingMapping, nil
	}

	// Create new wallet mapping
	mapping := &model.WalletIdentityMapping{
		ID:                uuid.New(),
		WalletAddress:     walletAddress,
		Blockchain:        blockchain,
		WalletAddressHash: model.ComputeWalletHash(walletAddress),
		UserID:            userID,
		KYCStatus:         user.KYCStatus,
		FirstSeenAt:       time.Now(),
		LastSeenAt:        time.Now(),
		PaymentCount:      0,
		TotalVolumeUSD:    decimal.Zero,
		IsFlagged:         false,
		Metadata:          make(map[string]interface{}),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.walletMappingRepo.Create(mapping); err != nil {
		return nil, fmt.Errorf("failed to create wallet mapping: %w", err)
	}

	// Cache the new mapping
	_ = s.cacheWalletMapping(ctx, mapping)

	return mapping, nil
}

// UpdateWalletActivity updates wallet payment activity and caching
//
// This should be called after each successful payment
func (s *IdentityMappingService) UpdateWalletActivity(
	ctx context.Context,
	walletAddress string,
	blockchain model.Chain,
	paymentID uuid.UUID,
	amountUSD decimal.Decimal,
) error {
	// Update database
	mapping, err := s.walletMappingRepo.GetByWalletAddress(walletAddress, blockchain)
	if err != nil {
		return fmt.Errorf("failed to get wallet mapping: %w", err)
	}

	// Update activity in DB
	if err := s.walletMappingRepo.UpdateActivity(mapping.ID, amountUSD); err != nil {
		return fmt.Errorf("failed to update wallet activity: %w", err)
	}

	// Update user payment stats
	if err := s.userRepo.IncrementPaymentStats(mapping.UserID, amountUSD); err != nil {
		return fmt.Errorf("failed to update user payment stats: %w", err)
	}

	// Refresh cache with updated activity
	mapping, err = s.walletMappingRepo.GetByWalletAddress(walletAddress, blockchain)
	if err == nil {
		_ = s.cacheWalletMapping(ctx, mapping)
	}

	return nil
}

// InvalidateWalletCache invalidates the cache for a wallet
//
// This should be called when:
// - KYC status changes
// - Wallet is flagged/unflagged
// - User is sanctioned
func (s *IdentityMappingService) InvalidateWalletCache(ctx context.Context, walletAddress string, blockchain model.Chain) error {
	cacheKey := s.getCacheKey(walletAddress, blockchain)
	return s.cache.Delete(ctx, cacheKey)
}

// FlagWallet flags a wallet as suspicious and invalidates cache
func (s *IdentityMappingService) FlagWallet(ctx context.Context, walletAddress string, blockchain model.Chain, reason string) error {
	mapping, err := s.walletMappingRepo.GetByWalletAddress(walletAddress, blockchain)
	if err != nil {
		return fmt.Errorf("failed to get wallet mapping: %w", err)
	}

	if err := s.walletMappingRepo.Flag(mapping.ID, reason); err != nil {
		return fmt.Errorf("failed to flag wallet: %w", err)
	}

	// Invalidate cache immediately
	_ = s.InvalidateWalletCache(ctx, walletAddress, blockchain)

	return nil
}

// UnflagWallet removes flag from a wallet and refreshes cache
func (s *IdentityMappingService) UnflagWallet(ctx context.Context, walletAddress string, blockchain model.Chain) error {
	mapping, err := s.walletMappingRepo.GetByWalletAddress(walletAddress, blockchain)
	if err != nil {
		return fmt.Errorf("failed to get wallet mapping: %w", err)
	}

	if err := s.walletMappingRepo.Unflag(mapping.ID); err != nil {
		return fmt.Errorf("failed to unflag wallet: %w", err)
	}

	// Refresh cache with unflagged status
	mapping, err = s.walletMappingRepo.GetByWalletAddress(walletAddress, blockchain)
	if err == nil {
		_ = s.cacheWalletMapping(ctx, mapping)
	}

	return nil
}

// GetWalletsByUser retrieves all wallet mappings for a user
func (s *IdentityMappingService) GetWalletsByUser(ctx context.Context, userID uuid.UUID) ([]*model.WalletIdentityMapping, error) {
	return s.walletMappingRepo.GetByUserID(userID)
}

// GetCacheStats returns cache performance statistics
//
// This is useful for monitoring cache hit rate (target > 90%)
func (s *IdentityMappingService) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	// Get total wallet mappings count
	totalMappings, err := s.walletMappingRepo.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to get total mappings count: %w", err)
	}

	// Get active mappings (last 7 days)
	activeMappings, err := s.walletMappingRepo.ListActive(1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get active mappings: %w", err)
	}

	stats := map[string]interface{}{
		"total_wallet_mappings": totalMappings,
		"active_mappings_7d":    len(activeMappings),
		"cache_ttl_seconds":     int(s.cacheTTL.Seconds()),
		"cache_ttl_days":        7,
	}

	return stats, nil
}

// Helper functions

// getCacheKey generates Redis cache key for wallet mapping
func (s *IdentityMappingService) getCacheKey(walletAddress string, blockchain model.Chain) string {
	return fmt.Sprintf("wallet_identity:%s:%s", blockchain, walletAddress)
}

// cacheWalletMapping caches a wallet mapping in Redis
func (s *IdentityMappingService) cacheWalletMapping(ctx context.Context, mapping *model.WalletIdentityMapping) error {
	if mapping == nil {
		return errors.New("mapping cannot be nil")
	}

	// Serialize to JSON
	data, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	// Store in Redis with 7-day TTL
	cacheKey := s.getCacheKey(mapping.WalletAddress, mapping.Blockchain)
	if err := s.cache.Set(ctx, cacheKey, string(data), s.cacheTTL); err != nil {
		return fmt.Errorf("failed to cache mapping: %w", err)
	}

	return nil
}
