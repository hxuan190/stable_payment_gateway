package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	secretKey := "test-secret-key-at-least-32-chars-long"
	expirationHours := 24

	manager := NewManager(secretKey, expirationHours)

	assert.NotNil(t, manager)
	assert.Equal(t, secretKey, manager.secretKey)
	assert.Equal(t, time.Duration(expirationHours)*time.Hour, manager.expirationDuration)
}

func TestGenerateToken(t *testing.T) {
	manager := NewManager("test-secret-key-at-least-32-chars-long", 24)

	tests := []struct {
		name    string
		adminID string
		email   string
		role    string
		wantErr bool
	}{
		{
			name:    "Valid admin token",
			adminID: "admin123",
			email:   "admin@example.com",
			role:    "super_admin",
			wantErr: false,
		},
		{
			name:    "Valid moderator token",
			adminID: "mod456",
			email:   "mod@example.com",
			role:    "moderator",
			wantErr: false,
		},
		{
			name:    "Empty admin ID",
			adminID: "",
			email:   "admin@example.com",
			role:    "admin",
			wantErr: false, // Still generates token, validation happens elsewhere
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.GenerateToken(tt.adminID, tt.email, tt.role)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	manager := NewManager("test-secret-key-at-least-32-chars-long", 24)

	t.Run("Valid token", func(t *testing.T) {
		adminID := "admin123"
		email := "admin@example.com"
		role := "super_admin"

		token, err := manager.GenerateToken(adminID, email, role)
		require.NoError(t, err)

		claims, err := manager.ValidateToken(token)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, adminID, claims.AdminID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
		assert.Equal(t, "payment-gateway", claims.Issuer)
		assert.Equal(t, adminID, claims.Subject)
	})

	t.Run("Invalid token format", func(t *testing.T) {
		claims, err := manager.ValidateToken("invalid-token")
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("Empty token", func(t *testing.T) {
		claims, err := manager.ValidateToken("")
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("Token with wrong secret", func(t *testing.T) {
		wrongManager := NewManager("wrong-secret-key-different-32-chars", 24)
		token, err := wrongManager.GenerateToken("admin123", "admin@example.com", "admin")
		require.NoError(t, err)

		claims, err := manager.ValidateToken(token)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("Expired token", func(t *testing.T) {
		// Create a manager with 0 hour expiration (immediate expiry)
		shortManager := NewManager("test-secret-key-at-least-32-chars-long", 0)
		token, err := shortManager.GenerateToken("admin123", "admin@example.com", "admin")
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(1 * time.Second)

		claims, err := shortManager.ValidateToken(token)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, ErrExpiredToken)
	})
}

func TestRefreshToken(t *testing.T) {
	manager := NewManager("test-secret-key-at-least-32-chars-long", 24)

	t.Run("Refresh valid token", func(t *testing.T) {
		adminID := "admin123"
		email := "admin@example.com"
		role := "super_admin"

		oldToken, err := manager.GenerateToken(adminID, email, role)
		require.NoError(t, err)

		// Wait a bit to ensure different timestamps
		time.Sleep(1 * time.Second)

		newToken, err := manager.RefreshToken(oldToken)
		assert.NoError(t, err)
		assert.NotEmpty(t, newToken)
		assert.NotEqual(t, oldToken, newToken)

		// Validate new token has same claims
		claims, err := manager.ValidateToken(newToken)
		assert.NoError(t, err)
		assert.Equal(t, adminID, claims.AdminID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
	})

	t.Run("Refresh expired token within grace period", func(t *testing.T) {
		shortManager := NewManager("test-secret-key-at-least-32-chars-long", 0)
		oldToken, err := shortManager.GenerateToken("admin123", "admin@example.com", "admin")
		require.NoError(t, err)

		// Wait for expiration
		time.Sleep(1 * time.Second)

		// Should still be able to refresh (within 7 day grace period)
		newToken, err := shortManager.RefreshToken(oldToken)
		assert.NoError(t, err)
		assert.NotEmpty(t, newToken)
	})

	t.Run("Refresh invalid token", func(t *testing.T) {
		newToken, err := manager.RefreshToken("invalid-token")
		assert.Error(t, err)
		assert.Empty(t, newToken)
	})
}

func TestExtractClaims(t *testing.T) {
	manager := NewManager("test-secret-key-at-least-32-chars-long", 24)

	t.Run("Extract claims from valid token", func(t *testing.T) {
		adminID := "admin123"
		email := "admin@example.com"
		role := "super_admin"

		token, err := manager.GenerateToken(adminID, email, role)
		require.NoError(t, err)

		claims, err := manager.ExtractClaims(token)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, adminID, claims.AdminID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
	})

	t.Run("Extract claims from expired token", func(t *testing.T) {
		shortManager := NewManager("test-secret-key-at-least-32-chars-long", 0)
		token, err := shortManager.GenerateToken("admin123", "admin@example.com", "admin")
		require.NoError(t, err)

		// Wait for expiration
		time.Sleep(1 * time.Second)

		// Should still extract claims even if expired
		claims, err := shortManager.ExtractClaims(token)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, "admin123", claims.AdminID)
	})

	t.Run("Extract claims from invalid token", func(t *testing.T) {
		claims, err := manager.ExtractClaims("invalid-token")
		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestGetExpirationTime(t *testing.T) {
	manager := NewManager("test-secret-key-at-least-32-chars-long", 24)

	t.Run("Get expiration time", func(t *testing.T) {
		token, err := manager.GenerateToken("admin123", "admin@example.com", "admin")
		require.NoError(t, err)

		expirationTime, err := manager.GetExpirationTime(token)
		assert.NoError(t, err)
		assert.False(t, expirationTime.IsZero())

		// Should expire in approximately 24 hours
		expectedExpiration := time.Now().Add(24 * time.Hour)
		timeDiff := expirationTime.Sub(expectedExpiration)
		assert.Less(t, timeDiff.Abs(), 10*time.Second) // Allow 10 second tolerance
	})

	t.Run("Get expiration from invalid token", func(t *testing.T) {
		expirationTime, err := manager.GetExpirationTime("invalid-token")
		assert.Error(t, err)
		assert.True(t, expirationTime.IsZero())
	})
}

func TestIsTokenExpired(t *testing.T) {
	t.Run("Valid non-expired token", func(t *testing.T) {
		manager := NewManager("test-secret-key-at-least-32-chars-long", 24)
		token, err := manager.GenerateToken("admin123", "admin@example.com", "admin")
		require.NoError(t, err)

		isExpired := manager.IsTokenExpired(token)
		assert.False(t, isExpired)
	})

	t.Run("Expired token", func(t *testing.T) {
		manager := NewManager("test-secret-key-at-least-32-chars-long", 0)
		token, err := manager.GenerateToken("admin123", "admin@example.com", "admin")
		require.NoError(t, err)

		// Wait for expiration
		time.Sleep(1 * time.Second)

		isExpired := manager.IsTokenExpired(token)
		assert.True(t, isExpired)
	})

	t.Run("Invalid token", func(t *testing.T) {
		manager := NewManager("test-secret-key-at-least-32-chars-long", 24)
		isExpired := manager.IsTokenExpired("invalid-token")
		assert.True(t, isExpired) // Invalid tokens are considered expired
	})
}

func TestTokenClaimsIntegrity(t *testing.T) {
	manager := NewManager("test-secret-key-at-least-32-chars-long", 24)

	adminID := "admin123"
	email := "admin@example.com"
	role := "super_admin"

	token, err := manager.GenerateToken(adminID, email, role)
	require.NoError(t, err)

	// Validate multiple times to ensure consistency
	for i := 0; i < 5; i++ {
		claims, err := manager.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, adminID, claims.AdminID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
	}
}

func TestMultipleManagers(t *testing.T) {
	manager1 := NewManager("secret-key-1-at-least-32-chars-long", 24)
	manager2 := NewManager("secret-key-2-at-least-32-chars-long", 24)

	token1, err := manager1.GenerateToken("admin1", "admin1@example.com", "admin")
	require.NoError(t, err)

	token2, err := manager2.GenerateToken("admin2", "admin2@example.com", "admin")
	require.NoError(t, err)

	// Manager 1 should validate its own token
	claims1, err := manager1.ValidateToken(token1)
	assert.NoError(t, err)
	assert.Equal(t, "admin1", claims1.AdminID)

	// Manager 2 should validate its own token
	claims2, err := manager2.ValidateToken(token2)
	assert.NoError(t, err)
	assert.Equal(t, "admin2", claims2.AdminID)

	// Manager 1 should NOT validate Manager 2's token
	_, err = manager1.ValidateToken(token2)
	assert.Error(t, err)

	// Manager 2 should NOT validate Manager 1's token
	_, err = manager2.ValidateToken(token1)
	assert.Error(t, err)
}
